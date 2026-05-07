# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Execute Task 11 cross-surface regression hardening for the provider model catalog: prove hard-cut residue is gone, add parity tests across runtime/API/CLI/Host API/web/docs contract surfaces, cover source-error redaction, refresh concurrency/timeout/detached lifetime behavior, run focused gates plus full verification, update tracking, and create one local commit.

## Important Decisions
- Treat `_tasks.md` listing Task 03 as pending as a tracking inconsistency, not a technical blocker for Task 11, because Task 11 depends on Tasks 06-10 and the shared workflow memory records the downstream service/surface work already completed.

## Learnings
- Shared memory records a known pre-existing blocker for this task: `internal/daemon TestDaemonModelCatalogWiring/ShouldCancelAndJoinRefreshWorkOnShutdown` returns `context.DeadlineExceeded` instead of the expected `context.Canceled` when request cancellation interacts with daemon-owned refresh lifetime.
- Root cause: `internal/daemon/model_catalog.go` used the injected logical clock to create a real `context.WithDeadline`, so tests with a fixed 2026-05-07 clock can create an already-expired runtime context. The runtime deadline must use a duration-based timeout while still passing the injected timestamp into catalog refresh options.
- Redaction audit found defensive gaps in API and Host API projection helpers: model/source/status `last_error` and partial refresh `error` payloads need redaction at the surface boundary even though the catalog store also redacts persisted source errors.
- Focused gates and full `make verify` pass after implementation. Post-commit `make verify` also passes for commit `7566e79d test: harden provider model catalog regressions`. Targeted modelcatalog coverage evidence: `go tool cover -func=/tmp/agh-task11-modelcatalog.cover` reports total statement coverage 80.8%.

## Files / Surfaces
- Planned surfaces: `internal/modelcatalog`, `internal/daemon`, `internal/api/...`, `internal/cli`, `internal/extension`, `internal/testutil/e2e`, `web/e2e/fixtures/runtime-seed.ts`, `web/e2e/__tests__/session-provider-override.spec.ts`, `openapi/agh.json`, generated web types, and site/docs residue guards.
- Current implementation surfaces: `internal/daemon/model_catalog.go`, `internal/modelcatalog/redact.go`, model catalog tests, API core model catalog handlers/converters/tests, Host API model handlers/tests, transport parity tests, and web session/provider override fixture/test.
- Verification-generated CLI reference MDX tables were formatted by the full monorepo gate; include only the generated docs churn relevant to keeping verified artifacts in sync, and do not stage unrelated pre-existing UI changes.

## Errors / Corrections
- Baseline hard-cut scan still finds `supported_models` on ACP capability payloads and web ACP mocks plus docs hard-cut warning copy; those are legitimate allowlisted surfaces, not provider config residue.
- The fixed daemon timeout regression now uses `context.WithTimeout` for real elapsed time while preserving the injected clock only for catalog refresh timestamps.

## Ready for Next Run
