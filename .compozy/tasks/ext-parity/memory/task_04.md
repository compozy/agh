# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Completed Task 04 by adding the static extension surface registry and authoritative resource grant config path without changing handshake payloads yet.
- Delivered `internal/extension/surfaces`, manifest request validation, `[extensions.resources]` config parsing/merge/validation, centralized grant derivation, and verified unit/integration coverage.

## Important Decisions
- Treat the PRD + TechSpec as the approved design artifact for this run instead of opening a separate brainstorming approval loop.
- Keep resource grant derivation centralized in `CapabilityChecker` and make manager state consume that computed grant snapshot instead of duplicating action/security calculations.
- Model manifest grant requests as a family-oriented manifest section validated by `internal/extension/surfaces`.
- Keep source-tier resource ceilings explicit: bundled/user extensions top out at `global`, while workspace/marketplace extensions top out at `workspace`.
- Move raw resource schema bootstrap ownership into `internal/resources` via `SchemaStatements()` so config can depend on resource types without reintroducing test bootstrap cycles through `globaldb`.

## Learnings
- `CapabilityChecker` now resolves resource grants from the surface registry, source-tier ceiling, operator policy, manifest request, and session scope ceiling in one path, then stores the grant snapshot for later consumers.
- `Manager.validateExtension` now persists granted resource kinds/scopes on managed extensions, and resource publication requests also count as a subprocess-requiring capability.
- Repo-wide verification exposed a resources test import cycle once config imported `internal/resources`; fixing test bootstrap to use `resources.SchemaStatements()` kept the authority boundary intact and restored `make verify`.

## Files / Surfaces
- `internal/extension/surfaces/`
- `internal/extension/manifest.go`
- `internal/extension/capability.go`
- `internal/extension/manager.go`
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/daemon/daemon.go`
- `internal/resources/schema.go`
- `internal/store/globaldb/global_db.go`
- `internal/resources/kernel_test.go`
- `internal/resources/kernel_integration_test.go`

## Errors / Corrections
- Corrected a repo-wide test bootstrap cycle by moving resource schema statements into `internal/resources` and removing `globaldb` from `internal/resources` test setup.
- Corrected pre-existing integration test call sites touched by the package changes in `internal/extension/host_api_integration_test.go` and `internal/extension/reference_integration_test.go`.

## Ready for Next Run
- Task 04 is verified complete.
- Follow-on handshake, startup, and CRUD tasks should consume the stored `CapabilityChecker` grant snapshot rather than recomputing resource publication policy ad hoc.
