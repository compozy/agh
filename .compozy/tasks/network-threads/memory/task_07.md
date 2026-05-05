# Task Memory: task_07.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 07 Network Hooks, Status Counters, and Observability.
- Success requires async-only `network.*` hook catalog/payload/matcher/introspection support, post-commit dispatch from durable conversation writes, failure isolation, aggregate runtime status counters, low-cardinality metrics, tests, clean `make verify`, tracking updates, and one local commit.

## Important Decisions

- Post-commit hook/stat dispatch will be wired from `internal/network.Manager.writeConversationMessage` after `WriteConversationMessage` returns a non-duplicate success.
- Network hook payloads may carry stable high-cardinality dedupe identifiers (`message_id`, `work_id`, `trace_id`, `causation_id`, container IDs) for hooks/logs, but metric labels must stay low-cardinality.
- Keep Task 07 status work manager/runtime-facing unless public API changes are required to compile; Task 08 owns public status payload exposure.
- Requested AGH-specific skills (`agh-code-guidelines`, `agh-test-conventions`, `deadlock-finder-and-fixer`, `nats`) were not installed in visible skill roots; using repo guidance plus installed Go/testing/debugging skills as fallback.

## Learnings

- Shared memory says Task 06 routes outbound/inbound conversation persistence through manager-owned `store.NetworkConversationStore` calls before publish/prompt side effects.
- Current hooks package has no `network` family/events yet; `allHookEvents` and `hookEventSpecs` are consistency-checked at init.
- Current network runtime status has transport/message kind counters but not open thread/direct/work/message total/direct resolve counters or low-cardinality metric samples.
- `internal/hooks` package-wide coverage is `82.2%` after adding focused dispatch-correlation/session-context and task/spawn async clone tests.
- `internal/network` coverage is `80.3%` with the new network hook/status tests.
- Codegen consumes hook introspection descriptors through `internal/extension/contract.HookContracts`; adding new hook payload schema names also requires registering those names in `namedHookTypes`.
- Adding flat string fields directly to `hooks.HookMatcher` can push `HookDecl` over gocritic's heavy-copy threshold; network and compaction matcher fields now use embedded pointer structs to keep external matcher JSON flat without increasing value-copy cost.
- Self-review caught that core network matcher support was not enough: config TOML/YAML parsing, extension manifest parsing, CLI hook info rows, settings maps, and tool-surface overlays also need to carry `channel`, `surface`, `kind`, `direction`, and `work_state`.

## Files / Surfaces

- Hook taxonomy/payload/matcher/introspection/dispatch: `internal/hooks/`.
- Post-commit runtime boundary and structured logs: `internal/network/manager.go`.
- Runtime counters/metrics: `internal/network/stats.go` and possibly `internal/network/delivery.go` for queue depth snapshots.
- Daemon bridge wiring: `internal/daemon/hooks_bridge.go`, `internal/daemon/boot.go`.
- Added code/tests: `internal/network/hooks.go`, `internal/network/hooks_test.go`, `internal/hooks/network_dispatch_test.go`.
- Extension SDK contract registry/codegen: `internal/extension/contract/sdk.go`, `internal/extension/contract/sdk_test.go`.
- Matcher config/operator surfaces: `internal/config/hooks.go`, `internal/config/tool_surface.go`, `internal/extension/manifest.go`, `internal/extension/manager.go`, `internal/settings/collections.go`, `internal/cli/hooks.go`.

## Errors / Corrections

- Daemon package test fakes needed no-op implementations for the new network hook runtime methods after `hookRuntime` was extended.
- First `make verify` failed at codegen with `unknown hook contract type "NetworkThreadOpenedPayload"`; root cause was the extension SDK hook-contract name registry missing the new network payload aliases.
- A later `make verify` failed at Go lint because the first `HookMatcher` shape made `HookDecl` 584 bytes, then exactly 512 bytes; fixed by embedding `NetworkMatcher` and `CompactionMatcher` pointer structs and regenerating contracts.
- The matcher surface gap was fixed with focused tests; validation passed for hooks/network/daemon/config/settings/cli/extension/contract packages after the correction.
- A follow-up full verify exposed real lint issues from heavier hook config structs (`rangeValCopy`, `hugeParam`) and `hookMatcherMap` complexity; fixed by passing/indexing config declarations by pointer and extracting matcher map helpers. `make lint` now reports `0 issues`.
- Full `make verify` passed after the self-review matcher-surface and lint corrections with `0 issues`, `DONE 8177 tests`, and package boundary validation OK.
- Pre-commit `make verify` passed after task tracking updates with frontend lint `Found 0 warnings and 0 errors`, Vitest `330 passed` / `2092 passed`, Go lint `0 issues`, `DONE 8177 tests`, and package boundary validation OK.
- Local implementation commit created as `797a8ad8 feat: add network observation hooks` and later amended to `4fffcc1d feat: add network observation hooks`. Tracking/memory files were intentionally kept out of the implementation commit per task staging guidance.
- Post-commit `make verify` passed with frontend lint `Found 0 warnings and 0 errors`, Vitest `330 passed` / `2092 passed`, Go lint `0 issues`, `DONE 8177 tests`, and package boundary validation OK.
- Follow-up coverage check found the broad `internal/hooks` package was still below the task coverage target, so focused tests were added; `go test ./internal/hooks -cover -count=1` now reports `82.2%`.
- Pre-amend full `make verify` passed after the coverage tests with frontend lint `Found 0 warnings and 0 errors`, Go lint `0 issues`, `DONE 8242 tests`, and package boundary validation OK.
- Final post-amend `make verify` passed for commit `4fffcc1d` with frontend lint `Found 0 warnings and 0 errors`, Go lint `0 issues`, `DONE 8242 tests`, and package boundary validation OK.

## Ready for Next Run

- Task 07 is complete in commit `4fffcc1d`; no task-local implementation work remains.
