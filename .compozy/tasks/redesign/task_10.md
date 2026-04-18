---
status: pending
title: Add ChatMessageBubble + ToolCallCard shells
type: frontend
complexity: medium
dependencies:
  - task_01
---

# Task 10: Add ChatMessageBubble + ToolCallCard shells

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Introduce two presentational shells: `ChatMessageBubble` (user/agent message container with role + timestamp slot) and `ToolCallCard` (bordered card for tool execution with icon + name + file path + status badge). Both are style-only — they do not know about session state, SSE, or live data; domain code composes them in task 20.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `packages/ui/src/components/chat-message-bubble.tsx` with props: `role` (`"user" | "agent" | "system" | "tool" | "diff"`), `meta?` (ReactNode for label + timestamp), `children` (body), `align?` (`"left" | "right"`, default derived from role).
- MUST create `packages/ui/src/components/tool-call-card.tsx` with props: `toolName` (string), `filePath?` (string), `status` (`"running" | "done" | "error"`), `children?` (optional body like code diff or output).
- MUST follow DESIGN.md §4 "Chat Components" — user right-aligned bubble (bg surface-elevated, radius 12px), agent left-aligned no bubble.
- MUST follow DESIGN.md §4 "Tool Call Card" — bg surface, border 1px divider, 8px radius, terminal `>_` icon, tool name + file path, status badge right-aligned.
- MUST export both from `packages/ui/src/index.ts`.
- MUST add stories with all five roles (user/agent/system/tool/diff) and all three statuses (running/done/error).
- MUST NOT import any `web/src/**` modules; pure presentational primitives.
</requirements>

## Subtasks

- [ ] 10.1 Implement `ChatMessageBubble` with role-based alignment + styling.
- [ ] 10.2 Implement `ToolCallCard` with status badge + file path composition.
- [ ] 10.3 Export both from `packages/ui/src/index.ts`.
- [ ] 10.4 Write stories covering all roles + all statuses.

## Implementation Details

DESIGN.md §4 "Chat Components" defines the shell layouts. The mock `docs/design/web-inspiration/src/pages-session.jsx` shows full composition (domain code consumes these primitives there, but the primitives themselves stay state-free).

### Relevant Files

- `packages/ui/src/components/chat-message-bubble.tsx` — new.
- `packages/ui/src/components/tool-call-card.tsx` — new.
- `packages/ui/src/index.ts` — add exports.
- `packages/ui/src/components/stories/` — destination for stories.
- `docs/design/web-inspiration/src/pages-session.jsx` — reference composition.
- DESIGN.md §4 — visual spec.

### Dependent Files

- Task 20 (session thread rewrite) consumes both primitives with real SSE data.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)

## Deliverables

- Two new primitives with stories.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests per role variant **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `ChatMessageBubble` with `role="user"` aligns right and renders the surface-elevated bubble.
  - [ ] `ChatMessageBubble` with `role="agent"` aligns left and renders without a bubble wrapper.
  - [ ] `ChatMessageBubble` with `role="system"` renders a full-width divider-style layout.
  - [ ] `meta` slot renders above the body for user role, beside the agent name for agent role.
  - [ ] `ToolCallCard` shows the terminal icon + tool name + file path.
  - [ ] `ToolCallCard` renders a DONE / RUNNING / ERROR badge based on `status` with the correct semantic tone.
  - [ ] `ToolCallCard` optional `children` slot renders below the header (used for diffs or output).
- Integration tests:
  - [ ] Storybook `play()` cycles through all five message roles and asserts alignment + styling.
  - [ ] Storybook `play()` cycles ToolCallCard `status` prop and asserts the badge tone matches.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Both primitives exported from `packages/ui/src/index.ts`.
- Stories render all five roles + all three statuses correctly.
- `make verify` passes.
