---
status: completed
domain: CLI
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_12
    - task_17
---

# Task 19: CLI Commands — Workgroups & Discovery

## Overview
Implement the Cobra CLI subcommands for workgroup management (workgroup create, list, destroy, topology) and discovery operations (roles list/get/create/approve, playbooks list/get/save/approve).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement agh workgroup create with --name and --parent flags
- MUST implement agh workgroup list with TOON output
- MUST implement agh workgroup destroy with workgroup argument
- MUST implement agh topology with hierarchical tree TOON output
- MUST implement agh roles list/get/create/approve per docs/spec-v2/06-cli.md
- MUST implement agh playbooks list/get/save/approve per docs/spec-v2/06-cli.md
- MUST render all output in TOON format
- MUST validate role type on create (master/worker/advisor/reviewer/researcher)
</requirements>

## Subtasks
- [x] 19.1 Implement workgroup commands: create, list, destroy
- [x] 19.2 Implement topology command with hierarchical tree rendering
- [x] 19.3 Implement roles commands: list, get, create, approve
- [x] 19.4 Implement playbooks commands: list, get, save, approve
- [x] 19.5 Implement TOON output for all commands

## Implementation Details
Refer to docs/spec-v2/06-cli.md for all command specs. Refer to docs/spec-v2/10-meta-learning.md for roles/playbooks create/approve behavior.

### Relevant Files
- `docs/spec-v2/06-cli.md` — CLI reference
- `docs/spec-v2/10-meta-learning.md` — meta-learning commands

### Dependent Files
- `internal/cli/root.go` — HTTP-over-UDS connection helper (~/.agh/daemon.sock)
- `internal/toon/renderer.go` — TOON rendering
- `internal/config/roles.go` — role catalog management
- `internal/config/playbooks.go` — playbook management

## Deliverables
- internal/cli/workgroup.go — workgroup create, list, destroy, topology
- internal/cli/roles.go — roles list, get, create, approve
- internal/cli/playbooks.go — playbooks list, get, save, approve
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Workgroup create builds correct request with name and parent
  - [x] Workgroup list renders TOON with id, name, parent, master, state
  - [x] Workgroup destroy sends correct destroy request
  - [x] Topology renders hierarchical tree matching spec example
  - [x] Roles list shows both approved and draft with status
  - [x] Roles create validates type is one of 5 valid types
  - [x] Roles approve renames .draft.toml to .toml
  - [x] Playbooks list shows both approved and draft
  - [x] Playbooks approve renames .draft.md to .md
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- All commands match docs/spec-v2/06-cli.md spec
