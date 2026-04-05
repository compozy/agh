---
status: completed
title: Streaming Core & Session Store
type: ""
complexity: high
dependencies:
    - task_03
---

# Task 04: Streaming Core & Session Store

## Overview

Build the streaming infrastructure: a `SimpleStreamingBuffer` class (adapted from harnss) for accumulating SSE deltas, an `event-mapper` for transforming daemon events to UIMessage format, a Zustand session store for active session state, and a `use-session-chat` hook that wires Vercel AI SDK `useChat` to the buffer with `requestAnimationFrame` flush. This is the performance-critical data pipeline from SSE events to React state.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Key Implementation Patterns" sections 1, 2, and 3 for streaming buffer, event mapping, and chat hook patterns
- REFERENCE `.resources/harnss/src/lib/streaming-buffer.ts` for SimpleStreamingBuffer implementation
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `streaming-buffer.ts` with `SimpleStreamingBuffer` class and `mergeStreamingChunk` helper (adapted from harnss — handles overlap detection for thinking deltas)
- MUST create `event-mapper.ts` with functions to transform SSE `AgentEventPayload` into `UIMessage` fields for each event type: `agent_message`, `thought`, `tool_call`, `tool_result`, `permission`, `done`, `error`
- MUST create Zustand `session-store.ts` with state: `activeSessionId`, `messages`, `isStreaming`, `pendingPermission`; actions: `setActiveSession`, `appendMessage`, `updateLastMessage`, `setPendingPermission`, `clearSession`
- MUST create `use-session-chat.ts` hook that: uses AI SDK `useChat` for SSE transport to `POST /api/sessions/:id/prompt`, intercepts stream events via callbacks, feeds deltas to `SimpleStreamingBuffer` ref, schedules `requestAnimationFrame` flush to Zustand store
- MUST create `tool-labels.ts` mapping tool names to: icons (lucide), labels in 3 tenses (active: "Reading...", past: "Read file", failure: "read file"), and compact summary extractors
- MUST ensure streaming updates are coalesced at ~60fps via rAF (never direct setState per SSE event)
- MUST handle SSE event types: `start`, `text-start`, `text-delta`, `text-end`, `reasoning-start`, `reasoning-delta`, `reasoning-end`, `tool-input-start`, `data-agh-event`, `tool-output-available`, `data-agh-permission`, `finish`, `error`, `[DONE]`
- MUST gracefully handle unknown event types without crashing
</requirements>

## Subtasks
- [x] 4.1 Create `SimpleStreamingBuffer` class with `mergeStreamingChunk` overlap detection
- [x] 4.2 Create `event-mapper.ts` with SSE event type → UIMessage transform functions
- [x] 4.3 Create Zustand `session-store.ts` with session state and actions
- [x] 4.4 Create `use-session-chat.ts` hook wiring useChat → buffer → rAF → store
- [x] 4.5 Create `tool-labels.ts` with icon/label/summary mapping for known tools
- [x] 4.6 Update session system barrel exports

## Implementation Details

See TechSpec "Key Implementation Patterns" sections 1 and 2 for code patterns. Reference `.resources/harnss/src/lib/streaming-buffer.ts` for the SimpleStreamingBuffer and `mergeStreamingChunk` implementations.

The `use-session-chat` hook is the critical integration point. It wraps AI SDK's `useChat` with the session-specific endpoint URL (`/api/sessions/${sessionId}/prompt`), and uses `onToolCall`, `onFinish`, and potentially raw stream callbacks to intercept events that AI SDK doesn't natively handle (permissions, thinking blocks).

The Zustand store uses selectors for fine-grained subscriptions — chat view subscribes to messages, sidebar subscribes to isStreaming and pendingPermission separately.

### Relevant Files
- `.resources/harnss/src/lib/streaming-buffer.ts` — Reference SimpleStreamingBuffer + mergeStreamingChunk
- `.resources/harnss/src/hooks/useACP.ts` — Reference ACP event handling pattern
- `web/src/systems/session/types.ts` — UIMessage, AgentEventPayload, PermissionRequest types (task_01)
- `web/src/systems/session/adapters/session-api.ts` — Session API adapter (task_03)
- `internal/httpapi/prompt.go` — Exact SSE event format from daemon

### Dependent Files
- Task 05 (chat view) consumes the Zustand store and use-session-chat hook
- Task 06 (tool cards) uses tool-labels for icon/label mapping
- Task 07 (permissions) reads pendingPermission from store

### Related ADRs
- [ADR-002: AI SDK + Harnss Hybrid Streaming](../adrs/adr-002.md) — Defines the streaming architecture
- [ADR-003: Zustand + TanStack Query](../adrs/adr-003.md) — Zustand for UI/streaming state

## Deliverables
- `SimpleStreamingBuffer` class with overlap detection
- Event mapper covering all daemon SSE event types
- Zustand session store with selector-friendly state shape
- `use-session-chat` hook with rAF-based flush pipeline
- Tool labels for all known tools (Read, Write, Edit, Bash, Grep, Glob, WebSearch, etc.)
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `mergeStreamingChunk` appends non-overlapping text correctly
  - [ ] `mergeStreamingChunk` detects and merges overlapping thinking deltas
  - [ ] `mergeStreamingChunk` handles empty strings (both current and incoming)
  - [ ] `mergeStreamingChunk` handles cumulative snapshot (incoming starts with current)
  - [ ] `SimpleStreamingBuffer.appendText` accumulates chunks correctly
  - [ ] `SimpleStreamingBuffer.appendThinking` merges with overlap detection
  - [ ] `SimpleStreamingBuffer.reset` clears all state
  - [ ] `event-mapper` maps `tool_call` event to UIMessage with toolName and toolInput
  - [ ] `event-mapper` maps `tool_result` event to UIMessage with toolResult
  - [ ] `event-mapper` returns empty partial for unknown event types (no crash)
  - [ ] `session-store` `setActiveSession` replaces messages and sets activeSessionId
  - [ ] `session-store` `appendMessage` adds message to end of array
  - [ ] `session-store` `updateLastMessage` merges partial into last message
  - [ ] `session-store` `setPendingPermission` sets and clears permission state
  - [ ] `tool-labels` returns correct icon for "Read", "Write", "Edit", "Bash"
  - [ ] `tool-labels` returns fallback icon for unknown tool name
  - [ ] `tool-labels` returns active/past/failure labels for known tools
- Integration tests:
  - [ ] `use-session-chat` sends message via useChat and receives mock SSE events
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- StreamingBuffer coalesces 100+ rapid text deltas into stable getText() output
- Event mapper handles all daemon event types without errors
- Zustand store state transitions are correct and selector-friendly
- `make web-typecheck` and `make web-lint` passing
