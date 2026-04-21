---
status: completed
title: Stories for session tool-renderers
type: frontend
complexity: medium
dependencies:
  - task_05
---

# Task 10: Stories for session tool-renderers

## Overview
Cover the seven tool-renderer components nested under `web/src/systems/session/components/tool-renderers/`: `bash-content`, `edit-content`, `expanded-tool-content`, `generic-content`, `read-content`, `search-content`, `write-content`. These render ACP tool invocations inline in the chat transcript; their stories must present realistic tool payloads for each renderer kind so the design review can validate formatting.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add stories under `web/src/systems/session/components/tool-renderers/stories/<component>.stories.tsx` for all seven components.
- MUST title stories `systems/session/tool-renderers/<ComponentName>`.
- MUST include at minimum a `Default` story per component; add a truncated/large-output variant for `bash-content`, `read-content`, `search-content`, and `write-content`.
- MUST include a `Running` variant for `expanded-tool-content` that exercises the in-progress renderer state.
- MUST source tool payload fixtures from `systems/session/mocks/fixtures.ts` — keep tool-renderer-specific fixtures co-located there under named exports (`bashFixture`, `editFixture`, …).
- MUST NOT duplicate tool-payload shapes across stories; import the typed fixture and override a single field when a variant demands it.
</requirements>

## Subtasks
- [x] 10.1 `bash-content` story: Default + Long output.
- [x] 10.2 `edit-content` story: Default (single-file diff) + Multi-hunk.
- [x] 10.3 `expanded-tool-content` story: Default + Running.
- [x] 10.4 `generic-content` story: Default (unknown tool fallback).
- [x] 10.5 `read-content` story: Default + Truncated.
- [x] 10.6 `search-content` story: Default + Empty result set.
- [x] 10.7 `write-content` story: Default + Overwrite warning.

## Implementation Details
Follow TechSpec "Core Interfaces" story template. Tool-renderer stories do not need MSW overrides — renderers are pure props-in/render-out. Add tool fixtures to `systems/session/mocks/fixtures.ts` so they're available to unit tests as well.

### Relevant Files
- `web/src/systems/session/components/tool-renderers/*.tsx` — seven story subjects.
- `web/src/systems/session/mocks/fixtures.ts` — fixture source to extend.
- `web/src/systems/session/components/tool-call-card.tsx` — renders these children; story for the parent lives in task_09.

### Dependent Files
- Seven new files under `web/src/systems/session/components/tool-renderers/stories/`.
- `task_11` — verification.

### Related ADRs
- [ADR-003: stories/ Subfolder Placement, Opt-in Autodocs](adrs/adr-003.md) — Nested `stories/` under `tool-renderers/` per the convention.

## Deliverables
- Seven new story files.
- Extended `systems/session/mocks/fixtures.ts` with per-tool fixtures.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for tool-renderer rendering **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `bash-content` `Long output` story renders a truncation indicator when output length exceeds the threshold.
  - [ ] `edit-content` `Multi-hunk` story renders exactly two diff-hunk sections.
  - [ ] `expanded-tool-content` `Running` story renders an in-progress indicator and no final artifact.
  - [ ] `generic-content` `Default` story renders the fallback copy for unknown tools.
  - [ ] `search-content` `Empty result set` story renders the empty affordance, not a zero-length list.
- Integration tests:
  - [ ] All seven stories index in `build-storybook` and render without warnings.
  - [ ] Tool fixtures imported by stories match the adapter-typed shape used in `tool-call-card`.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Seven story files present under `web/src/systems/session/components/tool-renderers/stories/`.
- Fixtures consolidated in `systems/session/mocks/fixtures.ts`.
