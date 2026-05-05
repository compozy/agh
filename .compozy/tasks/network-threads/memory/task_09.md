# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 09 CLI control-plane support for Task 08 conversation routes:
  `network threads`, `network directs`, `network work`, and revised `network send`
  flags.
- Completion requires structured output coverage, legacy flag/kind rejection tests,
  clean `make verify`, task tracking updates, and one local commit.

## Important Decisions
- Use Task 08 public routes and DTOs as the CLI source of truth.
- Use exact TechSpec send flags `--thread`, `--direct`, and `--work`; do not keep
  `--interaction-id` and do not accept `--kind direct`.
- Treat the existing `--thread-id`/`--direct-id`/`--work-id` CLI flags as
  pre-Task-09 drift to replace with the TechSpec names.

## Learnings
- Task 08 already added HTTP/UDS routes for:
  `/api/network/channels/{channel}/threads`,
  `/threads/{thread_id}/messages`, `/directs`, `/directs/resolve`,
  `/directs/{direct_id}/messages`, and `/api/network/work/{work_id}`.
- Pre-change CLI signal: `agh network` only exposes `status`, `peers`,
  `channels`, `send`, and `inbox`; `network send --help` shows
  `--thread-id`, `--direct-id`, and `--work-id`.
- Focused unit CLI tests now cover JSON/jsonl/toon output for new thread,
  direct, and work commands plus send payload construction and hard-cut flag
  rejection.
- Focused integration coverage now exercises daemon-served thread list/show/
  messages, direct resolve/list/show/messages, work lookup, and raw claim-token
  rejection.
- `make verify` exposed stale Task-09-adjacent expectations in
  `internal/cli/command_paths_test.go` and bundled `agh-network` skill content;
  both were updated to use `--thread`, `--direct`, and `--work`.
- Fresh full verification passed after the stale expectations were corrected.

## Files / Surfaces
- Touched code surfaces: `internal/cli/network.go`, `internal/cli/client.go`,
  `internal/cli/network_test.go`, `internal/cli/network_client_test.go`,
  `internal/cli/cli_integration_test.go`, `internal/cli/command_paths_test.go`,
  CLI test helpers, bundled `agh-network` skill content, and its bundled tests.

## Errors / Corrections
- Initial shell context was `/Users/pedronauck/dev/compozy/looper`, but the task
  repo and PRD memory are `/Users/pedronauck/Dev/compozy/agh2`; switched all code
  discovery and edits to the AGH repo before implementation.
- Integration retry assertion was corrected to match the current runtime contract:
  exact outbound duplicate message IDs are idempotent and do not increment
  `MessagesRejected`; the CLI integration now asserts one queued/accepted direct
  message instead.

## Ready for Next Run
- Focused checks passed:
  - `go test ./internal/cli -run 'TestNetwork' -count=1`
  - `go test -tags integration ./internal/cli -run 'TestCLINetwork' -count=1`
  - `go test ./internal/cli -run 'TestCommandPathsAndHelpers|TestNetwork' -count=1`
  - `go test ./internal/skills/bundled -run 'TestBundledAghNetworkSkillContent' -count=1`
- Full gate passed:
  - `make verify`
- Remaining: final diff self-review and local commit.
