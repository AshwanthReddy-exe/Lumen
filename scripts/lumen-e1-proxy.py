#!/usr/bin/env python3
"""One-shot loopback fault injector for the pinned macOS Runs journey."""

import http.client
import http.server
import json
import os
import threading
import time
import urllib.parse


UPSTREAM_PORT = int(os.environ["LUMEN_E1_UPSTREAM_PORT"])
STATE_FILE = os.environ["LUMEN_E1_PROXY_STATE"]
ARM_FILE = os.environ["LUMEN_E1_PROXY_ARM"]
CREATE_DROP_ARM = os.environ["LUMEN_E1_CREATE_DROP_ARM"]
GATEWAY_RESTART_ARM = os.environ["LUMEN_E1_GATEWAY_RESTART_ARM"]
REORDER_ARM = os.environ["LUMEN_E1_REORDER_ARM"]
REORDER_PREFIX_RELEASE = os.environ["LUMEN_E1_REORDER_PREFIX_RELEASE"]
REORDER_RELEASE = os.environ["LUMEN_E1_REORDER_RELEASE"]
LOCK = threading.Lock()
STATE = {"port": 0, "creates": 0, "e1_creates": 0, "cut_reserved": False,
         "cut_event": False, "event_id": "", "event_type": "", "run_id": "",
         "ambiguous_creates": 0, "drop_reserved": False, "create_dropped": False,
         "ambiguous_run_id": "", "ambiguous_runtime_status": "",
         "gateway_restart_creates": 0, "gateway_restart_run_id": "",
         "reordered_creates": 0, "reordered_run_id": "", "reorder_reserved": False,
         "reordered_prefix_sent": False, "reordered_older_sent": False,
         "reordered_frames_sent": 0,
         "reordered_barrier_sent": False, "reorder_release_timeout": False}


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
        drop_create = False
        e1_create = False
        gateway_restart_create = False
        reordered_create = False
        with LOCK:
            if self.command == "POST" and path == "/v1/runs":
                STATE["creates"] += 1
                if os.path.exists(ARM_FILE):
                    with open(ARM_FILE, encoding="utf-8") as armed:
                        if self.headers.get("Idempotency-Key") == armed.read().strip():
                            STATE["e1_creates"] += 1
                            e1_create = True
                if os.path.exists(CREATE_DROP_ARM):
                    with open(CREATE_DROP_ARM, encoding="utf-8") as armed:
                        if self.headers.get("Idempotency-Key") == armed.read().strip():
                            STATE["ambiguous_creates"] += 1
                            drop_create = not STATE["drop_reserved"]
                            STATE["drop_reserved"] = True
                if os.path.exists(GATEWAY_RESTART_ARM):
                    with open(GATEWAY_RESTART_ARM, encoding="utf-8") as armed:
                        if self.headers.get("Idempotency-Key") == armed.read().strip():
                            STATE["gateway_restart_creates"] += 1
                            gateway_restart_create = True
                if os.path.exists(REORDER_ARM):
                    with open(REORDER_ARM, encoding="utf-8") as armed:
                        if self.headers.get("Idempotency-Key") == armed.read().strip():
                            STATE["reordered_creates"] += 1
                            reordered_create = True
                save()
        connection = http.client.HTTPConnection("127.0.0.1", UPSTREAM_PORT, timeout=180)
        try:
            connection.request(self.command, self.path, body, headers)
            response = connection.getresponse()
            if e1_create or reordered_create:
                content_length = response.getheader("Content-Length", "")
                if 200 <= response.status < 300 and content_length.isdigit() and int(content_length) <= 65536:
                    reply = response.read(int(content_length))
                    try:
                        metadata = json.loads(reply)
                    except (ValueError, UnicodeDecodeError):
                        metadata = {}
                    if not isinstance(metadata, dict):
                        metadata = {}
                    with LOCK:
                        if e1_create:
                            STATE["run_id"] = metadata.get("run_id", "")
                        if reordered_create:
                            STATE["reordered_run_id"] = metadata.get("run_id", "")
                        save()
                    self.send_response(response.status)
                    for key, value in response.getheaders():
                        if key.lower() not in ("connection", "transfer-encoding", "content-length"):
                            self.send_header(key, value)
                    self.send_header("Content-Length", str(len(reply)))
                    self.send_header("Connection", "close")
                    self.end_headers()
                    self.wfile.write(reply)
                    return
            if gateway_restart_create:
                content_length = response.getheader("Content-Length", "")
                if content_length.isdigit() and int(content_length) <= 65536:
                    reply = response.read(int(content_length))
                    try:
                        metadata = json.loads(reply)
                    except (ValueError, UnicodeDecodeError):
                        metadata = {}
                    if not isinstance(metadata, dict):
                        metadata = {}
                    with LOCK:
                        STATE["gateway_restart_run_id"] = metadata.get("run_id", "")
                        save()
                    self.send_response(response.status)
                    for key, value in response.getheaders():
                        if key.lower() not in ("connection", "transfer-encoding", "content-length"):
                            self.send_header(key, value)
                    self.send_header("Content-Length", str(len(reply)))
                    self.send_header("Connection", "close")
                    self.end_headers()
                    self.wfile.write(reply)
                    return
            if drop_create:
                reply = response.read(65537)
                if len(reply) <= 65536 and 200 <= response.status < 300:
                    try:
                        metadata = json.loads(reply)
                    except (ValueError, UnicodeDecodeError):
                        metadata = {}
                    if not isinstance(metadata, dict):
                        metadata = {}
                    with LOCK:
                        STATE["ambiguous_run_id"] = metadata.get("run_id", "")
                        STATE["ambiguous_runtime_status"] = metadata.get("status", "")
                        STATE["create_dropped"] = True
                        save()
                self.close_connection = True
                return
            self.send_response(response.status)
            for key, value in response.getheaders():
                if key.lower() not in ("connection", "transfer-encoding", "content-length"):
                    self.send_header(key, value)
            self.send_header("Connection", "close")
            self.end_headers()
            if path.endswith("/events") and response.status == 200:
                with LOCK:
                    should_reorder = (STATE["reordered_run_id"] and
                                      path == f"/v1/runs/{STATE['reordered_run_id']}/events" and
                                      not STATE["reorder_reserved"])
                    should_cut = (os.path.exists(ARM_FILE) and STATE["run_id"] and
                                  path == f"/v1/runs/{STATE['run_id']}/events" and
                                  not STATE["cut_reserved"])
                    if should_reorder:
                        STATE["reorder_reserved"] = True
                        save()
                    if should_cut:
                        STATE["cut_reserved"] = True
                        save()
                if should_reorder:
                    first = b'id: 2\nevent: running\ndata: {"status":"running"}\n\n'
                    older = b'id: 1\nevent: queued\ndata: {"status":"queued"}\n\n'
                    duplicate = b'id: 2\nevent: failed\ndata: {"status":"failed"}\n\n'
                    barrier = b'id: 3\nevent: running\ndata: {"status":"running"}\n\n'
                    self.wfile.write(first)
                    self.wfile.flush()
                    with LOCK:
                        STATE["reordered_prefix_sent"] = True
                        save()
                    deadline = time.monotonic() + 30
                    while not os.path.exists(REORDER_PREFIX_RELEASE) and time.monotonic() < deadline:
                        time.sleep(0.05)
                    if not os.path.exists(REORDER_PREFIX_RELEASE):
                        with LOCK:
                            STATE["reorder_release_timeout"] = True
                            save()
                        return
                    self.wfile.write(older)
                    self.wfile.flush()
                    with LOCK:
                        STATE["reordered_older_sent"] = True
                        save()
                    deadline = time.monotonic() + 30
                    while not os.path.exists(REORDER_RELEASE) and time.monotonic() < deadline:
                        time.sleep(0.05)
                    if not os.path.exists(REORDER_RELEASE):
                        with LOCK:
                            STATE["reorder_release_timeout"] = True
                            save()
                        return
                    self.wfile.write(duplicate)
                    self.wfile.flush()
                    self.wfile.write(barrier)
                    self.wfile.flush()
                    with LOCK:
                        STATE["reordered_frames_sent"] = 4
                        STATE["reordered_barrier_sent"] = True
                        save()
                    while line := response.readline():
                        self.wfile.write(line)
                        if line in (b"\n", b"\r\n"):
                            self.wfile.flush()
                    return
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
