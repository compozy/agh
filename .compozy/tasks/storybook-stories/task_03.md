---
status: completed
title: Stories for web/ui overlays and navigation (13 components)
type: frontend
complexity: high
dependencies:
  - task_01
---

# Task 3: Stories for web/ui overlays and navigation (13 components)

## Overview
Author Storybook stories for the overlay and navigation families of the shadcn-layer in `web/src/components/ui`: dialog, sheet, popover, tooltip, dropdown-menu, accordion, collapsible, tabs, breadcrumb, scroll-area, sidebar, combobox, command. These components demand compound-composition renders and interactive triggers; they are the showcase for how interactive stories run under the web Storybook's decorators without requiring MSW.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `web/src/components/ui/stories/` and place one story file per component with the name `<component>.stories.tsx`.
- MUST title stories under `components/ui/<ComponentName>` and omit `tags: ["autodocs"]`.
- MUST use `render` functions for every compound component (`Dialog`, `Sheet`, `Popover`, `Tooltip`, `DropdownMenu`, `Accordion`, `Collapsible`, `Tabs`, `Breadcrumb`, `ScrollArea`, `Sidebar`, `Combobox`, `Command`) and include an `args: {}` property even when the render is static.
- MUST demonstrate open/closed interactions by default (uncontrolled) so reviewers can exercise overlays without code changes.
- MUST compose with `@agh/ui` primitives (Button, Card, Input, Label) rather than ad-hoc markup.
- MUST keep each component to 2–5 stories; at least include `Default` plus one state or variant story.
- MUST NOT introduce MSW handlers — overlays and navigation primitives are network-agnostic.
</requirements>

## Subtasks
- [x] 3.1 Author stories for overlays: dialog, sheet, popover, tooltip, dropdown-menu.
- [x] 3.2 Author stories for navigation: accordion, collapsible, tabs, breadcrumb, scroll-area, sidebar.
- [x] 3.3 Author stories for command primitives: combobox, command.
- [x] 3.4 Ensure every compound component exposes a triggerable interaction (button, trigger pill, etc.).
- [x] 3.5 Verify all stories render under the web Storybook's MSW + Query + Router decorators without warnings.

## Implementation Details
Follow the TechSpec's "Core Interfaces — Story module contract" and keep renders under 20 lines by hoisting sample data (lists, menu items) to module-level constants inside each story file. For `Sidebar`, reuse the existing `SidebarProvider` from `web/src/components/ui/sidebar.tsx` in the `render`. For `Command`/`Combobox`, include one interactive story with typed-through filtering to exercise keyboard navigation.

### Relevant Files
- `web/src/components/ui/{dialog,sheet,popover,tooltip,dropdown-menu,accordion,collapsible,tabs,breadcrumb,scroll-area,sidebar,combobox,command}.tsx` — story subjects.
- `web/.storybook/preview.ts` — provides the global providers.
- `web/src/components/design-system/stories/panel.stories.tsx` — reference decorator and copy style.
- `packages/ui/src/index.ts` — source of primitives used inside renders.

### Dependent Files
- `web/src/components/ui/stories/*.stories.tsx` — 13 new files.
- `task_11` — skill doc update references a compound-story example from this batch.

### Related ADRs
- [ADR-003: stories/ Subfolder Placement, Opt-in Autodocs](adrs/adr-003.md) — Defines the placement and autodocs-opt-in stance.

## Deliverables
- 13 new story files under `web/src/components/ui/stories/`.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for compound-component rendering **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Dialog `Default` story mounts with `Dialog.Trigger` and `Dialog.Content` without a prop warning.
  - [ ] Tabs `Default` story renders at least two `TabsTrigger` items with distinct `value`s.
  - [ ] Combobox interactive story filters its options when typing "al" (matches only items starting with "al").
  - [ ] Every meta in this batch has `title` matching `/^components\/ui\/[A-Z][A-Za-z-]+$/` and no `autodocs` tag.
- Integration tests:
  - [ ] `bun run --cwd web build-storybook` indexes exactly these 13 new modules in addition to existing stories.
  - [ ] Opening the Dialog story in Storybook's iframe toggles visibility via the default trigger.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- 13 story files added under `web/src/components/ui/stories/`.
- All stories pass a11y addon's critical checks.
- No console warnings about uncontrolled-to-controlled switching or missing ARIA in reviewed stories.
