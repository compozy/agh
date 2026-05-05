---
status: pending
title: Web Knowledge Surface
type: frontend
complexity: high
dependencies:
  - task_15
  - task_19
---

# Task 20: Web Knowledge Surface

## Overview

Update the dedicated memory browser in the web app so it reflects Memory v2’s final knowledge model, payloads, and operations. This task owns the `knowledge` route and system modules, including scope/tier awareness, controller-backed actions, and any decision/recall details exposed by the approved minimum UI surface.

<critical>
- ALWAYS READ `_techspec.md`, `web/CLAUDE.md`, and the relevant ADRs before implementation.
- REFERENCE the TechSpec sections `Web/Docs Impact`, `Public Interfaces / Types`, and `Agent Manageability Plan`.
- ACTIVATE `react`, `tailwindcss`, `tanstack-query-best-practices`, `app-renderer-systems`, and `vercel-react-best-practices` before editing web code.
- MINIMIZE CODE churn outside the listed route/system files; rely on regenerated types instead of handwritten contracts.
- TESTS REQUIRED: route loading, scope-aware list/detail/search behavior, mutation flows, error states, and empty states must ship here.
- NO WORKAROUNDS: do not hand-maintain API types that diverge from generated OpenAPI outputs.
</critical>

<requirements>
- MUST update the knowledge route and system modules to the final Memory v2 payloads and actions.
- MUST expose the approved minimum UI behavior only: truthful list/show/edit/delete/search/decision context surfaces, not speculative dashboards.
- MUST handle the expanded memory scope model and any new selection or filtering rules approved by the TechSpec.
- MUST keep route/data orchestration in hooks/adapters and UI components presentational.
- MUST update mocks, tests, and stories for the new memory contract.
</requirements>

## Subtasks
- [ ] 20.1 Update knowledge adapters, query options, and hooks to the regenerated Memory v2 contract.
- [ ] 20.2 Update knowledge list/detail UI for the new scope/tier and decision/recall semantics.
- [ ] 20.3 Refresh fixtures, MSW handlers, tests, and stories for the final memory payloads.
- [ ] 20.4 Confirm loading, empty, and failure states remain truthful to the daemon surface.

## Implementation Details

See TechSpec `Web/Docs Impact` and the current knowledge route/system implementation. This task should keep the web surface minimal and truthful rather than inventing unreleased memory dashboards or speculative operator controls.

### Relevant Files
- `web/src/routes/_app/knowledge.tsx` — main knowledge route shell.
- `web/src/hooks/routes/use-knowledge-page.ts` — route orchestration and state.
- `web/src/systems/knowledge/adapters/knowledge-api.ts` — API adapter layer over generated contract routes.
- `web/src/systems/knowledge/types.ts` — web-facing types that must align to generated payloads.
- `web/src/systems/knowledge/components/knowledge-list-panel.tsx` — list UI.
- `web/src/systems/knowledge/components/knowledge-detail-panel.tsx` — detail/edit/delete UI.

### Dependent Files
- `web/src/systems/knowledge/mocks/handlers.ts` — mock handlers must match the final contract.
- `web/src/routes/_app/-knowledge.test.tsx` — route-level test coverage.
- `web/src/routes/_app/stories/-knowledge.stories.tsx` — route/story fixtures for the updated surface.
- `.compozy/tasks/mem-v2/task_23.md` — docs task depends on the truthful UI behavior this task exposes.

### Related ADRs
- [ADR-002: Three Scopes with Agent Two-Tier](adrs/adr-002.md) — UI scope/tier semantics.
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — mutation/decision UI semantics.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — search/recall UI semantics.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none directly — checked surfaces are provider/host API/runtime hooks; this task only consumes the final public memory contract.
- Agent manageability: the web view must remain a truthful consumer of the same agent-manageable API/CLI surface, not a separate behavior.
- Config lifecycle: none directly — checked surfaces are memory settings/config keys; they appear in the settings page task instead.

### Web/Docs Impact

- `web/`: `web/src/routes/_app/knowledge.tsx`, `web/src/hooks/routes/use-knowledge-page.ts`, and `web/src/systems/knowledge/**` are expected to change here.
- `packages/site`: runtime memory docs and screenshots/examples may need updates later based on the final UI behavior.

## Deliverables

- Updated knowledge route, adapters, hooks, and components for Memory v2.
- Refreshed tests, mocks, and stories for the final memory contract.
- Truthful loading/empty/error states for the approved minimum UI surface.

## Tests

- Unit tests:
  - [ ] Adapters and hooks consume the regenerated memory contract correctly.
  - [ ] Scope/tier selection and search/list/detail behavior render the correct state.
  - [ ] Mutation flows surface deterministic success and error states from the daemon contract.
- Integration tests:
  - [ ] Route-level tests cover loading, empty, success, and failure states.
  - [ ] MSW fixtures and stories align with the final generated memory payloads.
- Test coverage target: workspace web coverage for all changed knowledge surfaces.
- All tests must pass.

## References

- `.resources/hermes/ui-tui/src/lib/memory.ts`
- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/claude-code/memdir/findRelevantMemories.ts`

## Success Criteria

- All tests passing.
- The knowledge page reflects the final Memory v2 contract and semantics without handwritten drift.
- The UI remains truthful, minimal, and aligned to the daemon surface.

