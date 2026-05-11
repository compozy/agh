# Task File Template

Use this structure for every individual task file. The file must start with YAML frontmatter containing the parseable metadata.

```markdown
---
status: pending
title: [Task title]
type: [one of frontend, backend, docs, test, infra, refactor, chore, bugfix, or a project-specific [tasks].types override]
complexity: [low, medium, high, critical]
dependencies:
  - task_01
  - task_02
---

# Task N: [Title]

## Overview
[2-3 sentences: what the task accomplishes and why it matters in the context of the project.]

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TEST DECISION REQUIRED — every task MUST identify the invariant, owning layer, canonical suite, and verification command; new tests are added only when justified
</critical>

<requirements>
- [Requirement 1 — specific technical requirement]
- [Requirement 2 — e.g., "MUST authenticate users via JWT tokens"]
- [Requirement 3]
</requirements>

## Subtasks
- [ ] N.1 [Subtask description — WHAT to accomplish]
- [ ] N.2 [Subtask description]
- [ ] N.3 [Subtask description]

## Implementation Details
[File paths to create or modify, integration points, and dependencies.
Reference the TechSpec implementation section for code patterns and interface designs.]

### Relevant Files
- `path/to/file` — [brief reason this file is relevant]

### Dependent Files
- `path/to/dependency` — [brief reason this file is affected]

### Related ADRs
- [ADR-NNN: Title](../adrs/adr-NNN.md) — Relevance to this task

## Deliverables
- [Concrete output 1]
- [Concrete output 2]
- Test placement decision **(REQUIRED)**: invariant, owning layer, canonical suite, and verification command
- Updates to the canonical test suite when the invariant requires automated coverage
- No-new-test rationale when an existing suite/gate already proves the invariant

## Tests
- Invariant: [Rule that must stay true]
- Owning layer: [unit | integration | end-to-end | static analysis | codegen | visual QA | documentation build | manual QA evidence]
- Canonical suite: [Existing suite to update, or "none - new suite justified because ..."]
- Test cases, when justified:
  - [ ] [Specific behavior, input/condition, and expected evidence]
  - [ ] [Specific failure mode, if relevant]
- No-new-test rationale, when applicable: [Existing suite/gate/evidence that already owns the invariant]
- Verification command: [Narrow command plus required repo gate]

## Success Criteria
- All tests passing
- Test coverage >=80%
- [Measurable outcome 1]
- [Measurable outcome 2]
```

## Guidelines

- Every task must be independently implementable when its dependencies are met.
- Every task MUST include a Tests section and a test placement decision in Deliverables.
- Do not create tests by checklist. Add or update tests only when the invariant, owning layer, and canonical suite justify them.
- Never create separate tasks dedicated solely to testing.
- Subtasks describe WHAT needs to happen, not HOW to implement it.
- Minimize code in tasks. Show code only to illustrate current structure or problem areas.
- Implementation details should reference the TechSpec for patterns rather than duplicating them.
