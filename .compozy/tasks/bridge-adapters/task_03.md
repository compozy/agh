---
status: completed
title: "Expand bridge v1 event and delivery contracts"
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 03: Expand bridge v1 event and delivery contracts

## Overview

Make the bridge protocol honest about the v1 scope approved in the ADRs. This task evolves inbound and outbound bridge contracts so typed interaction families, edit/delete behavior, and delivery metadata are first-class instead of leaking through opaque metadata blobs.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST extend the bridge core delivery and ingest types to represent the required bridge v1 conversational contract from the TechSpec "Bridge V1 Scope" section.
2. MUST model typed optional interaction families for `command`, `action`, and `reaction` explicitly in the protocol shape rather than hiding them in opaque metadata.
3. MUST support edit and delete semantics for previously delivered messages in the daemon-owned delivery contract.
4. SHOULD keep non-portable provider-specific payloads isolated behind typed extension points so the portable bridge contract remains coherent.
</requirements>

## Subtasks
- [x] 3.1 Extend inbound bridge event types for typed interaction families and provider-owned metadata
- [x] 3.2 Extend outbound delivery event types for edit and delete semantics alongside textual streaming
- [x] 3.3 Update contract validation and mapping helpers to enforce the new bridge v1 event surface
- [x] 3.4 Add unit coverage for typed event validation and delivery serialization

## Implementation Details

Follow the TechSpec sections "Bridge V1 Scope", "Required Conversational Contract", and "Typed Optional Interaction Contract". This task should stop at type-system and contract evolution; it should not yet change Host API authorization or provider runtime boot.

### Relevant Files
- `internal/bridges/types.go` — Current inbound envelope is still mostly message-and-attachment oriented
- `internal/bridges/delivery_types.go` — Outbound delivery events currently cover textual streaming only
- `internal/bridges/target.go` — Typed delivery target behavior must remain coherent as event families expand
- `internal/api/contract/bridges.go` — Shared transport mapping will need to serialize the expanded bridge protocol

### Dependent Files
- `internal/extension/host_api_bridges.go` — Host API ingest later needs to validate the richer event families
- `internal/extension/bridge_delivery_notifier.go` — Delivery projection later needs to emit the expanded event types
- `sdk/typescript/src/host-api.ts` — SDK host API typing may need the expanded bridge surface if the repo keeps parity

### Reference Sources (.resources/)
- `.resources/chat/packages/chat/src/types.ts` — Chat-SDK `MessageData` interface: `id`, `threadId`, `text`, `formatted` (mdast AST), `author`, `attachments`, `isMention`, `raw`; primary inbound event shape reference
- `.resources/chat/packages/chat/src/types.ts` — Chat-SDK handler registrations: `onAction`, `onReaction`, `onSlashCommand`, `onModalSubmit`; defines the typed interaction families v1 must model
- `.resources/chat/packages/chat/src/types.ts` — Chat-SDK `PostableMessage` union and `StreamChunk` types (`MarkdownTextChunk`, `TaskUpdateChunk`); reference for delivery event families
- `.resources/chat/packages/chat/src/message.ts` — Chat-SDK message normalization and attachment handling
- `.resources/hermes/gateway/platforms/base.py` — Hermes `MessageEvent` dataclass with `message_type` enum (`TEXT`, `PHOTO`, `COMMAND`, etc.) and `SendResult` with `retryable` flag; alternative normalized event design
- `.resources/goclaw/internal/channels/events.go` — GoClaw agent event routing (`AgentEventRunStarted`, `ChatEventThinking`, `AgentEventToolCall`); shows delivery event-to-streaming mapping

### Related ADRs
- [ADR-003: Bridge V1 Scope Instead of Full Chat-SDK Parity](adrs/adr-003.md) — Defines the exact v1 contract this task must encode

## Deliverables
- Expanded bridge ingest and delivery core types for the approved bridge v1 scope
- Validation and mapping helpers for typed interactions and edit/delete delivery
- Updated transport mappings for the new bridge event families
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for contract round-trip behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] typed `command`, `action`, and `reaction` inbound events validate the required identity and payload fields
  - [x] edit and delete outbound delivery events validate previously delivered message identifiers correctly
  - [x] unsupported event-family combinations are rejected before transport mapping
  - [x] provider metadata round-trips without changing the typed bridge event family selection
- Integration tests:
  - [x] an inbound typed interaction event survives transport mapping into the daemon-owned bridge contract unchanged
  - [x] a delivery request containing edit or delete semantics round-trips through the shared bridge contract without losing target information
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Bridge v1 can model typed interactions and edit/delete semantics without opaque protocol shortcuts
- The daemon-owned bridge contract matches the approved v1 scope
