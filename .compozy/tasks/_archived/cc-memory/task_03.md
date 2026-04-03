---
status: completed
domain: Prompt
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_01
---

# Task 03: Prompt Assembler Memory Integration

## Overview

Extend the prompt assembler to inject persistent memory indexes (global + workspace MEMORY.md) and team memory (blackboard `type="memory"` entries) into agent system prompts. This makes agents aware of institutional knowledge and provides instructions on the memory taxonomy, staleness policy, and CLI commands for reading/writing memories.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add `MemoryContext` struct to `AssembleOptions` with `GlobalIndex`, `WorkspaceIndex`, and `TeamMemories` string fields
- MUST inject memory section after the existing Context section and before AdditionalSections
- Memory section MUST include both global and workspace MEMORY.md index content (when non-empty)
- Memory section MUST include team memory entries (when non-empty)
- Memory section MUST include instructions: 4-type taxonomy description, what NOT to save guidance, staleness policy, `agh memory write/read` CLI command usage
- MUST skip memory section entirely when all 3 MemoryContext fields are empty
- MUST NOT change the ordering of existing sections (template, specialization, skills, roles, playbooks, context)
- Existing assembler tests MUST continue to pass unchanged
</requirements>

## Subtasks

- [x] 3.1 Define `MemoryContext` struct in the prompt package
- [x] 3.2 Add `MemoryContext` field to `AssembleOptions`
- [x] 3.3 Implement memory section rendering with global index, workspace index, and team memories
- [x] 3.4 Write memory instructions template (taxonomy, what NOT to save, staleness, CLI commands)
- [x] 3.5 Integrate memory section into `Assemble()` function between context and additional sections
- [x] 3.6 Ensure memory section is omitted when `MemoryContext` is empty
- [x] 3.7 Write unit and integration tests

## Implementation Details

Modify existing files in `internal/prompt/`:

- `assembler.go` — Add `MemoryContext` struct, add field to `AssembleOptions`, render memory section in `Assemble()`
- `context.go` — Potentially add `MemoryContext` type definition here alongside existing `Context` type
- `assembler_test.go` — Add tests for memory section injection, ordering, and omission

The memory instructions template should be a const string in the prompt package describing:
- The 4 memory types and when each is appropriate
- What NOT to save (code patterns, git history, debugging solutions, config file contents, ephemeral task details)
- Staleness policy (memories > 1 day old should be verified)
- How to write: `agh memory write <filename> --type <type> --description <desc> --content <content>`
- How to read: `agh memory read <filename>`
- Team memory: `agh state append --type memory --content '---\nname: ...\n---\n...'`

### Relevant Files

- `internal/prompt/assembler.go:12-22` — `AssembleOptions` struct to modify
- `internal/prompt/assembler.go:26-61` — `Assemble()` function, section ordering
- `internal/prompt/assembler.go:119-127` — `renderAdditionalSections()` pattern for filtering empty sections
- `internal/prompt/context.go` — `Context` struct, `RenderContext()` function (pattern reference)
- `internal/prompt/assembler_test.go` — Existing tests that must continue to pass

### Dependent Files

- `internal/kernel/session_manager.go` (task_04) — Will populate `MemoryContext` when calling `Assemble()`

### Related ADRs

- [ADR-005: MEMORY.md Index Injection](../adrs/adr-005.md) — Only indexes are injected, not full content
- [ADR-003: Team Memory via Blackboard](../adrs/adr-003.md) — Team memories are blackboard entries with type="memory"

## Deliverables

- `MemoryContext` struct added to prompt package
- Memory section injection in `Assemble()` with correct ordering
- Memory instructions template covering taxonomy, exclusions, staleness, and CLI usage
- All existing assembler tests still passing
- New unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests verifying section ordering **(REQUIRED)**

## Tests

- Unit tests:
  - [x] Assemble with non-empty MemoryContext includes memory section in output
  - [x] Assemble with empty MemoryContext omits memory section entirely
  - [x] Assemble with only GlobalIndex populated includes only global section
  - [x] Assemble with only WorkspaceIndex populated includes only workspace section
  - [x] Assemble with only TeamMemories populated includes only team section
  - [x] Memory section appears after context section and before additional sections
  - [x] Memory instructions include taxonomy descriptions for all 4 types
  - [x] Memory instructions include "what NOT to save" guidance
  - [x] Memory instructions include CLI command examples
  - [x] All existing assembler tests pass without modification
- Integration tests:
  - [x] Full Assemble() with all sections populated (template + role + skills + roles catalog + playbooks + context + memory + additional) produces correctly ordered output
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria

- All tests passing (including all pre-existing assembler tests)
- Test coverage >=80%
- `make verify` passes
- Memory section correctly ordered in prompt output
- Empty `MemoryContext` produces zero-change behavior (backward compatible)
