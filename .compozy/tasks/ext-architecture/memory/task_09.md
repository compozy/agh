# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Add the `agh extension` command tree with `list`, `install`, `enable`, `disable`, and `status`.
- Support human/json/toon output and online/offline behavior: use UDS when the daemon is running, otherwise operate directly on the extension registry.
- Deliver the required unit and integration tests plus tracking and verification updates.

## Important Decisions
- The current daemon/client/API stack has no extension endpoints, so task_09 needs a minimal supporting transport seam to satisfy the “use UDS if running” requirement.
- Use direct registry access only for offline flows; online state changes must go through the daemon so runtime extension state can be refreshed without a manual restart.
- Keep the daemon-side transport seam narrow: shared contract payloads plus UDS-only extension handlers/client methods, without widening the HTTP API or introducing a second extension state model.
- Boot the extension runtime/service even when zero extensions are installed so `agh extension install` and `agh extension status` can work against a running daemon immediately.

## Learnings
- `extension.Registry` already handles manifest-path resolution, checksum verification, install persistence, and enabled-state toggles.
- `extension.Manager` exposes `Get`, `List`, and `Statuses`, but it does not yet expose public install/enable/disable/reload operations.
- `extension.Manager.Reload()` is the right public seam for daemon-driven registry mutations; the CLI only needs runtime-aware UDS endpoints layered above it.
- Real manager-backed daemon tests need a helper subprocess fixture with a valid executable path; using a fake command causes reload-time startup failures unrelated to the service logic.
- Parallel CLI subtests that share one temp home/database can trigger `SQLITE_BUSY` while reopening the registry; keep format-output assertions serial when they intentionally share the same fixture DB.

## Files / Surfaces
- `internal/api/contract/contract.go`
- `internal/api/udsapi/server.go`
- `internal/api/udsapi/routes.go`
- `internal/api/udsapi/extensions.go`
- `internal/api/udsapi/handlers_test.go`
- `internal/cli/client.go`
- `internal/cli/root.go`
- `internal/cli/extension.go`
- `internal/cli/helpers_test.go`
- `internal/cli/client_test.go`
- `internal/cli/extension_test.go`
- `internal/cli/cli_integration_test.go`
- `internal/daemon/boot.go`
- `internal/daemon/daemon.go`
- `internal/daemon/extensions.go`
- `internal/daemon/daemon_test.go`
- `internal/extension/manager.go`
- `internal/extension/describe.go`

## Errors / Corrections
- Initial broad rerun failed because the new daemon service test installed an extension with a nonexistent executable; corrected by switching the fixture to the existing helper subprocess harness.
- CLI list format subtests initially ran in parallel against one temp SQLite DB and surfaced `database is locked`; corrected by serializing those subtests while keeping the parent test parallel.

## Ready for Next Run
- Implementation is complete and verified with targeted package tests, the CLI integration flow, CLI coverage at `80.0%`, and a clean `make verify`.
