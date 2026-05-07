---
status: completed
title: "Web Model Catalog Experience"
type: frontend
complexity: high
dependencies:
  - task_01
  - task_06
  - task_07
---

# Task 9: Web Model Catalog Experience

## Overview
This task updates the web app to consume the model catalog instead of static provider `supported_models`. It also makes active sessions prefer ACP `configOptions` while preserving manual model entry and clear stale/error states.

<critical>
- ALWAYS READ `_techspec.md` and every ADR before starting
- REFERENCE TECHSPEC for implementation details - do not duplicate here
- FOCUS ON "WHAT" - describe what needs to be accomplished, not how
- MINIMIZE CODE - show code only to illustrate current structure or problem areas
- TESTS REQUIRED - every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add/update a web model catalog system with adapters, query keys/options, hooks, and types using generated contracts.
- MUST update the new session dialog to load catalog models for the selected provider.
- MUST keep manual model entry available even when catalog rows are empty or stale.
- MUST show loading, empty, stale, source error, and refresh states without blocking session creation.
- MUST render `availability_state` distinctly so stale availability cannot look the same as live availability.
- MUST update Settings > Providers to edit the new nested `models` block and display discovery status/refresh.
- MUST consume `SessionConfigOptionPayload` from Task 06 and prefer active ACP `configOptions` over catalog metadata after session creation.
- MUST remove web usage of `default_model`, `supported_models`, and `supports_reasoning_effort` old fields.
</requirements>

## Subtasks
- [x] 9.1 Add model catalog web adapter/query/hook structure following existing systems patterns.
- [x] 9.2 Update new-session provider/model/reasoning view-model logic to use catalog rows and active ACP config options.
- [x] 9.3 Update session create dialog components for catalog rows, manual entry, stale/error/refresh states.
- [x] 9.4 Update Settings > Providers view-model and form for nested model config and source status.
- [x] 9.5 Update fixtures, mocks, component tests, hook tests, and high-risk E2E fixtures.

## Implementation Details
Follow `_techspec.md` sections `Web`, `Web/Docs Impact`, and `Safety Invariants`. Activate `react`, `tailwindcss`, `vercel-react-best-practices`, `tanstack-query-best-practices`, `app-renderer-systems`, `zod`, `vitest`, and `testing-anti-patterns`.

### Relevant Files
- `web/CLAUDE.md` - web architecture, systems, and test rules.
- `web/src/systems/session/hooks/use-session-create-dialog.ts` - currently builds options from `supported_models`.
- `web/src/systems/session/components/session-create-dialog.tsx` - session create UI.
- `web/src/routes/_app/settings/providers.tsx` - provider settings editor old controls.
- `web/src/hooks/routes/use-settings-providers-page.ts` - settings provider view model old fields.
- `web/src/systems/settings/components/provider-card.tsx` - provider summary old fields.
- `web/src/systems/settings/mocks/fixtures.ts` - old provider settings fixture data.
- `web/src/generated/agh-openapi.d.ts` - generated types from Task 01 hard cut and Task 10 final route/extension generation.

### Dependent Files
- `web/src/systems/session/components/__tests__/session-create-dialog.test.tsx` - component tests.
- `web/src/systems/session/hooks/__tests__/use-session-create-dialog.test.tsx` - hook behavior tests.
- `web/src/routes/_app/settings/__tests__/-providers.test.tsx` - settings route tests.
- `web/src/hooks/routes/__tests__/use-settings-providers-page.test.tsx` - settings hook tests.
- `web/e2e/__tests__/session-provider-override.spec.ts` - session provider/model override E2E fixture.
- `web/e2e/fixtures/runtime-seed.ts` - old default model fixture shape.

### Related ADRs
- [ADR-001: Daemon-Owned Provider Model Catalog](adrs/adr-001-daemon-owned-provider-model-catalog.md) - web consumes daemon catalog, not config hints.
- [ADR-002: Provider Model Config Hard Cut](adrs/adr-002-provider-model-config-hard-cut.md) - old web provider fields must be removed.

### Web/Docs Impact
- `web/`: affects session system, settings route/hooks/components, generated types, mocks, and E2E fixtures listed above.
- `packages/site`: no direct MDX edit here; Task 10 documents the user-facing provider settings/model catalog behavior.

### Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: displays extension source status through generic catalog source refs; no extension authoring UI is required.
- Agent manageability: UI must not be the only path; CLI/HTTP/UDS remain authoritative from Task 07.
- Config lifecycle: UI edits nested `providers.<id>.models` config and removes old field controls.

## Deliverables
- Web model catalog data system and hooks.
- Updated new-session dialog and provider settings UI.
- Removed web old-field assumptions.
- Component/hook tests and E2E fixture updates with 80%+ relevant coverage **(REQUIRED)**.

## Tests
- Unit tests:
  - [ ] selected provider loads catalog models and dedupes/sorts rows deterministically.
  - [ ] manual model entry remains available when catalog is empty.
  - [ ] stale rows render stale status and do not block session creation.
  - [ ] `available_stale`, `unavailable_stale`, `available_live`, `unavailable_live`, and `unknown` states render distinctly.
  - [ ] source error renders without hiding manual entry.
  - [ ] refresh action invalidates catalog queries.
  - [ ] active ACP model/reasoning `SessionConfigOptionPayload` entries override catalog assumptions after session creation.
  - [ ] provider settings form edits nested model default and curated model metadata.
- Integration tests:
  - [ ] settings provider save sends new nested models payload and no old flat fields.
  - [ ] session create submits selected catalog model and reasoning only when allowed by active/session state.
  - [ ] high-risk E2E fixture covers selecting provider, catalog model, manual model fallback, and refresh status.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make bun-typecheck`, `make bun-test`, and `make web-build` pass for web changes.
- Web no longer reads `supported_models` as the pre-session model source.
