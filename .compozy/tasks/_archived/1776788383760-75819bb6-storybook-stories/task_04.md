---
status: completed
title: Stories for web/ui forms and misc (14 components)
type: frontend
complexity: high
dependencies:
  - task_01
---

# Task 4: Stories for web/ui forms and misc (14 components)

## Overview
Author stories for the remaining web/ui composite primitives: field, input-group, button-group, select, native-select, textarea, toggle, toggle-group, switch, avatar, empty, item, direction, sonner. These power form pages and miscellaneous display affordances across every system, so their stories double as the design-system's form reference.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add one story file per component under `web/src/components/ui/stories/<component>.stories.tsx`.
- MUST title each meta `components/ui/<ComponentName>`; autodocs remains off.
- MUST pair form composites (`field`, `input-group`, `button-group`, `select`, `native-select`, `textarea`) with label, helper text, and error-state stories so error handling is visible without real validation.
- MUST demonstrate `Toggle`, `ToggleGroup`, and `Switch` in both single and multi-selection variants where supported.
- MUST use `Sonner` via the global provider (mount the `Toaster` in the render and trigger with a `Button` click) and do NOT mutate global state outside the iframe.
- MUST cover `Avatar` fallback and image states; `Empty` with an icon + title + action; `Item` inside a list container; `Direction` wrapping a paragraph in both LTR and RTL.
- MUST keep each component to 2–5 stories.
</requirements>

## Subtasks
- [x] 4.1 Author stories for form composites: field, input-group, button-group.
- [x] 4.2 Author stories for selection inputs: select, native-select, textarea.
- [x] 4.3 Author stories for toggles and switches: toggle, toggle-group, switch.
- [x] 4.4 Author stories for display affordances: avatar, empty, item.
- [x] 4.5 Author stories for `direction` (LTR/RTL) and `sonner` (toaster + trigger).
- [x] 4.6 Verify all stories run under the web Storybook and emit no runtime warnings.

## Implementation Details
Reference TechSpec's "Core Interfaces" section for the story module contract. Keep mock data minimal — two or three options per select, one avatar URL from an already-used source, one helper-text string per field story. The Sonner story must render `<Toaster />` locally in its `render` to avoid provider leakage between stories.

### Relevant Files
- `web/src/components/ui/{field,input-group,button-group,select,native-select,textarea,toggle,toggle-group,switch,avatar,empty,item,direction,sonner}.tsx` — story subjects.
- `web/.storybook/preview.ts` — theme + providers context.
- `packages/ui/src/index.ts` — composition source for Button, Label, etc.

### Dependent Files
- 14 new files under `web/src/components/ui/stories/`.
- `task_06`..`task_10` — system stories compose these primitives.

### Related ADRs
- [ADR-003: stories/ Subfolder Placement, Opt-in Autodocs](adrs/adr-003.md) — Placement + autodocs policy.

## Deliverables
- 14 new story files under `web/src/components/ui/stories/`.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for composite form and display primitives **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `field` error-state story renders a message associated with its input via `aria-describedby`.
  - [x] `select` story presents at least two options and defaults to the first.
  - [x] `toggle-group` multi-select story allows two items selected simultaneously.
  - [x] `sonner` story mounts a `Toaster` and triggers a toast with accessible role "status" on button click.
  - [x] `direction` story renders the text block with `dir="rtl"` in the RTL variant.
- Integration tests:
  - [x] `bun run --cwd web build-storybook` indexes 14 new modules with no indexing warnings.
  - [x] `avatar` fallback story displays the initials when the remote image 404s (use a broken URL + MSW unhandled-request bypass).
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- 14 story files present.
- All stories pass a11y critical checks.
- Toaster story does not leak a `<Toaster />` into other stories' iframes.
