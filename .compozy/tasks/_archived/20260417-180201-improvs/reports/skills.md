# Improvements Report — internal/skills

## Skill Invocation Log

| Skill | Status | Evidence / Artifact Reference |
| --- | --- | --- |
| refactoring-analysis | run | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run | benchmarks in `internal/skills/perf_bench_test.go`, numbers below |
| ubs | not-run | Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool. |
| deadlock-finder-and-fixer | run | goroutine/channel/mutex/select inventories below |
| security-review | run | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

Output from `gocyclo -over 0 internal/skills | sort -rn | head -10`:

| Complexity | Function | File |
| --- | --- | --- |
| 20 | `scanDirectoryWithSnapshots` | `internal/skills/loader.go:141` |
| 20 | `TestRegistrySetEnabled` | `internal/skills/registry_test.go:1512` |
| 16 | `validateSkillResourceSpec` | `internal/skills/resource.go:90` |
| 15 | `TestSkillTypesSupportMarketplaceDeclarations` | `internal/skills/registry_test.go:1125` |
| 14 | `TestSkillResourceCodecPreservesProvenanceAndSidecarMCP` | `internal/skills/resource_test.go:92` |
| 13 | `TestNewWatcherSeedsSnapshotsFromRegistryLoadAll` | `internal/skills/watcher_test.go:196` |
| 12 | `scanBundledFS` | `internal/skills/registry_snapshot.go:217` |
| 12 | `TestRegistryConcurrentGetAndListDoNotDeadlock` | `internal/skills/registry_test.go:808` |
| 12 | `TestMCPResolverResolveAutoApprovesTrustedSources` | `internal/skills/mcp_test.go:30` |
| 12 | `TestCloneSkillDeepCopiesExtendedFields` | `internal/skills/registry_test.go:1250` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| --- | ---: | --- |
| `internal/skills/loader.go` | 697 | Skill file parsing, frontmatter decoding, AGH metadata decoding, filesystem scan limits, and hook/MCP normalization live in one unit. |
| `internal/skills/registry.go` | 800 | Global/workspace loading, provenance enforcement, runtime overlays, resource projection, and enable/disable mutations are co-located in one runtime-heavy file. |

### Refactoring — Duplication

Baseline output from `dupl -plumbing -t 60 internal/skills`:

- scanned — no duplication output above the configured threshold

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| --- | --- | --- | --- |
| `mergedSkillList` | `internal/skills/registry_snapshot.go:49` | Allocation-heavy clone/merge path behind `Registry.List`, `Registry.ForWorkspace`, API skill listing, session startup, and daemon hook/runtime assembly (`internal/api/core/skills.go:41`, `internal/session/manager_lifecycle.go:193`, `internal/daemon/hooks_bridge.go:946`). | `BenchmarkMergedSkillList` |
| `BuildCatalog` | `internal/skills/catalog.go:64` | Prompt assembly path for active skills through `CatalogProvider` during daemon/session prompt composition (`internal/daemon/boot.go:271`). | `BenchmarkBuildCatalog` |
| `(*MCPResolver).Resolve` | `internal/skills/mcp.go:36` | Active-skill MCP resolution happens during session startup before merging with config MCP servers (`internal/session/manager_lifecycle.go:202`, `internal/daemon/boot.go:270`). | `BenchmarkMCPResolverResolve` |
| `ComputeDirectoryHash` | `internal/skills/provenance.go:46` | Marketplace installs and provenance verification hash entire skill directories during install/update and tamper checks (`internal/cli/skill_marketplace.go:219`, `internal/cli/skill_marketplace.go:324`, `internal/skills/registry.go:555`). | `BenchmarkComputeDirectoryHash` |
| `scanDirectoryWithSnapshots` | `internal/skills/loader.go:141` | Filesystem scan/snapshot loop is used by global skill discovery, reloads, and watcher polling (`internal/skills/registry.go:461`, `internal/skills/watcher.go:169`). | `BenchmarkScanDirectoryWithSnapshots` |
| `(*Registry).ForWorkspace` (cached path) | `internal/skills/registry.go:147` | Workspace skill projection runs in CLI/API/session/daemon runtime paths whenever active skills are requested (`internal/cli/skill_workspace.go:62`, `internal/api/core/skills.go:41`, `internal/session/manager_lifecycle.go:193`). | `BenchmarkRegistryForWorkspaceCached` |

### Optimization — Benchmark Results

Baseline `before` command: `go test -bench=. -benchmem -count=5 ./internal/skills/...`

Final `after` command: `go test -bench=. -benchmem -count=5 ./internal/skills/...`

Values below use the median of 5 runs from `/tmp/skills-bench-before.txt`.

| Benchmark | Before ns/op | Before B/op | After ns/op | After B/op | Decision |
| --- | ---: | ---: | ---: | ---: | --- |
| `BenchmarkMergedSkillList` | 142446 | 313768 | 138749 | 300673 | fixed-with-benchmark |
| `BenchmarkBuildCatalog` | 74829 | 338432 | 28143 | 174593 | fixed-with-benchmark |
| `BenchmarkMCPResolverResolve` | 55378 | 56023 | 55951 | 56149 | not-hot-confirmed-by-benchmark |
| `BenchmarkComputeDirectoryHash` | 6500580 | 1298988 | 6374661 | 465453 | fixed-with-benchmark |
| `BenchmarkScanDirectoryWithSnapshots` | 2493749 | 344856 | 2500219 | 344856 | not-hot-confirmed-by-benchmark |
| `BenchmarkRegistryForWorkspaceCached` | 310267 | 241048 | 306469 | 234480 | fixed-with-benchmark |

### UBS Invocation Output

`not-run` — Skill runner unavailable in this environment; this session exposes skill instructions but no dedicated UBS invocation tool.

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --- | --- | --- | --- |
| `none` | `n/a` | `n/a` | No production `go` statements are declared inside `internal/skills/`; callers run long-lived entrypoints such as `Watcher.Start` in their own goroutines (`internal/daemon/boot.go:1404`). |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --- | --- | --- | --- | --- | --- |
| `none` | `n/a` | `n/a` | `n/a` | `n/a` | No production channels are declared in `internal/skills/`. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --- | --- | --- | --- |
| `internal/skills/registry.go:30` | read-heavy (`sync.RWMutex`) | Global skill snapshots, workspace cache state, disabled overlays, and resource-authority projections on `Registry` | Guards all mutable registry state; long-running filesystem work happens outside the lock. |
| `internal/skills/watcher.go:39` | write-heavy (`sync.Mutex`) | Watcher initialization flag and file-snapshot map | Held only while comparing/committing snapshot maps. |

### Concurrency — Select Audit

| File:Line | Notes |
| --- | --- |
| `internal/skills/watcher.go:106` | Poll loop waits on `ctx.Done()` or `ticker.C`, so the only production `select` is context-aware. |

### Security — Threat Model

- Trust boundaries:
  - `internal/skills` sits between operator/workspace-controlled skill files and the in-memory runtime surfaces consumed by the daemon, CLI, API, and session manager.
  - It also sits between installed marketplace skill directories and downstream hook/MCP/resource consumers through parsed metadata and provenance records.
- Attacker capabilities:
  - A local operator or workspace author can control skill file paths, `SKILL.md` content, `.mcp.json` sidecars, and workspace-provided skill roots.
  - A malicious marketplace package can control installed skill directory contents and provenance sidecars on disk after install/update.
  - Attackers do not directly execute commands inside this package; execution of hooks/MCP commands is deferred to other packages.
- In-scope assets:
  - Safe parsing and normalization of skill metadata from filesystem and bundled sources.
  - Safe provenance loading and hash verification for marketplace-installed skills.
  - Correct source precedence and runtime projection of global/workspace skill sets.
  - Avoiding path traversal or unintended file access from bundled/local skill content loading helpers.
- Out-of-scope:
  - Actual execution of hook commands or MCP processes after this package returns parsed declarations.
  - Authorization decisions about who may call CLI/API endpoints that request skill content.
  - Trust policy for marketplace hook/MCP consent beyond the declarations and provenance this package surfaces.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --- | --- | --- | --- | --- |
| `internal/skills/loader.go:44` | Local or workspace-controlled `SKILL.md` paths and markdown content passed to `ParseSkillFile` / `ReadSkillContent` from CLI, registry, or tests | `filepath.Abs`, `os.ReadFile`, `frontmatter.Split`, YAML decoding with known-field checks for hooks, and explicit required-name/command/event validation | Parsed `Skill` metadata returned to registry/API/CLI callers | LOW — rejected; the package parses/normalizes content and never executes instructions or commands itself. |
| `internal/skills/mcp_sidecar.go:14` | Local `.mcp.json` files adjacent to skill directories | `aghconfig.LoadMCPServersJSONFile` / `ParseMCPServersJSON`, trimmed names/commands, env-map normalization, overlay-by-name logic | `Skill.MCPServers` declarations later consumed by runtime configuration | LOW — rejected; this package only parses declarations, and marketplace trust filtering happens later in `MCPResolver.Resolve`. |
| `internal/skills/registry_workspace_cache.go:67` | Workspace-resolver skill directories and declared source kinds (`resolved.Skills`) | `skillSourceFromWorkspacePath`, trim/empty checks, `filepath.Join(..., SKILL.md)`, `filesnap.FromPath`, and sidecar snapshot collection | Workspace skill loading and cache invalidation in `Registry.ForWorkspace` | LOW — rejected; workspace paths are treated as local filesystem inputs and are not interpolated into external commands or network requests by this package. |
| `internal/skills/provenance.go:115` | Marketplace-managed `.agh-meta.json` sidecars and skill directory contents | Required-field validation in `validateSidecarProvenance`, absolute-root resolution, sorted walk order, sidecar exclusion, and mismatch rejection in `Registry.verifyMarketplaceSkill` (`internal/skills/registry.go:546`) | Marketplace source promotion and tamper blocking | LOW — rejected; malformed or tampered sidecars fail closed and prevent marketplace promotion. |
| `internal/skills/bundled/content.go:23` | Skill name requested for bundled content loading | `strings.TrimSpace`, `validSkillName`, single-component path enforcement, embedded-FS read, frontmatter split | Bundled markdown body returned to session/CLI callers | LOW — rejected; the helper rejects path separators and only reads embedded files. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| --- | --- | --- | --- | --- | --- |
| `SKILLS-REF-001` | refactoring-analysis | medium | `internal/skills/loader.go:141` | `loader.go` still concentrates scan, parse, AGH metadata decode, and hook/MCP normalization in a single 697-LOC unit. | deferred |
| `SKILLS-REF-002` | refactoring-analysis | medium | `internal/skills/registry.go:147` | `registry.go` still combines global/workspace loading, provenance enforcement, resource-authority projection, and runtime enable/disable mutation in one 800-LOC unit. | deferred |
| `SKILLS-OPT-001` | extreme-software-optimization | low | `internal/skills/registry_snapshot.go:49` | `mergedSkillList` built an extra merged map before sorting/cloning, adding avoidable allocations to global/workspace skill projection. | fixed |
| `SKILLS-OPT-002` | extreme-software-optimization | low | `internal/skills/catalog.go:111` | `truncateCatalogDescription` allocated a rune slice for every description even when no truncation was required. | fixed |
| `SKILLS-OPT-003` | extreme-software-optimization | low | `internal/skills/provenance.go:160` | Directory hashing read each regular file fully into memory before hashing, inflating marketplace provenance verification allocations. | fixed |
| `SKILLS-OPT-004` | extreme-software-optimization | low | `internal/skills/mcp.go:36` | MCP server resolution remained effectively flat under benchmark and did not justify extra bucketing/caching complexity in this pass. | wontfix |
| `SKILLS-OPT-005` | extreme-software-optimization | low | `internal/skills/loader.go:141` | `scanDirectoryWithSnapshots` remained filesystem-bound in the full benchmark run, so extra complexity would not pay for itself in this pass. | wontfix |

## Per-Skill Notes

### refactoring-analysis

- `dupl -plumbing -t 60 internal/skills` produced no duplication matches above the configured threshold, so the structural findings are concentrated in file size / responsibility spread rather than repeated blocks.
- `loader.go` and `registry.go` remain the only non-test files over 300 LOC. Breaking either file apart cleanly would require a broader design step than this task’s package-local improvements pass.
- The new `cloneSortedSkillList` helper in `registry_snapshot.go` is a narrow helper extracted to support the measured merge-path optimization, not a new cross-package abstraction.

### extreme-software-optimization

- Baseline and final medians come from the exact same full command: `go test -bench=. -benchmem -count=5 ./internal/skills/...`.
- `BenchmarkMergedSkillList` improved from `142446 ns/op, 313768 B/op` to `138749 ns/op, 300673 B/op` after removing the intermediary merged map and sorting/deduplicating names directly.
- `BenchmarkRegistryForWorkspaceCached` improved from `310267 ns/op, 241048 B/op` to `306469 ns/op, 234480 B/op` from the same merge-path fix because cached workspace projection still flows through `mergedSkillList`.
- `BenchmarkBuildCatalog` improved from `74829 ns/op, 338432 B/op` to `28143 ns/op, 174593 B/op` after making truncation allocation-free for descriptions that do not cross the 200-rune limit.
- `BenchmarkComputeDirectoryHash` improved from `6500580 ns/op, 1298988 B/op` to `6374661 ns/op, 465453 B/op` after replacing whole-file reads with a reusable scratch-buffer read loop.
- `BenchmarkMCPResolverResolve` and `BenchmarkScanDirectoryWithSnapshots` stayed effectively flat in the required full benchmark run, so I left them alone.

#### Change: remove the intermediary merge map in `mergedSkillList`
- Ordering preserved: yes — names are still sorted lexically before clone.
- Tie-breaking unchanged: yes — workspace entries still override globals on duplicate names because the post-sort lookup checks the workspace map first.
- Floating-point: N/A
- RNG seeds: N/A
- Golden outputs: `go test ./internal/skills/...` and the existing workspace/global override tests passed after the change.

#### Change: make catalog truncation allocation-free for non-truncated descriptions
- Ordering preserved: yes — skill ordering still comes from the existing lexical sort in `BuildCatalog`.
- Tie-breaking unchanged: yes — description truncation does not affect skill selection or ordering.
- Floating-point: N/A
- RNG seeds: N/A
- Golden outputs: `go test ./internal/skills/...` passed, including the new unicode truncation regression in `TestBuildCatalogTruncatesUnicodeDescriptionsAtRuneBoundary`.

#### Change: stream hash input through one reusable buffer in `ComputeDirectoryHash`
- Ordering preserved: yes — entry order remains the sorted relative-path list.
- Tie-breaking unchanged: yes — the exact metadata prefixes, file contents, symlink targets, and trailing separators are unchanged.
- Floating-point: N/A
- RNG seeds: N/A
- Golden outputs: `go test ./internal/skills/...` passed, including the existing hash-stability / mismatch regression tests in `provenance_test.go`.

### ubs

- `not-run` due missing dedicated skill-runner support in this environment; no manual substitute will be used.

### deadlock-finder-and-fixer

- Inventory complete: no production goroutine launches, no production channels, one `sync.RWMutex` on `Registry`, one `sync.Mutex` on `Watcher`, and one production `select` loop with `ctx.Done()`.
- No concrete deadlock or goroutine-leak path survived validation. The package’s concurrency surface is small and bounded to cache/snapshot guards plus the watcher poll loop.

### security-review

- No HIGH-confidence or MEDIUM-confidence finding survived the threat-model review.
- The highest-risk-looking surfaces are skill file metadata, `.mcp.json` sidecars, workspace skill paths, and provenance sidecars, but this package only parses/normalizes them and fails closed on malformed marketplace metadata.
- Hook and MCP command execution are intentionally out of scope for this package; trust/consent enforcement for those declarations happens later in the daemon/runtime layers.

## Deferred Items (carry forward)

- **`SKILLS-REF-001`** — Split `loader.go` only when there is appetite for a focused decomposition of scan/parse/metadata responsibilities without mixing that design work into this pass.
- **`SKILLS-REF-002`** — Split `registry.go` only when a follow-up task can separate loading, resource-authority projection, and runtime mutation without changing package APIs mid-pass.

## `make verify`

Command: `make verify`

Exit code: `0`

Excerpt:

```text
Found 0 warnings and 0 errors.
RUN  v4.1.4 /Users/pedronauck/Dev/compozy/_worktrees/improvs/web
✓ built in 387ms
# github.com/golangci/golangci-lint/v2/cmd/golangci-lint
0 issues.
DONE 4495 tests in 27.768s
OK: all package boundaries respected
```

Observed non-blocking toolchain/environment warnings on this macOS setup:

- repeated Node warning: `The 'NO_COLOR' env is ignored due to the 'FORCE_COLOR' env being set.`
- linker warning while building the vendored linter binary: `ld: warning: -bind_at_load is deprecated on macOS`
