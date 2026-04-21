---
status: completed
title: Stories for session core components
type: frontend
complexity: high
dependencies:
  - task_05
---

# Task 9: Stories for session core components

## Overview
Cover the 12 top-level session components: `chat-header`, `chat-view`, `copy-button`, `message-bubble`, `message-composer`, `message-markdown`, `permission-prompt`, `processing-indicator`, `session-sidebar-item`, `thinking-block`, `tool-call-card`, `tool-group-section`. Sessions are the product's primary surface; these stories are the most-referenced visual documentation and exercise streaming, markdown rendering, and permission gating under MSW.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add story files under `web/src/systems/session/components/stories/<component>.stories.tsx` for each of the 12 components.
- MUST title stories `systems/session/<ComponentName>`.
- MUST render `chat-view` as the integrated example that composes `message-bubble`, `processing-indicator`, and `tool-call-card`; other stories must focus on one concept each.
- MUST include a streaming/loading variant for `chat-view`, `processing-indicator`, and `thinking-block` using handler `delay` or `ReadableStream` responses exposed in `systems/session/mocks`.
- MUST include a `permission-prompt` story with Accept, Reject, and Pending states driven by handler overrides.
- MUST render `message-markdown` with code block, heading, list, and link content covered by at least one story each (can share stories).
- MUST NOT inline large markdown fixtures — import from `systems/session/mocks/fixtures.ts`.
- MUST NOT create more than 5 stories per component; collapse variants using the same fixture when they don't change substantive UI.
</requirements>

## Subtasks
- [x] 9.1 Chat surface stories: `chat-header` (Default), `chat-view` (Default + Streaming + Error).
- [x] 9.2 Message stories: `message-bubble` (User + Assistant + System), `message-markdown` (Default covering code/heading/list/link).
- [x] 9.3 Composer + copy stories: `message-composer` (Default + Disabled), `copy-button` (Default + Copied).
- [x] 9.4 Permission + processing stories: `permission-prompt` (Pending + Accepted + Rejected), `processing-indicator` (Default + Long-running), `thinking-block` (Collapsed + Expanded).
- [x] 9.5 Tool + sidebar stories: `tool-call-card` (Running + Done + Error), `tool-group-section` (Default + Empty), `session-sidebar-item` (Default + Selected + Unread).

## Implementation Details
Follow the TechSpec "Core Interfaces" story template. For streaming, reuse the existing stream helpers under `web/src/systems/session/mocks/handlers.ts` (produce delayed chunks via `new ReadableStream`). For `permission-prompt`, override the websocket-ish permission API using MSW's `http` handlers with the appropriate delayed resolution. Keep every `render` under 20 lines; heavy fixture data lives in `mocks/fixtures.ts`.

### Relevant Files
- `web/src/systems/session/components/*.tsx` — 12 story subjects.
- `web/src/systems/session/mocks/{handlers,fixtures,index}.ts` — streaming + permission fixtures.
- `packages/ui/src/index.ts` and `web/src/components/ui/*` — primitives for composition.

### Dependent Files
- 12 new files under `web/src/systems/session/components/stories/`.
- `task_10` — session tool-renderer stories share the same mocks.
- `task_11` — verification.

### Related ADRs
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md)
- [ADR-004: Per-System Mocks Directory](adrs/adr-004.md)

## Deliverables
- 12 new story files under `web/src/systems/session/components/stories/`.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for streaming and permission flows **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `chat-view` `Streaming` story renders partial content, then the finalized message as chunks resolve.
  - [ ] `message-markdown` `Default` story renders the fixture markdown with a syntax-highlighted code block (check the presence of a highlighted `<pre>` wrapper).
  - [ ] `permission-prompt` `Rejected` story returns the rejection state and does not re-prompt.
  - [ ] `tool-call-card` `Error` story renders an error state with the error message from the fixture.
  - [ ] `session-sidebar-item` `Unread` story renders an unread indicator element with the correct `aria-label`.
- Integration tests:
  - [ ] All 12 stories index in `build-storybook` and render without runtime warnings.
  - [ ] `chat-view` `Streaming` story completes streaming within 5 seconds under Storybook's iframe timing.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- 12 session-core story files present.
- Streaming stories render incremental content and finalize.
- `permission-prompt` exercises all three outcomes via MSW overrides.
