# Plan 02-08 evidence — real provider qualification

Date: 2026-10-02

**Initial observation outcome: BLOCKED.** At this checkpoint no real-provider request had been sent. A read-only loopback health request succeeded on 2026-10-02 and identified Hermes `0.20.6`; `hermes gateway status` reports the launchd service is not loaded and the process is detached. The Lumen Host socket is absent. This is not the pinned Hermes profile declared by the Phase 02 plans, and its configured provider identity was not confirmed. Shell environment credential presence checks found no provider API key variables; credential file contents and Hermes configuration secrets were not read or emitted. The historical `provider_authentication_failed` result remains candidate evidence only. A later owner-approved one-shot against an isolated pinned profile is recorded in the addendum below.

At this earlier checkpoint, `scripts/lumen-mac-test` could not run because the Lumen Host socket was absent. The responding `0.20.6` endpoint was not queried for authenticated capabilities, and no profile/configuration was changed. No answer, task receipt, provider failure classification, or successful persistence was claimed then. Later approved request and current pinned-source credential trace are recorded below; the one-shot authorization is consumed and no further provider request should be sent without fresh owner authorization.

| Check | Result | Limitation |
|---|---|---|
| Real profile/provider identity | BLOCKED | Health confirms Hermes `0.20.6`; detached/unloaded service is not the pinned Lumen profile, and authenticated capabilities/provider identity remain unverified. |
| Host-mediated provider answer | BLOCKED | No provider request sent. |
| Honest real-provider failure | BLOCKED | No request sent and no new failure result observed. |
| Credential handling | PASS for this audit | Secret values were not read, emitted, or persisted. |

Synthetic-provider journeys in `02-01-EVIDENCE.md` remain synthetic and do not satisfy this plan.

## Addendum — isolated pinned profile attempt (2026-10-02)

A temporary Hermes container was started from pinned image `sha256:ae25a828a487c8950ed02e0bcd43dacbe37c22b54cfc1a80c953a0752d0b56bf` (Hermes 0.21.1). Its private config selected the configured custom provider and `deepseek-v4-flash`; a read-only `/v1/toolsets` check confirmed zero enabled API toolsets before the public Host task script was invoked. A temporary Host reached its local ready state. The Host task script returned nonzero, but the runner discarded the task receipt/status when handling that result. It remains unknown whether that first request dispatched or what durable outcome the Host recorded. A separate later owner-approved one-shot is documented below; it does not recover this first attempt.

Because dispatch may have occurred and the task ID/result was not retained, this one-shot attempt is not retried automatically; a retry needs the owner's decision to avoid duplicating a potentially billable operation. The temporary Host state, copied provider config/environment, container, and volume were removed. A bounded post-check found no matching temporary container, volume, or scratch directory. No prompt, assistant output, or credential was printed or added to evidence.

| Check | Result | Limitation |
|---|---|---|
| Pinned Hermes version/image and selected profile | PASS for preflight only | Image and configured model/provider were checked; a model response was not confirmed. |
| Hermes API tools disabled | PASS for preflight only | Live toolset list had no enabled entries before invoking the Host script. |
| Host-mediated provider answer | BLOCKED | Script returned nonzero; task dispatch and durable result were not retained. |
| Honest provider failure | BLOCKED | No reliable Host task status/receipt was captured. |
| Credential handling | PASS for observed output | Credentials stayed in private temporary config; no values or answer content were emitted. |

## Receipt-path hardening after the attempt

The macOS Host journey now emits a redacted receipt containing task ID, initial accepted state (`awaiting_permission` or `queued`), and Host ID immediately after the submit response; it emits a second receipt at terminal completion/failure. A contract assertion confirms accepted and terminal receipts appear in order. The earlier attempt ran before this change, and its discarded output cannot be reconstructed; dispatch and result remain unknown, so no provider PASS or failure classification is added. Four focused contract tests, shell syntax, and `git diff --check` pass for the current source.

## Addendum — owner-approved one-shot with durable outcome (2026-10-02)

The owner explicitly approved one harmless request to `agentrouter.org`. One request was sent through the isolated pinned Hermes 0.21.1 profile and temporary Host. The preflight observed authenticated capabilities and zero enabled API toolsets. The profile selected custom provider `deepseek-v4-flash`; its temporary config digest was `d280d628742e7bde92ee8f343d91cfc2ff070a966258a7ce6510fd72ebc20570`. The API key was present in the private source config (40 bytes); its value and fingerprint are deliberately omitted from durable evidence.

The Host emitted an accepted receipt and later a durable terminal receipt for task `mac-hermes-test-1790941638-29071-fde44794` on Host `host-e65ba72522832d61fca464f87bef53e702cfe4452ccd481f56067b9dff76a931`: `awaiting_permission` → `failed`. The durable `task show` result agreed. Pinned Hermes logs contained `AuthenticationError` and HTTP 401. This classifies the attempt as provider authentication rejection; it does not identify whether the key is invalid/expired or rejected by provider policy. The assistant output was discarded and no answer is claimed. The request authorization is consumed; do not retry absent a new owner authorization.

The harness cleanup trap itself errored while assigning zsh's read-only `status` variable. Cleanup was then performed explicitly: the exact temporary Host was stopped and its container, named volume, and scratch directory removed; post-checks found no matching process, container, volume, or directory. This successful targeted cleanup does not make the harness trap pass.

| Check | Result | Limitation |
|---|---|---|
| Pinned profile and disabled API toolsets | PASS for preflight | Exact temporary image/config only; not general runtime certification. |
| Host-mediated assistant answer | BLOCKED | Provider returned HTTP 401; no answer received. |
| Honest real-provider failure | PASS | Accepted and terminal Host receipts agree on `failed`; provider log classification is HTTP 401. |
| Credential handling | PASS for observed output | Key value and answer were not emitted or retained. |
| Automatic harness cleanup | FAIL | zsh `status` assignment errored; exact resources were manually removed and independently checked. |

## Offline credential-resolution trace (2026-10-02)

The pinned Hermes 0.21.1 image source was inspected in a read-only, network-disabled container. `gateway/run.py` loads the Hermes home `.env` before gateway runtime setup; `hermes_cli/config.py` expands `${ENV}` references; `runtime_provider_backends.py` selects the configured `model.api_key` for the explicit bare-custom endpoint. A disposable profile with a fabricated key exercised that exact chain locally: `.env` lookup, placeholder expansion, and runtime-selected key all matched the dummy value, and the selected host was `example.invalid`. No model/provider request was made by this check, and no real credential was included.

This rules out a generic pinned-build failure to load `.env`, expand the configured placeholder, or select the configured custom key. It does not prove what bytes reached the remote endpoint in the approved request, or whether the key, account, model entitlement, or provider policy caused HTTP 401. The evidence therefore remains **provider authentication rejection; specific cause unknown**. No code propagation defect was demonstrated, so no auth behavior was changed.

## Addendum — second owner-authorized attempt, outcome not recoverable (2026-10-02)

After the owner confirmed the provider key had changed and authorized exactly one harmless request to `agentrouter.org`, an isolated Host and the same pinned Hermes image were started from a temporary profile. Preflight read only the provider/model/endpoint identity and key presence/byte length (51); authenticated capabilities succeeded and the API toolset inventory was required to contain no enabled toolsets. The temporary Host reached `ready`. The harness accepted only the Host submit outcomes `queued` or `awaiting_permission` and proceeded to its bounded terminal-status poll; however, it assigned to zsh's read-only special variable `status` immediately before polling and aborted. The temporary Host was stopped before it could record a task-status file. A Host restart against the retained temporary state returned `task_unknown`; no durable receipt or provider response can be recovered. The provider request may have dispatched, so the single authorization is consumed and this request is not retried.

The exact temporary Hermes container and volume and the scratch directory were then removed and verified absent. No credential or assistant output was emitted or retained. This is **BLOCKED / outcome unknown**, not a real-answer pass and not evidence of a provider failure. The harness defect is a local variable-name collision (`status`); any future authorized retry must use a non-special variable name and persist the redacted accepted receipt before polling.

| Check | Result | Limitation |
|---|---|---|
| Updated local provider config identity | PASS for preflight | `custom`, `deepseek-v4-flash`, `agentrouter.org`; key presence and byte length only. |
| Pinned Hermes API toolset preflight | PASS for preflight | The isolated run required an empty enabled-tool inventory; no model-originated tool-injection probe was performed. |
| Host-mediated answer | BLOCKED | Request may have dispatched; no assistant answer was retained. |
| Durable Host receipt and terminal outcome | BLOCKED | Harness aborted before the first task poll; later lookup returned `task_unknown`. |
| Cleanup | PASS | Exact temporary container, volume, and scratch directory were removed and absence verified. |
| Credential/output handling | PASS for observed output | No credential value or assistant response was emitted or retained. |

## Addendum — NVIDIA NIM Host qualification (2026-10-02)

The owner directed Hermes to use NVIDIA NIM and keep `nvidia/nemotron-3.5-lightning-30b-a3b` as its default because AgentRouter was not working. The user's private `~/.hermes/config.yaml` was updated to provider `nvidia` and that model; the obsolete AgentRouter `base_url` and `api_key` fields were removed, while unrelated settings and the file's `0600` mode were preserved. The pinned Hermes source archive at commit `2237be355906fbe6065ce1815711eee52b2d646e` contains the NVIDIA provider overlay, its NVIDIA NIM endpoint, and the `nvidia` provider alias.

Using a private temporary profile with the same NVIDIA model and local `NVIDIA_API_KEY` environment reference, pinned Hermes image `sha256:f5db4a2c0e08887bb2beadabc1ccb3a10987307738a4ceab62946f46710ea2d2` passed authenticated capabilities and an empty enabled API-tool inventory. A fresh temporary local Host reached ready and submitted one harmless prompt with the `development` loopback profile. The Host accepted task `mac-hermes-test-nim-1790944213-47068` on Host `host-e951044a06b1327387dbbc87adf6621ba7c12986782bfc06d86330006835dc07` as `awaiting_permission`; after one-time approval, its durable terminal state was `completed` with non-empty output. The answer text was discarded and not assessed for quality. Only task/Host metadata and the non-empty-output boolean were retained. The API key was present (70 bytes); its value and digest are omitted.

The corrected Bash harness captured the accepted receipt before polling and used no zsh special variable. The exact temporary Hermes container, volume, and scratch directory were absent after completion. This is a **PASS for one real NVIDIA NIM-backed Host result on the named pinned image/profile**; it does not prove response quality, latency, ongoing key validity, or a model-originated attempted-tool denial. The earlier AgentRouter HTTP 401 and later AgentRouter outcome-unknown attempt remain historical and do not overwrite this successful NIM result.

| Check | Result | Limitation |
|---|---|---|
| Hermes configured provider/model default | PASS | Owner config validates as NVIDIA NIM with the requested model; old custom endpoint/key fields are absent. |
| Pinned runtime and no-tool inventory | PASS for preflight | Exact image and ephemeral config; no model-originated attempted-tool probe. |
| Host-mediated real provider response | PASS | Durable `awaiting_permission` → `completed`; non-empty output; content discarded and quality not assessed. |
| Durable task/Host mapping | PASS for this task | Receipt IDs recorded; the one response survived through Host terminal persistence. This does not exercise restart recovery. |
| Credential handling | PASS for observed output | Key presence/length only; no key value, fingerprint, prompt, or response content retained. |
| Cleanup | PASS | Exact container, volume, and temporary directory checked absent. |

## Owner confirmation — NIM remains the default (2026-10-02)

The owner reconfirmed NVIDIA NIM model `nvidia/nemotron-3.5-lightning-30b-a3b` as the Hermes default because AgentRouter is not working. Read-only inspection confirmed the private Hermes config still has `model.provider: nvidia` and the exact requested `model.default`; its mode remains `0600`, and AgentRouter endpoint/key fields are absent. No config edit or provider request was needed. This confirms the saved default only; the separately recorded active Hermes 0.20.6 service is unpinned and has no Phase 02 runtime qualification claim.


## Follow-up config correction — 2026-10-02

After the owner reiterated the NVIDIA NIM default, the private Hermes config was updated to remove the two legacy AgentRouter custom-provider entries; the NIM `model.provider` and `model.default` were preserved, all unrelated settings were preserved, and file mode remained `0600`. Ruby's YAML parser accepted the edited file. Hermes 0.20.6 `doctor` recognized the config version and configured provider; its NVIDIA NIM reachability check returned a DNS-resolution error under the restricted shell. No model-generation request was made. This is not evidence of NIM outage or successful pinned-Lumen runtime connectivity.


## Follow-up private config result — 2026-10-02

After the owner reiterated the NVIDIA NIM default, the private Hermes config was updated to remove the two legacy AgentRouter custom-provider entries while preserving the selected `model.provider: nvidia`, `model.default: nvidia/nemotron-3.5-lightning-30b-a3b`, unrelated settings, and file mode `0600`. YAML parsing passed. Hermes 0.20.6 `doctor` recognized the config version/provider, but its NVIDIA NIM reachability check returned a DNS-resolution error under the restricted shell. No model-generation request was made; this is neither a provider outage finding nor evidence for the pinned Lumen runtime.
