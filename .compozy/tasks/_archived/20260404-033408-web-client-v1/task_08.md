---
status: completed
domain: Frontend
type: Feature Implementation
scope: Full
complexity: high
dependencies:
  - task_05
  - task_06
  - task_07
---

# Task 08: Permissions, History & Session Navigation

## Overview

Build the permission prompt UI for interactive tool approval, session history loading for navigating to existing sessions, and session switch logic for preserving state across navigation. This is the final task that completes the full conversation loop: users can approve/reject agent tool requests, navigate between sessions loading their history, and switch sessions without losing state.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Key Implementation Patterns" section 5 for permission prompt
- REFERENCE TECHSPEC "Data Flow" items 4 and 5 for history loading and permission flow
- REFERENCE `.resources/harnss/src/components/` for permission UI patterns
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `permission-prompt.tsx` component rendering inline in chat when `pendingPermission` is set in Zustand store. Shows: tool name, action, resource, formatted tool input. Buttons: Allow Once, Allow Always, Reject Once, Reject Always
- MUST send permission response via `POST /api/sessions/:id/approve` (implemented in task_07). Request body: `{ "turn_id": string, "decision": "allow-once" | "allow-always" | "reject-once" | "reject-always" }`
- MUST clear `pendingPermission` in Zustand store after user responds
- MUST add pulsing amber dot indicator to `session-sidebar-item.tsx` when session has pending permission
- MUST disable message composer while permission is pending (agent is blocked)
- MUST create `use-session-history.ts` hook that fetches `GET /api/sessions/:id/history` and transforms `TurnHistoryPayload[]` into `UIMessage[]` for rendering in chat view
- MUST implement session switch logic: when navigating to a different session, save current Zustand messages, load target session's history into store, and reconnect SSE stream if session is active
- MUST handle session history reconstruction: map stored events (agent_message, tool_call, tool_result, thought) back to UIMessage array without streaming animation (static render)
- MUST handle edge cases: session not found (404 → navigate away + toast), session stopped mid-navigation (show history only, no composer), reconnection to active session's stream on switch
</requirements>

## Subtasks
- [ ] 8.1 Create `permission-prompt.tsx` with tool info display and allow/reject buttons
- [ ] 8.2 Wire permission prompt into chat view and connect to Zustand store
- [ ] 8.3 Add permission pending indicator (pulsing amber dot) to session sidebar item
- [ ] 8.4 Create `use-session-history.ts` hook to fetch and transform event history to UIMessages
- [ ] 8.5 Implement session switch logic: save/restore messages across navigation
- [ ] 8.6 Handle SSE stream reconnection when switching to an active session
- [ ] 8.7 Handle error states: 404 session not found, stopped session, approve endpoint errors

## Implementation Details

See TechSpec "Key Implementation Patterns" section 5 for permission prompt pattern and "Data Flow" items 4-5 for history and permission flows.

The permission prompt appears as a card in the chat view when `sessionStore.pendingPermission` is non-null. It's inserted at the bottom of the message list (above the composer). The response POSTs to `/api/sessions/:id/approve` — since this endpoint returns 501, show a toast "Permission sent (backend pending implementation)" and clear the store state.

Session history loading uses `GET /api/sessions/:id/history` which returns `TurnHistoryPayload[]`. Each turn contains ordered events that are transformed to UIMessage entries via `event-mapper.ts` (from task_04). History messages have `isStreaming: false` and no animation.

Session switch: on navigate to `/session/:newId`, call `sessionStore.setActiveSession(newId, historyMessages)`. If the new session is in `active` state, initialize `use-session-chat` to reconnect to its SSE stream for future prompts.

### Relevant Files
- `.resources/harnss/src/components/` — Reference permission prompt patterns
- `.resources/harnss/src/hooks/session/useSessionLifecycle.ts` — Reference session switch pattern
- `web/src/systems/session/stores/session-store.ts` — Zustand store with pendingPermission (task_04)
- `web/src/systems/session/hooks/use-session-chat.ts` — Chat hook for SSE reconnection (task_04)
- `web/src/systems/session/lib/event-mapper.ts` — Event → UIMessage transforms (task_04)
- `web/src/systems/session/adapters/session-api.ts` — fetchSessionHistory, approveSession (task_03)
- `web/src/systems/session/components/session-sidebar-item.tsx` — Add permission indicator (task_03)
- `web/src/systems/session/components/chat-view.tsx` — Insert permission prompt (task_05)
- `web/src/systems/session/components/message-composer.tsx` — Disable when permission pending (task_05)
- `internal/httpapi/prompt.go` — SSE permission event format reference

### Dependent Files
- `web/src/systems/session/components/session-sidebar-item.tsx` — Modified to show amber dot
- `web/src/systems/session/components/chat-view.tsx` — Modified to render permission prompt
- `web/src/systems/session/components/message-composer.tsx` — Modified to disable on pending permission
- `web/src/routes/_app/session.$id.tsx` — Modified for session switch logic
- `web/src/systems/session/index.ts` — Updated barrel

## Deliverables
- Permission prompt component with allow/reject options (4 buttons)
- Pulsing amber indicator on sidebar items with pending permissions
- Session history loading and UIMessage reconstruction
- Session switch save/restore logic
- SSE reconnection on session switch
- Error handling for approve endpoint failures
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for permission flow and history loading **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `permission-prompt` renders tool name, action, and resource from PermissionRequest
  - [ ] `permission-prompt` renders all 4 action buttons (Allow Once/Always, Reject Once/Always)
  - [ ] `permission-prompt` calls approve API and clears store on button click
  - [ ] `permission-prompt` handles approve API error gracefully (shows toast, clears state)
  - [ ] `use-session-history` transforms TurnHistoryPayload[] into UIMessage[] correctly
  - [ ] `use-session-history` maps tool_call events to tool_call role messages
  - [ ] `use-session-history` maps thought events to messages with thinking field
  - [ ] `use-session-history` handles empty history (no events)
  - [ ] Session switch saves current messages before loading new session
  - [ ] Session switch loads history messages into store for target session
- Integration tests:
  - [ ] Permission prompt appears when pendingPermission is set in store
  - [ ] Composer is disabled while permission is pending
  - [ ] Sidebar item shows amber dot when session has pending permission
  - [ ] Navigating to existing session loads its history into chat view
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Permission prompt renders correctly with all 4 action buttons
- Session history loads and renders correctly on navigation
- Session switch preserves state for previous session
- Pulsing amber dot visible on sidebar for sessions with pending permissions
- Approve endpoint errors are handled gracefully without breaking UI
- `make web-typecheck` and `make web-lint` passing
- Full end-to-end loop works: create session → send message → see streaming → approve permission → navigate to another session → return to original session with history
