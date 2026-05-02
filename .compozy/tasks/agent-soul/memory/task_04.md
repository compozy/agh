# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Integrate runtime Soul behavior for Task 04: session start snapshots, explicit refresh, prompt/context projection, spawn provenance, and task claim metadata without claim-time `SOUL.md` file I/O.

## Important Decisions
- Reuse the existing `internal/soul` resolver and GlobalDB snapshot/session provenance APIs from tasks 01-02; do not add alternate parsing or storage authority.
- Add concrete `AGENT.md` source-path plumbing to agent definitions so Soul resolution uses the selected agent artifact path when available.
- Session start/refresh persist only active valid snapshots into session provenance; invalid existing `SOUL.md` fails closed, missing/disabled Soul leaves no active session snapshot.
- Prompt Soul renders as an append startup section after the base agent prompt and before existing append sections. The provider returns empty unless startup context carries an active valid snapshot.
- Task claim provenance will flow through `task.ClaimCriteria` and be merged into `task_runs.metadata_json` inside the claim transaction; `ClaimNextRun` must not resolve, parse, or read `SOUL.md`.

## Learnings
- Current code already has `sessions.soul_snapshot_id`, `sessions.soul_digest`, and `sessions.parent_soul_digest` columns/store methods, but session `Info`/meta do not yet expose those fields.
- `/agent/context` is assembled by `internal/situation.Service`; prompt rendering uses the same payload through `situation.RenderPrompt`.
- The agent task claim handler builds `ClaimCriteria` from the validated caller session, so it is the right place to hold the session Soul lock while reloading current session provenance and invoking `ClaimNextRun`.
- The test-convention helper exists under `.agents/skills/agh-test-conventions/scripts/check-test-conventions.py`, not `scripts/check-test-conventions.py`; it only accepts one file per invocation. It passes the new `internal/session/soul_test.go`, while several older touched test files still fail on pre-existing inline-case/subtest-name patterns outside this task's additions.

## Files / Surfaces
- `internal/session`: start/runtime/meta/refresh/lock/spawn provenance.
- `internal/daemon`: prompt section provider/wiring and session-manager dependency injection.
- `internal/situation` + `internal/api/contract`: compact context Soul projection.
- `internal/task` + `internal/store/globaldb`: claim criteria and metadata persistence.
- `internal/agentidentity` + `internal/api/core`: caller snapshot propagation and claim lock integration.

## Errors / Corrections
- Corrected a test fixture from non-existent `taskpkg.PriorityNormal` to `taskpkg.PriorityMedium`.
- Corrected first `make verify` lint failures by splitting `RefreshSoul` into smaller helpers, wrapping the long persistence call, and passing heavy Soul snapshot/profile/resolved values by pointer.
- Corrected second `make verify` failure by running official `make codegen`; generated output reserialized `openapi/agh.json` heavily, and `make codegen-check` now passes.
- Corrected race-enabled `make test` failures where extension tests supplied partial workspace configs with zero-valued `Agents.Soul`; session Soul integration now defaults only the zero-value Soul config at the session boundary and has a regression test.

## Ready for Next Run
- Core implementation and focused tests are in place. Focused packages passed with `go test ./internal/session ./internal/daemon ./internal/situation ./internal/api/core ./internal/api/contract ./internal/task ./internal/store/globaldb ./internal/agentidentity ./internal/config ./internal/workspace ./internal/soul -count=1`.
- `make web-test` passed with 200 test files / 1506 tests. It emitted repeated Node warnings that `NO_COLOR` is ignored when `FORCE_COLOR` is set, but exited 0.
- After lint corrections, the focused affected Go package suite passed again and `make lint` passed with 0 issues. Golangci printed non-blocking `modernize: omitzero` alternative-fix notices.
- After the zero-config Soul fix, the focused regression test, representative extension tests, and `go test -race ./internal/session ./internal/extension -count=1` passed. The earlier stop lifecycle failure passed 20 focused `-race` repetitions and appears load/timing-related in the failed full gate run.
- Full `make verify` passed after the Soul config fix. Final output included `DONE 7479 tests in 62.198s` and `OK: all package boundaries respected`; non-blocking warnings remained from Node `NO_COLOR`/`FORCE_COLOR`, Turbo update notice, Vite chunk size, and macOS linker `-bind_at_load`.
- Added explicit resume/reopen regressions after task matrix review. Focused `TestManagerSoulSessionSnapshots` and `TestGlobalDBClaimNextRunPersistsSoulProvenanceMetadata` passed with session resume and GlobalDB reopen checks.
- Final pre-commit `make verify` passed after resume/reopen regressions. Final output included `DONE 7480 tests in 50.597s` and `OK: all package boundaries respected`.
- Pre-commit `make verify` also passed after tracking/memory updates. Final output included `DONE 7480 tests in 11.968s` and `OK: all package boundaries respected`.
- Created local commit `fd256b7b` (`feat: integrate agent soul runtime context`).
- Post-commit `make verify` passed. Final output included `DONE 7480 tests in 13.929s` and `OK: all package boundaries respected`.
