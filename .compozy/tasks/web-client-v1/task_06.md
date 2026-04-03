---
status: pending
domain: Frontend
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_05
---

# Task 06: Tool Cards & Renderers

## Overview

Build collapsible tool call cards and specialized renderers for different tool types. Each tool invocation by the agent (Read, Write, Edit, Bash, etc.) renders as an interactive card showing the tool name, status, compact summary, and expandable content with full input/output details. This directly adapts the patterns from `.resources/harnss/src/components/ToolCall.tsx` and `tool-renderers/`.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Key Implementation Patterns" section 4 for tool card pattern
- REFERENCE `.resources/harnss/src/components/ToolCall.tsx` for collapsible card implementation
- REFERENCE `.resources/harnss/src/components/tool-renderers/` for individual renderer patterns
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `tool-call-card.tsx` as a collapsible card using shadcn `Collapsible` with: trigger showing tool icon + tool name (from `tool-labels.ts`) + compact summary + chevron; content showing expanded input/output via `ExpandedToolContent` router
- MUST show three status modes on trigger: executing (shimmer animation), success (past tense label), error (red icon + failure label)
- MUST auto-expand card when tool result arrives, auto-collapse after 2s (unless user manually toggled)
- MUST persist expand/collapse state per tool card via localStorage key `tool:{messageId}`
- MUST create `expanded-tool-content.tsx` that routes to specific renderers by tool name
- MUST create `bash-content.tsx` rendering: command executed, stdout (pre-formatted), stderr (if present, red-styled)
- MUST create `read-content.tsx` rendering: file path, file content preview (with line numbers)
- MUST create `write-content.tsx` rendering: file path, written content preview
- MUST create `edit-content.tsx` rendering: file path, old_string → new_string diff view
- MUST create `search-content.tsx` rendering: search pattern, matched files/results list
- MUST create `generic-content.tsx` as fallback: renders tool input as formatted JSON, tool result as text
- MUST integrate tool cards into `chat-view.tsx` `tool_group` rows from task 05
</requirements>

## Subtasks
- [ ] 6.1 Create `tool-call-card.tsx` with collapsible trigger, status modes, and auto-expand behavior
- [ ] 6.2 Create `expanded-tool-content.tsx` router dispatching to specific renderers
- [ ] 6.3 Create file-oriented renderers: `read-content.tsx`, `write-content.tsx`, `edit-content.tsx`
- [ ] 6.4 Create `bash-content.tsx` and `search-content.tsx` renderers
- [ ] 6.5 Create `generic-content.tsx` fallback renderer
- [ ] 6.6 Integrate tool cards into chat-view tool_group row rendering
- [ ] 6.7 Implement localStorage persistence for expand/collapse state

## Implementation Details

See TechSpec "Key Implementation Patterns" section 4 for the card pattern. Reference `.resources/harnss/src/components/ToolCall.tsx` for the collapsible pattern with auto-expand/collapse timers and localStorage persistence.

The `expanded-tool-content.tsx` is a switch on `message.toolName` routing to the appropriate renderer. Each renderer receives a `UIMessage` prop and extracts its `toolInput` and `toolResult` to render.

Tool cards are rendered inside `tool_group` RowDescriptor kinds from the chat-view's `buildRows`. A tool group contains 1+ consecutive tool_call messages, each rendered as a tool card.

### Relevant Files
- `.resources/harnss/src/components/ToolCall.tsx` — Reference collapsible card with auto-expand, status labels
- `.resources/harnss/src/components/tool-renderers/` — Reference individual renderers (BashContent, ReadContent, etc.)
- `web/src/systems/session/lib/tool-labels.ts` — Tool name → icon, label, summary (task_04)
- `web/src/systems/session/types.ts` — UIMessage, ToolUseResult types (task_01)
- `web/src/systems/session/components/chat-view.tsx` — Chat view rendering tool_group rows (task_05)
- `web/src/components/ui/collapsible.tsx` — shadcn Collapsible component

### Dependent Files
- `web/src/systems/session/components/chat-view.tsx` — Modified to render tool cards inside tool_group rows
- Task 07 (permissions) may render permission UI alongside tool cards

## Deliverables
- Collapsible tool card component with three status modes
- 6 specialized tool renderers + 1 generic fallback
- Auto-expand/collapse behavior with localStorage persistence
- Tool cards integrated into chat view tool groups
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for card rendering **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `tool-call-card` renders executing state with shimmer for tool without result
  - [ ] `tool-call-card` renders success state with past-tense label for completed tool
  - [ ] `tool-call-card` renders error state with red icon for failed tool
  - [ ] `tool-call-card` auto-expands when toolResult arrives (mock timer)
  - [ ] `expanded-tool-content` routes "Bash" tool name to bash-content renderer
  - [ ] `expanded-tool-content` routes unknown tool name to generic-content
  - [ ] `bash-content` renders command and stdout from toolResult
  - [ ] `bash-content` renders stderr in error styling when present
  - [ ] `read-content` renders file path and content preview
  - [ ] `edit-content` renders file path with old → new diff
  - [ ] `generic-content` renders toolInput as formatted JSON
- Integration tests:
  - [ ] Tool card expands on click and shows expanded content
  - [ ] Tool card collapse state persists via localStorage across re-renders
  - [ ] Chat view renders tool_group with multiple tool cards
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Tool cards render correctly for Read, Write, Edit, Bash, Grep/Glob
- Unknown tools render via generic fallback without error
- Auto-expand/collapse behavior works with 2s timer
- `make web-typecheck` and `make web-lint` passing
