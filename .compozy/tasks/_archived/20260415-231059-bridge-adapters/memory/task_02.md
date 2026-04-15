# Task Memory: task_02.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Replace the instance-scoped bridge initialize handshake with a provider-scoped runtime context that can carry multiple managed bridge instances and their resolved secret bindings.
- Refactor daemon and extension manager lifecycle wiring so one provider extension process can launch for multiple enabled bridge instances owned by the same extension.

## Important Decisions

- Treat `_techspec.md` and ADR-001 as the approved design, so no extra design branch is needed before implementation.
- Keep daemon ownership of lifecycle transitions and secret binding resolution; the provider runtime context is a launch-time snapshot, not a source of truth.
- Limit host API changes in this task to whatever is required for the new runtime context to remain correct with existing bridge flows.
- Keep legacy no-argument bridge host methods (`bridges/instances/get`, `bridges/instances/report_state`) bound to `SingleManagedInstance()` for now; broader multi-instance host ergonomics stay in task_04.

## Learnings

- Provider-scoped runtime negotiation now uses `runtime_version`, `provider`, `platform`, and `managed_instances[]`, with each managed snapshot carrying its own bound secrets.
- Daemon launch now resolves all enabled bridge instances for an extension, locks lifecycle updates across the full set, materializes secret bindings per instance, and rolls persisted state back if a transition in the launch set fails.
- Extension manager restart and runtime issue bookkeeping now fan out across all managed bridge instance IDs instead of a single runtime-bound instance.
- The bridge harness, Telegram reference adapter, and TypeScript SDK test fixtures all needed the new provider-scoped runtime shape.

## Files / Surfaces

- `internal/subprocess/handshake.go`
- `internal/subprocess/handshake_test.go`
- `internal/daemon/bridges.go`
- `internal/daemon/bridges_test.go`
- `internal/daemon/daemon_integration_test.go`
- `internal/extension/manager.go`
- `internal/extension/manager_test.go`
- `internal/extension/manager_integration_test.go`
- `internal/extension/host_api_bridges.go`
- `internal/extension/host_api_test.go`
- `internal/extension/bridge_delivery_integration_test.go`
- `internal/extension/telegram_reference_integration_test.go`
- `internal/extensiontest/bridge_adapter_harness.go`
- `internal/extensiontest/bridge_adapter_harness_test.go`
- `sdk/examples/telegram-reference/main.go`
- `sdk/examples/telegram-reference/main_test.go`
- `sdk/typescript/src/extension.test.ts`
- `sdk/typescript/src/generated/contracts.ts`
- `openapi/agh.json`

## Errors / Corrections

- Existing dirty worktree includes unrelated task tracking/test files; do not modify or revert them as part of task_02.
- `make verify` initially failed on `staticcheck` because a test helper guard in `internal/extension/manager_test.go` did not make nil control flow obvious; fixed with an explicit return after `t.Fatal`.
- Broader `go test -tags integration -cover ./internal/extension` still hits the pre-existing reference-extension symlink-guard failure noted in shared memory, but the task-specific TypeScript handshake regression in `sdk/typescript/src/extension.test.ts` is fixed and `bun run test` now passes.

## Ready for Next Run

- Task implementation and verification are complete. Next run only needs task tracking updates and the local commit if they have not been performed yet.
