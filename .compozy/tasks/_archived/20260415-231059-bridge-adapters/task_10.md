---
status: completed
title: "Implement the Slack provider extension"
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 10: Implement the Slack provider extension

## Overview

Implement Slack as the first provider that materially exercises the optional typed interaction families in bridge v1. This task validates command, action, and reaction handling on top of the shared provider runtime and hardened webhook stack.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a Slack provider using the shared bridge SDK and provider-scoped Host API contract.
2. MUST support Slack message events plus the approved typed optional interaction families for commands, actions, and reactions.
3. MUST handle Slack signing-secret verification, webhook or events ingress hardening, and outbound delivery semantics within the approved bridge v1 scope.
4. SHOULD validate that the narrowed bridge v1 interaction model is sufficient for at least one interaction-heavy provider.
</requirements>

## Subtasks

- [x] 10.1 Create the Slack provider runtime and manifest on top of `internal/bridgesdk`
- [x] 10.2 Implement Slack event and interaction mapping into bridge v1 message, command, action, and reaction events
- [x] 10.3 Implement Slack outbound delivery, edit or delete behavior, and per-instance state reporting
- [x] 10.4 Add conformance and provider-specific interaction coverage for Slack

## Implementation Details

Follow the TechSpec sections "Provider Reference Notes", "Slack", "Typed Optional Interaction Contract", and "Operational Requirements". This task should implement Slack only and rely on the shared substrate for runtime boot, ingress hardening, and error classification.

### Relevant Files

- `internal/bridges/delivery_types.go` — Slack exercises the typed interaction and delivery contract introduced earlier
- `internal/extension/protocol/host_api.go` — Slack runtime consumes the provider-scoped bridge Host API surface
- `internal/extensiontest/bridge_adapter_harness.go` — Slack should pass the shared conformance harness
- `sdk/examples/telegram-reference/main.go` — Existing provider runtime patterns may still be useful as structural reference, not as copied implementation

### Dependent Files

- `extensions/bridges/slack/*` — New Slack provider package tree should live here if the repo follows the TechSpec layout
- `internal/extension/bridge_delivery_integration_test.go` — Delivery-path integration coverage can later reuse Slack as an interaction-heavy provider
- `docs/plans/2026-04-15-bridge-adapters-design.md` — Slack may inform follow-up notes about typed interaction coverage

### Reference Sources (.resources/)

- `.resources/chat/packages/adapter-slack/src/index.ts` — **Primary reference**: Chat-SDK Slack adapter; Events API, Web API, signing-secret HMAC-SHA256, slash commands, actions (Block Kit), reactions, streaming via progressive `chat.update`, modals via `views.open`, ephemeral via `chat.postEphemeral`, multi-workspace OAuth
- `.resources/chat/packages/adapter-slack/src/format.ts` — Slack Block Kit format converter (mdast ↔ Block Kit)
- `.resources/hermes/gateway/platforms/slack.py` — Hermes Slack adapter; mentions handling, approval workflows, thread logic
- `.resources/openclaw/extensions/discord/src/channel.ts` — OpenClaw interaction pattern (lazy-loaded runtime, component builders); applicable to Slack interaction UI design
- **KB Vault**: `.resources/chat/.kb/vault/chat-sdk/` — Use `kb search "slack" --topic chat-sdk` for Slack-specific patterns

### Related ADRs

- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Slack ingress and outbound behavior follow the hardened provider communication model
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Slack validates the approved optional interaction subset

## Deliverables

- Production Slack provider extension built on the shared bridge substrate
- Slack message and typed interaction mapping for commands, actions, and reactions
- Provider-specific conformance and interaction coverage for Slack
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for Slack provider behavior **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Slack command payloads map into typed bridge `command` events with stable target identity
  - [x] Slack action payloads map into typed bridge `action` events without losing provider-specific identifiers needed for follow-up delivery
  - [x] Slack reaction events map into typed bridge `reaction` events and reject malformed reaction payloads
  - [x] Slack request verification rejects invalid signing-secret signatures before event handling
- Integration tests:
  - [x] a provider-scoped Slack runtime ingests both standard message events and command or action interactions for owned bridge instances
  - [x] Slack delivery posts or edits messages successfully under the bridge v1 contract
  - [x] Slack provider passes the shared conformance harness plus provider-specific interaction scenarios
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Slack validates the typed optional interaction subset in the approved bridge v1 protocol
- The provider-scoped substrate works for an interaction-heavy platform
