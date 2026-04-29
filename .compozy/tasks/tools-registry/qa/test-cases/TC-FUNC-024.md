# TC-FUNC-024 — Bounded task tools cover only ADR-004 set

- **Priority:** P1
- **Type:** Functional / native task tools
- **Trace:** Task 05, ADR-004, Safety Invariant 18

## Objective

Prove only the bounded MVP task tools are registered: `agh__task_list`, `agh__task_read`, `agh__task_create`, `agh__task_child_create`, `agh__task_update`, `agh__task_cancel`, `agh__task_run_list`. Excluded tools (`claim`, `release`, `complete`, `fail`, `run_start`, `run_complete`, `run_cancel`) MUST NOT be registered.

## Test Steps

1. `agh tool list -o json` filtered by source `builtin` and `agh__task_*`.
   - **Expected:** Exactly the 7 listed tools, no more, no less.
2. Attempt to invoke `agh__task_claim`.
   - **Expected:** `tool_not_found`.
3. `agh__task_create` succeeds via `task.Service.CreateTask`; `agh__task_cancel` succeeds via `task.Service.CancelTask`; `agh__task_child_create` enforces lineage subset via `task.Service.CreateChildTask`.

## Automation

- **Target:** Integration
- **Status:** Existing
- **Command/Spec:** `go test ./internal/tools ./internal/task -run TestNativeTaskTools`
