---
status: completed
title: "Implement the Linear provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 16: Implement the Linear provider extension

## Overview

Implement the Linear provider to validate provider-owned mode switches inside `provider_config`. This task proves the bridge substrate can support a provider that explicitly splits comment mode and agent-session mode without collapsing everything into one generic adapter path.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a Linear provider using the shared provider runtime and provider-scoped Host API contract.
2. MUST support Linear ingress and outbound behavior for the approved bridge v1 scope while distinguishing provider modes through `provider_config`.
3. MUST use provider config and secret slots to represent API key versus OAuth mode and any comment versus agent-session mode split described in the TechSpec.
4. SHOULD validate that provider-owned mode switches live in provider config rather than leaking into daemon-global bridge semantics.
</requirements>

## Subtasks

- [x] 16.1 Create the Linear provider runtime and manifest on top of `internal/bridgesdk`
- [x] 16.2 Implement Linear ingress normalization and bridge-event mapping for supported provider modes
- [x] 16.3 Implement Linear outbound delivery, mode-aware behavior, and state reporting
- [x] 16.4 Add conformance and provider-mode coverage for Linear

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "Linear", and "Operational Requirements". This task should implement Linear only and keep provider-mode branching inside `provider_config` rather than in shared bridge semantics.

### Relevant Files

- `internal/bridges/types.go` — Linear relies on provider-owned config and the daemon-owned bridge identity model
- `internal/extension/protocol/host_api.go` — Linear runtime depends on the provider-scoped Host API methods
- `internal/extensiontest/bridge_adapter_harness.go` — Linear should pass the shared conformance harness
- `internal/bridges/delivery_types.go` — Linear outbound behavior must remain within the approved bridge v1 contract

### Dependent Files

- `extensions/bridges/linear/*` — New Linear provider package tree should live here if the repo follows the TechSpec layout
- `internal/daemon/bridges_test.go` — Linear can later serve as a provider-config-heavy integration target
- `docs/plans/2026-04-15-bridge-adapters-design.md` — Linear may inform later notes about provider-mode configuration patterns

### Reference Sources (.resources/)

- `.resources/chat/packages/adapter-linear/src/index.ts` — **Primary reference**: Chat-SDK Linear adapter; dual mode (`"comments"` vs `"agent-sessions"`), `LinearWebhookClient` HMAC verification, `Comment`/`AgentSessionEvent`/`Reaction` webhook types, API key vs OAuth modes with auto-refresh, multi-tenant per-organization installation via `AsyncLocalStorage`, agent session activities (`Response`/`Thought`/`Action`/`Error`), append-only semantics (cannot edit/delete agent activities), streaming flush strategy
- `.resources/chat/packages/adapter-linear/src/format.ts` — Linear markdown format converter
- `.resources/hermes/agent/credential_pool.py` — Hermes credential pooling with `fill_first`/`round_robin`/`random`/`least_used` strategies and auto-refresh; relevant for Linear OAuth token management patterns (though credential pooling is deferred beyond v1)
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "linear" --topic chat-sdk` for Linear-specific patterns

### Related ADRs

- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Linear relies on one provider runtime multiplexing many owned bridge instances or org modes
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Linear implementation must stay inside the approved bridge v1 surface while expressing its mode split through provider config

## Deliverables

- Production Linear provider extension built on the shared bridge substrate
- Linear ingress and delivery mapping within the approved bridge v1 contract
- Provider-specific conformance and mode-aware coverage for Linear
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for Linear provider behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Linear provider config validation accepts supported mode combinations and rejects malformed or conflicting mode settings
  - [x] Linear ingress payloads map into the expected bridge routing identity for supported provider modes
  - [x] Linear outbound delivery mapping preserves the target context needed for follow-up comment or session responses
- Integration tests:
  - [x] a provider-scoped Linear runtime ingests supported events for owned bridge instances successfully
  - [x] Linear outbound delivery posts responses and reports state transitions through the shared runtime path for each supported mode
  - [x] Linear provider passes the shared conformance harness plus provider-mode scenarios
- Test coverage target: >=80% (`go test -count=1 ./extensions/bridges/linear -cover` => `80.2%`)
- All tests pass (`go test -tags integration ./internal/extension -run 'TestLinearProvider' -count=1`, `make verify`)

## Success Criteria

- All tests passing
- Test coverage >=80%
- Linear validates provider-owned mode switching inside `provider_config`
- Shared bridge semantics remain clean while provider-specific behavior stays local to the provider
