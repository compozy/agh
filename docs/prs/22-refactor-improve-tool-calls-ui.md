# PR #22: refactor: improve tool calls ui

- **URL**: https://github.com/compozy/agh/pull/22
- **Author**: @pedronauck
- **State**: merged
- **Created**: 2026-04-15T13:24:00Z
- **Merged**: 2026-04-15T14:01:28Z

## Summary by CodeRabbit

- **New Features**
  - Copy-to-clipboard controls for messages, code blocks, and tool outputs
  - Expandable tool groups with “show more/less” toggles
  - “Show full content” controls for truncated tool edit/write outputs
  - Live elapsed-time display during AI processing ("Working for X")

- **Improvements**
  - Reduced UI churn/flicker during streaming for smoother updates
  - Updated message bubble styling with conditional copy/timestamp row
  - Tool summaries show tone-based styling and hoverable full-summary tooltips

- **Tests**
  - Added tests covering copy behavior, tool summaries, icons, and code-block copy visibility

## Walkthrough

Stabilizes chat row objects to reduce virtualizer churn during streaming; adds a CopyButton (messages, code blocks) and tests; replaces spinner with a live elapsed-time processing indicator; introduces a memoized ToolGroupSection; adds truncation toggles in tool renderers and a tone/icon system for tools.

## Changes

| Cohort / File(s)                                                                                                                                                               | Summary                                                                                                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Chat row stabilization & grouping** <br> `web/src/systems/session/components/chat-view.tsx`                                                                                  | Replaced per-card inline rendering for `"tool_group"` rows with a new `ToolGroupSection`; introduced `isRowUnchanged`, `computeStableRows`, and `useStableRows` to preserve `RowDescriptor` references during streaming; `ChatViewContent` now uses `useStableRows(messages, isStreaming)`.              |
| **Tool group component** <br> `web/src/systems/session/components/tool-group-section.tsx`                                                                                      | Added exported memoized `ToolGroupSection` and `ToolGroupSectionProps { tools: UIMessage[] }`; implements collapse/expand with `MAX_VISIBLE_ENTRIES` (6) and renders visible `ToolCallCard`s.                                                                                                            |
| **Tool labels / tone & icon logic** <br> `web/src/systems/session/lib/tool-labels.ts`, `web/src/systems/session/lib/tool-labels.test.ts`                                       | Added `ToolTone` type, `getToolTone`, `toolToneClass`; extended `getToolIcon(toolName, toolInput?)` semantic fallbacks by `toolInput` keys; separated compact vs full summaries (`getToolCompactSummary`, `getToolFullSummary`). Unit tests added for semantic icon fallbacks.                           |
| **Tool call card & tooltip behavior** <br> `web/src/systems/session/components/tool-call-card.tsx`, `web/src/systems/session/components/tool-call-card.test.tsx`               | Pass `message.toolInput` to icon/summary helpers; compute `tone` and apply `toolToneClass`; render truncated compact summary with tooltip showing full summary when different; tests updated/added for tooltip behavior.                                                                                 |
| **Message bubble & copy UI** <br> `web/src/systems/session/components/message-bubble.tsx`, `web/src/systems/session/components/message-bubble.test.tsx`                        | Added `CopyButton` to user and assistant bubbles; moved timestamp into a bottom row; adjusted bubble classes (e.g., `rounded-2xl`) and hover grouping for copy controls; test updated to expect styling change and absence of copy button for empty content.                                             |
| **Code blocks copy** <br> `web/src/systems/session/components/message-markdown.tsx`, `web/src/systems/session/components/message-markdown.test.tsx`                            | Wraps syntax-highlight output in `group/codeblock relative` and overlays `CopyButton` (ariaLabel: “Copy code”) that copies full code; tests assert copy button presence and focus/hover visibility classes.                                                                                              |
| **CopyButton component & tests** <br> `web/src/systems/session/components/copy-button.tsx`, `web/src/systems/session/components/copy-button.test.tsx`                          | New exported `CopyButton` and `CopyButtonProps` (`ariaLabel`, `text`, `className?`); writes `text` to clipboard, sets transient `copied` state/reset timer (1200ms), handles failures with console.error, cleans up timers on unmount; tests cover success/failure with fake timers and clipboard mocks. |
| **Processing indicator** <br> `web/src/systems/session/components/processing-indicator.tsx`                                                                                    | Replaced spinner with pulsing three-dot indicator and live elapsed-time display; added `formatElapsed` and `WorkingTimer` (ticks every second) and updated layout/text to “Working for <timer>”.                                                                                                         |
| **Tool renderers truncation** <br> `web/src/systems/session/components/tool-renderers/edit-content.tsx`, `web/src/systems/session/components/tool-renderers/write-content.tsx` | Introduced `showFull` state and `TRUNCATE_THRESHOLD`; compute `isTruncated` and `display*` values; add “Show full content” toggle (with `ChevronsUpDown`) to reveal full text and counts.                                                                                                                |
