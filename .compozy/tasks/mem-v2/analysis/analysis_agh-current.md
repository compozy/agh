# Analysis: AGH Current Memory Subsystem (Forensic Audit)

**Date**: 2026-05-04
**Scope**: read-only audit of `internal/memory/`, its persistence backend, the daemon seams, agent-operable surfaces, hooks integration, extension contract, configuration, and tests. Cross-referenced with `docs/_memory/` (institutional memory), `internal/CLAUDE.md`, and `CLAUDE.md`.
**Method**: direct file reads + `grep` against the codebase. No web search. No code modifications.

---

## 1. TL;DR

AGH today ships a **dual-scope, file-backed, frontmatter-validated Markdown memory store** (`global` + `workspace`) with a derived **SQLite + FTS5 catalog** for search and operation history, an **MEMORY.md prompt index** assembled into the system prompt, a bounded **recall augmenter** that prepends top-3 search hits to live user messages, and a **dream consolidation runtime** gated by a Time → Sessions → File-lock cascade that spawns a one-shot consolidation session against the configured agent. The control plane is solid: structured CLI verbs (`agh memory list|read|search|write|delete|reindex|consolidate|history|health`), full HTTP+UDS parity under `/api/memory/*`, partial native tool surfaces (read-only set: `agh__memory_list|read|search|history`), and a typed Host API for extensions (`memory/recall|store|forget`) with declared `memory.backend` capability. The on-disk schema versions through the existing `memory_schema_migrations` registry; mutations are mutex-serialized with atomic file writes; the consolidation lock is hardlink-PID-based with stale reclaim and rollback.

**Strengths**: clean four-type taxonomy (`user|feedback|project|reference`) with default-scope routing; full agent-manageability via CLI/HTTP/UDS/Host-API parity (with the gaps below); FTS5 + bm25 ranking; mutation observability through `memory_operation_log` joined into the canonical observability spine; truthful staleness warnings; lock rollback semantics that preserve mtime-as-`last_consolidated_at`.

**Biggest gaps**: (1) **no agent scope** — RFC 002 + glossary call out `agent | workspace | global` but only two scopes exist in code; (2) **no agent-local memory directory** despite RFC 001 promising `.agents/<name>/memory/`; (3) **no compaction integration** — `runContextCompaction` exists in `session/` with hooks plumbing, but is wired only in tests and never produces or persists durable summaries into memory; (4) **`memory.backend` extension capability is declared but unconsumed** — daemon never delegates to extensions, the Host API only exposes recall/store/forget against the local store; (5) **PreCompact / PostCompact / SessionStart / context.assemble hooks never read or write durable memory**; (6) the `memory.write|delete|reindex|consolidate` native tools are missing — agents cannot mutate memory through the tools binding; (7) no per-agent override and no `memory.scope` field on `AgentDef`; (8) consolidation gates default to `min_hours=24`, `min_sessions=3` but are not workspace-aware in the gate evaluation (only in the spawner's workspace selection); (9) no workspace-id qualifier — `workspace_root` is a path, not a stable workspace ID; (10) the future-only seams (`ContextRefResolver`, `ProviderHookRunner`) are typed but explicitly forbidden from runtime prompt assembly by `interfaces_test.go`.

---

## 2. Surface Inventory — `internal/memory/`

| File | LOC | One-line purpose | Key types / functions (`path:line`) |
|---|---|---|---|
| `types.go` | 293 | Public types, taxonomy + scope enums, `Backend` interface, future `ContextRef*` and `ProviderHook*` seams | `Type`/`MemoryTypeUser|Feedback|Project|Reference` (`types.go:14-23`); `Scope`/`ScopeGlobal|Workspace` (`types.go:26-33`); `Operation` constants (`types.go:38-47`); `Header` (`types.go:50-58`); `Backend` interface (`types.go:125-134`); `ContextRefResolver` (`types.go:182-184`); `ProviderHookRunner` (`types.go:215-217`); `DefaultScopeForType` (`types.go:237-248`) |
| `document.go` | 25 | Frontmatter parsing entry; canonical lock path helper | `ParseHeader` (`document.go:9-20`); `ConsolidationLockPath` (`document.go:23-25`) |
| `store.go` | 1232 | The `Store` (file backend) — list/read/write/delete/scan/index/search/reindex/health/history; index sync; validation; FS layout helpers | `Store` struct (`store.go:38-46`); `NewStore` + `WithCatalogDatabasePath` (`store.go:51-81`); `ForWorkspace` (`store.go:84-88`); `Read|Write|Delete|Exists` (`store.go:120-206`); `Scan` (`store.go:209-294`); `LoadIndex` (`store.go:297-340`); `Search` (`store.go:343-390`); `Reindex` (`store.go:393-430`); `HealthStats` (`store.go:433-493`); `History` (`store.go:496-512`); `lockMutations` (`store.go:1036-1042`); `truncateIndex` (`store.go:1044-1072`); workspace memory dir derivation (`store.go:1216-1223`) |
| `catalog.go` | 1270 | SQLite-backed derived catalog: `memory_catalog_entries` + FTS5 vtable + `memory_operation_log` + state KV; bm25 search; mutation write txns; migrations | `catalogSchemaStatements` (`catalog.go:34-87`); `catalogSchemaMigrations` (`catalog.go:89-101`); `catalog` struct (`catalog.go:103-110`); `catalogDocument` (`catalog.go:112-124`); `replaceScope`/`upsertDocument`/`deleteDocument` (`catalog.go:238-465`); `search` (`catalog.go:607-670`); `logEvent` (`catalog.go:672-718`); `listOperations` (`catalog.go:720-789`); `operationStats` (`catalog.go:791-842`); `buildCatalogMatchQuery` + `quoteCatalogMatchTerm` (`catalog.go:991-1005`); `buildCatalogDocument` + `hashMemoryContent` (`catalog.go:1039-1066`); `fallbackSearchDocuments` (`catalog.go:1068-1108`); `deriveWorkspaceRoot` (`catalog.go:1260-1270`) |
| `assembler.go` | 167 | `Assembler` — `PromptProvider`/`PromptAssembler` impl that prepends global+workspace MEMORY.md indexes plus taxonomy/commands/staleness sections to the system prompt | `Assembler` struct (`assembler.go:37-39`); constants for `memoryPromptIntro|memoryTaxonomySection|memoryCommandsSection|memoryStalenessSection` (`assembler.go:14-33`); `NewAssembler` (`assembler.go:43-46`); `PromptSection` (`assembler.go:50-95`); `Assemble` (`assembler.go:98-117`); `renderMemoryContext` (`assembler.go:126-146`) |
| `recall.go` | 101 | `NewRecallAugmenter` — `session.PromptInputAugmenter` that runs lexical/FTS search against the message and prepends top-3 hits with freshness warnings | `maxRecallResults=3`, `maxRecallCharacters=1500`, `RecallAugmenterBudget` (`recall.go:13-17`); `NewRecallAugmenter` (`recall.go:22-59`); `buildRecallBlock` (`recall.go:61-101`) |
| `staleness.go` | 45 | Calendar-day staleness math; `FreshnessWarning` returned when memory is ≥2 days old | `ageDays`/`ageText` (`staleness.go:8-26`); `freshnessWarning` (`staleness.go:28-35`); `FreshnessWarning` (`staleness.go:38-40`) |
| `prompt.go` | 56 | The four-phase consolidation prompt template embedded into the daemon | `consolidationPromptTemplate` (`prompt.go:5-51`); `ConsolidationPrompt` (`prompt.go:54-56`) |
| `dream.go` | 456 | `Service` — gate evaluator + Run orchestrator. Spawns one consolidation session and rolls back the lock on failure; counts completed sessions from `~/.agh/sessions/<id>/meta.json` | `defaultMinHours=24`, `defaultMinSessions=3`, `defaultGoal="memory-consolidation"` (`dream.go:18-22`); `Service` struct (`dream.go:45-64`); `NewService` (`dream.go:74-100`); `WithMemoryStore`/`WithSessionsDir`/`WithMinHours`/`WithMinSessions`/`WithLockPath`/`WithLogger`/`WithWorkspaceResolver` (`dream.go:102-161`); `ShouldRun` (`dream.go:172-213`); `Run` (`dream.go:217-262`); `prepareWorkspace` (`dream.go:264-287`); `scanCompletedSessionsSince` (`dream.go:314-361`); `acquireLock` + `ensureLock` + `completeRun` (`dream.go:373-432`) |
| `lock.go` | 273 | `ConsolidationLock` — hardlink-+PID file with stale reclaim, rollback, mtime-as-last-consolidated-at | `ConsolidationLock` struct (`lock.go:23-29`); `NewConsolidationLock` (`lock.go:32-42`); `LastConsolidatedAt` (`lock.go:45-59`); `TryAcquire` (`lock.go:63-120`); `Release` (`lock.go:123-129`); `Rollback` (`lock.go:132-144`); `canReclaim` (`lock.go:207-215`); `createLockFile` (hardlink atomic) (`lock.go:242-273`) |
| `consolidation/runtime.go` | 464 | Background dream runtime — ticker + queued check requests; `EnqueueCheck` for hook-driven re-evaluation; `NewSessionSpawner` derives recent workspaces from session metadata when no explicit workspace is given | `Service`/`SessionManager` interfaces (`consolidation/runtime.go:22-36`); `Runtime` (`consolidation/runtime.go:38-51`); `NewRuntime` (`consolidation/runtime.go:60-80`); `Trigger` (`consolidation/runtime.go:96-116`); `Start` (`consolidation/runtime.go:119-158`); `EnqueueCheck` (`consolidation/runtime.go:161-183`); `Shutdown` (`consolidation/runtime.go:186-201`); `runCheck` (`consolidation/runtime.go:203-247`); `NewSessionSpawner` (`consolidation/runtime.go:250-283`); `resolveWorkspaces` (`consolidation/runtime.go:285-364`) |
| `interfaces_test.go` | 58 | Compile-time assertions for `ContextRefResolver` + `ProviderHookRunner`, **plus a guard test** that fails if `assembler.go` or `recall.go` ever references those future seams | `TestFutureInterfacesRemainOutOfRuntimePromptAssembly` (`interfaces_test.go:34-58`) |
| `store_test.go` | 2028 | 28 top-level tests covering write/read/delete/scan/index sync/search/reindex/operation history/concurrency/migrations/dirs/staleness | — |
| `dream_test.go` | 878 | 22 top-level tests covering gate evaluation, lock rollback, workspace resolution, concurrent Run serialization, session-counter | — |
| `lock_test.go` | 373 | 17 top-level tests covering acquire/release/rollback/stale reclaim/concurrent acquisition/PID validation | — |
| `assembler_test.go` | 350 | 11 tests covering global-only, workspace-only, both-indexes, taxonomy section, commands section, staleness section, regression equivalence with `PromptSection` | — |
| `recall_test.go` | 133 | 3 tests covering empty session/query, recall block prepending, score-zero filtering, max-results cap, staleness | — |
| `perf_bench_test.go` | 153 | Benchmarks: `BenchmarkStoreScanCappedWorkspace` (512 files capped at 200), `BenchmarkAssemblerPromptSectionDualIndex`, `BenchmarkScanCompletedSessionsSince` | — |
| `consolidation/runtime_test.go` | 814 | 14 tests + ticker/EnqueueCheck/SessionSpawner/workspace-resolution coverage | — |
| `consolidation/perf_bench_test.go` | 58 | `BenchmarkResolveWorkspacesRecentSessions` | — |

Total package: **9,227 LOC** including tests; **~3,500 LOC** of production code. Grossly under-documented compared to its surface area.

---

## 3. Memory Taxonomy as Currently Implemented

### Closed type taxonomy (`internal/memory/types.go:14-23`)

```go
const (
    MemoryTypeUser      Type = "user"
    MemoryTypeFeedback  Type = "feedback"
    MemoryTypeProject   Type = "project"
    MemoryTypeReference Type = "reference"
)
```

Each type's default persistence scope is hard-coded (`internal/memory/types.go:237-248`):

```go
func DefaultScopeForType(t Type) (Scope, error) {
    switch t.Normalize() {
    case MemoryTypeUser, MemoryTypeFeedback:
        return ScopeGlobal, nil
    case MemoryTypeProject, MemoryTypeReference:
        return ScopeWorkspace, nil
```

### Scopes (`internal/memory/types.go:28-33`)

```go
const (
    ScopeGlobal    Scope = "global"
    ScopeWorkspace Scope = "workspace"
)
```

**Only two scopes exist.** No `agent` scope despite the glossary at `docs/_memory/glossary.md:138` listing it:

> Default write scope is declared per agent in `memory.scope`.

This `memory.scope` agent field also does not exist in `internal/config/agent.go` (no `Memory*` struct on `AgentDef`).

### Mapping to standard taxonomy (working / episodic / semantic / procedural)

| Standard class | AGH equivalent | Storage |
|---|---|---|
| Working | None — only the live ACP turn buffer + `events.db` per session | per-session SQLite (`internal/store/sessiondb/`) |
| Episodic | None as durable memory. Session events are journaled in `events.db`; the dream consolidation prompt asks the consolidator to "review recent completed session artifacts" but there is no persisted episodic abstraction | event store + `goal=memory-consolidation` session output |
| Semantic | `MemoryTypeUser`, `MemoryTypeReference`, `MemoryTypeProject` | `~/.agh/memory/` + `<workspace>/.agh/memory/` |
| Procedural | None in the memory package. Skills (`internal/skills/`) play this role today (capabilities = procedural memory) but live in an entirely separate package and are not unified with `internal/memory/` | `bundled` + `~/.agh/skills/` + `<workspace>/.agh/skills/` |
| Reflective / feedback | `MemoryTypeFeedback` | `~/.agh/memory/` (forced global by `DefaultScopeForType`) |

The taxonomy is closed (`Type.Validate` rejects anything outside the four constants — `internal/memory/types.go:225-234`); a fifth "type" cannot be introduced without code changes.

### Glossary cross-reference (`docs/_memory/glossary.md:127-144`)

```
### Memory Types (taxonomy)
Per RFC 002 / Claude Code AutoDream / AGH `internal/memory/consolidation/`:
…
### Memory Scopes
- agent — per-agent
- workspace — workspace-wide
- global — host-wide
Default write scope is declared per agent in `memory.scope`.
```

**Drift**: glossary specifies three scopes; code implements two. RFC 001 promises `.agents/<name>/memory/`; no such directory or scope is plumbed.

---

## 4. Persistence Backend

### On-disk file layout

```
~/.agh/                                      <- AGH home (configurable via AGH_HOME)
  memory/                                    <- ScopeGlobal directory (config: memory.global_dir)
    MEMORY.md                                <- prompt-safe index
    *.md                                     <- one file per memory document, with YAML frontmatter
    .consolidate-lock                        <- consolidation lock + last_consolidated_at mtime
  agh.db                                     <- SQLite catalog DB (shared global database)
  sessions/<session_id>/meta.json            <- read by dream gate to count completed sessions
<workspace_root>/
  .agh/                                      <- workspace AGH dir (config.DirName)
    memory/                                  <- ScopeWorkspace directory
      MEMORY.md
      *.md
```

Constants pinning these names live in `internal/memory/store.go:22-30`:

```go
const (
    indexFilename     = "MEMORY.md"
    maxScanEntries    = 200
    defaultIndexLines = 200
    defaultIndexBytes = 25_000
    dirPerm           = 0o755
    filePerm          = 0o644
    memoryDirName     = "memory"
)
```

Workspace memory dir derivation (`internal/memory/store.go:1216-1223`):

```go
func workspaceMemoryDir(workspaceRoot string) string {
    return filepath.Join(filepath.Clean(trimmed), aghconfig.DirName, memoryDirName)
}
```

### SQLite schema (`internal/memory/catalog.go:34-87`)

```sql
CREATE TABLE IF NOT EXISTS memory_catalog_entries (
    id             TEXT PRIMARY KEY,
    scope          TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
    workspace_id   TEXT NOT NULL DEFAULT '',
    workspace_root TEXT NOT NULL DEFAULT '',
    filename       TEXT NOT NULL,
    type           TEXT NOT NULL,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    content        TEXT NOT NULL,
    content_hash   TEXT NOT NULL,
    updated_at     TEXT NOT NULL,
    UNIQUE (scope, workspace_root, filename)
);
CREATE INDEX IF NOT EXISTS idx_memory_catalog_scope          ON memory_catalog_entries(scope);
CREATE INDEX IF NOT EXISTS idx_memory_catalog_workspace_root ON memory_catalog_entries(workspace_root);
CREATE INDEX IF NOT EXISTS idx_memory_catalog_updated_at     ON memory_catalog_entries(updated_at);

CREATE VIRTUAL TABLE IF NOT EXISTS memory_catalog_fts USING fts5(
    name, description, content,
    content='memory_catalog_entries',
    content_rowid='rowid',
    tokenize='porter unicode61'
);
-- AI / AD / AU triggers keep the FTS shadow in sync with memory_catalog_entries

CREATE TABLE IF NOT EXISTS memory_catalog_state (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_operation_log (
    id         TEXT PRIMARY KEY,
    type       TEXT NOT NULL,
    agent_name TEXT NOT NULL DEFAULT 'daemon',
    summary    TEXT NOT NULL DEFAULT '',
    timestamp  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_memory_operation_log_type      ON memory_operation_log(type);
CREATE INDEX IF NOT EXISTS idx_memory_operation_log_timestamp ON memory_operation_log(timestamp);
```

Migration `add_memory_operation_scope` (catalog v2, `internal/memory/catalog.go:96-101` + `internal/memory/catalog.go:173-206`) adds the `scope`, `workspace_root`, `filename` columns to `memory_operation_log` plus their indexes. The same evolution exists in the global DB schema (`internal/store/globaldb/global_db.go:546-551`, migration v6 `add_memory_operation_scope` with checksum `2026-04-25-add-memory-operation-scope`).

`memory_operation_log` is unioned into the canonical observability spine (`internal/store/globaldb/global_db_observe.go:93-105`), so memory mutations flow through `/api/observe/events?type=memory.write|memory.delete|memory.search|memory.reindex`.

### Catalog migrations (`internal/memory/catalog.go:89-101`)

```go
var catalogSchemaMigrations = []storepkg.Migration{
    {Version: 1, Name: "initial_memory_catalog_schema", Statements: catalogSchemaStatements},
    {
        Version:  2,
        Name:     "add_memory_operation_scope",
        Checksum: "catalog-add-memory-operation-scope-v1",
        Up:       migrateCatalogOperationScope,
    },
}
```

Catalog DB is **the same SQLite file** as the daemon's global DB (`d.homePaths.DatabaseFile` / `~/.agh/agh.db`) — wired in `internal/daemon/boot.go:285-288`:

```go
state.memoryStore = memory.NewStore(
    state.globalMemoryDir,
    memory.WithCatalogDatabasePath(d.homePaths.DatabaseFile),
)
```

Migrations table is `memory_schema_migrations` (separate from the global migrations table).

### Frontmatter format (per file)

Validated by `internal/memory/types.go:281-293` (`Header.Validate`), reading these YAML fields (`internal/memory/types.go:50-58`):

```go
type Header struct {
    Filename    string    `json:"filename"              yaml:"-"`
    FilePath    string    `json:"-"                     yaml:"-"`
    ModTime     time.Time `json:"mod_time"              yaml:"-"`
    Name        string    `json:"name"                  yaml:"name"`
    Description string    `json:"description,omitempty" yaml:"description,omitempty"`
    Type        Type      `json:"type"                  yaml:"type"`
    AgentName   string    `json:"agent_name,omitempty"  yaml:"agent_name,omitempty"`
}
```

Strict YAML decode (`internal/memory/store.go:1225-1232`):

```go
return frontmatter.Decode(content, func(data []byte) error {
    if err := yaml.UnmarshalWithOptions(data, dest, yaml.Strict()); err != nil {
        return fmt.Errorf("decode YAML: %w", err)
    }
    return nil
})
```

`Name` and `Type` are required; `Description` and `AgentName` are optional.

---

## 5. Public Go Interfaces & API Seams

### `Backend` — the canonical memory surface (`internal/memory/types.go:125-134`)

```go
type Backend interface {
    List(scope Scope) ([]Header, error)
    Read(scope Scope, filename string) ([]byte, error)
    Write(scope Scope, filename string, content []byte) error
    Delete(scope Scope, filename string) error
    Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
    Reindex(ctx context.Context, opts ReindexOptions) (ReindexResult, error)
    History(ctx context.Context, query OperationHistoryQuery) ([]OperationRecord, error)
    LoadPromptIndex(scope Scope) (content string, truncated bool, err error)
}
```

`*memory.Store` satisfies this with a compile-time assertion (`internal/memory/store.go:48`):

```go
var _ Backend = (*Store)(nil)
```

But — and this is load-bearing for memory v2 design — **nothing else implements it**. There is no in-memory test fake of `Backend` in this package (tests use the concrete `*Store` against `t.TempDir()`), and no extension or alternate backend wires it in. The Host API directly closes over `*memory.Store`, not over `Backend`.

### Future / explicitly-not-wired seams (`internal/memory/types.go:136-217`)

```go
type ContextRefResolver interface {
    Resolve(ctx context.Context, refs []ContextRef, budget TokenBudget) (ResolvedContext, error)
}

type ProviderHookRunner interface {
    RunMemoryHook(ctx context.Context, req ProviderHookRequest) (ProviderHookResult, error)
}
```

These are guarded by `interfaces_test.go:34-58`, which **fails the test if `assembler.go` or `recall.go` ever imports the symbol**:

```go
for _, forbidden := range []string{
    "ContextRefResolver", "ProviderHookRunner", "RunMemoryHook",
    "Resolve(ctx context.Context, refs []ContextRef",
} {
    if strings.Contains(source, forbidden) {
        t.Fatalf("%s references future interface %q; Task 07 must not wire prompt integration", filename, forbidden)
    }
}
```

So the seams are reserved for a future Task 07 (now memory v2) that explicitly opts in.

### Consumer-side interfaces (defined where consumed, Go-style)

**Session prompt seams** (`internal/session/interfaces.go:74-75, 351-354`):

```go
type PromptInputAugmenter func(ctx context.Context, session *Session, message string) (string, error)

type PromptAssembler interface {
    Assemble(ctx context.Context, agent aghconfig.AgentDef, workspace *workspacepkg.ResolvedWorkspace) (string, error)
}
```

`*memory.Assembler` implements `PromptAssembler` via `internal/memory/assembler.go:41`:

```go
var _ session.PromptProvider = (*Assembler)(nil)
```

(`PromptProvider` is the per-section composable variant defined in `internal/session/prompt_provider.go`.)

**Consolidation seam** (`internal/memory/consolidation/runtime.go:22-36`):

```go
type Service interface {
    ShouldRun() (bool, error)
    Run(ctx context.Context, spawn memory.SessionSpawner, workspace string) error
}
type ServiceFactory func(opts ...memory.Option) Service
type SessionManager interface {
    Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
    ListAll(ctx context.Context) ([]*session.Info, error)
    Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
    Stop(ctx context.Context, id string) error
}
```

**Composition root wiring** — `internal/daemon/boot.go:280-340`, `internal/daemon/boot.go:468-482`, `internal/daemon/boot.go:816-834`:

```go
if state.cfg.Memory.Enabled {
    state.memoryStore = memory.NewStore(
        state.globalMemoryDir,
        memory.WithCatalogDatabasePath(d.homePaths.DatabaseFile),
    )
    if err := state.memoryStore.EnsureDirs(); err != nil { … }
    prependProviders = append(prependProviders, memory.NewAssembler(state.memoryStore))
}
…
state.promptAugmenter, err = newPromptInputCompositeAugmenter(
    state.logger, state.harnessResolver, state.harnessRecorder,
    defaultPromptInputAugmenterDescriptors(
        memory.NewRecallAugmenter(state.memoryStore),
        state.situationContext.Augment,
    )...,
)
…
if state.cfg.Memory.Enabled && state.cfg.Memory.Dream.Enabled {
    state.dreamSvc = d.newDreamService(
        memory.WithMemoryStore(state.memoryStore),
        memory.WithSessionsDir(d.homePaths.SessionsDir),
        memory.WithMinHours(state.cfg.Memory.Dream.MinHours),
        memory.WithMinSessions(state.cfg.Memory.Dream.MinSessions),
        memory.WithLogger(state.logger),
        memory.WithWorkspaceResolver(state.workspaceResolver),
    )
}
```

The dream runtime is constructed late in boot (after the session manager exists) and wired into native hooks at `internal/daemon/boot.go:1178` via `daemonNativeHooks(state.lifecycleObservers, state.dreamRuntime)`.

---

## 6. Read Path

### Two reads happen per turn

**A — System-prompt assembly (once per session start)**

`Assembler.Assemble` (`internal/memory/assembler.go:98-117`) prepends a context block ahead of the resolved agent prompt. The block contains:

1. `# Persistent Memory` intro (`assembler.go:14-16`).
2. `## Global MEMORY.md Index` truncated to ≤ 200 lines / ≤ 25 000 bytes (`assembler.go:148-160`, `store.go:1044-1072`).
3. `## Workspace MEMORY.md Index` (same truncation policy).
4. `## Memory Taxonomy` listing the four types (`assembler.go:17-22`).
5. `## Memory Commands` — embedded CLI cheat-sheet for the agent (`assembler.go:23-29`).
6. `## Staleness Policy` — "Memories older than 1 day should be verified" (`assembler.go:30-33`).

Index loading (`store.go:297-340`) prefers the on-disk `MEMORY.md`. If the file is missing or its hash doesn't match the headers in the directory, the store **synthesizes** the index from `Scan` results and warns. There is no on-the-fly compaction or summarization in this read path.

**B — Per-message recall (every user turn)**

`NewRecallAugmenter` (`internal/memory/recall.go:22-59`) is wired as a `session.PromptInputAugmenter`. For every non-empty user message:

1. Resolve workspace from `sess.Info().Workspace` and create a `Store.ForWorkspace` view.
2. Call `Search(ctx, query, SearchOptions{Workspace: workspaceRoot, Limit: 3})`.
3. Call `buildRecallBlock`, which:
   - Skips zero-score entries (`recall.go:69-71`).
   - For each result, emits `- <name> [<scope>]` + optional `Snippet:` + optional `Freshness:` (`recall.go:72-79`).
   - Hard-caps at 3 entries and 1 500 characters total (`recall.go:13-17, 86-89`).
4. Prepends `Relevant durable memory for this turn:` + entries + closing reminder, then `\n\nUser message:\n` + the original.

The augmenter has `Order=100` (runs first) and `Budget=1500` with `BudgetBehavior="trim"` (`internal/daemon/prompt_input_composite.go:60-69`). The composite augmenter respects a budget aggregated across all augmenters and trims down rather than dropping when exceeded (`internal/daemon/prompt_input_composite.go:452-490`).

### Search algorithm

**Primary** (`internal/memory/store.go:343-390` → `internal/memory/catalog.go:607-670`):

1. Tokenize the query into letter/number runs (`catalog.go:1007-1025`).
2. Quote-escape each term and `AND`-join into an FTS5 MATCH expression (`catalog.go:991-1005`).
3. Run `MATCH ?` with `bm25(memory_catalog_fts)` ordering ASC, falling back to `updated_at DESC, filename ASC`.
4. Apply scope filter: `e.scope = 'global'`, or `e.scope = 'workspace' AND e.workspace_root = ?`, or both with `OR` (`catalog.go:871-888`).
5. Score = `-bm25(...)` (negated so higher = better).
6. Snippet from the FTS5 `snippet()` function with `[`/`]` markers (`catalog.go:638`); falls back to description.

**Fallback (no catalog wired or FTS returns 0 hits)** (`internal/memory/catalog.go:1068-1108`):

- Iterates documents in scope, scoring `+ count(term)` plus `+5` for name match, `+2` for description match.
- Sort by score desc, then `updated_at` desc, then filename asc.

### Scoring is purely lexical

There is no embedding-based ranking, no semantic similarity, no LLM rerank. Recall composition is byte-budget bounded, not token-bounded.

### Staleness ("freshness") handling

`internal/memory/staleness.go:28-35`:

```go
func freshnessWarning(modTime time.Time, now time.Time) string {
    age := ageDays(modTime, now)
    if age <= 1 { return "" }
    return fmt.Sprintf("This memory is %d days old. Verify against current state before asserting as fact.", age)
}
```

Day boundary uses `calendarDayNumber` (`staleness.go:42-45`) — pure UTC noon-day diff. No exponential decay, no half-life, no relevance recency boost in the FTS scoring — recency is only a tie-breaker.

---

## 7. Write Path

### Three write paths

1. **Direct mutation** — operator/agent calls `Backend.Write/Delete` via CLI/HTTP/UDS/Host-API/native-tool. Validates frontmatter, locks (`store.lockMutations` mutex), atomic writes (`fileutil.AtomicWriteFile`), syncs MEMORY.md, upserts the catalog entry, logs to `memory_operation_log`. (`internal/memory/store.go:152-206`).
2. **Reindex** — `Reindex` rescans the on-disk Markdown source of truth and replaces the catalog scope (`store.go:393-430` → `catalog.replaceScope`). Treated as the recovery path for catalog drift.
3. **Dream consolidation** — a one-shot AGH session is spawned with the embedded prompt. The session is expected to call back through the same `Backend.Write/Delete` API. There is no separate "consolidation write" code path — the consolidator is just another agent.

### Mutation guards

- Frontmatter must validate before file write (`store.go:151-160`).
- Filenames must not contain `/` or `\\`, must not be `.` or `..`, must be non-empty (`store.go:1197-1210`).
- Atomic file replace via `fileutil.AtomicWriteFile`.
- Mutex serializes all mutations on a single `*Store`: `store.go:1036-1042` — but **the mutex is per-Store-instance**. `Store.ForWorkspace` returns a clone (`store.go:84-88`) that **shares** the `*sync.Mutex` via the dereferenced `*sync.Mutex` field; multiple `Store` instances opened against the same global dir would NOT share locks (an `os.MkdirAll` race today is implicitly trusted to be idempotent).
- After mutation, `syncScopeAfterWriteErr` either updates the index incrementally or, if the index is missing while other docs exist, does a full rescan (`store.go:631-653`, `store.go:715-741`).
- Catalog write txns are serialized by `c.writeMu` (`internal/memory/catalog.go:282-321`).

### What writes memory automatically

**Nothing today.** The dream consolidator writes through the runtime, but a session must be spawned for that to happen. No hook automatically writes memory on `session.post_stop`, `context.post_compact`, `prompt.post_assemble`, `tool.post_call`, etc. The sole automatic side-effect is the **dream check enqueue on `session.post_stop`** (`internal/daemon/hooks_bridge.go:1237-1253`), which only triggers gate evaluation and may or may not actually run the consolidator.

### What survives across sessions

Only durable memory files plus their derived catalog. No "working set" carries across.

---

## 8. Compaction

### What exists

The session package has typed compaction hooks plumbing:

`internal/session/manager_hooks.go:492-565`:

```go
func (m *Manager) runContextCompaction(
    ctx context.Context, session *Session, turnID string,
    reason string, strategy string, summary string,
    contextBlocks []hookspkg.ContextBlock,
    compact func(context.Context, hookspkg.ContextPreCompactPayload) (hookspkg.ContextPostCompactPayload, error),
) (hookspkg.ContextPostCompactPayload, error) {
    …
    prePayload, err = m.hooks.compaction().DispatchContextPreCompact(ctx, prePayload)
    …
    postPayload, err := compact(ctx, prePayload)   // <-- caller-supplied compactor
    …
    if _, err := m.hooks.compaction().DispatchContextPostCompact(ctx, postPayload); err != nil { … }
}
```

`internal/session/hooks.go:84-93` defines:

```go
type CompactionHooks interface {
    DispatchContextPreCompact(ctx context.Context, payload hookspkg.ContextPreCompactPayload) (hookspkg.ContextPreCompactPayload, error)
    DispatchContextPostCompact(ctx context.Context, payload hookspkg.ContextPostCompactPayload) (hookspkg.ContextPostCompactPayload, error)
}
```

The hook events themselves (`internal/hooks/events.go:107-108`) are wired through `internal/daemon/hooks_bridge.go:733-746` (`HookContextPreCompact`, `HookContextPostCompact`).

### What is missing

`runContextCompaction` is **only invoked from tests** — `internal/session/manager_integration_test.go:1091` and `internal/session/manager_hooks_test.go:1073`. **No production code path calls it.** No compactor is registered. The hook events fire only if some other system (today: nothing) calls them.

There is no AGH-side prompt compaction, no token-budget enforcement on the agent transcript, no "summary into memory" path. Long-running sessions rely entirely on whatever compaction the underlying ACP agent (Claude Code, Hermes, etc.) performs internally.

### How compaction would interact with memory if wired

The plumbing is shape-compatible: `ContextPostCompactPayload.Summary` could become a memory write target. The `runContextCompaction` accepts an arbitrary `compact` function, so an LLM-driven summarizer could be plugged in. But **no current implementation does this**.

---

## 9. Lifecycle

### Hydration on session start

`Assembler.Assemble` runs once per session start, reads MEMORY.md (or synthesizes from headers), and prepends to the prompt. **No** per-turn hydration of memory bodies. **No** caching layer — every prompt assembly re-reads the index file from disk.

### Persistence on shutdown

There is **no shutdown persistence step** for memory. All writes are file-atomic at the time they happen. The dream runtime has a `Shutdown()` method (`internal/memory/consolidation/runtime.go:186-201`) that cancels the ticker and `WaitGroup`-joins the loop goroutine; it does not flush state.

### Daemon shutdown discipline

`internal/daemon/daemon.go:1216` and `:1235` shut down the dream runtime. The catalog DB closes through the shared `agh.db` lifecycle owned by the global store.

### Migration

Catalog uses two numbered migrations (`memory_schema_migrations` table). The global DB has its own `add_memory_operation_scope` (v6). Both follow the project's mandatory numbered-migration discipline (`docs/_memory/lessons/L-008-schema-migrations-mandatory.md`).

### GC / pruning

**No explicit garbage collection.** `Scan` caps at 200 entries (`maxScanEntries`, `store.go:24`) but this is just a query cap, not a deletion. The prompt template instructs the consolidator to "Remove duplicate, obsolete, or low-signal memory content" (`prompt.go:39-41`) but pruning is delegated to the agent — no automatic pruning policy.

### Lock discipline (`internal/memory/lock.go`)

The consolidation lock is a hardlink-PID file. `TryAcquire` (`lock.go:63-120`):

1. Read existing state (PID + mtime).
2. If lock is held by a live PID and not stale (`canReclaim` checks `staleAge=1h` and `processAlive`), return `false`.
3. Otherwise remove the existing lock, write to a temp file, hardlink the temp into `path`, verify ownership.
4. On any failure mid-acquisition, restore the prior mtime via `restoreUnlocked`.

`Release` writes an empty file with `now()` mtime (the new "last consolidated at"). `Rollback(priorMtime)` restores the prior mtime so a failed run does not move the gate forward. This is well-tested (17 tests, including liveness reclaim, corrupt-body reclaim, concurrent acquisition serialization).

### Cleanup discipline

`Service.completeRun` (`dream.go:411-432`) always pairs lock acquisition with `Release` on success or `Rollback(priorMtime)` on failure, in a `defer-`ed inner function pattern. The lock state struct (`pending`, `priorMtime`) is mutex-protected (`s.mu`).

---

## 10. Hooks Integration

### Hook events that touch memory today

| Hook event | Memory effect |
|---|---|
| `session.post_stop` | Enqueues a dream check via `dreamSessionStopExecutor` (`internal/daemon/hooks_bridge.go:1237-1253`). Excludes Dream-typed sessions to prevent self-recursion. Calls `dreamRuntime.EnqueueCheck("session_stop", payload.WorkspaceID)`. |
| All other hook events | None — no memory hook is registered. |

The `daemonNativeHooks` registration (`internal/daemon/hooks_bridge.go:1255-1305`) registers exactly three native hooks:

1. `daemon.observe.session_post_create` — observe lifecycle.
2. `daemon.observe.session_post_stop` — observe lifecycle.
3. `daemon.dream.session_post_stop` — enqueue a dream check.

### Hook events that **could** touch memory but don't

The taxonomy in `internal/hooks/events.go:50-130` includes:

- `HookContextPreCompact` (`events.go:107`) — defined, payload struct exists, dispatched only when session code explicitly calls `runContextCompaction`. No memory side-effect.
- `HookContextPostCompact` (`events.go:108`) — same.
- `HookPromptPostAssemble` (`events.go:69`) — fires inside `manager_hooks.go:277` after prompt assembly. No memory inspection or augmentation.
- `HookSessionPreCreate / HookSessionPostCreate / HookSessionPreResume / HookSessionPostResume` — no memory hydration hook.
- `HookSessionPreStop / HookSessionPostStop` — only the dream check enqueue.
- `HookTurnStart / HookTurnEnd / HookMessageStart / HookMessageEnd` — no memory side-effect.

### No "context.assemble" event

There is no `HookContextAssemble` event family. Memory injection into the system prompt happens via the `PromptProvider` composition root (`internal/daemon/boot.go:318-327`), not via hooks. The `Assembler.Assemble` and `Assembler.PromptSection` paths are direct function calls.

### RFC 002 lifecycle hooks (`on_session_created`, `on_session_stopped`)

`internal/CLAUDE.md` Memory & Skills section documents:

> **Lifecycle hooks** (`on_session_created`, `on_session_stopped`) execute in hierarchy precedence then alphabetical order; configurable timeout (default 5s); fail-open semantics (errors logged, never block); JSON over stdin.

These are skill hook events — **not** memory hook events. They do not write to durable memory. They are routed through `internal/hooks/` skill hook executors.

---

## 11. Agent-Manageability

### CLI commands (`internal/cli/memory.go:91-511`)

| Command | Verb | Output | Notes |
|---|---|---|---|
| `agh memory list [--scope]` | GET | `memoryListItem[]` JSON | Auto-includes both global + workspace when no scope flag |
| `agh memory health` | GET | `MemoryHealthPayload` | Status + counts + last consolidation |
| `agh memory history [--scope --operation --since --limit]` | GET | `memoryHistoryItem[]` | Bounded, redacted operation log |
| `agh memory search <terms...> [--scope --workspace --limit]` | GET | `memorySearchItem[]` | Lexical/FTS5 match |
| `agh memory read <filename> [--scope]` | GET | `memoryReadView` | Full file content |
| `agh memory write <filename> --type --description [--content / stdin] [--scope]` | PUT | `memoryMutationView` | `--type` and `--description` required |
| `agh memory delete <filename> [--scope]` | DELETE | `memoryMutationView` | |
| `agh memory reindex [--scope]` | POST | `memoryReindexView` | Rebuilds catalog |
| `agh memory consolidate` | POST | `memoryMutationView` (with reason) | Manual dream trigger |

All commands respect `-o json` / `-o text`. Filename arg validation is via `exactOneNonBlankArg`.

### HTTP routes (`internal/api/httpapi/routes.go:240-251`)

```go
memoryGroup.GET("",            handlers.ListMemory)
memoryGroup.GET("/health",     handlers.MemoryHealth)
memoryGroup.GET("/history",    handlers.MemoryHistory)
memoryGroup.GET("/search",     handlers.SearchMemory)
memoryGroup.POST("/reindex",   handlers.ReindexMemory)
memoryGroup.POST("/consolidate", handlers.ConsolidateMemory)
memoryGroup.GET("/:filename",  handlers.ReadMemory)
memoryGroup.PUT("/:filename",  handlers.WriteMemory)
memoryGroup.DELETE("/:filename", handlers.DeleteMemory)
```

### UDS routes (`internal/api/udsapi/routes.go:270-282`)

Identical group/path/verb shape to HTTP — full parity. UDS is for the local CLI; HTTP is for the web UI and remote agents.

### Native tools (extension/agent-callable, `internal/daemon/native_tools.go:606-627`)

| Tool ID | Coverage |
|---|---|
| `agh__memory_list` | yes (`tools/builtin_ids.go:54`) |
| `agh__memory_read` | yes |
| `agh__memory_search` | yes |
| `agh__memory_history` | yes |
| `agh__memory_write` | **missing** |
| `agh__memory_delete` | **missing** |
| `agh__memory_reindex` | **missing** |
| `agh__memory_consolidate` | **missing** |

The native tool surface is **read-only**. To mutate memory, an agent must call the Host API (extensions) or shell out to `agh memory write` (which is heavier and depends on the CLI binary being available in the sandbox).

### Host API (`internal/extension/host_api.go:524-526`, handlers at `:1109-1217`)

```go
"memory/forget": handler.handleMemoryForget,
"memory/recall": handler.handleMemoryRecall,
"memory/store":  handler.handleMemoryStore,
```

`handleMemoryStore` validates key/content, infers scope (workspace if explicit, else global), renders frontmatter (synthesizing `Name` from filename, `Description` from first line, `Type` from tags or workspace fallback) and calls `Store.Write`. `handleMemoryRecall` does an in-process Scan + per-document `scoreMemoryRecall` (reading file body each time) — **not** the catalog's FTS5 path, so it duplicates the search effort instead of reusing it. `handleMemoryForget` calls `Store.Delete`.

### Agent-manageability gaps

- Native tools cannot mutate (write/delete/reindex/consolidate).
- Host API recall ignores the FTS5 catalog and uses its own scorer (`extension/host_api.go:2318` `scoreMemoryRecall`).
- No way for an agent to express agent-scoped memory because there is no agent scope.
- No way for an agent to query "memory at time T" or "memory history filtered by `agent_name`" beyond the operation log.

---

## 12. Extensibility

### Declared capability

`internal/extension/protocol/host_api.go:12-13`:

```go
const (
    // CapabilityProvideMemoryBackend is the provide surface for daemon-managed memory backends.
    CapabilityProvideMemoryBackend = "memory.backend"
)
```

### Negotiated services for that capability (`protocol/host_api.go:185-198`)

```go
var capabilityServiceMethods = map[string][]ExtensionServiceMethod{
    CapabilityProvideMemoryBackend: {
        ExtensionServiceMethodMemoryStore,
        ExtensionServiceMethodMemoryRecall,
        ExtensionServiceMethodMemoryForget,
    },
    …
}
```

### Where it is consumed

**Nowhere.** A `grep -rn "memory.backend\|MemoryBackend"` returns only the protocol definition and tests that assert the capability declaration is honored at handshake time. No daemon path delegates `Backend.Search/Read/Write/Delete` to an extension. The `*memory.Store` is used directly. So the extensibility surface is **declarative only** — an extension can announce it provides `memory.backend`, but nothing in AGH today routes calls to it.

### Other extensibility surfaces that could matter

- **MCP servers** — none of the bundled tools target memory.
- **Skills** — there is no `memory.skill` lifecycle hook for skills to write memory; skills are procedural memory but have no integration with `internal/memory/`.
- **Hooks** — see Section 10. No memory-specific hook event is defined.

### Memory is only consumed by:

- `internal/cli/` (memory.go + client.go).
- `internal/api/core/` (handlers + DTOs + spec.go OpenAPI definitions).
- `internal/api/httpapi/`, `internal/api/udsapi/` (route registration).
- `internal/daemon/` (boot, native_tools, prompt_input_composite, settings).
- `internal/extension/` (Host API).
- `internal/memory/consolidation/` (dream runtime).

**No** consumption from `internal/skills/`, `internal/situation/`, `internal/soul/`, `internal/automation/`, `internal/coordinator/`, `internal/scheduler/`, or `internal/network/`.

---

## 13. Config Surface

### Top-level (`internal/config/config.go:161-175, 477-487`)

```toml
[memory]
enabled    = true
global_dir = "<homePaths.MemoryDir>"     # default: ~/.agh/memory

[memory.dream]
enabled        = true
agent          = "<DefaultAgentName>"
min_hours      = 24
min_sessions   = 3
check_interval = "30m"
```

Validation (`config.go:1080-1083, 1276-1286`):

```go
func (c MemoryConfig) Validate() error { return c.Dream.Validate() }
…
if strings.TrimSpace(c.Agent) == "" { return errors.New("memory.dream.agent is required") }
if c.MinHours <= 0 { return … }
if c.MinSessions <= 0 { return … }
if c.CheckInterval <= 0 { return … }
```

### Bootstrap default (`internal/config/bootstrap.go:74`)

`memory.dream.agent` is auto-set during `agh init` to whichever default agent the operator chose.

### Tool surface (agent-callable config keys, `internal/config/tool_surface.go:94-99`)

```go
"memory.enabled":              ConfigValueBool,
"memory.dream.enabled":        ConfigValueBool,
"memory.dream.agent":          ConfigValueString,
"memory.dream.min_hours":      ConfigValueFloat,
"memory.dream.min_sessions":   ConfigValueInt,
"memory.dream.check_interval": ConfigValueDuration,
```

These keys are exposed to `agh config get/set` and through the corresponding native config tools.

### Missing config

- No `memory.scope` agent-default (RFC 002 mandate).
- No `memory.recall.*` knob (the recall augmenter's `maxRecallResults=3`, `maxRecallCharacters=1500`, `maxScanEntries=200`, index size limits are all hard-coded constants).
- No `memory.indexing.*` for the FTS catalog.
- No `memory.compaction.*` because compaction does not exist.
- No `memory.staleness.*` — the 1-day threshold is hard-coded.

---

## 14. Test Coverage

### What is tested

- **`store_test.go`**: 28 top-level tests + many subtests covering write / read / delete / scan / index / search / reindex / operation history / concurrency / catalog migration / dirs / staleness / `Exists` / explicit-workspace normalization / mutation-success-when-derived-sync-fails (graceful degradation of the catalog).
- **`dream_test.go`**: 22 tests on gates, lock rollback, workspace resolution, concurrent Run serialization, session counter, `withGoal`, joined errors, `ErrLockUnavailable`.
- **`lock_test.go`**: 17 tests on acquire/release/rollback/stale reclaim/concurrent acquisition (including 16-goroutine race), PID validation, parent-is-file, `LastConsolidatedAt`, restore-unlocked.
- **`assembler_test.go`**: 11 tests covering global-only, workspace-only, both-indexes, taxonomy, commands, staleness sections, regression equivalence with `PromptSection`.
- **`recall_test.go`**: 3 tests covering nil session, recall block prepending, score-zero filtering, max-results cap, staleness annotation.
- **`consolidation/runtime_test.go`**: 14 tests covering Trigger states, ticker loop, `EnqueueCheck`, disabled runtime no-op, error paths, `NewSessionSpawner`, workspace ref resolution, recent-workspace derivation, `IsPathLikeWorkspaceRef`, `ResolveWorkspaceRef`, `SpawnSessionWrapsPromptAndStopErrors`.
- **Daemon-level integration**: `daemon_memory_e2e_integration_test.go` exercises CLI + HTTP + observe events end-to-end (write/search/reindex/observe spine consumption).
- **Benchmarks**: `BenchmarkStoreScanCappedWorkspace`, `BenchmarkAssemblerPromptSectionDualIndex`, `BenchmarkScanCompletedSessionsSince`, `BenchmarkResolveWorkspacesRecentSessions`.

### What is NOT tested

- **No race-flag explicit gating** — but tests use `t.Parallel()` widely, so `-race` covers the package by default.
- **No fuzz tests** for frontmatter parsing or filename validation.
- **No property tests** for the FTS5 query builder. The `quoteCatalogMatchTerm` only escapes `"` — adversarial queries could still produce surprising MATCH semantics.
- **No tests for `Backend` interface conformance from external implementations** (because there are none).
- **No tests for catalog drift detection** beyond `Reindex` and `HealthStats.OrphanedFiles`.
- **No tests** for the `ContextRefResolver` / `ProviderHookRunner` future seams beyond the "do not import" guard.
- **No compaction integration tests** at the memory layer (because compaction does not write memory).

### Discipline against the AGH testing rules (`agh-test-conventions`)

- Every subtest uses `t.Run("Should …", …)` ✅ (verified in `recall_test.go:38, 79`, `assembler_test.go:14, 141, 196`, etc.).
- `t.Parallel()` is the default ✅.
- Status-code AND body assertions are present at the API level (E2E integration test verifies search results, observe summaries, and reindex counts).
- Coverage floor (80%) — not measured here but the package is heavily exercised.

---

## 15. Lessons / Standing Directives Constraining Memory

### `internal/CLAUDE.md` (Memory & Skills Runtime, RFC-backed)

> **Five-layer skill/memory/agent precedence**: Bundled → Marketplace → User → Additional → Workspace, with agent-local overriding all. Higher precedence wins on collision; an audit trail logs every shadow.
> **Memory taxonomy**: `user | feedback | project | reference` types; scopes `agent | workspace | global`. Default write scope declared per agent in `memory.scope`.
> **Memory consolidation gates**: Time → Sessions → Lock cascade ordered by computational cost. Default gates: 24h, 5 touched sessions, file-lock. **Never replace gates with naive heuristics.**
> **Lifecycle hooks** (`on_session_created`, `on_session_stopped`) execute in hierarchy precedence then alphabetical order; configurable timeout (default 5s); fail-open semantics (errors logged, never block); JSON over stdin.

**Drift to flag for v2**: the runtime today says `min_sessions=3` (default in `dream.go:20`) — the documented default is **5**. Either the doc is wrong or the code regressed.

### `docs/_memory/glossary.md:127-160`

Authoritative term definitions. Memory types are exactly the four implemented; scopes are documented as `agent | workspace | global`. Glossary trumps older RFCs per CLAUDE.md "Authoritative when older RFCs / ledgers conflict."

### `docs/_memory/standing_directives.md`

- **SD-002** (Greenfield, no legacy): no compat shims for memory v1 — when v2 ships, delete v1.
- **SD-008** (Composition root discipline): only `internal/daemon/` wires components. Memory v2's wiring belongs in `daemon/boot.go`, not in subordinate packages.
- **SD-009** (Data exists / consumer missing): the `Backend` interface and `memory.backend` extension capability are "data shape, no consumer" exemplars — v2 should build the consumer (delegating to extensions, agent scope), not redesign the data.
- **SD-011** (Extensible and agent-manageable): every memory v2 change must close the loop end-to-end across CLI + HTTP + UDS + native-tools + Host-API + extension contract + config + docs.

### `docs/_memory/lessons/`

- **L-006** (greenfield-delete-not-adapt): same rule.
- **L-008** (schema migrations mandatory): any v2 column change uses a numbered migration, never `EnsureSchema` boot reconciliation.
- **L-007** (E2E follows runtime contract): if v2 changes the Backend or Host API, ship E2E + matchers in the same change.

No memory-specific lesson exists yet (e.g., no "L-NNN consolidation gate inversion" or "L-NNN hooks dispatch at memory write").

### `_synthesis.md` cross-source posture

The synthesis emphasizes the **two-touch rule** (CLAUDE.md): if the same package has been patched twice in the same workstream, the third change is a structural redesign. Memory has had at least two distinct rounds of work (initial Task 06/07 RFC + the `add_memory_operation_scope` migration). v2 IS the structural redesign.

---

## 16. Honest Assessment

### 5 biggest current strengths

1. **Closed type taxonomy with default-scope routing** is small, predictable, and auditable. `Type.Validate` + `DefaultScopeForType` mean `agh memory write --type project ...` always lands in workspace scope without a flag.
2. **Catalog-as-derived-state architecture** with file-Markdown source-of-truth and SQLite/FTS5 as pure index. Drift recovery via `Reindex` is bullet-proof; `HealthStats.OrphanedFiles` exposes drift quantitatively. Mutations gracefully degrade — `TestStoreMutationsStaySuccessfulWhenDerivedSyncFails` confirms file write succeeds even if the catalog upsert fails.
3. **End-to-end observability** — every mutation lands in `memory_operation_log` with redacted summaries (`diagnostics.RedactAndBound`, max 2048 bytes), and that log is unioned into the canonical `event_summaries` view (`globaldb/global_db_observe.go:97`). Agents and operators both see the same audit trail.
4. **Consolidation lock semantics are forensic-grade** — hardlink-based atomic acquisition, PID liveness probe, stale reclaim by age, mtime-as-`last_consolidated_at`, full rollback on any mid-acquire failure, 17 tests including a 16-goroutine race. This is the most-tested area in the package.
5. **Full agent-manageability via CLI/HTTP/UDS parity for operator-grade verbs**. `agh memory health` is structured; `history` is filterable by scope/operation/since/limit; `search` is FTS5-backed with snippets; `consolidate` exposes the trigger reason. SD-011 is honored for the verbs that exist.

### 10 biggest current gaps vs documented patterns

1. **No `agent` scope.** Glossary, RFC 001, RFC 002, and `internal/CLAUDE.md` Memory & Skills Runtime all promise `agent | workspace | global`. Code has only `global | workspace`. `.agents/<name>/memory/` does not exist on disk. Per-agent isolation is impossible today.
2. **`AgentDef.memory.scope` is missing** — there is no per-agent memory configuration on `internal/config/agent.go`. The "default write scope declared per agent" is not enforceable.
3. **Compaction is plumbing without engine.** `runContextCompaction` + `HookContextPreCompact/PostCompact` are typed and dispatched but never invoked from production code. No compactor writes summaries into memory. Long-running sessions have no AGH-level transcript management.
4. **Native tools cannot mutate.** `agh__memory_write/delete/reindex/consolidate` do not exist (`internal/daemon/native_tools.go:606-627`). Agents who want to write durable memory inside a turn must call the Host API (only available to extensions, not direct ACP agents) or shell out to `agh memory write`.
5. **`memory.backend` extension capability is declarative only.** No daemon path delegates `Backend` calls to extensions. The hook plumbing (`ExtensionServiceMethodMemoryStore/Recall/Forget`) is allocated but not consumed. SD-009 directly applies: "data shape exists, consumer missing".
6. **Recall is purely lexical.** No embedding-based ranking, no semantic recall, no LLM rerank. Recall composition is byte-budget bounded (1500 chars / 3 results), not token-budget bounded. The `scoreMemoryRecall` in the Host API (`extension/host_api.go:2318`) is a *third* scoring algorithm — neither bm25 nor the fallback — duplicating effort.
7. **Per-agent provenance is metadata-only.** `Header.AgentName` exists in YAML but no enforcement that an agent only writes to its own scope, no read-time filter "show only memories I authored", no "ownership" constraint.
8. **Workspace identity is by path, not by stable ID.** `workspace_root` is a canonicalized filepath; `workspace_id` exists in the schema (default `''`) but is never populated by `internal/memory`. Move a workspace and the catalog rows are orphans. `dream.prepareWorkspace` does fetch `resolved.ID` (`dream.go:264-287`), but writes only set `WorkspaceRoot`.
9. **No skill / capability integration.** Skills are procedural memory in every competitor (Claude Code's `.skills`, Codex's recipes, Hermes's procedures). AGH skills live in a separate package with no shared types, no shared discovery, no shared prompt assembly. Agents must learn two completely different mental models.
10. **No turn-level / session-level / cross-session abstraction.** There is no concept of "working memory" (current turn buffer, lost on exit), "session memory" (this session only), "persistent memory" (the durable taxonomy). Everything is either ephemeral session state in `events.db` or durable Markdown files. The middle tier — explicitly abstracted — is missing.

### Architectural tensions / two-touch candidates

- **`store.go` mutex is per-Store-instance with shared mutex pointer through `ForWorkspace`** (`store.go:84-88`: `clone := *s` shallow-copies the `*sync.Mutex` field). This works because the field is a pointer, but the design depends on a non-obvious sharing invariant. Two different `NewStore` calls against the same global dir would NOT share locks. Two-touch flag: any "now we want stricter cross-process locking" or "now we want catalog-level transactions wrapping the file write" would pull this thread.
- **Search has three implementations**: `catalog.search` (FTS5), `fallbackSearchDocuments` (in-process scoring), `extension/host_api.go scoreMemoryRecall` (yet another scorer). Each evolves independently; the Host API explicitly does NOT use the FTS5 catalog. This is the third change to recall scoring across the codebase. v2 should unify on a single algorithm.
- **Index synthesis path** (`store.go:716-741`) is the second compatibility path — it handles "MEMORY.md missing but other docs present" by falling back to a full rescan. This was added to handle a real bug. Any further drift case ("MEMORY.md present but stale" — already handled) would be the third touch and signal that the index-sync model needs a rethink (e.g., make the catalog the source of truth and treat MEMORY.md as a derived view).
- **Consolidation gate vs documented defaults**: code has `min_sessions=3` (`dream.go:20`), `internal/CLAUDE.md` says **5**. This is documentation drift, not a code bug, but it suggests the gate parameters are not load-bearing in design — anyone can tweak them. v2 should pin the documented contract.
- **Dream agent lifetime**: `NewSessionSpawner` creates a session, prompts it, blocks until events drain, then `Stop`s it (`consolidation/runtime.go:410-464`). The dream session's events are not specially marked beyond `SessionTypeDream`; if the consolidator hangs, the dream lock holds for the configured stale-age (1 hour). No specific deadline pressure on the consolidator.

### Things that scream for structural redesign

- **Three scopes vs two**: requires schema column (`scope` CHECK constraint), agent-level config (`AgentDef.Memory.Scope`), prompt-assembly path that knows "this turn's agent's home memory dir", and a CLI/HTTP/UDS verb that lets an agent address its own scope explicitly. This cannot be patched in — it is a structural pivot.
- **Procedural memory unification**: skills are procedural memory; treating them as a separate registry forks the discovery model, the precedence model, and the prompt-assembly model. v2 should consider whether skills are "procedural memory of type=procedure" inside a unified memory subsystem, OR whether durable memory and skills become two facets of one "knowledge" abstraction.
- **Compaction-as-memory-write**: `HookContextPreCompact` payload already carries `Summary` + `ContextBlocks`. v2 could specify "PostCompact summary becomes a durable `feedback` or `project` memory entry, scoped to the session's workspace". This requires a compactor implementation **plus** a memory-writer hook executor. The two-touch rule applies: instead of patching a compactor and a memory-writer separately, v2 should architect the cycle (transcript → summary → memory → next-session prompt index) explicitly.
- **Backend extension delegation**: `memory.backend` capability + `ExtensionServiceMethodMemoryStore/Recall/Forget` exist but no daemon path uses them. v2 should either delete the unused capability (greenfield, no legacy) OR build the consumer (forwarding `Store.Search` to a registered extension when the agent or operator opts in). Leaving it half-built is the worst of both worlds.
- **Working-memory abstraction**: today there is no representation of "what the current turn knows that did not exist before this turn started". No "scratch pad", no "ephemeral cache", no per-turn key-value store. Competitors (Codex, Hermes, Claude Code) all have at least a turn-scoped working set. v2 should decide whether AGH needs one or whether the ACP agent's own context buffer is the answer — either way, the answer should be authored in the spec, not inferred by absence.

---

## Appendix A — File path quick-reference

- Memory package: `/Users/pedronauck/Dev/compozy/agh/internal/memory/`
- Consolidation runtime: `/Users/pedronauck/Dev/compozy/agh/internal/memory/consolidation/`
- Composition root wiring: `/Users/pedronauck/Dev/compozy/agh/internal/daemon/boot.go:280-340, 468-482, 816-834, 1178`
- Native tool bindings: `/Users/pedronauck/Dev/compozy/agh/internal/daemon/native_tools.go:606-627, 1411-1551`
- Prompt input composite: `/Users/pedronauck/Dev/compozy/agh/internal/daemon/prompt_input_composite.go:56-82`
- Hooks bridge (dream session_post_stop): `/Users/pedronauck/Dev/compozy/agh/internal/daemon/hooks_bridge.go:1237-1305`
- Session compaction plumbing: `/Users/pedronauck/Dev/compozy/agh/internal/session/manager_hooks.go:492-565`, `/Users/pedronauck/Dev/compozy/agh/internal/session/hooks.go:84-93`
- HTTP routes: `/Users/pedronauck/Dev/compozy/agh/internal/api/httpapi/routes.go:240-251`
- UDS routes: `/Users/pedronauck/Dev/compozy/agh/internal/api/udsapi/routes.go:270-282`
- Core handlers: `/Users/pedronauck/Dev/compozy/agh/internal/api/core/memory.go`
- CLI commands: `/Users/pedronauck/Dev/compozy/agh/internal/cli/memory.go:91-511`
- Host API memory methods: `/Users/pedronauck/Dev/compozy/agh/internal/extension/host_api.go:524-526, 1109-1217, 1861-1950, 2198-2360`
- Extension protocol: `/Users/pedronauck/Dev/compozy/agh/internal/extension/protocol/host_api.go:12-30, 45-47, 122-124, 185-198`
- Extension contract DTOs: `/Users/pedronauck/Dev/compozy/agh/internal/extension/contract/host_api.go:34-36, 177-200, 552-560, 651-665`
- Config struct: `/Users/pedronauck/Dev/compozy/agh/internal/config/config.go:161-175, 477-487, 1080-1083, 1276-1286`
- Tool surface keys: `/Users/pedronauck/Dev/compozy/agh/internal/config/tool_surface.go:94-99`
- Home paths: `/Users/pedronauck/Dev/compozy/agh/internal/config/home.go:16-17, 45, 121, 140`
- SQLite schema (memory_operation_log on global): `/Users/pedronauck/Dev/compozy/agh/internal/store/globaldb/global_db.go:110-118, 546-551, 950-983`
- Catalog schema + migrations: `/Users/pedronauck/Dev/compozy/agh/internal/memory/catalog.go:34-101, 173-206`
- Glossary memory section: `/Users/pedronauck/Dev/compozy/agh/docs/_memory/glossary.md:127-160`
- Standing directives: `/Users/pedronauck/Dev/compozy/agh/docs/_memory/standing_directives.md` (SD-002, SD-008, SD-009, SD-011)
- Lessons (none memory-specific yet): `/Users/pedronauck/Dev/compozy/agh/docs/_memory/lessons/README.md`

## Appendix B — What v2 must NOT lose

- Lock-with-rollback semantics (forensically tested).
- Mutation observability through the canonical event spine.
- Markdown-as-source-of-truth + catalog-as-derived (drift-recoverable).
- Closed taxonomy with explicit `Validate` (no string typos for `Type`).
- CLI/HTTP/UDS verb parity (SD-011).
- The four-phase consolidation prompt as a starting point (replace if needed, but the four-phase structure is good).
- The future-seam guard test (`interfaces_test.go`) — preserve as a forcing function against the same temptation in v2.
