# Phase 02 plan 02 evidence — in progress

| Check | Result | Evidence and limit |
| --- | --- | --- |
| Pinned GHCR image visibility | Pass on 2026-09-27 from macOS arm64 for both claimed platforms | With separate newly created, empty Docker client configs and an explicit Docker Desktop socket, `docker pull` of `ghcr.io/ashwanthreddy-exe/lumen-hermes@sha256:314d1e03ec846a6b0b0d81381ceca84b59ad4acd0101ae301f9879c5428ac103` succeeded without registry credentials for native `linux/arm64` and explicit `--platform linux/amd64`. Both pulls reported the exact requested digest; `docker image inspect` confirmed the arm64 RepoDigest. The first arm64 attempt with an empty config failed before registry contact only because Docker Desktop's socket path was absent from that config. The successful retries set `DOCKER_HOST` to the already verified local socket. No image visibility setting was changed by this test. |
| Linux clean installation and reboot | Unverified in this run | Historical Azure evidence is dated; no current Linux host was selected. |
| Real executable update, failed replacement and rollback | Unverified in this run | Automated release tests do not substitute for a named-machine lifecycle. |

The anonymous pulls resolve the former private-image observation for this pinned digest and its two claimed Linux platforms. They do not qualify updates, other digests or the complete Phase 02 distribution gate.
