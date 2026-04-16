# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Land task 05 by extending the extension initialize handshake with daemon-derived resource grants and a live session nonce, adding extension Host API support for `resources/list`, `resources/get`, and `resources/snapshot`, and extending the TypeScript SDK/test harness accordingly.

## Important Decisions
- Reuse the existing `internal/resources` raw kernel and source-session authority rules for same-source reads and snapshot sequencing instead of duplicating that logic in extension transport code.
- Treat the current PRD/TechSpec as the approved design baseline for this run; no separate design artifact is needed before implementation.
- Share a single daemon-owned `internal/resources` kernel between Host API handlers and the extension manager so runtime nonce activation and resource reads operate on the same authoritative state.
- Keep generic resource reads same-source-only and preserve bridge operational Host APIs as separate methods rather than reinterpreting `bridges/instances/*` through `resources/get|list`.
- Normalize resource-method protocol failures to HTTP-like JSON-RPC status codes (`403`, `409`, `413`, `429`) so the SDK can handle nonce, conflict, payload, and rate-limit failures uniformly.

## Learnings
- `CapabilityChecker` already stores `ResourceKinds` and `ResourceScopes`; the missing work is transport/session wiring and SDK exposure.
- `internal/resources` already enforces same-source reads, granted kind/scope filtering, non-active nonce rejection, and stale `source_version` rejection.
- Handshake validation is now strict about daemon-issued `session_nonce`, so extension fixtures and SDK harness initialize payloads must populate it even when tests are not exercising resource methods directly.
- Registry-backed extension tests that exercise source-session state need `resources.SchemaStatements()` appended to the test schema bootstrap because `resource_source_state` is now part of the protocol path.

## Files / Surfaces
- `internal/daemon/daemon.go`
- `internal/subprocess/handshake.go`
- `internal/extension/protocol/host_api.go`
- `internal/extension/contract/host_api.go`
- `internal/extension/host_api.go`
- `internal/extension/host_api_resources.go`
- `internal/extension/manager.go`
- `internal/extension/registry.go`
- `sdk/typescript/src/extension.ts`
- `sdk/typescript/src/host-api.ts`
- `sdk/typescript/src/testing/harness.ts`
- `sdk/typescript/src/generated/contracts.ts`
- related Go and TypeScript tests, including bridge fixtures and subprocess handshake coverage

## Errors / Corrections
- `internal/extension/registry_test.go` needed the resource schema statements in its bootstrap path once source-session state started flowing through registry-backed tests.
- `internal/extension/manager.go` required a small helper split around `launchRuntime` to satisfy lint without changing the startup sequence or task scope.

## Ready for Next Run
- Task 05 is implemented and verified.
- Local code commit: `f33d77d` (`feat: add extension resource protocol support`)
- Verification evidence:
  - `bun run test` in `sdk/typescript`
  - `go test ./internal/extension ./internal/subprocess`
  - `go test -cover ./internal/extension` -> `80.1%`
  - `go test -cover ./internal/subprocess` -> `82.8%`
  - `go test -tags integration ./internal/extension -run 'TestManagerIntegrationInitializeIncludesSessionNonceAndResourceGrants|TestHostAPIIntegrationResourcesSnapshotPublishesAndReadsBack|TestHostAPIIntegrationBridgeProviderKeepsOperationalMethodsAlongsideGenericResourceReads|TestHostAPIIntegrationSecondResourceSessionInvalidatesOlderNonce'`
  - `make verify` before commit and again on `f33d77d`
