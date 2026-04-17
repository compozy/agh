# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add composition-root daemon runtime E2E for network direct reply lifecycle and whois/recipe exchange using the shared subprocess-backed runtime harness and fixture-backed ACP mock agents.
- Keep assertions on public product surfaces plus daemon-owned artifacts: network messages, network audit, API projections, CLI visibility, and selected transcript/events when they clarify the scenario.

## Important Decisions
- Reuse `internal/testutil/e2e/` as the only runtime boot seam instead of adding package-local boot helpers in `internal/daemon`.
- Treat network correlation truth as a daemon/runtime responsibility; browser work should later consume this lane rather than re-prove RFC semantics.
- Persist network enablement into the seeded config file and let the real daemon read it back, instead of relying on in-memory config mutations alone.
- Drive the collaboration exchanges through the shipped `agh network ...` CLI while using the mock agents as deterministic network-delivered recipients.

## Learnings
- The shared harness needed explicit `EnableNetwork` support plus typed helpers for network status, peers, channels, channel detail, channel messages, inbox, send, and audit reads so composition-root scenarios can assert through public surfaces instead of bespoke JSON calls.
- The original idea of having the fixture issue nested `agh network ...` sends through terminal `environment_exec` was not reliable under the real runtime path; the stable shape is to send through the runtime test's real CLI surface and let the fixtures behave as deterministic recipients.
- Stable network-audit assertions are easier to review when the artifact helper rewrites daemon JSONL audit logs into ordered JSON arrays for each scenario run.

## Files / Surfaces
- `internal/testutil/e2e/config_seed.go`
- `internal/testutil/e2e/config_seed_test.go`
- `internal/testutil/e2e/runtime_harness.go`
- `internal/testutil/e2e/runtime_harness_helpers_test.go`
- `internal/testutil/e2e/runtime_harness_test.go`
- `internal/testutil/acpmock/testdata/`
- `internal/daemon/daemon_network_collaboration_integration_test.go`
- `internal/daemon/network_e2e_assertions_test.go`
- `/api/network/status`
- `/api/network/peers`
- `/api/network/channels`
- `/api/network/channels/{channel}`
- `/api/network/channels/{channel}/messages`
- `/api/network/inbox`
- real CLI `agh network ...`

## Errors / Corrections
- Corrected the harness seeding path so `[network]` config is actually written to disk before the daemon boots.
- Replaced raw JSONL audit artifact snapshots with stable decoded JSON arrays.
- Moved message initiation out of brittle mock `environment_exec` sends and into the runtime test's real CLI surface after the nested terminal path proved unreliable.

## Ready for Next Run
- Task complete after clean `make verify`: composition-root daemon runtime E2E now covers direct reply lifecycle plus whois/recipe exchange with correlation, duplicate rejection, channel history, peer visibility, audit capture, transcript checks, and CLI/API parity.
