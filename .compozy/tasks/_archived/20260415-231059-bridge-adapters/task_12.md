---
status: completed
title: "Implement the WhatsApp provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 12: Implement the WhatsApp provider extension

## Overview

Implement the WhatsApp Cloud API provider on top of the shared provider runtime. This task validates verify-challenge handling, webhook signature protection, delivery retries, and direct-message style routing within the approved bridge v1 scope.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a WhatsApp provider using the shared provider runtime and hardened webhook ingress model.
2. MUST support WhatsApp verify-challenge handling, signature validation, inbound message mapping, and outbound delivery behavior consistent with bridge v1.
3. MUST consume provider-specific secret slots and provider config for access tokens, app secrets, and verification configuration without leaking them into generic delivery defaults.
4. SHOULD exercise the shared runtime's rate-limit and retry classification against a provider with explicit API throttling behavior.
</requirements>

## Subtasks

- [x] 12.1 Create the WhatsApp provider runtime and manifest on top of `internal/bridgesdk`
- [x] 12.2 Implement verify-challenge, webhook verification, and inbound bridge-event mapping for WhatsApp
- [x] 12.3 Implement WhatsApp outbound delivery, error classification, and state-reporting behavior
- [x] 12.4 Add conformance and retry or recovery coverage for WhatsApp

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "WhatsApp", and "Operational Requirements". This task should stay within the approved bridge v1 scope and rely on the shared SDK for ingress hardening, dedup, and retry classification.

### Relevant Files

- `internal/bridges/types.go` — WhatsApp must map direct-message style conversations into the daemon-owned bridge identity model
- `internal/bridges/delivery_types.go` — Outbound WhatsApp delivery depends on the approved textual and edit or delete contract where supported
- `internal/extensiontest/bridge_adapter_harness.go` — WhatsApp should pass the shared conformance harness
- `internal/extension/protocol/host_api.go` — WhatsApp runtime depends on the provider-scoped Host API methods

### Dependent Files

- `extensions/bridges/whatsapp/*` — New WhatsApp provider package tree should live here if the repo follows the TechSpec layout
- `internal/extension/bridge_delivery_integration_test.go` — Retry and delivery integration coverage can later include WhatsApp-specific scenarios
- `docs/plans/2026-04-15-bridge-adapters-design.md` — WhatsApp may inform later rate-limit and retry guidance

### Reference Sources (.resources/)

- `.resources/chat/packages/adapter-whatsapp/src/index.ts` — **Primary reference**: Chat-SDK WhatsApp adapter; Cloud API `v21.0`, `accessToken`/`appSecret`/`phoneNumberId`/`verifyToken` config, `lockScope: "channel"`, `persistMessageHistory: true`, interactive buttons, media handling, 4096-char message splitting, verify-challenge GET handler
- `.resources/chat/packages/adapter-whatsapp/src/format.ts` — WhatsApp format converter (mdast ↔ WhatsApp formatting)
- `.resources/hermes/gateway/platforms/whatsapp.py` — Hermes WhatsApp adapter; whatsmeow library integration (different approach but useful for message flow understanding)
- `.resources/goclaw/internal/channels/whatsapp/whatsapp.go` — GoClaw WhatsApp channel; Go-native WhatsApp implementation reference
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "whatsapp" --topic chat-sdk` for WhatsApp-specific patterns

### Related ADRs

- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — WhatsApp directly exercises verify-challenge, signature verification, and retry behavior

## Deliverables

- Production WhatsApp provider extension built on the shared bridge substrate
- WhatsApp verify-challenge, webhook, and delivery mapping into the bridge v1 contract
- Provider-specific conformance and retry coverage for WhatsApp
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for WhatsApp provider behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] WhatsApp verify-challenge requests succeed only when the configured verification token matches
  - [x] WhatsApp signature verification rejects invalid webhook signatures before event mapping
  - [x] WhatsApp inbound payloads map into the expected bridge routing identity and message envelope
  - [x] WhatsApp provider error mapping classifies representative rate-limit and auth failures into the expected shared runtime classes
- Integration tests:
  - [x] a provider-scoped WhatsApp runtime ingests verified webhook traffic for one owned bridge instance successfully
  - [x] WhatsApp outbound delivery posts responses and reports state transitions through the shared runtime path
  - [x] WhatsApp provider passes the shared conformance harness plus retry or rate-limit scenarios
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- WhatsApp validates the hardened webhook plus REST provider pattern
- The shared retry and error-classification substrate works for a rate-limit-sensitive provider
