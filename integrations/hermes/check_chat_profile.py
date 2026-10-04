"""Preflight a Hermes installation's ordinary API-agent chat configuration.

Run with the same HERMES_HOME and environment as the dedicated API server.
This does not exercise a live API run or qualify the installed artifact.
"""

import json
import os
import sys


def check():
    if os.environ.get("HERMES_SAFE_MODE") != "1":
        raise ValueError("Hermes plugins are not disabled")

    from hermes_cli.config import load_config
    from hermes_cli.tools_config import _get_platform_tools
    from run_agent import AIAgent

    config = load_config()
    if (config.get("platform_toolsets") or {}).get("api_server") != ["no_mcp"]:
        raise ValueError("API server toolsets are not restricted")
    memory = config.get("memory") or {}
    if (memory.get("memory_enabled") is not False
            or memory.get("user_profile_enabled") is not False
            or memory.get("provider")):
        raise ValueError("Hermes memory is not disabled")
    if (config.get("context") or {}).get("engine", "compressor") != "compressor":
        raise ValueError("Hermes context engine is not restricted")

    toolsets = sorted(_get_platform_tools(config, "api_server"))
    if toolsets:
        raise ValueError("API server resolved nonempty toolsets")

    # Construct the same agent type as the API server without making a model call.
    agent = AIAgent(
        model="lumen-profile-check", provider="custom",
        base_url="http://127.0.0.1:1/v1", api_key="synthetic-probe-only",
        enabled_toolsets=toolsets, platform="api_server", quiet_mode=True,
    )
    if (agent.tools or agent.valid_tool_names or agent._memory_enabled
            or agent._user_profile_enabled or agent._memory_store is not None
            or agent._memory_manager is not None):
        raise ValueError("API agent has tools or memory")


if __name__ == "__main__":
    try:
        check()
    except Exception as exc:
        print(json.dumps({"status": "unqualified", "reason": str(exc)}))
        sys.exit(1)
    print(json.dumps({"status": "preflight_passed", "constructedAgentTools": 0,
                      "constructedAgentMemory": False}))
