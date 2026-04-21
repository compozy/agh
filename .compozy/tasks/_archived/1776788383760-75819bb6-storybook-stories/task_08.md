---
status: completed
title: Stories for network system
type: frontend
complexity: medium
dependencies:
  - task_05
---

# Task 8: Stories for network system

## Overview
Cover the six components of the network system: `network-channel-detail-panel`, `network-channels-list-panel`, `network-create-channel-dialog`, `network-empty-state`, `network-peer-detail-panel`, `network-peers-list-panel`. These document the peer-to-peer surface that ships with Phase 3 of AGH and serve as the reference for how paired list/detail panels render under MSW.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add stories under `web/src/systems/network/components/stories/<component>.stories.tsx` for all six components.
- MUST title stories `systems/network/<ComponentName>`.
- MUST exercise both list panels (`channels`, `peers`) with Default + Loading + Empty variants.
- MUST exercise both detail panels with Default + Not-found (404 override).
- MUST exercise `network-create-channel-dialog` with Default + Error (server rejects duplicate name).
- MUST exercise `network-empty-state` with Default only (no async path).
- MUST NOT invent new peer or channel identifiers outside the shapes exported by `systems/network/mocks/fixtures.ts`.
</requirements>

## Subtasks
- [x] 8.1 `network-channels-list-panel` stories: Default + Loading + Empty.
- [x] 8.2 `network-channel-detail-panel` stories: Default + Not-found.
- [x] 8.3 `network-peers-list-panel` stories: Default + Loading + Empty.
- [x] 8.4 `network-peer-detail-panel` stories: Default + Not-found.
- [x] 8.5 `network-create-channel-dialog` stories: Default + Duplicate-name error.
- [x] 8.6 `network-empty-state` story: Default.

## Implementation Details
Follow TechSpec "Core Interfaces" for the story template and `parameters.msw.handlers` override pattern. The 404 stories override GET handlers to respond `HttpResponse.json(null, { status: 404 })`; the 422 for create-channel returns `{ error: "channel_name_conflict" }`. Keep renders under 20 lines.

### Relevant Files
- `web/src/systems/network/components/*.tsx` — six story subjects.
- `web/src/systems/network/mocks/{handlers,fixtures,index}.ts` — data layer.
- `web/src/systems/network/adapters/network-api.ts` — path source.

### Dependent Files
- Six new files under `web/src/systems/network/components/stories/`.
- `task_11` — verification.

### Related ADRs
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md)
- [ADR-004: Per-System Mocks Directory](adrs/adr-004.md)

## Deliverables
- Six new story files for network components.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for network list/detail pairing **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `network-channels-list-panel` `Empty` story overrides GET channels to `[]` and renders the empty-state CTA.
  - [ ] `network-channel-detail-panel` `Not-found` story renders the 404 affordance, not a crash.
  - [ ] `network-create-channel-dialog` `Duplicate-name error` story keeps the dialog open with the conflict message.
  - [ ] `network-peer-detail-panel` `Default` story renders peer id from the fixture.
- Integration tests:
  - [ ] All six stories index in `build-storybook` without indexing warnings.
  - [ ] Navigating from list → detail inside Storybook via the provided in-story link updates the rendered panel without triggering real navigation.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Six story files present under `web/src/systems/network/components/stories/`.
- List + detail stories share the same fixture source via `systems/network/mocks`.
