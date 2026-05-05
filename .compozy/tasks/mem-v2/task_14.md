---
status: pending
title: Public Memory Contract Surface
type: backend
complexity: critical
dependencies:
  - task_05
  - task_06
  - task_07
  - task_10
  - task_11
  - task_12
  - task_13
---

# Task 14: Public Memory Contract Surface

## Overview

Define the full public request/response contract for Memory v2 before transport-specific route work begins. This task is where the new scope/tier semantics, controller-backed mutations, recall outputs, ledger metadata, and provider/config payloads become canonical public types for CLI, HTTP, UDS, generated TS consumers, and docs.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Public Interfaces / Types`, `API Endpoints`, `Agent Manageability Plan`, and `Config Lifecycle`.
- ACTIVATE `agh-contract-codegen-coship`, `agh-code-guidelines`, and `golang-pro` before editing contract/spec code.
- MINIMIZE CODE churn outside public DTO/spec surfaces; route implementation comes later.
- TESTS REQUIRED: contract schema coverage, redaction safety, required/optional field semantics, and transport-agnostic shape assertions must ship here.
- NO WORKAROUNDS: do not leave old two-scope or legacy-verb payloads alive beside the new contract.
</critical>

<requirements>
- MUST define canonical public DTOs for Memory v2 CRUD, recall, decisions/revert, dreaming, sessions/ledger, provider/config metadata, and scope/tier selection where approved by the TechSpec.
- MUST align memory contract changes across operator settings payloads and agent-manageability surfaces.
- MUST keep secret/redacted fields out of public payloads and reflect deterministic error semantics where required.
- MUST update OpenAPI/spec-facing sources so later codegen consumes the same contract.
- MUST remove or rename old payload shapes that the TechSpec explicitly hard-cuts.
</requirements>

## Subtasks
- [ ] 14.1 Expand memory-related DTOs and payloads in the shared contract package.
- [ ] 14.2 Update spec/OpenAPI-facing sources to match the new payload shapes.
- [ ] 14.3 Add focused contract tests for field presence, redaction, and backward hard cuts.
- [ ] 14.4 Confirm settings, ledger, recall, and provider payloads align to the repaired TechSpec.

## Implementation Details

See TechSpec `Public Interfaces / Types`, `API Endpoints`, and `Development Sequencing` step 25. This task ends when public shapes are canonical and codegen-ready, not when every route or command is wired.

### Relevant Files
- `internal/api/contract/contract.go` — shared memory DTOs and payloads.
- `internal/api/contract/responses.go` — shared response envelopes and health/status shapes.
- `internal/api/contract/settings.go` — memory settings payloads that must align with the new public model.
- `internal/api/spec/spec.go` — spec/OpenAPI shaping for generated consumers.
- `internal/api/contract/contract_test.go` — payload shape and redaction coverage.
- `internal/api/spec/spec_test.go` — spec-facing contract assertions.

### Dependent Files
- `openapi/agh.json` — later codegen task depends on these finalized public shapes.
- `internal/api/core/memory.go` — later transport task binds shared handlers to this contract.
- `internal/cli/memory.go` — later CLI hard cut depends on final names and payload fields.
- `web/src/generated/agh-openapi.d.ts` — later generated consumer refresh depends on this contract.
- `.compozy/tasks/mem-v2/task_15.md` — codegen task depends on this public contract completing.

### Related ADRs
- [ADR-002: Three Scopes with Agent Two-Tier](adrs/adr-002.md) — public scope/tier semantics.
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — decision/WAL payload semantics.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — recall payload semantics.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: public provider-related payloads must be sufficient for extension/host surfaces without leaking private runtime types.
- Agent manageability: this task defines the canonical machine-readable surfaces that CLI/HTTP/UDS/native-tool adapters will expose.
- Config lifecycle: this task reflects the backend config/settings truth in public payload form and must stay aligned with `task_13`.

### Web/Docs Impact

- `web/`: generated TS consumers and memory/settings/session pages will change after codegen, but no UI code is edited in this task.
- `packages/site`: API/CLI/runtime docs and generated references depend on these finalized payloads; docs updates are deferred.

## Deliverables

- Final public Memory v2 DTOs and response envelopes in `internal/api/contract`.
- Updated spec/OpenAPI-facing sources ready for regeneration.
- Contract tests covering redaction, field shapes, and hard-cut removals.

## Tests

- Unit tests:
  - [ ] Scope/tier, decision, recall, ledger, and provider payloads marshal with the approved field set only.
  - [ ] Secret or redacted fields remain absent from all public memory payloads.
  - [ ] Hard-cut legacy payload names or fields are no longer accepted where the TechSpec removes them.
- Integration tests:
  - [ ] Spec/OpenAPI generation inputs reflect the same shapes as the shared contract package.
  - [ ] Downstream transport and consumer packages can compile against the new contract without ad-hoc shims.
- Test coverage target: >=80%.
- All tests must pass.

## References

- `.resources/hermes/tools/memory_tool.py`
- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/codex/codex-rs/memories/write/src/control.rs`
- `.resources/claude-code/memdir/findRelevantMemories.ts`

## Success Criteria

- All tests passing.
- Test coverage >=80%.
- Memory v2 has one canonical public contract for all downstream surfaces.
- Later codegen, route, CLI, web, and docs tasks can build from this contract without renaming churn.

