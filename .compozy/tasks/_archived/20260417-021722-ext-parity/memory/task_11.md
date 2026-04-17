# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Migrate `bridge.instance` desired state into canonical shared resources while keeping delivery, routes, health, assigned-instance visibility, and status/reporting as bridge-owned operational state.
- Required proof points: codec validation against provider manifest metadata, resource-backed projector Build/Apply semantics, legacy bridge-definition authority removal, boot rebuild, degraded/rollback coverage, 80%+ relevant package coverage, and clean `make verify`.

## Important Decisions
- Treat the approved task/TechSpec/ADRs as the execution design; no separate design-approval loop is needed for this PRD task.
- Follow the established cutover pattern from tasks 07-10: typed resource codec/store at the boundary, daemon/domain projector registration, full-snapshot boot/post-write reconcile, and no dual-write or compatibility path.
- Keep generic resources authoritative only for bridge desired configuration. Runtime status, degradation, routes, delivery queues/metrics, assigned-instance visibility, and provider reports remain bridge-owned operational state.
- Bundle-managed bridge definitions now publish through canonical `bridge.instance` resources via managed source sync instead of writing directly to `bridge_instances`.
- Projection applies full canonical snapshots into the daemon-visible bridge registry and removes stale legacy rows; after the cutover, legacy bridge-definition rows are demoted to projected runtime state rather than desired-state authority.

## Learnings
- Shared workflow memory says tasks 01-10 are complete; `bridge.instance` can rely on existing raw/typed resource kernel, reconcile driver, resource CRUD, Host API resource protocol, and prior migrated families.
- TechSpec explicitly keeps bridge-provider visibility on `bridges/instances/list|get` rather than generic same-source `resources/list|get`; resource authority only covers desired bridge configuration.
- Baseline search found `bridge.instance` only as a surface/test kind string; no bridge codec, typed store, projector, or daemon registration exists yet.
- Existing bridge runtime persists desired fields and operational state in `bridge_instances`; cutover must make resource records authoritative while preserving lifecycle/status/degradation updates as bridge-owned operational writes.
- Existing bundle bridge sync writes directly to `bridge_instances`; it must become resource-backed or it would remain a competing desired-state authority.
- `bridge.instance` validation needs the installed provider manifest at the codec boundary; daemon boot wires provider lookup through the live bridge runtime so resource writes can enforce provider platform, `secret_slots`, and `config_schema`.
- `BuildResourceState` is side-effect free and uses existing bridge rows only to preserve operational fields; `ApplyBridgeResourceState` atomically swaps the projected rows and rolls back on extension reload failure.
- Updating a dynamic bridge instance through the bridge runtime must mutate the canonical resource record first, then apply projection, and only then write bridge-owned operational degradation changes if requested.

## Files / Surfaces
- Touched implementation surfaces:
  - `internal/bridges/{resource.go,resource_projection.go,managed_sync.go}`
  - `internal/daemon/{bridge_resources.go,bridges.go,boot.go,daemon.go}`
  - `internal/store/globaldb/global_db_bridge.go`
- Touched test surfaces:
  - `internal/bridges/resource_test.go`
  - `internal/daemon/{bridges_test.go,daemon_integration_test.go}`
  - `internal/store/globaldb/global_db_bridges_test.go`

## Errors / Corrections
- First full `make verify` run failed at lint:
  - `(*bridgeRuntime).UpdateInstance` exceeded gocyclo threshold.
  - Three lines exceeded `lll`.
- Correction: split resource-backed bridge update into focused helpers (`loadMutableBridgeInstanceResource`, `updatedBridgeInstanceSpec`, `putBridgeInstanceResource`, operational degradation status helper) and wrapped long calls/comments.
- Verification after correction:
  - `go test ./internal/bridges ./internal/daemon ./internal/store/globaldb` passed.
  - `go test -tags integration ./internal/daemon -run TestBridgeResourceProjectionReconcilesWritesAndBootRebuild` passed.
  - `go test -tags integration ./internal/extension -run 'TestHostAPIIntegrationBridgeProviderKeepsOperationalMethodsAlongsideGenericResourceReads|TestHostAPIIntegrationBridgesInstancesReportStatePublishesAuthRequired|TestHostAPIIntegrationBridgesInstancesListAndGetReturnOwnedInstances'` passed.
  - `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi -run Bridge` passed.
  - `go test -cover ./internal/bridges` passed at 80.2% statements.
  - Final `make verify` passed with web tests `82 passed`, Go `DONE 4093 tests`, golangci-lint `0 issues`, and package boundaries respected.

## Ready for Next Run
- Task 11 implementation and verification are complete. Remaining finalization: update task tracking, create the required local code commit, and run/record post-commit verification.
