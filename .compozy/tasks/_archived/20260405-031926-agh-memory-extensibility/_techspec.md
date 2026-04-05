# TechSpec: Cross-Session Memory System for AGH v2

## Executive Summary

This TechSpec defines the implementation of a persistent cross-session memory system for AGH v2. The system adds a single new package (`internal/memory/`) containing three subsystems: (1) a file-based persistent memory store (memdir) with dual global/workspace directories and MEMORY.md indexes; (2) a dream consolidation service that spawns ephemeral ACP agent sessions to synthesize session transcripts into durable memory files; and (3) team memory via workspace-scoped files with agent metadata for cross-agent knowledge sharing.

Key architectural decisions: `PromptAssembler` interface defined in `session/` and implemented by `memory/`, injected via functional option — no new packages beyond `memory/`. Frozen snapshot injection — memory loaded once at session start, immutable during session. Dream spawning decoupled via `SessionSpawner` callback wired in `daemon/`. Four-type taxonomy (user, feedback, project, reference) — procedural memory deferred to skills phase. Dream-only extraction — no per-turn background extraction, agents write memories manually via CLI.

Primary trade-off: simplicity and proven patterns over novel features. The memdir and dream implementations are direct ports from the cc-memory project, adapted to v2's ACP-based session model and Notifier fanout pattern. Team memory uses workspace-scoped files instead of the old blackboard subsystem.

## System Architecture

### Component Overview

```
┌──────────────────────────────────────��──────────────────────────┐
│                        Go Binary (agh)                          │
├─────────────────────────────────────────────��───────────────────┤
│  daemon/ (composition root)                                     │
│    ├── Initializes memory.Store at boot                         │
│    ├── Initializes dream.Service with SessionSpawner callback   │
│    ├── Injects PromptAssembler into session.Manager             │
│    ├── Adds periodic dream ticker to lifecycle                  │
│    └── Registers memory HTTP/UDS routes                         │
├─────────────┬───────────────────────────────────────────────────┤
│  memory/    │  session/ (modified)                              │
│  ├── Store  │  ├── PromptAssembler interface (NEW)              │
│  ├── Dream  │  ├── SessionType enum (NEW)                      │
│  ├── Lock   │  ├── Manager.Create() calls assembler            │
│  ├── Stale  │  └── CreateOpts.Type for dream sessions          │
│  └── Assembler (implements PromptAssembler)                     │
├─────────────┼───────────���───────────────────────────────────────┤
│  httpapi/   │  udsapi/ (modified)                               │
│  (modified) │  ├── GET/PUT/DELETE /api/memory/:filename         │
│  ├── memory │  ├── GET /api/memory                              │
│  │  routes  │  └── POST /api/memory/consolidate                 │
│  └──────────┤                                                   │
│  cli/ (modified)                                                │
│  └── agh memory list|read|write|delete|consolidate              │
├──────────────────────────────────────────��──────────────────────┤
│  config/ (modified)                                             │
│  ├── MemoryConfig in Config struct                              │
│  ├── DreamConfig in Config struct                               │
│  └── HomePaths.MemoryDir                                        │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Session start (memory injection)**: `Manager.Create()` → `PromptAssembler.Assemble(agent, workspace)` → loads MEMORY.md indexes from global + workspace scopes → renders memory context string → concatenates with `AgentDef.Prompt` → passes assembled prompt in `acp.StartOpts` → agent receives frozen snapshot.

2. **Agent writes memory**: Agent calls `terminal/create` ACP request → runs `agh memory write <filename> --type project --description "..."` → CLI connects to daemon via UDS → `PUT /api/memory/:filename` → `memory.Store.Write()` → file persisted to disk → MEMORY.md index untouched (dream handles index updates).

3. **Dream consolidation**: Daemon ticker (30min) → `dream.Service.ShouldRun()` checks 3 gates (time ≥ 24h, sessions ≥ 3, lock acquired) → if all pass, `SessionSpawner` callback → `Manager.Create(SessionTypeDream)` → ephemeral ACP session with approve-all permissions → dream agent reads session event databases + existing memories → writes updated memory files via `agh memory write` → session auto-stops → lock released with updated mtime.

4. **Team memory**: Workspace-scoped memory files at `<workspace>/.agh/memory/` with `agent_name` in frontmatter. Any agent in the workspace reads these via MEMORY.md index at session start. Cross-agent consolidation is inherent — dream reads ALL session event databases regardless of agent type.

### Write Path Ownership

- **memory.Store** owns all memory file writes (global + workspace directories)
- **dream.Service** owns the consolidation lock file (`~/.agh/memory/.consolidate-lock`)
- **session.Manager** owns session event recording (per-session `events.db`) — unchanged
- **observe.Observer** owns global DB writes — unchanged
- Dream sessions write memories via the same Store path as user sessions (CLI → API → Store)

### Package Dependency Graph

```
cli/ ──→ daemon/ ──→ session/ ──→ acp/
                │        │
                │        └──→ store/
                │
                ├──→ memory/ ──→ (no internal deps, only stdlib + config types)
                │       │
                │       └── implements session.PromptAssembler
                │
                ├──→ httpapi/ ──→ session/ + memory/ (read-only Store interface)
                ├──→ udsapi/  ──→ session/ + memory/ (read-only Store interface)
                ├──→ observe/
                └──→ config/
```

Rules (additions to existing):
- `memory/` does NOT import `session/`, `daemon/`, `httpapi/`, `udsapi/`, or `cli/`
- `memory/` imports only `config/` (for `HomePaths`, `AgentDef` types) and stdlib
- `session/` does NOT import `memory/` — it only knows the `PromptAssembler` interface
- `daemon/` wires `memory.Assembler` into `session.Manager` via `WithPromptAssembler()`
- `httpapi/` and `udsapi/` accept a `memory.ReadWriter` interface for API handlers

## Implementation Design

### Core Interfaces

```go
// session/interfaces.go — NEW: what session needs from prompt assembly
type PromptAssembler interface {
    Assemble(ctx context.Context, agent aghconfig.AgentDef, workspace string) (string, error)
}
```

```go
// memory/store.go — file-based memory store
type Store struct {
    globalDir     string
    workspaceDir  string // set per-workspace at assembly time
    maxIndexLines int
    maxIndexBytes int
}

func NewStore(globalDir string) *Store
func (s *Store) Read(scope Scope, filename string) ([]byte, error)
func (s *Store) Write(scope Scope, filename string, content []byte) error
func (s *Store) Delete(scope Scope, filename string) error
func (s *Store) Scan(scope Scope) ([]MemoryHeader, error)
func (s *Store) LoadIndex(scope Scope) (string, bool, error)
func (s *Store) EnsureDirs() error
```

```go
// memory/assembler.go — implements session.PromptAssembler
type Assembler struct {
    store     *Store
    globalDir string
}

func NewAssembler(store *Store) *Assembler
func (a *Assembler) Assemble(ctx context.Context, agent aghconfig.AgentDef, workspace string) (string, error)
```

```go
// memory/dream.go — consolidation service
type SessionSpawner func(ctx context.Context, goal, prompt string) error

type Service struct { /* fields below */ }

func NewService(opts ...Option) *Service
func (s *Service) ShouldRun() (bool, error)
func (s *Service) Run(ctx context.Context, spawn SessionSpawner) error
```

```go
// session/types.go — NEW: session type enum
type SessionType string

const (
    SessionTypeUser   SessionType = "user"
    SessionTypeDream  SessionType = "dream"
    SessionTypeSystem SessionType = "system"
)
```

### Data Models

**Memory types** (closed taxonomy, ported from cc-memory):

```go
type MemoryType string

const (
    MemoryTypeUser      MemoryType = "user"
    MemoryTypeFeedback  MemoryType = "feedback"
    MemoryTypeProject   MemoryType = "project"
    MemoryTypeReference MemoryType = "reference"
)

type Scope string

const (
    ScopeGlobal    Scope = "global"
    ScopeWorkspace Scope = "workspace"
)
```

**Memory header** (parsed from file frontmatter):

```go
type MemoryHeader struct {
    Filename    string     `yaml:"-"`
    FilePath    string     `yaml:"-"`
    ModTime     time.Time  `yaml:"-"`
    Name        string     `yaml:"name"`
    Description string     `yaml:"description,omitempty"`
    Type        MemoryType `yaml:"type"`
    AgentName   string     `yaml:"agent_name,omitempty"` // NEW: for team memory tracking
}
```

**Memory file format** (on-disk):

```markdown
---
name: auth-middleware-decision
description: JWT chosen over session cookies for API auth
type: project
agent_name: claude
---

Team decided on JWT for API authentication because the service
is stateless and deployed across multiple regions. Session cookies
would require sticky sessions or a shared session store.

Decision made during session on 2026-04-02.
```

**MEMORY.md index format**:

```markdown
- [Auth Middleware Decision](auth-middleware-decision.md) — JWT chosen over session cookies
- [Testing Feedback](feedback-testing.md) — No mocks in integration tests
```

Capped at 200 lines / 25KB. Each entry under ~150 characters.

**Consolidation lock file** (`~/.agh/memory/.consolidate-lock`):

- Body: holder PID (plain text integer)
- mtime: serves as `lastConsolidatedAt` timestamp
- Stale detection: PID dead (via `syscall.Kill(pid, 0)`) OR lock older than 1 hour

**Staleness thresholds**:

| Age | Behavior |
|---|---|
| ≤ 1 day | No warning |
| > 1 day | Caveat appended: "This memory is N days old. Verify against current state before asserting as fact." |

**Session CreateOpts** (modified):

```go
type CreateOpts struct {
    AgentName string
    Name      string
    Workspace string
    Type      SessionType // NEW: defaults to SessionTypeUser
}
```

**MemoryContext** (internal to assembler):

```go
type MemoryContext struct {
    GlobalIndex    string // MEMORY.md content from global scope
    WorkspaceIndex string // MEMORY.md content from workspace scope
}
```

**Configuration additions** (TOML):

```toml
[memory]
enabled = true
global_dir = ""  # Override ~/.agh/memory/

[memory.dream]
enabled = true
agent = "claude"           # Agent to use for consolidation
min_hours = 24             # Minimum hours between consolidations
min_sessions = 3           # Minimum completed sessions since last consolidation
check_interval = "30m"     # Ticker interval for gate checks
```

**Config types**:

```go
type MemoryConfig struct {
    Enabled  bool        `toml:"enabled"`
    GlobalDir string     `toml:"global_dir,omitempty"`
    Dream    DreamConfig `toml:"dream"`
}

type DreamConfig struct {
    Enabled       bool          `toml:"enabled"`
    Agent         string        `toml:"agent"`
    MinHours      float64       `toml:"min_hours"`
    MinSessions   int           `toml:"min_sessions"`
    CheckInterval duration      `toml:"check_interval"`
}
```

**SQLite schemas**: No new tables. Memory is file-based. Dream sessions use existing session/event tables. The only addition is `session_type TEXT` column to the global `sessions` table:

```sql
ALTER TABLE sessions ADD COLUMN session_type TEXT NOT NULL DEFAULT 'user';
```

### Filesystem Layout

```
~/.agh/                                 # AGH_HOME
├── memory/                             # Global persistent memory (NEW)
│   ├── MEMORY.md                       # Global index
│   ├── user_preferences.md             # user type
│   ├── feedback_testing.md             # feedback type
│   └── .consolidate-lock               # Dream lock file
├── sessions/
│   ├── <xid>/
│   │   ├── events.db                   # Per-session SQLite (existing)
│   │   └── meta.json                   # Session metadata (existing)
│   └── <dream-xid>/                    # Dream sessions look like regular sessions
│       ├── events.db
│       └── meta.json
├── agh.db                              # Global SQLite (existing, +session_type column)
└── config.toml                         # +[memory] and [memory.dream] sections

<workspace>/.agh/                       # Workspace overlay
├── memory/                             # Workspace persistent memory (NEW)
│   ├── MEMORY.md                       # Workspace index
│   ├── project_auth_rewrite.md         # project type
│   └── reference_linear_board.md       # reference type
└── config.toml                         # Existing workspace config
```

### API Endpoints

**Memory HTTP/UDS API** (new routes, both HTTP and UDS):

| Method | Path | Description | Request | Response |
|---|---|---|---|---|
| GET | `/api/memory` | List memory headers | Query: `scope`, `workspace` | `[]MemoryHeader` |
| GET | `/api/memory/:filename` | Read memory file | Query: `scope`, `workspace` | `{content: string}` |
| PUT | `/api/memory/:filename` | Write memory file | Body: `{content, scope, workspace}` | `{ok: true}` |
| DELETE | `/api/memory/:filename` | Delete memory file | Query: `scope`, `workspace` | `{ok: true}` |
| POST | `/api/memory/consolidate` | Trigger dream consolidation | Body: `{workspace}` | `{triggered: bool, reason: string}` |

**Scope resolution** (when scope not specified):
- `user`, `feedback` → global
- `project`, `reference` → workspace

**Error mapping**:
- `os.ErrNotExist` → 404
- Validation errors (`scope`, `filename`, `content`) → 400
- All other → 500

**CLI commands** (`agh memory` subcommand group):

| Command | Description |
|---|---|
| `agh memory list [--scope global\|workspace]` | List headers (name, type, age, description) |
| `agh memory read <filename> [--scope]` | Read full memory file content |
| `agh memory write <filename> --type <type> --description <desc> [--scope] [--content <c>]` | Write/update memory. Content from stdin or `--content` flag |
| `agh memory delete <filename> [--scope]` | Delete memory file and remove from index |
| `agh memory consolidate` | Manually trigger dream consolidation |

### Dream Consolidation Prompt

Ported from cc-memory with adaptation for AGH v2 (reads session event databases instead of transcript files):

**4-phase consolidation guide**:

1. **Orient**: Read existing MEMORY.md indexes (global + workspace). Inspect relevant memory files. Review session count and time since last consolidation.

2. **Gather**: Read session event databases (`events.db`) from completed sessions since last consolidation. Extract high-signal content: user corrections, decisions, preferences, recurring patterns. Filter noise: routine tool calls, system messages, transient debugging.

3. **Consolidate**: Merge gathered signal into existing memory files. Prefer updating existing files over creating new ones. Convert relative dates to absolute. Resolve contradictions (newer wins). New files only for genuinely new topics. Keep each file focused on a single topic.

4. **Prune**: Remove duplicate entries across files. Delete obsolete memories (superseded by newer knowledge). Rebuild MEMORY.md indexes for both scopes. Keep each index under 200 lines / 25KB.

**Dream agent instructions**: The dream agent uses `agh memory list`, `agh memory read`, `agh memory write`, and `agh memory delete` CLI commands via `terminal/create` ACP calls. It reads session transcripts by querying event databases via `agh session events <id>` commands.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|---|---|---|---|
| `internal/memory/` | New | Memory store, dream service, consolidation lock, staleness, assembler. No risk — isolated new package | Implement and test |
| `internal/session/interfaces.go` | Modified | Add `PromptAssembler` interface. Low risk — additive, no existing code changes | Add interface definition |
| `internal/session/types.go` | Modified | Add `SessionType` enum and `CreateOpts.Type` field. Low risk — additive with default value | Add type and field |
| `internal/session/manager.go` | Modified | Call `PromptAssembler.Assemble()` in `Create()` before `driver.Start()`. Apply permission defaults based on `SessionType`. Medium risk — touches session creation hot path | Add assembly step, add WithPromptAssembler option |
| `internal/daemon/daemon.go` | Modified | Initialize memory Store, dream Service, periodic ticker. Wire PromptAssembler and SessionSpawner. Medium risk — touches boot sequence | Add initialization steps, dream ticker goroutine |
| `internal/config/config.go` | Modified | Add `MemoryConfig` and `DreamConfig` to Config struct. Add `HomePaths.MemoryDir`. Low risk — additive | Add config types and defaults |
| `internal/httpapi/` | Modified | Add memory endpoint handlers. Low risk — new routes only | Add handlers file |
| `internal/udsapi/` | Modified | Add memory endpoint handlers (mirrors httpapi). Low risk — new routes only | Add handlers file |
| `internal/cli/` | Modified | Add `agh memory` command group with 5 subcommands. Low risk — new commands only | Add memory.go file |
| `internal/store/global_db.go` | Modified | Add `session_type` column to sessions table schema. Low risk — additive column with default | Update schema, add migration |

## Testing Approach

### Unit Tests

**`internal/memory/` (memdir)**:
- Store operations (Write/Read/Delete/Scan) with `t.TempDir()`. Table-driven with global vs workspace scope, missing directories, invalid frontmatter, file not found
- Frontmatter parsing: valid 4-type taxonomy, missing required fields, unknown types, agent_name optional field
- Index management: `LoadIndex` truncation at 200 lines and 25KB, UTF-8 boundary handling, `wasTruncated` flag
- Scanner: sorted newest-first, 200-file cap, frontmatter parsing errors handled gracefully
- Staleness: `AgeDays()`, `AgeText()`, `FreshnessWarning()` with boundary tests at 0/1/2 day thresholds
- Filename validation: path separators rejected, `.` and `..` rejected

**`internal/memory/` (dream)**:
- Gate evaluation: `ShouldRun()` with each gate independently — time gate (mock mtime), session gate (mock session dirs), lock gate (mock lock file). Table-driven with all gate combinations
- Lock file: acquire/release, stale PID detection (dead process), mtime-as-timestamp, concurrent acquirers (two goroutines, one loses), rollback on failure
- Service: functional options, defaults validation, `Run()` with mock `SessionSpawner`

**`internal/memory/` (assembler)**:
- `Assemble()` with global-only, workspace-only, and both scopes populated
- Empty memory directories (no MEMORY.md exists) — returns unmodified agent prompt
- Agent prompt concatenation with memory context sections
- Staleness warnings in output

**`internal/session/` (modified)**:
- `SessionType` defaults to `SessionTypeUser` when not specified
- `CreateOpts` with `SessionTypeDream` applies `approve-all` permissions
- `PromptAssembler` integration: nil assembler returns raw agent prompt; non-nil assembler called with correct args

### Integration Tests

- **Memory + prompt assembly**: Write memory files to temp dir, create Manager with real Assembler, call `Create()`, verify the assembled prompt contains memory indexes in correct format
- **Dream lifecycle**: Boot test daemon, create/stop sessions, verify `ShouldRun()` gate evaluation, mock `SessionSpawner` to verify consolidation prompt and cleanup
- **CLI commands**: Mock daemon client, test all 5 `agh memory` subcommands with human and JSON output
- **HTTP/UDS API**: Start test server, exercise all 5 memory endpoints, verify CRUD and consolidation trigger

All tests follow project conventions: `t.Parallel()`, `t.TempDir()`, `t.Helper()`, `-race` flag, 80%+ coverage target.

## Development Sequencing

### Build Order

1. **`internal/memory/` — memdir core** (no dependencies on other AGH packages besides `config/` types)
   - Types: `MemoryType`, `MemoryHeader`, `Scope`, `Store`
   - Frontmatter parsing (YAML with `github.com/goccy/go-yaml`, already a project dependency)
   - Store operations: `Write`, `Read`, `Delete`, `Scan`, `LoadIndex`, `EnsureDirs`
   - Staleness: `AgeDays`, `AgeText`, `FreshnessWarning`
   - Filename and scope validation
   - Unit tests

2. **`internal/memory/` — dream service** (depends on step 1 for `Store` type)
   - Lock file: `ConsolidationLock` with acquire, release, stale detection, rollback
   - Gate evaluation: `ShouldRun()` with 3-gate ordering (time, session, lock)
   - `Service` struct with functional options
   - `SessionSpawner` callback type
   - `Run()` orchestration
   - Consolidation prompt template (4-phase markdown, embedded)
   - Session counting via `meta.json` scanning
   - Unit tests

3. **`internal/memory/` — assembler** (depends on step 1 for `Store.LoadIndex`)
   - `Assembler` struct implementing `session.PromptAssembler` interface
   - Memory context rendering (global + workspace sections with instructions)
   - Prompt concatenation with agent definition
   - Unit tests

4. **`internal/session/` — interface and type additions** (no dependencies on memory/)
   - `PromptAssembler` interface in `interfaces.go`
   - `SessionType` enum in `types.go`
   - `CreateOpts.Type` field with default
   - `WithPromptAssembler()` functional option in `manager.go`
   - Modify `Create()` to call assembler when non-nil
   - Apply permission overrides based on `SessionType`
   - Unit tests for new paths

5. **`internal/config/` — config additions** (depends on step 1 for understanding types)
   - `MemoryConfig` and `DreamConfig` in config struct
   - `HomePaths.MemoryDir` (`~/.agh/memory/`)
   - TOML defaults for `[memory]` and `[memory.dream]`
   - Unit tests for config loading with memory sections

6. **`internal/store/` — schema migration** (no dependencies on memory/)
   - Add `session_type` column to global `sessions` table
   - Update `RegisterSession` to include session type
   - Update `ListSessions` to return session type
   - Migration test

7. **`internal/daemon/` — composition root wiring** (depends on steps 1-6)
   - Initialize `memory.Store` with `HomePaths.MemoryDir`
   - Call `Store.EnsureDirs()` in boot sequence
   - Create `memory.Assembler` and inject via `session.WithPromptAssembler()`
   - Create `dream.Service` with configured options
   - Wire `SessionSpawner` callback that calls `Manager.Create(SessionTypeDream)`
   - Add periodic dream ticker goroutine (30min default) with context cancellation
   - Hook dream check on session stop (via Notifier fanout)
   - Integration tests

8. **`internal/httpapi/` + `internal/udsapi/` — memory API routes** (depends on steps 1, 7)
   - Memory handler: list, read, write, delete, consolidate
   - Route registration in server setup
   - Scope resolution and error mapping
   - Integration tests

9. **`internal/cli/` — memory commands** (depends on step 8)
   - `agh memory list/read/write/delete/consolidate` subcommands
   - Daemon client methods for memory HTTP API
   - Human and JSON output formatters
   - CLI tests

### Technical Dependencies

- **No new external dependencies** — `github.com/goccy/go-yaml` already in go.mod for frontmatter parsing, `syscall` for PID checking, `time.Ticker` for scheduling
- **Existing packages reused**: `config` (HomePaths, AgentDef), `store` (SessionMeta for session counting), `acp` (StartOpts for prompt injection)

## Monitoring and Observability

### Key Metrics

- Memory file count per scope (global, workspace)
- MEMORY.md index size (lines, bytes) per scope
- Dream consolidation: gate pass/fail rates, consolidation duration, success/failure count
- Dream session events recorded in event store (visible like regular sessions)
- Memory API call counts (list, read, write, delete, consolidate)

### Log Events

All structured via `slog` with consistent fields:

| Event | Fields | Level |
|---|---|---|
| Memory write | `scope`, `filename`, `type`, `agent_name` | Info |
| Memory delete | `scope`, `filename` | Info |
| Memory index truncated | `scope`, `lines`, `bytes`, `truncated_at` | Warn |
| Dream gate check | `time_gate`, `session_gate`, `lock_gate`, `passed` | Debug |
| Dream consolidation start | `workspace`, `sessions_since`, `hours_since` | Info |
| Dream consolidation complete | `workspace`, `duration_ms`, `memories_written` | Info |
| Dream consolidation failed | `workspace`, `error`, `duration_ms` | Error |
| Dream lock acquired | `prior_mtime` | Debug |
| Dream lock released | `new_mtime` | Debug |
| Dream lock stale reclaimed | `stale_pid`, `age_hours` | Warn |
| Prompt assembly | `workspace`, `global_index_lines`, `workspace_index_lines` | Debug |

### Health Endpoint Extension

`GET /api/observe/health` returns additional fields:

```json
{
  "memory": {
    "global_files": 12,
    "workspace_files": 5,
    "last_consolidation": "2026-04-04T03:30:00Z",
    "dream_enabled": true
  }
}
```

## Technical Considerations

### Key Decisions

See Architecture Decision Records below for full rationale on each decision.

1. **PromptAssembler in session/**: Interface defined where consumed, implemented by memory/. Keeps session decoupled from memory while allowing prompt enrichment.
2. **Frozen snapshot injection**: Memory loaded once at session start. Preserves prompt cache, prevents mid-conversation context shifts.
3. **Dream-only extraction**: No per-turn background extraction. Agents write manually, dream catches patterns in bulk. Lower cost, proven pattern.
4. **Four-type taxonomy**: Proven from cc-memory. Procedural memory deferred to skills phase to avoid format conflicts.
5. **SessionSpawner callback**: Dream decoupled from session via callback function. Daemon wires the concrete implementation.
6. **SessionType enum**: Dream sessions get `approve-all` permissions. Prevents permission model from blocking memory writes during consolidation.

### Known Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Dream agent not suitable for consolidation (poor summarization) | Medium | Configurable `dream.agent` in TOML. Default to Claude (strongest summarization). Allow override. |
| Memory files grow unbounded | Low | Dream pruning phase actively manages size. MEMORY.md hard cap at 200 lines / 25KB. Scanner cap at 200 files. |
| Dream consumes user session slot | Low | Dream sessions count toward `max_sessions`. Check available capacity before spawning. Skip consolidation if at limit. |
| Concurrent memory writes (two agents in same workspace) | Low | File-level writes are atomic (write-to-temp + rename). MEMORY.md index managed by dream only. |
| ACP agent can't run `agh memory` CLI | Medium | Depends on `terminal/create` ACP support. If agent doesn't support terminal, memory writes fail silently. Log warning. |
| Dream prompt is too large for some agents | Low | Dream prompt is ~2KB. Well within context limits. Can be trimmed if needed. |

### Failure Handling

**Daemon restart with dream in progress**:
1. Dream lock has PID in body. On restart, check if PID is alive.
2. If PID dead and lock age < 1 hour: remove stale lock, allow next dream cycle.
3. If PID dead and lock age ≥ 1 hour: remove stale lock (already handled by stale age).
4. Memory files are always in consistent state (atomic writes).

**Dream session crash**:
- Detected via `cmd.Wait()` returning non-zero exit (same as any session crash).
- Lock rollback: `Rollback(priorMtime)` restores previous consolidation timestamp.
- Next dream cycle re-attempts consolidation.

**Memory file corruption**:
- Frontmatter parsing errors: skip file with warning log, continue scanning.
- Missing MEMORY.md: return empty string from `LoadIndex()`, not an error.
- Missing memory directory: `EnsureDirs()` creates it at boot.

## Architecture Decision Records

- [ADR-001: Interleaved Extensibility — Build Memory With Intentional Seams, Defer Formal Plugin System](adrs/adr-001.md) — Build memory with clean interfaces but without a formal plugin architecture
- [ADR-002: PromptAssembler Interface in session/ With memory/ Implementation](adrs/adr-002.md) — Define assembly interface where consumed, implement in memory package
- [ADR-003: Frozen Snapshot Memory Injection With Dream-Only Extraction](adrs/adr-003.md) — Load memory once at session start, extract via periodic dream consolidation only
- [ADR-004: Four-Type Memory Taxonomy — Drop Procedural, Defer to Skills Phase](adrs/adr-004.md) — Keep proven 4-type taxonomy, avoid format conflict with future skills system
