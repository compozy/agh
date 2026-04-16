# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Migrate `automation.job` and `automation.trigger` desired-state definitions to the canonical resource runtime.
- Keep automation runs, history, locks, and other runtime execution state in automation-owned storage and APIs.
- Deliver side-effect-free automation projector build plus atomic apply semantics, boot rebuild, cutover, and runtime-state preservation coverage.

## Important Decisions

- Treat the approved PRD/TechSpec as the design artifact for this execution task; no separate design-approval loop is needed.
- Keep legacy automation definition CRUD code present for non-daemon/test harness callers, but make daemon-wired automation managers resource-backed so legacy definition rows no longer drive desired state after cutover.
- Keep runs, overlays, webhook secrets, and runtime locks in automation/globaldb-owned operational tables; remove FK coupling from those operational tables to legacy definition tables.
- Make projector `Build` create off-path scheduler/trigger-engine instances and make `Apply` start-and-swap atomically, preserving the previous runtime when the new runtime fails to start.
- Avoid direct apply during boot-time managed definition sync; daemon boot publishes config/package definitions into resources, then `ReconcileDriver.RunBoot()` projects the persisted snapshot.

## Learnings

- Prior tasks already established the raw/typed resource kernel, reconcile driver, post-commit triggers, hook/tool/agent/skill resource cutover patterns, and daemon-owned source sync patterns.
- Automation manager `Start` must not read resource-backed definitions while holding its startup lock; boot projection now happens through the resource reconcile driver after automation starts.
- Operator `/api/resources` writes already trigger the shared reconcile driver once codecs/projectors are registered, so automation resource writes can fan out without adding family-specific transport callbacks.
- Config/package `enabled` changes are operational overlays in resource mode; canonical resource specs remain unchanged for those managed sources.

## Files / Surfaces

- Expected surfaces: `internal/automation`, `internal/daemon`, `internal/store/globaldb`, `internal/api/core`, `internal/api/udsapi`, `.compozy/tasks/extensibility-parity/task_10.md`, and `_tasks.md`.
- Added/updated implementation: `internal/automation/resource.go`, `internal/automation/resource_projection.go`, `internal/automation/manager.go`, `internal/daemon/automation_resources.go`, `internal/daemon/boot.go`, `internal/daemon/daemon.go`, `internal/store/globaldb/global_db.go`, `internal/store/globaldb/global_db_automation.go`.
- Added/updated tests: `internal/automation/resource_test.go`, `internal/daemon/daemon_integration_test.go`, `internal/api/udsapi/udsapi_integration_test.go`, `internal/store/globaldb/global_db_automation_test.go`.

## Errors / Corrections

- Fixed a startup deadlock introduced by resource-mode `Start` reading effective resource definitions while holding the manager lock; resource-mode start now initializes empty runtime and lets boot reconcile apply resources.
- Updated daemon boot overlay integration expectations from legacy `automation_jobs` / `automation_triggers` rows to canonical resource records.
- Replaced obsolete globaldb source-lookup helper coverage after overlay persistence was decoupled from legacy definition rows.
- Added cleanup for Build-only projection plans in tests and shutdown-on-apply-failure paths to avoid leaking off-path runtime components.

## Ready for Next Run

- Full verification is clean: `make verify` exited 0 after frontend checks, lint, Go tests/build, and package-boundary checks. `internal/automation` coverage is 80.1%.
- Self-review found no blocking issues.
- Implementation was committed locally as `91c4d33 refactor: migrate automation definitions to resources`.
- Post-commit `make verify` exited 0, so the committed tree has fresh full-gate evidence.
