---
status: completed
title: HTTP and UDS Route Parity
type: backend
complexity: high
dependencies:
  - task_14
---

# Task 16: HTTP and UDS Route Parity

## Overview

Bind the new Memory v2 contract to shared handlers and register it consistently across HTTP and UDS transports. This task makes the daemon’s public memory surface truthful again by replacing legacy route shapes and payload assumptions with the final Slice 1 behavior.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `API Endpoints`, `Agent Manageability Plan`, and `Development Sequencing` step 26.
- ACTIVATE `agh-code-guidelines` and `golang-pro` before editing production Go.
- MINIMIZE CODE churn outside shared handler code and route registration.
- TESTS REQUIRED: handler happy/failure paths, HTTP/UDS parity, deterministic errors, and redaction-safe responses must ship here.
- NO WORKAROUNDS: do not maintain a legacy route family beside the approved Slice 1 route shape.
</critical>

<requirements>
- MUST bind Memory v2 behavior into shared core handlers first and keep HTTP/UDS as thin route registration layers.
- MUST update both HTTP and UDS route families to the final Slice 1 memory surface and payload semantics.
- MUST preserve deterministic error mapping and structured output across both transports.
- MUST add or update parity coverage so the two transports cannot drift silently.
- MUST keep the transport layer free of duplicated decision/recall business logic.
</requirements>

## Subtasks
- [x] 16.1 Update shared memory handlers to consume the finalized Memory v2 services and contract.
- [x] 16.2 Register the final route families in both HTTP and UDS transports.
- [x] 16.3 Add handler tests for happy path, validation errors, redaction, and unsupported operations.
- [x] 16.4 Add or refresh HTTP/UDS parity coverage for the affected routes.

## Implementation Details

See TechSpec `API Endpoints`, `Agent Manageability Plan`, and `Development Sequencing` step 26. Shared handler logic belongs in `internal/api/core`; HTTP and UDS should differ only in registration and transport concerns.

### Relevant Files
- `internal/api/core/memory.go` — shared memory handlers to update for Slice 1 services and payloads.
- `internal/api/httpapi/routes.go` — HTTP route registration for the memory family.
- `internal/api/udsapi/routes.go` — UDS route registration for the memory family.
- `internal/api/udsapi/memory.go` — UDS-specific memory transport helpers/tests.
- `internal/api/httpapi/memory_test.go` — HTTP memory handler coverage.
- `internal/api/udsapi/transport_parity_integration_test.go` — parity guardrails to extend.

### Dependent Files
- `internal/cli/memory.go` — later CLI hard cut depends on the final daemon route behavior.
- `web/src/systems/knowledge/adapters/knowledge-api.ts` — later web task depends on the final route family and payloads.
- `packages/site/content/runtime/api-reference/memory.mdx` — later docs/reference task depends on final route semantics.
- `.compozy/tasks/mem-v2/task_17.md` — CLI hard cut depends on final route semantics.
- `.compozy/tasks/mem-v2/task_24.md` — API reference co-ship depends on this transport truth.

### Related ADRs
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — mutation route semantics.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — recall/search route semantics.
- [ADR-006: Session Ledger Hybrid (events.db Live + ledger.jsonl Forensic)](adrs/adr-006.md) — lineage/ledger route semantics.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none directly — checked surfaces are provider hooks and extension host interfaces, which are handled elsewhere.
- Agent manageability: this task is the transport owner for the final machine-readable HTTP/UDS memory surface.
- Config lifecycle: transport responses must reflect the backend memory settings/config truth established earlier.

### Web/Docs Impact

- `web/`: downstream adapters and generated consumers depend on these final routes, but no web code is changed in this task.
- `packages/site`: API reference and runtime docs must be updated later to reflect these route shapes; no docs content changes here.

## Deliverables

- Updated shared memory handlers for the Slice 1 service graph.
- HTTP and UDS route registration aligned to the final memory surface.
- Handler and parity tests covering happy path, failures, and redaction.

## Tests

- Unit tests:
  - [x] Shared handlers return the approved success payloads and deterministic errors for memory operations.
  - [x] Redacted or forbidden fields do not leak through error or history/health payloads.
- Integration tests:
  - [x] HTTP and UDS routes expose the same memory state and payload structure for the same persisted input.
  - [x] `internal/api/httpapi/transport_parity_integration_test.go` and `internal/api/udsapi/transport_parity_integration_test.go` pass with Memory v2 changes.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/codex/codex-rs/memories/write/src/runtime.rs`
- `.resources/claude-code/server`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- HTTP and UDS expose one truthful Memory v2 surface with no legacy route drift.
- Later CLI/web/docs tasks can target stable memory transport semantics.
