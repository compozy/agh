---
status: completed
title: "Add extension resource protocol and SDK support"
type: backend
complexity: high
dependencies:
  - task_02
  - task_04
---

# Task 05: Add extension resource protocol and SDK support

## Overview

Extend the extension handshake and Host API so extensions can read back their own desired state and publish dynamic resources through snapshots. This task lands the negotiated resource protocol and SDK helpers that later family migrations depend on, without yet moving any concrete family authority to the new runtime.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST extend extension initialize and handshake payloads to include daemon-computed `granted_resource_kinds`, `granted_resource_scopes`, and the daemon-issued `session_nonce`.
2. MUST add `resources/list`, `resources/get`, and `resources/snapshot` to the extension Host API and keep generic resource reads filtered by same-source, granted kinds, and maximum scope.
3. MUST keep direct `resources/put` and `resources/delete` absent from the extension Host API in v1, enforce snapshot-only publication for extension actors, and avoid collapsing pre-existing family-specific operational methods such as `bridges/instances/list|get|report_state` into generic resource reads.
4. MUST add TypeScript SDK helpers and test harness support for the new protocol so reference fixtures can exercise nonce, version, same-source behavior, and coexistence with family-specific bridge Host APIs.
</requirements>

## Subtasks

- [x] 5.1 Extend handshake payloads with resource grants and session nonce
- [x] 5.2 Add Host API request and response contracts for resource reads and snapshots
- [x] 5.3 Add SDK helpers, generated contract updates, test harness plumbing, and fixture support for the new resource protocol
- [x] 5.4 Add coverage for nonce enforcement, same-source reads, and stale snapshot rejection

## Implementation Details

Follow the TechSpec sections "API Endpoints", "Authority and Validation Rules", and "Integration Points". This task should land the negotiated protocol and SDK path only; concrete family publication cutovers happen in later tasks, including removal of `provide_tools` once tools migrate. The generic resource protocol must coexist with bridge-family operational Host APIs because managed bridge providers read daemon-assigned instances through `bridges/instances/*`, not through same-source `resources/get|list`.

### Relevant Files

- `internal/subprocess/handshake.go` — Initialize handshake must carry grants and daemon-issued session nonce
- `internal/extension/contract/host_api.go` — Shared Host API DTOs need resource read and snapshot contracts
- `internal/extension/host_api.go` — Daemon-side Host API handlers must enforce same-source read and snapshot semantics
- `sdk/typescript/src/extension.ts` — Extension runtime must advertise and handle the new resource methods
- `sdk/typescript/src/host-api.ts` — SDK caller surface must expose typed helpers for list, get, and snapshot
- `sdk/typescript/src/generated/contracts.ts` — Generated SDK contracts must be refreshed alongside the new protocol fields and methods

### Dependent Files

- `internal/extension/manager.go` — Extension lifecycle later depends on negotiated grants and session nonce tracking
- `internal/extension/host_api_bridges.go` — Bridge operational Host APIs must remain distinct from generic resource reads after the protocol lands
- `sdk/typescript/src/generated/contracts.ts` — Generated SDK contracts must stay aligned with the new Host API surface
- `sdk/typescript/src/testing/harness.ts` — Harness fixtures later need the new methods for integration coverage

### Related ADRs

- [ADR-005: Make Resource Access Server-Authoritative](adrs/adr-005.md) — Defines read filtering, snapshot-only publication, and same-source restrictions
- [ADR-007: Use Optimistic Concurrency and Serialized Source Snapshots](adrs/adr-007.md) — Defines nonce and source-version rules that the protocol must surface
- [ADR-008: Confine Raw JSON to the Persistence Boundary and Expose Typed Domain Adapters](adrs/adr-008.md) — Keeps protocol payloads aligned with typed resource boundaries

## Deliverables

- Extended initialize and handshake payloads with negotiated resource grants and session nonce
- New Host API contracts and handlers for `resources/list`, `resources/get`, and `resources/snapshot`
- TypeScript SDK helpers, regenerated contracts, and harness support for the resource protocol
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for handshake, same-source reads, and snapshot sequencing **(REQUIRED)**

## Tests

- Unit tests:
  - [x] initialize responses include only daemon-computed resource grants rather than manifest self-declaration
  - [x] `resources/list` and `resources/get` deny records outside the caller source, granted kinds, or max scope
  - [x] `resources/snapshot` rejects stale `source_version` and non-active `session_nonce` values
  - [x] bridge-family operational Host API contracts continue to coexist with `resources/*` without being reinterpreted as generic same-source reads
  - [x] SDK helpers validate payload shape and surface protocol errors for 403, 409, 413, and 429 responses
- Integration tests:
  - [x] an extension fixture receives `session_nonce`, granted kinds, and granted scopes during initialize
  - [x] an extension snapshot publishes records for its own source and can read them back through `resources/list` and `resources/get`
  - [x] a managed bridge-provider fixture still reads daemon-assigned instances through `bridges/instances/list|get` while generic `resources/get|list` remain same-source-only
  - [x] a second live session for the same extension source invalidates the older nonce and causes stale snapshot calls to fail
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Extensions can publish and inspect their own desired-state records through one negotiated resource protocol
- Same-source filtering, snapshot sequencing, and nonce enforcement are proven before family cutovers begin

## Verification Evidence

- `bun run test` in `sdk/typescript`
- `go test ./internal/extension ./internal/subprocess`
- `go test -cover ./internal/extension`
  - `internal/extension`: `80.1%`
- `go test -cover ./internal/subprocess`
  - `internal/subprocess`: `82.8%`
- `go test ./internal/bridgesdk ./internal/extension/protocol ./extensions/bridges/gchat ./extensions/bridges/linear ./extensions/bridges/slack ./extensions/bridges/teams ./extensions/bridges/telegram ./extensions/bridges/whatsapp ./sdk/examples/telegram-reference`
- `go test -tags integration ./internal/extension -run 'TestManagerIntegrationInitializeIncludesSessionNonceAndResourceGrants|TestHostAPIIntegrationResourcesSnapshotPublishesAndReadsBack|TestHostAPIIntegrationBridgeProviderKeepsOperationalMethodsAlongsideGenericResourceReads|TestHostAPIIntegrationSecondResourceSessionInvalidatesOlderNonce'`
- `make verify`
