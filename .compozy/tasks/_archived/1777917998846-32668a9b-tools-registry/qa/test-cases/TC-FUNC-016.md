# TC-FUNC-016 — Canonical ID collision keeps both tools operator-visible and session-hidden

- **Priority:** P0
- **Type:** Functional / collision
- **Trace:** Task 03, ADR-007, Safety Invariant 7

## Objective

Prove that registering two providers with the same canonical `ToolID` fails closed. Operator surfaces show both as `conflicted` with provenance; session projection hides both.

## Test Steps

1. Provider A registers `ext__demo__report` from extension `demo_a`.
2. Provider B registers same canonical `ext__demo__report` from extension `demo_a` (or different extension that sanitizes to same id).
3. `GET /api/tools` shows both with reason `conflicted_id` and full `SourceRef` for each.
4. Session projection contains neither.
5. Invoking the conflicted ID returns `tool_conflict` (HTTP 409).

## Automation

- **Target:** Unit + Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestRegistryCollision`
