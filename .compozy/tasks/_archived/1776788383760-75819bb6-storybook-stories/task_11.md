---
status: completed
title: Update SKILL.md and final verification
type: chore
complexity: low
dependencies:
  - task_02
  - task_03
  - task_04
  - task_06
  - task_07
  - task_08
  - task_09
  - task_10
---

# Task 11: Update SKILL.md and final verification

## Overview
Reconcile `.claude/skills/storybook-stories/SKILL.md` with the decisions made in this rollout (replace `@compozy/ui` with `@agh/ui`, document the autodocs-opt-in policy, and link the four ADRs) and run the final verification gate that merges the PR: `make web-lint`, `make web-typecheck`, and both Storybook builds. This task is the single source of truth that the rollout is landable.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST replace every `@compozy/ui` reference in `.claude/skills/storybook-stories/SKILL.md` with `@agh/ui` and update the example imports accordingly.
- MUST document in SKILL.md that `tags: ["autodocs"]` applies only to `packages/ui` primitives (pointer to ADR-003).
- MUST add a "References" section to SKILL.md linking `_techspec.md` and the four ADRs.
- MUST run and report green results for: `make web-lint`, `make web-typecheck`, `bun run --cwd web build-storybook`, `bun run --cwd packages/ui build-storybook`.
- MUST run `@storybook/addon-a11y` pass over a sampled set of stories (at minimum: one primitive, one web/ui overlay, one session-core, one system-panel) and fix any critical violations surfaced.
- MUST NOT edit the SKILL.md to alter unrelated guidance (e.g., max-stories-per-component rules).
</requirements>

## Subtasks
- [x] 11.1 Rewrite `@compozy/ui` → `@agh/ui` throughout SKILL.md, refreshing the example snippets.
- [x] 11.2 Add autodocs policy paragraph and cross-links to `_techspec.md` and ADR-001..ADR-004.
- [x] 11.3 Run `make web-lint` and `make web-typecheck` from the repo root; record outcomes.
- [x] 11.4 Run `bun run --cwd web build-storybook` and `bun run --cwd packages/ui build-storybook`; record outcomes.
- [x] 11.5 Perform a sampled a11y pass and fix any critical violations before opening the PR.

## Implementation Details
The SKILL.md lives at `.claude/skills/storybook-stories/SKILL.md`. Keep the existing structure; modify wording and the example code blocks. The PR description should reference `.compozy/tasks/storybook-stories/_techspec.md` and list the ADRs.

### Relevant Files
- `.claude/skills/storybook-stories/SKILL.md` — primary edit target.
- `.compozy/tasks/storybook-stories/_techspec.md` — authoritative source for policies.
- `.compozy/tasks/storybook-stories/adrs/adr-00{1..4}.md` — decisions to cite.
- `Makefile` — `web-lint`, `web-typecheck` targets.
- `web/package.json`, `packages/ui/package.json` — `build-storybook` scripts.

### Dependent Files
- None downstream; this is the final task.

### Related ADRs
- [ADR-001: Dual Storybook Topology](adrs/adr-001.md) — cited in SKILL.md update.
- [ADR-002: MSW + Shared Decorators for System Stories](adrs/adr-002.md) — cited in SKILL.md update.
- [ADR-003: stories/ Subfolder Placement, Opt-in Autodocs](adrs/adr-003.md) — cited in SKILL.md update.
- [ADR-004: Per-System Mocks Directory](adrs/adr-004.md) — cited in SKILL.md update.

## Deliverables
- Updated `.claude/skills/storybook-stories/SKILL.md`.
- Recorded green runs of lint, typecheck, and both Storybook builds.
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for final verification **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] SKILL.md contains zero occurrences of `@compozy/ui` after edits.
  - [ ] SKILL.md contains exactly one `tags: ["autodocs"]` guidance block that references ADR-003.
  - [ ] SKILL.md includes links to all four ADRs under a "References" section.
- Integration tests:
  - [ ] `make web-lint` exits 0.
  - [ ] `make web-typecheck` exits 0.
  - [ ] `bun run --cwd web build-storybook` exits 0 and reports an index size equal to the sum of authored stories from tasks 03–10 plus pre-existing `design-system` stories.
  - [ ] `bun run --cwd packages/ui build-storybook` exits 0 and indexes exactly 12 story modules (task_02 output).
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- SKILL.md reconciled with this rollout's decisions.
- Both Storybook instances build green on CI.
- A sampled a11y pass surfaces no critical violations.
