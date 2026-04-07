# TechSpec: Workspace Entity

## Executive Summary

Introduce a formal Workspace domain entity to AGH, replacing bare path strings with a Resolver that computes workspace state from the filesystem backed by a persistent SQLite table. The Resolver scans `.agh/` directories to discover local agents and skills, merges them with globals by name (local wins), and produces `ResolvedWorkspace` snapshots consumed by session creation, dream consolidation, and the CLI.

**Primary trade-off:** Adding a new `internal/workspace/` package with cross-cutting integration across 12 packages in exchange for structured workspace management, multi-root support, and elimination of `os.Getwd()` bugs in daemon-side code.

## System Architecture

### Component Overview

```
                         daemon/ (composition root)
                            │
               ┌────────────┼────────────────┐
               ▼            ▼                ▼
          workspace/    session/          httpapi/
          Resolver      Manager           Handlers
            │              │                 │
            ▼              ▼                 │
          store/        acp/                 │
          GlobalDB      Driver               │
            │                                │
            └───── ~/.agh/agh.db ◄───────────┘
```

**Data flow for session creation:**
1. CLI/API sends workspace name or path to daemon
2. `session.Manager` calls `workspace.Resolver.ResolveOrRegister(ctx, nameOrPath)`
3. Resolver canonicalizes path (`EvalSymlinks`), checks backing table, auto-registers if new
4. Resolver scans `root_dir/.agh/` + `add_dirs[N]/.agh/` for agents/skills
5. Resolver loads config cascade (defaults → global → `root_dir/.agh/config.toml`)
6. Returns `ResolvedWorkspace` with merged config, agents, skills
7. Manager creates session with `WorkspaceID` FK and resolved config
8. ACP driver starts subprocess with `cmd.Dir = RootDir` and `AdditionalDirs` in RPC

## Implementation Design

### Core Interfaces

**Resolver** — primary type in `workspace/`:

```go
type Resolver struct {
    store      WorkspaceStore
    homePaths  config.HomePaths
    loadConfig func(string) (config.Config, error)
    logger     *slog.Logger
    cache      map[string]*cachedEntry
    mu         sync.RWMutex
}
```

**WorkspaceStore** — interface defined in `workspace/`, implemented by `store/GlobalDB`:

```go
type WorkspaceStore interface {
    InsertWorkspace(ctx context.Context, ws Workspace) error
    UpdateWorkspace(ctx context.Context, ws Workspace) error
    DeleteWorkspace(ctx context.Context, id string) error
    GetWorkspace(ctx context.Context, id string) (Workspace, error)
    GetWorkspaceByPath(ctx context.Context, rootDir string) (Workspace, error)
    GetWorkspaceByName(ctx context.Context, name string) (Workspace, error)
    ListWorkspaces(ctx context.Context) ([]Workspace, error)
}
```

**WorkspaceResolver** — interface consumed by `session.Manager`:

```go
type WorkspaceResolver interface {
    Resolve(ctx context.Context, idOrPath string) (ResolvedWorkspace, error)
    ResolveOrRegister(ctx context.Context, path string) (ResolvedWorkspace, error)
}
```

### Data Models

**Workspace** (persisted entity):

```go
type Workspace struct {
    ID             string    // ws_<random-hex>
    RootDir        string    // canonical absolute path (post-EvalSymlinks)
    AdditionalDirs []string  // optional extra directories
    Name           string    // human-friendly slug (unique)
    DefaultAgent   string    // optional agent override
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

**ResolvedWorkspace** (computed snapshot, not persisted):

```go
type ResolvedWorkspace struct {
    Workspace                      // embedded registration data
    Config    config.Config        // fully resolved cascade
    Agents    []config.AgentDef    // merged local > global
    Skills    []SkillPath          // discovered skill paths for skills registry
    ResolvedAt time.Time
}
```

**SkillPath** (pointer for skills registry):

```go
type SkillPath struct {
    Dir    string      // directory containing the skill
    Source string      // "global", "workspace", "additional"
}
```

**Error types:**

```go
var (
    ErrWorkspaceNotFound    = errors.New("workspace not found")
    ErrWorkspaceRootMissing = errors.New("workspace root directory no longer exists")
    ErrAgentNotAvailable    = errors.New("agent not available in workspace")
    ErrWorkspaceNameTaken   = errors.New("workspace name already in use")
    ErrWorkspacePathTaken   = errors.New("workspace path already registered")
)
```

**SQLite schema:**

```sql
CREATE TABLE IF NOT EXISTS workspaces (
    id            TEXT PRIMARY KEY,
    root_dir      TEXT NOT NULL UNIQUE,
    add_dirs      TEXT DEFAULT '[]',
    name          TEXT NOT NULL UNIQUE,
    default_agent TEXT DEFAULT '',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workspaces_name ON workspaces(name);
```

**Session schema change:**

```sql
-- Before:
workspace TEXT NOT NULL

-- After:
workspace_id TEXT NOT NULL  -- references workspaces(id)
```

### API Endpoints

**Workspace CRUD:**

| Method | Path | Description | Request | Response |
|--------|------|-------------|---------|----------|
| POST | `/api/workspaces` | Register workspace | `{root_dir, name?, add_dirs?, default_agent?}` | `{workspace: {...}}` |
| GET | `/api/workspaces` | List workspaces | — | `{workspaces: [{...}]}` |
| GET | `/api/workspaces/:id` | Get workspace detail | — | `{workspace: {...}, sessions: [...], agents: [...], skills: [...]}` |
| PATCH | `/api/workspaces/:id` | Update workspace | `{name?, add_dirs?, default_agent?}` | `{workspace: {...}}` |
| DELETE | `/api/workspaces/:id` | Unregister workspace | — | `204 No Content` |
| POST | `/api/workspaces/resolve` | Resolve or register | `{path: "/abs/path"}` | `{workspace: {...}}` |

**Session creation change:**

| Method | Path | Before | After |
|--------|------|--------|-------|
| POST | `/api/sessions` | `{workspace?: "/path"}` | `{workspace: "name-or-id" \| workspace_path: "/path"}` — at least one required |
| GET | `/api/sessions` | no filter | `?workspace=<id-or-name>` filter supported |

### CLI Commands

```
agh workspace add <path> [--name <slug>] [--add-dir <path>]... [--default-agent <name>]
agh workspace list
agh workspace info <name-or-id>
agh workspace edit <name-or-id> [--name <new>] [--add-dir <path>] [--remove-dir <path>] [--default-agent <name>]
agh workspace remove <name-or-id>

agh session new --workspace <name-or-id>    # by registered workspace
agh session new --cwd <path>                # by path (auto-register)
agh session new                             # CLI caller CWD (auto-register)
agh session list --workspace <name-or-id>   # filter
```

### Caching

- In-memory cache keyed by workspace ID in the Resolver
- On `Resolve()`: compare file snapshots (mtime + size) of `config.toml`, `agents/`, `skills/` dirs
- If unchanged, return cached `ResolvedWorkspace`. Otherwise re-scan.
- TTL eviction for entries not accessed in 10 minutes (matches skills registry pattern)
- `Invalidate(workspaceID)` method for programmatic cache busting
- `--force-refresh` CLI flag calls Invalidate before Resolve
- Cache is in-memory only — not persisted across daemon restarts

### Resolution Algorithm

`Resolve(ctx, idOrNameOrPath)`:

```
1. Determine lookup type:
   - starts with "ws_" → lookup by ID
   - starts with "/" → lookup by canonical path (EvalSymlinks first)
   - else → lookup by name

2. Fetch Workspace from store

3. Validate root_dir exists (os.Stat):
   - missing → return ErrWorkspaceRootMissing
   
4. Re-evaluate symlinks on root_dir:
   - if canonical path changed → update stored root_dir

5. Check cache (mtime + size snapshot comparison):
   - cache hit → return cached ResolvedWorkspace

6. Scan resources:
   a. Load config: defaults → ~/.agh/config.toml → root_dir/.agh/config.toml
   b. Scan agents from [root_dir, add_dirs...] + ~/.agh/agents/
      - For each dir: glob <dir>/.agh/agents/*/AGENT.md
      - Merge by name: root_dir wins > add_dirs[0] > ... > global
   c. Collect skill paths from [root_dir, add_dirs...] + ~/.agh/skills/
      - For each dir: list <dir>/.agh/skills/*/
      - Collect paths (actual loading delegated to skills registry)

7. Build ResolvedWorkspace, update cache, return
```

`ResolveOrRegister(ctx, path)`:

```
1. Canonicalize: filepath.EvalSymlinks(filepath.Abs(path))
2. Try: store.GetWorkspaceByPath(canonical)
   - found → Resolve(ctx, workspace.ID)
   - not found → auto-register:
     a. Generate ID: ws_<random-hex>
     b. Derive name: filepath.Base(canonical), dedup with -2 suffix
     c. Insert into store
     d. Resolve(ctx, newID)
```

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/workspace/` | new | New package: Resolver, types, store interface | Create package with 4 files |
| `internal/store/` | modified | Add `workspaces` table, implement WorkspaceStore, change `sessions.workspace` to `workspace_id` | Add schema, CRUD methods, migrate session column |
| `internal/session/` | modified | `Session.Workspace` → `WorkspaceID`, Manager receives Resolver, delete `resolveWorkspace()` | Medium risk — touches session creation and resume hot paths |
| `internal/config/` | modified | Remove `os.Getwd()` fallback in `resolveWorkspaceRoot()`, agent loading accepts workspace paths | Low risk — removing fallback, adding parameters |
| `internal/daemon/` | modified | Wire Resolver, update dream consolidation to use workspace IDs | Medium risk — composition root changes affect boot sequence |
| `internal/httpapi/` | modified | New `/api/workspaces` route group, session creation requires workspace | Medium risk — API contract change |
| `internal/udsapi/` | modified | Mirror httpapi workspace endpoints | Low risk — follows httpapi pattern |
| `internal/cli/` | modified | New `workspace` command tree, session commands gain `--workspace` flag | Low risk — additive CLI changes |
| `internal/acp/` | modified | `StartOpts.AdditionalDirs []string`, ACP RPC sends additional_dirs | Low risk — additive field |
| `internal/observe/` | modified | Session reconciliation uses workspace ID | Low risk — field rename |
| `internal/memory/` | modified | Dream receives workspace ID, resolves path via Resolver when needed | Low risk — indirection change |
| `internal/skills/` | modified | Delegates workspace scanning to Resolver, receives skill paths instead of scanning | Medium risk — changes discovery ownership |
| `web/` | modified | Workspace selector, session grouping by workspace, new API endpoints | High risk — significant frontend effort |

## Testing Approach

### Unit Tests

**workspace/resolver_test.go:**
- `TestResolve_ByID` / `TestResolve_ByName` / `TestResolve_ByPath` — lookup routing
- `TestResolveOrRegister_ExistingWorkspace` — returns existing without re-registering
- `TestResolveOrRegister_AutoRegister` — creates workspace from new path
- `TestResolveOrRegister_NameDedup` — deduplicates with `-2` suffix
- `TestResolve_CacheHit` — returns cached when mtime unchanged
- `TestResolve_CacheInvalidation` — re-scans when mtime changes
- `TestResolve_RootMissing` — returns `ErrWorkspaceRootMissing`
- `TestResolve_SymlinkChanged` — updates stored root_dir when symlink target changes
- `TestResolve_AgentMerge` — local agents override global by name
- `TestResolve_ConfigFromRootOnly` — additional dirs don't contribute config
- Mock `WorkspaceStore` for all unit tests

**store/global_db_test.go (workspace additions):**
- Table-driven CRUD tests: Insert, Update, Delete, Get, GetByPath, GetByName, List
- Constraint tests: duplicate root_dir, duplicate name
- Session schema migration: workspace_id column

### Integration Tests

**workspace/resolver_integration_test.go:**
- Real SQLite via `t.TempDir()`
- Real filesystem with `.agh/agents/`, `.agh/skills/`, `.agh/config.toml`
- End-to-end: register → resolve → verify agents/skills merged correctly
- Symlink handling: create symlink, register, resolve, change symlink target, re-resolve

**session/manager_integration_test.go (workspace additions):**
- Session creation with workspace resolution (auto-register flow)
- Session resume with workspace still valid
- Session resume with workspace removed

**Target: 80% coverage per package** (per CLAUDE.md requirement).

## Development Sequencing

### Build Order

1. **`internal/workspace/` types** — `Workspace`, `ResolvedWorkspace`, `SkillPath`, error types, `WorkspaceStore` interface, `WorkspaceResolver` interface. No dependencies.
2. **`internal/store/` workspace table** — Add `workspaces` schema to `globalSchemaStatements`, implement `WorkspaceStore` on `GlobalDB`. Depends on step 1.
3. **`internal/workspace/` Resolver** — `NewResolver`, `Resolve`, `ResolveOrRegister`, `Register`, `Unregister`, `Update`, `List`, `Get`, caching logic. Depends on steps 1-2.
4. **`internal/workspace/` tests** — Unit tests with mock store + integration tests with real SQLite. Depends on step 3.
5. **`internal/session/` integration** — Add `WithWorkspaceResolver` option, replace `resolveWorkspace()` with Resolver calls, change `Session.Workspace` to `WorkspaceID`. Depends on step 3.
6. **`internal/config/` cleanup** — Remove `os.Getwd()` fallback in `resolveWorkspaceRoot()`, add workspace-scoped agent loading path. Depends on step 5.
7. **`internal/daemon/` wiring** — Wire Resolver into session manager factory, update dream consolidation to use workspace IDs. Depends on steps 3, 5, 6.
8. **`internal/skills/` delegation** — Modify `ForWorkspace()` to accept skill paths from Resolver instead of doing its own scan. Depends on step 3.
9. **`internal/cli/` workspace commands** — New `newWorkspaceCommand(deps)` with add/list/info/edit/remove subcommands. Depends on step 7.
10. **`internal/httpapi/` + `internal/udsapi/` endpoints** — New `/api/workspaces` route group, update session creation to require workspace. Depends on step 7.
11. **`internal/acp/` AdditionalDirs** — Add `AdditionalDirs []string` to `StartOpts`, include in ACP RPC `session/new` and `session/load`. Depends on step 5.
12. **`internal/observe/` + `internal/memory/` updates** — Change workspace path references to workspace ID. Depends on step 5.
13. **`web/` workspace UI** — Workspace selector, session grouping, new API consumption. Depends on step 10.
14. **`make verify` gate** — Full fmt → lint → test → build pass. Depends on all previous steps.

### Technical Dependencies

- No external dependencies needed — all within existing Go stdlib + SQLite
- No infrastructure changes — uses existing `~/.agh/agh.db`
- Skills registry refactoring (step 8) should be coordinated with step 3 to avoid duplicate scanning during transition

## Monitoring and Observability

- **Structured logging** via `slog`:
  - `workspace.register`: `workspace_id`, `root_dir`, `name`
  - `workspace.resolve`: `workspace_id`, `cache_hit`, `agents_count`, `skills_count`, `duration_ms`
  - `workspace.resolve.error`: `workspace_id`, `error_type`
  - `workspace.cache.evict`: `workspace_id`, `age_minutes`
- **Health metric**: count of registered workspaces with missing `root_dir` (staleness indicator)
- **Session events**: workspace ID included in session creation/stop events for cross-referencing

## Technical Considerations

### Key Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| Resolver, not Manager | Filesystem owns workspace existence; system remembers how to find it | Two concepts (stored hint vs resolved snapshot) to reason about |
| `ws_<hex>` IDs | Stable across renames, consistent with `sess_`/`turn_` pattern | Requires name field for CLI ergonomics |
| Config from root only | Prevents N-layer merge chain debugging nightmare | Asymmetry may confuse users |
| New package | Clean boundary, independent evolution | One more package, cross-cutting integration |
| mtime-based caching | Proven pattern (skills registry), adequate for local-first alpha | Best-effort on some filesystems (FAT32, NFS) |

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Symlink target changes after registration | Low | Re-evaluate `EvalSymlinks` on every `Resolve()`, update stored root_dir |
| Directory renamed/moved orphans workspace | Medium | `os.Stat()` check on `List()` results, clear error messages on `Resolve()` |
| Web UI breaks without workspace selector | High | Scope web/ changes explicitly; `POST /api/workspaces/resolve` endpoint for gradual adoption |
| Skills double-scanning during transition | Low | Implement step 8 (skills delegation) immediately after step 3 |
| Name dedup suffix (-2, -3) confusion | Low | Allow rename via `agh workspace edit --name` |

## Architecture Decision Records

- [ADR-001: Resolver with Persistent Backing over Manager Entity](adrs/adr-001.md) — Use a Resolver that treats the filesystem as source of truth, backed by a SQLite hint table for stable IDs and CLI queries
- [ADR-002: Random Hex ID with Human-Friendly Name Field](adrs/adr-002.md) — Use `ws_<random-hex>` as PK with a separate unique `name` field for CLI ergonomics, avoiding slug-from-dirname collisions
- [ADR-003: Config from Root Only, Agents/Skills from All Dirs](adrs/adr-003.md) — Asymmetric discovery: config stays single-source from root_dir, agents/skills discovered from all workspace dirs
- [ADR-004: New internal/workspace/ Package](adrs/adr-004.md) — Dedicated package for workspace resolution logic, keeping store/ as pure persistence and config/ as stateless loader
