---
status: pending
title: Codegen and Generated Consumer Refresh
type: backend
complexity: high
dependencies:
  - task_14
---

# Task 15: Codegen and Generated Consumer Refresh

## Overview

Regenerate the public API artifacts that downstream consumers rely on after the new Memory v2 contract is defined. This task turns the shared contract/spec work into updated OpenAPI and generated TypeScript outputs so web and docs tasks can build on stable machine-generated types instead of hand-maintained drift.

<critical>
- ALWAYS READ `_techspec.md` and ADR-001 through ADR-012 before implementation.
- REFERENCE the TechSpec sections `Public Interfaces / Types`, `Web/Docs Impact`, and `Development Sequencing` step 28.
- ACTIVATE `agh-contract-codegen-coship` before editing codegen inputs or generated artifacts.
- MINIMIZE CODE churn outside codegen inputs and generated outputs.
- TESTS REQUIRED: codegen success, drift-free `make codegen-check`, and consumer compile health must ship here.
- NO WORKAROUNDS: do not hand-edit generated TS types to paper over contract drift.
</critical>

<requirements>
- MUST regenerate OpenAPI and generated TypeScript artifacts from the finalized Memory v2 contract.
- MUST update any generated-consumer inputs or wrappers that need to understand the new memory shapes.
- MUST keep generated outputs consistent with `make codegen` and `make codegen-check`.
- MUST avoid introducing handwritten divergence between OpenAPI and TS consumers.
- MUST leave web/docs tasks with stable generated types and route schemas.
</requirements>

## Subtasks
- [ ] 15.1 Regenerate OpenAPI and generated TS artifacts from the new public memory contract.
- [ ] 15.2 Update any thin consumer wrappers that depend on generated-memory route typing.
- [ ] 15.3 Add or update focused tests/assertions for codegen drift and consumer health.
- [ ] 15.4 Confirm generated outputs are committed in the same change as the contract updates.

## Implementation Details

See TechSpec `Web/Docs Impact`, `Impact Analysis`, and `Development Sequencing` step 28. This task exists so web/docs work can depend on stable generated inputs rather than manual type edits.

### Relevant Files
- `internal/codegen/openapits/generate.go` — OpenAPI to TS generation entry point.
- `openapi/agh.json` — generated OpenAPI artifact.
- `web/src/generated/agh-openapi.d.ts` — generated TypeScript contract consumed by the web app.
- `web/src/lib/api-contract.ts` — web-side API wrapper over the generated OpenAPI types.
- `makefile` / codegen targets — command path for `make codegen` and `make codegen-check`.

### Dependent Files
- `web/src/systems/knowledge/**` — later knowledge UI task depends on refreshed generated types.
- `web/src/systems/settings/**` — later settings UI task depends on refreshed generated types.
- `packages/site/content/runtime/api-reference/**` — later docs/reference task depends on the regenerated contract truth.
- `.compozy/tasks/mem-v2/task_20.md` — web knowledge task depends on this refresh.
- `.compozy/tasks/mem-v2/task_24.md` — API reference/regeneration task depends on drift-free outputs.

### Related ADRs
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — recall payload implications for generated consumers.
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — mutation/decision payload implications for generated consumers.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none directly — checked surfaces are provider/extension host interfaces, which remain runtime code; this task updates generated API consumers only.
- Agent manageability: generated artifacts become the shared source for later web/operator consumption of the same agent-manageable memory contract.
- Config lifecycle: generated outputs must reflect the final memory settings payloads from `task_13`.

### Web/Docs Impact

- `web/`: `web/src/generated/agh-openapi.d.ts` and `web/src/lib/api-contract.ts` are expected to change here.
- `packages/site`: API reference generation depends on the new OpenAPI artifact, but docs content is updated later in `task_24`.

## Deliverables

- Regenerated `openapi/agh.json`.
- Regenerated `web/src/generated/agh-openapi.d.ts`.
- Any required thin consumer refresh for generated API wrappers.
- Codegen drift checks and focused coverage updated.

## Tests

- Unit tests:
  - [ ] Generated-memory route and payload typings line up with the finalized contract.
  - [ ] Consumer wrappers compile against the regenerated types without local patches.
- Integration tests:
  - [ ] `make codegen` regenerates cleanly.
  - [ ] `make codegen-check` passes with no drift after committing the artifacts.
  - [ ] Downstream web typecheck/build inputs accept the generated memory contract.
- Test coverage target: command/codegen validation for all affected generated surfaces.
- All tests must pass.

## References

- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/codex/sdk`
- `.resources/claude-code/server`

## Success Criteria

- All tests passing.
- Generated OpenAPI and TypeScript memory artifacts are refreshed and drift-free.
- Web and docs tasks can consume the new memory contract without handwritten type patches.

