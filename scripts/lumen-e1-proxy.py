#!/usr/bin/env python3
"""One-shot loopback fault injector for the pinned macOS Runs journey."""

import http.client
import http.server
import json
import os
import threading
import urllib.parse


UPSTREAM_PORT = int(os.environ["LUMEN_E1_UPSTREAM_PORT"])
STATE_FILE = os.environ["LUMEN_E1_PROXY_STATE"]
ARM_FILE = os.environ["LUMEN_E1_PROXY_ARM"]
LOCK = threading.Lock()
STATE = {"port": 0, "creates": 0, "e1_creates": 0, "cut_reserved": False,
         "cut_event": False, "event_id": "", "event_type": "", "run_id": ""}


def save():
    temporary = STATE_FILE + ".tmp"
    with open(temporary, "w", encoding="utf-8") as output:
        json.dump(STATE, output)
    os.replace(temporary, STATE_FILE)


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.0"

    def log_message(self, *_args):
        pass

    def do_GET(self):
        self.forward()

    def do_POST(self):
        self.forward()

    def forward(self):
        path = urllib.parse.urlsplit(self.path).path
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        headers = {key: value for key, value in self.headers.items()
                   if key.lower() not in ("host", "connection", "content-length")}
        if body:
            headers["Content-Length"] = str(len(body))
        with LOCK:
            if self.command == "POST" and path == "/v1/runs":
                STATE["creates"] += 1
                if os.path.exists(ARM_FILE):
                    with open(ARM_FILE, encoding="utf-8") as armed:
                        if self.headers.get("Idempotency-Key") == armed.read().strip():
                            STATE["e1_creates"] += 1
                save()
        connection = http.client.HTTPConnection("127.0.0.1", UPSTREAM_PORT, timeout=180)
        try:
            connection.request(self.command, self.path, body, headers)
            response = connection.getresponse()
            self.send_response(response.status)
            for key, value in response.getheaders():
                if key.lower() not in ("connection", "transfer-encoding", "content-length"):
                    self.send_header(key, value)
            self.send_header("Connection", "close")
            self.end_headers()
            if path.endswith("/events") and response.status == 200:
                with LOCK:
                    should_cut = os.path.exists(ARM_FILE) and not STATE["cut_reserved"]
                    if should_cut:
                        STATE["cut_reserved"] = True
                        STATE["run_id"] = path.split("/")[3]
                        save()
                if should_cut:
                    event = bytearray()
                    while line := response.readline():
                        event.extend(line)
                        if line in (b"\n", b"\r\n") and b"data:" in event:
                            self.wfile.write(event)
                            self.wfile.flush()
                            event_id = ""
                            event_type = ""
                            for field in event.splitlines():
                                if field.startswith(b"id:"):
                                    event_id = field[3:].strip().decode("utf-8", "replace")
                                if field.startswith(b"event:"):
                                    event_type = field[6:].strip().decode("utf-8", "replace")
                            with LOCK:
                                STATE["cut_event"] = True
                                STATE["event_id"] = event_id
                                STATE["event_type"] = event_type
                                save()
                            return
                    return
            while chunk := response.read(65536):
                self.wfile.write(chunk)
        except (BrokenPipeError, ConnectionResetError):
            pass
        finally:
            connection.close()


server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
STATE["port"] = server.server_port
save()
server.serve_forever()
