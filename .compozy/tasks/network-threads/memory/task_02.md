# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement task_02 runtime hard cut for `internal/network`: add `surface`, `thread_id`, `direct_id`, and `work_id`; delete active `interaction_id`, `kind:"direct"`, `KindDirect`, and `DirectBody`; enforce container/work validation and trust canonicalization tests.
- Scope is wire/runtime validation only. Store-backed conversation persistence, public API contracts, CLI/web/tooling/docs updates belong to later tasks unless compilation forces a narrow internal adjustment.

## Important Decisions

- Follow `_techspec.md`, ADR-002, and ADR-003 for validation semantics; do not add compatibility aliases/readers for legacy wire fields.
- Requested skills `nats`, `agh-code-guidelines`, and `agh-test-conventions` are not installed in this session; fallback is the available Go/test/no-workarounds skills plus root/internal repo guidance.
- Add the pure `DirectRoomIdentity` helper in runtime validation scope because direct-surface routing needs deterministic `direct_id` generation without persistence.
- Keep public API/CLI legacy DTO field names outside task scope; existing public `InteractionID` values are mapped into the new internal `WorkID` boundary until later contract/codegen tasks hard-cut them.

## Learnings

- Task_01 completed active RFC/glossary docs, so runtime tests can use those docs as the canonical protocol vocabulary.
- Current worktree already contains unrelated task/QA tracking changes; they must not be reverted or clobbered.
- `make verify` can reach lint after web tests/build; first full run found only style issues introduced by this task in `internal/network`, now corrected.
- Fresh `make verify` after lint cleanup passed (`DONE 8097 tests`, package boundaries OK), then self-review found and corrected one stricter TechSpec invariant: `kind:"capability"` now requires `work_id`.
- Final verification after the capability correction passed: `go test ./internal/network -count=1`; `go test ./internal/network -cover -count=1` (`81.9%`); `make verify` (`0 issues.`, `DONE 8098 tests`, `OK: all package boundaries respected`).
- Fresh pre-commit verification after tracking/memory updates also passed: `make verify` (`0 issues.`, `DONE 8098 tests`, `OK: all package boundaries respected`).
- Local commit created: `cc6194c3 feat: hard cut network wire model`; post-commit `make verify` passed (`0 issues.`, `DONE 8098 tests`, `OK: all package boundaries respected`).

## Files / Surfaces

- Expected runtime surfaces: `internal/network/envelope.go`, `internal/network/validate.go`, `internal/network/lifecycle.go`, `internal/network/router.go`, `internal/network/stats.go`, and focused `internal/network/*_test.go`.
- Touched additional compile surfaces: `internal/network/delivery.go`, `internal/network/audit.go`, `internal/acp/types.go`, `internal/api/core/*network*`, `internal/api/core/agent_channels.go`, `internal/situation/service.go`, and store network-message types/tests.

## Errors / Corrections

- Fixed lint fallout from the runtime rewrite:
  - Replaced a one-case type switch in `internal/network/audit.go`.
  - Passed `receiveState` by pointer through router dispatch helpers to avoid large value copies.
  - Factored trace-transition predicates in `internal/network/lifecycle.go` to satisfy line-length lint.
- Added the missing validator invariant and test for capability envelopes without `work_id`.

## Ready for Next Run

- Task implementation, tracking, local commit, and post-commit verification are complete.
