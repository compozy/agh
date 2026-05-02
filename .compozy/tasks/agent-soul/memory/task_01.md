# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_01 only: `[agents.soul]` config plus isolated `SOUL.md` resolver/parser/validator/digest/projection/diagnostics. No session, task, heartbeat, network, API, CLI, UDS, Host API, or storage behavior changes in this task.

## Important Decisions
- Use `_techspec_soul.md` as the field-level source of truth for Soul defaults when it conflicts with the aggregate spec. This sets `agents.soul.context_projection_bytes = 2048` for task_01 even though the aggregate config summary mentions `4096`.
- Keep resolver code in a new `internal/soul` package unless source exploration exposes a stronger existing package fit.
- Treat `SourcePath` values passed to `soul.Parse` as workspace-relative when `WorkspaceRoot` is provided, then enforce containment against that root.

## Learnings
- Pre-change baseline: `internal/soul` does not exist and `rg "agents\\.soul|type Soul|SoulConfig|SOUL\\.md|soul_digest" internal cmd packages web` found no implementation.
- Existing worktree has unrelated pre-existing changes in `.compozy/tasks/agent-soul/_tasks.md`, `task_16.md`, and `task_17.md`; avoid staging unrelated tracking changes.
- Current run resumes with implementation already present in `internal/config` and new `internal/soul`; validate and refine the existing changes instead of replacing them.
- Execution checklist is: config defaults/validation/overlay; parser/frontmatter; forbidden authority diagnostics; deterministic digest/profile/projection/read model; oversized/missing/disabled cases; resolver callable after `AGENT.md` load without runtime mutation; >=80% package coverage; no session/task/heartbeat/network behavior changes.
- Focused validation: `go test ./internal/config ./internal/soul ./internal/frontmatter -count=1` passed; `go test -race ./internal/config ./internal/soul ./internal/frontmatter -count=1` passed; `go test ./internal/soul -coverprofile=/tmp/agh-soul-cover.out -count=1` reports 84.3% statement coverage after lint fixes.
- `make lint` passed after fixing gocritic findings in `internal/soul` (`emptyStringTest`, `hugeParam`).
- `scripts/check-test-conventions.py` is referenced by the skill but is not present in this repo; `rg --files | rg 'check-test-conventions\\.py$|test-conventions'` found no script.
- Final focused validation after self-review correction: `go test ./internal/config ./internal/soul ./internal/frontmatter -count=1` passed; `go test -race ./internal/config ./internal/soul ./internal/frontmatter -count=1` passed; `go test ./internal/soul -coverprofile=/tmp/agh-soul-cover.out -count=1` reports 84.6% statement coverage.
- Final pre-commit gate: `make verify` passed after the `ReadModel.Valid` invalid-state correction. The command emitted existing environment/tool warnings (`NO_COLOR` ignored due to `FORCE_COLOR`, Vite chunk-size warning, macOS linker warning) but exited 0 with `0 issues`, `DONE 7396 tests`, and `OK: all package boundaries respected`.
- Local implementation commit: `cd68c9ca feat: add soul resolver foundation` with only `internal/config` and `internal/soul` files.
- Post-commit gate: `make verify` passed with `0 issues`, `DONE 7396 tests`, and `OK: all package boundaries respected`.

## Files / Surfaces
- Expected implementation surfaces: `internal/config`, `internal/frontmatter` only if reusable helpers need extension, new `internal/soul`, and focused tests.
- Current implementation surfaces touched: `internal/config/config.go`, `internal/config/merge.go`, `internal/config/config_test.go`, `internal/config/tool_surface.go`, `internal/config/tool_surface_test.go`, `internal/soul/soul.go`, `internal/soul/soul_test.go`.

## Errors / Corrections
- Self-review found invalid parse diagnostics left `ReadModel.Valid=true` through `emptyResult`; corrected `resultWithDiagnostics` to mark the read model inactive/invalid and added assertions that invalid present content reports `Present=true`, `Active=false`, and `ReadModel.Valid=false`.

## Ready for Next Run
- Task 01 implementation is complete and committed. Tracking/memory files remain intentionally uncommitted per task instruction to keep tracking-only files out of the automatic code commit.
