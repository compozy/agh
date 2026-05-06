# Free Iteration 032

## Slice

Add task execution profile native read, update, and delete tools wired to task.Service profile authority.

## Decisions

- Added native task tool IDs for execution profile get/set/delete:
  - `agh__task_execution_profile_get`
  - `agh__task_execution_profile_set`
  - `agh__task_execution_profile_delete`
- Exposed model-facing native names through the task toolset:
  - `task_execution_profile_get`
  - `task_execution_profile_set`
  - `task_execution_profile_delete`
- Kept profile authority in `task.Service`; native tools do not write GlobalDB directly.
- `set` accepts a typed execution-profile payload and lets `task.Service.SetExecutionProfile` own validation, active-run rejection, normalization, persistence, and audit events.
- `delete` delegates to `task.Service.DeleteExecutionProfile`; default profile behavior remains service-owned rather than encoded in the tool layer.
- Tool JSON schemas and Go decoding reject unknown/server-owned fields, such as `created_at`, before any profile write.

## Files

- `internal/tools/builtin_ids.go`
- `internal/tools/builtin/tasks.go`
- `internal/tools/builtin/toolsets.go`
- `internal/tools/builtin/builtin_test.go`
- `internal/daemon/native_tools.go`
- `internal/daemon/native_profile_tools.go`
- `internal/daemon/native_tools_test.go`

## Verification

- `go test ./internal/tools/builtin -count=1` passed.
- `go test ./internal/daemon -run 'TestDaemonNativeTools/Should_route_task_execution_profile|TestDaemonNativeTools/Should_reject_malformed_task_execution_profile' -count=1` passed.
- `go test ./internal/tools/builtin ./internal/daemon -count=1` passed.
- `go test -race ./internal/daemon -run 'TestDaemonNativeTools/Should_route_task_execution_profile|TestDaemonNativeTools/Should_reject_malformed_task_execution_profile' -count=1` passed.
- `go test -race ./internal/daemon -run TestDaemonNativeTools -count=1` passed.
- First `make lint` failed on `gocritic hugeParam` for a large profile input receiver in `native_profile_tools.go`; fixed by changing the helper receiver to a pointer.
- `go test ./internal/tools/builtin ./internal/daemon -run 'TestBuiltinNativeDescriptors|TestDaemonNativeTools/Should_route_task_execution_profile|TestDaemonNativeTools/Should_reject_malformed_task_execution_profile' -count=1` passed after the lint fix.
- `make lint` passed with `0 issues`.
- `make verify` passed with Bun lint/typecheck/test, web build, `golangci-lint` 0 issues, Go race gate `DONE 8232 tests in 141.445s`, and package boundaries respected.

## Remaining

- Native review request/list/show or diagnostics tools remain.
- HTTP/UDS/CLI contract and transport surfaces remain.
- Web package, packages/site docs, and docs/_memory lessons remain.
- QA report/execution and three clean CodeRabbit rounds remain.
- The tracked `.agents/skills/cy-codex-loop/scripts/__pycache__/_state_io.cpython-314.pyc` artifact remains dirty and must not be cleaned without explicit user permission.
