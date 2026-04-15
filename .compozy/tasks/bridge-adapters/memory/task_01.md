# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Extend the bridge core model, global DB persistence, and provider manifest metadata with `provider_config`, typed DM policy, structured degradation data, required secret slots, and optional config schema/version hints.
- Keep scope limited to model/storage/manifest concerns; runtime handshake and Host API redesign stay out of scope for this task.

## Important Decisions
- Use the TechSpec/ADR-approved design directly; no separate design phase for this run.
- Keep `provider_config` distinct from `delivery_defaults` all the way through validation and persistence.
- Treat broad package coverage in `internal/extension` and `internal/store/globaldb` as pre-existing package debt; verify the task against the bridge/manifest/globaldb task surfaces plus the repository `make verify` gate.

## Learnings
- Current `BridgeConfig` only carries `platform` and `display_name`.
- Current `BridgeInstance` and `bridge_instances` storage only carry `routing_policy` and `delivery_defaults`; the new provider-scoped fields do not exist yet.
- Task-specific verification now passes: `make verify`, `go test ./internal/daemon -run TestBridgeRuntimeListProviders`, `go test -tags integration ./internal/extension -run TestLoadManifestBridgeMetadataRoundTrip`, and `go test -tags integration ./internal/store/globaldb -run 'TestGlobalDBBridgeInstanceRoundTripAcrossReopen|TestOpenGlobalDBMigratesLegacyBridgeInstancesWithoutProviderConfig'`.
- Package-wide `go test -cover ./internal/extension ./internal/store/globaldb` still reports `78.2%` and `78.5%` because those packages contain older unrelated surfaces outside this task’s bridge/manifest changes.

## Files / Surfaces
- `internal/bridges/types.go`
- `internal/bridges/registry.go`
- `internal/store/globaldb/global_db.go`
- `internal/store/globaldb/global_db_bridge.go`
- `internal/extension/manifest.go`
- `internal/daemon/bridges.go`
- `internal/bridges/types_test.go`
- `internal/store/globaldb/global_db_bridges_test.go`
- `internal/store/globaldb/global_db_bridges_integration_test.go`
- `internal/extension/manifest_test.go`
- `internal/extension/manifest_integration_test.go`
- `internal/extension/registry_test.go`
- `internal/store/globaldb/global_db_extra_test.go`

## Errors / Corrections
- `go test -tags integration ./internal/extension ./internal/store/globaldb` fails outside task scope because `reference_integration_test.go` installs `sdk/examples/prompt-enhancer`, whose `node_modules/.bin/tsc` symlink escapes the example root and is rejected by the extension source guard.

## Ready for Next Run
- Task 01 implementation and verification are complete; leave tracking-only artifacts out of the auto-commit unless the workflow explicitly requires them.
