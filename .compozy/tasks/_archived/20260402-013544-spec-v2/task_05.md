---
status: completed
domain: Kernel
type: Feature Implementation
scope: Full
complexity: low
dependencies:
    - task_01
    - task_02
    - task_03
---

# Task 5: Registry

> **Note:** Role catalog merge (global+workspace) is handled by Task 08.

## Overview
Implement the four in-memory registries (agent, workgroup, role catalog, driver) protected by sync.RWMutex, with SQLite persistence for agent and workgroup state, and concurrent access safety verified under the race detector.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement AgentRegistry with sync.RWMutex protecting a map[string]*AgentInfo per docs/spec-v2/02-kernel.md
- MUST implement WorkgroupRegistry with sync.RWMutex protecting a map[string]*WorkgroupInfo
- MUST implement RoleCatalog with sync.RWMutex protecting a map[string]*RoleConfig
- MUST implement DriverRegistry with sync.RWMutex protecting a map[string]AgentDriver
- MUST persist agent and workgroup state to SQLite on register/deregister/update
- MUST support lookup by ID, name, workgroup, and type
- MUST support listing agents by workgroup for scope validation
- MUST handle concurrent access from multiple goroutines safely
</requirements>

## Subtasks
- [x] 5.1 Implement AgentRegistry with CRUD operations and SQLite persistence
- [x] 5.2 Implement WorkgroupRegistry with hierarchy tracking and SQLite persistence
- [x] 5.3 Implement RoleCatalog loaded from config (roles/*.toml)
- [x] 5.4 Implement DriverRegistry initialized from config.toml [runtime.drivers.*]
- [x] 5.5 Implement lookup methods (by ID, name, workgroup, type)
- [x] 5.6 Add concurrent access tests with race detector

## Implementation Details
Refer to docs/spec-v2/02-kernel.md for registry struct definitions. Refer to docs/spec-v2/08-data-models.md for AgentInfo, WorkgroupInfo, RoleConfig types.

### Relevant Files
- `docs/spec-v2/02-kernel.md` — registry definitions
- `docs/spec-v2/08-data-models.md` — registry types
- `internal/kernel/types.go` — type definitions from task_01

### Dependent Files
- `internal/state/` — SQLite write operations for persistence
- `internal/config/` — role catalog and driver config loading

## Deliverables
- internal/registry/agents.go — agent registry
- internal/registry/workgroups.go — workgroup registry with hierarchy
- internal/registry/roles.go — role catalog
- internal/registry/drivers.go — driver registry
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Register agent, retrieve by ID — correct fields
  - [x] Deregister agent — no longer retrievable
  - [x] Update agent state — persisted correctly
  - [x] List agents by workgroup — only correct workgroup returned
  - [x] Workgroup create with parent reference — hierarchy correct
  - [x] Workgroup destroy — removed from registry
  - [x] Role catalog loads approved and draft roles with correct status
  - [x] Driver registry initializes from config
  - [x] 100 goroutines reading + 10 writing — no data race under -race
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Race detector passes with 100+ concurrent goroutines
- All 4 registries functional with correct CRUD operations
