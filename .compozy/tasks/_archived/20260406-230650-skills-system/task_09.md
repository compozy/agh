---
status: completed
title: ComposedAssembler
type: backend
complexity: medium
dependencies:
  - task_07
---

# Task 09: ComposedAssembler

## Overview

Implement the `ComposedAssembler` in the daemon package that chains multiple `PromptProvider` instances into a single `session.PromptAssembler`. This is the composition logic that wires memory and skills prompt sections together with the base agent prompt, preserving the current prompt ordering.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `daemon/composed_assembler.go` with `ComposedAssembler` struct
- MUST implement `session.PromptAssembler` interface
- MUST accept variadic `session.PromptProvider` instances
- MUST preserve prompt ordering: prepend providers (memory) → agent prompt → append providers (skills)
- MUST handle zero providers gracefully (return base agent prompt only)
- MUST handle nil providers (skip them)
- MUST handle provider errors (return error, do not silently drop sections)
- MUST pass workspace parameter to each provider's `PromptSection()`
- MUST add compile-time interface check: `var _ session.PromptAssembler = (*ComposedAssembler)(nil)`
- Regression: with only a memory provider, output MUST be byte-identical to current `memory.Assembler.Assemble()`
</requirements>

## Subtasks
- [x] 9.1 Implement `ComposedAssembler` struct with prepend/append provider slots
- [x] 9.2 Implement `Assemble()` method with correct ordering
- [x] 9.3 Implement `NewComposedAssembler()` constructor
- [x] 9.4 Add compile-time interface check
- [x] 9.5 Write unit tests including regression test for backward compatibility

## Implementation Details

See TechSpec "Prompt Assembly Pipeline" section. The ordering is:
1. Prepend providers (memory context) — placed before agent prompt
2. Agent prompt (base system prompt from AgentDef)
3. Append providers (skill catalog) — placed after agent prompt

This matches the current behavior where memory is prepended (`contextBlock + "\n\n" + basePrompt`).

### Relevant Files
- `internal/session/interfaces.go` — PromptAssembler interface (line 162)
- `internal/session/prompt_provider.go` — PromptProvider interface (task_07)
- `internal/memory/assembler.go` — Current assembler for regression comparison

### Dependent Files
- `daemon/daemon.go` — Will construct ComposedAssembler at boot (task_10)

### Related ADRs
- [ADR-003: Composed PromptAssembler with PromptProvider Interface](../adrs/adr-003.md) — Design rationale

## Deliverables
- `daemon/composed_assembler.go` with ComposedAssembler implementation
- `daemon/composed_assembler_test.go` with comprehensive tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Zero providers: returns base agent prompt only
  - [x] One prepend provider (memory): output matches current assembler behavior
  - [x] One append provider (skills): agent prompt + skills section
  - [x] Both prepend and append: memory + agent prompt + skills
  - [x] Nil provider in chain: skipped gracefully
  - [x] Provider returns error: Assemble returns error
  - [x] Provider returns empty string: no extra whitespace in output
  - [x] Workspace parameter passed correctly to all providers
  - [x] Regression: memory-only output byte-identical to current Assemble()
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- Backward compatibility with memory-only assembler verified
- Compile-time interface check passes
