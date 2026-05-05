---
status: completed
title: Extension Host API, SDK, and Bridge Mapping
type: backend
complexity: high
dependencies:
  - task_08
---

# Task 11: Extension Host API, SDK, and Bridge Mapping

## Overview

Expose the network conversation model through extension Host API methods, SDK exports, capability gates, and bridge mapping rules. This task keeps provider/platform thread IDs distinct from AGH public thread IDs so bridge ingress does not conflate external routing with AGH conversation containers.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and task_08 before editing.
- ACTIVATE `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns`.
- REFERENCE TECHSPEC for Host API method names, capability gates, and bridge SDK constraints.
- FOCUS ON extension and bridge surfaces; native tools are task_10.
- TESTS REQUIRED for capability denial, validation parity, direct resolve race behavior, SDK exports, and bridge thread mapping.
- NO WORKAROUNDS: do not let bridge provider `ThreadID` silently become AGH `public_thread.thread_id` without an explicit mapping decision.
</critical>

<requirements>
- MUST add Host API methods: `network/status`, `network/channels`, `network/peers`, `network/threads`, `network/thread/get`, `network/thread/messages`, `network/directs`, `network/direct/resolve`, `network/direct/messages`, `network/work/get`, and `network/send`.
- MUST require `network.read` for read methods and `network.write` for `network/direct/resolve` and `network/send`.
- MUST reuse shared contract DTOs where practical.
- MUST update protocol, contract, handler, and SDK generation roots.
- MUST document and test bridge ingress mapping from provider/platform thread concepts to AGH conversation refs.
- MUST preserve raw-token rejection and validation parity with HTTP/UDS.
</requirements>

## Subtasks

- [x] 11.1 Add Host API protocol and contract method definitions.
- [x] 11.2 Implement Host API handlers and dependency injection.
- [x] 11.3 Add capability gates and denial tests for read/write operations.
- [x] 11.4 Update SDK generation/exports for network methods.
- [x] 11.5 Define and test bridge mapping for provider `ThreadID` versus AGH `thread_id`.
- [x] 11.6 Add validation parity, direct resolve, and raw-token tests.

## Implementation Details

Bridge ingress may map external Slack-like thread/direct constructs into AGH conversation refs, but it cannot fabricate direct-room membership outside the deterministic direct-room resolver.

### Relevant Files

- `internal/extension/protocol/host_api.go` - Host API wire method definitions.
- `internal/extension/contract/host_api.go` - shared Host API contract.
- `internal/extension/host_api.go` - handler implementation.
- `internal/extension/capability.go` - capability gates.
- `internal/extension/host_api_bridges.go` - bridge Host API mapping.
- `internal/bridges/types.go` - provider routing fields.
- `internal/bridges/dimensions.go` - routing dimensions.
- `internal/bridges/routing.go` - bridge routing logic.
- `internal/bridges/target.go` - target mapping.

### Dependent Files

- SDK generation roots under `sdk/` - generated exports must co-ship if affected.
- `packages/site/content/runtime/core/extensions/*` - task_16 documents Host API behavior.
- `internal/hooks/*` - hook events from task_07 are observation-only and not Host API authority.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - bridge/direct-room mapping boundary.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - Host API field names.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - Host API validation.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: this task owns extension Host API, SDK, capability gates, and bridge ingress semantics.
- Agent manageability: extensions can read and write network conversations through explicit grants.
- Config lifecycle: no new config keys; capability grants remain manifest/config governed by existing extension mechanisms.

### Web/Docs Impact

- Web impact: no direct web code change.
- Docs impact: task_16 must document Host API methods, grants, SDK usage, and bridge mapping constraints.

## Deliverables

- Extension Host API network methods.
- Capability gates and validation parity.
- SDK exports/generation updates.
- Bridge mapping rules and tests for provider thread identity separation.

## Tests

- Unit tests:
  - [x] Read methods require `network.read`.
  - [x] Direct resolve and send require `network.write`.
  - [x] Host API payloads reject `interaction_id` and `kind:"direct"`.
  - [x] Raw claim tokens are rejected or redacted.
  - [x] Bridge provider `ThreadID` is not conflated with AGH `thread_id` without explicit mapping.
- Integration tests:
  - [x] Host API direct resolve is idempotent and parity-aligned with HTTP/UDS.
  - [x] Host API send validates surface/container fields consistently.
  - [x] SDK exports compile and expose new network methods.
  - [x] Bridge ingress preserves provider routing dimensions while producing valid AGH conversation refs when configured.
- Test coverage target: >=80% for touched extension/bridge packages.
- All tests must pass.

## Success Criteria

- Extensions can operate network conversations through capability-gated Host API methods.
- Bridge integrations have explicit, test-covered mapping between provider threads and AGH public threads.
- No extension surface accepts legacy fields or bypasses direct-room membership rules.
