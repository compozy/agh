# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Completed the tranche-1 tool/MCP cutover to canonical resources.
- `tool` and `mcp_server` now have typed codecs/projectors, static publication rebuilds from daemon config plus extension manifests, and dynamic extension tool contributions publish through `resources/snapshot`.
- `provide_tools` is no longer advertised by the SDK or handshake path.

## Important Decisions

- Kept `internal/tools` and `internal/config` resource-agnostic; daemon owns the `tool` and `mcp_server` resource projectors and static publication sync.
- Used one daemon-owned sync actor/source (`daemon/tool-mcp-sync`) to reconcile config and manifest declarations into canonical resources during boot and extension reload.
- Removed manager-owned MCP catalog authority in the same cutover instead of leaving dual-read or dual-write compatibility paths.

## Learnings

- `core.NewOperatorResourceService` triggers reconcile for every successful CRUD write; integration runtimes that intentionally exercise raw CRUD without a projector must filter trigger calls for unregistered kinds.
- The daemon config publication helper needs a pointer receiver shape and index-based workspace iteration to satisfy repo lint rules on large config/workspace structs.
- Tool specs sent through Host API tests must already satisfy the typed codec contract (`name`, canonical schema object/null handling, source validation) because the resource boundary now canonicalizes and validates them before persistence.

## Files / Surfaces

- `internal/tools/resource.go`
- `internal/config/mcp_resource.go`
- `internal/daemon/{boot.go,daemon.go,extensions.go,tool_mcp_resources.go}`
- `internal/extension/{manifest.go,resource_publication.go,manager.go,host_api_test.go,host_api_integration_test.go}`
- `internal/api/udsapi/udsapi_integration_test.go`
- `internal/subprocess/handshake.go`
- `sdk/typescript/src/{extension.ts,types.ts,extension.test.ts,testing/harness.ts,index.ts,generated/contracts.ts}`
- `openapi/agh.json`

## Errors / Corrections

- UDS integration harness initially failed after the cutover because operator CRUD writes for raw `bundle.activation` resources now triggered the shared reconcile driver, which had only the migrated `tool` projector registered. Fixed the test runtime trigger to ignore unregistered kinds while still projecting `tool`.
- `make verify` exposed daemon lint issues in the new syncer (`hugeParam`, `rangeValCopy`, `unused-parameter`) and one pre-existing daemon-package `goconst` violation. Fixed them with pointer-based config access, index iteration over resolved workspaces, explicit unused closure context, and a shared permission decision constant.
- The daemon integration test needed the updated config-provider pointer signature after the lint-driven helper change.

## Ready for Next Run

- Task implementation and verification are complete.
- Fresh evidence:
  - `make verify` passed.
  - `go test -tags integration ./internal/daemon -run 'TestToolMCPStaticPublicationAndBootRebuild'` passed.
  - `go test -tags integration ./internal/api/udsapi -run 'TestUDSResourceCRUDRoundTrip|TestUDSDeleteResourceRejectsStaleVersionAndRequiresCurrentVersion|TestUDSToolResourceCRUDRoundTripTriggersProjection'` passed.
  - `go test -tags integration ./internal/extension -run 'TestHostAPIIntegrationResourcesSnapshotPublishesAndReadsBack|TestHostAPIIntegrationSecondResourceSessionInvalidatesOlderNonce|TestHostAPIIntegrationResourceSnapshotReplacesToolSetAndRemovesStaleTools'` passed.
  - `bun run --cwd sdk/typescript test` passed.
  - Coverage: `internal/tools` 86.8%, `internal/config` 83.1%, `internal/extension` 80.0%, `internal/api/udsapi` 84.3%, `internal/subprocess` 82.8%, `internal/daemon` 80.1% (`-tags integration`).
