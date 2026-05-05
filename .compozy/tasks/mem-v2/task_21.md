---
status: pending
title: Web Memory Settings Surface
type: frontend
complexity: high
dependencies:
  - task_15
  - task_19
---

# Task 21: Web Memory Settings Surface

## Overview

Update the operator-facing memory settings page to reflect the full backend Memory v2 config model. This task owns the `/settings/memory` route and supporting systems, ensuring the web UI exposes only the real config, status, and action surfaces the runtime actually supports.

<critical>
- ALWAYS READ `_techspec.md`, `web/CLAUDE.md`, and `packages/site/CLAUDE.md` before implementation.
- REFERENCE the TechSpec sections `Config Lifecycle`, `Web/Docs Impact`, and `Agent Manageability Plan`.
- ACTIVATE `react`, `tailwindcss`, `tanstack-query-best-practices`, `app-renderer-systems`, and `vercel-react-best-practices` before editing web code.
- MINIMIZE CODE churn outside settings route/system files; generated/backend types win over guessed UI models.
- TESTS REQUIRED: load/edit/save/reset/restart flows, validation states, action buttons, and truthful status lines must ship here.
- NO WORKAROUNDS: do not render controls or metrics the backend settings surface does not actually support.
</critical>

<requirements>
- MUST update the memory settings route and system modules to the final backend memory settings payloads.
- MUST expose only the approved Memory v2 configuration, action, and status surfaces.
- MUST keep validation and draft state aligned to backend defaults and deterministic errors.
- MUST refresh stories, tests, and mocks for the new settings shape.
- MUST preserve truthful restart/action messaging and error handling.
</requirements>

## Subtasks
- [ ] 21.1 Update memory settings adapters, types, hooks, and route orchestration to the final backend payloads.
- [ ] 21.2 Update settings UI sections and controls for the approved Memory v2 config surface.
- [ ] 21.3 Refresh tests, mocks, and stories for load/save/reset/restart/action behavior.
- [ ] 21.4 Confirm no speculative provider/recall/dream controls appear beyond backend truth.

## Implementation Details

See TechSpec `Config Lifecycle`, `Web/Docs Impact`, and the existing settings memory route. This task should keep the settings page faithful to the daemon’s supported config and action surfaces, including restart behavior and error messaging.

### Relevant Files
- `web/src/routes/_app/settings/memory.tsx` — main memory settings route.
- `web/src/hooks/routes/use-settings-memory-page.ts` — route orchestration and draft management.
- `web/src/systems/settings/adapters/settings-api.ts` — backend settings API adapter.
- `web/src/systems/settings/types.ts` — web-facing settings types.
- `web/src/systems/settings/components/settings-section-card.tsx` — shared section UI used by memory settings.
- `web/src/routes/_app/settings/-memory.test.tsx` — route-level settings coverage.

### Dependent Files
- `web/src/systems/settings/mocks/fixtures.ts` — settings fixtures must match the final memory payload shape.
- `web/src/routes/_app/settings/stories/-memory.stories.tsx` — memory settings stories.
- `packages/site/content/runtime/core/configuration/config-toml.mdx` — docs task depends on the truthful settings surface.
- `.compozy/tasks/mem-v2/task_23.md` — docs task depends on the final UI and backend config truth.

### Related ADRs
- [ADR-008: MemoryProvider Extension ABC — Hermes 10-Hook Lifecycle](adrs/adr-008.md) — provider-related settings implications.
- [ADR-009: Write Controller — Hybrid Rule-First with LLM-as-Tiebreaker](adrs/adr-009.md) — controller settings implications.
- [ADR-011: Recall Pipeline — Deterministic-First with Optional Vector + LLM Ranker](adrs/adr-011.md) — recall/dream settings implications.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: none directly — this task consumes the backend settings/config contract rather than defining new extension surfaces.
- Agent manageability: the UI must remain a truthful consumer of the same backend settings/manageability surfaces agents can operate over CLI/HTTP/UDS.
- Config lifecycle: this task is the web projection of the backend config lifecycle already implemented in `task_13`.

### Web/Docs Impact

- `web/`: `web/src/routes/_app/settings/memory.tsx`, `web/src/hooks/routes/use-settings-memory-page.ts`, and `web/src/systems/settings/**` are expected to change here.
- `packages/site`: runtime configuration docs and examples may need updates later based on the final UI/state model.

## Deliverables

- Updated memory settings route, adapters, hooks, and components for the final Memory v2 config surface.
- Refreshed tests, mocks, and stories for the new settings payloads and actions.
- Truthful validation, action, and restart UI behavior.

## Tests

- Unit tests:
  - [ ] Settings adapters and hooks consume the final backend memory settings contract correctly.
  - [ ] Draft/reset/save/restart flows behave correctly for the approved memory settings fields.
  - [ ] Validation and error states reflect backend truth rather than guessed frontend rules.
- Integration tests:
  - [ ] Route-level tests cover loading, success, invalid input, save failure, and restart messaging.
  - [ ] Fixtures/stories remain aligned with the final generated/backend settings payloads.
- Test coverage target: web coverage for all changed settings memory surfaces.
- All tests must pass.

## References

- `.resources/hermes/website/docs/user-guide/features/memory.md`
- `.resources/hermes/website/docs/user-guide/features/memory-providers.md`
- `.resources/claude-code/memdir/memdir.ts`

## Success Criteria

- All tests passing.
- The memory settings page reflects the final backend config lifecycle truthfully.
- No unsupported or speculative memory controls are rendered in the UI.

