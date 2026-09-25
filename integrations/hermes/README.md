# Hermes chat profile probe

`chat-config.yaml` is a candidate config for a **dedicated** Hermes API process at the pinned `v2026.9.7` source commit. Keep its `HERMES_HOME` separate from the general-purpose runtime and set `HERMES_SAFE_MODE=1`. The config excludes API tools, default MCP servers, built-in memory, and the external memory provider.

Run `check_chat_profile.py` using the installed Hermes Python interpreter, from its source/install environment, with the same `HERMES_HOME` and environment as that API process. The preflight resolves ordinary API toolsets and constructs an agent; it exits nonzero if it sees any tools or built-in/provider memory. It does not verify the installed source pin, bind to a running process, exercise API sessions or context files, or issue a runtime certificate.

The next gate is a real isolated API server negative probe: attempt a tool and a memory read/write through `/v1/runs`, check the grant-protected room route that can override API toolsets, then bind the verified source artifact, process, config, and endpoint identity to a short-lived Host certificate. Until that gate passes, Lumen does not send canonical conversation content.
