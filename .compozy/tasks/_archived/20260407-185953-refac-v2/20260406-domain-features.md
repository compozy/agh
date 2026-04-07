# Refactoring Analysis: Domain Features (memory, skills, workspace, observe)

**Date**: 2026-04-06
**Analyst**: Claude Opus 4.6 (1M context)
**Scope**: 4 packages, ~43 Go source files, ~5,155 LOC (non-test, current tree)
**Method**: Martin Fowler code smells catalog + SOLID at package level

> **Corrections applied (2026-04-06)**: F-SKL-04/05 (`fileSnapshot`/`snapshotsEqual` duplication) marked as RESOLVED -- `internal/filesnap` already exists and is used by both `skills` and `workspace`. LOC updated. Coupling analysis missing `filesnap` as a dependency of both `skills` and `workspace`.

---

## Executive Summary

| Severity | Count |
|----------|-------|
| P0 Critical | 2 |
| P1 High | 5 |
| P2 Medium | 8 |
| P3 Low | 5 |
| **Total** | **20** |

**Top 5 highest-impact opportunities:**

1. **P0 -- Cross-package duplication of frontmatter parsing, snapshot types, and line-ending normalization** (3 packages duplicate the same logic)
2. **P0 -- `memory` package has 4 distinct responsibilities crammed into one flat package** (Store, Dream/Consolidation, Lock, Assembler)
3. **P1 -- `observe` has excessive efferent coupling** (imports 7 internal packages)
4. **P1 -- `workspace.Resolver` is a God Object** (CRUD + resolution + caching + scanning + config loading in one struct)
5. **P1 -- Context-checking helpers duplicated across 3 packages with trivially different signatures**

---

## Package 1: `internal/memory` (12 files, ~1,300 LOC)

### 1.1 Architectural Boundaries

#### F-MEM-01: Package mixes 4 distinct subdomains
**Severity**: P0 | **Action**: (B) Package-level split

The `memory` package contains four weakly-related responsibilities:

| Subdomain | Files | Responsibility |
|-----------|-------|---------------|
| Store | `store.go`, `types.go`, `document.go` | CRUD for memory files + frontmatter parsing + MEMORY.md index |
| Dream/Consolidation | `dream.go` | Consolidation service with time/session gates, spawner orchestration |
| Lock | `lock.go` | Cross-process PID-based consolidation lock with mtime tracking |
| Assembler | `assembler.go`, `prompt.go`, `staleness.go` | Prompt assembly, staleness helpers, consolidation prompt template |

The `Service` struct in `dream.go` (449 lines) is a self-contained orchestrator that could live in its own subpackage (e.g., `memory/consolidation`). The `ConsolidationLock` (274 lines) is a generic cross-process PID lock that has zero memory-domain knowledge -- it could be extracted to a shared utility.

**Recommendation**: Split into:
- `internal/memory` -- types, store, assembler, staleness
- `internal/memory/consolidation` -- Service, prompt template
- `internal/filelock` or keep lock inlined in consolidation subpackage

#### F-MEM-02: `dream.go` Service scans session directories directly
**Severity**: P1 | **Action**: (D) Inline fix / interface extraction

`scanCompletedSessionsSince()` (lines 306-353) reads `meta.json` files from the sessions directory and unmarshals `persistedSessionMetadata`. This is:
1. Feature envy -- it knows internal session persistence layout
2. Tightly coupled to the filesystem representation of sessions
3. Duplicates knowledge that `store` or `session` packages already own

The `countSessionsSince` field is already a `func(time.Time) (int, error)` -- the default implementation should be injected from `daemon/` rather than hardcoded here.

#### F-MEM-03: `assembler.go` imports both `session` and `workspace`
**Severity**: P2 | **Action**: (D) Inline fix

`Assembler` implements `session.PromptProvider` and accepts `workspace.ResolvedWorkspace`. The `session` import is only for the compile-time interface check (`var _ session.PromptProvider = (*Assembler)(nil)`). The `workspace` import is structural because `PromptSection` accepts `workspacepkg.ResolvedWorkspace`. Both are interface-level dependencies, which is acceptable -- but the direct coupling to `workspace.ResolvedWorkspace` concrete type means memory cannot be tested without workspace types.

### 1.2 Code Smells

#### F-MEM-04: Duplicated `parseFrontmatter` implementation
**Severity**: P0 | **Action**: (C) Extraction

`parseFrontmatter()` exists in three packages with nearly identical logic:
- `memory/store.go:422` -- `func parseFrontmatter(content []byte, dest any) (string, error)`
- `config/agent.go:227` -- `func parseFrontmatter(content []byte, dest any) (string, error)` (identical signature)
- `skills/loader.go:75` -- `func parseFrontmatter(content string) (SkillMeta, string, error)` (string variant)

All three:
1. Normalize line endings
2. Find opening `---`
3. Find closing `---`
4. Unmarshal YAML between delimiters
5. Return body after closing delimiter

**Recommendation**: Extract `internal/frontmatter` package with `Parse[T any](content []byte) (T, string, error)` using generics.

#### F-MEM-05: Duplicated `normalizeLineEndings`
**Severity**: P1 | **Action**: (C) Extraction

Three identical implementations:
- `memory/store.go:455` -- `func normalizeLineEndings(content []byte) []byte`
- `config/agent.go:260` -- `func normalizeLineEndings(content []byte) []byte`
- `skills/loader.go:230` -- `func normalizeLineEndings(content string) string` (string variant)

All are `strings.ReplaceAll(content, "\r\n", "\n")`. Should be extracted alongside frontmatter.

#### F-MEM-06: Duplicated `findClosingDelimiter`
**Severity**: P1 | **Action**: (C) Extraction

Three implementations with minor signature differences:
- `memory/store.go:471` -- `func findClosingDelimiter(content []byte, start int) (int, int, bool)`
- `config/agent.go:276` -- `func findClosingDelimiter(content []byte, start int) (int, int, bool)`
- `skills/loader.go:234` -- `func findClosingDelimiter(content string) (int, int, bool)`

Same algorithm, same control flow. Should move to `internal/frontmatter`.

#### F-MEM-07: `contextErr` vs `checkContext` vs `checkRegistryContext`
**Severity**: P2 | **Action**: (C) Extraction

Three packages have trivially different context-checking helpers:
- `memory/assembler.go:151` -- `func contextErr(ctx context.Context) error` (nil-safe, returns `ctx.Err()`)
- `workspace/helpers.go:98` -- `func checkContext(ctx context.Context) error` (nil = error, returns `ctx.Err()`)
- `skills/registry.go:431` -- `func checkRegistryContext(ctx context.Context) error` (nil = error, returns `ctx.Err()`)

The workspace and skills versions are identical except for error messages. The memory version is lenient on nil ctx. These could be a single helper, but the divergent nil-ctx policy suggests keeping them local unless a shared `contextutil` is warranted.

#### F-MEM-08: `store.go` is 489 lines mixing CRUD, frontmatter parsing, and utility functions
**Severity**: P2 | **Action**: (A) File-level split

`store.go` contains:
- `Store` struct + CRUD methods (Read/Write/Delete/Scan/LoadIndex/EnsureDirs) ~250 lines
- Frontmatter parsing primitives ~70 lines
- Filename/path validation helpers ~50 lines
- Index truncation/UTF-8 helpers ~50 lines
- Line-ending normalization ~10 lines

**Recommendation**: Move parsing to `frontmatter.go` (or shared package), keep CRUD in `store.go`.

### 1.3 Coupling Analysis

| Direction | Dependencies |
|-----------|-------------|
| Afferent (who imports memory) | `daemon`, `cli`, `httpapi`, `udsapi` |
| Efferent (memory imports) | `config`, `workspace`, `session`, `fileutil`, `procutil`, `testutil` |

The `workspace` dependency is warranted -- memory is workspace-scoped. The `session` dependency is only for the `PromptProvider` interface assertion, which is lightweight. The `procutil` dependency is only in `lock.go` for PID liveness -- acceptable.

**Verdict**: Coupling is moderate and mostly warranted. The main concern is `dream.go` scanning session directories directly rather than through an injected interface.

---

## Package 2: `internal/skills` (11 files + `bundled/`, ~1,100 LOC)

### 2.1 Architectural Boundaries

#### F-SKL-01: `registry.go` is 715 lines with mixed responsibilities
**Severity**: P1 | **Action**: (A) File-level split or (B) Package-level split

`registry.go` handles:
1. Global skill loading (bundled + user-level) ~100 lines
2. Workspace skill loading + cache management ~150 lines
3. Deep-clone utilities for skills/metadata ~80 lines
4. Snapshot equality/diffing ~40 lines
5. Source name/path resolution helpers ~60 lines
6. Cache eviction ~20 lines
7. Bundled FS scanning ~70 lines

The clone utilities and snapshot functions are generic enough to extract. The bundled FS scanner could live alongside `loader.go`.

**Recommendation**: File-level split at minimum:
- `registry.go` -- core Registry struct, LoadAll, Get, List, ForWorkspace
- `registry_helpers.go` -- clone, snapshot, source name helpers
- Move `parseBundledSkill` and `scanBundledFS` to `loader.go` or `loader_bundled.go`

#### F-SKL-02: `bundled/` subpackage is a good pattern
**Severity**: P3 (positive observation)

The `bundled/` subpackage cleanly separates the `go:embed` filesystem from the skill parsing logic. This is a pattern that `memory` could follow for its consolidation prompt.

### 2.2 Code Smells

#### F-SKL-03: `errFrontmatterMissing` sentinel shadows memory's sentinel
**Severity**: P2 | **Action**: (C) Extraction

Both `skills/loader.go` and `memory/store.go` define:
```go
errFrontmatterMissing      = errors.New("... missing YAML frontmatter")
errFrontmatterUnterminated = errors.New("... unterminated YAML frontmatter")
```

These would naturally live in a shared `frontmatter` package.

#### ~~F-SKL-04: `fileSnapshot` type duplicated in `skills` and `workspace`~~
**Severity**: ~~P1~~ RESOLVED | **Action**: N/A

> **CORRECTION (2026-04-06)**: This finding is **stale**. The `internal/filesnap` package already exists (60 LOC, `filesnap.go`) and is imported by both `skills` (loader.go, registry.go, watcher.go) and `workspace` (scanner.go, resolver.go, clone.go). No action needed.

#### ~~F-SKL-05: `snapshotsEqual` duplicated between skills and workspace~~
**Severity**: ~~P1~~ RESOLVED | **Action**: N/A

> **CORRECTION (2026-04-06)**: Already resolved by `internal/filesnap`. The shared `snapshotsEqual` function lives in `filesnap.go`.

#### F-SKL-06: `slog.Warn` used directly in `loader.go` instead of injected logger
**Severity**: P3 | **Action**: (D) Inline fix

`loader.go` lines 68, 142, 170, 177 use `slog.Warn()` directly instead of going through the `Registry.logger`. This means warning messages bypass any custom logger configuration.

#### F-SKL-07: `warnUnknownFields` uses global `slog.Warn`
**Severity**: P3 | **Action**: (D) Inline fix

`loader.go:210-228` calls `slog.Warn` directly. Since `ParseSkillFile` doesn't receive a logger, this is a pragmatic choice, but inconsistent with the rest of the package.

### 2.3 Coupling Analysis

| Direction | Dependencies |
|-----------|-------------|
| Afferent (who imports skills) | `daemon`, `cli`, `httpapi`, `udsapi` |
| Efferent (skills imports) | `workspace` (types only), `skills/bundled` |

**Verdict**: Skills has excellent coupling discipline. It only imports `workspace` for the `ResolvedWorkspace` and `SkillPath` types, and its own `bundled/` subpackage. No circular dependencies. The `workspace` dependency is structural and unavoidable since skills are workspace-scoped.

### 2.4 SOLID Analysis

#### F-SKL-08: Registry violates SRP at method granularity
**Severity**: P2 | **Action**: (A) File-level split

`Registry` is responsible for:
1. Loading/refreshing global skills
2. Caching workspace skills
3. Verifying skill content safety
4. Deep-cloning skill metadata
5. Managing disabled-skill policy

Items 3-5 are support concerns. The verification could be a standalone function (it already is: `VerifyContent`), and `processSkill` orchestrates them. This is acceptable for now but worth watching as complexity grows.

---

## Package 3: `internal/workspace` (12 files, ~1,000 LOC)

### 3.1 Architectural Boundaries

#### F-WS-01: `Resolver` is a God Object (CRUD + Resolution + Caching + Scanning + Config)
**Severity**: P1 | **Action**: (B) Package-level split

`Resolver` has 10+ public methods spanning distinct responsibilities:

| Responsibility | Methods | Files |
|---------------|---------|-------|
| Resolution | `Resolve`, `ResolveOrRegister` | `resolver.go` |
| CRUD | `Register`, `Unregister`, `Update`, `List`, `Get` | `resolver_crud.go` |
| Caching | `Invalidate`, `evictExpiredLocked`, `canReuse` | `resolver.go` |
| Scanning | `scanWorkspace`, `scanAgentSource`, `scanSkillSource` | `scanner.go` |
| Config | `buildResolvedWorkspace`, `loadConfig` | `resolver.go` |
| Naming | `nextWorkspaceName` | `resolver_crud.go` |

This violates SRP -- adding a new CRUD operation requires touching the same struct that manages caching and filesystem scanning.

**Recommendation**: Consider splitting into:
- `workspace.Resolver` -- resolution + caching only
- `workspace.Manager` or keep CRUD on Resolver but separate scanning into a `workspace/scan` subpackage

Alternatively, since the codebase philosophy is "pragmatic flat with discipline," the current file-level split (`resolver.go` + `resolver_crud.go` + `scanner.go`) is a reasonable compromise. But watching the method count -- if it grows further, extraction is warranted.

#### F-WS-02: `clone.go` deeply clones `config.Config` internals
**Severity**: P2 | **Action**: (D) Inline fix or (C) Extraction

`clone.go` (150 lines) manually clones:
- `Config` with all nested fields
- `ProviderConfig` + `MCPServer` arrays
- `AgentDef` slices

This creates tight coupling between `workspace` and `config` struct layouts. If `config.Config` gains new fields, `clone.go` silently drops them. This is a maintenance trap.

**Recommendation**: Either:
1. Move clone functions to `config` package (where they belong, since they know the struct intimately)
2. Use a generic deep-copy approach
3. Add a `Clone()` method on `Config` in the `config` package

#### F-WS-03: `workspace.go` defines domain models + interface + errors -- clean separation
**Severity**: P3 (positive observation)

The `workspace.go` file cleanly defines:
- Domain types (`Workspace`, `ResolvedWorkspace`, `SkillPath`)
- Resolver interface (`WorkspaceResolver`)
- Sentinel errors

This is good Go-style interface-where-consumed design.

### 3.2 Code Smells

#### F-WS-04: `canonicalRoot` duplicates logic in `refreshRootDir`
**Severity**: P2 | **Action**: (D) Inline fix

Both `canonicalRoot()` (helpers.go:26-62) and `refreshRootDir()` (resolver.go:244-286) perform:
1. `os.Stat` to check existence
2. `filepath.EvalSymlinks` to resolve symlinks
3. `filepath.Abs` to get absolute path

`refreshRootDir` is essentially `canonicalRoot` + update-if-changed. The stat+eval+abs sequence could be extracted into a single helper.

#### F-WS-05: `options.go` calls `aghconfig.ResolveHomePaths()` in `resolveOptions`
**Severity**: P2 | **Action**: (D) Inline fix

`resolveOptions()` calls `aghconfig.ResolveHomePaths()` as a default, which reads `$HOME` and creates directories. This side effect happens during `NewResolver()`, making it hard to test without filesystem access. The home paths should always be explicitly provided in tests (and they are, via `WithHomePaths`), but the default fallback adds implicit coupling.

### 3.3 Coupling Analysis

| Direction | Dependencies |
|-----------|-------------|
| Afferent (who imports workspace) | `memory`, `skills`, `observe`, `session`, `daemon`, `cli`, `httpapi`, `udsapi` |
| Efferent (workspace imports) | `config`, `store` |

**Verdict**: `workspace` has very high afferent coupling (8+ dependents) because it defines the `Workspace`, `ResolvedWorkspace`, and `WorkspaceResolver` types that everyone consumes. This is appropriate for a domain model package. Its efferent coupling is low (2 packages), which is healthy.

**Risk**: Changes to `Workspace`/`ResolvedWorkspace` types propagate widely. The types are stable enough that this is acceptable.

---

## Package 4: `internal/observe` (8 files, ~700 LOC)

### 4.1 Architectural Boundaries

#### F-OBS-01: `observe` imports 7 internal packages -- highest efferent coupling
**Severity**: P1 | **Action**: (D) Inline fix / interface narrowing

`observe` imports:
1. `acp` -- for `AgentEvent` type and event type constants
2. `session` -- for `Notifier` interface, `Session` type, `SessionState`, `SessionInfo`
3. `store` -- for `SessionRegistry`, query/result types
4. `config` -- for `HomePaths`, `Config`, `AgentDef`, config loading
5. `workspace` -- for `WorkspaceResolver`, `ResolvedWorkspace`, `ErrAgentNotAvailable`
6. `version` -- for `Info` type and `Current()` function
7. `testutil` -- test-only

This is the most coupled package in the domain group. However, observe is architecturally a "leaf" that consumes data from many sources to produce observability output. High efferent coupling is somewhat inherent to observer/metrics packages.

**Concern**: The `defaultPermissionModeResolver` function (observer.go:335-378) is 43 lines of config + workspace + agent resolution logic that belongs in `daemon/` or a dedicated resolver, not in `observe`. It loads configs, resolves workspaces, and matches agents -- all composition-root behavior.

**Recommendation**: Move `defaultPermissionModeResolver` to `daemon/` and inject it via `WithPermissionModeResolver`.

#### F-OBS-02: `observer.go` New() auto-opens database -- side effect in constructor
**Severity**: P2 | **Action**: (D) Inline fix

`New()` (lines 131-191) opens a SQLite database if no registry is provided:
```go
if observer.registry == nil {
    registry, err := store.OpenGlobalDB(ctx, observer.homePaths.DatabaseFile)
    ...
    observer.registry = registry
}
```

This makes the constructor impure and hard to test in isolation. The happy path in production always provides a registry from `daemon/`, so this fallback is convenience code that complicates the constructor.

**Recommendation**: Make `Registry` a required parameter. Remove the auto-open fallback. Let `daemon/` always provide the registry.

### 4.2 Code Smells

#### F-OBS-03: `OnAgentEvent` is a long function (76 lines)
**Severity**: P2 | **Action**: (D) Inline fix / Extract Function

`OnAgentEvent` (observer.go:238-314) handles three distinct concerns in sequence:
1. Guard checks (empty session, unknown session, empty type) -- 15 lines
2. Write event summary -- 15 lines
3. Aggregate token usage (conditional) -- 20 lines
4. Write permission log (conditional) -- 25 lines

Each section could be a private method: `writeEventSummary`, `aggregateUsage`, `writePermissionLog`.

#### F-OBS-04: `summarizeEvent` has duplicated candidate fields
**Severity**: P3 | **Action**: (D) Inline fix

`summarizeEvent()` (observer.go:413-440) builds a `candidates` slice, then for permission events prepends a different slice that partially overlaps:
```go
candidates := []string{Text, Title, Error, Resource, StopReason, ToolCallID}
if event.Type == "permission" {
    candidates = append([]string{Title, Resource, Decision}, candidates...)
}
```

`Title` and `Resource` appear twice. Minor, but the prepend+original creates unnecessary allocation. A cleaner approach is a single priority-ordered slice per event type.

#### F-OBS-05: `health.go` mixes active-count logic for two sources
**Severity**: P2 | **Action**: (D) Inline fix

`activeCounts()` (health.go:59-86) has two branches:
1. When `sessionSource` is set: count from in-memory list
2. When not: query from registry

Both return `(count, count, nil)` where `ActiveSessions == ActiveAgents`. This conflation of sessions and agents suggests the metric model is incomplete. If agents and sessions ever diverge (multiple agents per session), this will silently return wrong data.

### 4.3 Coupling Analysis

| Direction | Dependencies |
|-----------|-------------|
| Afferent (who imports observe) | `daemon` (only) |
| Efferent (observe imports) | `acp`, `session`, `store`, `config`, `workspace`, `version`, `testutil` |

**Verdict**: Observe has the correct architectural position -- it's a high-level package imported only by the composition root (`daemon`). Its high efferent coupling is the price of being an observer that needs to understand all domain events. The main refactoring opportunity is moving `defaultPermissionModeResolver` out.

### 4.4 SOLID Analysis

#### F-OBS-06: `Registry` interface is appropriately narrow but wraps a fat interface
**Severity**: P3 | **Action**: None needed

```go
type Registry interface {
    store.SessionRegistry
    Path() string
}
```

`store.SessionRegistry` itself is a large interface (12+ methods). However, `observe` uses most of them (RegisterSession, UpdateSessionState, WriteEventSummary, UpdateTokenStats, WritePermissionLog, ListSessions, ListEventSummaries, ListTokenStats, ListPermissionLog, ReconcileSessions). This is a case where ISP violation is acceptable because the interface genuinely represents a cohesive persistence surface.

---

## Cross-Package Analysis

### DRY Violations Summary

| Duplicated Element | Packages | Lines Duplicated | Recommendation |
|---|---|---|---|
| `parseFrontmatter` | memory, config, skills | ~45 lines x3 | Extract `internal/frontmatter` |
| `normalizeLineEndings` | memory, config, skills | ~3 lines x3 | Move to frontmatter pkg |
| `findClosingDelimiter` | memory, config, skills | ~20 lines x3 | Move to frontmatter pkg |
| ~~`fileSnapshot` struct~~ | ~~skills, workspace~~ | ~~RESOLVED~~ | Already in `internal/filesnap` |
| ~~`snapshotsEqual`~~ | ~~skills, workspace~~ | ~~RESOLVED~~ | Already in `internal/filesnap` |
| `errFrontmatterMissing/Unterminated` | memory, skills | ~2 lines x2 | Move to frontmatter pkg |
| Context check helpers | memory, workspace, skills | ~5 lines x3 | Consider shared helper or accept |

### Subpackage Grouping Recommendations

The user asked whether these packages should form a subpackage group. My assessment:

**No -- these should NOT be grouped under a single parent package.** Here is why:

1. **`workspace` is foundational** -- it defines domain types consumed by memory, skills, observe, session, and daemon. Nesting it under a group would create awkward import paths.

2. **`memory` and `skills` are peers** that both consume `workspace` but don't depend on each other. They share a pattern (prompt provider) but different domains.

3. **`observe` is a cross-cutting concern** that consumes all other packages. It belongs at the same level, not grouped with its dependencies.

**What SHOULD happen instead:**

1. **Extract shared utilities** that these packages duplicate:
   - `internal/frontmatter` -- YAML frontmatter parsing (eliminates the worst cross-package duplication)
   - Consider adding `fileSnapshot` + `snapshotsEqual` to `internal/fileutil` or a new `internal/snapshot` package

2. **Use subpackages within large packages** (follow the `skills/bundled` pattern):
   - `memory/consolidation` -- dream service, consolidation lock, consolidation prompt
   - Leave `workspace` as-is (the file-level split is adequate for current size)

3. **Move composition logic out of observe**:
   - `defaultPermissionModeResolver` belongs in `daemon/`

### Suggested Refactoring Order

| Priority | Effort | Finding | Action |
|----------|--------|---------|--------|
| 1 | Moderate | F-MEM-04/05/06 + F-SKL-03: Extract `internal/frontmatter` | Eliminates ~200 lines of duplication across 3 packages |
| 2 | Moderate | F-MEM-01: Split `memory/consolidation` subpackage | Reduces memory package surface area by ~50% |
| ~~3~~ | ~~Low~~ | ~~F-SKL-04/05: Extract shared `fileSnapshot` + `snapshotsEqual`~~ | **RESOLVED**: `internal/filesnap` already exists |
| 4 | Low | F-OBS-01: Move `defaultPermissionModeResolver` to daemon | Reduces observe coupling by 2 packages |
| 5 | Low | F-OBS-03: Extract sub-methods from `OnAgentEvent` | Improves readability |
| 6 | Moderate | F-WS-02: Move `Config.Clone()` to config package | Fixes maintenance trap |
| 7 | Low | F-SKL-01: File-level split of `registry.go` | Improves navigation |
| 8 | Low | F-MEM-02: Inject session counter from daemon | Removes feature envy |

---

## Appendix: File Inventory

### memory (12 files)
| File | Lines | Role |
|------|-------|------|
| `types.go` | 115 | MemoryType, Scope, MemoryHeader domain types |
| `store.go` | 489 | Store CRUD, frontmatter parsing, index truncation |
| `assembler.go` | 157 | Prompt assembly, PromptProvider impl |
| `dream.go` | 449 | Consolidation Service, gates, spawner orchestration |
| `staleness.go` | 43 | Age/freshness helpers |
| `prompt.go` | 53 | Consolidation prompt template |
| `document.go` | 26 | ParseHeader, ConsolidationLockPath public helpers |
| `lock.go` | 274 | ConsolidationLock PID-based cross-process lock |
| `store_test.go` | 743 | Store + staleness tests |
| `assembler_test.go` | 278 | Assembler tests |
| `dream_test.go` | 859 | Service + session scanner tests |
| `lock_test.go` | 390 | ConsolidationLock tests |

### skills (13 files)
| File | Lines | Role |
|------|-------|------|
| `types.go` | 72 | SkillMeta, Skill, SkillSource, Warning types |
| `catalog.go` | 137 | CatalogProvider, BuildCatalog XML renderer |
| `registry.go` | 715 | Registry, caching, loading, cloning, helpers |
| `loader.go` | 296 | ParseSkillFile, frontmatter, directory scanner |
| `verify.go` | 135 | Content verification patterns |
| `watcher.go` | 265 | Poll-based file watcher for global skills |
| `bundled/embed.go` | 16 | go:embed FS for bundled skills |
| `catalog_test.go` | 152 | Catalog tests |
| `registry_test.go` | 862 | Registry tests |
| `loader_test.go` | ~410 | Loader + scanner tests |
| `verify_test.go` | ~200 | Verification pattern tests |
| `watcher_test.go` | ~250 | Watcher tests |
| `bundled/bundled_test.go` | ~30 | Bundled FS tests |

### workspace (12 files)
| File | Lines | Role |
|------|-------|------|
| `workspace.go` | 58 | Domain types, interface, errors |
| `resolver.go` | 276 | Resolve, ResolveOrRegister, caching |
| `resolver_crud.go` | 204 | Register, Unregister, Update, List, Get |
| `scanner.go` | 253 | Workspace filesystem scanning |
| `store.go` | 15 | WorkspaceStore interface |
| `naming.go` | 26 | UniqueWorkspaceName |
| `options.go` | 112 | Functional options for Resolver |
| `clone.go` | 150 | Deep-clone helpers for Config/Workspace/Agent |
| `helpers.go` | 145 | canonicalRoot, generateID, errorType, etc. |
| `workspace_test.go` | ~200 | Domain type tests |
| `resolver_test.go` | ~500 | Resolver tests |
| `resolver_integration_test.go` | ~300 | Integration tests |

### observe (8 files)
| File | Lines | Role |
|------|-------|------|
| `observer.go` | 470 | Observer struct, Notifier impl, event handlers |
| `health.go` | 140 | Health metrics, DB size helpers |
| `query.go` | 23 | Query delegation to registry |
| `reconcile.go` | 91 | Session reconciliation from filesystem |
| `observer_test.go` | 488 | Observer unit tests |
| `helpers_test.go` | 474 | Permission resolver, health, helper tests |
| `reconcile_test.go` | 201 | Reconciliation tests |
| `observer_integration_test.go` | 82 | Full flow integration test |
