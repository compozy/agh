# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement session lifecycle integration for the environment abstraction: provider registry injection, environment metadata persistence/restore, prepare/sync/destroy calls, API/CLI visibility, daemon local-provider wiring, tests, `make verify`, tracking updates, and one local commit.

## Important Decisions
- Treat task_04 plus `_techspec.md` build-order steps 7-8 and ADR-003 as the authoritative design for this run.
- Keep `session` provider-agnostic: it depends only on `internal/environment.Registry`; daemon boot composes `internal/environment/local`.
- Use per-start ACP `Launcher`/`ToolHost` overrides on `acp.StartOpts` so each prepared environment can supply runtime transport/tool behavior without turning the ACP driver into session-scoped mutable state.
- Persist `environment_profile`, `environment_last_sync_at`, and `environment_last_sync_error` in addition to the required environment columns so list/status payloads can expose profile and sync error consistently.

## Learnings
- Shared memory confirms task 03 completed `internal/environment/local`; task 04 can rely on `ResolvedWorkspace.Environment` and `store.SessionEnvironmentMeta` existing, but lifecycle persistence/integration is intentionally still missing.
- Baseline scan confirmed session manager does not yet accept an environment registry, `sessions` DB rows do not yet carry environment fields, session API payloads do not expose environment data, and daemon boot does not yet compose the local environment provider registry.
- Existing session tests now exercise the new environment path by default through a path-preserving fake provider registry in the session harness.
- `manager_stop_integration_test.go` now passes a real `local.NewRegistry()` in the direct real-ACP lifecycle test and includes a prompt before stop to cover create -> prompt -> stop -> resume with the local provider.
- `internal/extension` test harnesses also construct real session managers; they need a local environment registry just like daemon boot after task 04.

## Files / Surfaces
- Touched implementation surfaces: `internal/session`, `internal/acp`, `internal/environment`, `internal/environment/local`, `internal/store`, `internal/store/globaldb`, `internal/api/contract`, `internal/api/core`, `internal/observe`, `internal/daemon`, and `internal/cli`.
- Added focused tests in `internal/session/manager_environment_test.go` plus updates to globaldb, API contract/core, daemon, extension, and session integration tests.

## Errors / Corrections
- First final `make verify` surfaced missing environment registry injection in `internal/extension` test harnesses; fixed the test setup instead of weakening session manager behavior.
- A Telegram bridge test failed once during the same full run, but passed repeatedly in isolation and under `-race`; the final full gate passed with it cached and green.

## Ready for Next Run
- Task 04 implementation, verification, and tracking updates are complete.
- Fresh final verification passed: `make verify` exit 0, web tests 82 files / 676 tests, Go `DONE 4209 tests`, golangci-lint `0 issues`, package boundaries OK.
- Source-only local commit `c483ae27` (`feat: wire session environment lifecycle`) was created.
- Post-commit `make verify` passed with exit 0: web tests 82 files / 676 tests, Go `DONE 4209 tests`, golangci-lint `0 issues`, package boundaries OK.
- `.compozy` tracking/memory files remain uncommitted as required by the run instructions.
