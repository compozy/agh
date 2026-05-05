# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement task_10 native AGH network tool hard cut: add thread/direct/work native tools, align hosted/MCP schemas, keep `agh__network_send` on `surface` + container IDs + `work_id`, and verify closed-schema/raw-token behavior.

## Important Decisions

- Scope stays on native/hosted tool surfaces; extension Host API remains task_11.
- Preserve strict schema behavior with `additionalProperties:false`; deleted `interaction_id` must fail before dispatch.
- New direct-room tool descriptions must say visibility is restricted to two peers plus runtime/audit access, and must not present direct rooms as cryptographic privacy.

## Learnings

- Pre-change code exposes only `agh__network_status`, `agh__network_channels`, `agh__network_inbox`, `agh__network_peers`, and `agh__network_send` in built-in descriptors/toolsets/native daemon bindings.
- Existing `agh__network_send` already carries `surface`, `thread_id`, `direct_id`, and `work_id`, but native schema validation only enforces type/required/properties/additionalProperties, so it cannot currently enforce enum or surface/container combinations.
- Hosted MCP descriptor projection is backed by native registry `ToolView` schemas, so descriptor parity should fall out of built-in descriptor updates but needs an explicit parity test.
- Implemented schema validator support for `enum`, `allOf`, `anyOf`, `oneOf`, and `not` so `agh__network_send` can fail closed on surface/container mismatches before dispatch.
- Focused package test result after first implementation/test pass: `go test ./internal/tools ./internal/tools/builtin ./internal/daemon` passed.
- Added final native/hosted tool set: `agh__network_threads`, `agh__network_thread_messages`, `agh__network_directs`, `agh__network_direct_resolve`, `agh__network_direct_messages`, and `agh__network_work`.
- Native read tools dispatch through `NetworkStore`; `agh__network_direct_resolve` resolves deterministic two-peer direct rooms from the caller session and target peer via `Network.ListPeers` + `NetworkStore.ResolveDirectRoom`.
- Network native tool results now route through `structuredNetworkResult`, which redacts raw `agh_claim_` material from structured JSON, previews, and content blocks.
- Focused coverage evidence after final test pass: `internal/tools` 81.0%, `internal/tools/builtin` 93.5%, broad `internal/daemon` package 72.8%; touched task_10 daemon functions are >=80% (`networkDirects` raised to 85.7%).
- Final verification passed: `make verify` exit 0 after frontend format/lint/typecheck/test/build, Go lint with `0 issues`, Go tests (`DONE 8315 tests`), and package boundary checks.

## Files / Surfaces

- Planned production surfaces: `internal/tools/schema.go`, `internal/tools/builtin_ids.go`, `internal/tools/builtin/network.go`, `internal/tools/builtin/toolsets.go`, `internal/daemon/native_tools.go`.
- Planned tests: built-in descriptor/schema tests, native daemon dispatch/validation tests, hosted/MCP descriptor parity checks, focused tool schema validation tests.
- Touched production surfaces: `internal/tools/schema.go`, `internal/tools/builtin_ids.go`, `internal/tools/builtin/network.go`, `internal/tools/builtin/toolsets.go`, `internal/daemon/native_tools.go`.
- Touched tests: `internal/tools/native_test.go`, `internal/tools/builtin/builtin_test.go`, `internal/daemon/native_tools_test.go`.

## Errors / Corrections

- First full `make verify` after implementation failed Go lint on schema validator cyclomatic complexity and long descriptor lines; fixed by extracting schema combinator helpers and wrapping descriptions.
- One later full `make verify` failed unrelated frontend Vitest timeout cases in site route metadata and extension SDK integration; both passed in isolation, no production/test weakening was applied, and the next full `make verify` passed.
- `scripts/check-test-conventions.py` referenced by AGH test conventions is not present in this repo; could not run that optional script.

## Ready for Next Run

- Code/test implementation was committed locally as `49e235ab` (`feat: add native network thread and direct tools`).
- Tracking/memory files are intentionally tracking artifacts and should not be staged unless the repo policy changes.
