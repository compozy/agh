# TC-FUNC-009 — Toolset cycle/unknown-member rejection

- **Priority:** P2
- **Type:** Functional / toolsets
- **Trace:** Task 03, TechSpec Skills/Tools/Resources, Safety Invariant 15

## Objective

Prove toolset expansion rejects cycles and unknown members deterministically and that expansion runs once before session projection (not lazily during dispatch).

## Test Steps

1. Define toolset `A` referencing `B`; `B` referencing `A`.
   - **Expected:** Config validation rejects with cycle error.
2. Define toolset referencing unknown `ToolID`.
   - **Expected:** Reject at workspace resolution (when concrete validation runs).
3. Define toolset referencing valid concrete IDs and another known toolset.
   - **Expected:** Expansion produces deterministic concrete `ToolID` set.
4. Verify session lineage stores concrete `ToolID`s, not unresolved patterns or toolset references.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestToolsetExpansion`
