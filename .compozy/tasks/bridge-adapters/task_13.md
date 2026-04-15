---
status: completed
title: "Implement the Microsoft Teams provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 13: Implement the Microsoft Teams provider extension

## Overview

Implement the Teams provider using the shared provider-scoped substrate and per-instance provider configuration. This task validates service URL handling, tenant-aware configuration, and Bot Framework activity mapping without promising full parity beyond the approved bridge v1 scope.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a Teams provider using the shared provider runtime and provider-scoped Host API contract.
2. MUST support inbound Bot Framework activity mapping and outbound delivery semantics that fit the approved bridge v1 scope.
3. MUST use provider config and secret slots to represent tenant pinning, bot identity, and service URL behavior without relying on process-scoped single-instance assumptions.
4. SHOULD validate that the provider-scoped runtime can support enterprise-flavored tenant configuration cleanly.
</requirements>

## Subtasks
- [x] 13.1 Create the Teams provider runtime and manifest on top of `internal/bridgesdk`
- [x] 13.2 Implement Teams activity ingestion, verification, and bridge-event mapping
- [x] 13.3 Implement Teams outbound delivery, service URL handling, and state reporting
- [x] 13.4 Add conformance and tenant-configuration coverage for Teams

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "Teams", and "Operational Requirements". This task should stay inside bridge v1 while using provider config for Teams-specific tenancy and endpoint behavior.

### Relevant Files
- `internal/bridges/types.go` — Teams needs the daemon-owned bridge model to carry provider config and DM policy
- `internal/extension/protocol/host_api.go` — Teams runtime depends on the provider-scoped Host API methods
- `internal/extensiontest/bridge_adapter_harness.go` — Teams should pass the shared conformance harness
- `internal/bridges/delivery_types.go` — Teams delivery must map into the approved textual and edit or delete contract where supported

### Dependent Files
- `extensions/bridges/teams/*` — New Teams provider package tree should live here if the repo follows the TechSpec layout
- `internal/daemon/bridges_test.go` — Teams can later serve as a tenant-heavy integration target
- `docs/plans/2026-04-15-bridge-adapters-design.md` — Teams may inform later notes about enterprise provider configuration

### Reference Sources (.resources/)
- `.resources/chat/packages/adapter-teams/src/index.ts` — **Primary reference**: Chat-SDK Teams adapter; Bot Framework `BridgeHttpAdapter`, `appId`/`appPassword`/`appTenantId` config, Adaptive Cards v1.4, Task Modules (modals), streaming via post+edit, DM creation with `tenantId`, Graph API reader for message history, mention normalization
- `.resources/chat/packages/adapter-teams/src/format.ts` — Teams Adaptive Card builder (mdast ↔ Adaptive Card v1.4)
- `.resources/chat/packages/adapter-teams/src/graph-reader.ts` — Teams `TeamsGraphReader` using Microsoft Graph API for `ChatMessage.Read.Chat`; reference for message fetch implementation
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "teams" --topic chat-sdk` for Teams-specific patterns

### Related ADRs
- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Teams validates the provider-scoped runtime under tenant-aware configuration pressure
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Teams implementation must stay within the approved narrowed v1 scope

## Deliverables
- Production Teams provider extension built on the shared bridge substrate
- Teams activity and delivery mapping within the approved bridge v1 contract
- Provider-specific conformance and tenant-configuration coverage for Teams
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for Teams provider behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Teams activity payloads map into bridge message or typed interaction events with stable routing identity
  - [x] Teams provider config validation accepts tenant pinning and rejects malformed service URL or tenant settings
  - [x] Teams outbound delivery mapping preserves the service URL or conversation identity needed for follow-up delivery
- Integration tests:
  - [x] a provider-scoped Teams runtime ingests Bot Framework activity payloads for owned bridge instances successfully
  - [x] Teams outbound delivery posts responses and reports state transitions through the shared runtime path
  - [x] Teams provider passes the shared conformance harness plus tenant-aware configuration scenarios
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Teams validates the provider-scoped runtime for enterprise-leaning tenant configuration
- The bridge v1 implementation stays within the approved narrowed scope for Teams
