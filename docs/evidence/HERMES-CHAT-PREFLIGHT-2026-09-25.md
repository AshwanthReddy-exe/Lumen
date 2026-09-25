# Hermes chat preflight, 2026-09-25

- **Source:** pinned Hermes tag `v2026.9.7`, commit `2237be355906fbe6065ce1815711eee52b2d646e`, `uv sync --frozen`; API health reported `0.21.1`.
- **Topology:** disposable local Hermes gateway and synthetic model provider on loopback, separate `HERMES_HOME`; no personal content or credentials.
- **Configuration:** [`chat-config.yaml`](../../integrations/hermes/chat-config.yaml), `HERMES_SAFE_MODE=1`.
- **Preflight:** [`check_chat_profile.py`](../../integrations/hermes/check_chat_profile.py) returned `preflight_passed`, zero constructed-agent tools, and no built-in or provider memory. Disabling safe mode or enabling memory caused a nonzero `unqualified` result.
- **Live ordinary Runs probe:** synthetic provider sent a `memory` tool call despite the configured empty toolset. `/v1/runs` completed with synthetic text and its public event stream contained only message/reasoning and terminal events. No `MEMORY.md` or `USER.md` appeared in the disposable home. This does not establish whether an internal invalid-tool result was generated.
- **Result:** preliminary preflight only. It does not bind source and config to the running endpoint, prove session/context isolation, or cover the room-policy override route. **No production chat certificate was issued.**
