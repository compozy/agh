# TC-FUNC-005 — `ToolsetID` shares the `ToolID` grammar

- **Priority:** P2
- **Type:** Functional / grammar
- **Trace:** Task 01, Task 02, TechSpec Data Models

## Objective

Prove `ToolsetID` validation uses the same grammar as `ToolID`: lowercase ASCII segments separated by `__`, ≤ 64 chars, no dots/hyphens/uppercase/empty segments.

## Test Steps

1. Accept: `agh__bootstrap`, `agh__catalog`, `agh__coordination`, `agh__tasks`, `linear__read`.
2. Reject: dotted, hyphenated, uppercase, > 64 chars, `__leading`, trailing `__`, empty.
3. Confirm `agent.toolsets` config field rejects invalid `ToolsetID`s and accepts a recursive set defined elsewhere.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestToolsetIDValidation`
