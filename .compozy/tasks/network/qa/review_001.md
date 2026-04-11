# QA Review 001

Date: 2026-04-11
Environment: isolated daemon with `AGH_HOME=.tmp/agh-network-qa-home`, `http.port=2324`, `network.port=4522`, provider `codex`
Status: resolved in this run

## Issue 001: Inbound network prompts omit the response guidance footer

Severity: high

Evidence:
- Real `say` delivery reached live sessions, but the persisted `session events` payload contained only the `<network-message ...>` XML wrapper.
- The tech spec example requires the footer `Use \`agh network send\` to respond. See \`agh network --help\` for options.`
- Without that footer, the auto-prompt gives agents less explicit response guidance than the designed UX.

Impact:
- Agents receiving inbound network turns are more likely to inspect the environment ad hoc instead of responding through the intended audited CLI path.
- This weakens the usability contract of the feature and makes real-world response behavior less reliable.

## Issue 002: Agent subprocesses resolve `agh` from host PATH instead of the daemon-matched binary

Severity: critical

Evidence:
- In live session events, a network-delivered turn executed `which agh` and resolved `/Users/pedronauck/.local/bin/agh`.
- The same live turn showed `agh --help` output from that older binary, which does not include `network`.
- Another live turn attempted `agh skill view agh-network` and failed with `unknown config keys ... network.*`, proving the agent was not using the same binary/config generation as the running daemon.

Impact:
- Real agents cannot reliably use `agh network ...` even when the daemon/runtime supports it, because shell resolution may hit a stale installation.
- This breaks the core network UX in multi-worktree and source-build setups, exactly the scenario exercised in this QA run.

## Issue 003: Network-participating agents do not receive their local session identity for replies

Severity: high

Evidence:
- A live receiver session that got an inbound network turn immediately started searching local runtime state to discover which daemon-local session id it should use for `agh network send --session ...`.
- The bundled `agh-network` skill requires `--session <local-session-id>` for `inbox` and `send`, but it does not provide a reliable, unambiguous way to obtain that value from inside the agent.
- Direct inspection of the live ACP subprocess environment showed `AGH_HOME` and `AGH_BIN`, but no `AGH_SESSION_ID`, `AGH_SESSION_SPACE`, or equivalent session-scoped identity variables.

Impact:
- A receiving agent cannot reply through the audited network CLI path without extra environment probing or heuristic `agh session list` matching.
- In workspaces with multiple active sessions in the same space, that reply path becomes ambiguous and brittle, weakening the intended agent UX of the feature.

## Issue 004: `agh daemon stop -o json` can report stale network status after shutdown

Severity: medium

Evidence:
- In the isolated QA daemon, `agh daemon stop -o json` returned `"status":"stopped"` while still including `"network":{"enabled":true,"status":"running","listener_host":"127.0.0.1","listener_port":4522}`.
- A fresh `agh daemon status -o json` immediately afterward correctly returned a stopped daemon with `pid: 0` and no network block.

Impact:
- The one-shot stop response can mislead users and automation into thinking the embedded network runtime is still active after shutdown.
- This weakens the observability contract of the control plane during operational stop/restart workflows.

## Resolution Notes

- Issue 001 fixed in `internal/network/delivery.go` and covered in `internal/network/delivery_test.go`.
  - Live revalidation: a fresh direct delivery to `qa-receiver` persisted the expected wrapper plus the footer `Use \`agh network send\` to respond. See \`agh network --help\` for options.`
- Issue 002 fixed in `internal/acp/client.go` and covered in `internal/acp/client_test.go`.
  - Live revalidation: a post-fix probe session resolved `agh` to `/Users/pedronauck/Dev/projects/_worktrees/network/bin/agh`, and `agh network status -o json` succeeded from inside the agent runtime.
- Issue 003 fixed in `internal/session/manager_start.go`, `internal/session/manager_test.go`, and `internal/skills/bundled/skills/agh-network/SKILL.md`.
  - Live revalidation: active ACP subprocesses now expose `AGH_SESSION_ID`, `AGH_SESSION_SPACE`, and `AGH_PEER_ID` for network-participating sessions, while non-space sessions expose only `AGH_SESSION_ID`.
- Issue 004 fixed in `internal/cli/daemon.go` and covered in `internal/cli/daemon_wait_test.go` plus `internal/cli/command_paths_test.go`.
  - Live revalidation: a clean rebuilt `bin/agh daemon stop -o json` now returns a stopped payload without a stale `network` block, and a follow-up `daemon status -o json` matches that stopped state.

## Additional QA Coverage

- Verified `agh network peers builders -o json` and `agh network spaces -o json` reflect active opted-in sessions only.
- Verified a non-space session is rejected by `agh network send` with `network: local peer not found`.
- Verified busy-session gating with real runtime behavior:
  - two direct messages queued in `agh network inbox --session <busy-session>`
  - after the active turn completed, the first queued message was delivered as a new network turn
  - the second message remained queued behind that new inbound turn, preserving serialized FIFO delivery semantics
