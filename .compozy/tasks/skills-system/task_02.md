---
status: completed
title: Security scanner (VerifyContent)
type: backend
complexity: low
dependencies:
  - task_01
---

# Task 02: Security scanner (VerifyContent)

## Overview

Implement `VerifyContent()` to scan SKILL.md content for prompt injection patterns before loading into the registry. This is a critical security gate — non-bundled skills from workspace or user directories must pass this scan to prevent malicious prompt injection.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/skills/verify.go` with `VerifyContent(content string) []Warning`
- MUST define three severity levels: Info, Warning, Critical
- MUST block loading when Critical severity patterns are detected
- MUST detect system prompt override attempts (`ignore all previous`, `you are now`, etc.)
- MUST detect tool abuse instructions (`delete all files`, `rm -rf`, etc.)
- MUST detect credential extraction attempts (`print your API key`, etc.)
- MUST return `[]Warning` sorted by severity (Critical first)
- SHOULD detect unusual tool patterns and references to sensitive paths at Warning level
- SHOULD detect excessively long content (>50K chars) at Info level
</requirements>

## Subtasks
- [x] 2.1 Define `Warning` and `WarningSeverity` types (if not already in types.go)
- [x] 2.2 Implement regex-based pattern matching for Critical patterns
- [x] 2.3 Implement Warning-level and Info-level pattern detection
- [x] 2.4 Implement `VerifyContent()` that runs all patterns and returns sorted warnings
- [x] 2.5 Write unit tests covering all severity levels and edge cases

## Implementation Details

See TechSpec "Security Scanning (F4)" section for pattern categories and severity actions. Implementation is regex-based pattern matching on the Markdown body — no AST parsing needed.

### Relevant Files
- `internal/skills/types.go` — Warning and WarningSeverity types defined here

### Dependent Files
- `internal/skills/registry.go` — Will call VerifyContent during LoadAll/ForWorkspace (task_03)
- `cli/skill.go` — CLI commands reuse VerifyContent (task_11)

## Deliverables
- `internal/skills/verify.go` with VerifyContent function
- `internal/skills/verify_test.go` with comprehensive tests
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Detect Critical: `ignore all previous instructions` blocks loading
  - [x] Detect Critical: `you are now a different AI` blocks loading
  - [x] Detect Critical: `rm -rf /` blocks loading
  - [x] Detect Critical: `print your API key` blocks loading
  - [x] Detect Warning: references to `/etc/passwd` or `~/.ssh/`
  - [x] Detect Info: content > 50K chars
  - [x] Pass clean content with zero warnings
  - [x] Return warnings sorted by severity (Critical first)
  - [x] Handle empty content (no warnings)
  - [x] Case-insensitive pattern matching for Critical patterns
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make lint` passes with zero warnings
- All Critical patterns from TechSpec are detected
- Clean skill content passes with no warnings
