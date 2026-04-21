# Task Memory: task_20.md

## Objective Snapshot

Rewrite `web/src/systems/session/components/**` message thread on top of `@agh/ui` primitives (ChatMessageBubble, ToolCallCard, CodeBlock, StatusDot, MonoBadge, Button, ScrollArea). SSE pipeline + virtualizer untouched; only JSX + styling change. All 5 message roles supported (system/user/agent/tool/diff). Stories + Playwright baselines + unit + integration tests for 80%+ coverage.

## Important Decisions

- Extend `UIMessage` with `"diff"` role + optional `diff?: { language?; content; path?; additions?; removals? }` shape — additive, does NOT break the existing pipeline; MessageBubble now paints diffs natively inside ChatMessageBubble role="diff" wrapping a CodeBlock. Pipeline doesn't emit diff messages yet; future tasks can light them up.
- Extend `@agh/ui` ToolCallCard primitive with an optional `icon` prop (Lucide component OR pre-rendered ReactNode) — mirrors the Empty primitive's icon handling. When omitted, defaults to TerminalIcon (backwards-compatible with existing primitive tests).
- Session `ToolCallCard` composition: the session component wraps the primitive inside a click-to-expand `<button>` with `data-testid="tool-card-trigger"`. Inside the button we render the primitive (header only; no body). Below the button, we conditionally render `ExpandedToolContent` in a sibling div when expanded — this keeps valid HTML (button cannot wrap block children that contain interactive content).
- Session `MessageBubble` maps roles: `user → ChatMessageBubble role="user"`, `assistant → role="agent"`, `system → role="system"`, `diff → role="diff"`. Tool messages still flow through `ToolGroupSection → ToolCallCard` (not MessageBubble).
- Agent status dot: use `StatusDot` primitive at `size="md"` (8px) with tone derived from `isStreaming ? "accent" : "success"`. StatusDot uses inline style `backgroundColor` not a class, so tests assert via `data-tone="success"` attribute.
- Tool status badge text flips from Title-Case (`Running/Done/Error`) to UPPER-CASE (`RUNNING/DONE/ERROR`) per DESIGN.md §4 and the primitive's `STATUS_LABEL` lookup. Tests query via `[data-slot="tool-call-card-status"]` + check text + `data-tone` attribute.
- Tool status tone class assertions (`bg-[color:var(--color-accent-tint)]` etc.) still pass — the primitive's MonoBadge uses identical tint + text color classes.
- Copy button kept — `@agh/ui` doesn't ship a `CopyButton`. Moved styling to tokens-only (already token-based).
- Processing indicator: kept inline SVG dots but moved all colors to `StatusDot` with tone="neutral" + pulse. Cleaner + respects prefers-reduced-motion via the primitive.

## Learnings

- `toHaveTextContent` is case-sensitive — Title-Case → Upper-Case migration requires updating assertions.
- The primitive's `data-slot="tool-call-card-status"` + `data-tone` attribute is a stable test selector, no need for a bespoke test-id prop.
- `ChatMessageBubble role="user"` sets `justify-end` on the outer flex container; the bubble body has `bg-[color:var(--color-surface-elevated)]` + `rounded-[var(--radius-lg)]` (not `rounded-2xl`). Tests that asserted Tailwind's `rounded-2xl` shorthand need switching to the CSS-var radius token.
- Fixtures file already exports `systemMessageFixture`, `streamingAssistantMessageFixture`, and all tool fixtures — no new fixtures needed aside from a single `diffMessageFixture`.

## Files / Surfaces

- `packages/ui/src/components/tool-call-card.tsx` — add `icon?` prop + isComponentType helper.
- `packages/ui/src/components/tool-call-card.test.tsx` — 1 new test for custom icon.
- `web/src/systems/session/types.ts` — UIMessageRole adds "diff"; UIMessage adds optional `diff`.
- `web/src/systems/session/components/message-bubble.tsx` — rewrite on ChatMessageBubble.
- `web/src/systems/session/components/message-bubble.test.tsx` — update assertions.
- `web/src/systems/session/components/tool-call-card.tsx` — rewrite composing primitive.
- `web/src/systems/session/components/tool-call-card.test.tsx` — update status badge queries.
- `web/src/systems/session/components/tool-group-section.tsx` — minor token cleanup.
- `web/src/systems/session/components/chat-view.tsx` — swap empty state to `Empty` primitive.
- `web/src/systems/session/components/chat-view.integration.test.tsx` — update empty-state assertion.
- `web/src/systems/session/components/chat-header.tsx` — StatusDot + MonoBadge.
- `web/src/systems/session/components/chat-header.test.tsx` — update status-dot assertion via data-tone.
- `web/src/systems/session/components/processing-indicator.tsx` — StatusDot dots.
- `web/src/systems/session/mocks/fixtures.ts` — add `diffMessageFixture`.
- `web/src/systems/session/components/stories/*.stories.tsx` — update stories to exercise new primitives + add diff + running-tool variants.
- `web/tests/visual/__snapshots__/` — regenerate baselines.

## Errors / Corrections

- Initial primitive tests failed because `FileEditIcon` lucide classname is `lucide-file-pen` in Lucide 1.8.0 (the icon was renamed). Fixed test assertion.
- Tool-call-card session component's `card.status` widened to `string` by TS inference — fixed by annotating the hook's return type with `ToolCallStatus` import.
- Removed old `@agh/ui` mocks from `permission-prompt.integration.test.tsx` and `chat-view.integration.test.tsx`; once message-bubble started importing `ChatMessageBubble`, those tests needed either the real module or importActual — replaced with `importActual`-based `cn` mock for `@/lib/utils` only.
- Stale Playwright baselines removed: `systems-session-messagebubble--assistant-...png` (replaced by `--agent-`) and `systems-session-toolgroupsection--empty-...png` (story removed because empty tool group now renders null).

## Ready for Next Run

- Task 20 complete. Session message thread + all chrome rewritten on `@agh/ui`.
- Phase 4 step 1 landed. Task 21 (composer) + task 22 (inspector) depend on this.
- Follow-ups to consider for task 21/22:
  - Extended `ToolCallCard` primitive with optional `icon` prop — task 21's composer affordances may also need per-action icon slots; reuse same pattern (Lucide component OR ReactNode).
  - `UIMessageRole` now includes `"diff"`; session-store / transcript assembler still emit Edit tool results as `tool_result` with `structuredPatch`. A follow-up task can light up native `diff` messages by extending the assembler — out of scope for task 20.
  - Stale `systems-session-toolgroupsection--empty-...png` baseline removed; no replacement story because `ToolGroupSection` returns null for empty input by design.
