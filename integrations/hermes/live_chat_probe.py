"""Exercise pinned Hermes Runs against a forced, forbidden tool call.

Uses a disposable Hermes home and a loopback synthetic model. No user data or
real provider credentials are used. This is deployment qualification evidence,
not a Host runtime certificate.
"""

import argparse
from contextlib import ExitStack
import hashlib
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def request(url, key=None, body=None, authorization=None):
    headers = {"Authorization": authorization or f"Bearer {key}"} if key or authorization else {}
    if body is not None:
        headers["Content-Type"] = "application/json"
    with urlopen(Request(url, data=json.dumps(body).encode() if body is not None else None,
                         headers=headers), timeout=5) as response:
        return json.load(response)


def settled_run(api, key, body):
    run_id = request(api + "/v1/runs", key, body)["run_id"]
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        status = request(api + "/v1/runs/" + run_id, key)
        if status.get("status") in {"completed", "failed", "cancelled"}:
            return status
        time.sleep(0.2)
    raise TimeoutError("Hermes run did not settle")


def provider_class(observed):
    class Provider(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def do_GET(self):
            if self.path != "/v1/models":
                self.send_error(404)
                return
            self.reply({"object": "list", "data": [{"id": "synthetic-model", "object": "model",
                                                   "owned_by": "lumen-probe"}]})

        def do_POST(self):
            if self.path != "/v1/chat/completions":
                self.send_error(404)
                return
            body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))) or b"{}")
            observed.append(body)
            followup = any(message.get("role") == "tool" for message in body.get("messages", []))
            user_text = " ".join(str(message.get("content", "")) for message in body.get("messages", [])
                                 if message.get("role") == "user")
            normal = "LUMEN_NORMAL_CHAT_PROBE" in user_text
            if normal or followup:
                deltas = [({"role": "assistant"}, None), ({"content": "synthetic finish"}, None),
                          ({}, "stop")]
            else:
                tool_name = ("terminal" if "LUMEN_TERMINAL_PROBE" in user_text else
                             "memory" if "LUMEN_MEMORY_WRITE_PROBE" in user_text else "session_search")
                arguments = ({"command": "touch LUMEN_FORBIDDEN_SIDE_EFFECT"} if tool_name == "terminal"
                             else {"action": "write", "target": "memory", "content": "LUMEN_FORBIDDEN_MEMORY_WRITE"}
                             if tool_name == "memory" else {"query": "LUMEN_OLD_SESSION_SENTINEL"})
                call = {"index": 0, "id": "call-forbidden", "type": "function",
                        "function": {"name": tool_name, "arguments": json.dumps(arguments)}}
                deltas = [({"role": "assistant"}, None), ({"tool_calls": [call]}, None),
                          ({}, "tool_calls")]
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            for delta, finish in deltas:
                chunk = {"id": "synthetic-chat", "object": "chat.completion.chunk", "created": 0,
                         "model": "synthetic-model", "choices": [{"index": 0, "delta": delta,
                                                                   "finish_reason": finish}]}
                self.wfile.write(("data: " + json.dumps(chunk) + "\n\n").encode())
            self.wfile.write(b"data: [DONE]\n\n")

        def reply(self, body):
            encoded = json.dumps(body).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)

    return Provider


def stop_gateway(gateway):
    gateway.terminate()
    try:
        gateway.wait(timeout=10)
    except subprocess.TimeoutExpired:
        gateway.kill()
        gateway.wait()


def probe(source):
    expected = "2237be355906fbe6065ce1815711eee52b2d646e"
    commit = subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip()
    if commit != expected:
        raise ValueError("Hermes source is not the pinned commit")
    patch = Path(__file__).with_name("patches") / "0001-lumen-chat-zero-tool.patch"
    expected_files = {"agent/turn_tool_round.py", "gateway/platforms/api_server.py",
                      "gateway/platforms/api_server_room_dispatch.py", "gateway/platforms/api_server_runs.py"}
    changed_files = set(subprocess.check_output(["git", "-C", str(source), "diff", "--name-only"],
                                                text=True).splitlines())
    if changed_files != expected_files:
        raise ValueError("Hermes source has unexpected tracked modifications")
    diff = subprocess.check_output(["git", "-C", str(source), "diff", "--",
                                    *sorted(expected_files)])
    if diff != patch.read_bytes():
        raise ValueError("Hermes source does not match Lumen's pinned chat patch")
    hermes = source / ".venv/bin/hermes"
    if not hermes.is_file():
        raise ValueError("pinned Hermes virtual environment is missing")
    model_port, api_port = free_port(), free_port()
    observed = []
    provider = ThreadingHTTPServer(("127.0.0.1", model_port), provider_class(observed))
    worker = Thread(target=provider.serve_forever, daemon=True)
    worker.start()
    try:
        with ExitStack() as stack:
            home = stack.enter_context(tempfile.TemporaryDirectory(prefix="lumen-hermes-chat-"))
            shutil.copyfile(Path(__file__).with_name("chat-config.yaml"), Path(home) / "config.yaml")
            (Path(home) / "memories").mkdir()
            markers = {"AGENTS.md": "LUMEN_CONTEXT_FILE_SENTINEL",
                       "memories/MEMORY.md": "LUMEN_MEMORY_SENTINEL",
                       "memories/USER.md": "LUMEN_USER_SENTINEL"}
            for name, marker in markers.items():
                (Path(home) / name).write_text(marker)
            env = {"PATH": os.environ.get("PATH", ""), "HOME": os.environ.get("HOME", "")}
            env.update({"HERMES_HOME": home, "HERMES_SAFE_MODE": "1",
                        "LUMEN_CHAT_ZERO_TOOL": "1",
                        "TERMINAL_CWD": home,
                        "API_SERVER_ENABLED": "true", "API_SERVER_KEY": "synthetic-loopback-key-1234567890",
                        "API_SERVER_HOST": "127.0.0.1", "API_SERVER_PORT": str(api_port),
                        "API_SERVER_MODEL_NAME": "synthetic-model",
                        "HERMES_INFERENCE_PROVIDER": "custom", "HERMES_INFERENCE_MODEL": "synthetic-model",
                        "OPENAI_API_KEY": "synthetic-only", "OPENAI_BASE_URL": f"http://127.0.0.1:{model_port}/v1",
                        "OPENROUTER_API_KEY": "synthetic-only",
                        "OPENROUTER_BASE_URL": f"http://127.0.0.1:{model_port}/v1"})
            gateway = subprocess.Popen([str(hermes), "gateway", "run", "--no-supervise", "--force"],
                                       cwd=home, env=env, stdout=subprocess.DEVNULL,
                                       stderr=subprocess.DEVNULL)
            stack.callback(stop_gateway, gateway)
            api = f"http://127.0.0.1:{api_port}"
            deadline = time.monotonic() + 30
            while time.monotonic() < deadline:
                if gateway.poll() is not None:
                    raise RuntimeError("Hermes gateway exited before API readiness")
                try:
                    if request(api + "/health").get("status") == "ok":
                        break
                except (URLError, TimeoutError):
                    time.sleep(0.2)
            else:
                raise TimeoutError("Hermes API did not become ready")
            key = env["API_SERVER_KEY"]
            policy = {"version": 1, "target_profile": "default",
                      "enabled_toolsets": ["bot_room", "terminal"], "approval_mode": "manual",
                      "max_iterations": 1}
            digest = hashlib.sha256(json.dumps(policy, sort_keys=True,
                                               separators=(",", ":")).encode("ascii")).hexdigest()
            try:
                request(api + "/v1/runs", key,
                        {"input": "Synthetic room-policy denial probe.",
                         "hosted_room_dispatch": {},
                         "_room_execution_policy": {**policy, "policy_digest": digest}})
            except HTTPError as exc:
                if exc.code not in {400, 403}:
                    raise AssertionError("room-policy injection returned an unexpected status") from exc
            else:
                raise AssertionError("API key alone accepted a tool-enabled room policy")
            try:
                request(api + "/v1/runs", body={"input": "room grant route probe"},
                        authorization="HermesRoom deliberately-invalid-token")
            except HTTPError as exc:
                error = json.load(exc)
                if exc.code != 403 or error.get("error", {}).get("code") != "invalid_room_dispatch":
                    raise AssertionError("room-token route did not reach the dedicated chat denial") from exc
            else:
                raise AssertionError("room-token route accepted a chat run")
            old = "LUMEN_OLD_SESSION_SENTINEL"
            status = settled_run(api, key, {"input": "LUMEN_TERMINAL_PROBE " + old,
                                            "session_id": "lumen-probe-session",
                                            "conversation_history": [{"role": "user", "content": "canonical first"}]})
            if not observed or any(body.get("tools") for body in observed):
                raise AssertionError("model received tool definitions")
            wire = json.dumps(observed)
            if any(marker in wire for marker in markers.values()):
                raise AssertionError("Hermes context or memory leaked into the model request")
            if any((Path(home) / name).read_text() != marker for name, marker in markers.items()):
                raise AssertionError("Hermes changed a context or memory file")
            if status.get("status") != "failed" or status.get("error") != "agent run failed":
                raise AssertionError(f"forced-tool run was not rejected: {status.get('status')}, {status.get('error')!r}")
            if (Path(home) / "LUMEN_FORBIDDEN_SIDE_EFFECT").exists():
                raise AssertionError("forbidden terminal call executed")
            if any(message.get("role") == "tool" for body in observed for message in body.get("messages", [])):
                raise AssertionError("forbidden tool call reached a second model request")
            first_calls = len(observed)
            second = settled_run(api, key, {"input": "canonical second", "session_id": "lumen-probe-session",
                                            "conversation_history": [{"role": "user", "content": "canonical only"}]})
            if second.get("status") != "failed" or old in json.dumps(observed[first_calls:]):
                wire_second = json.dumps(observed[first_calls:])
                point = wire_second.find(old)
                raise AssertionError(f"Hermes replayed hidden session history or accepted a tool call: {second.get('status')}, {wire_second[max(0, point-150):point+150]}")
            if any(message.get("role") == "tool" for body in observed for message in body.get("messages", [])):
                raise AssertionError("forbidden session-search call reached a second model request")
            memory_write = settled_run(api, key, {"input": "LUMEN_MEMORY_WRITE_PROBE"})
            if memory_write.get("status") != "failed":
                raise AssertionError("forced memory write was accepted")
            if any((Path(home) / name).read_text() != marker for name, marker in markers.items()):
                raise AssertionError("forced memory write changed a context or memory file")
            normal = settled_run(api, key, {"input": "LUMEN_NORMAL_CHAT_PROBE"})
            if normal.get("status") != "completed" or normal.get("output") != "synthetic finish":
                raise AssertionError("ordinary zero-tool chat did not complete")
            time.sleep(0.5)
            if len(observed) != 4:
                raise AssertionError(f"unexpected auxiliary or follow-up model requests: {len(observed)}")
            return {"status": "probe_passed", "pinnedCommit": commit, "providerCalls": len(observed),
                    "advertisedTools": 0, "memoryDisclosure": False, "rejectedToolCalls": 3,
                    "hiddenSessionReplay": False}
    finally:
        provider.shutdown()
        provider.server_close()
        worker.join(timeout=2)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True, help="pinned Hermes Git checkout")
    args = parser.parse_args()
    print(json.dumps(probe(args.source)))
