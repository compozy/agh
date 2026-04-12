---
status: completed
title: Implement channel Host API ingest and instance state reporting
type: backend
complexity: high
dependencies:
  - task_02
  - task_04
---

# Task 05: Implement channel Host API ingest and instance state reporting

## Overview

Implement the extension-to-daemon Host API entry points that let channel adapters deliver normalized inbound messages and report channel-instance state changes. This task turns the approved inbound flow into real typed Host API methods with validation, capability enforcement, and idempotent ingress behavior.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add `channels/messages/ingest`, `channels/instances/get`, and `channels/instances/report_state` to the extension Host API with typed request and response contracts.
2. MUST validate channel instance identity, scope, enabled status, and routing-policy expectations before creating prompts or mutating state.
3. MUST enforce idempotent inbound message handling backed by the `channel_ingest_dedup` table from task 01, with a configurable TTL and periodic cleanup so the dedup store does not grow unbounded.
4. MUST serialize route resolution and session creation per routing key so that concurrent inbound messages for the same new routing key do not race to create duplicate sessions, leaving one orphaned and unrouted.
5. SHOULD return explicit invalid-parameter, unavailable, and not-found errors that match the existing Host API error model.
</requirements>

## Subtasks
- [x] 5.1 Add channel Host API method identifiers and typed request/response payloads
- [x] 5.2 Implement Host API handlers for inbound message ingest, instance lookup, and state reporting
- [x] 5.3 Add idempotent ingress handling (backed by dedup table with TTL), serialized per-routing-key route/session resolution, and dedup cleanup
- [x] 5.4 Add unit and integration tests for validation, duplicate suppression, and state transitions

## Implementation Details

Follow the TechSpec sections "Extension Host API", "InboundMessageEnvelope", and "Error handling conventions". This task should stop at the Host API boundary and session prompt initiation; outbound projection and delivery streaming belong to later tasks.

### Relevant Files
- `internal/extension/host_api.go` — Central Host API request dispatch where the new channel methods belong
- `internal/extension/host_api_test.go` — Existing Host API coverage should be expanded for channel method validation and authorization
- `internal/extension/contract/host_api.go` — Shared typed contracts for Host API requests and responses belong here
- `internal/extension/capability.go` — Host API authorization must cover the new channel methods
- `internal/session/manager_prompt.go` — Inbound channel prompts eventually flow through the existing prompt path and need compatible integration

### Dependent Files
- `internal/channels/` — Registry and route-resolution logic from earlier tasks are the core dependency for inbound message processing
- `internal/daemon/boot.go` — Daemon composition later injects the Host API dependencies for channel methods
- `internal/extension/host_api_integration_test.go` — End-to-end Host API behavior should be extended for channel ingest flows

### Related ADRs
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — Requires daemon-owned route and instance resolution before prompting a session
- [ADR-007: Negotiated Channel Delivery Stream for Real-Time Outbound Messaging](adrs/adr-007.md) — Establishes the separation between inbound Host API calls and outbound delivery streaming
- [ADR-008: Bound Secret Injection per Channel Instance](adrs/adr-008.md) — Constrains instance lookup and runtime visibility for channel adapters

## Deliverables
- New typed Host API channel methods for ingest, instance lookup, and state reporting
- Idempotent inbound message handling and registry-backed route/session resolution
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for inbound message processing and duplicate suppression **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `channels/messages/ingest` rejects payloads that omit `channel_instance_id` or required routing fields
  - [x] `channels/messages/ingest` rejects disabled or unknown channel instances before prompting a session
  - [x] Duplicate ingress with the same `idempotency_key` does not create a second prompt or second route mutation
  - [x] `channels/instances/report_state` rejects unsupported state transitions and malformed status payloads
  - [x] `channels/instances/get` does not expose metadata for a different channel instance than the running extension is authorized to serve
  - [x] Concurrent inbound messages for the same new routing key result in exactly one session and one route, not duplicates
  - [x] Expired dedup records are cleaned up and no longer suppress re-ingestion of the same idempotency key
- Integration tests:
  - [x] A fake channel extension can ingest one normalized inbound message and receive a created or reused session association through the registry
  - [x] Retrying the same inbound webhook payload results in one prompt path and one stored route update
  - [x] An adapter-reported `auth_required` state transition becomes visible through the channel instance state model without crashing the Host API handler
  - [x] Two concurrent ingest calls for the same routing key (no prior session) produce one session and one route record
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Channel adapters can send normalized inbound traffic into AGH through typed Host API methods
- Duplicate inbound deliveries are suppressed before they can create duplicate prompts
