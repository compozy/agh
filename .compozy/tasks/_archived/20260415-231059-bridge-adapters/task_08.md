---
status: completed
title: "Replace the Telegram reference path with a provider-scoped conformance harness"
type: backend
complexity: high
dependencies:
  - task_02
  - task_04
  - task_05
---

# Task 08: Replace the Telegram reference path with a provider-scoped conformance harness

## Overview

Retire the old instance-scoped reference assumptions before provider work fans out. This task updates or supersedes the current `telegram-reference` and bridge adapter harness so they validate the new provider-scoped runtime, Host API surface, and shared SDK behavior.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST update or replace the current Telegram reference adapter path so it no longer encodes the old single-instance handshake or Host API contract.
2. MUST evolve the reusable bridge conformance harness to validate provider-scoped runtime negotiation, owned-instance access, delivery acknowledgments, and state reporting under the new contract.
3. MUST keep the harness generic enough that real provider binaries can reuse it for conformance evidence later.
4. SHOULD clearly mark any remaining legacy-only scaffolding so it cannot be mistaken for the active bridge v1 path.
</requirements>

## Subtasks

- [x] 8.1 Refactor or replace `telegram-reference` to consume the provider-scoped runtime and Host API surface
- [x] 8.2 Update the bridge adapter harness marker contract and validation rules for provider-scoped runtimes
- [x] 8.3 Add shared conformance scenarios for owned-instance lookup, delivery, and state-reporting behavior
- [x] 8.4 Document the new conformance path and remove ambiguity around legacy reference behavior

## Implementation Details

Follow the TechSpec sections "Impact Analysis", "Testing Approach", and "Development Sequencing". This task should create the conformance baseline only; it should not implement the production Telegram provider yet.

### Relevant Files

- `sdk/examples/telegram-reference/main.go` — Current reference runtime still consumes the old bridge initialize contract
- `sdk/examples/telegram-reference/extension.toml` — Manifest and Host API actions must match the provider-scoped runtime
- `internal/extensiontest/bridge_adapter_harness.go` — Reusable harness currently validates single-instance assumptions
- `sdk/examples/telegram-reference/main_test.go` — Existing reference tests should become provider-scoped conformance evidence

### Dependent Files

- `internal/extension/telegram_reference_integration_test.go` — Integration coverage later needs the updated conformance path
- `sdk/examples/telegram-reference/README.md` — Documentation should point to the new runtime expectations
- `extensions/bridges/telegram/*` — The production Telegram provider later depends on the conformance baseline introduced here

### Reference Sources (.resources/)

- `.resources/chat/packages/adapter-telegram/src/index.ts` — Chat-SDK Telegram adapter: webhook/polling modes, message parsing, Bot API delivery; domain reference for conformance harness test scenarios
- `.resources/hermes/tests/gateway/test_webhook_adapter.py` — Hermes webhook test patterns: HMAC validation, prompt rendering, idempotency, rate limiting; reference for conformance test design

### Related ADRs

- [ADR-001: Provider-Scoped Bridge SDK and Runtime Model](adrs/adr-001.md) — Drives the conformance contract shift from instance-scoped to provider-scoped
- [ADR-002: Hardened Webhook + REST Provider Communication](adrs/adr-002.md) — Conformance cases should include ingress ownership and state-reporting behavior

## Deliverables

- Provider-scoped bridge conformance harness and updated reference path
- Updated marker and validation contract for provider-scoped bridge runtimes
- Documentation clarifying the active conformance path versus any legacy scaffolding
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for provider-scoped conformance scenarios **(REQUIRED)**

## Tests

- Unit tests:
  - [x] conformance validation rejects a runtime handshake that omits provider-scoped bridge context
  - [x] conformance validation rejects owned-instance access or delivery markers that do not match the provider-scoped contract
  - [x] helper cloning and marker parsing support many managed bridge instances without aliasing state
- Integration tests:
  - [x] the updated reference path boots with a provider-scoped runtime and writes conformance markers that validate successfully
  - [x] the harness captures state reporting and delivery acknowledgments for a provider runtime that owns multiple bridge instances
  - [x] legacy single-instance handshake expectations no longer pass against the updated harness
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- The reference and harness path validates the approved provider-scoped contract
- Provider implementations have a reusable conformance baseline before platform-specific work begins
