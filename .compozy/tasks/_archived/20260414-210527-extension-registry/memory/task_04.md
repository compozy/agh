# Task Memory: task_04.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add remote-install tracking to the extension registry schema and records, add `[extensions.marketplace]` config validation, and wire extension CLI `search`, remote `install`, `remove`, and `update` commands through the shared registry installer stack.
- Phase 1 must remain offline-first: no daemon reload on install, only a restart guidance message.

## Important Decisions
- Source of truth is task_04 + `_techspec.md` + ADR-004; keep scope limited to extension-side persistence/config/CLI wiring and record any wider follow-up instead of expanding scope.
- Reuse the existing local extension CLI flow and extend it for remote installs rather than creating a separate command subsystem.
- Remote installs and updates use the shared registry `Installer` for download/extraction, then register extension-specific metadata in SQLite via `extension.Registry.Install(...)`; updates reuse the same registry path with replace-existing upsert semantics.
- Marketplace-managed extension payloads live under `<AGH_HOME>/extensions`, and phase 1 continues to print restart guidance instead of attempting daemon reload or RPC notification.

## Learnings
- Task 03 already delivered verified `internal/registry/clawhub` and `internal/registry/github` adapters, and the shared workflow memory confirms GitHub source-archive fallback is already handled by the installer’s manifest-root traversal.
- Current baseline: extension CLI only exposes `list/install/enable/disable/status`, and the codebase has no `registry_slug`, `registry_name`, `remote_version`, or `[extensions.marketplace]` support yet.
- `make verify` now passes after updating the `internal/store/globaldb` schema assertions to include the new remote-install columns.

## Files / Surfaces
- `internal/store/globaldb/global_db.go`
- `internal/extension/registry.go`
- `internal/cli/root.go`
- `internal/cli/extension.go`
- `internal/cli/extension_marketplace.go`
- `internal/cli/extension_marketplace_test.go`
- `internal/cli/extension_marketplace_integration_test.go`
- `internal/config/config.go`
- `internal/config/merge.go`
- `internal/config/config_test.go`
- `internal/extension/registry_test.go`

## Errors / Corrections
- Added an integration-tagged `agh extension remove missing-ext` assertion after reviewing the task checklist; the missing-extension path had command-level coverage but needed explicit integration coverage to match the task spec.
- The first full-gate run failed in `internal/store/globaldb` because the schema tests still expected the pre-task extension columns; updating those assertions resolved the real task-owned failure.

## Ready for Next Run
- Task 04 is ready for tracking completion and the local implementation commit; later tasks can rely on the new extension marketplace CLI/config/schema surfaces.
