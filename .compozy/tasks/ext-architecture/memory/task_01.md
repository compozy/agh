# Task Memory: task_01.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Create `internal/tools/tool.go` and `internal/tools/tool_test.go` for the minimal tool foundation in ext-architecture task_01.
- Deliver `Tool`, `ToolSource`, `ToolProvider`, hook-compatible JSON coverage, and clean verification before any tracking update or commit.

## Important Decisions
- Use the `_techspec.md` / `_protocol.md` tool JSON shape as canonical (`name`, `input_schema`, `read_only`, `source`).
- Handle hook-compatibility without importing `internal/hooks` into the new package.
- Accept `tool_name` during JSON decode so hook-compatible payloads round-trip into `Tool` without changing the canonical wire shape.

## Learnings
- `internal/hooks.ToolCallRef` uses `tool_name`, `tool_namespace`, and `read_only`; the new `Tool` type only overlaps partially with that payload.
- `internal/tools/` does not exist in the baseline workspace.
- `encoding/json` already honors the enum text marshaler pattern used in `internal/hooks`, so `ToolSource` can follow the same int-enum implementation style.

## Files / Surfaces
- `.compozy/tasks/ext-architecture/task_01.md`
- `.compozy/tasks/ext-architecture/_tasks.md`
- `.compozy/tasks/ext-architecture/_techspec.md`
- `.compozy/tasks/ext-architecture/adrs/adr-005.md`
- `.compozy/tasks/ext-architecture/memory/MEMORY.md`
- `internal/hooks/payloads.go`
- `internal/hooks/types.go`
- `internal/tools/tool.go`
- `internal/tools/tool_test.go`

## Errors / Corrections
- Resolved the spec tension between canonical `Tool` JSON (`name`) and hook payload naming (`tool_name`) by keeping the canonical marshal shape and adding decode compatibility for the hook alias.

## Ready for Next Run
- Focused validation passed with `go test ./internal/tools -coverprofile=/tmp/internal-tools.cover.out -covermode=count` at 96.8% coverage.
- Full verification passed with `make verify` before and after the local commit.
- Local commit created: `e9e4f9f` (`feat: add minimal tool foundation`).
