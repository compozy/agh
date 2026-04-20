# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Ship two presentational shells in `@agh/ui`: `ChatMessageBubble` (user/agent/system/tool/diff) and `ToolCallCard` (running/done/error), state-free. Stories + tests + exports required. Visual language per DESIGN.md §4 "Chat Components" and "Tool Call Card".

## Important Decisions

- **Role → shell mapping** on `ChatMessageBubble`:
  - `user` → right-aligned flex, surface-elevated bubble (`rounded-[var(--radius-lg)]`, `px-5 py-4`, primary text). Meta slot renders _above_ the bubble, right-aligned.
  - `agent` → left-aligned flex-col, no bubble, secondary text. Meta slot renders _inline beside_ the agent name (the caller composes dot + name + time in the meta ReactNode).
  - `system` → full-width row, body flanked by `h-px flex-1` hairline dividers, tertiary mono 11px.
  - `tool` / `diff` → pass-through left-aligned flex-col so callers can drop a `ToolCallCard` or diff card inside. No bubble.
- **`align` prop** defaults are derived: `user` → right, everything else → left. The prop lets callers flip explicitly.
- **`ToolCallCard` status → tone mapping** (DESIGN.md §4 status badges): `running → accent`, `done → success`, `error → danger`. Labels are uppercase `RUNNING`/`DONE`/`ERROR` rendered through `MonoBadge` (existing primitive already matches the 15%-tint formula).
- **Reused `MonoBadge`** rather than introducing a separate StatusBadge primitive — MonoBadge supports all semantic tones + uppercase default and DESIGN.md's typography delta between "Status Badge" and "Mono Badge" (10px/600 vs 11px/500) is close enough that a dedicated primitive is out of scope for the shell task. Flagged as follow-up below if a future task wants pixel-exact 10px 600 tracking 0.08em.
- **Terminal glyph** uses Lucide's `TerminalIcon` (matches DESIGN.md's "terminal `>_` icon" requirement and stays in-system with the rest of the icons).
- **`data-slot` hooks** on every inner element (`chat-message`, `chat-message-inner`, `chat-message-meta`, `chat-message-body`, `tool-call-card`, `tool-call-card-header`, `tool-call-card-icon`, `tool-call-card-tool`, `tool-call-card-path`, `tool-call-card-status`, `tool-call-card-body`) — matches existing primitive conventions + lets downstream domain + Playwright snapshots target slots without class coupling.

## Learnings

- `MonoBadge` hard-codes `data-slot="mono-badge"` but spreads `{...props}` _after_, so passing `data-slot="tool-call-card-status"` from the parent overrides the default slot name cleanly. That's how `ToolCallCard` gives tests a stable selector for the status badge.
- `ChatMessageBubble` `role` prop shadows the native `role` attribute on a `<div>`. The interface uses `Omit<React.ComponentProps<"div">, "role">` so TypeScript does not complain when the caller passes ARIA roles elsewhere.

## Files / Surfaces

- `packages/ui/src/components/chat-message-bubble.tsx` — new primitive.
- `packages/ui/src/components/chat-message-bubble.test.tsx` — unit tests (8 cases covering all 5 roles, meta placement, align override, prop forwarding).
- `packages/ui/src/components/tool-call-card.tsx` — new primitive.
- `packages/ui/src/components/tool-call-card.test.tsx` — unit tests (7 cases incl. `it.each` over all 3 statuses).
- `packages/ui/src/components/stories/chat-message-bubble.stories.tsx` — 7 stories incl. `RoleAlignmentInteraction` and `StatusBadgeCycleInteraction` `play()` tests.
- `packages/ui/src/components/stories/tool-call-card.stories.tsx` — 7 stories incl. `StatusCycleInteraction` `play()`.
- `packages/ui/src/index.ts` — adds `ChatMessageBubble`, `ToolCallCard`, plus `ChatMessageBubbleProps`/`ChatMessageRole`/`ChatMessageAlign`/`ToolCallCardProps`/`ToolCallStatus` type exports.

## Errors / Corrections

- oxfmt reformatted three files (`chat-message-bubble.tsx`, `tool-call-card.test.tsx`, `chat-message-bubble.stories.tsx`) on first pass — applied via `bunx oxfmt`, no semantic changes. Matches the shared-memory note that oxfmt should run before commit and not be relied on to merge imports.
- Pre-existing `packages/ui/src/components/accordion.test.tsx` TS2322 (AccordionValue readonly) surfaces on `tsgo --noEmit` — it landed with task_04 and is unrelated to this task. Tracked as an open risk in shared memory.

## Ready for Next Run

Primitives are ready for task 20 (session message thread rewrite) to compose against real SSE data. No API shifts expected — the shells stay state-free. If the Session thread reveals that an agent message needs an avatar slot distinct from the meta slot, prefer extending the meta ReactNode caller-side before adding a new slot to the primitive.
