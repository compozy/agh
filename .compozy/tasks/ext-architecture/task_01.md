---
status: pending
title: Minimal Tool struct and ToolProvider interface
type: backend
complexity: low
dependencies: []
---

# Task 01: Minimal Tool struct and ToolProvider interface

## Overview

Create the foundational `Tool` struct and `ToolProvider` interface in a new `internal/tools/` package. This grounds the existing hook tool dispatch (`tool.pre_call`, `tool.post_call`, `tool.post_error`) which already operates against tool semantics (ToolName, ToolNamespace matchers) that have no corresponding data type. Extensions will later implement `ToolProvider` to register tools with AGH.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/tools/` package with `Tool` struct containing `Name`, `Description`, `InputSchema json.RawMessage`, `ReadOnly bool`, and `Source ToolSource`
- MUST define `ToolSource` enum with values: `ToolSourceBuiltin`, `ToolSourceMCP`, `ToolSourceExtension`, `ToolSourceDynamic`
- MUST define `ToolProvider` interface with `Tools(ctx context.Context) ([]Tool, error)` method
- MUST include compile-time interface verification
- MUST ensure `Tool` JSON serialization matches the hook payload field names for `ToolCallRef` in `internal/hooks/`
</requirements>

## Subtasks
- [ ] 1.1 Create `internal/tools/` package with `Tool` struct and `ToolSource` enum
- [ ] 1.2 Define `ToolProvider` interface
- [ ] 1.3 Verify JSON serialization compatibility with existing hook `ToolCallRef` payload fields
- [ ] 1.4 Write unit tests for Tool serialization and ToolSource enum

## Implementation Details

New package `internal/tools/` with two files: `tool.go` (types) and `tool_test.go` (tests).

See TechSpec "Core Interfaces" section for the `Tool` struct and `ToolProvider` interface definitions.

### Relevant Files
- `internal/hooks/payloads.go` — Contains `ToolPreCallPayload`, `ToolPostCallPayload` with tool-related fields that `Tool` must be compatible with
- `internal/hooks/types.go` — `HookMatcher` has `ToolName`, `ToolNamespace`, `ToolReadOnly` fields that reference tool semantics
- `internal/hooks/dispatch.go` — Tool dispatch methods that will eventually consume `Tool` types

### Dependent Files
- `internal/extension/manager.go` — Will use `ToolProvider` to collect tools from extensions (future task)

### Related ADRs
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Tools are part of the "capabilities" dimension

## Deliverables
- New `internal/tools/tool.go` with `Tool` struct, `ToolSource` enum, `ToolProvider` interface
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] `Tool` struct JSON marshaling produces expected field names matching hook payloads
  - [ ] `Tool` struct JSON unmarshaling from hook-compatible JSON succeeds
  - [ ] `ToolSource` string values are correct (`builtin`, `mcp`, `extension`, `dynamic`)
  - [ ] `ToolSource` validation rejects unknown values
  - [ ] Compile-time interface verification for `ToolProvider`
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `internal/tools/` package exists and compiles
- `make verify` passes
