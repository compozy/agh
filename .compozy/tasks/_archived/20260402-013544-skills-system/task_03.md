---
status: completed
title: Kernel Integration
type: ""
complexity: medium
dependencies:
    - task_02
---

# Task 3: Kernel Integration

## Overview

Integrate the skills system into the AGH kernel boot sequence and agent spawn pipeline. Add skills loading as a new boot step, build skill snapshots during agent spawn, and inject the XML catalog as a new layer in the prompt assembly system. This wires the skills package into the kernel without modifying any driver code.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add skillRegistry field to Kernel struct
- MUST add a new boot step (load_skills) between init_drivers and load_prompts
- MUST call skills.NewRegistry() → LoadAll() → Freeze() during boot
- MUST build SkillSnapshot in buildBootstrapPrompt() before prompt assembly
- MUST add skills catalog as 4th layer in prompt.Assemble() (between role and context)
- MUST NOT modify any driver code
- MUST pass LoadConfig with correct directory paths (workspace from session, user from home dir)
</requirements>

## Subtasks
- [x] 3.1 Add `skillRegistry *skills.Registry` field to the Kernel struct in `internal/kernel/kernel.go`
- [x] 3.2 Add `bootStepLoadSkills` constant and implement the boot step (NewRegistry → LoadAll → Freeze)
- [x] 3.3 Modify `prompt.AssembleOptions` to accept a skills catalog string
- [x] 3.4 Modify `prompt.Assemble()` to insert skills catalog between role specialization and context
- [x] 3.5 Modify `buildBootstrapPrompt()` in session_manager.go to build SkillSnapshot and pass catalog to prompt assembly

## Implementation Details

### Relevant Files
- `internal/kernel/kernel.go` — Modify: add skillRegistry field, add boot step (insert after line ~335, before line ~337)
- `internal/kernel/session_manager.go` — Modify: buildBootstrapPrompt() (lines 426-458) to include skills catalog
- `internal/prompt/assembler.go` — Modify: AssembleOptions struct and Assemble() function (lines 23-49) to accept/insert skills catalog
- `internal/prompt/context.go` — Reference: RenderContext() pattern for formatting sections
- `internal/config/home.go` — Reference: home directory resolution for user-level skill paths

### Dependent Files
- `internal/prompt/assembler_test.go` — Update tests to verify skills layer insertion
- `internal/kernel/session_manager_test.go` — Update tests to verify skill snapshot in prompt

### Related ADRs
- [ADR-003: System Prompt + CLI Access](adrs/adr-003.md) — Defines how skills integrate via system prompt injection
- [ADR-004: Four-Level Loading Hierarchy](adrs/adr-004.md) — Defines the directory paths for LoadConfig

## Deliverables
- Modified kernel with skills loading boot step
- Modified prompt assembly with 4th layer (skills catalog)
- Modified session manager with skill snapshot generation
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for boot and spawn flow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Prompt assembly includes skills catalog between role and context
  - [x] Prompt assembly omits skills section when snapshot is nil
  - [x] Prompt assembly omits skills section when snapshot has no skills
  - [x] Skills catalog appears in correct position within assembled prompt
- Integration tests:
  - [x] Kernel boot loads skills registry and freezes it
  - [x] Agent spawn produces system prompt containing `<available_skills>` XML
  - [x] System prompt includes behavioral instructions for `agh skill view`
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Kernel boot sequence includes skills loading without errors
- Agent system prompts contain the skills catalog XML
- No driver code modified
- `make verify` passes (fmt + lint + test + build)
