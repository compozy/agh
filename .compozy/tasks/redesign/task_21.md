---
status: pending
title: Rewrite Session domain composer
type: frontend
complexity: medium
dependencies:
  - task_20
---

# Task 21: Rewrite Session domain composer

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite the session composer that sits at the bottom of the Session page — the auto-growing textarea, inline attach/skills/channels pills, and the circular send button. The rewrite composes `@agh/ui` primitives (`Textarea`, `Pills`, `Button`, `Popover`, `Combobox`) and preserves all send logic, keyboard shortcuts, disabled state handling, and draft-persistence wiring from the existing session store. Visual spec comes from DESIGN.md §4 "Chat Input" and the redesign mock's composer.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `web/src/systems/session/components/message-composer.tsx` on top of `@agh/ui` `Textarea`, `Pills`, `Button`, `Popover` (for the attach menu), and `Combobox` (for skill + channel pickers).
- MUST match DESIGN.md §4 "Chat Input": 12px radius, `--color-surface` fill, 1px divider border, `--color-accent` focus border, 36px circular accent send button with `SendHorizontal` icon.
- MUST preserve the existing send-logic contract: `Enter` sends, `Shift+Enter` inserts a newline, whitespace-only input is ignored, the send button + keyboard shortcut are disabled when `disabled` is true, the textarea auto-grows to a 200px cap.
- MUST read and write draft text through the session store (or dedicated draft store when task 20 introduces one) so that unsent text survives route navigations.
- MUST render the three inline controls as `Pills`: an attach affordance opening a `Popover`, a skill picker opening a `Combobox` over installed skills, and a channel picker opening a `Combobox` over the session's bound channels.
- MUST NOT import from `@/components/ui/**` or `@/components/design-system/**`.
- MUST NOT regress existing message-composer unit tests; where props change, update the tests in the same PR.
- SHOULD gracefully hide skill/channel pills when no skills/channels are available in the current workspace.
</requirements>

## Subtasks

- [ ] 21.1 Inventory the current `message-composer.tsx` behaviors (send, keyboard, disabled, auto-grow) and the store reads it needs to become draft-aware.
- [ ] 21.2 Rebuild the composer shell using `@agh/ui` primitives with the new visual spec.
- [ ] 21.3 Wire the attach `Popover`, skill `Combobox`, and channel `Combobox`; surface the selected values back to `onSend`'s payload shape.
- [ ] 21.4 Wire draft persistence through the session store so drafts survive route changes.
- [ ] 21.5 Write or update Storybook stories: empty, typing, disabled, with attach open, with skill picker open, with channel picker open.
- [ ] 21.6 Run `make web-lint`, `make web-typecheck`, `make web-test`, and smoke the live route.

## Implementation Details

See TechSpec "Impact Analysis" — Phase 4 Session domain. DESIGN.md §4 "Chat Input" is the visual spec for the container, focus border, and send button. The `Pills` + `Combobox` + `Popover` primitives ship from `@agh/ui` (tasks 02/03/06). Draft persistence follows the Zustand patterns documented in `web/CLAUDE.md`.

### Relevant Files

- `web/src/systems/session/components/message-composer.tsx` — rewrite target.
- `web/src/systems/session/stores/session-store.ts` — draft storage.
- `web/src/systems/session/hooks/use-session-chat.ts` — send entrypoint consumed by the composer.

### Dependent Files

- `web/src/routes/_app/session.$id.tsx` — composes the composer next to the thread (task 20).
- `web/src/systems/session/components/stories/*` — composer stories refresh against new primitives.
- `web/src/systems/skills/**` and `web/src/systems/network/**` — public barrels consumed to source skill + channel options via their query hooks.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout — Phase 4 Session](adrs/adr-004.md)
- [ADR-005: Playwright visual snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `message-composer.tsx` composed from `@agh/ui` primitives.
- Draft persistence wired through the session store.
- Refreshed Storybook stories covering empty / typing / disabled / each open picker.
- Playwright visual snapshot baselines for each story variant.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for keyboard send + picker flow **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Typing text and pressing `Enter` calls `onSend` once with the trimmed text, and clears the textarea.
  - [ ] Pressing `Shift+Enter` inserts a newline and does NOT call `onSend`.
  - [ ] Submitting whitespace-only text does NOT call `onSend`.
  - [ ] When `disabled` is true, clicking the send button and pressing `Enter` both no-op and the send button renders with `opacity-50 cursor-not-allowed`.
  - [ ] The textarea auto-grows up to 200px and stops growing past the cap.
  - [ ] Selecting a skill through the skill `Combobox` attaches `{ skillId }` to the next `onSend` payload.
  - [ ] Typed draft text persists after unmount/remount via the session store read.
- Integration tests:
  - [ ] Storybook interaction opens the attach `Popover`, picks a file, closes the popover, and asserts the attach pill shows the file name.
  - [ ] Storybook interaction focuses the textarea, asserts the container border switches to `--color-accent`, blurs, asserts it returns to divider color.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80% for `message-composer.tsx`.
- `make verify` and `make web-lint` + `make web-typecheck` pass with zero warnings.
- No imports from `@/components/ui/**` or `@/components/design-system/**` inside the composer.
- Playwright baselines committed for every story variant.
- Draft text survives navigating away from and back to the session route in dev mode.
