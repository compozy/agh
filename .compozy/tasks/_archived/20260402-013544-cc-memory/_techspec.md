# TechSpec: Cross-Session Memory System

## Executive Summary

This TechSpec defines a persistent memory system for AGH that enables agents to build and recall institutional knowledge across sessions. The system comprises three subsystems: (1) a file-based persistent memory store (`memdir`) with dual global/workspace directories, YAML frontmatter metadata, and MEMORY.md indexes injected into agent prompts; (2) a dream consolidation service that spawns ephemeral agent sessions to synthesize session transcripts and blackboard entries into durable memory files; and (3) team memory that enhances the existing blackboard with memory-typed entries for structured cross-agent knowledge sharing within sessions.

Key architectural decisions: dual storage follows the existing config layering pattern (global + workspace); dream consolidation uses ephemeral sessions rather than direct LLM calls (preserving the kernel's orchestration-only role); team memory reuses the blackboard (no new tables); scheduling uses `time.Ticker` (no external cron library); and only MEMORY.md indexes are injected into prompts (agents read full content on demand via CLI).

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        AGH Kernel                           │
│                                                             │
│  ┌──────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │   memdir     │  │  dream           │  │  team memory  │  │
│  │  (package)   │  │  (package)       │  │  (blackboard) │  │
│  │              │  │                  │  │              │  │
│  │ • Store      │  │ • DreamService   │  │ • Entry type │  │
│  │ • Scanner    │  │ • ConsolidLock   │  │   "memory"   │  │
│  │ • Staleness  │  │ • 3-gate trigger │  │ • Frontmatter│  │
│  │ • CLI cmds   │  │ • Session spawn  │  │   in content │  │
│  └──────┬───────┘  └────────┬─────────┘  └──────┬───────┘  │
│         │                   │                    │          │
│  ┌──────┴───────────────────┴────────────────────┴───────┐  │
│  │              Prompt Assembler (modified)               │  │
│  │  + MEMORY.md indexes injection (global + workspace)   │  │
│  │  + Team memory entries injection                      │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Data flow:**

1. **Write path**: Agents write persistent memories via `agh memory write` CLI command → kernel HTTP API → `memdir.Store.Write()` → filesystem. Team memories via `agh state append --type memory` → existing blackboard path.
2. **Read path (prompt)**: On agent spawn, prompt assembler calls `memdir.Store.LoadIndex()` for both scopes + queries blackboard for `type="memory"` entries → injects into system prompt.
3. **Read path (on demand)**: Agents call `agh memory read <filename>` → kernel HTTP API → `memdir.Store.Read()` → content returned to agent.
4. **Consolidation path**: Daemon ticker → `DreamService.ShouldRun()` (3-gate check) → if gates pass, `SessionSpawner` callback creates ephemeral session → worker agent reads transcripts + memories → writes updated memory files → session auto-stops.

**Package layout:**

| New Package | Location | Responsibility |
|---|---|---|
| `memdir` | `internal/kernel/memdir/` | Memory file store, scanner, staleness, index management |
| `dream` | `internal/kernel/dream/` | Consolidation service, lock, gate evaluation |

**Storage layout:**

```
~/.agh/memory/                          # global persistent memory
  MEMORY.md                             # global index (always in prompt)
  user_preferences.md                   # user type
  feedback_testing.md                   # feedback type
  .consolidate-lock                     # dream lock file (mtime = lastConsolidatedAt)

<workspace>/.agh/memory/                # workspace persistent memory
  MEMORY.md                             # workspace index (always in prompt)
  project_auth_rewrite.md               # project type
  reference_linear_board.md             # reference type

~/.agh/sessions/<id>/session.db         # team memory (blackboard type="memory")
```

## Implementation Design

### Core Interfaces

**Memory Store** (`internal/kernel/memdir/memdir.go`):

```go
type Store struct {
    globalDir    string
    workspaceDir string
    maxIndexLines int // 200
    maxIndexBytes int // 25000
}

func NewStore(globalDir, workspaceDir string) *Store
func (s *Store) Scan(scope Scope) ([]MemoryHeader, error)
func (s *Store) Read(scope Scope, filename string) ([]byte, error)
func (s *Store) Write(scope Scope, filename string, content []byte) error
func (s *Store) Delete(scope Scope, filename string) error
func (s *Store) LoadIndex(scope Scope) (content string, truncated bool, err error)
func (s *Store) EnsureDirs() error
```

**Dream Service** (`internal/kernel/dream/dream.go`):

```go
type SessionSpawner func(ctx context.Context, goal, prompt string) error

type DreamService struct {
    memStore    *memdir.Store
    sessionsDir string
    lockPath    string
    minHours    float64
    minSessions int
    logger      *slog.Logger
}

func New(opts ...Option) *DreamService
func (d *DreamService) ShouldRun() (bool, error)
func (d *DreamService) Run(ctx context.Context, spawn SessionSpawner) error
```

**Staleness** (`internal/kernel/memdir/staleness.go`):

```go
func AgeDays(modTime time.Time) int
func AgeText(modTime time.Time) string
func FreshnessWarning(modTime time.Time) string
```

**Consolidation Lock** (`internal/kernel/dream/lock.go`):

```go
type ConsolidationLock struct {
    path     string
    staleAge time.Duration // 1 hour
}

func NewLock(path string) *ConsolidationLock
func (l *ConsolidationLock) LastConsolidatedAt() (time.Time, error)
func (l *ConsolidationLock) TryAcquire() (priorMtime time.Time, ok bool, err error)
func (l *ConsolidationLock) Release() error
func (l *ConsolidationLock) Rollback(priorMtime time.Time) error
```

### Data Models

**Memory types** (closed taxonomy):

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
    Filename    string
    FilePath    string
    ModTime     time.Time
    Name        string
    Description string
    Type        MemoryType
}
```

**Memory file format** (on-disk):

```markdown
---
name: user-preferences
description: Senior Go engineer, prefers table-driven tests
type: user
---

User is a senior Go engineer with 10+ years experience.
Prefers explicit error handling over panic/recover.
```

**MEMORY.md index format**:

```markdown
- [User Preferences](user_preferences.md) — Senior Go engineer
- [Testing Feedback](feedback_testing.md) — No mocks in integration tests
```

Capped at 200 lines / 25KB. Each entry should be under ~150 characters.

**Consolidation lock file** (`~/.agh/memory/.consolidate-lock`):

- Body: holder PID (plain text integer)
- mtime: serves as `lastConsolidatedAt` timestamp
- Stale detection: PID dead (via `syscall.Kill(pid, 0)`) OR lock older than 1 hour

**Team memory** (existing `state.BlackboardEntry`):

```go
// No new types. Uses existing BlackboardEntry with type="memory".
// content field contains YAML frontmatter + markdown body.
BlackboardEntry{
    Scope:   workgroupID,
    Author:  agentName,
    Type:    "memory",
    Content: "---\nname: auth-decision\ndescription: JWT chosen\ntype: project\n---\nContent...",
}
```

**Staleness thresholds**:

| Age | Behavior |
|---|---|
| ≤ 1 day | No warning appended |
| > 1 day | Caveat: "This memory is N days old. Verify against current state before asserting as fact." |

**Prompt context model** (new field in `AssembleOptions`):

```go
type MemoryContext struct {
    GlobalIndex    string // MEMORY.md content from ~/.agh/memory/
    WorkspaceIndex string // MEMORY.md content from <workspace>/.agh/memory/
    TeamMemories   string // Formatted blackboard entries where type="memory"
}
```

### API Endpoints

**Kernel HTTP API** (UDS, following existing patterns in `api.go`):

| Method | Path | Description | Request | Response |
|---|---|---|---|---|
| `GET` | `/api/memory` | List memory headers | Query: `scope` (global\|workspace) | `[]MemoryHeader` |
| `GET` | `/api/memory/:filename` | Read memory file | Query: `scope` | `{content: string}` |
| `PUT` | `/api/memory/:filename` | Write memory file | Body: `{content, scope}` | `{ok: true}` |
| `DELETE` | `/api/memory/:filename` | Delete memory file | Query: `scope` | `{ok: true}` |
| `POST` | `/api/memory/consolidate` | Trigger dream consolidation | None | `{triggered: bool, reason: string}` |

**CLI commands** (`agh memory` subcommand group):

| Command | Description |
|---|---|
| `agh memory list [--scope global\|workspace]` | List memory headers (name, type, age, description) |
| `agh memory read <filename> [--scope global\|workspace]` | Read full memory file content |
| `agh memory write <filename> --type <type> --description <desc> [--scope]` | Write/update memory. Content from stdin or `--content` flag |
| `agh memory delete <filename> [--scope global\|workspace]` | Delete memory file and remove from index |
| `agh memory consolidate` | Manually trigger dream consolidation |

**Scope defaults** (when `--scope` is omitted):
- `user`, `feedback` → global
- `project`, `reference` → workspace

**Team memory** uses existing CLI:
- Write: `agh state append --type memory --content '---\nname: ...\n---\nContent'`
- Read: `agh state read --type memory`

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|---|---|---|---|
| `internal/kernel/memdir/` | new | Memory store package. No risk — isolated new code | Implement and test |
| `internal/kernel/dream/` | new | Dream consolidation package. Low risk — isolated, uses callback pattern | Implement and test |
| `internal/prompt/assembler.go` | modified | Add `MemoryContext` to `AssembleOptions`, inject memory sections. Low risk — additive change, existing tests verify ordering | Add field, inject section, update tests |
| `internal/kernel/session_manager.go` | modified | Initialize memdir store on session create, build `MemoryContext` for prompts, trigger dream on session stop. Medium risk — touches session lifecycle | Add init/teardown, update tests |
| `internal/kernel/kernel.go` | modified | Initialize DreamService at boot, add periodic ticker to lifecycle goroutine. Low risk — additive | Add init, ticker goroutine |
| `internal/kernel/api.go` | modified | Register memory HTTP endpoints. Low risk — additive routes | Add handlers, tests |
| `internal/cli/` | modified | Add `agh memory` command group with 5 subcommands, daemon client methods. Low risk — follows existing patterns | New file `memory.go`, update `root.go` |
| `internal/cli/human/` | modified | Add human renderers for memory list/read. Low risk | Add render functions |
| `internal/state/` | none | No changes. Team memory uses existing blackboard with `type="memory"` convention | None |

## Testing Approach

### Unit Tests

**`internal/kernel/memdir/`:**
- Store operations (Write/Read/Delete/Scan) with `t.TempDir()`. Table-driven with global vs workspace scope, missing directories, invalid frontmatter, file not found, permission errors
- Index management: LoadIndex truncation at 200 lines and 25KB, `wasTruncated` flag accuracy
- Scanner: sorted newest-first, 200-file cap, frontmatter parsing, malformed file handling
- Staleness: `AgeDays()`, `AgeText()`, `FreshnessWarning()` with boundary tests at 0/1/2 day thresholds
- Frontmatter parsing: valid 4-type taxonomy, missing fields, unknown types, empty content

**`internal/kernel/dream/`:**
- Gate evaluation: `ShouldRun()` with each gate independently — time gate (mock mtime), session gate (mock session dirs), lock gate (mock lock file). Table-driven with all gate combinations
- Lock file: acquire/release, stale PID detection (dead process), mtime-as-timestamp, concurrent acquirers (two writers, one loses), rollback on failure
- DreamService: functional options, defaults validation

### Integration Tests

- Prompt assembler + memdir: write memory files to temp dir, call `Assemble()` with `MemoryContext`, verify output contains both indexes in correct section order
- Session lifecycle + dream: boot test kernel, create/stop session, verify `DreamService.ShouldRun()` is called, mock `SessionSpawner` to verify consolidation prompt
- CLI commands: mock daemon client, test all 5 `agh memory` subcommands with both human and TOON output modes
- End-to-end dream: boot kernel, write blackboard entries and session dirs, trigger consolidation, verify memory files created

All tests follow project conventions: `t.Parallel()`, `t.TempDir()`, `t.Helper()`, hand-written interface mocks (no testify), `-race` flag.

## Development Sequencing

### Build Order

1. **`internal/kernel/memdir/` — core package** (no dependencies on other new code)
   - Types: `MemoryType`, `MemoryHeader`, `Scope`, `Store`
   - Frontmatter parsing (reuse existing `internal/frontmatter/` package)
   - Store operations: `Write`, `Read`, `Delete`, `Scan`, `LoadIndex`, `EnsureDirs`
   - Staleness: `AgeDays`, `AgeText`, `FreshnessWarning`
   - Unit tests

2. **`internal/kernel/dream/` — consolidation service** (depends on step 1)
   - Lock file: acquire, release, stale detection, rollback, mtime-as-state
   - Gate evaluation: `ShouldRun()` with 3-gate ordering
   - `DreamService` struct with functional options
   - Consolidation prompt template (4-phase markdown)
   - `SessionSpawner` callback type
   - Unit tests

3. **Prompt assembler integration** (depends on step 1)
   - Add `MemoryContext` to `AssembleOptions`
   - Inject global + workspace MEMORY.md indexes as additional section
   - Inject team memory entries (blackboard type="memory") as additional section
   - Add memory instructions (types, what NOT to save, staleness policy, CLI commands)
   - Integration tests

4. **Kernel integration** (depends on steps 1, 2, 3)
   - Initialize `memdir.Store` in `Session.Create()` with global + workspace paths
   - Build `MemoryContext` when assembling agent prompts
   - Query blackboard for team memory entries, include in `MemoryContext`
   - Initialize `DreamService` at kernel boot
   - Add periodic `time.Ticker` in daemon lifecycle goroutine (30-min default)
   - Hook `DreamService.ShouldRun()` + spawn on session stop
   - Register HTTP API routes
   - Integration tests

5. **CLI commands** (depends on step 4)
   - Add `agh memory` command group: `list`, `read`, `write`, `delete`, `consolidate`
   - Add daemon client methods for memory HTTP API
   - Human + TOON output renderers
   - CLI tests with mock daemon client

### Technical Dependencies

- **No external dependencies needed** — `time.Ticker` for scheduling, existing `internal/frontmatter/` for YAML parsing, existing `internal/state/` for team memory
- **Existing packages reused**: `frontmatter`, `state`, `prompt`, `config` (HomePaths)
- **No blocking external deliverables**

## Monitoring and Observability

- **Structured logs** (slog): memory write/read/delete operations, dream gate evaluations (which gate passed/failed), lock acquire/release/rollback, consolidation session spawn/complete/fail, index truncation warnings
- **Event store**: dream consolidation start/complete/fail events written to SQLite events table via existing `state.AppendEvent()`
- **Dashboard**: consolidation sessions visible as regular sessions in the dashboard topology view (goal: "memory-consolidation")
- **Metrics** (future): memory file count per scope, index size, consolidation frequency, gate pass rate. Not implemented in v1 — logs provide sufficient observability

## Technical Considerations

### Key Decisions

1. **Dual storage (global + workspace)**: Follows existing config layering. User/feedback memories are global; project/reference are workspace-scoped. See [ADR-001](adrs/adr-001.md).
2. **Dream via ephemeral session**: Preserves kernel's orchestration-only role. Uses existing session infrastructure for lifecycle, resilience, and observability. See [ADR-002](adrs/adr-002.md).
3. **Team memory via blackboard**: Zero new infrastructure. Reuses existing `type` field with `"memory"` convention. See [ADR-003](adrs/adr-003.md).
4. **time.Ticker over cron library**: Zero dependencies, sufficient for single fixed-interval task. See [ADR-004](adrs/adr-004.md).
5. **Index-only prompt injection**: Constant prompt size, agents read full content on demand. See [ADR-005](adrs/adr-005.md).

### Known Risks

1. **Scope ambiguity**: Agents may write project memories to global scope. Mitigated by defaulting scope based on memory type.
2. **Consolidation session slot**: Dream session counts against `max_sessions`. Mitigated by checking available slots before spawning and skipping if at capacity.
3. **Lock file corruption**: Power loss during consolidation could leave a stale lock. Mitigated by stale PID detection and 1-hour age timeout.
4. **Index bloat**: Without pruning, MEMORY.md could exceed 200 lines. Mitigated by dream consolidation's prune phase and truncation with warning at load time.
5. **Team memory noise**: Malformed frontmatter in blackboard entries. Mitigated by validating at prompt injection time and skipping with warning log.

## Architecture Decision Records

- [ADR-001: Dual Storage — Global + Workspace Memory Directories](adrs/adr-001.md) — Memory files split between ~/.agh/memory/ (global) and <workspace>/.agh/memory/ (project-scoped)
- [ADR-002: Dream Consolidation via Ephemeral Agent Session](adrs/adr-002.md) — Consolidation runs as a spawned session with restricted worker, not direct LLM calls
- [ADR-003: Team Memory via Blackboard Enhancement](adrs/adr-003.md) — Session-scoped team memory reuses blackboard with type="memory" convention
- [ADR-004: time.Ticker Over Cron Library for Dream Scheduling](adrs/adr-004.md) — Plain goroutine ticker for periodic dream checks, no external scheduler dependency
- [ADR-005: MEMORY.md Index Injection Over Full Content Loading](adrs/adr-005.md) — Only indexes are injected into prompts; agents read full content via CLI on demand
