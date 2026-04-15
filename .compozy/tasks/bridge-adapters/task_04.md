---
status: pending
title: "Implement provider-scoped Host API instance management and authorization"
type: backend
complexity: critical
dependencies:
  - task_01
  - task_02
  - task_03
---

# Task 04: Implement provider-scoped Host API instance management and authorization

## Overview

The current Host API trusts a single runtime-bound bridge instance and cannot serve provider-scoped runtimes. This task opens the Host API surface needed by multiplexing providers while keeping daemon ownership of instance state, routing, ingest dedup, and delivery authorization.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add the provider-scoped Host API methods required by the TechSpec "Required Host API Surface" section so a provider runtime can list or fetch owned bridge instances, ingest events by `bridge_instance_id`, and report per-instance state or degradation.
2. MUST replace single-instance Host API authorization with provider-owned instance authorization tied to the negotiated provider runtime context.
3. MUST preserve daemon-side route ownership, ingest dedup, and state-transition validation when multiple bridge instances share one extension session.
4. SHOULD keep Host API errors stable and operator-facing so providers can distinguish invalid instance ownership, unavailable instances, and retryable failures.
</requirements>

## Subtasks
- [ ] 4.1 Add provider-scoped Host API methods and request/response types for owned-instance lookup and synchronization
- [ ] 4.2 Refactor bridge Host API authorization to validate `bridge_instance_id` against provider ownership instead of one bound instance
- [ ] 4.3 Update ingest, state-reporting, and routing flows to use provider-scoped authorization and the expanded bridge v1 contract
- [ ] 4.4 Add integration coverage for owned-instance access, unauthorized access, and multi-instance ingestion

## Implementation Details

Follow the TechSpec sections "Host API and Runtime Changes", "Required Host API Surface", and "Operational Requirements". This task should stop at Host API surface and authorization; it should not yet introduce the shared SDK or provider implementations.

### Relevant Files
- `internal/extension/host_api_bridges.go` — Current bridge Host API authorization is explicitly single-instance
- `internal/extension/protocol/host_api.go` — Host API method registry must grow for provider-scoped runtime support
- `internal/extension/contract/host_api.go` — Shared Host API contracts must carry any new instance-management payloads
- `internal/extension/host_api.go` — Runtime context plumbing and request dispatch need to carry provider-scoped bridge ownership

### Dependent Files
- `sdk/typescript/src/host-api.ts` — SDK clients may need the new Host API methods for bridge providers
- `sdk/examples/telegram-reference/main.go` — The reference/conformance path later uses the new Host API surface
- `internal/extension/host_api_integration_test.go` — Bridge Host API integration coverage needs the new authorization model

### Reference Sources (.resources/)
- `.resources/chat/packages/chat/src/chat.ts` — Chat-SDK `processMessage()` flow: acquire lock → check subscription → route to handler; shows authorization/routing decisions the Host API parallels
- `.resources/goclaw/internal/channels/channel.go` — GoClaw `IsAllowed(senderID)` per-channel authorization check
- `.resources/hermes/gateway/run.py` — Hermes `_is_user_authorized()` with per-platform allowlist maps and `platform_allow_all_map`

### Related ADRs
- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Requires provider-owned instance management over the Host API
- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Assumes providers ingest inbound webhook events through explicit Host API calls tied to `bridge_instance_id`

## Deliverables
- Provider-scoped Host API methods and contracts for owned bridge-instance management
- Refactored bridge Host API authorization keyed by provider ownership instead of a single bound instance
- Updated ingest and state-reporting handlers for multi-instance provider runtimes
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for authorization, ownership validation, and multi-instance ingest **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] provider-scoped authorization accepts a bridge instance owned by the runtime's extension and rejects an instance owned by another extension
  - [ ] Host API request validation rejects missing or mismatched `bridge_instance_id` values for provider-scoped methods
  - [ ] per-instance state reporting rejects operator-controlled disabled transitions where the contract forbids them
- Integration tests:
  - [ ] a provider runtime can fetch or list the bridge instances owned by its extension session
  - [ ] an inbound event for an owned `bridge_instance_id` ingests successfully when a sibling instance shares the same provider runtime
  - [ ] an inbound event for a non-owned `bridge_instance_id` fails with a stable authorization or not-found error
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Provider runtimes can manage many owned bridge instances through the Host API
- Host API bridge authorization no longer assumes a single process-bound instance
