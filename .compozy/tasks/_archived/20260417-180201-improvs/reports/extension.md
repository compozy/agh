# Improvements Report — internal/extension

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/extension/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo $(rg --files internal/extension -g '*.go' -g '!**/*_test.go' | sort) | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 20 | `(*HostAPIHandler).replayPromptDeliveryEvents` | `internal/extension/host_api_bridges.go:588` |
| 18 | `normalizeManagerDefaults` | `internal/extension/manager.go:388` |
| 17 | `(*Manifest).Validate` | `internal/extension/manifest.go:235` |
| 17 | `(*HostAPIHandler).handleBridgesInstancesReportState` | `internal/extension/host_api_bridges.go:92` |
| 16 | `(*HostAPIHandler).applyTriggerUpdateParams` | `internal/extension/host_api.go:1955` |
| 14 | `(*Manager).hookConfigToDecl` | `internal/extension/manager.go:1481` |
| 13 | `(*Manager).resolveBridgeRuntime` | `internal/extension/manager.go:2025` |
| 13 | `(*HostAPIHandler).handleSessionsList` | `internal/extension/host_api.go:660` |
| 13 | `(*HostAPIHandler).handleEnvironmentList` | `internal/extension/host_api.go:836` |
| 12 | `extensionHealth` | `internal/extension/describe.go:73` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/extension/host_api.go` | 2456 | Host API dispatch, capability/rate limiting, session/environment/memory/observe/automation handling, and response helpers are concentrated in one unit. |
| `internal/extension/manager.go` | 2338 | Extension lifecycle orchestration, subprocess launch/supervision, resource registration, hook declaration, and bridge runtime handling all share one file. |
| `internal/extension/manifest.go` | 1096 | Manifest schema, parsing, validation, semantic-version checks, and helper cloning live together. |
| `internal/extension/host_api_bridges.go` | 1074 | Bridge instance queries, ingress routing, delivery registration/replay, dedup, and retry helpers are bundled together. |
| `internal/extension/host_api_tasks.go` | 886 | Task list/get/create/update/cancel/run mutation handlers, request translation, error mapping, and payload mappers are concentrated in one file. |
| `internal/extension/registry.go` | 857 | Install/list/get/update/uninstall persistence, manifest loading, checksum logic, and SQL row mapping share one unit. |
| `internal/extension/contract/host_api.go` | 749 | Contract enums, params, payloads, and method specs for the whole Host API surface are concentrated together. |
| `internal/extension/capability.go` | 715 | Capability/security ceilings, resource-grant resolution, host-method permission checks, and clone helpers live in one file. |
| `internal/extension/bundle.go` | 691 | Bundle schema loading, validation, file parsing, and manifest-derived bundle resolution are concentrated together. |
| `internal/extension/install_managed.go` | 564 | Managed install staging, registry install, bundle cleanup, and filesystem copy semantics are bundled together. |
| `internal/extension/surfaces/registry.go` | 324 | Surface catalog, manifest-family normalization, publishable-kind lookup, and scope grant normalization share one unit. |
| `internal/extension/contract/sdk.go` | 316 | SDK root-type catalog and hook contract type graph live together in one contract file. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/extension | rg -v '_test.go'` notable production duplicates:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/extension/host_api_tasks.go:225-347` | `internal/extension/host_api_tasks.go:256-378` | Task-run mutation handlers repeat the same decode -> id validation -> actor derivation -> manager call -> payload response shape. |
| `internal/extension/host_api_tasks.go:132-192` | `internal/extension/host_api_tasks.go:225-285` | Task cancel and task-run mutation handlers share near-identical request translation and RPC error mapping scaffolding. |
| `internal/extension/host_api.go:1220-1235` | `internal/extension/host_api.go:1326-1341` | Automation job and trigger lookup handlers share the same decode/manager/get/return shape. |
| `internal/extension/capability.go:481-504` | `internal/extension/protocol/host_api.go:177-200` | `normalizeUniqueStrings` is duplicated across packages inside `internal/extension/`. |
| `internal/extension/capability.go:650-673` | `internal/extension/surfaces/registry.go:294-317` | `intersectScopes` logic is duplicated across package-local capability/surface helpers. |
| `internal/extension/bundle.go:272-294` | `internal/extension/bundle.go:320-342` | Bundle channel and trigger normalization paths repeat the same trim/clone/validate helper pattern. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `decodeHostAPIParams` | `internal/extension/host_api.go:2192` | Every Host API method decodes raw JSON params before any capability-specific work, so decode overhead is paid on every extension RPC. | `BenchmarkDecodeHostAPIParamsTaskCreate` |
| `taskSummaryPayloadsFromSummaries` | `internal/extension/host_api_tasks.go:642` | The task list endpoint converts every summary row into API payloads on the read path. | `BenchmarkTaskSummaryPayloadsFromSummaries` |
| `taskRunPayloadsFromRuns` | `internal/extension/host_api_tasks.go:707` | The task-run list endpoint converts every run and clones JSON results on the read path. | `BenchmarkTaskRunPayloadsFromRuns` |

### Optimization — Benchmark Results

Baseline command for `before` numbers: `go test -bench=. -benchmem -count=5 ./internal/extension/...` before production fixes.
Final command for `after` numbers: `go test -bench=. -benchmem -count=5 ./internal/extension/...` after all `internal/extension/` changes land.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkDecodeHostAPIParamsTaskCreate` | 3535.4 | 1992 | 3461.8 | 1096 | fixed-with-benchmark |
| `BenchmarkTaskSummaryPayloadsFromSummaries` | 27900.0 | 147456 | 30611.6 | 147456 | measured-no-win |
| `BenchmarkTaskRunPayloadsFromRuns` | 48713.0 | 379264 | 55777.2 | 379264 | measured-no-win |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `internal/extension/manager.go:543` | `(*Manager).Stop` | `stopWG` joins each goroutine after `proc.Shutdown(ctx)` and state cleanup. | Per-extension stop workers drain active subprocesses in parallel during manager shutdown. |
| `internal/extension/manager.go:961` | `(*Manager).initializeExtension` / manager lifecycle | Exits when the supervised process finishes, recovery stops, or `lifecycleDone()` closes. | Supervises health polling and crash recovery for one managed extension generation. |
| `internal/extension/host_api.go:1609` | `(*HostAPIHandler).submitPrompt` | Exits when the prompt event channel closes. | Background drainer ensures prompt event channels do not block producer progress once submission starts. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| `internal/extension/manager.go:534` | `len(names)` | `(*Manager).Stop` | `Stop` closes after `stopWG.Wait()` | `Stop` ranges after all workers finish | Aggregates per-extension shutdown errors without blocking worker goroutines. |
| `internal/extension/manager.go:1926` | 0 | `(*Manager).lifecycleDone` | Closed immediately inside `lifecycleDone` when no lifecycle context exists | `monitorProcess`, `waitBackoff` | Ephemeral closed channel avoids nil-blocking before manager startup or after shutdown. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/extension/manager.go:197` | read-heavy | Manager lifecycle state, extension map, and per-extension mutable runtime state | Shared across start/stop/reload/get/list/supervision helpers. |
| `internal/extension/capability.go:155` | read-heavy | Effective grants map and resource policy snapshot | Guards extension capability registration and check lookups. |
| `internal/extension/host_api.go:91` | write-heavy | Bridge-ingest cleanup scheduling (`bridgeLastCleanup`) | Prevents concurrent cleanup passes in bridge ingress. |
| `internal/extension/host_api.go:2341` | write-heavy | Host API token-bucket entries per extension name | Serializes rate-limit state updates and retry-after calculations. |
| `internal/extension/host_api_bridges.go:985` | write-heavy | Per-routing-key lock registry | Maintains the map of keyed ingress locks. |
| `internal/extension/host_api_bridges.go:991` | write-heavy | One keyed bridge-ingest critical section | Serializes concurrent ingress processing for the same routing-key hash. |

### Concurrency — Select Audit

All production `select` statements under `internal/extension/` are context-aware or manager-lifecycle-aware:

- `internal/extension/manager.go:1026` waits on `ticker.C`, `proc.Done()`, and `lifecycleDone()`.
- `internal/extension/manager.go:1912` waits on `timer.C` and `lifecycleDone()`.
- `internal/extension/host_api_bridges.go:635` and `:639` wait on timer expiration or `ctx.Done()` while replaying prompt delivery events.
- `internal/extension/host_api_bridges.go:1043` and `:1047` wait on retry timers or `ctx.Done()` during SQLite busy retries.

### Security — Threat Model

- Trust boundaries:
  - Extension subprocesses call the Host API through JSON-RPC methods implemented in `internal/extension/host_api*.go`.
  - Installed extension manifests and bundle specs are loaded from the local filesystem through `registry.go`, `manifest.go`, `bundle.go`, and `resource_publication.go`.
  - Bridge-provider runtimes send ingress payloads and state reports through the Host API bridge methods.
- Attacker capabilities:
  - A malicious or compromised extension subprocess can choose Host API method names and method parameters allowed by its granted capabilities.
  - A malicious bridge provider can send arbitrary inbound message envelopes and bridge state updates for its managed instances.
  - A local operator can provide malformed extension manifests, bundle files, or install paths, but operator-controlled filesystem/env inputs are treated as higher-trust than extension RPC payloads.
- In-scope assets:
  - Session/environment/task/resource integrity exposed through the Host API.
  - Workspace-scoped memory files and resource snapshots.
  - Bridge route ownership, dedup state, and delivery registration correctness.
  - Extension lifecycle state and granted capability boundaries.
- Out-of-scope:
  - Trust of extension code after the daemon intentionally grants a capability and forwards execution to other packages.
  - Malicious operators with direct control over AGH home directories, install roots, or process environment.
  - Security properties implemented exclusively in downstream packages (`session`, `resources`, `task`, `memory`, `bridges`) after validated requests leave `internal/extension/`.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/extension/host_api.go:448-477` | Extension-supplied Host API method name and raw params | Method lookup, capability check, and per-extension rate limit run before handler dispatch. | Handler execution inside `HostAPIHandler.Handle`. | LOW — rejected as safe because unauthorized methods are denied before any lower-level operation runs. |
| `internal/extension/host_api.go:913-933` | Extension-supplied environment exec request (`session_id`, `command`, `timeout`) | Non-empty validation plus upstream `CheckHostAPI(..., "sandbox/exec")` capability gate in `Handle`. | `sessions.ExecEnvironment(ctx, session.EnvironmentExecRequest{...})`. | LOW — rejected because execution requires explicit capability grant and active-session authorization in downstream session handling. |
| `internal/extension/host_api.go:985-1008` | Extension-supplied memory store request (`key`, `content`, `tags`, `workspace`) | `normalizeMemoryFilename` plus `memory.Store.cleanFilename` reject path separators; workspace scope resolves through `memoryStoreFor`. | `storeHandle.Write(scope, filename, []byte(doc))`. | LOW — rejected because filename/path traversal is blocked before filesystem writes. |
| `internal/extension/host_api_resources.go:49-96` | Extension-supplied resource snapshot records (`kind`, `scope`, `id`, `spec`) | Actor context restricts granted kinds/scopes; optional codec validation/canonicalization runs before apply. | `resourceStore.ApplySourceSnapshotRaw(...)`. | LOW — rejected because actor grants and codec validation bound writes to authorized resource shapes. |
| `internal/extension/host_api_bridges.go:137-183` | Bridge-provider inbound message envelope (`peer_id`, `thread_id`, `group_id`, `body`, `idempotency_key`) | Envelope validation, owned-instance authorization, runtime status checks, routing-key construction, keyed locking, and dedup handling. | `promptBridgeRoute` and `recordBridgeIngressDedup`. | LOW — rejected because ingress is restricted to the extension’s managed instances and serialized per routing key before session side effects. |
| `internal/extension/host_api_tasks.go:71-88` | Extension-supplied task create request (`scope`, `workspace`, `network_channel`, `metadata`) | Scope/workspace binding validation, channel validation, and task-domain validation before create. | `manager.CreateTask(ctx, spec, actor)`. | LOW — rejected because request translation validates scope/workspace/channel before durable task writes. |
| `internal/extension/resource_publication.go:36-125` | Manifest-controlled MCP/tool declarations and env templates from installed extension files | Command/path resolution confines relative paths to the extension root and validates MCP server specs. | `aghconfig.MCPServer` / `toolspkg.Tool` publication into daemon resource state. | LOW — rejected because extension installation is already a trusted local action and relative command resolution stays within the extension root. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `EXT-REF-001` | refactoring-analysis | low | `internal/extension/host_api_tasks.go:17-365` | Task handlers repeated the same task-manager and actor lookup scaffolding across the file. | fixed |
| `EXT-OPT-001` | extreme-software-optimization | low | `internal/extension/host_api.go:2193-2202` | `decodeHostAPIParams` converted raw JSON to strings on every Host API decode path before unmarshalling. | fixed |
| `EXT-OPT-002` | extreme-software-optimization | low | `internal/extension/host_api_tasks.go:640-710` | The task summary/run payload projection paths were benchmarked as hot-path candidates, but the attempted index-fill rewrite did not produce a durable win under the required rerun. | wontfix — the micro-optimization was reverted after the final benchmark pass regressed versus baseline. |
| `EXT-REF-003` | refactoring-analysis | medium | `internal/extension/manager.go:348-2085` | `manager.go` still concentrates lifecycle orchestration, subprocess launch, supervision, hook declaration, and bridge runtime resolution in one 2.3k LOC unit. | deferred — splitting lifecycle, runtime launch, and hook/bridge concerns would widen this pass into a package redesign. |
| `EXT-REF-004` | refactoring-analysis | medium | `internal/extension/host_api.go:324-2369` | `host_api.go` still combines dispatch, capability/rate limiting, and multiple Host API surfaces in one 2.4k LOC file. | deferred — decomposition is desirable but broader than this measured improvements pass. |

## Per-Skill Notes

### refactoring-analysis
- Reduced the highest-value in-file duplication in `host_api_tasks.go` by centralizing the repeated task-manager plus actor lookup in `taskManagerAndActor`.
- The remaining dominant structural issues are concentration, not small helper noise: `manager.go` and `host_api.go` still exceed 2k LOC and carry multiple reasons to change.
- I left the larger file splits deferred because they would expand this task beyond a package-local improvements pass into a broader architectural rewrite.

### extreme-software-optimization
- Added `internal/extension/perf_bench_test.go` before production changes to establish package-local baseline numbers.
- `decodeHostAPIParams` now trims raw bytes once and treats whitespace-wrapped `null` like an empty params object, reducing decode allocations from `1992 B/op` to `1096 B/op` and improving mean latency from `3535.4 ns/op` to `3461.8 ns/op`.
- `taskSummaryPayloadsFromSummaries` and `taskRunPayloadsFromRuns` were benchmarked as required hot-path candidates, but the attempted slice/index rewrite failed the evidence bar on the final rerun (`27900.0 ns/op` -> `30611.6 ns/op` for summaries, `48713.0 ns/op` -> `55777.2 ns/op` for runs).
- The summary/run projection change was reverted, and the report records those candidates as benchmarked-without-fix instead of claiming an inferred optimization.

### ubs
- `not-run` due missing skill-runner interface in this session; no manual substitute will be used.

### deadlock-finder-and-fixer
- Inventory complete; no confirmed deadlock or goroutine-leak finding yet.
- All production `select` sites are context-aware or manager-lifecycle-aware, and the package’s only owned goroutines have explicit join/exit paths.

### security-review
- Threat model and attacker-input inventory completed before any verdict.
- No HIGH or MEDIUM finding survived source-to-sink review. The exposed Host API and bridge-ingress surfaces are gated by capability checks, instance ownership, scope validation, or downstream store/session validation before side effects.

## Deferred Items (carry forward)

- **`EXT-REF-003`** — Split `internal/extension/manager.go` along lifecycle supervision, runtime launch, and declaration/resource preparation seams when a future task can absorb a larger refactor.
- **`EXT-REF-004`** — Split `internal/extension/host_api.go` by surface area (sessions/environment, memory/observe, automation, shared dispatch helpers) in a dedicated follow-up.

## `make verify`

Final command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
✓ built in 411ms
0 issues.
✓  internal/extension (7.66s)
✓  internal/cli (8.203s)
✓  internal/daemon (8.749s)
DONE 4466 tests in 10.586s
OK: all package boundaries respected
```

Observed non-fatal toolchain noise during the command:
- Node repeatedly warned that `NO_COLOR` is ignored because `FORCE_COLOR` is set.
- The macOS linker emitted `ld: warning: -bind_at_load is deprecated on macOS` while running the vendored `golangci-lint` command.

`make verify` exited with code `0` after the final post-fix rerun.
