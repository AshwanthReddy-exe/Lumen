# Platform and topology coverage contract

Lumen's Space identity is platform-independent. A platform is supported only for a **named role**, OS/build, architecture, Host topology and evidence revision. Cross compilation, emulator tests and web rendering establish parts of a row, never the complete claim. The candidate rows below are planned, not currently certified.

| Platform/role | Planned first route | Owning plan/gate | Physical acceptance before advertising |
| --- | --- | --- | --- |
| macOS Host | Native Go binary, interactive setup, user-login LaunchAgent where proven | 10-01/02 | Fresh one-command install, authenticated doctor, login/reboot, update/rollback, backup/restore and real provider Run. Pre-login availability is a separate unsupported claim unless proven. |
| Linux/VPS Host | Native Go with systemd and declared container profile | 10-01/02/03 | Clean VPS install, anonymous pinned-image pull where used, real reboot, external/combined Hermes route, update/rollback and network/auth failures. |
| Android Termux Host | Native Go foreground/service profile only if validated | 10-01/02 and 15-01 matrix | Exact phone/Android/Termux build, restart/service and storage/thermal checks; otherwise do not advertise Host role. |
| Mac capability node | Swift node with Keychain key, selected-root file read and receipt store | 07-01/02/03 | Physical separate-source→Mac action, local permission denial, symlink/rename/restart/revocation/lost-ACK cases. |
| Android conversation client | D-056 shared client if physical gate passes, else Kotlin shell plus responsive phone web | 05-01/04, 06-03 | Named device key/pairing, outbox process death, approval, replay, foreground audio bridge, network change. |
| iPhone conversation client | D-056 shared client if physical gate passes, else authenticated responsive phone web | 05-01/04, 06-03 | Named iPhone/OS browser or native build, pairing/auth, send/receipt, replay, approval and foreground voice as claimed. Background custom wake is not promised. |
| Android desk voice node | Existing Android shell plus licensed local wake/STT/TTS where E6/E7 pass | 08-01/02 | Physical wake counts, eight-hour negative audio, zero pre-wake egress, mute/stop, noisy speech, battery/thermal. Tap-to-talk is the fallback. |
| Windows capability node | Windows-specific local key/grant/file/receipt adapter | 13-04 | Native/VM Windows tests plus real pairing, ACL/path escape, restart, update/stop and cross-node receipt; unsupported until adapter and evidence exist. |
| Linux capability node | Linux-specific key/grant/file/receipt adapter | 13-04 | Native Linux tests plus file permissions, descriptor-relative path denial, restart, update/stop and cross-node receipt. |
| Browser client | Authenticated responsive web UI against Host APIs | 03-04, 05/06, 09 | Real browser send/replay/stop, accessibility, session/CSRF denial and reconnect on claimed desktop/phone OS. |

At Phase 10 alpha, publish only rows with required functionality and live evidence. Phase 15 rechecks every paid `(OS, arch, topology, assurance profile)` row with an independent clean-machine owner; Phase 16 managed is a separate topology; Phase 17 public SDK does not imply a device platform is supported.

For each row, evidence must record exact device/model, OS/build/architecture, Lumen revision/artifact digest, Hermes/provider version, network/Host placement, test commands, redacted receipt IDs and result. A missing row is `BLOCKED`, a technically impossible boundary is `UNSUPPORTED`, and neither is reported as PASS.
