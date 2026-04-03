---
status: completed
domain: Config
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_02
    - task_19
---

# Task 21: Meta-Learning

## Overview
Implement the meta-learning system that allows the supervisor to dynamically create roles and save playbooks during sessions, with a draft-to-approve workflow that requires human validation before artifacts become defaults in future sessions.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST gate all meta-learning operations behind [meta].enabled config per docs/spec-v2/10-meta-learning.md
- MUST implement dynamic role creation: validate type/driver, write draft role file
- MUST implement dynamic playbook saving: write draft playbook file
- MUST save draft roles to workspace .agh/roles/ if session has a workspace, else global ~/.agh/roles/
- MUST save draft playbooks to workspace .agh/playbooks/ if session has a workspace, else global ~/.agh/playbooks/
- MUST implement approval: rename .draft.toml → .toml and .draft.md → .md
- MUST support auto_approve mode (skip draft suffix when enabled)
- MUST reject creation commands when meta.enabled = false
- MUST make draft roles usable in current session but not in future sessions until approved
- MUST validate driver exists in config before accepting role creation
</requirements>

## Subtasks
- [x] 21.1 Implement meta-learning gate (reject when disabled)
- [x] 21.2 Implement dynamic role creation with validation and draft file writing
- [x] 21.3 Implement dynamic playbook saving with draft file writing
- [x] 21.4 Implement approval flow (rename draft to final)
- [x] 21.5 Implement auto_approve mode
- [x] 21.6 Ensure draft roles are loadable in current session

## Implementation Details
Refer to docs/spec-v2/10-meta-learning.md for the complete meta-learning spec and cross-session learning flow.

### Relevant Files
- `docs/spec-v2/10-meta-learning.md` — complete meta-learning spec
- `docs/spec-v2/07-configuration.md` — [meta] config section

### Dependent Files
- `internal/config/` — config loading, role catalog
- `internal/cli/roles.go` — roles create/approve CLI commands
- `internal/cli/playbooks.go` — playbooks save/approve CLI commands

## Deliverables
- Meta-learning logic integrated into config/roles/playbooks packages
- Draft-to-approve workflow
- auto_approve mode support
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Role creation writes .draft.toml to workspace .agh/roles/ when workspace available
  - [x] Role creation writes .draft.toml to global ~/.agh/roles/ when no workspace
  - [x] Role creation rejects invalid type
  - [x] Role creation rejects non-existent driver
  - [x] Role creation rejected when meta.enabled = false
  - [x] Playbook save writes .draft.md to workspace .agh/playbooks/ when workspace available
  - [x] Playbook save writes .draft.md to global ~/.agh/playbooks/ when no workspace
  - [x] Playbook save rejected when meta.enabled = false
  - [x] Approval renames .draft.toml → .toml
  - [x] Approval renames .draft.md → .md
  - [x] Approval of non-draft returns error "already approved"
  - [x] Approval of non-existent role returns error "not found"
  - [x] Auto-approve mode writes .toml directly (no .draft suffix)
  - [x] Draft roles appear in roles list with "draft" status
  - [x] Draft roles are usable for spawning in current session
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Draft-approve workflow matches docs/spec-v2/10-meta-learning.md
