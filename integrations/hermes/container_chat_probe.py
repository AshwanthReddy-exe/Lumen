"""Probe a disposable dedicated chat container against a synthetic model."""

import argparse
import hashlib
from http.server import ThreadingHTTPServer
from http.client import RemoteDisconnected
import json
from pathlib import Path
import subprocess
from threading import Thread
import time
from urllib.error import HTTPError, URLError

from live_chat_probe import free_port, provider_class, request, settled_run


def docker(*args):
    return subprocess.check_output(["docker", *args], text=True).strip()


def probe(image):
    model_port, api_port = free_port(), free_port()
    observed = []
    provider = ThreadingHTTPServer(("0.0.0.0", model_port), provider_class(observed))
    worker = Thread(target=provider.serve_forever, daemon=True)
    worker.start()
    container = ""
    try:
        config = Path(__file__).with_name("chat-config.yaml").resolve()
        config_digest = "sha256:" + hashlib.sha256(config.read_bytes()).hexdigest()
        container = docker("run", "--detach", "--rm", "--read-only", "--user", "65532:65532",
                           "--add-host", "host.docker.internal:host-gateway",
                           "--tmpfs", "/var/lib/hermes:rw,uid=65532,gid=65532,mode=0700",
                           "--mount", f"type=bind,src={config},dst=/var/lib/hermes/config.yaml,readonly",
                           "--publish", f"127.0.0.1:{api_port}:8642",
                           "--env", "HERMES_HOME=/var/lib/hermes", "--env", "TERMINAL_CWD=/var/lib/hermes",
                           "--env", "HERMES_SAFE_MODE=1", "--env", "LUMEN_CHAT_ZERO_TOOL=1",
                           "--env", "API_SERVER_ENABLED=true", "--env", "API_SERVER_HOST=0.0.0.0",
                           "--env", "API_SERVER_PORT=8642", "--env", "API_SERVER_KEY=synthetic-container-key",
                           "--env", "API_SERVER_MODEL_NAME=synthetic-model",
                           "--env", "HERMES_INFERENCE_PROVIDER=custom",
                           "--env", "HERMES_INFERENCE_MODEL=synthetic-model",
                           "--env", "OPENAI_API_KEY=synthetic-only",
                           "--env", f"OPENAI_BASE_URL=http://host.docker.internal:{model_port}/v1",
                           "--env", "OPENROUTER_API_KEY=synthetic-only",
                           "--env", f"OPENROUTER_BASE_URL=http://host.docker.internal:{model_port}/v1",
                           image, "gateway", "run", "--no-supervise", "--force")
        image_id = docker("image", "inspect", "--format", "{{.Id}}", image)
        details = json.loads(docker("inspect", container))[0]
        env = set(details["Config"]["Env"])
        mounts = details["Mounts"]
        if (details["Image"] != image_id or details["Config"]["User"] != "65532:65532"
                or not details["HostConfig"]["ReadonlyRootfs"]
                or not {"HERMES_SAFE_MODE=1", "LUMEN_CHAT_ZERO_TOOL=1"} <= env
                or not any(m["Destination"] == "/var/lib/hermes/config.yaml" and not m["RW"] for m in mounts)):
            raise AssertionError("container image, process configuration, or read-only chat mount changed")
        api = f"http://127.0.0.1:{api_port}"
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            try:
                if request(api + "/health").get("status") == "ok":
                    break
            except (URLError, TimeoutError, RemoteDisconnected):
                time.sleep(0.2)
        else:
            logs = docker("logs", "--tail", "20", container)
            raise TimeoutError("chat container did not become ready: " + logs[-1000:])
        key = "synthetic-container-key"
        for marker in ("LUMEN_TERMINAL_PROBE", "LUMEN_MEMORY_WRITE_PROBE", "LUMEN_SESSION_SEARCH_PROBE"):
            result = settled_run(api, key, {"input": marker,
                                            "conversation_history": [{"role": "user", "content": "canonical only"}]})
            if result.get("status") != "failed":
                raise AssertionError(f"container accepted forced tool call: {marker}")
        try:
            request(api + "/v1/runs", body={"input": "room route"},
                    authorization="HermesRoom deliberately-invalid-token")
        except HTTPError as exc:
            if exc.code != 403 or json.load(exc).get("error", {}).get("code") != "invalid_room_dispatch":
                raise AssertionError("container room route did not deny before grant validation") from exc
        else:
            raise AssertionError("container accepted room route")
        normal = settled_run(api, key, {"input": "LUMEN_NORMAL_CHAT_PROBE"})
        if normal.get("status") != "completed" or normal.get("output") != "synthetic finish":
            raise AssertionError("container ordinary chat failed")
        time.sleep(0.5)
        if len(observed) != 4 or any(body.get("tools") for body in observed):
            raise AssertionError("container made unexpected model requests or advertised tools")
        return {"status": "container_probe_passed", "imageId": image_id, "configDigest": config_digest,
                "providerCalls": len(observed), "rejectedToolCalls": 3}
    finally:
        if container:
            subprocess.run(["docker", "stop", "--time", "5", container], stdout=subprocess.DEVNULL,
                           stderr=subprocess.DEVNULL, check=False)
        provider.shutdown()
        provider.server_close()
        worker.join(timeout=2)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--image", required=True, help="locally built pinned chat image")
    print(json.dumps(probe(parser.parse_args().image)))
