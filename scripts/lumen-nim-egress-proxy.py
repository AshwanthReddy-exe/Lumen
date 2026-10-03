#!/usr/bin/env python3
"""Test-only CONNECT proxy: allow exactly NVIDIA NIM on TCP/443."""

from __future__ import annotations

import ipaddress
import json
import select
import socket
import socketserver
import sys

ALLOWED_HOST = "integrate.api.nvidia.com"
LISTEN = ("0.0.0.0", 3128)
MAX_HEADER = 8192
CONNECT_TIMEOUT = 8
last_upstream_ip: str | None = None


def public_addresses(host: str) -> list[ipaddress.IPv4Address | ipaddress.IPv6Address]:
    if host != ALLOWED_HOST:
        raise ValueError("destination_denied")
    addresses = {
        ipaddress.ip_address(item[4][0])
        for item in socket.getaddrinfo(host, 443, type=socket.SOCK_STREAM)
    }
    if not addresses or any(not address.is_global for address in addresses):
        raise ValueError("destination_resolution_denied")
    return sorted(addresses, key=lambda address: (address.version, int(address)))


def response(sock: socket.socket, status: str, body: bytes = b"") -> None:
    sock.sendall(
        f"HTTP/1.1 {status}\r\nConnection: close\r\nContent-Length: {len(body)}\r\n\r\n".encode()
        + body
    )


def read_headers(sock: socket.socket) -> bytes:
    data = bytearray()
    while not data.endswith(b"\r\n\r\n"):
        chunk = sock.recv(1)
        if not chunk:
            raise ValueError("incomplete_request")
        data.extend(chunk)
        if len(data) > MAX_HEADER:
            raise ValueError("request_too_large")
    return bytes(data)


def relay(client: socket.socket, upstream: socket.socket) -> None:
    # ponytail: blocking sendall keeps TLS records intact; 120s caps backpressure stalls.
    client.settimeout(120)
    upstream.settimeout(120)
    peers = {client: upstream, upstream: client}
    while True:
        readable, _, _ = select.select(list(peers), [], [], 120)
        if not readable:
            return
        for source in readable:
            try:
                data = source.recv(65536)
            except OSError:
                return
            if not data:
                return
            try:
                peers[source].sendall(data)
            except OSError:
                return


class Handler(socketserver.BaseRequestHandler):
    def handle(self) -> None:
        global last_upstream_ip
        client = self.request
        client.settimeout(CONNECT_TIMEOUT)
        upstream: socket.socket | None = None
        try:
            raw = read_headers(client)
            first = raw.split(b"\r\n", 1)[0].decode("ascii")
            method, target, version = first.split(" ")
            if version not in {"HTTP/1.0", "HTTP/1.1"}:
                response(client, "400 Bad Request")
                return
            if method == "GET" and target == "/__lumen_test/last-upstream":
                if last_upstream_ip is None:
                    response(client, "503 Service Unavailable", b"null")
                    return
                body = json.dumps(last_upstream_ip).encode()
                response(client, "200 OK", body)
                return
            if method != "CONNECT" or target.count(":") != 1:
                response(client, "403 Forbidden")
                return
            host, port_text = target.rsplit(":", 1)
            if not host or "@" in host or any(char.isspace() for char in host):
                response(client, "403 Forbidden")
                return
            if host.lower().removesuffix(".") != ALLOWED_HOST or port_text != "443":
                response(client, "403 Forbidden")
                return

            addresses = public_addresses(ALLOWED_HOST)
            last_upstream_ip = None
            last_error: OSError | None = None
            for address in addresses:
                try:
                    upstream = socket.create_connection((str(address), 443), timeout=CONNECT_TIMEOUT)
                    last_upstream_ip = str(address)
                    break
                except OSError as error:
                    last_error = error
            if upstream is None:
                raise last_error or OSError("no public endpoint address")
            client.sendall(b"HTTP/1.1 200 Connection Established\r\n\r\n")
            client.settimeout(None)
            relay(client, upstream)
        except ValueError as error:
            status = "403 Forbidden" if str(error).startswith("destination_") else "400 Bad Request"
            response(client, status)
        except (OSError, UnicodeError, ValueError):
            try:
                response(client, "502 Bad Gateway")
            except OSError:
                pass
        finally:
            if upstream is not None:
                upstream.close()


class Server(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True


def self_test() -> None:
    assert public_addresses.__name__ == "public_addresses"
    for denied in ("integrate.api.nvidia.com.evil.test", "evil.integrate.api.nvidia.com", "127.0.0.1"):
        try:
            public_addresses(denied)
        except ValueError as error:
            assert str(error) == "destination_denied"
        else:
            raise AssertionError(f"accepted unapproved host {denied}")
    print("proxy policy self-test passed")


if __name__ == "__main__":
    if sys.argv[1:] == ["--self-test"]:
        self_test()
    elif len(sys.argv) > 1:
        raise SystemExit("usage: lumen-nim-egress-proxy.py [--self-test]")
    else:
        with Server(LISTEN, Handler) as server:
            server.serve_forever()
