---
status: completed
title: Chat View, Messages & Composer
type: ""
complexity: high
dependencies:
    - task_04
---

# Task 05: Chat View, Messages & Composer

## Overview

Build the main chat interface: a virtualized message list using `@tanstack/react-virtual`, message bubble components with markdown rendering, a thinking block display, an input composer with send capability, a chat header with session info, and a processing indicator. Wire everything into the `session.$id.tsx` route to create the complete conversation view.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Key Implementation Patterns" section 3 for ChatView virtualization
- REFERENCE `.resources/harnss/src/components/ChatView.tsx` for row model and virtualization patterns
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `chat-view.tsx` with `@tanstack/react-virtual` virtualizer: `RowDescriptor` union type (message | tool_group | processing), pure `buildRows` function, height estimation per row kind, bottom-lock scroll behavior, scroll-to-bottom button
- MUST create `message-bubble.tsx` rendering user and assistant messages with `react-markdown` + `remark-gfm` for markdown, code blocks with syntax highlighting via `react-syntax-highlighter`
- MUST memoize markdown rendering to avoid re-parsing during streaming (only re-render when content string changes)
- MUST create `thinking-block.tsx` as a collapsible component for reasoning/thinking text, visually distinct from regular messages
- MUST create `message-composer.tsx` with textarea input, Send button, Enter to send (Shift+Enter for newline), disabled state during streaming or pending permission
- MUST create `chat-header.tsx` showing: session name, agent name, session state badge, Stop/Resume buttons
- MUST create `processing-indicator.tsx` showing an animated indicator when agent is processing
- MUST wire all components into `routes/_app/session.$id.tsx`: load session via route params, connect to Zustand store, initialize `use-session-chat` hook
- MUST support bottom-lock: auto-scroll during streaming, disable on manual scroll up, button to re-engage
- MUST handle loading state (fetching session), error state (session not found), and empty state (no messages yet)
</requirements>

## Subtasks
- [x] 5.1 Create `chat-view.tsx` with virtualized row rendering and bottom-lock scroll
- [x] 5.2 Create `message-bubble.tsx` with markdown rendering and syntax highlighting
- [x] 5.3 Create `thinking-block.tsx` as collapsible reasoning display
- [x] 5.4 Create `message-composer.tsx` with send behavior and disabled states
- [x] 5.5 Create `chat-header.tsx` with session info and Stop/Resume actions
- [x] 5.6 Create `processing-indicator.tsx` with streaming animation
- [x] 5.7 Wire all components into `session.$id.tsx` route with store + chat hook

## Implementation Details

See TechSpec "Key Implementation Patterns" section 3 for the `RowDescriptor` union and `buildRows` function pattern. Reference `.resources/harnss/src/components/ChatView.tsx` for virtualization setup and scroll behavior.

The `chat-view.tsx` consumes `messages` from Zustand store and `isStreaming` flag. The `buildRows` function groups consecutive tool_call/tool_result messages into `tool_group` rows for task 06 to render. The `message-composer.tsx` calls `use-session-chat`'s send function.

For markdown, use `react-markdown` with `remark-gfm` plugin. Wrap in `React.memo` with content string as dependency. Code blocks use `react-syntax-highlighter` with a Tailwind-compatible theme.

### Relevant Files
- `.resources/harnss/src/components/ChatView.tsx` — Reference virtualization, row model, scroll behavior
- `.resources/harnss/src/components/ToolCall.tsx` — Reference for how tool groups are rendered
- `web/src/systems/session/stores/session-store.ts` — Zustand store (task_04)
- `web/src/systems/session/hooks/use-session-chat.ts` — Chat hook (task_04)
- `web/src/systems/session/hooks/use-session-actions.ts` — Stop/Resume mutations (task_03)
- `web/src/components/ui/scroll-area.tsx` — shadcn scroll area component
- `web/src/components/ui/collapsible.tsx` — For thinking block collapse
- `web/src/components/ui/badge.tsx` — For session state in chat header
- `web/src/components/ui/button.tsx` — For send, stop, resume buttons
- `web/src/components/ui/textarea.tsx` — For message input

### Dependent Files
- `web/src/routes/_app/session.$id.tsx` — Updated from placeholder to full chat view
- Task 06 (tool cards) will render inside `tool_group` rows defined here
- Task 07 (permissions) will render permission prompt inside chat view

## Deliverables
- Virtualized chat view handling 1000+ messages at 60fps scroll
- Message bubbles with full markdown rendering and syntax-highlighted code blocks
- Collapsible thinking blocks for agent reasoning
- Message composer with keyboard shortcuts and proper disabled states
- Chat header with session info and lifecycle actions
- Processing indicator during agent streaming
- Session route fully wired with all components
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for component rendering **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `buildRows` groups consecutive tool_call + tool_result messages into tool_group
  - [x] `buildRows` adds processing row when isStreaming is true
  - [x] `buildRows` handles empty messages array
  - [x] `buildRows` preserves non-tool messages as individual message rows
  - [x] `message-bubble` renders markdown headings, code blocks, and links
  - [x] `message-bubble` memoizes and doesn't re-render when content is unchanged
  - [x] `message-composer` calls send on Enter key press
  - [x] `message-composer` inserts newline on Shift+Enter
  - [x] `message-composer` is disabled when isStreaming is true
- Integration tests:
  - [x] Chat view renders user and assistant messages from Zustand store
  - [x] Chat view auto-scrolls to bottom during streaming
  - [x] Chat header shows session name and correct state badge
  - [x] Stop button calls stopSession mutation
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Chat view renders messages with markdown at 60fps scroll
- Bottom-lock scroll works during streaming
- Composer sends messages and disables during streaming
- Session route loads session data and connects to streaming
- `make web-typecheck` and `make web-lint` passing
