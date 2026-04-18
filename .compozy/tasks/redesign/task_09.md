---
status: pending
title: Add CodeBlock primitive
type: frontend
complexity: low
dependencies:
  - task_01
---

# Task 09: Add CodeBlock primitive

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Introduce the `CodeBlock` primitive in `@agh/ui` per DESIGN.md §4: canvas-deep (`#0E0E0F`) container with JetBrains Mono, accent-colored `$ ` prompt, copy-to-clipboard button that swaps to a checkmark for 1.5s on success, and an optional language eyebrow.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `packages/ui/src/components/code-block.tsx` with props: `code` (string), `language?` (string), `showPrompt?` (boolean, default true), `copyable?` (boolean, default true).
- MUST render with `bg: --color-canvas-deep`, `radius: --radius-diagram (12px)`, JetBrains Mono 14px line-height 1.6.
- MUST color the `$ ` prompt in `--color-accent` when `showPrompt` is true.
- MUST render a top-right ghost copy button that copies `code` to clipboard and swaps to a check icon for 1.5s on success.
- MUST render an optional language eyebrow top-left (JetBrains Mono 11px uppercase tracking 0.06em, tertiary text).
- MUST export from `packages/ui/src/index.ts`.
- MUST add a story with variants: shell command (with prompt), multiline code (no prompt), language label, disabled copy.
</requirements>

## Subtasks

- [ ] 9.1 Implement `CodeBlock` with prompt coloring + copy button state.
- [ ] 9.2 Export from `packages/ui/src/index.ts`.
- [ ] 9.3 Write story with the four required variants.

## Implementation Details

DESIGN.md §4 "Code Block" specifies the exact container, font, prompt, and copy button behavior. Reuse `@agh/ui` `Button` (ghost variant) for the copy trigger; use Lucide `Copy` and `Check` icons.

### Relevant Files

- `packages/ui/src/components/code-block.tsx` — new.
- `packages/ui/src/index.ts` — add export.
- `packages/ui/src/components/stories/code-block.stories.tsx` — new.
- DESIGN.md §4 — visual spec.

### Dependent Files

- Future domain tasks (tasks detail view, knowledge detail, session tool calls) consume `CodeBlock`.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)

## Deliverables

- `CodeBlock` primitive with stories.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction test for copy behavior **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Renders the provided `code` inside a `<pre>` / `<code>` element with JetBrains Mono.
  - [ ] When `showPrompt=true`, the `$ ` prefix is wrapped in an element with `--color-accent` foreground.
  - [ ] Language eyebrow renders only when `language` is provided.
  - [ ] Copy button is absent when `copyable=false`.
  - [ ] Clicking copy calls `navigator.clipboard.writeText` with `code`.
  - [ ] After successful copy, the button shows a check icon for 1.5s, then reverts.
- Integration tests:
  - [ ] Storybook `play()` clicks the copy button, stubs clipboard, asserts check icon appears and reverts.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `CodeBlock` exported from `packages/ui/src/index.ts`.
- Story renders all four variants correctly in Storybook.
- `make verify` passes.
