# Task Memory: task_12.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add durable, typed session lineage/spawn metadata for Task 13 without implementing the spawn API or reaper.
- Required surfaces: `session.CreateOpts`/`session.Info`, on-disk session meta, `globaldb.sessions`, session read DTO conversion, generated contract checks, manager/store/API tests.

## Important Decisions
- Manual user/dream/system sessions are normalized as root sessions with `root_session_id = session_id`, no parent, and `spawn_depth = 0`.
- Coordinator sessions are modeled as root sessions with distinct `session_type = coordinator` and must carry a future TTL deadline at creation.
- Spawned child sessions use `session_type = spawned` and must carry parent/root/depth plus a future TTL deadline; Task 13 will add public spawn commands and reaper enforcement.
- Parent/root/depth/role/TTL/auto-stop are typed session columns; budget and permission policy are typed Go structs stored in the TechSpec-required JSON columns.

## Learnings
- Task 02 already introduced safe lineage DTO shapes in generated contracts, but canonical `session.Info`, `CreateOpts`, and `globaldb.sessions` had no source-of-truth lineage persistence.
- Current worktree has pre-existing `.compozy/tasks/autonomous` tracking/doc changes and untracked memory files; avoid reverting or staging unrelated tracking changes.
- Public contract structs already contained the lineage DTOs from Task 02, so Task 12 updated conversion/tests rather than changing generated OpenAPI shapes.
- Legacy workspace migration preflight must recognize every new `sessions` column, or old DBs fail schema validation before the normal migration path can run.
- `store.SessionLineage` validation is intentionally strict enough for Task 13 to enforce spawn caps and permission narrowing without parsing free-form metadata.

## Files / Surfaces
- `internal/store/session_lineage.go`
- `internal/store/types.go`
- `internal/session/session.go`
- `internal/session/manager.go`
- `internal/session/manager_start.go`
- `internal/session/query.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_session.go`
- `internal/store/globaldb/migrate_workspace.go`
- `internal/api/core/conversions.go`
- `internal/api/core/agent_identity.go`
- `internal/agentidentity/identity.go`
- `internal/daemon/sandbox_reconcile.go`
- `internal/observe/observer.go`

## Errors / Corrections
- Lint flagged argument-count and cognitive-complexity issues in the session catalog mapping; corrected by introducing `sessionCatalogRecord`, scan-part helpers, and focused normalization helpers.
- Store coverage was raised with `internal/store/session_lineage_test.go`; `go test ./internal/store -cover -count=1` now reports 81.7%.

## Ready for Next Run
- Task 13 can build spawn behavior on `session.CreateOpts.Lineage`, `session.SessionTypeSpawned`, `session.SessionTypeCoordinator`, `store.SessionLineage`, and globaldb list filters for `SessionType`, `ParentSessionID`, `RootSessionID`, and `SpawnRole`.
- No spawn API, agent-facing spawn command, spawn reaper, or coordinator bootstrap behavior was implemented in Task 12.
- Verification evidence: focused package tests passed; `make codegen-check`, `make web-typecheck`, `make web-test`, and post-commit full `make verify` passed.
- Local implementation commit: `0eff7b38 feat: add session lineage metadata`.
