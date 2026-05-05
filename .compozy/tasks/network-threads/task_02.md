---
status: pending
title: Network Wire Model, Validation, and Hard-Cut Symbol Deletion
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 02: Network Wire Model, Validation, and Hard-Cut Symbol Deletion

## Overview

Implement the core runtime wire model for conversation containers. This task replaces flat `kind:"direct"` and `interaction_id` semantics with `surface`, `thread_id`, `direct_id`, and `work_id` at the `internal/network` boundary.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and the task memory before editing.
- ACTIVATE `nats`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, and `testing-anti-patterns` before Go changes.
- REFERENCE TECHSPEC for validation rules; do not add compatibility readers or aliases.
- FOCUS ON runtime envelope correctness, not persistence, API contracts, or web behavior.
- TESTS REQUIRED for every safety invariant enforced in `internal/network`.
- NO WORKAROUNDS: delete obsolete symbols instead of translating them.
</critical>

<requirements>
- MUST add first-class `SurfaceThread` and `SurfaceDirect` runtime concepts.
- MUST replace `Envelope.InteractionID` with `Envelope.WorkID`.
- MUST delete `KindDirect` and `DirectBody` from active runtime code.
- MUST reject `interaction_id`, `kind:"direct"`, `surface` without matching container ID, and container IDs without `surface`.
- MUST reject conversation fields on `greet` and `whois`.
- MUST require `work_id` for `receipt` and `trace`.
- MUST preserve raw claim-token rejection in body and extension fields.
- MUST update RFC 004 canonical signed-content tests for new nullable field behavior.
</requirements>

## Subtasks

- [ ] 2.1 Update envelope structs, constants, enums, marshal/unmarshal behavior, and JSON field names.
- [ ] 2.2 Implement symmetric container validation for `surface`, `thread_id`, and `direct_id`.
- [ ] 2.3 Delete `kind:"direct"` handling and ensure normal direct-room chat is `kind:"say"` with `surface:"direct"`.
- [ ] 2.4 Rename network lifecycle-facing symbols from interaction to work where they live in `internal/network`.
- [ ] 2.5 Update unit tests for all legacy rejections, container invariants, raw token rejection, and trust canonicalization.

## Implementation Details

Keep this task focused on pure wire/runtime validation. Store-backed thread/direct/work creation happens in later tasks.

### Relevant Files

- `internal/network/envelope.go` - envelope fields and kind/surface constants.
- `internal/network/validate.go` - validation invariants and raw-token rejection.
- `internal/network/lifecycle.go` - rename lifecycle-facing work concepts used by validators.
- `internal/network/router.go` - remove `KindDirect` assumptions from dispatch decisions where required to compile.
- `internal/network/stats.go` - update runtime counters to accept surface fields without final metrics wiring.
- `internal/network/validate_test.go` - core validator coverage.
- `internal/network/lifecycle_test.go` - lifecycle rename coverage touched by this task.

### Dependent Files

- `internal/api/contract/contract.go` - task_08 exposes equivalent public DTOs.
- `internal/store/types.go` - task_04 adds store DTOs.
- `internal/skills/bundled/skills/agh-network/SKILL.md` - task_12 updates agent-facing guidance.

### Related ADRs

- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - hard-cut rename.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - kind/surface split.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: establish protocol primitives that extension Host API and native tools will expose later.
- Agent manageability: no CLI/HTTP/UDS changes yet; this task ensures later surfaces cannot bypass runtime validation.
- Config lifecycle: no new config keys.

### Web/Docs Impact

- Web impact: generated types are not updated in this task; task_08 handles codegen.
- Docs impact: RFC docs from task_01 must be treated as source of truth.

## Deliverables

- Runtime envelope model with `surface`, `thread_id`, `direct_id`, and `work_id`.
- Deleted active `KindDirect`, `DirectBody`, and `InteractionID` runtime symbols.
- Validator tests for every new and deleted field path.
- RFC 004 canonicalization tests that distinguish absent nullable fields from present zero-valued fields.

## Tests

- Unit tests:
  - [ ] `surface:"thread"` requires `thread_id` and rejects `direct_id`.
  - [ ] `surface:"direct"` requires `direct_id` and rejects `thread_id`.
  - [ ] `thread_id` or `direct_id` without `surface` is invalid.
  - [ ] `greet` and `whois` reject `surface`, `thread_id`, `direct_id`, and `work_id`.
  - [ ] `receipt` and `trace` require `work_id`.
  - [ ] Legacy `interaction_id` and `kind:"direct"` payloads fail closed.
  - [ ] Raw `claim_token` remains rejected in body and extension fields.
  - [ ] RFC 004 canonical bytes change when `surface`, `thread_id`, `direct_id`, or `work_id` changes.
- Integration tests:
  - [ ] Existing router tests compile against the new surface model without compatibility branches.
- Test coverage target: >=80% for touched `internal/network` packages.
- All tests must pass.

## Success Criteria

- `internal/network` can no longer construct or validate active messages using `interaction_id` or `kind:"direct"`.
- Every conversation-bearing message has exactly one container identity.
- Later tasks can build persistence and public contracts on a strict runtime boundary.
