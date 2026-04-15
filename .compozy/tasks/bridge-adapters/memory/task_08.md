# Task Memory: task_08.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Replace the single-instance `telegram-reference` path with provider-scoped conformance coverage built on `internal/bridgesdk`.
- Expand the reusable bridge adapter harness to validate negotiated managed instances, explicit owned-instance access, delivery acknowledgments, and per-instance state reporting.

## Important Decisions
- Use the approved PRD/TechSpec/ADR design directly instead of reopening design approval.
- Refactor `sdk/examples/telegram-reference` onto `internal/bridgesdk` rather than carrying forward its custom JSON-RPC bootstrap.
- Treat the updated `telegram-reference` example as conformance evidence only; keep production-provider behavior out of scope for this task.

## Learnings
- The current reference adapter still calls `InitializeBridgeRuntime.SingleManagedInstance()` during initialize and still uses `bridges/instances/get` without an explicit `bridge_instance_id`.
- The current harness validates only one expected bridge instance even though task 02/04 changed the runtime and Host API contract to provider scope.
- The provider-scoped harness must register `bridges/instances/list` alongside `get` and `report_state`; otherwise the reference runtime fails ownership negotiation during boot with `Method not found`.
- The provider-scoped example needs explicit `bridge_instance_id` routing in fake inbound updates to exercise multi-instance ownership, state, and delivery evidence without aliasing ack state.

## Files / Surfaces
- `internal/extensiontest/bridge_adapter_harness.go`
- `internal/extensiontest/bridge_adapter_harness_test.go`
- `internal/extensiontest/bridge_adapter_harness_integration_test.go`
- `sdk/examples/telegram-reference/main.go`
- `sdk/examples/telegram-reference/main_test.go`
- `sdk/examples/telegram-reference/README.md`
- `sdk/examples/telegram-reference/extension.toml`
- `internal/extension/telegram_reference_integration_test.go`
- `internal/bridgesdk/runtime.go`

## Errors / Corrections
- Added the missing harness host-method handler for `bridges/instances/list` after the first provider-scoped integration run exposed the gap during runtime initialize.

## Ready for Next Run
- Implementation, task-specific unit/integration coverage, package coverage checks, and `make verify` all passed after the provider-scoped harness/runtime refactor.
- Task tracking was updated locally, and the code/doc/test surfaces were committed as `e647bff` (`feat: add provider-scoped bridge conformance harness`).
