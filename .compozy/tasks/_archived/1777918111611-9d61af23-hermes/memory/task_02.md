# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implemented Task 02 observability retention and health base.
- Success evidence: `make verify` passed after web lint/typecheck/tests/build, Go lint, race-enabled Go tests, Go build, and boundary checks.

## Important Decisions

- `observability.retention_days = 0` is the keep-history/no-op mode. Negative values fail config validation.
- Retention deletion is store-owned in `globaldb` and lifecycle-owned by the daemon through explicit `Observer.StartRetention` / `Observer.ShutdownRetention`.
- Retention sweep scope is limited to global observability rows: `event_summaries.timestamp`, `token_stats.updated_at`, and `permission_log.timestamp`.
- Session catalog rows, per-session event DBs, and active debugging evidence outside the global observe tables remain untouched.
- Health payload extensions are typed: `health.persistence` and `health.retention` in `internal/api/contract`, generated OpenAPI, web generated types, and the TypeScript SDK contracts.

## Learnings

- Existing config/settings/web already allowed a `0` value for retention input; the required web follow-up was generated type/fixture/test updates, not new settings UI.
- `make codegen` also updates `sdk/typescript/src/generated/contracts.ts`; include that with OpenAPI/web generated changes.
- Full `make verify` surfaced an unrelated timing failure in `internal/session`; the exact test and package both passed under race on rerun, then the full gate passed cleanly.

## Files / Surfaces

- Backend: `internal/config`, `internal/store/globaldb`, `internal/observe`, `internal/daemon`, `internal/api/contract`, `internal/api/core`, `internal/cli`.
- Tests: `internal/store/globaldb/global_db_test.go`, `internal/observe/observer_test.go`, `internal/api/core/*_test.go`, `internal/config/config_test.go`.
- Generated/API clients: `openapi/agh.json`, `web/src/generated/agh-openapi.d.ts`, `sdk/typescript/src/generated/contracts.ts`.
- Web consumers: `web/src/systems/daemon/*`, `web/src/hooks/routes/use-home-page.test.tsx`.
- Docs: `packages/site/content/runtime/core/configuration/config-toml.mdx`, `packages/site/content/runtime/cli-reference/observe/health.mdx`.

## Errors / Corrections

- Fixed lint issues from first `make verify`: checked transaction rollback cleanup, changed health conversion to pointer input, and documented the stored retention cancel function for gosec.
- Did not stage unrelated existing worktree changes under design assets or unrelated landing-page files.

## Ready for Next Run

- Task 03 and Task 07 can consume `health.persistence` and `health.retention` instead of adding ad hoc health fields.
- Retention health exposes enabled state, retention days, sweep interval seconds, last sweep status/error/timestamps, cutoff timestamp, and deleted row counts.
