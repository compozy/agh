---
status: pending
title: PromptProvider interface and memory assembler refactor
type: refactor
complexity: medium
dependencies: []
---

# Task 07: PromptProvider interface and memory assembler refactor

## Overview

Define the `PromptProvider` interface in the session package and refactor `memory.Assembler` to implement it. This decouples memory from prompt assembly orchestration, enabling the composed assembler pipeline where memory and skills are independent providers.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `session/prompt_provider.go` with `PromptProvider` interface
- MUST define `PromptSection(ctx context.Context, workspace string) (string, error)` as the single method
- MUST add `PromptSection()` method to `memory.Assembler` that returns ONLY the memory context block (no base prompt)
- MUST preserve existing `Assemble()` method on `memory.Assembler` for backward compatibility during transition
- MUST ensure memory ordering is preserved: `PromptSection()` returns the same memory block currently prepended before the base prompt
- MUST add compile-time interface verification: `var _ session.PromptProvider = (*Assembler)(nil)`
</requirements>

## Subtasks
- [ ] 7.1 Create `session/prompt_provider.go` with PromptProvider interface
- [ ] 7.2 Add `PromptSection()` method to `memory.Assembler` returning memory-only content
- [ ] 7.3 Add compile-time interface check
- [ ] 7.4 Write unit tests verifying PromptSection returns correct content
- [ ] 7.5 Write regression test: Assemble() output unchanged

## Implementation Details

See TechSpec "Prompt Assembly Pipeline" section. The key insight is that the current `Assemble()` method at `memory/assembler.go:42` does `contextBlock + "\n\n" + basePrompt`. The new `PromptSection()` returns only `contextBlock`.

### Relevant Files
- `internal/session/interfaces.go` — Existing PromptAssembler interface (line 162)
- `internal/memory/assembler.go` — Current assembler to refactor

### Dependent Files
- `daemon/composed_assembler.go` — Will consume PromptProvider interface (task_09)
- `internal/skills/catalog.go` — CatalogProvider will implement PromptProvider (task_04)

### Related ADRs
- [ADR-003: Composed PromptAssembler with PromptProvider Interface](../adrs/adr-003.md) — Design rationale for the pipeline refactor

## Deliverables
- `internal/session/prompt_provider.go` with PromptProvider interface
- Modified `internal/memory/assembler.go` with PromptSection method
- `internal/memory/assembler_test.go` updated with new tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] PromptSection returns memory context block for global + workspace indexes
  - [ ] PromptSection returns empty string when no memory indexes exist
  - [ ] PromptSection respects context cancellation
  - [ ] Regression: Assemble() produces byte-identical output to before refactor
  - [ ] PromptSection does NOT include the base agent prompt
  - [ ] Compile-time interface check passes
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- Existing Assemble() behavior unchanged (regression test)
- memory.Assembler satisfies session.PromptProvider at compile time
