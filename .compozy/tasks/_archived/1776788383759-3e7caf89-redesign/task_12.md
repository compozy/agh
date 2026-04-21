---
status: completed
title: Write packages/ui contributor guide (README.md)
type: docs
complexity: low
dependencies:
  - task_02
  - task_03
  - task_04
  - task_05
  - task_06
  - task_07
  - task_08
  - task_09
  - task_10
  - task_11
---

# Task 12: Write packages/ui contributor guide (README.md)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Produce `packages/ui/README.md` — the canonical contributor guide for `@agh/ui`. Covers: primitive inventory, design-system alignment, when to use CSS vs `motion`, story contribution rules, Playwright snapshot workflow, and the `UIProvider` wiring.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `packages/ui/README.md` covering: purpose, primitive inventory (grouped: foundations, structural, form, feedback, chat), token source, UIProvider wiring, story contribution rules, motion vs CSS decision rules, Playwright snapshot workflow, CI gate expectations.
- MUST link to the root `DESIGN.md` as the authoritative visual spec.
- MUST link to ADR-001 through ADR-005.
- MUST include a one-paragraph "when to add a primitive here vs in web/" decision rule matching ADR-001.
- MUST include a short "anti-patterns" section (no domain imports inside `@agh/ui`, no AGH-specific defaults in primitive props).
- SHOULD keep the file under 500 lines.
</requirements>

## Subtasks

- [x] 12.1 Draft README structure with the required sections.
- [x] 12.2 Enumerate every primitive exported from `packages/ui/src/index.ts` after task 10 + link each to its story.
- [x] 12.3 Document the `UIProvider` setup and the expected `reducedMotion` behavior.
- [x] 12.4 Write the motion vs CSS decision rules with three concrete examples (hover color change → CSS; unmount animation → motion; route transition → motion).
- [x] 12.5 Document the Playwright snapshot workflow (generate, update, review).

## Implementation Details

This is a docs task. No unit tests in the traditional sense — replace the test requirement with a markdown lint + link-check step (below). Reference TechSpec "Development Sequencing" and ADR-004 for rollout context.

### Relevant Files

- `packages/ui/README.md` — new.
- `DESIGN.md` — linked from the README.
- `.compozy/tasks/20260418-132824-web-redesign/adrs/adr-001.md` through `adr-005.md` — linked.
- `packages/ui/src/index.ts` — authoritative export list.
- **Design references** (read-only, do not edit):
  - `DESIGN.md` — link target for foundational rules.
  - `docs/design/design-system/README.md` — structural + voice reference; the packages/ui README mirrors its sectioning and tone.
  - `docs/design/design-system/SKILL.md` — Claude-invocable skill contract; the "when to add a primitive" rule block lives here and should be cross-referenced.

### Dependent Files

- Every contributor reading `packages/ui/README.md` during Phase 2–6 rollout.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-003: Adopt motion for UI animations](adrs/adr-003.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- `packages/ui/README.md` covering all required sections.
- Automated link-check passes (e.g., `markdown-link-check`) **(REQUIRED as test substitute)**.
- Every primitive in `src/index.ts` is mentioned in the inventory section **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Markdown link-check (`markdown-link-check packages/ui/README.md`) reports zero broken links.
  - [ ] Script or lint rule verifies every export in `packages/ui/src/index.ts` appears by name in the README.
- Integration tests:
  - [ ] Snapshot test of README headers (ensures sections are not accidentally renamed).
- Test coverage target: >=80% (link-check + inventory-parity check coverage)
- All tests must pass

## Success Criteria

- All tests passing
- `packages/ui/README.md` exists with the required sections.
- Zero broken internal / external links.
- Every exported primitive is mentioned in the README inventory.
- `make verify` passes.
