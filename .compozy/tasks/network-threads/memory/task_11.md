# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement Task 11: extension Host API network read/write methods, SDK exports, capability gates, and bridge mapping rules separating provider/platform `ThreadID` from AGH conversation refs.
- Success requires required Host API methods, capability denial coverage, validation parity, bridge mapping tests, SDK compile/export coverage, clean `make verify`, tracking updates, and one local commit.

## Important Decisions
- Provider/platform bridge `ThreadID` must remain routing metadata. AGH conversation mapping must be explicit via a separate conversation ref surface, not inferred from provider `ThreadID`.
- Host API `network/send` intentionally reuses the public `NetworkSendRequest` decoder and mirrors the HTTP/UDS validation rules in `internal/extension` because importing `internal/api/core` from `internal/extension` would create a package cycle.
- Host API direct resolve uses runtime peer discovery plus `network.DirectRoomIdentity` and `store.ResolveDirectRoom`; it does not fabricate direct membership from bridge/provider fields.

## Learnings
- Pre-change signal: `internal/extension/protocol/host_api.go` has no `network/*` Host API method constants or registry entries, so Task 11 is not implemented yet.
- Focused checks passed after SDK finalization: `go test ./sdk/go ./internal/extension ./internal/bridges ./internal/extension/contract ./internal/extension/protocol ./internal/daemon -count=1`.
- TypeScript SDK checks passed: `bun run --cwd sdk/typescript typecheck` and `bun run --cwd sdk/typescript test`.
- Coverage snapshot for touched bridge package is above target (`internal/bridges` 80.8%); `internal/extension` package remains below 80% overall due existing package breadth, while new Task 11 read/write/mapping paths have focused coverage.
- Full verification passed after lint cleanup: `make verify` completed frontend format/lint/typecheck/tests/build, Go lint `0 issues`, race tests (`DONE 8361 tests in 116.211s`), build, and package-boundary checks (`OK: all package boundaries respected`).
- Final pre-commit verification passed again after tracking/memory updates: `make verify` completed frontend format/lint/typecheck/tests/build, Go lint `0 issues`, race tests (`DONE 8361 tests in 10.774s`), build, and package-boundary checks (`OK: all package boundaries respected`).
- Local implementation commit created: `ac76924f feat: expose network host api`.
- Final post-commit verification passed after recording the commit hash in memory: `make verify` completed frontend format/lint/typecheck/tests/build, Go lint `0 issues`, race tests (`DONE 8361 tests in 13.211s`), build, and package-boundary checks (`OK: all package boundaries respected`).

## Files / Surfaces
- Planned surfaces: `internal/extension/{protocol,contract,capability,host_api*}`, `internal/daemon`, `internal/bridges`, SDK roots under `sdk/`, and Task 11 tracking/memory files.
- Implemented surfaces: `internal/extension/protocol/host_api.go`, `internal/extension/contract/host_api.go`, `internal/extension/capability.go`, `internal/extension/host_api.go`, `internal/extension/host_api_network.go`, `internal/extension/host_api_bridges.go`, `internal/daemon/{boot,daemon}.go`, `internal/bridges/types.go`, `sdk/go/host_api.go`, and `sdk/typescript/src/{generated/contracts,host-api,index}.ts`.
- Test surfaces: `internal/extension/{capability,host_api_network}.go` tests, `internal/bridges/types_test.go`, protocol tests, Go SDK sensitive request test, and TypeScript Host API helper test.

## Errors / Corrections
- Added an extra store-backed Host API read regression after coverage review showed the new read handlers were under-exercised.
- First `make verify` run failed on lint only: `funlen` for expanded Host API registries, `gocritic appendCombine`, and two long contract lines. Fixed by splitting network method registration helpers and wrapping/combining the relevant lines; `make lint` then reported `0 issues`.

## Ready for Next Run
- Continue from repo `/Users/pedronauck/Dev/compozy/agh2` on branch `network-threads`; avoid destructive git commands and do not touch pre-existing dirty task/QA files outside Task 11 scope.
- Task 11 implementation is complete in local commit `ac76924f`. Tracking and workflow-memory files remain uncommitted per task instruction to keep tracking-only files out of the automatic code commit.
