# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Run the full improvements pass for `internal/api/`, with inventories first, benchmarks before perf claims, scoped fixes under `internal/api/`, a complete report at `.compozy/tasks/improvs/reports/api.md`, and clean `make verify`.

## Important Decisions
- Treat the package as the full `internal/api/*` surface; source edits remain inside `internal/api/`, while report/tracking/memory artifacts live under the task directory per spec.
- Use production-code inventories for goroutines/channels/mutexes/selects and call out runtime-facing surfaces explicitly; test-only concurrency scaffolding is supporting evidence, not the primary runtime audit.
- Optimization hot-path candidates are:
  - `core.WriteSSE`
  - `core.EmitObserveEvents`
  - `core.SessionPayloadsFromInfos`
  - `core.AgentEventPayloadFromEvent`

## Learnings
- Main package coverage baseline is already healthy: `internal/api/core` 80.0%, `httpapi` 83.3%, `udsapi` 84.0%, `contract` 91.7%, `spec` 91.2%.
- `internal/memory.Store` rejects path separators in filenames, so memory filename path traversal is blocked downstream; workspace roots are still a reviewed input boundary, but the filename sink is sanitized.
- Current `gocyclo` top production hotspots include `resolveMemoryLocation`, `writeSSERaw`, `ParseResourceFilter`, `StreamBridgeHealth`, and network-channel handlers.
- `dupl` reports notable production duplication in `internal/api/spec/spec.go`, `internal/api/core/tasks.go`, and `internal/api/core/automation.go`.
- The measured optimization win in this task is `core.AgentEventPayloadFromEvent`: after switching raw payload handling to `payloadJSONBytes`, the benchmark improved from `215.2 ns/op, 256 B/op, 3 allocs/op` to `203.3 ns/op, 192 B/op, 2 allocs/op`.

## Files / Surfaces
- `internal/api/core/conversions.go`
- `internal/api/core/sse.go`
- `internal/api/core/memory.go`
- `internal/api/core/resources.go`
- `internal/api/core/tasks.go`
- `internal/api/core/network_details.go`
- `internal/api/httpapi/prompt.go`
- `internal/api/httpapi/server.go`
- `internal/api/httpapi/middleware.go`
- `internal/api/udsapi/server.go`
- `internal/api/udsapi/prompt.go`

## Errors / Corrections
- UBS invocation is still unresolved because no callable skill runner/tool has been exposed yet; if that remains true, report it as `not-run` with the literal tooling limitation instead of substituting a manual review.
- `.compozy/tasks/improvs/reports/api.md` now exists with the mandatory inventories and findings; the remaining completion gate is fresh `make verify` plus task tracking updates.

## Ready for Next Run
- Next concrete step is to run `make verify`, capture the final excerpt in the report, then update `task_02.md` and `_tasks.md` only after the gate is clean.
