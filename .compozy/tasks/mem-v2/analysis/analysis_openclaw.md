# OpenClaw Memory System — Forensic Analysis for AGH mem-v2

> Source-of-truth: `~/dev/knowledge/.resources/openclaw/` (TypeScript/Node.js, pnpm monorepo). Wiki at `~/dev/knowledge/openclaw/wiki/` cross-checked.
> No source under `~/dev/compozy/agh/.resources/openclaw/` — only the actual repo and the topic dir matter.
> All paths cited are absolute under `~/dev/knowledge/.resources/openclaw/` unless otherwise noted.

## TL;DR (≤200 words)

OpenClaw treats memory as a **first-class plugin domain** with a unified `MemoryPluginCapability` (`promptBuilder`, `flushPlanResolver`, `runtime`, `publicArtifacts`). The bundled `memory-core` extension is the reference implementation; alternatives (`memory-lancedb`, `memory-wiki`, `active-memory`) plug into the same surface. Persistence has **two distinct stores**: (1) **conversation transcripts** as `.jsonl` files under `~/.openclaw/agents/<agentId>/sessions/` plus a `sessions.json` metadata index protected by a queue-serialized write-lock, and (2) **memory corpus** as **plain Markdown** (`MEMORY.md`, `memory/YYYY-MM-DD.md`, `DREAMS.md`) indexed into a per-agent **SQLite** DB (`~/.openclaw/memory/<agentId>.sqlite`) with FTS5 + sqlite-vec hybrid search. Auto-memory is driven by **two fences**: a *pre-compaction memory flush* (silent agent turn that writes durable facts to `memory/YYYY-MM-DD.md`) and a *short-term recall promotion* loop driven by a three-phase **dreaming** cron (light → REM → deep) that scores recalls and promotes only items passing `minScore`/`minRecallCount`/`minUniqueQueries` gates into `MEMORY.md`. Compaction itself is a separate `compactEmbeddedPiSession` call with locking, summary-and-truncate semantics, and a memory index resync afterwards. The architecture is **filesystem-first, agent-inspectable, and runtime-pluggable** — its biggest novelty is the *dreaming* consolidation loop and the *short-term promotion* analytics store.

---

## 1. Codebase Topology

OpenClaw memory code is split across three layers:

### 1.1 Host SDK (consumed by every memory plugin)

`src/memory-host-sdk/` — the public host contract that bundled and external memory plugins build against.

- `runtime.ts` (`src/memory-host-sdk/runtime.ts`) — barrel: `runtime-core.js` + `runtime-cli.js` + `runtime-files.js`.
- `runtime-core.ts:1-32` — exports `loadConfig`, `resolveStateDir`, `resolveSessionTranscriptsDirForAgent`, `resolveDefaultAgentId`, `parseAgentSessionKey`, `buildActiveMemoryPromptSection`, `listActiveMemoryPublicArtifacts`, `resolveMemorySearchConfig`, the SILENT_REPLY_TOKEN, and the **MemoryFlushPlan**, **MemoryPluginCapability**, **MemoryPluginRuntime** types from `plugins/memory-state.ts`.
- `runtime-files.ts:1-11` — `listMemoryFiles`, `normalizeExtraMemoryPaths`, `readAgentMemoryFile`, `resolveMemoryBackendConfig`, plus `MemorySearchManager` interface.
- `engine.ts` — barrel for the engine surface (`engine-foundation`, `engine-storage`, `engine-embeddings`, `engine-qmd`).
- `engine-foundation.ts:1-48` — re-exports `parseDurationMs`, `loadConfig`, `resolveStateDir`, `resolveSessionTranscriptsDirForAgent`, etc.
- `engine-storage.ts:1-38` — re-exports `buildFileEntry`, `chunkMarkdown`, `cosineSimilarity`, `ensureDir`, `hashText`, `listMemoryFiles`, `parseEmbedding`, `remapChunkLines`, `ensureMemoryIndexSchema`, `loadSqliteVecExtension`, `requireNodeSqlite`, plus `MemoryChunk` / `MemoryFileEntry` types.
- `engine-embeddings.ts:1-64` — concrete embedding adapters: OpenAI, Gemini, Voyage, Mistral, Ollama, LM Studio, plus a local GGUF provider; batch endpoints for OpenAI/Gemini/Voyage; multimodal classification helpers.
- `engine-qmd.ts:1-22` — `extractKeywords`, `buildSessionEntry`, `listSessionFilesForAgent`, `parseQmdQueryJson`, scope helpers, `runCliCommand`.
- `host/types.ts:68-94` — defines the canonical `MemorySearchManager` interface (search/readFile/status/sync/probeEmbedding/probeVector/close).
- `host/memory-schema.ts:1-103` — **canonical SQLite schema** for the index (see §3.2).
- `host/internal.ts:18-505` — the chunking + indexing engine (file walking, multimodal handling, markdown chunking, cosine similarity).
- `host/sqlite.ts:1-21` — uses **`node:sqlite`** (the experimental built-in module).
- `host/backend-config.ts` — resolves `builtin` vs `qmd` backend with all QMD knobs.
- `host/session-files.ts` — turns JSONL transcripts into `SessionFileEntry` records for indexing into the memory store.
- `dreaming.ts:1-630` — defaults, types and resolvers for the dreaming cycle (light/deep/REM).

### 1.2 Plugin contract layer

`src/plugins/memory-state.ts:1-328` is the in-process plugin registry. Two key surfaces:

- `MemoryPluginCapability` (lines 127-132): `promptBuilder?`, `flushPlanResolver?`, `runtime?`, `publicArtifacts?`.
- Legacy single-purpose registrations (`registerMemoryPromptSection`, `registerMemoryRuntime`, `registerMemoryFlushPlanResolver`) — explicitly marked `LEGACY(memory-v1)` with todo comments to remove.
- `registerMemoryCorpusSupplement(pluginId, supplement)` for *additional* corpora beyond memory (lines 159-168). The `memory-wiki` plugin registers via this surface.
- `buildMemoryPromptSection({availableTools, citationsMode})` aggregates primary capability output + sorted supplements into the system prompt (lines 206-219).
- `resolveMemoryFlushPlan({cfg, nowMs})` (lines 234-243) is what auto-reply asks for the flush gate parameters.
- The state object is a single in-process singleton (`memoryPluginState`) — only one capability registration at a time, plus N supplements.

### 1.3 Reference implementation: `memory-core` extension

`extensions/memory-core/`:

- `openclaw.plugin.json:1-139` — declares `kind: "memory"`, exposes the `dreaming` slash command, and provides a JSON Schema for `dreaming.{enabled,frequency,timezone,verboseLogging,storage,phases.{light,deep,rem}}`.
- `index.ts:1-75` — registers everything in one call:
  - `registerBuiltInMemoryEmbeddingProviders(api)` (OpenAI, Gemini, Voyage, Mistral, Ollama, local GGUF)
  - `registerShortTermPromotionDreaming(api)` (auto-managed cron)
  - `registerDreamingCommand(api)` (CLI)
  - `api.registerMemoryCapability({ promptBuilder, flushPlanResolver, runtime, publicArtifacts })`
  - Two tools: `memory_search` and `memory_get` (lines 42-58).
  - One CLI namespace: `memory` with subcommands.
- `src/memory/manager*.ts` (~70 files) — the indexer/search manager: `manager.ts`, `search-manager.ts`, `qmd-manager.ts`, `manager-fts-state.ts`, `manager-vector-write.ts`, `manager-embedding-cache.ts`, `manager-cache.ts`, `manager-sync-control.ts`, `manager-sync-ops.ts`, `manager-targeted-sync.ts`, `manager-session-reindex.ts`, `mmr.ts`, `temporal-decay.ts`, `hybrid.ts`. This is the heaviest chunk of memory code.
- `src/short-term-promotion.ts` — the core of *dreaming-driven recall promotion*. State on disk in `memory/.dreams/short-term-recall.json`, `phase-signals.json`, `short-term-promotion.lock`.
- `src/dreaming.ts`, `src/dreaming-phases.ts`, `src/dreaming-narrative.ts`, `src/dreaming-markdown.ts`, `src/dreaming-command.ts`, `src/dreaming-repair.ts`, `src/dreaming-shared.ts` — the cron-driven sweep, three phases, narrative subagent, and DREAMS.md formatting.
- `src/flush-plan.ts:1-139` — `buildMemoryFlushPlan(cfg, nowMs)` builds the system+user prompt that runs as the silent compaction-flush turn, plus the relative path target (`memory/YYYY-MM-DD.md`).
- `src/prompt-section.ts:1-38` — the system-prompt block injected when `memory_search` / `memory_get` are available.
- `src/tools.ts:1-401` — the `memory_search` and `memory_get` tool implementations, citation decoration, and short-term recall tracking on every search hit.
- `src/runtime-provider.ts:1-19` — the bare `MemoryPluginRuntime` impl that wires `getMemorySearchManager` and `closeAllMemorySearchManagers` into the host.
- `src/concept-vocabulary.ts` — concept-tag vocabulary used by promotion ranking.

### 1.4 Other memory plugins (interface compatibility test)

- `extensions/active-memory/` (`index.ts:1-150+`) — pre-turn recall sub-agent. Reads recent conversation, runs a *recall* turn against `memory_search`, injects results into the next prompt as a "🧩 active memory:" line. Demonstrates that the plugin contract is genuinely composable — active-memory does not own storage, it composes on top.
- `extensions/memory-lancedb/` — alternative vector backend swapping SQLite-vec for LanceDB. Uses the same capability registration path.
- `extensions/memory-wiki/` — registers a *corpus supplement* (NOT a runtime), exposing `wiki_search`, `wiki_get`, `wiki_apply`, `wiki_lint`. Adds provenance-tracked claims/evidence on top of normal recall. This is the cleanest example of `registerMemoryCorpusSupplement(pluginId, supplement)`.

### 1.5 Auto-reply / flush-gate layer

`src/auto-reply/reply/`:

- `memory-flush.ts:1-132` — the gate logic: `shouldRunMemoryFlush`, `shouldRunPreflightCompaction`, `hasAlreadyFlushedForCurrentCompaction`, `computeContextHash` (SHA-256 truncated to 16 hex chars over last 3 user/assistant messages, used for state-based dedup).
- `agent-runner-memory.ts:1-849` — the integration point. Two top-level functions:
  - `runPreflightCompactionIfNeeded(...)` — runs `compactEmbeddedPiSession({trigger: "budget"})` *before* the next agent turn when projected token count exceeds threshold and totals are stale.
  - `runMemoryFlushIfNeeded(...)` — runs a silent **`runEmbeddedPiAgent({trigger: "memory", memoryFlushWritePath, prompt, extraSystemPrompt})`** turn and tracks `memoryFlushAt` / `memoryFlushCompactionCount` on the session entry to dedup.
  - Logic respects sandbox mode (lines 526-539), heartbeat sessions, CLI providers (skips both), and computes projected tokens by reading the **transcript tail** when `entry.totalTokensFresh === false`.

### 1.6 Sessions layer

`src/sessions/` (small, surface-only) and `src/config/sessions/` (the meaty store).

- `src/config/sessions/types.ts:111-259` — the **`SessionEntry`** record schema (50+ fields, see §4).
- `src/config/sessions/store.ts:1-838` — `loadSessionStore`, `saveSessionStore`, `updateSessionStore`, `updateSessionStoreEntry`, `recordSessionMetaFromInbound`, `updateLastRoute`, with **per-store-path FIFO lock queue** (`getOrCreateLockQueue` lines 576-647) and atomic write via `writeTextAtomic`.
- `src/config/sessions/transcript.ts:1-100` — `resolveSessionTranscriptFile`, `appendAssistantMessageToSessionTranscript` (depends on `pi-coding-agent`'s `SessionManager.appendMessage`), session header writer (`{type:"session", version:CURRENT_SESSION_VERSION, id, timestamp, cwd}` as line 1 of each `.jsonl`).
- `src/config/sessions/paths.ts:1-330` — path resolution, validation (`SAFE_SESSION_ID_RE = /^[a-z0-9][a-z0-9._-]{0,127}$/i`), absolute→relative migration for legacy entries, `~`/`$HOME` expansion.
- `src/config/sessions/store-maintenance.ts` (referenced) — `pruneStaleEntries`, `capEntryCount`, `rotateSessionFile`, `getActiveSessionMaintenanceWarning`.
- `src/config/sessions/disk-budget.ts` (referenced) — disk budget enforcement (configurable max bytes for transcripts).
- `src/config/sessions/store-cache.ts` — read-through object cache + serialized cache to skip JSON parse on repeated loads.

### 1.7 Hooks (built-in user-facing memory hooks)

`src/hooks/bundled/session-memory/`:

- `HOOK.md:1-109` — declares `events: ["command:new", "command:reset"]` and the message-count config knob.
- `handler.ts:1-225` — on `/new` or `/reset`:
  1. Resolve previous session file (search for `.reset.` rotated transcripts if the current is empty).
  2. Read last N user/assistant messages (default 15) via `getRecentSessionContentWithResetFallback`.
  3. Generate a **descriptive slug via LLM** (`generateSlugViaLLM({sessionContent, cfg})`, fallback to `HHMM` timestamp).
  4. Write `<workspace>/memory/YYYY-MM-DD-<slug>.md` with header `# Session: ...`, session key, ID, source, then `## Conversation Summary` + content. Uses `writeFileWithinRoot` (alias-safe).

So `/new` and `/reset` are also **memory snapshot triggers** in addition to compaction-flush.

---

## 2. Memory Taxonomy

OpenClaw distinguishes (using the source's own naming):

| Layer | Lifetime | Where stored | Visibility | Promotion |
|---|---|---|---|---|
| **Conversation transcript** | Per session | `~/.openclaw/agents/<agentId>/sessions/<sid>.jsonl` | Read by GW for replay | Compacted in place |
| **Tool buffer** | Per turn | In-memory queue, flushed to JSONL | Internal | Flushed via `flushPendingToolResultsAfterIdle` before compaction |
| **Daily memory** | "Short-term" | `<workspaceDir>/memory/YYYY-MM-DD.md` | Auto-loaded today + yesterday | Auto-flush writes here |
| **Long-term memory** | Durable | `<workspaceDir>/MEMORY.md` (and `memory.md` legacy) | Always loaded | Only deep-phase dreaming writes here |
| **Dream Diary** | Reviewable | `<workspaceDir>/DREAMS.md` (or `dreams.md`) | Human review surface | Light/REM/deep narrative subagent appends |
| **Phase reports** | Optional | `<workspaceDir>/memory/dreaming/{light,deep,rem}/YYYY-MM-DD.md` | Per-phase audit | Written when `storage.mode = separate|both` |
| **Short-term recall analytics** | Internal | `<workspaceDir>/memory/.dreams/short-term-recall.json` | Internal | Updated on every `memory_search` hit, locked via `short-term-promotion.lock` |
| **Phase signals** | Internal | `<workspaceDir>/memory/.dreams/phase-signals.json` | Internal | Updated by light/REM phases |
| **Daily ingestion checkpoints** | Internal | `<workspaceDir>/memory/.dreams/daily-ingestion.json` | Internal | Tracks which daily-memory chunks already ingested |
| **Session ingestion checkpoints** | Internal | `<workspaceDir>/memory/.dreams/session-ingestion.json` | Internal | Tracks which transcript messages already ingested |
| **Session corpus** | Internal | `<workspaceDir>/memory/.dreams/session-corpus/YYYY-MM-DD.{md,txt}` | Internal | Redacted JSONL slices for dreaming use |
| **SQLite index** | Derived | `~/.openclaw/memory/<agentId>.sqlite` | Internal | Rebuilt on file change / config change |
| **Embedding cache** | Derived | Same SQLite DB, separate table | Internal | Refreshed by `embeddings_*` provider flow |

The documentation summarizes this as a three-file user-facing surface (`MEMORY.md`, `memory/YYYY-MM-DD.md`, `DREAMS.md`) but the actual code uses ~10 distinct on-disk files for state and analytics.

---

## 3. Persistence Backends

### 3.1 Sessions (transcripts)

**Tech**: plain JSONL files. **No DB.**

Layout (canonical, derived from `src/config/sessions/paths.ts:9-17`):

```
<stateDir>/agents/<agentId>/sessions/
├── sessions.json                        ← per-agent metadata index
├── <sessionId>.jsonl                    ← transcript
├── <sessionId>-topic-<encodedTopic>.jsonl  ← thread/topic transcript variant
└── <sessionId>.reset.<timestamp>.jsonl  ← rotated reset archive
```

`stateDir` defaults to `~/.openclaw` (`resolveStateDir(env, homedir)` in `src/config/paths.ts`).

**`sessions.json`** is a single JSON object: `Record<sessionKey, SessionEntry>` (see §4 for schema).

Atomic writes: `writeTextAtomic(storePath, serialized, {mode: 0o600})` (`src/config/sessions/store.ts:537`). On Windows, retry up to 5 times with exponential backoff (lines 404-424). Mutex via `withSessionStoreLock(storePath, fn, opts)` — a per-path FIFO lock queue, draining microtask-by-microtask, with `timeoutMs` (default 10s), `staleMs` (30s), and a derived `maxHoldMs` (≥5s).

Session ID validation: `SAFE_SESSION_ID_RE = /^[a-z0-9][a-z0-9._-]{0,127}$/i` (`paths.ts:61`). Containment-check ensures relative paths resolve inside `sessionsDir`; older absolute-path entries are normalized to relative.

JSONL transcript format: line 1 is a **session header** (`{type:"session", version:CURRENT_SESSION_VERSION, id, timestamp, cwd}`, `transcript.ts:25-36`), subsequent lines are one message each. Append-only writes via the upstream `SessionManager.appendMessage` from `@mariozechner/pi-coding-agent`.

### 3.2 Memory index (per-agent SQLite)

**Tech**: built-in **`node:sqlite`** module (`src/memory-host-sdk/host/sqlite.ts:8`). This is the experimental Node ≥20 builtin — explicit error message tells users to upgrade if it's missing.

**Path**: `~/.openclaw/memory/<agentId>.sqlite` (`src/agents/memory-search.ts:135-143`). Path supports `{agentId}` substitution.

**Schema** (`src/memory-host-sdk/host/memory-schema.ts:11-89`):

```sql
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
  path TEXT PRIMARY KEY,
  source TEXT NOT NULL DEFAULT 'memory',     -- 'memory' | 'sessions'
  hash TEXT NOT NULL,
  mtime INTEGER NOT NULL,
  size INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chunks (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'memory',
  start_line INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  hash TEXT NOT NULL,
  model TEXT NOT NULL,                        -- embedding model used
  text TEXT NOT NULL,
  embedding TEXT NOT NULL,                    -- JSON array string, parsed lazily
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chunks_path ON chunks(path);
CREATE INDEX IF NOT EXISTS idx_chunks_source ON chunks(source);

-- Optional embedding cache (cross-file reuse keyed on provider+model+key+hash)
CREATE TABLE IF NOT EXISTS <embeddingCacheTable> (
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  provider_key TEXT NOT NULL,
  hash TEXT NOT NULL,
  embedding TEXT NOT NULL,
  dims INTEGER,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (provider, model, provider_key, hash)
);
CREATE INDEX IF NOT EXISTS idx_embedding_cache_updated_at ON <embeddingCacheTable>(updated_at);

-- Optional FTS5 virtual table (tokenizer = unicode61 default, or trigram for CJK)
CREATE VIRTUAL TABLE IF NOT EXISTS <ftsTable> USING fts5(
  text,
  id UNINDEXED,
  path UNINDEXED,
  source UNINDEXED,
  model UNINDEXED,
  start_line UNINDEXED,
  end_line UNINDEXED
  [, tokenize='trigram case_sensitive 0']
);
```

`ensureColumn` (lines 92-103) does **runtime ALTER TABLE ADD COLUMN** for `files.source` and `chunks.source` — this is OpenClaw's only "migration" mechanism, basically the same `EnsureSchema`-style boot reconciliation that AGH's `agh-schema-migration` skill *forbids*. Worth flagging.

`sqlite-vec` extension is loaded optionally by `loadSqliteVecExtension` (re-exported via `engine-storage.ts:35`); without it, vector similarity is computed in-process via `cosineSimilarity` over JSON-decoded embedding strings.

### 3.3 Short-term promotion analytics

`<workspaceDir>/memory/.dreams/short-term-recall.json` — a JSON file (no SQLite). Schema (`extensions/memory-core/src/short-term-promotion.ts:61-86`):

```ts
type ShortTermRecallStore = {
  version: 1;
  updatedAt: string;          // ISO8601
  entries: Record<string, ShortTermRecallEntry>;
};

type ShortTermRecallEntry = {
  key: string;                // "<source>:<path>:<startLine>:<endLine>[:<claimHash>]"
  path: string;
  startLine: number;
  endLine: number;
  source: "memory";
  snippet: string;
  recallCount: number;        // organic memory_search hits
  dailyCount: number;         // light-phase ingestion hits
  groundedCount: number;      // grounded backfill hits
  totalScore: number;         // cumulative search score
  maxScore: number;
  firstRecalledAt: string;
  lastRecalledAt: string;
  queryHashes: string[];      // up to 32 sha1-12 hashes
  recallDays: string[];       // up to 16 ISO days
  conceptTags: string[];      // up to MAX_CONCEPT_TAGS
  claimHash?: string;
  promotedAt?: string;
};
```

Locked via filesystem lock at `memory/.dreams/short-term-promotion.lock` (timeout 10s, stale after 60s, retry 40ms). In-process locking layered via `inProcessShortTermLocks: Map<string, Promise<void>>`. The lock file is *also* an audit artifact — `repairShortTermPromotionArtifacts` will remove stale locks.

### 3.4 Honcho (alternative external service)

The wiki references this as a third backend — multi-document memory service with vector embeddings hosted at `api.honcho.dev` or self-hosted. No source under `extensions/memory-honcho/` was found; the doc lists it as an install-on-demand plugin. The plugin slot mechanism (`cfg.plugins.slots.memory`) is the routing primitive (`src/memory-host-sdk/dreaming.ts:321-332`).

### 3.5 QMD (Quantum Memory Daemon, OpenClaw’s sibling local-first sidecar)

QMD is a separate **CLI subprocess**: spawned on demand (or via mcporter MCP server with `lifecycle: keep-alive` for amortized cost). Configured via `memory.qmd.{command,mcporter,searchMode,searchTool,paths,sessions,update,limits,scope}`. Resolved into `ResolvedQmdConfig` (`src/memory-host-sdk/host/backend-config.ts:66-77`).

QMD-specific defaults:

```ts
DEFAULT_QMD_INTERVAL = "5m";
DEFAULT_QMD_DEBOUNCE_MS = 15_000;
DEFAULT_QMD_TIMEOUT_MS = 4_000;
DEFAULT_QMD_SEARCH_MODE = "search";        // "search" | "vsearch" | "query"
DEFAULT_QMD_EMBED_INTERVAL = "60m";
DEFAULT_QMD_COMMAND_TIMEOUT_MS = 30_000;
DEFAULT_QMD_UPDATE_TIMEOUT_MS = 120_000;
DEFAULT_QMD_EMBED_TIMEOUT_MS = 120_000;
DEFAULT_QMD_LIMITS = { maxResults: 6, maxSnippetChars: 700, maxInjectedChars: 4_000, timeoutMs: 4_000 };
```

QMD scope policy (`DEFAULT_QMD_SCOPE`, lines 103-115) is a `SessionSendPolicyConfig` that defaults to *deny except direct/channel chats* — i.e. group chats can't bleed into the QMD index unless explicitly opted in. This is a privacy/operator invariant, not a runtime knob.

---

## 4. SessionEntry — the canonical session record

`src/config/sessions/types.ts:111-259` defines a 65+-field `SessionEntry`. Key field groups, with the load-bearing ones for memory:

### Identity / persistence
- `sessionId: string`, `sessionFile?: string`
- `updatedAt: number`
- `chatType?`, `groupId?`, `groupChannel?`, `space?`, `origin?`, `deliveryContext?`
- `parentSessionKey?: string`, `spawnedBy?: string`, `spawnDepth?: number`, `subagentRole?: "orchestrator" | "leaf"`

### Compaction / token gating
- `inputTokens? / outputTokens? / totalTokens?: number`
- **`totalTokensFresh?: boolean`** — `false` means stale/unknown, force transcript-tail re-read. Critical to memory flush gating (`agent-runner-memory.ts:393`).
- `cacheRead?`, `cacheWrite?`, `estimatedCostUsd?`
- `compactionCount?: number` — incremented on every successful compaction.
- `compactionCheckpoints?: SessionCompactionCheckpoint[]` (lines 92-104) — each checkpoint pins `pre`/`post` transcript references, summary, tokensBefore/After, reason (`"manual" | "auto-threshold" | "overflow-retry" | "timeout-retry"`), and a `firstKeptEntryId` so resume-after-compaction works deterministically.
- **`memoryFlushAt?: number`**, **`memoryFlushCompactionCount?: number`**, **`memoryFlushContextHash?: string`** — these three fields are the *dedup gate* for auto-memory flush (lines 232-234).

### Reset / heartbeat
- `lastHeartbeatText?`, `lastHeartbeatSentAt?`, `heartbeatTaskState?: Record<string,number>`
- `heartbeatIsolatedBaseSessionKey?`
- `abortCutoffMessageSid?`, `abortCutoffTimestamp?`

### Skills / system prompt
- `skillsSnapshot?: SessionSkillSnapshot { prompt, skills, skillFilter?, resolvedSkills?, version? }`
- `systemPromptReport?: SessionSystemPromptReport` (lines 447-497) — full diagnostic record of last system prompt (chars, project context, injected workspace files, skills block sizes, tools list/schema sizes, sandbox state). Used by the verbose UI and trace.

### ACP / runtime
- `acp?: SessionAcpMeta` (lines 32-52) — backend, agent name, runtimeSessionName, identity (state/source), mode (`persistent`/`oneshot`), runtimeOptions (mode/model/cwd/permissionProfile/timeoutSeconds/backendExtras), state, lastActivityAt, lastError.
- `cliSessionBindings?: Record<string, CliSessionBinding>` — CLI session ID indexed by hash (`authProfileId | authEpoch | extraSystemPromptHash | mcpConfigHash`).

### Plugin trace
- `pluginDebugEntries?: SessionPluginDebugEntry[]` — lines per plugin, parsed into status vs trace channels via `resolveSessionPluginStatusLines` / `resolveSessionPluginTraceLines` (lines 266-298).

### Merge policy

`mergeSessionEntryWithPolicy(existing, patch, options)` (lines 372-394) supports `"touch-activity"` (default) and `"preserve-activity"` (used by inbound metadata that shouldn't bump idle-reset clocks). A subtle invariant: when `patch.model` is set without `patch.modelProvider`, the provider is *cleared* to avoid stale-provider carryover (lines 386-393). This is the kind of invariant that tends to leak silently in less careful code.

---

## 5. Auto-memory: the flush + compaction interplay

The pre-compaction memory flush is the most novel part. Sequence (`src/auto-reply/reply/agent-runner-memory.ts:505-848`):

### 5.1 Flush plan resolution
`resolveMemoryFlushPlan({cfg, nowMs})` calls the registered `flushPlanResolver` (provided by `memory-core.flush-plan.ts:95-139`). Returns:

```ts
type MemoryFlushPlan = {
  softThresholdTokens: number;            // default 4_000
  forceFlushTranscriptBytes: number;       // default 2 MiB
  reserveTokensFloor: number;              // from agents.defaults.compaction
  prompt: string;                          // user-prompt sent on the flush turn
  systemPrompt: string;                    // extra system prompt for the flush turn
  relativePath: string;                    // memory/YYYY-MM-DD.md (current local day)
};
```

The prompt is hardened with "safety hints":

> "Pre-compaction memory flush. Store durable memories only in memory/YYYY-MM-DD.md (create memory/ if needed). Treat workspace bootstrap/reference files such as MEMORY.md, DREAMS.md, SOUL.md, TOOLS.md, and AGENTS.md as read-only during this flush; never overwrite, replace, or edit them. If memory/YYYY-MM-DD.md already exists, APPEND new content only and do not overwrite existing entries. Do NOT create timestamped variant files (e.g., YYYY-MM-DD-HHMM.md); always use the canonical YYYY-MM-DD.md filename. If nothing to store, reply with __OPENCLAW_SILENT_REPLY_TOKEN__."

The plan is mandatory: if no plugin registers a resolver, *no flush runs* (`agent-runner-memory.ts:521-524`).

### 5.2 Gate function
`shouldRunMemoryFlush(...)` returns true iff:
1. `tokenCount >= contextWindow - reserveTokensFloor - softThresholdTokens`
2. `entry.memoryFlushCompactionCount !== entry.compactionCount` (no flush since the last compaction).

`hasAlreadyFlushedForCurrentCompaction` is the single-flight dedup: a flush has happened for the current compaction cycle if `memoryFlushCompactionCount === compactionCount`.

A second trigger is **transcript byte size**: if `forceFlushTranscriptBytes` (default 2 MiB) is hit, force-flush regardless of token math. This protects against runaway tool output that doesn't show up in usage.

### 5.3 Sandbox / heartbeat / CLI guards
Skip the flush when:
- The session is sandboxed and `sandboxConfig.workspaceAccess !== "rw"` (would have nowhere to write).
- `isHeartbeat` is true (heartbeat sessions are synthetic; flushing them wastes tokens).
- `isCliProvider(provider, cfg)` is true (CLI providers handle their own context).

### 5.4 The flush turn
The flush runs as a **silent embedded agent turn** (`memoryDeps.runEmbeddedPiAgent({trigger: "memory", memoryFlushWritePath, prompt: plan.prompt, extraSystemPrompt: plan.systemPrompt})`). The result:
- Writes to `memory/YYYY-MM-DD.md` (new file or append).
- If the agent decides nothing is worth saving, it replies with `__OPENCLAW_SILENT_REPLY_TOKEN__` (the `SILENT_REPLY_TOKEN` constant).
- The compaction counter is incremented when the flush itself triggers a compaction event (`evt.stream === "compaction" && evt.data.phase === "end"`, lines 770-772).

After completion, persist `memoryFlushAt` and `memoryFlushCompactionCount` via `updateSessionStoreEntry({storePath, sessionKey, update})`. This is the *source of truth* for "did we already flush for this turn?".

### 5.5 Preflight compaction
A separate gate, `runPreflightCompactionIfNeeded`, runs `compactEmbeddedPiSession({trigger: "budget", currentTokenCount})` *before* the next agent turn when:
- `entry.totalTokensFresh === false || !persistedTotalTokens`
- The transcript-derived projected token count exceeds the threshold.

Compaction increments `compactionCount`, updates `tokensAfter`, refreshes the queued followup session id (so subsequent turns route to the post-compaction session), and *appends a "post-compaction refresh prompt"* (`appendPostCompactionRefreshPrompt`) to the followup's `extraSystemPrompt` so the model doesn't lose its bearings.

---

## 6. Compaction lifecycle

Per the wiki + `src/agents/compaction.ts` + `src/agents/pi-embedded.ts` (referenced via `compactEmbeddedPiSession`):

1. **Overflow detection** — `evaluateContextWindowGuard` runs at the start of every embedded turn.
2. **Acquire write lock** — `acquireSessionWriteLock` (per session file).
3. **Flush pending tool results** — `flushPendingToolResultsAfterIdle` drains the in-memory tool-result buffer to the JSONL.
4. **Run `before_compaction` hooks** — `event: compaction.before` fires.
5. **Summarize** — `compactEmbeddedPiSessionDirect` calls the (potentially separate) compaction model. Model selection precedence: `agents.defaults.compaction.model` > session model. Provider change clears the cached `authProfileId`.
6. **Truncate** — `truncateSessionAfterCompaction` rewrites the JSONL: removes old messages, inserts a single summary block, updates session metadata.
7. **`SessionToolResultGuard`** validates result-injection pairing during the rewrite.
8. **Run `after_compaction` hooks** — `runPostCompactionSideEffects` refreshes the memory search index (`postCompactionForce: true` triggers full reindex).
9. **Release lock** — `emitSessionTranscriptUpdate` event fires so subscribers (Canvas UI, channel adapters) re-render.

Safeguards:
- `compactWithSafetyTimeout` (env: `EMBEDDED_COMPACTION_TIMEOUT_MS`).
- Hardcoded retry attempt cap.
- "Benign cancel reasons" (user cancel, insufficient tokens) surface as non-error states.
- Failed compaction *retains* the pre-compaction transcript; the next turn will trigger again.

Token thresholds (from wiki, validated by source defaults):
- **Hard minimum**: 8_192 tokens — below this, compaction is not even attempted.
- **Soft warning**: 32_768 — queues async compaction.

Compaction summary instructions (`src/agents/compaction.ts:24-37`):

```text
Merge these partial summaries into a single cohesive summary.

MUST PRESERVE:
- Active tasks and their current status (in-progress, blocked, pending)
- Batch operation progress (e.g., '5/17 items completed')
- The last thing the user requested and what was being done about it
- Decisions made and their rationale
- TODOs, open questions, and constraints
- Any commitments or follow-ups promised

PRIORITIZE recent context over older history. The agent needs to know
what it was doing, not just what was discussed.
```

Plus an identifier-preservation policy (`strict` default): "Preserve all opaque identifiers exactly as written (no shortening or reconstruction), including UUIDs, hashes, IDs, tokens, API keys, hostnames, IPs, ports, URLs, and file names."

This is **prompt engineering as load-bearing code** — the failure mode if these instructions degrade is that summaries lose IDs, batch counters, and the active task state, which silently breaks long-running agent workflows. AGH should treat these instructions as a versioned, testable artifact rather than as inline prose.

`splitMessagesByTokenShare` (lines 120-209) handles a particularly nasty edge: tool calls and their results must stay in the same chunk, so the splitter keeps a `pendingToolCallIds` set and only splits at message boundaries where all pending tool-call IDs have been resolved.

---

## 7. Read API: search and recall

### 7.1 The `MemorySearchManager` interface (`src/memory-host-sdk/host/types.ts:68-94`)

```ts
export interface MemorySearchManager {
  search(
    query: string,
    opts?: {
      maxResults?: number;
      minScore?: number;
      sessionKey?: string;
      qmdSearchModeOverride?: "query" | "search" | "vsearch";
      onDebug?: (debug: MemorySearchRuntimeDebug) => void;
    },
  ): Promise<MemorySearchResult[]>;
  readFile(params: { relPath: string; from?: number; lines?: number }): Promise<{ text: string; path: string }>;
  status(): MemoryProviderStatus;
  sync?(params?: { reason?, force?, sessionFiles?, progress? }): Promise<void>;
  probeEmbeddingAvailability(): Promise<MemoryEmbeddingProbeResult>;
  probeVectorAvailability(): Promise<boolean>;
  close?(): Promise<void>;
}
```

`MemorySearchResult`:
```ts
type MemorySearchResult = {
  path: string;
  startLine: number;
  endLine: number;
  score: number;
  snippet: string;
  source: "memory" | "sessions";
  citation?: string;
};
```

`MemoryProviderStatus` exposes deep diagnostics: `backend`, `provider`, `model`, file/chunk counts per source, FTS availability (with error if the SQLite build doesn't include it), vector availability + `extensionPath` + `loadError` + `dims`, batch state, fallback metadata. This is what `openclaw memory status` prints.

### 7.2 The agent-facing tools

**`memory_search`** (`extensions/memory-core/src/tools.ts:181-318`):
- Required `query`, optional `maxResults`, `minScore`, `corpus` (`memory` | `wiki` | `all`).
- For QMD backend, results are clamped by `maxInjectedChars` to keep the system prompt bounded.
- Each surfaced result triggers `recordShortTermRecalls(...)` (best-effort, never blocks).
- Citations are decorated according to `cfg.memory.citations` (`auto` | `on` | `off`). When `off`, the agent is instructed not to mention paths in replies unless asked.

**`memory_get`** (lines 320-401):
- Required `path`, optional `from`, `lines`, `corpus`.
- For `builtin` backend, reads via `readAgentMemoryFile` (filesystem-direct).
- For `qmd` backend, delegates to the `MemorySearchManager.readFile`.

### 7.3 System-prompt injection

`buildPromptSection({availableTools, citationsMode})` (`extensions/memory-core/src/prompt-section.ts:1-38`) injects a "## Memory Recall" block that *mandates* a `memory_search` step before answering recall questions:

> "Before answering anything about prior work, decisions, dates, people, preferences, or todos: run memory_search on MEMORY.md + memory/*.md + indexed session transcripts; then use memory_get to pull only the needed lines. If low confidence after search, say you checked."

This is a forcing function — the model is told to run search first, not the user. The block disappears if the tools aren't available.

### 7.4 Hybrid scoring config (`src/agents/memory-search.ts:97-112`)

Defaults:
- `chunking.tokens = 400`, `chunking.overlap = 80`.
- `query.maxResults = 6`, `query.minScore = 0.35`.
- `query.hybrid.enabled = true`, `vectorWeight = 0.7`, `textWeight = 0.3`, `candidateMultiplier = 4`.
- MMR (Maximal Marginal Relevance) is opt-in: `mmr.enabled = false`, `mmr.lambda = 0.7`.
- Temporal decay opt-in: `halfLifeDays = 30`.
- `cache.enabled = true`.

`sync.sessions.deltaBytes = 100_000`, `deltaMessages = 50` — the indexer will only re-ingest a session JSONL after these thresholds (avoids per-message reindex churn). `postCompactionForce = true` — full reindex after any compaction (because line numbers shift).

---

## 8. Write API: how things get into memory

### 8.1 User-facing writes (always Markdown)

The agent has **direct file access** via standard tools (e.g., `Edit`, `Write`). The `memory_*` tools are *read-only*. There is no `memory_write` JSON-RPC method. To remember a fact, either the user says "remember that …" and the agent writes the file, or the auto-flush turn fires and the agent decides what to save.

The flush prompt explicitly *forbids* the agent from editing `MEMORY.md`, `DREAMS.md`, `SOUL.md`, `TOOLS.md`, `AGENTS.md` during a flush — those are reserved for human/dreaming-only writes.

### 8.2 Indexer auto-sync

The `MemoryIndexManager` runs in-process inside the bundled `memory-core` plugin. Triggers:
- **File watcher** with `watchDebounceMs = 1500`.
- **Interval rebuild** if `intervalMinutes > 0` (default 0 = disabled).
- **On session start** if `onSessionStart = true` (default).
- **On every search** if `onSearch = true` (default) — does a fast delta scan.
- **Auto-reindex** when embedding provider/model/chunking change (deletes the SQLite and rebuilds).
- **Force reindex** post-compaction (`sync.sessions.postCompactionForce`).

### 8.3 Programmatic writes via tool-result tracking

`memory_search` records every surfaced result into `short-term-recall.json`. Each search bumps `recallCount`, refreshes `queryHashes`, `recallDays` (capped at MAX_RECALL_DAYS=16), and recomputes `conceptTags`. This is the *evidence base* for promotion.

### 8.4 The dreaming loop (auto-promotion)

Three phases, run in order by a single cron (default `0 3 * * *`):

| Phase | Reads | Writes | Promotes to MEMORY.md |
|---|---|---|---|
| **Light** | recent daily memory + recall traces + redacted session transcripts | `## Light Sleep` block in DREAMS.md, `phase-signals.json` | No |
| **Deep** | short-term recall store + phase signals | promoted entries to `MEMORY.md`, `## Deep Sleep` block in DREAMS.md | **Yes** |
| **REM** | recent short-term traces + memory + daily | `## REM Sleep` block in DREAMS.md, phase signals | No |

**Deep promotion** weights (`extensions/memory-core/src/short-term-promotion.ts:43-59`):

```ts
DEFAULT_PROMOTION_WEIGHTS = {
  frequency: 0.24,         // Σ recallCount + dailyCount + groundedCount
  relevance: 0.30,         // avgScore from search
  diversity: 0.15,         // unique queryHashes
  recency: 0.15,           // exp(-ageDays / halfLife), halfLife default 14d
  consolidation: 0.10,     // multi-day spread (log1p over recallDays)
  conceptual: 0.06,        // conceptTags.length / 6
};
```

Plus phase reinforcement boosts (capped):
- `PHASE_SIGNAL_LIGHT_BOOST_MAX = 0.06`
- `PHASE_SIGNAL_REM_BOOST_MAX = 0.09`
- `PHASE_SIGNAL_HALF_LIFE_DAYS = 14`

Promotion gates:
- `minScore` (default 0.8 in deep — configurable)
- `minRecallCount` (default 3)
- `minUniqueQueries` (default 3, doc says 2; check `DEFAULT_PROMOTION_MIN_UNIQUE_QUERIES = 2` in source)
- `maxAgeDays` (default 30) — older candidates are dropped

There is also a **recovery sub-loop** for promotion-store health (`MemoryDeepDreamingRecoveryConfig`):
- `triggerBelowHealth = 0.35` — if the recall store's "health" score drops below this, kick off recovery.
- Looks back `lookbackDays = 30`, takes up to `maxRecoveredCandidates = 20` items, and only promotes those above `minRecoveryConfidence = 0.9` (auto-write at `≥ 0.97`).

### 8.5 Concept vocabulary

`extensions/memory-core/src/concept-vocabulary.ts` (referenced) provides `deriveConceptTags({path, snippet})` — at most `MAX_CONCEPT_TAGS` per entry. This is what fills `entry.conceptTags` and feeds the `conceptual` ranking signal.

### 8.6 Narrative subagent ("dream diary entries")

After each phase has enough material, `memory-core` runs a *best-effort background subagent turn* (using the default runtime model) and appends a short narrative entry to `DREAMS.md`. The agent run is created with a special bootstrap context: `runId` starts with `dreaming-narrative-` and the bootstrap record's `customType` is `openclaw:bootstrap-context:full`. `host/session-files.ts:26-46` filters these out of normal session indexing (`generatedByDreamingNarrative` flag) so dream-narrative transcripts don't pollute regular search.

---

## 9. Lifecycle

### 9.1 Hydration on session start

When a message arrives that maps to a new session key (per `dmScope`):
1. `recordSessionMetaFromInbound` writes a new `SessionEntry` to `sessions.json` (under write lock).
2. The transcript header is written on first append (`ensureSessionHeader`) — `{type:"session", version, id, timestamp, cwd}`.
3. The Gateway assembles the system prompt: persona + memory prompt section (mandatory recall instruction) + skills + tool catalog. This last bundle is what `SessionSystemPromptReport` snapshots for diagnostics.
4. If `sync.onSessionStart = true`, the indexer is kicked.

Auto-loaded files (the wiki claims; confirmed by the `memory_search` corpus):
- `MEMORY.md` (always)
- `memory/today.md` and `memory/yesterday.md` (the doc says auto-loaded; the search index always covers all `memory/*.md`)

### 9.2 Persistence on shutdown

There is **no graceful shutdown flush** — every write is atomic. On crash:
- The JSONL transcript truncates at the last complete line (JSONL invariant).
- `sessions.json` is the result of the last successful `writeTextAtomic`.
- The SQLite index might be slightly stale; the next `sync()` call brings it current.

### 9.3 Migrations

- Sessions store: legacy uppercase keys are detected by `resolveSessionStoreEntry` and merged into the normalized lowercase key (`store.ts:83-122`). Legacy absolute paths are converted to relative on read (`paths.ts:185-229`).
- Memory schema: `ensureColumn` does runtime ALTER TABLE for the `source` column. **No numbered migration registry exists** — this is greenfield-style schema-on-boot reconciliation, the exact pattern AGH's standing directives reject.

### 9.4 GC / maintenance

`pruneStaleEntries(store, pruneAfterMs)` and `capEntryCount(store, maxEntries)` (`store-maintenance.ts`, referenced from `store.ts:169-173`). Sessions older than `pruneAfter` (default `30d`) are pruned; entries beyond `maxEntries` (default 500) are evicted oldest-first. Pruned and capped entries archive their transcripts (`archiveSessionTranscripts`). `mode: "warn"` only logs without acting; `mode: "enforce"` applies the policy.

`cleanupArchivedSessionTranscripts({directories, olderThanMs, reason: "deleted"|"reset"})` runs after every save when `pruneAfterMs` is configured.

`enforceSessionDiskBudget` is a separate sweep that respects per-store byte caps (defaults reside in `disk-budget.ts`, not directly inspected, but called from `saveSessionStoreUnlocked`).

For memory: `repairShortTermPromotionArtifacts` removes invalid recall entries and stale lock files. `auditShortTermPromotionArtifacts` provides a structured `ShortTermAuditSummary` exposing `entryCount`, `promotedCount`, `spacedEntryCount`, `conceptTaggedEntryCount`, `invalidEntryCount`, plus optional QMD index stats.

### 9.5 Daily reset

Configured via `session.reset.{daily, dailyAt, idleMinutes}`. At the configured time (default 04:00 local), each active session is **archived in place** (`status: archived`) and a fresh session is created with the same key. Old transcripts stay on disk; the new session starts fresh. This does not roll up content into memory — that's done by the dreaming loop.

---

## 10. Tools / hooks / sub-agents interaction

### 10.1 Tools

- `memory_search` and `memory_get` are bundled. They're registered with `api.registerTool((ctx) => createMemorySearchTool({config: ctx.config, agentSessionKey: ctx.sessionKey}), {names: ["memory_search"]})` so the runtime knows exactly which tool name they correspond to.
- The plugin can also register a CLI namespace (`memory`) with subcommands (`status`, `search`, `index`, `rem-harness`, `rem-backfill`, etc.). Subcommands are auto-described via `descriptors`.

### 10.2 Hooks

The session-memory bundled hook (`src/hooks/bundled/session-memory/`) listens on `command:new` and `command:reset`. It is *additive* to the auto-flush — `/new` doesn't trigger the auto-flush (which only triggers on overflow), but the hook saves a final snapshot so context isn't lost on manual reset.

Compaction hooks: `before_compaction`, `after_compaction`, and the generic `session` event are exposed for plugins. The hook bridge in `src/auto-reply/reply/agent-runner-memory.ts:763-776` listens for `evt.stream === "compaction" && evt.data.phase === "end"` to know when to bump `compactionCount` after a flush turn that itself triggered a compaction.

### 10.3 Sub-agents

`active-memory` (the plugin) is a sub-agent that runs *before* every turn:
1. Builds a query from recent conversation (last N user turns + last assistant turn, configurable).
2. Calls `memory_search` (with active-memory-specific QMD search-mode override) and an LLM filter to pick the most relevant memories.
3. Returns a one-line context insertion, or `__OPENCLAW_SILENT_REPLY_TOKEN__` if nothing relevant.
4. The result is injected as a system note (`🧩 active memory: …`) into the next turn.

This is opt-in (`enabled: false` by default), uses a separate model (default `cheap`), respects `allowedChatTypes`, has its own cache TTL (`cacheTtlMs = 15_000`) and prompt styles (`balanced` | `strict` | `contextual` | `recall-heavy` | `precision-heavy` | `preference-only`).

The dreaming narrative subagent uses a `dreaming-narrative-<runId>` namespace and is filtered out of normal session indexing.

### 10.4 Skills

Skills are *not* directly part of memory — they are persona/tool bundles loaded from the workspace. But `SessionSkillSnapshot` (`types.ts:438-445`) is part of `SessionEntry`, and `skillsSnapshot.prompt` becomes part of the system prompt that compaction must summarize. The compaction summarizer is told to preserve "TODOs, decisions, ongoing tasks" — that includes skill state if the agent recorded it. There is no skill-aware memory taxonomy.

---

## 11. Failure modes / open issues observed in code

1. **Schema-on-boot reconciliation** (`memory-schema.ts:84-103`) — `ensureColumn` does runtime ALTER TABLE for the `source` column. No version table, no numbered migration registry. Acceptable for OpenClaw's small surface, but exactly the pattern AGH's `agh-schema-migration` standing directive forbids. AGH's mem-v2 must use numbered migrations from day one even though the surface is similar.
2. **Single in-process memory plugin singleton** (`memory-state.ts:154`) — only one `MemoryPluginCapability` can be registered at a time; subsequent registrations overwrite. Supplements (`registerMemoryCorpusSupplement`) are additive, but the runtime/promptBuilder/flushResolver triple is exclusive. This means you can't compose two backends in one runtime — you have to fork the deployment.
3. **JSONL header is ad hoc** (`transcript.ts:25-36`) — written directly with `JSON.stringify`, no schema versioning beyond `CURRENT_SESSION_VERSION` from `pi-coding-agent`. If the upstream version bumps, OpenClaw silently writes new headers but never migrates old transcripts.
4. **No transactional multi-file writes** — when compaction rewrites the JSONL and updates `sessions.json`, the window between the JSONL atomic-rename and the `sessions.json` update is small but non-zero. A crash between them could yield "compactionCount in store is N, but transcript shows N+1 summaries". The lock queue per `storePath` reduces but does not eliminate this. The recovery story is "the next turn will reconcile".
5. **`hasAlreadyFlushedForCurrentCompaction`** (`memory-flush.ts:108-114`) — only compares `memoryFlushCompactionCount === compactionCount`. If the user manually resets the session (`/new`) without bumping `compactionCount`, the next overflow will re-flush even though the session-memory hook just dumped state. Not a bug per se, just an interesting interaction.
6. **`computeContextHash`** (`memory-flush.ts:125-131`) — hashes only message count + last 3 messages. Two distinct conversations whose tails happen to match (e.g., system-generated nudges) would produce the same hash. The dedup is best-effort, not strict.
7. **Lock-file stale detection** (`store.ts:660`, `short-term-promotion.ts:33`) — uses fixed timeouts (30s for store, 60s for short-term). On a heavily loaded box or while a debugger is paused, these will misfire. The repair tool exists precisely because of this.
8. **"Workspace bootstrap" implicit invariant** — the flush prompt enumerates `MEMORY.md, DREAMS.md, SOUL.md, TOOLS.md, AGENTS.md` as read-only. There's no programmatic enforcement; if the agent ignores the instruction, it can clobber any of them. The system relies on prompt obedience for filesystem invariants.
9. **The `wiki` corpus mechanism** (`registerMemoryCorpusSupplement`) is a soft contract — supplements run after the primary, sorted by `pluginId` (alphabetically). Order is not user-controllable and not documented.
10. **No observability for read paths** — `memory_search` exposes `MemorySearchRuntimeDebug` to the caller via `onDebug` callback, but writes nothing structured to logs/events for retroactive analysis. Recall analytics are only visible through the short-term store (which is itself only updated, not queryable beyond `audit`).

---

## 12. Notable design choices / strengths

1. **Filesystem-first**. All durable state is plain Markdown. The user can `tail`, `grep`, `rsync`, `git commit` memory without daemon involvement. This is a deliberate inversion of the "memory = opaque vector store" pattern.
2. **Index is derived, not authoritative**. `~/.openclaw/memory/<agentId>.sqlite` can be deleted without losing memory — the next `memory index --force` rebuilds it from the Markdown files. The vector embeddings are cached in the same DB, so reindex with the same provider is fast.
3. **Per-agent isolation**. Storage is scoped by agent ID at every level (transcripts, store, memory DB, workspace). Multi-tenancy at the OS level by default.
4. **DM scope as a policy, not a hardcode**. `per-channel-peer` is the default — Alice's WhatsApp and Telegram conversations have separate sessions. The four scopes (`main`, `per-peer`, `per-channel-peer`, `per-account-channel-peer`) trade isolation for convenience.
5. **Compaction summary instructions are explicit and recent-biased**. "PRIORITIZE recent context over older history. The agent needs to know what it was doing, not just what was discussed." Plus identifier-preservation is a separate, custom-overridable policy. This level of care is rare.
6. **Pre-compaction memory flush is a clean pattern**. Treats compaction as lossy and gives the model one chance to write durable facts before it happens. Dedups by `compactionCount` so it can't loop.
7. **Three-phase dreaming with explicit gates**. `minScore`/`minRecallCount`/`minUniqueQueries` are *not* hidden. The promotion machinery has a dedicated `audit` and `repair` surface. This is the most differentiating feature relative to typical "summarize on shutdown" memory systems.
8. **Citations are a first-class config**. `cfg.memory.citations: "auto" | "on" | "off"` controls whether the agent mentions paths/lines. The prompt section adapts. This is exactly the kind of config knob AGH should mirror.
9. **The plugin contract is minimal but sufficient**. `MemoryPluginCapability` is four nullable fields. A plugin can register only `runtime` and inherit the prompt builder; a plugin can register only `flushPlanResolver` and stay out of search. This is composable in the right ways.
10. **Read-only corpus supplements are a separate channel**. `memory-wiki` doesn't replace recall — it adds a second corpus. The agent gets `corpus: "memory" | "wiki" | "all"` as a tool parameter. The runtime composes results, then sorts/clamps.

---

## 13. Lessons applicable to AGH mem-v2

> Speculation here is intentional and clearly flagged. Drop into a TechSpec only after pressure-testing.

1. **Adopt the four-axis capability registration**. `MemoryPluginCapability { promptBuilder, flushPlanResolver, runtime, publicArtifacts }` is the cleanest plugin contract observed across the eight reference repos surveyed by AGH so far. AGH's memory plugin SDK should mirror it (with our own naming: `MemoryCapability`).
2. **Keep transcripts and memory storage separate.** OpenClaw uses JSONL for transcripts (cheap, append-only) and SQLite for the search index (derived). AGH already has `events.db` and `agh.db`; adding a per-agent memory DB at `<AGH_HOME>/memory/<agentId>.db` (instead of mixing into `agh.db`) is consistent with this layering.
3. **Promotion is a first-class lifecycle, not a side effect.** The short-term-recall store + dreaming sweep + scored gates is the right level of explicit. AGH's "auto-memory" should not just append to a daily file — it should track recall counts, query diversity, and expose `audit`/`repair` agent-callable commands.
4. **Writes are file-system-first, reads are index-first.** Don't put a JSON-RPC `memory.write` on the daemon; let the agent edit Markdown via existing tools. Provide `memory.search`/`memory.get` for read.
5. **Compaction is a contract, not a function call.** `before_compaction` / `after_compaction` hooks let plugins re-index, persist analytics, or cancel. AGH's compaction code should expose the same hook points.
6. **Prompt-engineering primitives should be addressable as artifacts.** The compaction summary instructions and the flush prompt are load-bearing. They should live in versioned files (testable, diff-able) rather than as inline strings.
7. **Schema migrations matter even at this scale.** OpenClaw's `ensureColumn` is a smell. AGH should ship the memory DB with a numbered migration registry from v1.
8. **Citations as config, not as model behavior.** `citations: "auto" | "on" | "off"` is a clean knob. AGH should mirror it because operators (especially in regulated environments) need it.
9. **Per-agent namespaces all the way down.** Workspace, transcripts, memory DB, indexed files — every path resolves through `agentId`. This is the foundation for AGH's multi-agent operator UX.
10. **Two trigger gates, one for compaction (overflow) and one for memory (durable save) — both deduped via session-entry counters.** This is the right primitive. Mirror with `compaction_count` and `memory_flush_count` columns on AGH's session table.

---

## 14. Glossary (OpenClaw → AGH terminology guesses)

| OpenClaw | AGH likely equivalent |
|---|---|
| `SessionEntry` | session row in `agh.db.sessions` |
| `sessions.json` | denormalized session-store cache; AGH would normalize |
| `MEMORY.md` | long-term memory artifact (flat Markdown) |
| `memory/YYYY-MM-DD.md` | episodic / daily working memory |
| `DREAMS.md` | promotion audit trail |
| `memory/.dreams/short-term-recall.json` | recall analytics — equivalent to AGH's promotion candidate table |
| dreaming "phases" | consolidation passes (light = ingest, deep = promote, REM = reflect) |
| `MemoryPluginCapability` | `MemoryCapability` interface |
| `flushPlanResolver` | pre-compaction memory-write strategy |
| `MemorySearchManager` | `MemorySearcher` interface |
| `compactionCount` / `memoryFlushCompactionCount` | dedup counters on session row |
| `corpus supplement` | additional searchable corpus (wiki, project-context, etc.) |
| QMD | external local-first sidecar (vector + rerank); analogous to AGH's "external memory daemon" pattern if we go that route |
| `SILENT_REPLY_TOKEN` | sentinel that lets a silent agent turn return without surfacing |

---

## 15. Source code citation table (high-signal files)

| What | Path | Lines |
|---|---|---|
| Plugin capability registry | `src/plugins/memory-state.ts` | 96-263 |
| Memory plugin contract types | `src/memory-host-sdk/runtime-core.ts` | 1-32 |
| Memory search manager interface | `src/memory-host-sdk/host/types.ts` | 68-94 |
| SQLite schema (canonical) | `src/memory-host-sdk/host/memory-schema.ts` | 11-103 |
| Markdown chunker | `src/memory-host-sdk/host/internal.ts` | 335-438 |
| Memory file walker | `src/memory-host-sdk/host/internal.ts` | 116-183 |
| Backend resolver (builtin/qmd) | `src/memory-host-sdk/host/backend-config.ts` | 349-437 |
| MemorySearchConfig defaults | `src/agents/memory-search.ts` | 95-114 |
| SessionEntry schema | `src/config/sessions/types.ts` | 111-259 |
| Session store atomic write + lock | `src/config/sessions/store.ts` | 276-478, 576-681 |
| Session paths + ID safety regex | `src/config/sessions/paths.ts` | 9-330 |
| JSONL header writer | `src/config/sessions/transcript.ts` | 18-37 |
| Pre-compaction flush gate | `src/auto-reply/reply/memory-flush.ts` | 60-131 |
| Memory flush integration (silent run) | `src/auto-reply/reply/agent-runner-memory.ts` | 505-848 |
| Preflight compaction integration | `src/auto-reply/reply/agent-runner-memory.ts` | 346-503 |
| Compaction summarizer | `src/agents/compaction.ts` | 19-281 |
| Dreaming defaults / configuration | `src/memory-host-sdk/dreaming.ts` | 1-630 |
| Memory-core plugin entry | `extensions/memory-core/index.ts` | 1-75 |
| Flush plan builder | `extensions/memory-core/src/flush-plan.ts` | 1-139 |
| memory_search / memory_get tools | `extensions/memory-core/src/tools.ts` | 1-401 |
| Prompt section injection | `extensions/memory-core/src/prompt-section.ts` | 1-38 |
| Short-term promotion store | `extensions/memory-core/src/short-term-promotion.ts` | 1-300+ |
| Dreaming phases (light/deep/rem) | `extensions/memory-core/src/dreaming.ts` | 1-200 |
| Dreaming phase logic | `extensions/memory-core/src/dreaming-phases.ts` | 1-80 |
| Session-memory hook | `src/hooks/bundled/session-memory/handler.ts` | 1-225 |
| Active-memory plugin | `extensions/active-memory/index.ts` | 1-150+ |

---

## 16. Outstanding questions worth answering before AGH commits

(Intentionally left as open questions for the human pairing on mem-v2.)

1. Does AGH's session schema already have `compactionCount` / `memoryFlushCompactionCount` equivalents? (Quick `grep` suggests not — this is one of the load-bearing additions.)
2. Will AGH support multiple memory backends in one runtime, or pick one per agent? OpenClaw's singleton pattern suggests "one per runtime" is the simpler operating model.
3. Where do we draw the line between *memory plugin* and *auxiliary corpus*? OpenClaw's `memory-wiki` registers as a *supplement*, not a *runtime*. Should AGH adopt the same split?
4. Does AGH want the dreaming loop in v1, or is short-term-recall analytics + manual promotion enough? OpenClaw's dreaming machinery is the most complex part of the codebase and is opt-in even there.
5. Should AGH treat flush prompts as user-overridable configuration (OpenClaw lets `cfg.agents.defaults.compaction.memoryFlush.{prompt,systemPrompt}` override) or as runtime-fixed templates? The OpenClaw approach trades safety for power.
