# Improvements Report — internal/config

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | 4 benchmarks in `internal/config/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 $(rg --files internal/config -g '!**/*_test.go') | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 19 | `(AutomationTrigger).validateWithEnv` | `internal/config/automation.go:172` |
| 14 | `SaveBootstrapConfig` | `internal/config/bootstrap.go:33` |
| 14 | `(AutomationJob).Validate` | `internal/config/automation.go:125` |
| 13 | `(NetworkConfig).Validate` | `internal/config/config.go:880` |
| 13 | `(*Config).ResolveAgent` | `internal/config/provider.go:112` |
| 11 | `(ExtensionsMarketplaceConfig).Validate` | `internal/config/config.go:966` |
| 10 | `LoadWorkspaceAgentDefs` | `internal/config/agent.go:155` |
| 10 | `(MarketplaceConfig).Validate` | `internal/config/config.go:935` |
| 9 | `validateMCPServerSpec` | `internal/config/mcp_resource.go:23` |
| 9 | `loadWithHome` | `internal/config/config.go:331` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/config/config.go` | 1111 | Schema definitions, env scoping, load orchestration, defaults, validation, environment resolution, and path helpers are combined in one monolith. |
| `internal/config/merge.go` | 656 | Overlay schema, TOML decode, per-section merge behavior, and overlay helpers live in one large file with repeated setter patterns. |
| `internal/config/provider.go` | 385 | Provider registry, agent resolution, MCP merge behavior, and cloning helpers are co-located. |
| `internal/config/agent.go` | 339 | Agent frontmatter parsing, workspace discovery, validation, and MCP sidecar merge all share one unit. |
| `internal/config/automation.go` | 330 | Automation schema validation, webhook secret handling, overlay application, and task/trigger conversion are tightly packed. |

### Refactoring — Duplication

`dupl -plumbing -t 60 internal/config | rg -v '_test.go'` notable production duplicates:

| Duplicate A | Duplicate B | Notes |
| --- | --- | --- |
| `internal/config/config.go:203-221` | `internal/config/merge.go:16-34` | `Config` and `configOverlay` repeat the same top-level section inventory with different field types. |
| `internal/config/merge.go:358-380` | `internal/config/merge.go:488-510` | `daytonaProfileOverlay.Apply` and `networkOverlay.Apply` are mirrored setter blocks. |
| `internal/config/merge.go:580-593` | `internal/config/merge.go:595-608` | Provider and environment overlay application use the same map-overlay loop shape. |

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `LoadForHome` | `internal/config/config.go:306` | Workspace and daemon flows repeatedly load merged config for resolved workspaces, so the full load/validate path is the package’s primary I/O-heavy entry point. | `BenchmarkLoadForHomeWorkspaceOverlay` |
| `ResolveAgent` | `internal/config/provider.go:112` | Session and observer startup repeatedly resolve agent/provider defaults and merge MCP server layers. | `BenchmarkResolveAgentMergedMCPServers` |
| `ParseMCPServersJSON` | `internal/config/mcpjson.go:32` | Global/workspace/agent/skill sidecars all route through strict JSON decode, normalization, and validation. | `BenchmarkParseMCPServersJSONLarge` |
| `HookDeclarations` | `internal/config/hooks.go:73` | Hook registry rebuilds normalize every config + agent hook declaration into dispatcher-ready form. | `BenchmarkHookDeclarationsNormalization` |

### Optimization — Benchmark Results

Baseline command for `before` numbers: `go test -bench=. -benchmem -count=5 ./internal/config/...` in a temporary worktree at `HEAD`, with the benchmark harness plus the dotenv-scoping fix copied in, but without the `provider.go` optimization.
Final command for `after` numbers: `go test -bench=. -benchmem -count=5 ./internal/config/...` in the current workspace after all changes.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkLoadForHomeWorkspaceOverlay` | 116967.0 | 73792.4 | 116481.0 | 72886.8 | not-hot-confirmed-by-benchmark — the security fix slightly reduced allocations, but no dedicated perf-only change was warranted on the broader load path. |
| `BenchmarkResolveAgentMergedMCPServers` | 39147.4 | 91590.4 | 28127.4 | 60623.0 | fixed-with-benchmark |
| `BenchmarkParseMCPServersJSONLarge` | 81741.8 | 96517.4 | 81158.8 | 96519.0 | not-hot-confirmed-by-benchmark — decode/validation remained effectively flat in this pass. |
| `BenchmarkHookDeclarationsNormalization` | 46625.0 | 122112.0 | 46047.6 | 122112.0 | not-hot-confirmed-by-benchmark — normalization stayed flat enough that no separate hook-path optimization was justified. |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| none | — | — | No production `go` statements exist under `internal/config/`. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | ---: | --- | --- | --- | --- |
| none | — | — | — | — | No production channels are declared under `internal/config/`. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| none | — | — | No production `sync.Mutex` or `sync.RWMutex` fields exist under `internal/config/`. |

### Concurrency — Select Audit

No production `select` statements exist under `internal/config/`.

### Security — Threat Model

- Trust boundaries:
  - Local callers in daemon, workspace, session, observe, CLI, and API layers invoke `Load`, `LoadForHome`, `ResolveAgent`, `LoadWorkspaceAgentDefs`, `ParseMCPServersJSON`, and `HookDeclarations`.
  - Workspace-owned files (`<workspace>/.env`, `<workspace>/.agh/config.toml`, `<workspace>/.agh/mcp.json`, workspace agent definitions) are lower-trust than the operator-owned AGH home.
  - Global AGH home files (`$AGH_HOME/config.toml`, `$AGH_HOME/mcp.json`, `$AGH_HOME/agents/...`) are assumed operator-controlled unless the caller explicitly points elsewhere.
- Attacker capabilities:
  - A malicious repository or workspace can control `.env`, `.agh/config.toml`, `.agh/mcp.json`, and `AGENT.md` contents for that workspace.
  - A caller can pass explicit workspace roots and agent definitions that route through this package.
  - A local attacker cannot bypass strict parser validation, but can attempt to influence path selection, provider/MCP declarations, and webhook secret references through configuration inputs.
- In-scope assets:
  - Correct AGH home resolution and isolation across workspace loads.
  - Integrity of provider commands, MCP server declarations, hook declarations, and automation trigger validation.
  - Webhook secret-env binding used by automation trigger configuration.
- Out-of-scope:
  - Compromise of downstream packages after config resolution has completed.
  - OS-level compromise or an already-malicious operator-controlled AGH home.
  - Execution of provider commands after they leave this package and enter runtime/session layers.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/config/config.go:274-301`, `1085-1110`; `internal/config/home.go:59-85` | Workspace `.env` plus `AGH_HOME` lookup when config loads include a workspace root. | `.env` is parsed into a scoped lookup via `godotenv.Read`; process env still wins; `resolveAbsoluteDir` normalizes the selected home path. | Home-path selection for global config/MCP sidecar loading. | LOW — fixed in this pass: workspace `.env` can affect only the current load and no longer mutates process env for later workspaces. |
| `internal/config/automation.go:96-121`, `172-221` | `automation.*` config entries, especially `webhook_secret_env`, from global/workspace `config.toml`. | Strict schema validation, scope binding checks, trigger filter validation, template validation, and scoped env lookup for webhook secret resolution. | Automation trigger acceptance during config validation. | LOW — fixed in this pass: webhook secret env resolution is scoped to the active load, preventing cross-workspace secret bleed. |
| `internal/config/config.go:331-355`; `internal/config/merge.go:206-250` | Global/workspace `config.toml` content. | TOML decode rejects unknown keys; every section re-validates after merge. | Final merged `Config` returned to callers. | LOW — strict decode plus section validation rejects malformed or unknown input. |
| `internal/config/mcpjson.go:32-60` | Global/workspace/agent/skill `mcp.json` sidecar content. | Strict JSON decode, EOF check, duplicate-name rejection after normalization, per-server validation. | `cfg.MCPServers` and later provider/agent MCP merge flows. | LOW — malformed or ambiguous MCP declarations fail closed. |
| `internal/config/agent.go:155-239` | Workspace/global `AGENT.md` content and adjacent `mcp.json` sidecar. | Strict YAML/TOML frontmatter parsing, agent validation, hook validation, MCP sidecar validation, duplicate-name precedence in discovery. | Agent definitions exposed to workspace/session/runtime callers. | LOW — malformed agent definitions fail before runtime resolution. |
| `internal/config/home.go:155-176` | `HOME` from the caller-provided getenv function in `ResolveUserAgentsSkillsDir`. | Path resolution via `ResolvePath`; only used to compute a filesystem location. | User skill-directory path derivation. | REJECTED — operator/process-controlled input, not a lower-trust workspace ingress in this threat model. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| 01 | extreme-software-optimization | medium | `internal/config/provider.go:112` | `ResolveAgent` built effective MCP servers through two nested merge passes, duplicating clone and index work before every agent resolution. | fixed |
| 02 | security-review | high | `internal/config/config.go:274` | Workspace `.env` loading mutated process env, allowing `AGH_HOME` and webhook secret references to bleed into later workspace loads. | fixed |
| 03 | refactoring-analysis | medium | `internal/config/config.go:1` | `config.go` remains a 1111-LOC monolith that mixes schema, defaults, load orchestration, validation, environment resolution, and path helpers. | deferred — cleanly splitting schema/load/validation seams would widen this task into a larger package refactor. |
| 04 | refactoring-analysis | low | `internal/config/merge.go:580` | `merge.go` retains duplicated overlay-application loops and mirrored setter blocks across overlay sections. | deferred — de-duplicating the overlay helpers would reshape merge internals beyond this fix-focused pass. |

## Per-Skill Notes

### refactoring-analysis

- The main structural pressure is file-size concentration rather than many small smells. `config.go` and `merge.go` dominate both LOC and duplication signals.
- The `Config` vs `configOverlay` section inventory duplication is real, but untangling it cleanly would require a larger schema refactor than this task can absorb without widening scope.
- I kept the structural issues as deferred because this pass yielded two higher-value in-scope fixes with tighter blast radius.

### extreme-software-optimization

- Added `internal/config/perf_bench_test.go` so every chosen candidate has a co-located benchmark.
- The strongest signal was `ResolveAgent`, where nested MCP merge passes were doing redundant slice cloning and name indexing.
- Fixed that by collapsing the effective MCP merge into one internal `mergeMCPServerLayers` pass and by pre-sizing cloned slices in the merge path.
- Isomorphism proof for the perf change:
  - Ordering preserved: yes — layers are still applied in `global -> provider -> agent` order.
  - Tie-breaking unchanged: yes — later layers still overwrite same-name servers exactly as the previous nested merge sequence did.
  - Floating-point: N/A.
  - RNG seeds: N/A.
  - Golden outputs: existing MCP merge/agent resolution tests plus full verification remained green.
- `BenchmarkResolveAgentMergedMCPServers` improved from `39147.4 ns/op, 91590.4 B/op` to `28127.4 ns/op, 60623.0 B/op`.
- `LoadForHome`, `ParseMCPServersJSON`, and `HookDeclarations` stayed effectively flat enough that no extra optimization was justified in this pass.

### ubs

- `not-run` due missing skill-runner interface in this session; no manual substitute was used.

### deadlock-finder-and-fixer

- No production goroutines, channels, mutexes, or `select` statements exist in `internal/config/`.
- The package is concurrency-light by construction; the deadlock audit reduced to verifying that no hidden goroutine/channel state had been introduced.

### security-review

- Fixed a high-confidence isolation bug: workspace `.env` files were previously loaded into the process environment, which could influence later config loads outside the originating workspace.
- Source -> sink trace for the fixed issue:
  - Source: workspace `.env` parsed during `Load` / `LoadForHome` in [config.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/config/config.go:274>) and [config.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/config/config.go:1085>).
  - Sinks before the fix: home resolution in [home.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/config/home.go:59>) and webhook secret validation in [automation.go](</Users/pedronauck/Dev/compozy/_worktrees/improvs/internal/config/automation.go:172>).
  - Fix: keep dotenv values in a scoped lookup for the active load instead of mutating global process env.
- Added regressions proving both sides of the bug:
  - `TestLoadUsesDotEnvForAGHHomeWithoutMutatingProcessEnv`
  - `TestLoadForHomeDoesNotLeakDotEnvSecretsAcrossWorkspaceLoads`
- No other high-confidence vulnerabilities were identified after tracing config, mcp.json, AGENT.md, and webhook secret input flows.

## Deferred Items (carry forward)

- **03** — Split `internal/config/config.go` along schema/defaults, load orchestration, and validation/environment seams when a future task can absorb a larger refactor.
- **04** — Consolidate duplicated overlay-application loops and mirrored setter blocks in `internal/config/merge.go` if a follow-up refactor is willing to reshape overlay helpers.

## `make verify`

Final gate command: `make verify`

```text
Found 0 warnings and 0 errors.
Test Files  82 passed (82)
Tests  677 passed (677)
DONE 4460 tests in 2.561s
OK: all package boundaries respected
```

Additional toolchain noise during the successful run:

- repeated Node warnings that `NO_COLOR` was ignored because `FORCE_COLOR` is set
- one macOS linker warning: `ld: warning: -bind_at_load is deprecated on macOS`
