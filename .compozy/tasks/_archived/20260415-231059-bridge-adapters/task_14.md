---
status: completed
title: "Implement the Google Chat provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 14: Implement the Google Chat provider extension

## Overview

Implement the Google Chat provider on top of the shared runtime while keeping provider-specific event-shape differences contained. This task validates that the bridge substrate can normalize Google Chat webhook or Pub/Sub style ingress into the daemon-owned bridge model.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a Google Chat provider using the shared provider runtime and provider-scoped Host API contract.
2. MUST normalize Google Chat ingress and outbound behavior into the approved bridge v1 contract without requiring a provider-specific daemon protocol fork.
3. MUST use provider config and secret slots to model Google Chat credentials and provider-mode differences cleanly.
4. SHOULD validate that the provider-scoped runtime can absorb event-shape variance while preserving bridge identity and delivery semantics.
</requirements>

## Subtasks

- [x] 14.1 Create the Google Chat provider runtime and manifest on top of `internal/bridgesdk`
- [x] 14.2 Implement Google Chat ingress normalization and bridge-event mapping
- [x] 14.3 Implement Google Chat outbound delivery and state-reporting behavior
- [x] 14.4 Add conformance and event-shape normalization coverage for Google Chat

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "Google Chat", and "Operational Requirements". This task should implement Google Chat only and rely on the shared substrate for runtime, ingress guards, and retry behavior.

### Relevant Files

- `internal/bridges/types.go` — Google Chat needs the daemon-owned bridge event model to normalize provider event shapes
- `internal/extension/protocol/host_api.go` — Google Chat runtime depends on the provider-scoped Host API methods
- `internal/extensiontest/bridge_adapter_harness.go` — Google Chat should pass the shared conformance harness
- `internal/bridges/delivery_types.go` — Google Chat delivery must map into the bridge v1 contract

### Dependent Files

- `extensions/bridges/gchat/*` — New Google Chat provider package tree should live here if the repo follows the TechSpec layout
- `internal/daemon/bridges_test.go` — Google Chat can later serve as an ingress-normalization integration target
- `docs/plans/2026-04-15-bridge-adapters-design.md` — Google Chat may inform later notes about provider mode and event-shape configuration

### Reference Sources (.resources/)

- `.resources/chat/packages/adapter-gchat/src/index.ts` — **Primary reference**: Chat-SDK Google Chat adapter; dual webhook modes (direct Add-ons + Pub/Sub push), JWT bearer token verification, Google Cards v2, ephemeral messages via `privateMessageViewer`, reactions, DM via `spaces.findDirectMessage`/`spaces.setup`, auto-managed Workspace Events subscriptions with 25h TTL, bot user ID learning from annotations
- `.resources/chat/packages/adapter-gchat/src/format.ts` — Google Cards v2 builder (mdast ↔ Cards v2 widgets)
- `.resources/chat/packages/adapter-gchat/src/auth.ts` — GChat auth: service account JWT, Application Default Credentials, domain-wide delegation
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "google chat" --topic chat-sdk` for GChat-specific patterns

### Related ADRs

- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Google Chat ingress still follows the hardened provider communication pattern despite shape variance

## Deliverables

- Production Google Chat provider extension built on the shared bridge substrate
- Google Chat ingress and delivery mapping within the approved bridge v1 contract
- Provider-specific conformance and ingress-normalization coverage for Google Chat
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for Google Chat provider behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Google Chat ingress payloads normalize into the expected bridge message or typed interaction events
  - [x] Google Chat provider config validation accepts the required credential and mode settings while rejecting malformed values
  - [x] Google Chat outbound delivery mapping preserves the target identity needed for follow-up messages
- Integration tests:
  - [x] a provider-scoped Google Chat runtime ingests supported event shapes for owned bridge instances successfully
  - [x] Google Chat outbound delivery posts responses and reports state transitions through the shared runtime path
  - [x] Google Chat provider passes the shared conformance harness plus ingress-normalization scenarios
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Google Chat validates that provider-specific ingress shape differences can still map into the common bridge substrate
- The provider-scoped runtime remains generic enough for Google Chat without protocol forks
