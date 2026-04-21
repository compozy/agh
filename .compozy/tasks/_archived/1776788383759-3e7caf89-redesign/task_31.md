---
status: completed
title: Rewrite Settings pages batch 1 (general, memory, skills, providers, automation)
type: frontend
complexity: high
dependencies:
  - task_30
---

# Task 31: Rewrite Settings pages batch 1 (general, memory, skills, providers, automation)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite the five Phase 6 sub-route pages `/settings/general`, `/settings/memory`, `/settings/skills`, `/settings/providers`, and `/settings/automation` on top of the Settings shell from task 30 and `@agh/ui` primitives. Each page keeps its current query hooks, mutations, and validation intact — only the visual layer (sections, fields, tables, pill filters, dialogs, empty states) is replaced.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `web/src/routes/_app/settings/general.tsx`, `memory.tsx`, `skills.tsx`, `providers.tsx`, and `automation.tsx` using `@agh/ui` `Section`, `Field`, `Input`, `Switch`, `Pills`, `Combobox`, `NativeSelect`, `Table`, `Button`, `Dialog`, `Empty`, and the `SettingsPageShell` / `SettingsSaveBar` / `SettingsRestartBanner` from task 30.
- MUST preserve every existing hook call (`useSettingsGeneralPage`, `useSettingsMemoryPage`, `useSettingsSkillsPage`, `useSettingsProvidersPage`, `useSettingsAutomationPage`) unchanged — this task is presentational only.
- MUST preserve every mutation payload shape; PATCH requests to `/api/settings` keep their current JSON body so backend tests stay valid.
- MUST preserve validation, dirty detection, Save / Discard wiring, and restart-banner triggers across all five pages.
- MUST NOT introduce any imports from `@/components/ui/*` or `@/components/design-system/*` (those folders are removed in Phase 2); `PillButton` call sites migrate to `@agh/ui` `Pills`.
- MUST render every form field through `@agh/ui` `Field` with label + description + error slots, and use `Switch` for booleans, `Combobox` for searchable selects, `NativeSelect` for short enums, `Pills` for segmented choices.
- MUST render list / table content through `@agh/ui` `Table` with mono meta columns where appropriate (provider IDs, skill IDs) per DESIGN.md §4.
- MUST render empty states through `@agh/ui` `Empty` (48px Lucide icon + title + description) for `skills` and `providers` when the list is empty.
- MUST produce Playwright snapshot baselines per sub-route in at least the idle and dirty states.
- SHOULD keep file sizes manageable by extracting per-page section components inside the same route file when a page exceeds ~300 lines.
</requirements>

## Subtasks

> Subtasks below describe sub-routes, not individual files — each sub-route owns its route file plus any in-file section subcomponents. No new component files land in `web/src/systems/settings/components/`.

- [x] 31.1 Rewrite `/settings/general` — app / session timeout / permission-mode sections with `Section` + `Field` + `Switch` + `Pills`; Save-bar + Restart-banner wired through `useSettingsGeneralPage`.
- [x] 31.2 Rewrite `/settings/memory` — scope toggles, retention fields, dream-trigger thresholds, and the stats grid using `Section` + `Field` + `SettingsStatGrid`; confirm large-number formatting.
- [x] 31.3 Rewrite `/settings/skills` — installed skills `Table` (name, id, version, source, enabled `Switch`), filter `Pills`, `Empty` state when list is empty, optional `Dialog` for skill detail.
- [x] 31.4 Rewrite `/settings/providers` — provider `Table` (name, kind, base URL, API key status), add-provider `Dialog` with `Field` + `Input` + `Combobox` for kind, delete confirmation via `Dialog`.
- [x] 31.5 Rewrite `/settings/automation` — automation rules `Table` with enabled `Switch`, trigger-kind `Pills`, create / edit `Dialog`, and restart-banner wiring for daemon-sensitive toggles.
- [x] 31.6 Update the matching `*.stories.tsx` files under `web/src/routes/_app/settings/stories/` to consume the new components and exercise idle + dirty + loading + empty states.
- [x] 31.7 Regenerate Playwright snapshot baselines per sub-route (idle + dirty minimum; add empty for `skills` / `providers`).

## Implementation Details

See TechSpec Impact Analysis row `web/src/routes/_app/settings/**` and ADR-004 Phase 6. The Settings shell primitives from task 30 are the top-level composition; each route file adds only sections + fields + tables on top. No new files land in `web/src/systems/settings/components/` — shared primitives stay in task 30's set. DESIGN.md §4 defines the Field / Table / Switch / Pills visual rules; the mock `docs/design/web-inspiration/src/pages-session.jsx` `SettingsPage` "general" section is the visual reference for sectioning and rhythm.

### Relevant Files

- `web/src/routes/_app/settings/general.tsx` — rewrite target.
- `web/src/routes/_app/settings/memory.tsx` — rewrite target.
- `web/src/routes/_app/settings/skills.tsx` — rewrite target.
- `web/src/routes/_app/settings/providers.tsx` — rewrite target.
- `web/src/routes/_app/settings/automation.tsx` — rewrite target.
- `web/src/routes/_app/settings/stories/-general.stories.tsx`, `-memory.stories.tsx`, `-skills.stories.tsx`, `-providers.stories.tsx`, `-automation.stories.tsx` — update.
- `web/src/hooks/routes/use-settings-*.ts` — consumed unchanged.

### Dependent Files

- `web/src/systems/settings/components/**` — consumed (task 30 output).
- `web/src/integrations/tanstack-query/**` — MSW handlers untouched; mutation contracts assert the PATCH payloads remain identical.
- `web/e2e/settings-*.spec.ts` — snapshot baselines regenerate here.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md) — primitive imports for Field / Switch / Pills / Combobox / Table / Dialog / Empty.
- [ADR-002: Greenfield migration — delete without backwards-compat](adrs/adr-002.md) — delete legacy `PillButton` call sites.
- [ADR-004: Phased rollout](adrs/adr-004.md) — Phase 6 batch 1.
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md) — per-route snapshot baselines.

## Deliverables

- Five rewritten route files consuming only `@agh/ui` primitives and the task-30 Settings shell.
- Updated `.stories.tsx` files with idle / dirty / loading / empty variants exercised for each page.
- Playwright snapshot baselines for `/settings/general`, `/settings/memory`, `/settings/skills`, `/settings/providers`, `/settings/automation` in idle + dirty states, plus empty states for `skills` and `providers` **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Integration tests asserting mutation payload contracts per sub-route **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `/settings/general` — toggling the `auto-resume` `Switch` marks the draft dirty and enables the Save button.
  - [ ] `/settings/general` — clicking Save with `permission-mode="approve-reads"` calls PATCH `/api/settings` with `{ general: { permissionMode: "approve-reads" } }` and shows "Saving…" until the mutation settles.
  - [ ] `/settings/general` — entering a session-timeout `Input` of `"45m"` parses to 2700 seconds and submits `{ general: { sessionTimeoutSeconds: 2700 } }`.
  - [ ] `/settings/memory` — toggling the `dream-enabled` `Switch` off disables the dream-threshold `Field`s and submits `{ memory: { dreamEnabled: false } }`.
  - [ ] `/settings/skills` — toggling a row's enabled `Switch` calls PATCH `/api/settings/skills/<id>` with `{ enabled: false }` and invalidates the skills query.
  - [ ] `/settings/skills` — empty skills list renders `@agh/ui` `Empty` with the "No skills installed" title.
  - [ ] `/settings/providers` — opening the "Add provider" `Dialog`, filling `name="OpenAI"` + `kind="openai"`, and submitting calls POST `/api/settings/providers` with that payload.
  - [ ] `/settings/providers` — delete confirmation `Dialog` calls DELETE `/api/settings/providers/<id>` only after the confirm button is clicked; Cancel closes the dialog without a request.
  - [ ] `/settings/automation` — toggling an automation rule `Switch` calls PATCH `/api/settings/automation/<id>` with `{ enabled: true }` and surfaces the restart-banner when the hook marks restart required.
  - [ ] Each of the five pages renders the shared `SettingsPageShell` with `slug="general"`, `"memory"`, `"skills"`, `"providers"`, `"automation"` respectively so existing `data-testid` selectors match.
- Integration tests:
  - [ ] Storybook `play()` on each sub-route story submits a valid change and asserts the MSW-intercepted PATCH payload matches the frozen contract.
  - [ ] Playwright snapshot baselines for the five sub-routes in idle + dirty states match within 0.1% threshold; `/settings/skills` and `/settings/providers` additionally match the empty-state baseline.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing and `make verify` green.
- Test coverage >=80% across the rewritten routes.
- `grep -r "from \"@/components/ui/" web/src/routes/_app/settings` returns zero hits.
- `grep -r "from \"@/components/design-system" web/src/routes/_app/settings` returns zero hits.
- Mutation PATCH / POST / DELETE payloads verified byte-equivalent to the pre-rewrite contract via MSW fixtures.
- Playwright baselines committed for every listed state per sub-route.
