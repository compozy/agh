# Improvements Report — internal/workspace

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/workspace/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/workspace | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 19 | `TestWorkspaceHelperFunctions` | `internal/workspace/resolver_test.go:915` |
| 18 | `TestResolveCacheHitInvalidateAndEviction` | `internal/workspace/resolver_test.go:461` |
| 17 | `TestResolverCRUDFlow` | `internal/workspace/resolver_test.go:369` |
| 13 | `TestResolveRoutesByIdentifierType` | `internal/workspace/resolver_test.go:22` |
| 12 | `TestResolverIntegrationEnvironmentConfigRoundTrip` | `internal/workspace/resolver_integration_test.go:145` |
| 11 | `(*Resolver).createWorkspaceRegistration` | `internal/workspace/resolver_crud.go:139` |
| 11 | `(*Resolver).Update` | `internal/workspace/resolver_crud.go:64` |
| 11 | `(*Resolver).ResolveOrRegister` | `internal/workspace/resolver.go:161` |
| 10 | `TestResolverIntegrationRegisterResolveAndMergeResources` | `internal/workspace/resolver_integration_test.go:22` |
| 10 | `scanSkillSource` | `internal/workspace/scanner.go:129` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/workspace/resolver.go` | 306 | Resolution, cache reuse, eviction, logging, and auto-registration fallback all live in one unit; behavior is still coherent, but the file sits just over the size threshold. |

### Refactoring — Duplication

Scanned with `dupl -plumbing -t 20 internal/workspace`.

Production duplicates at or above the reporting threshold:

- `internal/workspace/resolver.go:196-211` ↔ `internal/workspace/resolver_crud.go:24-39` — duplicated rollback/logging block around failed eager resolution.
- `internal/workspace/clone.go:83-93` ↔ `internal/workspace/clone.go:124-134` — mirrored map-clone loops for sandbox profiles and providers.

Notable below-threshold duplication that did not reach the 8-line reporting bar:

- `internal/workspace/scanner.go:97-102` ↔ `internal/workspace/scanner.go:140-145`
- `internal/workspace/options.go:29-33` ↔ `internal/workspace/options.go:56-60`
- `internal/workspace/resolver_crud.go:258-263` ↔ `internal/workspace/resolver_crud.go:269-274`

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `(*Resolver).Resolve` | `internal/workspace/resolver.go:82` | Primary runtime lookup path for workspace-backed session creation, workspace detail responses, and daemon hook rebuilds. Cache hits still rescan snapshots and deep-clone the resolved payload, so the steady-state path needs measurement. | `BenchmarkResolverResolve/cache_hit` |
| `(*Resolver).Resolve` | `internal/workspace/resolver.go:82` | Cache misses execute the full store lookup, root validation, config load, file snapshot walk, agent load, and skill merge pipeline. This is the package's dominant I/O-heavy path. | `BenchmarkResolverResolve/cache_miss` |
| `scanWorkspace` | `internal/workspace/scanner.go:37` | Performs repeated `os.ReadDir` / snapshot work over workspace, additional-root, and global discovery trees on cache misses. It is benchmarked transitively through the miss path because it is not called independently in production. | `BenchmarkResolverResolve/cache_miss` |
| `cloneResolvedWorkspace` | `internal/workspace/clone.go:16` | Allocation-heavy deep clone used on every cache hit before a resolved workspace leaves the resolver. | `BenchmarkCloneResolvedWorkspace` |
| `(*Resolver).List` | `internal/workspace/resolver_crud.go:112` | API/UI workspace listing path that clones the full store result on every request. | `BenchmarkResolverList` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/workspace/...`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/workspace/...`

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkResolverResolve/cache_hit-16` | 143534 | 35216 | 193307 | 35221 | not-hot-confirmed-by-benchmark |
| `BenchmarkResolverResolve/cache_miss-16` | 272352 | 81818 | 254898 | 81816 | not-hot-confirmed-by-benchmark |
| `BenchmarkResolverList-16` | 22927 | 40960 | 24968 | 40960 | not-hot-confirmed-by-benchmark |
| `BenchmarkCloneResolvedWorkspace-16` | 364.5 | 768 | 362.3 | 768 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| none | none | none | No production `go` statements exist in `internal/workspace/`. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| none | 0 | none | none | none | No production channel declarations exist in `internal/workspace/`. |

### Concurrency — Mutex Inventory

| File:Line | Read/write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/workspace/resolver.go:45` | read-heavy | `Resolver.cache` and cached-entry `lastAccess` timestamps | Guards cache hit reuse, cache replacement, cache invalidation, and TTL eviction. No lock is held across config loads or filesystem scans. |

### Concurrency — Select Audit

No production `select` statements exist in `internal/workspace/`.

### Security — Threat Model

- Trust boundaries:
  - External callers reach this package through the HTTP/UDS workspace handlers (`internal/api/core/workspaces.go:16-195`) and session creation/resume flows (`internal/session/manager_workspace.go:14-71`).
  - Once a workspace is registered, the resolver reads local filesystem content from the workspace root, any additional roots, and the global AGH home to assemble runtime-visible config, agents, and skills.
- Attacker capabilities:
  - An authenticated local caller can supply workspace IDs, names, root paths, additional directories, default-agent names, and environment refs.
  - A caller who controls the chosen local filesystem tree can influence `AGENT.md`, `SKILL.md`, `.agh/config.toml`, and `.agh/mcp.json` contents that this package reads.
- In-scope assets:
  - Correct workspace registration/rollback semantics.
  - Safe path canonicalization before store writes and resolver filesystem walks.
  - Integrity of the resolved workspace snapshot returned to other runtime packages.
- Out-of-scope:
  - Authorization policy in the HTTP/UDS/API layers.
  - Trustworthiness of already-selected local files; this package parses them but does not sandbox or attest their contents.
  - Downstream command execution or network behavior driven by config/agent contents after the snapshot leaves this package.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/workspace/resolver_crud.go:13-41` | JSON request body fields from `internal/api/core/workspaces.go:16-45` (`root_dir`, `add_dirs`, `name`, `default_agent`, `sandbox_ref`) | API layer trims and requires absolute paths; package canonicalizes `RootDir` with `canonicalRoot` and de-dupes `AdditionalDirs` via `normalizeAdditionalDirs` | `Store.InsertWorkspace`, then eager `Resolve` and local filesystem scans | LOW — rejected; attacker input only reaches store writes and local path canonicalization after absolute-path validation, with no package-local command execution or network sink. |
| `internal/workspace/resolver.go:161-213` | JSON `path` from `internal/api/core/workspaces.go:170-195` or session `workspace_path` from `internal/session/manager_workspace.go:20-38` | API layer validates absolute paths; package canonicalizes the path before store lookup/insert | `Store.GetWorkspaceByPath` / `Store.InsertWorkspace`, then eager `Resolve` over local filesystem state | LOW — rejected; path input is normalized into a local directory lookup/registration flow only. |
| `internal/workspace/resolver.go:82-157` and `internal/workspace/resolver_crud.go:212-249` | Path/id/name references from HTTP path params (`internal/api/core/workspaces.go:73-188`), session resume (`internal/session/manager_workspace.go:57-70`), and daemon rebuilds (`internal/daemon/hooks_bridge.go:1003-1029`) | `strings.TrimSpace`; absolute paths are canonicalized before path lookups; store APIs enforce registration existence | Store lookups, `refreshRootDir`, `scanWorkspace`, config/agent/skill parsing | LOW — rejected; the package performs lookup and local file reads only, with no package-local injection sink. |
| `internal/workspace/resolver_crud.go:64-108` | JSON update body fields from `internal/api/core/workspaces.go:95-150` (`name`, `add_dirs`, `default_agent`, `sandbox_ref`) | API layer trims and validates paths; package canonicalizes `AdditionalDirs` relative to the stored root | `Store.UpdateWorkspace` and cache invalidation | LOW — rejected; inputs are persisted after normalization and do not reach a package-local execution sink. |
| `internal/workspace/scanner.go:37-244` | Local filesystem contents under workspace/additional/global discovery roots (`AGENT.md`, `SKILL.md`, `.agh/config.toml`, `.agh/mcp.json`) | Roots come from prior canonicalization; file presence is snapshot-checked before loading; downstream parsers validate syntax | `filesnap.FromPath`, `os.ReadDir`, `aghconfig.LoadAgentDefFile`, and config loading inside `buildResolvedWorkspace` | LOW — rejected; this is expected local file parsing inside the workspace trust boundary, not untrusted remote input crossing into command or network execution in this package. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `WS-REF-001` | refactoring-analysis | medium | `internal/workspace/resolver.go:195`, `internal/workspace/resolver_crud.go:23` | Failed eager-resolution cleanup reused the caller context in both registration paths, so a canceled request could leave a partially registered workspace behind when rollback delete needed the same canceled context. | fixed |
| `WS-REF-002` | refactoring-analysis | low | `internal/workspace/resolver.go:195`, `internal/workspace/resolver_crud.go:23` | The rollback/logging block remains duplicated across `ResolveOrRegister` and `Register` after the bug fix. | wontfix |
| `WS-REF-003` | refactoring-analysis | low | `internal/workspace/clone.go:83`, `internal/workspace/clone.go:124` | `cloneEnvironmentProfiles` and `cloneProviders` still mirror one another line-for-line. | wontfix |
| `WS-OPT-001` | extreme-software-optimization | low | `internal/workspace/resolver.go:82`, `internal/workspace/resolver_crud.go:112`, `internal/workspace/clone.go:16` | A clone/list micro-optimization was benchmarked and rejected because the final full benchmark command did not show a stable package-level win. | wontfix |

## Per-Skill Notes

### refactoring-analysis

- `WS-REF-001` is the package-local fix for this task. The duplication scan surfaced the same rollback block in both registration paths; root-cause review showed both blocks reused the caller context for cleanup, which breaks rollback when the request is canceled after insert succeeds. Both sites now delete with `context.WithoutCancel(ctx)`, and regression coverage was added in `TestRegisterRollsBackWhenResolveContextCanceled` and `TestResolveOrRegisterRollsBackWhenResolveContextCanceled`.
- `WS-REF-002` is a deliberate `wontfix`. The duplicated rollback/logging block is now behaviorally correct, and extracting it would require a control-flow helper across two short call sites with different return types for minimal maintainability gain.
- `WS-REF-003` is a deliberate `wontfix`. Collapsing the mirrored map-clone loops would require a generic callback-style abstraction in a file whose remaining production duplication is already small and mechanically obvious.

### extreme-software-optimization

- The package's hot paths were measured with the required full command before and after the final production code.
- No optimization landed. A clone/list micro-optimization was tested and then removed because the final full benchmark command did not show a stable improvement across the direct clone benchmark or the list path.
- The remaining benchmarked paths are either stable at their current cost or dominated by unavoidable filesystem/config work on cache misses, so they remain `not-hot-confirmed-by-benchmark`.

### ubs

- `not-run` due to missing skill-runner support in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- Inventory complete.
- `internal/workspace` has no production goroutines, channels, or `select` statements. The only shared-state surface is `Resolver.mu`, which is released before config loads, filesystem scans, and store-backed resolution work.

### security-review

- No HIGH-confidence or MEDIUM-confidence vulnerability survived the threat-model review.
- The package's attacker-reachable surfaces are local workspace registration/lookups and local file parsing under the chosen workspace/home roots. Those surfaces do not execute commands, issue network requests, or make authorization decisions inside this package.

## Deferred Items (carry forward)

None.

## `make verify`

Command: `make verify`

Exit code: `0`

Excerpt:

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
0 issues.
DONE 4524 tests in 1.322s
OK: all package boundaries respected
```

Non-fatal warnings observed during the run:

- Repeated Node warnings: `The 'NO_COLOR' env is ignored due to the 'FORCE_COLOR' env being set.`
- macOS linker warning while building `golangci-lint`: `ld: warning: -bind_at_load is deprecated on macOS`
