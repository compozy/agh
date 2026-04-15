---
status: completed
title: "Implement the Discord provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 11: Implement the Discord provider extension

## Overview

Implement Discord as another interaction-heavy provider with tighter timing constraints around ingress acknowledgment. This task validates that the shared runtime and bridge v1 contract can support Discord message delivery and interaction handling without reopening the protocol scope.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a Discord provider using the shared provider runtime and bridge Host API contract.
2. MUST support Discord message ingress and the approved typed optional interaction families where Discord surfaces them in bridge v1.
3. MUST honor Discord interaction verification and acknowledgment timing constraints without bypassing shared ingress-hardening or bridge-contract validation.
4. SHOULD provide another concrete proof that the narrowed bridge v1 interaction model works across more than one provider.
</requirements>

## Subtasks

- [x] 11.1 Create the Discord provider runtime and manifest on top of `internal/bridgesdk`
- [x] 11.2 Implement Discord webhook or interaction verification and bridge-event mapping
- [x] 11.3 Implement Discord outbound delivery, edit or delete behavior, and provider state reporting
- [x] 11.4 Add conformance and interaction timing coverage for Discord

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "Discord", "Typed Optional Interaction Contract", and "Operational Requirements". This task should stay within the approved bridge v1 scope and reuse the shared SDK for runtime and ingress concerns.

### Relevant Files

- `internal/bridges/types.go` — Discord exercises typed bridge event families and target identity handling
- `internal/bridges/delivery_types.go` — Discord delivery behavior depends on the approved edit or delete semantics
- `internal/extensiontest/bridge_adapter_harness.go` — Discord should pass the shared conformance harness
- `internal/extension/protocol/host_api.go` — Discord runtime depends on the provider-scoped Host API methods

### Dependent Files

- `extensions/bridges/discord/*` — New Discord provider package tree should live here if the repo follows the TechSpec layout
- `internal/extension/bridge_delivery_integration_test.go` — Discord can later serve as an additional interaction-delivery integration target
- `docs/plans/2026-04-15-bridge-adapters-design.md` — Discord results may inform later bridge v1 capability notes

### Reference Sources (.resources/)

- `.resources/chat/packages/adapter-discord/src/index.ts` — **Primary reference**: Chat-SDK Discord adapter; Ed25519 signature verification, Interactions API (3-second ACK deadline), slash commands, modal interactions, thread support, rich embeds, `applicationId` config
- `.resources/chat/packages/adapter-discord/src/format.ts` — Discord embed format converter (mdast ↔ Discord embeds)
- `.resources/goclaw/internal/channels/discord/discord.go` — GoClaw Discord channel: discordgo gateway events, typing controller, placeholder messages with dedup, group history flushing; Go-native Discord reference
- `.resources/hermes/gateway/platforms/discord.py` — Hermes Discord adapter (2300+ lines): voice channel integration, slash vs prefix commands, thread participation tracker, reconnection with backoff, emoji mapping, multi-platform proxy mode
- `.resources/openclaw/extensions/discord/src/` — OpenClaw Discord extension: lazy-loaded runtime, outbound send, inbound handling, components (buttons/menus/forms), probe health checks, approval UI; most complete plugin-architecture reference
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "discord" --topic chat-sdk` for Discord-specific patterns

### Related ADRs

- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Discord requires hardened webhook ingress and explicit recovery behavior
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Discord validates the approved v1 interaction family boundaries

## Deliverables

- Production Discord provider extension built on the shared bridge substrate
- Discord ingress, interaction, and delivery mappings within the bridge v1 contract
- Provider-specific conformance and timing coverage for Discord
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for Discord provider behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Discord interaction verification rejects invalid public-key signatures
  - [x] Discord interaction payloads map into typed bridge command or action events with stable target identity
  - [x] Discord reaction payloads map into typed bridge reaction events where the provider supports them
  - [x] Discord delivery mapping validates edit or delete operations against previously delivered message identifiers
- Integration tests:
  - [x] a provider-scoped Discord runtime ingests interaction payloads within the required acknowledgment timing envelope
  - [x] Discord outbound delivery posts or edits bridge responses successfully for owned bridge instances
  - [x] Discord provider passes the shared conformance harness plus provider-specific interaction-timing scenarios
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Discord validates the shared runtime under interaction timing pressure
- The bridge v1 interaction model remains sufficient across a second interaction-heavy provider
