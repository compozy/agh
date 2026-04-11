# TechSpec: Kodebase Refactoring — Code Quality & Structural Improvements

## Executive Summary

This TechSpec documents refactoring opportunities identified by static analysis of the AGH codebase using kodebase vault inspection across six dimensions: cyclomatic complexity, dead code, code smells, coupling/instability, circular dependencies, and architecture health. The codebase is structurally sound — zero circular dependencies, clean DAG, proper composition root discipline — but carries complexity hotspots, god files, duplicated logic, dead exports, and coupling concentrations that will compound as the system grows. The refactoring targets are organized into three priority tiers with a recommended attack order.

## Codebase Health Baseline

| Metric | Value |
|--------|-------|
| Total files | 516 |
| Total symbols | 5,296 |
| Total relations | 16,313 |
| Go files | 290 |
| TS/TSX files | 226 |
| Circular dependencies | 0 |
| Highest cyclomatic complexity | 54 (`boot()`) |
| Highest blast radius | 89 (`newHandlers`) |
| Confirmed dead Go exports | 5 |
| Test-only production exports | 5 |
| Unused UI components | 16 |

## P0 — Critical Refactoring Targets

### 1. `boot()` — Monolithic Daemon Bootstrap

- **File**: `internal/daemon/boot.go`
- **Complexity**: 54 | **LOC**: 351
- **Problem**: Sequential initialization of ~15 subsystems (config, logger, memory, skills, lock, registry, workspace, dream, hooks, observer, servers) with deeply nested `if cfg.X.Enabled` branches and nil-guarding every optional subsystem. Error-handling cleanup callbacks accumulate in a slice, further inflating branch count.
- **Blast radius**: High — composition root, changes here ripple through the entire daemon startup path.

**Refactoring approach**: Extract initialization into phase methods:

```go
func (d *Daemon) boot(ctx context.Context) error {
    cfg, err := d.bootConfig(ctx)
    if err != nil { return err }
    mem, cleanupMem, err := d.bootMemory(ctx, cfg)
    if err != nil { return err }
    skills, cleanupSkills, err := d.bootSkills(ctx, cfg)
    if err != nil { return err }
    servers, cleanupServers, err := d.bootServers(ctx, cfg)
    if err != nil { return err }
    // ... register cleanups, return
}
```

Each phase method returns its deps and cleanup function. Main `boot()` becomes a ~15-line pipeline calling phases in sequence.

### 2. `Create` / `Resume` Duplication

- **File**: `internal/session/manager_lifecycle.go`
- **`Create`**: 128 LOC | **`Resume`**: 154 LOC | **Shared**: ~80 duplicated lines
- **Problem**: Both follow an identical sequence: validate context, resolve workspace, resolve agent, build `startupPrompt`, build `SessionContext`, `reserve`, `openStore`, construct `Session` struct, build `acp.StartOpts`, `writeMeta`, `driver.Start`, `activateAndWatch`. The only differences are metadata source (opts vs stored meta) and the hook name.

**Refactoring approach**: Extract a shared `startSession(ctx, sessionSetup)` helper method. `Create` and `Resume` become thin preambles that prepare a `sessionSetup` struct from their respective inputs, then delegate to the common path. Eliminates ~80 lines of near-identical code and ensures future lifecycle changes are applied once.

### 3. `handleInbound()` — Repetitive JSON-RPC Dispatch

- **File**: `internal/acp/handlers.go`
- **Complexity**: 28 | **LOC**: 91
- **Problem**: Switch statement dispatching 9 JSON-RPC methods. Every case repeats the identical unmarshal-call-error pattern.

**Refactoring approach**: Replace the switch with a handler registry map:

```go
type handlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

var handlers = map[string]handlerFunc{
    "session/started":   h.handleSessionStarted,
    "message/create":    h.handleMessageCreate,
    "tool/execute":      h.handleToolExecute,
    // ...
}

func (h *Handlers) handleInbound(ctx context.Context, method string, params json.RawMessage) (any, error) {
    fn, ok := handlers[method]
    if !ok {
        return nil, fmt.Errorf("unknown method: %s", method)
    }
    return fn(ctx, params)
}
```

Reduces cyclomatic complexity from 28 to ~5.

### 4. `HookDispatcher` — 21-Method Interface

- **File**: `internal/session/interfaces.go:160-182`
- **Downstream impact**: Drives 794 lines of boilerplate in `manager_hooks.go`
- **Problem**: Each hook event gets its own typed dispatch method. While type-safe, this violates Go's small-interface principle and creates massive boilerplate in every implementor.

**Refactoring approach**: Either:
- **(a)** Generic dispatch with generics: `Dispatch[P Payload](ctx context.Context, p P) (P, error)` — single method, type-safe via generics.
- **(b)** Group into sub-interfaces by domain: `SessionLifecycleHooks`, `AgentHooks`, `MessageHooks` — each with 3-5 methods. Implementors embed only what they need.

Option (b) is lower risk and doesn't require generics plumbing. Both approaches eliminate ~800 lines of boilerplate in `manager_hooks.go`.

## P1 — High Priority

### 5. God File: `cli/skill.go` (1,778 LOC)

- **File**: `internal/cli/skill.go`
- **Problem**: Single file bundles all skill-related CLI subcommands, formatting, and install logic.
- **Fix**: Split into separate files per subcommand (`skill_list.go`, `skill_install.go`, `skill_show.go`, etc.).

### 6. God File: `hooks/dispatch.go` (868 LOC)

- **File**: `internal/hooks/dispatch.go`
- **Coupling**: Ce=4, Ca=0 (instability 1.0), 103 relations
- **Problem**: Mixes dispatch orchestration, async submission, filter/match logic, and result aggregation.
- **Fix**: Extract `async_dispatch.go` for the async path. Move filter/match logic into `matcher.go`.

### 7. God File: `skills/registry.go` (971 LOC)

- **File**: `internal/skills/registry.go`
- **Problem**: Mixes global skill loading, workspace caching with TTL, snapshot diffing, and hook registration in a single file.
- **Fix**: Extract workspace cache into `ws_cache.go` and snapshot diffing into `snapshot.go`.

### 8. `httpapi/server.go` — Blast Radius 84-89

- **File**: `internal/api/httpapi/server.go`
- **Symbols**: `newHandlers` (BR=89), `RegisterRoutes` (BR=84), `corsMiddleware` (BR=84), `errorMiddleware` (BR=84), `requestLoggingMiddleware` (BR=84)
- **Problem**: All route registration and all middleware definitions in a single file. Any change ripples to 84+ transitive dependents.
- **Fix**: Move middleware to `middleware.go`. Split `RegisterRoutes` into per-domain registration functions (`registerSessionRoutes()`, `registerWorkspaceRoutes()`, `registerMemoryRoutes()`, etc.).

### 9. `daemon.New()` — Composition Root Defaults

- **File**: `internal/daemon/daemon.go`
- **Complexity**: 21 | **LOC**: 130 | **Blast radius**: 35
- **Problem**: 15+ sequential `if d.X == nil { d.X = defaultX }` nil-guards setting default factory functions.
- **Fix**: Consolidate into an `applyDefaults()` method, or set defaults at struct initialization and let options only override.

### 10. `acp/client.Start()` — Mixed Concerns

- **File**: `internal/acp/client.go`
- **LOC**: 141
- **Problem**: Subprocess setup, MCP server spawning, and JSON-RPC initialization all in one function.
- **Fix**: Split into `spawnProcess()`, `initializeMCPServers()`, and `negotiateSession()`.

## P2 — Medium Priority

### 11. Struct Bloat / Primitive Obsession

- **Files**: `internal/api/contract/contract.go` (339 LOC), `internal/acp/types.go` (429 LOC), `internal/store/types.go` (381 LOC), `internal/hooks/types.go` (258 LOC)
- **Problem**: `WorkspaceID` and `WorkspacePath` repeated as flat string fields across 4+ payload structs in different packages.
- **Fix**: Introduce a `WorkspaceRef` value object and embed it in all payload types:

```go
type WorkspaceRef struct {
    ID   string `json:"workspace_id,omitempty"`
    Path string `json:"workspace_path,omitempty"`
}
```

### 12. `StreamSession` — Deep Nesting

- **File**: `internal/api/core/handlers.go:309`
- **LOC**: 107 | **Nesting depth**: 7
- **Problem**: Polling loop contains nested `select > range > if > if` chains building SSE messages inline.
- **Fix**: Extract `pollAndSendEvents()` and `writeEventBatch()` helpers to flatten nesting.

### 13. `cli/client.go` `decodeSSE` — Feature Envy

- **File**: `internal/cli/client.go`
- **LOC**: 65 | **Nesting depth**: 5
- **Problem**: SSE parsing logic inlined in the CLI client. This SSE parsing belongs in a shared utility, not the CLI package.
- **Fix**: Extract to a shared `internal/sse` or `internal/api/core` utility.

### 14. Dependency Hotspots

- **`hooks/payloads.go`** (168 relations): Most coupled file in the codebase. Consider splitting payload types by domain (session payloads, agent payloads, lifecycle payloads) so consumers only import what they need.
- **`cli/client.go`** (157 relations): Wide API surface touching all contract types. Could benefit from per-domain client files.
- **`session/manager_lifecycle.go`** (instability 1.0, Ce=5, Ca=0): Pure efferent coupling, maximally unstable. Acceptable for leaf orchestration but worth monitoring.

## Dead Code — Safe to Remove

### Confirmed Dead Go Exports

| Symbol | File | Evidence |
|--------|------|----------|
| `AgeDays()` | `internal/memory/staleness.go:9` | Only called from `store_test.go` |
| `AgeText()` | `internal/memory/staleness.go:19` | Only called from `store_test.go` |
| `FreshnessWarning()` | `internal/memory/staleness.go:31` | Only called from `store_test.go` |
| `CanonicalPayload()` | `internal/transcript/transcript.go:618` | Only called from `transcript_test.go` |
| `WithSessionStopTimeout()` | `internal/memory/consolidation/runtime.go:68` | Defined, zero callers anywhere |

### Should Unexport

| Symbol | File | Reason |
|--------|------|--------|
| `MergeProvider()` | `internal/config/provider.go:165` | Only used within its own file |

### Test-Only Production Exports (Violates Project Guidelines)

These exported functions exist in production code but are only called from tests. Per project rules: "Mock via interfaces, not test-only methods in production code."

| Symbol | File |
|--------|------|
| `config.Default()` | `internal/config/config.go:248` |
| `config.WithoutDotEnv()` | `internal/config/config.go:158` |
| `config.WithoutValidation()` | `internal/config/config.go:165` |
| `workspace.WithNow()` | `internal/workspace/options.go:50` |
| `memory.WithGoal()` | `internal/memory/dream.go:164` |

### Unused shadcn/ui Components (16 files)

Installed by the shadcn CLI but never imported by any component or route:

`carousel.tsx`, `chart.tsx`, `calendar.tsx`, `context-menu.tsx`, `drawer.tsx`, `menubar.tsx`, `navigation-menu.tsx`, `pagination.tsx`, `radio-group.tsx`, `slider.tsx`, `alert-dialog.tsx`, `hover-card.tsx`, `input-otp.tsx`, `resizable.tsx`, `aspect-ratio.tsx`, `checkbox.tsx`

## Development Sequencing

### Recommended Attack Order

| Phase | Target | Risk | Impact |
|-------|--------|------|--------|
| 1 | Dead code removal (exports + unused UI) | Zero | Immediate cleanup, smaller surface |
| 2 | `Create`/`Resume` deduplication | Low | Eliminates ~80 lines of highest-severity duplication |
| 3 | `boot()` decomposition into phases | Low-Medium | Reduces highest complexity from 54 to ~10 per phase |
| 4 | `handleInbound` registry pattern | Low | Mechanical transform, complexity 28 to ~5 |
| 5 | `HookDispatcher` interface shrink | Medium | Cascading ~800-line reduction in `manager_hooks.go` |
| 6 | God file splits (skill.go, dispatch.go, registry.go, server.go) | Low | Better navigation, lower per-file blast radius |
| 7 | Struct consolidation (`WorkspaceRef` value object) | Medium | Cross-cutting change across 4+ packages |

### Phase Dependencies

```
Phase 1 (dead code)       -- independent, do first
Phase 2 (Create/Resume)   -- independent
Phase 3 (boot)            -- independent
Phase 4 (handleInbound)   -- independent
Phase 5 (HookDispatcher)  -- should follow Phase 6 (god file splits) for hooks
Phase 6 (god file splits) -- independent per file
Phase 7 (WorkspaceRef)    -- should be last, touches multiple packages
```

Phases 1-4 are independent and can be parallelized. Phase 5 benefits from Phase 6 completing on hooks first. Phase 7 is cross-cutting and should be last.

### Verification Gates

Every phase must pass before closing:

- `make verify` (fmt, lint, test, build) — zero warnings, zero errors
- `make test` with `-race` flag
- No new golangci-lint issues introduced
- 80% coverage minimum per modified package

## Technical Considerations

### Key Decisions

- **Incremental phases over big-bang refactor**: Each phase is independently shippable and verifiable. No phase requires another to be useful.
- **File splits over package splits**: God files are split within their current package (new files, same package) rather than creating sub-packages. This avoids import churn while improving navigability.
- **Registry pattern for dispatch**: `handleInbound` uses a map-based registry rather than generated code or reflection. Keeps the codebase simple and grep-friendly.
- **Sub-interfaces over generics for HookDispatcher**: Grouping methods into domain sub-interfaces is lower risk than introducing a generic dispatch pattern and doesn't require changes to the Go type system usage.
- **Value object for WorkspaceRef**: Embedding a struct is the simplest way to reduce primitive repetition without changing serialization behavior (JSON field names remain stable).

### Known Risks

- **HookDispatcher shrink (Phase 5)**: This is the highest-risk change because it touches the session-hooks integration boundary. The 21-method interface is consumed by `manager_hooks.go` and any test doubles. All implementors must be updated atomically.
  - Mitigation: Do the god file split on `hooks/` first (Phase 6) to reduce the blast radius, then tackle the interface.
- **WorkspaceRef embedding (Phase 7)**: Cross-cutting struct change across `contract`, `acp/types`, `store/types`, and `hooks/types`. JSON serialization must remain backward-compatible for the web UI.
  - Mitigation: Verify JSON output parity with table-driven tests before and after.
- **Test-only export removal**: Removing `config.Default()`, `config.WithoutDotEnv()`, etc. requires finding alternative test setup patterns (e.g., constructing config structs directly, using interfaces).
  - Mitigation: Address per-package, verify test coverage doesn't drop.

## Architecture Decision Records

- [ADR-001: Incremental Phase-Based Refactoring Over Big-Bang](adrs/adr-001.md) — Each refactoring target is an independent, shippable phase with its own verification gate.
- [ADR-002: File Splits Over Package Splits for God Files](adrs/adr-002.md) — God files are decomposed within their current package to avoid import churn.
- [ADR-003: Sub-Interfaces Over Generics for HookDispatcher](adrs/adr-003.md) — Domain-grouped sub-interfaces reduce the 21-method interface without introducing generics complexity.
- [ADR-004: Map-Based Handler Registry for ACP Dispatch](adrs/adr-004.md) — Replaces the switch statement with a typed handler map for maintainability.

## Implementation Status (2026-04-10)

The refactor plan described above has been implemented and re-verified against the live codebase.

- **Phase 1 complete**: Removed dead/test-only exports, collapsed test-only option surfaces back to unexported helpers, deleted 16 unused shadcn components, and removed their unused frontend dependencies.
- **Phase 2 complete**: Extracted a shared session-start pipeline so `Create` and `Resume` both delegate through the same startup path.
- **Phase 3 complete**: Decomposed daemon boot into focused boot phases, split ACP client startup into subprocess/init/session stages, and replaced the ACP inbound switch with a typed handler registry.
- **Phase 4 complete**: Replaced the monolithic session hook dependency with grouped hook sub-interfaces collected in `session.HookSet`, then split hook matcher and async dispatch concerns into dedicated files.
- **Phase 5 complete**: Split `httpapi` routing/middleware/handler wiring, split the skills registry cache/snapshot concerns into dedicated files, extracted shared SSE decoding into `internal/sse`, flattened `StreamSession`, and decomposed the CLI skill implementation into focused files (`skill.go`, `skill_workspace.go`, `skill_marketplace.go`, `skill_output.go`, `skill_commands.go`).
- **Phase 7 adapted**: The original embedded `WorkspaceRef` value-object proposal was attempted and rejected because it introduced keyed-composite-literal breakage and `ID` field collisions across existing structs. The shipped solution centralizes workspace reference construction through `internal/workref.PathRef` and `internal/workref.RootRef`, which reduces repeated workspace reference assembly without destabilizing public struct layouts.

### Final Verification

- `make verify` passed on 2026-04-10.
- `make test-integration` passed on 2026-04-10.
