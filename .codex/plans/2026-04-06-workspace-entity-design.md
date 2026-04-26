# Design: Workspace Entity

**Date:** 2026-04-06
**Status:** Approved
**Author:** Pedro Nauck + Claude Council

## Problem

AGH's workspace handling has three fundamental issues:

1. **Daemon CWD is meaningless** — the daemon starts from wherever `agh daemon start` was run, but that CWD has no semantic significance. Sessions created via API without explicit workspace fall back to `os.Getwd()` of the daemon process, producing silent bugs.
2. **Workspaces are implicit strings** — `Session.Workspace` is a bare `string` (absolute path) with no registry, no lifecycle, no queryability. Dream consolidation, memory scoping, and session grouping all derive workspace identity from raw string comparison.
3. **No multi-root support** — an agent operating across multiple directories (monorepo + infra repo, frontend + backend) has no way to express that context.

## Decisions

| Decision             | Choice                                                          |
| -------------------- | --------------------------------------------------------------- |
| Role of workspace    | Logical grouping with multi-root (organization, not security)   |
| Session to workspace | Session belongs to exactly 1 workspace                          |
| Missing workspace    | Auto-register from CWD of CLI caller                            |
| Multi-root model     | root_dir (primary) + additional_dirs (optional)                 |
| Resource discovery   | Agents/skills from all dirs; config only from root_dir          |
| Merge strategy       | Local overrides global by name                                  |
| Architecture         | Resolver with persistent backing (not Manager/entity lifecycle) |
| ID strategy          | `ws_<random-hex>` with human-friendly `name` field              |
| Package              | New `internal/workspace/`                                       |
| Daemon model         | Workspace-agnostic, multiplexes N workspaces                    |

## Architecture

### Core Insight: Resolver, Not Registry

The council debate surfaced a key distinction: this is a **resolution problem**, not an entity problem. The filesystem owns the workspace — AGH just remembers how to find it. The design uses a Resolver with a persistent backing table, not a Manager with entity lifecycle semantics.

The `agh workspace add` command does not "create" a workspace — the directory already exists. It **registers a resolution hint**: "when I say 'myapp', resolve to `/projects/myapp` and apply these config overlays."

Deletion (`agh workspace remove`) does not delete the directory, config, or sessions — it only removes the alias from the hint table.

### Data Model

#### `workspaces` table (in `~/.agh/agh.db`)

```sql
CREATE TABLE workspaces (
    id            TEXT PRIMARY KEY,      -- ws_<random-hex>
    root_dir      TEXT NOT NULL UNIQUE,  -- canonical path (post-EvalSymlinks)
    add_dirs      TEXT,                  -- JSON array of additional directories
    name          TEXT NOT NULL UNIQUE,  -- human-friendly slug (default: dirname, dedup with -2 suffix)
    default_agent TEXT,                  -- optional agent override for this workspace
    created_at    DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL
);
```

- **`id`**: Generated random ID (`ws_<hex>`), like sessions use `sess_<hex>`. Stable across renames.
- **`root_dir`**: Canonical absolute path after `filepath.EvalSymlinks()`. The real identity anchor.
- **`name`**: Human-friendly display label for CLI usage. Default `filepath.Base(root_dir)`, auto-deduplicated with `-2` suffix on collision. Editable.
- **`add_dirs`**: JSON array of additional directory paths. Not queryable individually (acceptable for alpha).
- **`default_agent`**: Optional. Overrides `defaults.agent` from config when creating sessions in this workspace.

#### `ResolvedWorkspace` (computed, not persisted)

```go
type ResolvedWorkspace struct {
    ID             string
    Name           string
    RootDir        string
    AdditionalDirs []string
    DefaultAgent   string
    Config         config.Config    // fully resolved cascade
    Agents         []config.AgentDef // merged local > global
    Skills         []SkillInfo       // merged local > global
    ResolvedAt     time.Time
}
```

Produced by `Resolver.Resolve()` on demand. Cached in-memory with TTL + mtime snapshot comparison (same pattern as skills registry).

### Package: `internal/workspace/`

```
internal/workspace/
    workspace.go      -- Workspace struct, ResolvedWorkspace, types
    resolver.go       -- Resolver: resolve, scan, merge
    store.go          -- WorkspaceStore interface (consumed here, implemented in store/)
    options.go        -- functional options for NewResolver
```

#### Resolver API

```go
type Resolver struct {
    store      WorkspaceStore
    homePaths  config.HomePaths
    loadConfig ConfigLoader
    logger     *slog.Logger
    cache      map[string]*cachedWorkspace  // keyed by workspace ID
}

// Registration (persistent backing)
func (r *Resolver) Register(ctx context.Context, opts RegisterOpts) (Workspace, error)
func (r *Resolver) Unregister(ctx context.Context, id string) error
func (r *Resolver) Update(ctx context.Context, id string, opts UpdateOpts) error
func (r *Resolver) List(ctx context.Context) ([]Workspace, error)
func (r *Resolver) Get(ctx context.Context, idOrPath string) (Workspace, error)

// Core resolution: workspace -> full snapshot
func (r *Resolver) Resolve(ctx context.Context, idOrPath string) (ResolvedWorkspace, error)

// Auto-register: find or create workspace from path
func (r *Resolver) ResolveOrRegister(ctx context.Context, path string) (ResolvedWorkspace, error)
```

#### WorkspaceStore interface

```go
type WorkspaceStore interface {
    InsertWorkspace(ctx context.Context, ws Workspace) error
    UpdateWorkspace(ctx context.Context, ws Workspace) error
    DeleteWorkspace(ctx context.Context, id string) error
    GetWorkspace(ctx context.Context, id string) (Workspace, error)
    GetWorkspaceByPath(ctx context.Context, rootDir string) (Workspace, error)
    ListWorkspaces(ctx context.Context) ([]Workspace, error)
}
```

Defined in `workspace/`, implemented in `store/`.

#### Dependency flow (downward only)

```
daemon/ (composition root)
  └── wires workspace.Resolver into session.Manager

session/  ──depends on──> workspace.Resolver (via interface)
workspace/ ──depends on──> config.HomePaths, config.Config (types only)
store/    ──implements──> workspace.WorkspaceStore
```

No package imports `daemon/`. `workspace/` does not import `session/`.

### Scan & Discovery

#### When scanning happens

- **Eager on Register** — `agh workspace add` scans `.agh/` dirs immediately
- **Re-scan on Resolve** — session creation/resume checks mtime snapshots, re-scans only if changed
- **Manual refresh** — `agh workspace info <name>` or `--force-refresh` flag

#### What is scanned

For each dir in `[root_dir, add_dirs[0], add_dirs[1], ...]`:

```
<dir>/.agh/
    config.toml     -> config overlay (ONLY from root_dir)
    agents/         -> agent definitions (AGENT.md files)
    skills/         -> skill definitions
```

#### Scan rules

- Depth: 1 level within each dir (looks for `.agh/` only at the root of each dir)
- Skip list: `.git/`, `node_modules/`, `vendor/`, dot-dirs (except `.agh/`)
- `filepath.EvalSymlinks()` on every dir before scanning
- Re-evaluate symlinks on `Resolve()` — update stored `root_dir` if target changed

#### Config vs Resources: distinct rules

| Resource      | Source                           | From additional_dirs? |
| ------------- | -------------------------------- | --------------------- |
| `config.toml` | Only `root_dir/.agh/config.toml` | No                    |
| `agents/`     | `root_dir` + all `add_dirs`      | Yes                   |
| `skills/`     | `root_dir` + all `add_dirs`      | Yes                   |

**Rationale:** Config defines project behavior (providers, permissions, limits) — multiple sources create ambiguity. Agents and skills are additive resources where multi-source discovery is natural.

#### Merge precedence

```
Config:   defaults -> ~/.agh/config.toml -> root_dir/.agh/config.toml
                                            (add_dirs do NOT contribute config)

Agents:   ~/.agh/agents/ <- add_dirs[N] <- ... <- add_dirs[0] <- root_dir/.agh/agents/
                                                                  (root wins)

Skills:   ~/.agh/skills/ <- add_dirs[N] <- ... <- add_dirs[0] <- root_dir/.agh/skills/
                                                                  (root wins)
```

Root dir has maximum priority. Among additional dirs, declared order matters (first = higher priority). Global (`~/.agh/`) has lowest priority.

#### Integration with existing systems

The Resolver **owns** directory scanning and path discovery. Existing consumers adapt:

- **Skills registry** (`internal/skills/registry.go`): stops doing its own workspace scan via `ForWorkspace()`. Instead receives skill paths from the Resolver.
- **Agent loader** (`config.LoadAgentDef()`): signature changes to accept workspace-scoped paths from Resolver, not just `homePaths.AgentsDir`.

This eliminates duplicate scan/cache systems.

### Session Integration

#### Session creation flow

```
1. User: agh session new --cwd /projects/myapp
   (or without --cwd -> CLI uses os.Getwd() of caller)

2. session.Manager.Create():
   - Calls resolver.ResolveOrRegister(ctx, path)
     |-- filepath.EvalSymlinks(path)
     |-- store.GetWorkspaceByPath(canonical)
     |   |-- found -> use existing workspace
     |   |-- not found -> auto-register:
     |       |-- id = ws_<random-hex>
     |       |-- name = filepath.Base(path) (dedup if conflict)
     |       |-- scan .agh/ of root_dir
     |       |-- store.InsertWorkspace(...)
     |-- returns ResolvedWorkspace (config, agents, skills merged)

3. Session created with:
   - session.WorkspaceID = resolved.ID  (FK to workspaces table)
   - Config = resolved.Config
   - Agent = resolved from agent list

4. ACP subprocess receives:
   - cmd.Dir = resolved.RootDir           (process CWD)
   - ACP RPC: Cwd = resolved.RootDir
   - ACP RPC: AdditionalDirs = resolved.AdditionalDirs  (new field)
```

#### Session resume flow

```
1. session.Manager.Resume(ctx, sessionID):
   - Read meta.json -> get WorkspaceID
   - Call resolver.Resolve(ctx, workspaceID)
     |-- workspace exists -> re-resolve (stat + re-scan if mtime changed)
     |-- workspace removed -> ErrWorkspaceNotFound
     |-- root_dir missing -> ErrWorkspaceRootMissing
     |-- agent unavailable -> ErrAgentNotAvailable
   - Rebuild ResolvedWorkspace from current filesystem state
```

#### Session struct change

```go
// Before:
Workspace string  // absolute path

// After:
WorkspaceID string  // ws_<hex> referencing workspaces table
```

### Daemon Model

The daemon is **workspace-agnostic**. It:

- Runs from `~/.agh/` (home dir)
- Loads only global config (`~/.agh/config.toml`)
- Manages N workspaces simultaneously via the Resolver
- Has no project CWD — it is a background system service

```
          daemon (no workspace)
            |
    +-------+--------+
    v       v        v
 session A  session B  session C
 ws: myapp  ws: infra  ws: myapp
```

All `os.Getwd()` fallbacks removed from daemon-side code:

- `internal/session/manager.go:1064` — `resolveWorkspace()` deleted, replaced by Resolver
- `internal/config/config.go:451` — `resolveWorkspaceRoot()` returns error if empty
- `internal/daemon/daemon.go:527` — `Boundaries()` receives explicit root

### Error Types

```go
var (
    ErrWorkspaceNotFound    = errors.New("workspace not found")
    ErrWorkspaceRootMissing = errors.New("workspace root directory no longer exists")
    ErrAgentNotAvailable    = errors.New("agent not available in workspace")
    ErrWorkspaceNameTaken   = errors.New("workspace name already in use")
    ErrWorkspacePathTaken   = errors.New("workspace path already registered")
)
```

### Caching

- `ResolvedWorkspace` cached in-memory keyed by workspace ID
- On `Resolve()`: compare file snapshots (mtime + size) of `config.toml`, agents dir, skills dir
- If unchanged, return cached. Otherwise re-scan.
- TTL eviction for workspaces not accessed recently
- Not persisted across daemon restarts
- `--force-refresh` CLI flag and `Resolver.Invalidate(workspaceID)` for manual cache busting

## CLI Commands

```bash
# CRUD
agh workspace add <path> [flags]
  --name <slug>              # Name/ID (default: basename)
  --add-dir <path>           # Additional directory (repeatable)
  --default-agent <name>     # Default agent for this workspace

agh workspace list
  # NAME        ROOT                    DIRS  SESSIONS  AGENTS  SKILLS
  # myapp       /projects/myapp         1     2 active  3       1
  # infra       /projects/infra         0     0         1       0

agh workspace info <name>
  # Full detail: dirs, discovered agents, skills, active sessions, config overrides

agh workspace edit <name> [flags]
  --name <new-name>
  --add-dir <path>
  --remove-dir <path>
  --default-agent <name>

agh workspace remove <name>
  # Unregisters (does not delete files or kill sessions)

# Session integration
agh session new --workspace <name>     # By registered workspace name
agh session new --cwd <path>           # By path (auto-register if new)
agh session new                        # CLI caller CWD (auto-register if new)
agh session list --workspace <name>    # Filter by workspace
```

## HTTP/UDS API

```
POST   /api/workspaces          -> register
GET    /api/workspaces          -> list
GET    /api/workspaces/:id      -> info
PATCH  /api/workspaces/:id      -> edit
DELETE /api/workspaces/:id      -> remove
POST   /api/workspaces/resolve  -> resolve or register (for web UI)

POST   /api/sessions            -> { workspace: "name" | workspace_path: "/abs/path" }
GET    /api/sessions?workspace= -> filter by workspace
```

`POST /api/sessions` requires `workspace` (registered name) OR `workspace_path` (absolute path, auto-register). At least one is mandatory — no fallback to daemon CWD.

## Package Impact

| Package               | Change                                                               |
| --------------------- | -------------------------------------------------------------------- |
| `internal/workspace/` | **NEW** — Resolver, types, WorkspaceStore interface                  |
| `internal/store/`     | New `workspaces` table, implements WorkspaceStore                    |
| `internal/session/`   | `WorkspaceID` replaces `Workspace` string, Manager receives Resolver |
| `internal/config/`    | Remove `os.Getwd()` fallback, accept workspace-scoped paths          |
| `internal/daemon/`    | Wires Resolver, dream consolidation by workspace ID                  |
| `internal/httpapi/`   | New `/api/workspaces` endpoints, session creation requires workspace |
| `internal/udsapi/`    | Same as httpapi                                                      |
| `internal/cli/`       | New `workspace` command, session commands gain `--workspace` flag    |
| `internal/acp/`       | `StartOpts` gains `AdditionalDirs []string`                          |
| `internal/observe/`   | Reconciliation uses workspace ID                                     |
| `internal/memory/`    | Dream receives workspace ID, resolves via Resolver for paths         |
| `internal/skills/`    | Delegates workspace scanning to Resolver                             |
| `web/`                | Workspace selector, session grouping, new API endpoints              |
| `internal/logger/`    | Untouched                                                            |
| `internal/version/`   | Untouched                                                            |

## Implementation Order

1. `internal/workspace/` + `internal/store/` (table + Resolver core)
2. `internal/session/` (integration with Resolver)
3. `internal/config/` (remove CWD fallbacks, workspace-scoped loading)
4. `internal/daemon/` (wiring, dream consolidation update)
5. `internal/skills/` (delegate scanning to Resolver)
6. `internal/cli/` + `internal/httpapi/` + `internal/udsapi/` (surface)
7. `internal/acp/` (AdditionalDirs in StartOpts)
8. `internal/observe/` + `internal/memory/` (workspace ID updates)
9. `web/` (workspace selector, session grouping)
10. Cleanup: remove all `os.Getwd()` fallbacks, verify no stale path references

## Known Limitations (alpha-acceptable)

- `add_dirs` as JSON column prevents relational queries on individual dirs
- mtime-based re-scan is best-effort (1-2s granularity on some filesystems)
- No workspace lifecycle events/hooks in Notifier (deferred to Phase 2)
- Config asymmetry (root-only) may surprise users — needs documentation
- No filesystem watcher — changes detected only on Resolve() or manual refresh

## Council Review

Design validated by a 4-advisor council (architect-advisor, pragmatic-engineer, devils-advocate, the-thinker) and reviewed by Codex. Key tensions resolved:

- **Entity vs Resolver**: Hybrid — Resolver with persistent backing table
- **New package vs existing**: New `internal/workspace/` (all advisors agreed)
- **Identity strategy**: Random ID (Codex identified slug collision bug, fixed)
- **Symlink safety**: `filepath.EvalSymlinks()` mandatory, re-evaluated on Resolve
