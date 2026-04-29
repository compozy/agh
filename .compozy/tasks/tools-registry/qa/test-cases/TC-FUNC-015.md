# TC-FUNC-015 — `dynamic` source kind rejected in MVP

- **Priority:** P2
- **Type:** Functional / source validation
- **Trace:** Task 02, TechSpec Data Models, Validation

## Objective

Prove any descriptor or resource publication attempt that uses `source.kind = dynamic` is rejected because MVP has no dynamic producer.

## Test Steps

1. Test provider attempts to register `Descriptor{ source.kind = "dynamic" }`.
   - **Expected:** Rejected.
2. Resource manifest declares `source.kind = "dynamic"`.
   - **Expected:** Resource validation fails at load.
3. Confirm operator surfaces never list a `dynamic` source kind in MVP.

## Automation

- **Target:** Unit
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools -run TestDynamicSourceRejected`
