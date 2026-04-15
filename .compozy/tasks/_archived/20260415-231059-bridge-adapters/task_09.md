---
status: completed
title: "Implement the Telegram provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 09: Implement the Telegram provider extension

## Overview

Deliver the first production provider on top of the shared bridge substrate to validate the approved architecture end to end. Telegram is the proving ground for provider-scoped runtime boot, webhook ingress, message mapping, delivery, DM policy, and recovery behavior.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a real Telegram provider binary that uses the shared `internal/bridgesdk` runtime instead of the legacy reference-only bootstrap path.
2. MUST support Telegram inbound message mapping, outbound delivery, edit/delete behavior where the platform supports it, webhook verification, and provider-config-driven DM policy.
3. MUST consume the provider-scoped Host API surface with explicit `bridge_instance_id` ownership rather than assuming one instance per process.
4. SHOULD serve as the first conformance-validated provider and reference implementation for later providers.
</requirements>

## Subtasks

- [x] 9.1 Create the Telegram provider runtime using the shared bridge SDK and provider-scoped configuration
- [x] 9.2 Implement Telegram webhook parsing, verification, and bridge-event mapping
- [x] 9.3 Implement Telegram delivery, edit or delete handling, and status-reporting behavior
- [x] 9.4 Add conformance, integration, and recovery coverage for Telegram

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "Telegram", "Operational Requirements", and "Development Sequencing". This task should implement Telegram only; it should not introduce provider-generic runtime features already owned by `internal/bridgesdk`.

### Relevant Files

- `sdk/examples/telegram-reference/main.go` — Closest existing Telegram mapping reference, but it must not remain the production runtime path
- `sdk/examples/telegram-reference/extension.toml` — Current Telegram manifest is the baseline for provider metadata and action requirements
- `internal/extensiontest/bridge_adapter_harness.go` — Telegram should pass the shared provider conformance harness
- `internal/extension/telegram_reference_integration_test.go` — Existing Telegram-focused integration patterns can guide production provider coverage

### Dependent Files

- `extensions/bridges/telegram/*` — New production provider package tree should live here if the repo adopts the TechSpec layout
- `docs/plans/2026-04-15-bridge-adapters-design.md` — Design doc may need follow-up implementation notes once Telegram lands
- `internal/daemon/bridges_test.go` — End-to-end runtime and delivery behavior later depend on a real provider implementation

### Reference Sources (.resources/)

- `.resources/chat/packages/adapter-telegram/src/index.ts` — **Primary reference**: Chat-SDK Telegram adapter; webhook/polling modes, `lockScope: "channel"`, `persistMessageHistory: true`, inline keyboard buttons, forum topic threading, message entities reconstruction
- `.resources/chat/packages/adapter-telegram/src/format.ts` — Telegram markdown format converter (mdast ↔ Telegram entities)
- `.resources/goclaw/internal/channels/telegram/telegram.go` — GoClaw Telegram channel: native HTTP polling + Bot API, group history flushing, dedup, typing indicators; Go-native implementation reference
- `.resources/hermes/gateway/platforms/telegram.py` — Hermes Telegram adapter: long polling with backoff, per-topic thread sessions, voice transcription, media caching, caption merging, format preservation
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "telegram" --topic chat-sdk` for Telegram-specific patterns

### Related ADRs

- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Telegram is the first validation target for the provider-scoped runtime
- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Telegram ingress and outbound behavior must follow the hardened webhook plus REST pattern

## Deliverables

- Production Telegram provider extension built on `internal/bridgesdk`
- Telegram webhook, delivery, DM policy, and state-reporting behavior mapped into the bridge v1 contract
- Conformance and integration evidence for Telegram runtime and recovery flows
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for Telegram provider behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Telegram webhook mapping produces the expected bridge routing identity for direct chats and threaded or forum-style contexts
  - [x] Telegram webhook verification rejects invalid secret or signature state according to the provider config
  - [x] Telegram delivery mapping supports text, edit, and delete behavior within the platform's supported semantics
- Integration tests:
  - [x] a provider-scoped Telegram runtime ingests inbound messages for one owned `bridge_instance_id` and routes them correctly
  - [x] Telegram delivery requests post or edit messages successfully and acknowledge completion through the shared runtime path
  - [x] Telegram provider restart recovers owned-instance state and continues delivery or ingest without violating conformance
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Telegram validates the provider-scoped runtime model end to end
- The repository has one real provider implementation built on the shared bridge substrate
