---
status: done
title: Restyle session chat view
type: frontend
complexity: medium
dependencies:
  - task_01
  - task_02
---

# Task 03: Restyle session chat view

## Overview

Update the existing session chat view components to match the Paper design: right-aligned user message bubbles, left-aligned agent messages without bubbles, restyled tool call cards with status badges, new chat input with accent send button, and updated session header with breadcrumb pattern. The data layer and hooks remain unchanged — only styling and markup change.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "Session Chat View (Updated Styling)" and "Page Designs" sections
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST style user messages as right-aligned bubbles: bg `#2C2C2E`, radius 12px, padding 16px 20px
- MUST style agent messages as left-aligned, no bubble: agent label with status dot + JetBrains Mono 11px uppercase name + timestamp
- MUST style tool call cards: bg `#1C1C1E`, border 1px `#3A3A3C`, terminal icon, tool name, file path, right-aligned status badge (DONE=green, RUNNING=accent, ERROR=red)
- MUST style chat input: bg `#1C1C1E`, radius 12px, border `#3A3A3C`, focused border `#E8572A`
- MUST style send button: 36px circle, bg `#E8572A`, white send icon
- MUST style session header with breadcrumb pattern: agent avatar dot + name + session dropdown + "+" button
- MUST update empty state (index page): centered terminal icon 48px, text per Paper spec
</requirements>

## Subtasks
- [x] 3.1 Restyle `chat-header.tsx` with breadcrumb/session header pattern per Paper
- [x] 3.2 Restyle `message-bubble.tsx` for user (right-aligned bubble) and agent (left-aligned, no bubble) messages
- [x] 3.3 Restyle `tool-call-card.tsx` with new card styling and status badges
- [x] 3.4 Restyle `message-composer.tsx` with new input and accent send button
- [x] 3.5 Update `index.tsx` empty state to match Paper empty state design
- [x] 3.6 Write tests verifying message alignment, status badge rendering, and input styling

## Implementation Details

See TechSpec "Session Chat View (Updated Styling)" and DESIGN.md "Chat Components" sections for exact styling specs.

All existing hooks (`useSessionChat`, `useSessionTranscript`, `useSessionStore`) and data flow remain unchanged. This task only modifies component JSX and Tailwind classes.

### Relevant Files
- `web/src/systems/session/components/chat-header.tsx` — Session header, restyle with breadcrumb
- `web/src/systems/session/components/message-bubble.tsx` — Message rendering, restyle per role
- `web/src/systems/session/components/tool-call-card.tsx` — Tool execution card, restyle
- `web/src/systems/session/components/message-composer.tsx` — Chat input + send, restyle
- `web/src/systems/session/components/chat-view.tsx` — Message list container
- `web/src/routes/_app/index.tsx` — Empty state page
- `web/src/routes/_app/session.$id.tsx` — Session page (may need minor layout adjustments)

### Dependent Files
- `web/src/systems/session/components/processing-indicator.tsx` — May need styling update
- `web/src/systems/session/components/thinking-block.tsx` — May need styling update

### Related ADRs
- [ADR-001: Full Replace of Design Token System](../adrs/adr-001.md) — New tokens used for all colors/typography

## Deliverables
- Restyled chat header, message bubbles, tool call cards, composer, and empty state
- All components use DESIGN.md tokens exclusively
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] User message renders with right-aligned class and `#2C2C2E` background
  - [x] Agent message renders left-aligned with no bubble background
  - [x] Agent label shows JetBrains Mono uppercase name with status dot
  - [x] Tool call card shows terminal icon and tool name
  - [x] Tool call card DONE status renders green badge
  - [x] Tool call card RUNNING status renders accent badge
  - [x] Tool call card ERROR status renders red badge
  - [x] Chat input gains accent border color on focus
  - [x] Send button is circular with accent background
  - [x] Empty state shows terminal icon and descriptive text
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Session page matches Paper artboard "AGH Sidebar — Sessions in Header"
- Empty state matches Paper artboard "AGH Sidebar — Collapsed" (content area)
- `make web-lint && make web-typecheck` passes
