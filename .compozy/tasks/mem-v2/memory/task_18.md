# Task Memory: task_18.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Hard-cut builtin/native Memory tools and extension Host API memory operations to the approved Memory v2 Slice 1 surface.
- Task references checked: `_techspec.md` `Native tools`, `Extensibility Integration Plan`, `Agent Manageability Plan`, `Development Sequencing` steps 14 and 24, ADR-008, ADR-009, and ADR-010.

## Important Decisions

- Builtin memory descriptors now expose only `agh__memory_list`, `agh__memory_show`, `agh__memory_search`, `agh__memory_propose`, and `agh__memory_note`; legacy `agh__memory_read` and `agh__memory_history` remain only as negative assertions in tests.
- `agh__memory_show` and `agh__memory_search` are read-only native tools. `agh__memory_propose` and `agh__memory_note` are mutating proposal tools gated by tool policy; root agents receive them through the memory toolset while child projections can inherit only read tools.
- Native writes enter through `Store.ProposeWrite` / `Store.ProposeDelete` with `OriginTool`; native output redacts public decision payloads and does not expose WAL-only replay content.
- `agh__memory_note` is controller-backed and uses a generated ad-hoc filename compatible with `Store.ProposeWrite`. It does not bypass the store to materialize `_system/ad_hoc` paths because current proposal validation rejects path separators.
- Extension Host API recall now prefers the active/default `MemoryProvider` and falls back to `Store.Recall` only when the provider registry is absent, no provider is selected, or the provider returns `ErrNotImplemented`.
- Extension Host API store/forget paths remain controller-backed through the Store proposal seam from task 05; this task avoided reintroducing direct raw write/delete bypasses.

## Learnings

- Recall tests must use non-trivial query text; short one-token queries intentionally skip recall under the deterministic recall policy.
- Host API recall fixtures need a catalog DB path under the test home so `Store.Recall` sees catalog-backed matches.
- Broad `internal/extension` `-race` runs with default Go test parallelism can hit SQLite migration deadlines on macOS. The project gate profile uses `-parallel=4`; the package passed under that profile.
- `make verify` can still hit existing macOS cleanup flakes in `internal/extension`; targeted reruns and the final gate passed without changing unrelated cleanup code.

## Files / Surfaces

- Builtin descriptors and IDs: `internal/tools/builtin_ids.go`, `internal/tools/builtin/memory.go`, `internal/tools/builtin/toolsets.go`, `internal/tools/builtin/builtin_test.go`.
- Daemon native runtime: `internal/daemon/native_tools.go`, `internal/daemon/native_tools_test.go`.
- Extension Host API memory recall/provider parity: `internal/extension/host_api.go`, `internal/extension/host_api_test.go`.

## Errors / Corrections

- Initial focused tests exposed stale `memory_read`/`memory_history` assertions and trivial recall fixture text; tests were updated to the final tool surface and durable recall wording.
- `make lint` initially failed on `goconst`, `lll`, and an unused helper in `internal/daemon/native_tools.go`; the native helpers were tightened without changing behavior.
- First full `make verify` hit the known `@agh/extension-sdk` integration timeout. The isolated test passed in 181ms, and standalone `make bun-test` passed.
- A later `make verify` hit `internal/extension` `TempDir` cleanup failures. The two failing tests passed isolated, and `go test -race -parallel=4 ./internal/extension -count=1` passed before the final gate rerun.

## Ready for Next Run

- Focused validation passed: `go test ./internal/tools/builtin ./internal/daemon ./internal/extension -run 'TestBuiltinNativeDescriptors|TestHostAPIHandlerMemory|TestDaemonNative' -count=1`.
- Broader validation passed: `go test ./internal/tools/... ./internal/daemon ./internal/extension ./internal/api/... -count=1`.
- Race validation passed: `go test -race ./internal/tools/builtin ./internal/daemon ./internal/extension -run 'TestBuiltinNativeDescriptors|TestDaemonNativeTools|TestDaemonNativeRuntimePolicyResolver|TestHostAPIHandlerMemory' -count=1`.
- Gate-profile extension validation passed: `go test -race -parallel=4 ./internal/extension -count=1`.
- Coverage validation passed for the focused descriptor package: `internal/tools/builtin` 93.5%. Broad `internal/daemon` and `internal/extension` packages remain below 80% overall due existing package breadth.
- `git diff --check` passed.
- Full pre-tracking `make verify` passed with Bun tests 330 files / 2090 tests, Go lint `0 issues`, Go tests `DONE 8356 tests`, and package boundaries `OK`.
- Full final post-tracking `make verify` passed with Bun tests 330 files / 2090 tests, Go lint `0 issues`, Go tests `DONE 8356 tests`, and package boundaries `OK`.
- Next task after state update should be `task_19` (Daemon Wiring and Boundary Registration).
