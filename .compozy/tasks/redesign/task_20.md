---
status: pending
title: Rewrite Session domain message thread
type: frontend
complexity: high
dependencies:
  - task_10
  - task_13
  - task_14
---

# Task 20: Rewrite Session domain message thread

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite the scrollable message thread in `web/src/systems/session/components/**` on top of `@agh/ui` primitives. The thread renders five message roles (system, user, agent, tool, diff) sourced from the live SSE event stream; the redesign is purely visual — SSE wiring, the session store, the transcript assembler, and the virtualizer all stay untouched. This is the first visible surface of Phase 4 (ADR-004) and the consumer of the `ChatMessageBubble` + `ToolCallCard` shells introduced in task 10.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `chat-view.tsx`, `message-bubble.tsx`, `tool-call-card.tsx`, `tool-group-section.tsx`, `processing-indicator.tsx`, and `chat-header.tsx` against `@agh/ui` exports (`ChatMessageBubble`, `ToolCallCard`, `CodeBlock`, `StatusDot`, `MonoBadge`, `Button`, `ScrollArea`).
- MUST support all five message roles — `system`, `user`, `agent`, `tool`, `diff` — mapped onto `ChatMessageBubble`'s `role` prop; diffs render inside `CodeBlock` nested under a `ChatMessageBubble` with `role="diff"`.
- MUST preserve the existing SSE pipeline: `useSessionPage`, `useChatViewContent`, `useChatViewRows`, `session-store.ts`, `transcript` hooks, and the TanStack Virtual virtualizer stay as-is; only JSX + styling changes.
- MUST render the agent-label row with an 8px `StatusDot` (tone `success` for live, `warning` for degraded, `danger` for errored, `neutral` for idle) + mono agent-name glyph per DESIGN.md §4 "Agent Message".
- MUST preserve all existing behaviors: auto-scroll to bottom, "scroll to bottom" affordance, streaming "..." indicator, copy-message button, thinking block, tool group collapsing, empty state.
- MUST NOT import from `@/components/ui/**` or `@/components/design-system/**` (both folders are gone after task 08).
- MUST NOT alter component public props where existing Vitest tests assert them — updated rendering must pass existing role/timestamp/streaming assertions unchanged wherever possible; where props change, update the tests in the same PR (TechSpec "Testing Approach").
- SHOULD delete `copy-button.tsx` if `@agh/ui` ships an equivalent `CopyButton`; otherwise keep it and migrate its styling to tokens.
</requirements>

## Subtasks

- [ ] 20.1 Audit `web/src/systems/session/components/` and map each file to the `@agh/ui` primitives it will consume.
- [ ] 20.2 Rewrite `message-bubble.tsx` as a thin composition over `ChatMessageBubble`, preserving role-aware meta + copy button + thinking block.
- [ ] 20.3 Rewrite `tool-call-card.tsx` + `tool-group-section.tsx` + each `tool-renderers/*-content.tsx` on top of `@agh/ui` `ToolCallCard` and `CodeBlock` for diffs.
- [ ] 20.4 Rewrite `chat-view.tsx` (virtualized list) and `chat-header.tsx` against new primitives; keep virtualizer wiring intact.
- [ ] 20.5 Update or add Storybook stories covering: empty thread, single user+agent turn, tool call (running/done/error), diff, long mixed thread, streaming in-flight.
- [ ] 20.6 Run `make web-lint`, `make web-typecheck`, `make web-test`, and dev-mode smoke against a live session.

## Implementation Details

See TechSpec "Impact Analysis" — `web/src/systems/session/**` is a Phase 4 visual rewrite. DESIGN.md §4 "Chat Components" is the spec for bubble + tool-call-card + input styling. The redesign mock `docs/design/web-inspiration/` (see the extracted `design-system/` reference and the prior `pages-session.jsx` sample) shows the full SessionPage composition. The consumer contract for `@agh/ui` shells lives in `task_10.md`.

### Relevant Files

- `web/src/systems/session/components/chat-view.tsx` — virtualized thread container.
- `web/src/systems/session/components/message-bubble.tsx` — per-message renderer (user + agent + system).
- `web/src/systems/session/components/tool-call-card.tsx` — tool execution card.
- `web/src/systems/session/components/tool-group-section.tsx` — tool-call grouping.
- `web/src/systems/session/components/processing-indicator.tsx` — streaming loader row.
- `web/src/systems/session/components/chat-header.tsx` — session header (status + resume/stop).
- `web/src/systems/session/components/tool-renderers/*.tsx` — per-tool content renderers (bash, edit, read, search, write, generic) consumed inside the new `ToolCallCard` body.

### Dependent Files

- `web/src/systems/session/hooks/use-chat-view-content.ts`, `use-chat-view-rows.ts`, `use-session-chat.ts`, `use-session-transcript.ts` — unchanged, consumed by the rewritten view.
- `web/src/systems/session/stores/session-store.ts` — unchanged.
- `web/src/routes/_app/session.$id.tsx` — composes the rewritten thread alongside the composer (task 21) and inspector (task 22).
- `web/src/systems/session/components/stories/*` — refreshed to consume new primitives.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md) — Primitive source of truth.
- [ADR-002: Greenfield migration](adrs/adr-002.md) — No shims; old JSX leaves with the PR.
- [ADR-003: Adopt motion](adrs/adr-003.md) — Streaming/processing indicator may use `motion` only for unmount; otherwise CSS.
- [ADR-004: Phased rollout](adrs/adr-004.md) — Phase 4 Session domain.
- [ADR-005: Playwright visual snapshots](adrs/adr-005.md) — Snapshots for every story variant + the live route.

## Deliverables

- Rewritten message-thread components composed from `@agh/ui` primitives.
- Refreshed Storybook stories covering all five roles + tool statuses + streaming + empty states.
- Playwright visual snapshot baselines per story variant and for the `/session/$id` route with a deterministic MSW fixture.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Integration tests driving the virtualized thread with a simulated SSE fixture **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `MessageBubble` with `role="user"` renders a right-aligned `ChatMessageBubble` whose meta slot shows `YOU` + formatted timestamp.
  - [ ] `MessageBubble` with `role="agent"` and `agentName="claude-code"` renders a left-aligned `ChatMessageBubble` with an 8px success `StatusDot` and mono agent name.
  - [ ] `MessageBubble` with `role="system"` renders a full-width divider-style row without a bubble wrapper.
  - [ ] `MessageBubble` with `role="diff"` nests a `CodeBlock` whose language prop matches `message.diff.language` and whose body contains the diff text.
  - [ ] `ToolCallCard` with `status="running"` renders the accent-tinted status badge plus a spinner; `status="done"` renders the success tint; `status="error"` renders the danger tint and danger-toned card border.
  - [ ] `ChatView` with `messages=[]` and `isStreaming=false` renders the empty-state with the `MessageSquare` 48px tertiary icon.
  - [ ] `ChatView` auto-scrolls to bottom when a new row arrives and the user has not scrolled up (existing hook contract preserved).
- Integration tests:
  - [ ] Feeding a fixture SSE stream (`user → agent → tool running → tool done → agent → diff`) renders rows in order and applies the correct primitive variants per role.
  - [ ] Storybook interaction: clicking the copy button on a user bubble copies the message text and shows the checkmark swap for ~1.5s.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing with `-race`-equivalent parallel subtests in Vitest.
- Test coverage >=80% for `web/src/systems/session/components/**`.
- `make verify` and `make web-lint` + `make web-typecheck` pass with zero warnings.
- No remaining imports from `@/components/ui/**` or `@/components/design-system/**` inside `web/src/systems/session/components/**`.
- Playwright baselines committed for every story variant and for the `/session/$id` route.
- Dev-mode session renders the full five-role thread against a real daemon without regressions in auto-scroll, copy, or streaming behaviors.
