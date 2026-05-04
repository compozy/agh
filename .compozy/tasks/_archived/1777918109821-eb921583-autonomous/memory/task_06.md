# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 06: UDS/CLI `agh me`, `agh me context`, and `agh ch` list/recv/send/reply using Task 05 caller identity and Task 04 situation context.
- Acceptance includes identity validation, coordination metadata/message-kind validation, raw `claim_token` rejection, JSON/JSONL output, long-poll receive behavior, operator network regression coverage, `make verify`, tracking updates, and one local commit.

## Important Decisions
- Keep agent channel verbs as wrappers over the existing local network service; do not alter the broader network protocol or operator `agh network ...` semantics.
- Resolve `agh ch reply --to-message` from the caller's current inbox first. Existing persisted network timeline rows do not retain envelope extension metadata, so durable reply lookup beyond queued/inbox messages is follow-up scope unless storage changes are explicitly approved by a later task.
- `agh ch reply` always produces `message_kind=reply`; explicit non-reply metadata is rejected before send.
- `agh ch recv --wait` is backed by delivery-coordinator waiters that wake on accepted inbound messages instead of sleeps or shell polling.

## Learnings
- Task 02 already added the contract/OpenAPI DTOs for agent context and channels, but Task 06 must wire runtime UDS handlers and CLI methods.
- Task 04 `situation.Service.ContextForSession` already produces the stable `/agent/context` payload order and should be reused directly.
- Task 05 wired `/api/agent/me`, `DaemonClient.AgentMe`, `resolveAgentCallerFromEnv`, and `BaseHandlers.requireAgentCaller`; agent commands/endpoints should build on those paths.
- Baseline signal: `rg` found no wired `/agent/context` or `/agent/channels` UDS routes and no CLI `me`/`ch` commands before Task 06 edits.
- Runtime UDS routes are mounted under `/api/agent/...`; the PRD shorthand `/agent/...` maps to those local UDS HTTP paths.
- CLI zero-value reply metadata encodes as an object, so the UDS reply decoder treats an all-empty metadata object as absent and resolves metadata from the source message.

## Files / Surfaces
- Expected surfaces: `internal/api/core`, `internal/api/udsapi`, `internal/cli`, `internal/network`, daemon server wiring, tests, workflow/task tracking.
- Touched code surfaces: `internal/api/core/agent_channels.go`, `agent_identity.go`, handler interfaces/config, UDS route/server wiring, CLI client/root/format/agent commands, network delivery/manager wait support, daemon UDS injection, and focused tests.

## Errors / Corrections
- Corrected handler field naming to avoid `AgentContext` method/field collision.
- Corrected reply handling so CLI zero metadata can be resolved server-side and explicit non-reply kinds are rejected.
- Updated UDS route registry test after adding five agent routes.

## Ready for Next Run
- Focused verification passed: `go test ./internal/api/core ./internal/api/udsapi ./internal/cli ./internal/network ./internal/daemon`.
- Coverage evidence captured: `go test -coverpkg=./internal/api/core ./internal/api/core ./internal/api/udsapi` total 80.4% for core handler package; `go test -cover ./internal/cli ./internal/api/core ./internal/api/udsapi ./internal/network` reports UDS 84.3%, network 81.5%, CLI 78.7%, core 78.8% package-level (existing broad package denominator remains below 80 while new command/helper functions are directly covered).
- Final verification passed: `make verify` completed with exit code 0 after lint fixes for `funlen`, `hugeParam`, `lll`, and `unparam`.
- Task tracking updated: `task_06.md` frontmatter/subtasks/tests and `_tasks.md` row 06 marked completed.
- Local code commit created: `9cccc2f3 feat: add agent self and channel verbs`.
- Post-commit verification passed: `make verify` exit code 0 against the committed implementation.
