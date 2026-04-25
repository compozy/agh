# BUG-002: Fresh daemon memory history schema misses scope columns

## Status

Fixed in Task 11.

## Severity

P0 runtime/API regression. A fresh daemon database could not persist memory operation history for `agh memory write`, breaking `agh memory history` and `GET /api/memory/history`.

## Reproduction

1. Start a daemon with a fresh isolated `AGH_HOME`.
2. Run `agh memory write global-note.md --scope global --type user --description "..." --content "..." -o json`.

## Observed

The write failed with:

```text
memory: open catalog database ".../agh.db": store: initialize sqlite database ".../agh.db": SQL logic error: no such column: scope (1)
```

## Expected

Fresh global databases include the memory operation history columns required by the memory catalog: `scope`, `workspace_root`, and `filename`.

## Root Cause

The global DB base schema created the legacy `memory_operation_log` table with only `id`, `type`, `agent_name`, `summary`, and `timestamp`. The memory catalog expected the newer scoped history shape, but no global schema migration added those columns.

## Fix

Added global schema migration version 6, `add_memory_operation_scope`, to add the missing history columns and indexes. Added schema assertions in `internal/store/globaldb/global_db_test.go`.

## Verification Evidence

- Failure captured during TC-FUNC-001 live memory execution.
- Regression tests:
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/regression-globaldb-memory-operation-scope.log`
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/regression-memory-history-schema.log`
- Build after fix: `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/make-build-after-memory-schema-fix.log`
- Post-fix live CLI/API evidence:
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/memory-write-global.json`
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/memory-write-workspace.json`
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/memory-health-cli.json`
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/memory-history-cli.json`
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/memory-health-api.json`
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/memory-history-api.json`
  - `.compozy/tasks/hermes/qa/logs/TC-FUNC-001/memory-redaction-check.log`
