---
status: completed
title: TOON Renderer
type: ""
complexity: low
dependencies:
    - task_03
---

# Task 10: TOON Renderer

## Overview
Implement the TOON (Token-Oriented Object Notation) renderer that converts SQLite query results into the compact, LLM-optimized TOON format used by all CLI responses. This includes renderers for agents, workgroups, blackboard, status, events, and topology views.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST use toon-format/toon-go library for TOON encoding per docs/spec-v2/00-executive-summary.md
- MUST implement renderers for all CLI output views per docs/spec-v2/08-data-models.md TOON section
- MUST render single objects: type{field1,field2,...}: value1,value2,...
- MUST render arrays: type[count]{field1,field2,...}: followed by rows
- MUST handle values containing commas or newlines (quoted)
- MUST render: agents view (agh ps), workgroups view (agh workgroup list), blackboard view (agh state read), status view (agh status), context view (agh context), topology view (agh topology)
- MUST produce output matching examples in docs/spec-v2/06-cli.md and docs/spec-v2/08-data-models.md
</requirements>

## Subtasks
- [x] 10.1 Implement base TOON rendering functions (single object, array, headers)
- [x] 10.2 Implement agents renderer for agh ps output
- [x] 10.3 Implement workgroups renderer for agh workgroup list output
- [x] 10.4 Implement blackboard/status/events renderers for state commands
- [x] 10.5 Implement context renderer (aggregated view) for agh context
- [x] 10.6 Implement topology renderer (tree view) for agh topology

## Implementation Details
Refer to docs/spec-v2/08-data-models.md for TOON output format examples. Refer to docs/spec-v2/06-cli.md for expected CLI output.

### Relevant Files
- `docs/spec-v2/08-data-models.md` — TOON format examples
- `docs/spec-v2/06-cli.md` — CLI output examples per command

### Dependent Files
- `internal/state/queries.go` — read helpers that provide data to render

## Deliverables
- internal/toon/renderer.go — TOON rendering functions for all views
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Single object renders as type{fields}: values
  - [x] Array renders as type[count]{fields}: with correct row count
  - [x] Values with commas are quoted correctly
  - [x] Agents view matches agh ps example from spec
  - [x] Workgroups view matches agh workgroup list example
  - [x] Blackboard view matches agh state read example
  - [x] Context view aggregates workgroup + agents + blackboard
  - [x] Topology view renders hierarchical tree correctly
  - [x] Empty result sets render cleanly (e.g., "agents[0]{...}: (empty)")
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Output matches TOON examples from docs/spec-v2/08-data-models.md
