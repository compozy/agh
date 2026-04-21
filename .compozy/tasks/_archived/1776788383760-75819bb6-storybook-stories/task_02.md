---
status: completed
title: Stories for @agh/ui primitives (12 components)
type: frontend
complexity: medium
dependencies:
  - task_01
---

# Task 2: Stories for @agh/ui primitives (12 components)

## Overview
Author Storybook stories for every primitive exported by `@agh/ui`: Button, Badge, Card, Alert, Kbd, Separator, Input, Label, Progress, Skeleton, Spinner, Table. These are the design-system foundation and are the only stories in the rollout that receive `tags: ["autodocs"]` so that their prop documentation is automatically generated for reviewers.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create one story file per primitive under `packages/ui/src/components/stories/<name>.stories.tsx`.
- MUST import each component from the local source (`../button`, etc.) so typings bind to the authored surface; do not import via `@agh/ui`.
- MUST include exactly one `Default` story plus 1–4 targeted variants (size, variant, state). Never exceed 5 stories per component.
- MUST set `title: "ui/<ComponentName>"` on every meta and declare `tags: ["autodocs"]`.
- MUST use design tokens (`bg-background`, `text-foreground`, `border-border`, `--color-*` CSS variables) in every decorator and inline className — no hard-coded hex or tailwind slate/gray shades.
- MUST type meta explicitly as `Meta<typeof Component>` and every story as `StoryObj<typeof meta>` with an `args` property present (use `{}` when rendering manually).
- MUST NOT add MSW handlers, QueryClient wrappers, or router stubs — primitives are render-only.
</requirements>

## Subtasks
- [x] 2.1 Create `packages/ui/src/components/stories/` directory with one story file per primitive.
- [x] 2.2 Author `Default` stories for button, badge, card, alert, kbd, separator (display + structural).
- [x] 2.3 Author `Default` stories for input, label, progress, skeleton, spinner, table (form + feedback).
- [x] 2.4 Add 1–4 variants per component that cover every exported `variant`, `size`, or compound sub-component (e.g., `Table.Header`, `Alert.Description`).
- [x] 2.5 Run the packages/ui Storybook build and confirm every story renders under both light and dark themes.

## Implementation Details
Follow the "Core Interfaces — Story module contract" fragment of the TechSpec for the canonical template. Titles live under the `ui/` prefix. For compound components (Card, Alert, Table) demonstrate the full sub-component composition via `render`. Use `args` for leaf primitives (Button, Badge, Input). Keep every render under 20 lines.

### Relevant Files
- `packages/ui/src/components/*.tsx` — one story per file.
- `packages/ui/src/index.ts` — canonical export list for cross-reference.
- `packages/ui/.storybook/preview.ts` — decorator and theme context from task_01.
- `web/src/components/design-system/stories/panel.stories.tsx` — existing reference style.

### Dependent Files
- `packages/ui/.storybook/main.ts` — picks up `**/*.stories.@(ts|tsx)` automatically.
- `task_11` — skill doc update references these stories as the canonical autodocs example.

### Related ADRs
- [ADR-003: stories/ Subfolder Placement, Opt-in Autodocs](adrs/adr-003.md) — Defines both the folder convention and the autodocs policy applied here.

## Deliverables
- 12 new story files under `packages/ui/src/components/stories/`.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for primitive story rendering **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Every story meta has `title` matching `/^ui\/[A-Z][A-Za-z]+$/` and declares `tags: ["autodocs"]`.
  - [ ] Every exported story object has a defined `args` property (including `{}` when using `render`).
  - [ ] Button `Default` story renders with accessible name "Action" (or documented label) and `data-variant="default"`.
  - [ ] Alert `Default` story renders `Alert`, `AlertTitle`, and `AlertDescription` composition.
- Integration tests:
  - [ ] `bun run --cwd packages/ui build-storybook` indexes exactly 12 story modules and exits 0.
  - [ ] Dark-theme render of every story emits no Storybook runtime warnings.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- 12 primitive story files present, each with 2–5 stories.
- Storybook a11y addon reports no "critical" violations on `Default` stories.
- Tokens-only styling verified by `grep`-based review for `bg-white|text-black|#[0-9a-f]{3,6}`.
