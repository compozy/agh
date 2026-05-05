---
title: OpenFang Memory & Context System — Source-Truth Forensic Analysis
project: openfang
version: openfang-memory v0.4 (schema v8)
sources:
  - /Users/pedronauck/dev/knowledge/.resources/openfang/crates/openfang-memory
  - /Users/pedronauck/dev/knowledge/.resources/openfang/crates/openfang-runtime
  - /Users/pedronauck/dev/knowledge/.resources/openfang/crates/openfang-kernel
  - /Users/pedronauck/dev/knowledge/.resources/openfang/crates/openfang-types/src/memory.rs
  - /Users/pedronauck/dev/knowledge/.resources/openfang/crates/openfang-api/src/routes.rs
  - /Users/pedronauck/dev/knowledge/openfang/wiki/concepts/Knowledge Graph Engine.md (cross-referenced; wiki found stale on multiple counts)
created: 2026-05-04
audience: AGH mem-v2 redesign
---

## TL;DR (≤200 words)

OpenFang's `openfang-memory` crate is a **single shared SQLite database** (WAL, busy_timeout 5000ms) wrapped behind one `Arc<Mutex<Connection>>` and exposed via the `Memory` trait (async methods that internally `spawn_blocking` onto the SQLite mutex). It composes six stores on top of one connection:

1. **`StructuredStore`** — per-agent KV (`kv_store`).
2. **`SemanticStore`** — semantic memory (`memories`) with optional `BLOB` embeddings, cosine re-ranking, soft-delete, optional **HTTP backend** (memory-api gateway with PostgreSQL + pgvector + Jina AI).
3. **`KnowledgeStore`** — RDF-style triples (`entities`, `relations`) but **without `agent_id`** — the wiki claim of agent-scoped graphs is contradicted by source.
4. **`SessionStore`** — per-channel `sessions` (rmp_serde Vec<Message> blob) **plus** `canonical_sessions` (one per agent, cross-channel, threshold-compacted with text concat fallback summary).
5. **`UsageStore`** — LLM cost/token metering (`usage_events`).
6. **`ConsolidationEngine`** — confidence decay only (`memories_merged: 0` is hard-coded — merging is **not implemented**).

Memory is hydrated at agent-loop start (top-5 recall via embedding-aware semantic search filtered by `agent_id`), the assistant turn is auto-`remember`ed with embedding, the canonical session is auto-appended, and a **JSONL session mirror** is auto-written to `<workspace>/sessions/<session-id>.jsonl`. Identity files (`SOUL.md`, `USER.md`, `MEMORY.md`, `IDENTITY.md`, `AGENTS.md`, `BOOTSTRAP.md`, `HEARTBEAT.md`) plus a per-turn `context.md` are auto-loaded by the kernel and assembled into the system prompt by `prompt_builder.rs`. Tools `memory_store` / `memory_recall` write to a **single shared cross-agent UUID** (not the calling agent's namespace) — surprising and undocumented. The wiki overstates the system in several places (4 memory tools vs. 2 actual, agent-scoped KG vs. global KG, `traverse(max_depth)` vs. flat single-hop join, memory consolidation vs. decay only).

---

## 1. Source-of-Truth Map

| Concern | File path | Notes |
|---|---|---|
| Crate root + module export | `crates/openfang-memory/src/lib.rs` | 22 lines; exports `MemorySubstrate` (re-export) plus 8 store modules |
| Substrate composition root | `crates/openfang-memory/src/substrate.rs:30-122` | Single `Arc<Mutex<Connection>>` shared by every store |
| SQLite schema (8 migrations) | `crates/openfang-memory/src/migration.rs:8-329` | `SCHEMA_VERSION = 8`, uses `PRAGMA user_version`, idempotent |
| Structured (KV) | `crates/openfang-memory/src/structured.rs` | Per-agent KV; also stores `agents`, ad-hoc `ALTER` for `session_id`/`identity` |
| Semantic | `crates/openfang-memory/src/semantic.rs` | LIKE / vector cosine; optional HTTP backend |
| Knowledge graph | `crates/openfang-memory/src/knowledge.rs` | `entities` + `relations`; flat join, no traversal |
| Sessions | `crates/openfang-memory/src/session.rs` | Per-channel + canonical; rmp_serde blobs; threshold compaction |
| Consolidation | `crates/openfang-memory/src/consolidation.rs` | Decay only (`memories_merged = 0`) |
| Usage metering | `crates/openfang-memory/src/usage.rs` | Cost / token rollups |
| HTTP gateway client | `crates/openfang-memory/src/http_client.rs` | reqwest blocking; Jina-backed memory-api |
| Memory trait + types | `crates/openfang-types/src/memory.rs` | `Memory: Send + Sync + async`; `MemoryFragment`, `Entity`, `Relation`, `MemoryFilter`, `GraphPattern` |
| Memory config | `crates/openfang-types/src/config.rs:1620-1675` | `MemoryConfig` with `backend`, `decay_rate`, `consolidation_interval_hours`, `embedding_*`, `http_*` |
| Capability gates | `crates/openfang-types/src/capability.rs:47-49` | `Capability::MemoryRead(String)`, `Capability::MemoryWrite(String)` |
| Memory recall in agent loop | `crates/openfang-runtime/src/agent_loop.rs:293-365` (and streaming twin at 1497-1565) | Top-5 hydrated before LLM call |
| Auto-remember | `crates/openfang-runtime/src/agent_loop.rs:692-734` (and streaming twin at 1899-1934) | Persists `User asked X / I responded Y` per turn |
| Compactor | `crates/openfang-runtime/src/compactor.rs` | LLM summarizer with chunked / minimal fallback |
| Per-turn `context.md` | `crates/openfang-runtime/src/agent_context.rs` | `cache_context = false` default; 32 KB cap |
| Workspace context (project type, AGENTS/SOUL/etc.) | `crates/openfang-runtime/src/workspace_context.rs` | mtime cache; 32 KB cap |
| Prompt assembly (memory section, persona, etc.) | `crates/openfang-runtime/src/prompt_builder.rs` | 13 ordered sections; subagent skips most |
| Kernel handle (memory FFI for tools/extensions) | `crates/openfang-runtime/src/kernel_handle.rs:46-96` | `memory_store/recall`, `knowledge_*`, `task_*` |
| Kernel impl of handle | `crates/openfang-kernel/src/kernel.rs:6748-6872` | Critical: `memory_store/recall` route to `shared_memory_agent_id()` |
| Shared-memory UUID | `crates/openfang-kernel/src/kernel.rs:6471-6478` | All zeros + `01` — fixed UUID |
| Identity-file auto-generation | `crates/openfang-kernel/src/kernel.rs:313-447` | `create_new` writes; never overwrites |
| Identity-file auto-loading per turn | `crates/openfang-kernel/src/kernel.rs:2040-2103` (canonical) and `2611-2654` (streaming) | Re-read every turn |
| Daily memory log appender | `crates/openfang-kernel/src/kernel.rs:449-475` | `memory/<YYYY-MM-DD>.md`, capped 1 MB |
| Memory consolidation cron | `crates/openfang-kernel/src/kernel.rs:4327-4361` | `consolidation_interval_hours` (default 24h) |
| HTTP REST surface | `crates/openfang-api/src/routes.rs:3260-3362` | `/api/memory/agents/:id/kv*` (KV only — collapsed to shared UUID) |
| CLI surface | `crates/openfang-cli/src/main.rs:737-772, 1109-1112, 6403-6498` | `openfang memory list/get/set/delete <agent>` (KV only) |
| Tool runner: memory + knowledge | `crates/openfang-runtime/src/tool_runner.rs:697-862, 1703-1967` | 2 memory + 3 knowledge tools (wiki claim of 4 memory tools is wrong) |
| Host functions FFI (extensions) | `crates/openfang-runtime/src/host_functions.rs:320-369` | `host_kv_get/set` capability-gated `MemoryRead/Write(<key>)` |

---

## 2. Storage Architecture

### 2.1 One-DB design — composition root

Every store shares a single `Arc<Mutex<Connection>>`:

```rust
// crates/openfang-memory/src/substrate.rs:46-68
pub fn open(
    db_path: &Path,
    decay_rate: f32,
    memory_config: &MemoryConfig,
) -> OpenFangResult<Self> {
    let conn = Connection::open(db_path).map_err(|e| OpenFangError::Memory(e.to_string()))?;
    conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;")
        .map_err(|e| OpenFangError::Memory(e.to_string()))?;
    run_migrations(&conn).map_err(|e| OpenFangError::Memory(e.to_string()))?;
    let shared = Arc::new(Mutex::new(conn));

    let semantic = Self::create_semantic_store(Arc::clone(&shared), memory_config);

    Ok(Self {
        conn: Arc::clone(&shared),
        structured: StructuredStore::new(Arc::clone(&shared)),
        semantic,
        knowledge: KnowledgeStore::new(Arc::clone(&shared)),
        sessions: SessionStore::new(Arc::clone(&shared)),
        usage: UsageStore::new(Arc::clone(&shared)),
        consolidation: ConsolidationEngine::new(shared, decay_rate),
    })
}
```

**Concurrency model.** Every async API on `Memory` (`get`, `set`, `recall`, `add_entity`, …) wraps the synchronous SQLite call in `tokio::task::spawn_blocking`:

```rust
// crates/openfang-memory/src/substrate.rs:621-639
async fn get(&self, agent_id: AgentId, key: &str) -> ... {
    let store = self.structured.clone();
    let key = key.to_string();
    tokio::task::spawn_blocking(move || store.get(agent_id, &key))
        .await
        .map_err(|e| OpenFangError::Internal(e.to_string()))?
}
```

This is a deliberate one-mutex chokepoint — every read and every write serializes on the same `Mutex<Connection>`. Tradeoff: simple, but the entire memory subsystem is single-writer single-reader per process, with all stores contending. WAL mitigates reader-writer contention but the in-process mutex does not — two readers will still block each other.

### 2.2 SQLite schema (current = v8)

`crates/openfang-memory/src/migration.rs` runs eight numbered migrations. Each migration uses `column_exists` guards and `INSERT OR IGNORE` into a `migrations` table for human-readable history; the canonical version is held in `PRAGMA user_version`.

**v1 — initial schema:**

```sql
-- agents
CREATE TABLE agents (
  id TEXT PRIMARY KEY, name TEXT, manifest BLOB, state TEXT,
  created_at TEXT, updated_at TEXT
);

-- per-channel session blobs
CREATE TABLE sessions (
  id TEXT PRIMARY KEY, agent_id TEXT, messages BLOB,
  context_window_tokens INTEGER DEFAULT 0,
  created_at TEXT, updated_at TEXT
);

-- in-process event log (rarely used; not the primary event bus)
CREATE TABLE events (
  id TEXT PRIMARY KEY, source_agent TEXT, target TEXT,
  payload BLOB, timestamp TEXT
);

-- per-agent KV (composite PK)
CREATE TABLE kv_store (
  agent_id TEXT, key TEXT, value BLOB, version INTEGER DEFAULT 1,
  updated_at TEXT, PRIMARY KEY (agent_id, key)
);

-- task queue (cross-agent collaboration)
CREATE TABLE task_queue (
  id TEXT PRIMARY KEY, agent_id TEXT, task_type TEXT,
  payload BLOB, status TEXT DEFAULT 'pending', priority INTEGER DEFAULT 0,
  scheduled_at TEXT, created_at TEXT, completed_at TEXT
);

-- semantic memory fragments
CREATE TABLE memories (
  id TEXT PRIMARY KEY, agent_id TEXT, content TEXT, source TEXT,
  scope TEXT DEFAULT 'episodic', confidence REAL DEFAULT 1.0,
  metadata TEXT DEFAULT '{}',
  created_at TEXT, accessed_at TEXT, access_count INTEGER DEFAULT 0,
  deleted INTEGER DEFAULT 0
);

-- knowledge graph (NO agent_id — global per-database)
CREATE TABLE entities (
  id TEXT PRIMARY KEY, entity_type TEXT, name TEXT,
  properties TEXT DEFAULT '{}', created_at TEXT, updated_at TEXT
);
CREATE TABLE relations (
  id TEXT PRIMARY KEY, source_entity TEXT, relation_type TEXT,
  target_entity TEXT, properties TEXT DEFAULT '{}',
  confidence REAL DEFAULT 1.0, created_at TEXT
);

-- migration ledger (alongside PRAGMA user_version)
CREATE TABLE migrations (
  version INTEGER PRIMARY KEY, applied_at TEXT, description TEXT
);
```

Indexes from v1: `idx_events_timestamp`, `idx_events_source`, `idx_task_status_priority`, `idx_memories_agent`, `idx_memories_scope`, `idx_relations_source`, `idx_relations_target`, `idx_relations_type`.

**v2 (`task_queue`)** — adds `title, description, assigned_to, created_by, result` columns for human-readable task delegation.

**v3 (`memories.embedding BLOB`)** — adds raw little-endian f32 vector blob; nullable.

**v4 (`usage_events`)** — adds cost-tracking table (id, agent_id, timestamp, model, input_tokens, output_tokens, cost_usd, tool_calls) + `idx_usage_agent_time`, `idx_usage_timestamp`.

**v5 (`canonical_sessions`)** — single row per agent for cross-channel persistence:

```sql
CREATE TABLE canonical_sessions (
  agent_id TEXT PRIMARY KEY, messages BLOB,
  compaction_cursor INTEGER DEFAULT 0,
  compacted_summary TEXT, updated_at TEXT
);
```

**v6** — `sessions.label TEXT` (human-readable session names).

**v7 (`paired_devices`)** — push-token registry for mobile pairing.

**v8 (`audit_entries`)** — Merkle-chained tamper-evident audit log (seq, timestamp, agent_id, action, detail, outcome, prev_hash, hash).

> **Migration discipline observation.** The migration registry is clean and well-bounded. However, `crates/openfang-memory/src/structured.rs:128-136` still hot-patches the `agents` table at every `save_agent` call:
>
> ```rust
> let _ = conn.execute("ALTER TABLE agents ADD COLUMN session_id TEXT DEFAULT ''", []);
> let _ = conn.execute("ALTER TABLE agents ADD COLUMN identity TEXT DEFAULT '{}'", []);
> ```
>
> Errors are intentionally swallowed — works because the second-to-Nth call returns "duplicate column name" — but it's the kind of "EnsureSchema-style" pattern AGH explicitly bans. AGH should not copy this.

### 2.3 Schema vs. wiki claims (truth check)

The wiki article `wiki/concepts/Knowledge Graph Engine.md` is **wrong** in several places. Source-of-truth findings:

| Wiki claim | Source reality |
|---|---|
| `entities.agent_id` and `relations.agent_id` columns | Source has **no** `agent_id` on entities/relations — KG is **global per database**, all agents share. |
| `facts` table for entity-to-literal | **Does not exist**. Properties live in `entities.properties` JSON only. |
| `traverse(start, max_depth)` BFS API | **Not implemented**. `query_graph()` is a flat single-hop SQL JOIN with `LIMIT 100`; `GraphPattern.max_depth` is **read but unused** (see `knowledge.rs:83-188`). |
| Confidence on entities | Schema has `relations.confidence REAL DEFAULT 1.0`, but **no `confidence` column on entities** in v1+ schema. |
| Per-agent KG consolidation `consolidate_kg(from, to)` | **Not implemented**. `ConsolidationEngine::consolidate()` only decays `memories.confidence`. |
| 4 memory tools (`memory_store/recall/delete/list`) | Source has **2** memory tools (`memory_store`, `memory_recall`) plus 3 knowledge tools (`knowledge_add_entity`, `knowledge_add_relation`, `knowledge_query`). No `memory_delete` / `memory_list` LLM tools — those exist only in CLI/REST. |

This is exactly the kind of "marketing-as-spec" rot the AGH greenfield-delete rule prevents — the wiki was generated from product positioning, not source.

---

## 3. Memory Taxonomy & Sources

The runtime distinguishes memories by `MemorySource` (`crates/openfang-types/src/memory.rs:34-49`):

```rust
pub enum MemorySource {
    Conversation,
    Document,
    Observation,
    Inference,
    UserProvided,
    System,
}
```

…and by `scope: String` (free-text, default `"episodic"`). The runtime only ever writes `scope = "episodic"` (`agent_loop.rs:704, 718, 730`) — there is no first-class `semantic`, `procedural`, or `working` scope; `scope` is just an opaque label that operators can use to filter recall.

### 3.1 Five concrete memory layers (run-time view)

| Layer | Storage | Granularity | Lifetime | Per-agent? | Auto-write? |
|---|---|---|---|---|---|
| **Per-channel session** | `sessions` table, rmp_serde Vec<Message> blob | Message (with content blocks) | Created on first user turn; persists until deleted | Yes | Yes (every turn end) |
| **Canonical session (cross-channel)** | `canonical_sessions` (one row/agent) | Message | One per agent across all channels | Yes (PK = agent_id) | Yes (every turn end) |
| **Semantic memory fragment** | `memories` table, with optional embedding BLOB | Free-text + metadata + scope | Soft-deleted via `deleted = 1`, decayed by background job | Yes (`memories.agent_id`) | Yes — `User asked X / I responded Y` per turn |
| **Structured KV** | `kv_store` (`agent_id`,`key`) | JSON value | Until deleted | Yes — but the LLM-facing `memory_store/recall` tools collapse all agents into one namespace | Tool-driven (LLM decision) |
| **Knowledge graph** | `entities` + `relations` | Triple | Until deleted | **No** (global per database) | Tool-driven (LLM decision) |

### 3.2 File-based memory layer (workspace identity files)

OpenFang treats workspace files as a *first-class* prompt input. The kernel auto-generates them when an agent is created (`kernel.rs:313-447`) using `create_new` so user edits are never overwritten:

| File | Purpose | Cap | Loaded into prompt as |
|---|---|---|---|
| `SOUL.md` | Persona ("You are X. Be helpful.") | 32 KB | `## Persona` (`prompt_builder.rs:375-383`) |
| `USER.md` | What the agent has learned about the user | 32 KB | `## User Context` |
| `MEMORY.md` | Curated long-term notes the agent maintains | 32 KB | `## Long-Term Memory` |
| `IDENTITY.md` | YAML frontmatter (archetype, vibe, color, emoji, avatar_url) + visual identity | 32 KB | `## Identity` |
| `AGENTS.md` | Behavioral guidelines (tool-usage protocols, response style) | 32 KB | top-of-prompt (`prompt_builder.rs:93-99`) |
| `BOOTSTRAP.md` | First-run protocol — only injected when `user_name` is unknown | 32 KB | `## First-Run Protocol` |
| `HEARTBEAT.md` | Autonomous-agent checklist (every-heartbeat / daily / weekly) | 32 KB | `## Heartbeat Checklist` (only for `is_autonomous`) |
| `TOOLS.md` | Operator notes about agent tools (not prompt-injected currently) | 32 KB | — (read but not surfaced in prompt) |
| `context.md` | **Per-turn refreshable** external-writer file (e.g., cron writes market data) | 32 KB | `## Workspace Context` (re-read every turn unless `cache_context=true`) |
| `AGENT.json` | Best-effort metadata (created_at, workspace path) — not prompt-loaded | — | — |
| `memory/<YYYY-MM-DD>.md` | Daily append-only log of assistant responses (truncated to 500 chars/entry, capped at 1 MB/file) | 1 MB | — (operator artifact, not prompt-loaded) |

`context.md` deserves special attention — see §4.4. It is the **only** file that is re-read on every agent turn by default; everything else uses `read_identity_file` which mtime-caches on `WorkspaceContext`.

### 3.3 Identity-file generation defaults

`generate_identity_files` (`kernel.rs:315-447`) writes opinionated default content. Notable:

- `SOUL.md` is **template-substituted** with `manifest.name` and `manifest.description`.
- `BOOTSTRAP.md` instructs: greet → discover name → call `memory_store("user_name", X)` → orient → serve. This is how the system trains agents to use the KV.
- `HEARTBEAT.md` is **only** generated when `manifest.autonomous.is_some()`.
- `AGENTS.md` is generated by default with prescriptive content — different from competitor `AGENTS.md` conventions where the file is operator-authored.

---

## 4. Read/Write Lifecycle

### 4.1 The unified `Memory` trait

`crates/openfang-types/src/memory.rs:262-335` defines the contract:

```rust
#[async_trait]
pub trait Memory: Send + Sync {
    // KV (structured)
    async fn get(&self, agent_id: AgentId, key: &str) -> ...;
    async fn set(&self, agent_id: AgentId, key: &str, value: serde_json::Value) -> ...;
    async fn delete(&self, agent_id: AgentId, key: &str) -> ...;

    // Semantic
    async fn remember(&self, agent_id, content, source, scope, metadata) -> MemoryId;
    async fn recall(&self, query, limit, filter: Option<MemoryFilter>) -> Vec<MemoryFragment>;
    async fn forget(&self, id) -> ...;

    // Knowledge graph
    async fn add_entity(&self, entity) -> String;
    async fn add_relation(&self, relation) -> String;
    async fn query_graph(&self, pattern: GraphPattern) -> Vec<GraphMatch>;

    // Maintenance
    async fn consolidate(&self) -> ConsolidationReport;
    async fn export(&self, format) -> Vec<u8>;          // unimplemented (returns empty)
    async fn import(&self, data, format) -> ImportReport; // returns "Import not yet implemented"
}
```

`MemorySubstrate` provides this trait plus a **much wider** synchronous API surface that the kernel uses directly without going through the trait — `save_agent`, `load_canonical`, `append_canonical`, `store_llm_summary`, `write_jsonl_mirror`, `task_post/claim/complete/list`, `recall_with_embedding_async`, `remember_with_embedding_async`, `save_paired_device`, etc. **The trait is more façade than enforced contract.**

### 4.2 Recall (per-turn hydration)

```rust
// crates/openfang-runtime/src/agent_loop.rs:293-338
let memories = if let Some(emb) = embedding_driver {
    match emb.embed_one(user_message).await {
        Ok(query_vec) => {
            memory.recall_with_embedding_async(
                user_message, 5,
                Some(MemoryFilter { agent_id: Some(session.agent_id), ..Default::default() }),
                Some(&query_vec),
            ).await.unwrap_or_default()
        }
        Err(e) => {
            warn!("Embedding recall failed, falling back to text search: {e}");
            memory.recall(user_message, 5,
                Some(MemoryFilter { agent_id: Some(session.agent_id), ..Default::default() }),
            ).await.unwrap_or_default()
        }
    }
} else {
    memory.recall(user_message, 5, /* same filter */).await.unwrap_or_default()
};
```

Hydration rules:
- Always `limit = 5`.
- Always filtered to `agent_id = session.agent_id` (so the per-channel session and canonical session share an agent's memory pool).
- Embedding recall is preferred; text `LIKE %query%` is the fallback.
- **Errors are silently swallowed** with `unwrap_or_default()` — recall failure never aborts the turn (a deliberate availability-over-correctness tradeoff).

The recalled fragments are stitched into the system prompt **after** the kernel-built base prompt:

```rust
// agent_loop.rs:355-365
if !memories.is_empty() {
    let mem_pairs: Vec<(String, String)> =
        memories.iter().map(|m| (String::new(), m.content.clone())).collect();
    system_prompt.push_str("\n\n");
    system_prompt.push_str(&crate::prompt_builder::build_memory_section(&mem_pairs));
}
```

`build_memory_section` (`prompt_builder.rs:310-334`) outputs at most 5 entries, each capped at 500 chars:

```text
## Memory
- Use the recalled memories below to inform your responses.
- Only call memory_recall if you need information not already shown here.
- Store important preferences, decisions, and context with memory_store for future use.

Recalled memories:
- <content capped to 500 chars>
- ...
```

When the recall returns nothing, a static stub appears with prompts to use `memory_recall` (the KV tool, **not** the semantic store — there is no LLM tool for the semantic store).

### 4.3 Auto-remember on turn completion

```rust
// agent_loop.rs:692-734 (and streaming twin at 1899-1934)
let interaction_text = format!(
    "User asked: {}\nI responded: {}",
    user_message, final_response
);
if let Some(emb) = embedding_driver {
    match emb.embed_one(&interaction_text).await {
        Ok(vec) => { memory.remember_with_embedding_async(
            session.agent_id, &interaction_text,
            MemorySource::Conversation, "episodic",
            HashMap::new(), Some(&vec),
        ).await; }
        Err(_) => { memory.remember(session.agent_id, &interaction_text, ..., "episodic", ...).await; }
    }
}
```

**Findings:**
- Auto-remember is **always on** at end of every turn — there is no tag/extraction step deciding what's worth remembering. Every successful interaction generates one `MemoryFragment` with `source=Conversation, scope="episodic"`.
- Format is locked to `User asked: <X>\nI responded: <Y>` — coarse-grained.
- Errors are silently dropped (`let _ = ...`).
- Embedding generation happens lazily per-turn (one extra LLM/embedding call per turn). No batching.

### 4.4 Per-turn `context.md` reload

The most distinctive piece of OpenFang's context handling. From `crates/openfang-runtime/src/agent_context.rs:43-89`:

```rust
pub fn load_context_md(workspace: &Path, cache_context: bool) -> Option<String> {
    let path = workspace.join(CONTEXT_FILENAME);

    if cache_context {
        if let Some(cached) = get_cached(&path) { return Some(cached); }
    }

    match read_capped(&path) {
        Ok(Some(content)) => { store_cached(&path, &content); Some(content) }
        Ok(None) => if cache_context { get_cached(&path) } else { None },
        Err(e) => {
            // I/O error mid-write: fall back to the last good content
            if let Some(prev) = get_cached(&path) {
                warn!(... "Failed to re-read context.md; falling back to cached content");
                Some(prev)
            } else { None }
        }
    }
}
```

This is wired into the kernel turn lifecycle (`kernel.rs:2098-2103`):

```rust
// Re-read context.md per turn by default so external writers
// (cron jobs, integrations) reach the LLM on the next message.
// Opt out via `cache_context = true` on the manifest. (#843)
context_md: manifest.workspace.as_ref().and_then(|w| {
    openfang_runtime::agent_context::load_context_md(w, manifest.cache_context)
}),
```

**Why this matters for AGH.** This is OpenFang's solution to the "external state needs to reach the LLM" problem — instead of pushing events into the runtime, the runtime pulls a single agreed-upon file every turn. Writers (cron jobs, OS hooks, data feeders) only need to write to a known path. Tradeoff: per-turn disk read (mitigated by mtime fallback), and the file is single-namespace per workspace.

It also has a **mid-write resilience** pattern: if the read fails (file truncated mid-write, encoding error), it returns the last successfully cached content with a warning — context never disappears mid-conversation. This is the kind of "best-effort with fallback" pattern AGH should consider for any file-backed memory.

### 4.5 Session save lifecycle

After every successful turn:

1. `memory.save_session_async(session)` — rmp_serde-encode `Vec<Message>` and write the per-channel session row.
2. `memory.append_canonical(agent_id, &new_messages, None)` — append the new turn to the cross-channel canonical session, possibly triggering compaction (see §5).
3. `memory.write_jsonl_mirror(&session, &workspace.join("sessions"))` — write a *human-readable* JSONL transcript file at `<workspace>/sessions/<session-id>.jsonl`. Each line: `{"timestamp", "role", "content", "tool_use"?}`. Tool-use blocks become a list in `tool_use`; image blocks become `[image: <mime>]`; thinking blocks become `[thinking: <truncated>]`.
4. `append_daily_memory_log(workspace, &result.response)` — append a 500-char snippet of the assistant response to `<workspace>/memory/<YYYY-MM-DD>.md`, capped at 1 MB/file.
5. `kernel.metering.record(UsageRecord{...})` — write usage_event row.
6. **Post-loop compaction check** (kernel.rs:2268-2285): if estimated tokens > 70% of context window, schedule background compaction.

**Strong observation:** OpenFang treats the JSONL mirror as a *secondary* artifact. The primary store of conversation is the rmp_serde blob in SQLite. The JSONL is for operator inspection (`tail -f`) and for OpenClaw → OpenFang migration. AGH already has its own SQLite + events.db split — this dual-format pattern is more of a *side benefit* than something to copy.

### 4.6 Canonical session and threshold compaction

```rust
// crates/openfang-memory/src/session.rs:415-475
pub fn append_canonical(&self, agent_id, new_messages, compaction_threshold) -> CanonicalSession {
    let mut canonical = self.load_canonical(agent_id)?;
    canonical.messages.extend(new_messages.iter().cloned());
    let threshold = compaction_threshold.unwrap_or(DEFAULT_COMPACTION_THRESHOLD); // 100

    if canonical.messages.len() > threshold {
        let keep_count = DEFAULT_CANONICAL_WINDOW; // 50
        let to_compact = canonical.messages.len().saturating_sub(keep_count);
        if to_compact > canonical.compaction_cursor {
            // Build a TEXT-CONCAT summary (NO LLM): "User: ...\nAssistant: ..."
            //   with each individual message capped at 200 chars
            // Concatenate to existing compacted_summary
            // Truncate full summary to 4000 chars (UTF-8 safe)
            // Trim the message list to recent window
        }
    }
    canonical.updated_at = Utc::now().to_rfc3339();
    self.save_canonical(&canonical)?;
}
```

**Observation:** `append_canonical` does **text-concatenation** compaction, not LLM summarization. The LLM-summarized variant lives in `crates/openfang-runtime/src/compactor.rs` and is invoked separately by the kernel (`kernel.rs:3404`: `store.store_llm_summary(...)`). So there are **two compaction engines**:

| Engine | Trigger | Method | Storage write |
|---|---|---|---|
| `append_canonical` (in `openfang-memory`) | Auto, every turn when `messages > 100` | Text concatenation, individual msg → 200 chars, summary → 4000 chars | `canonical_sessions.compacted_summary` |
| `compactor.rs` (in `openfang-runtime`) | Post-loop check when est_tokens > 70% of context window | Multi-stage LLM summarization (full → chunked merge → minimal fallback) | `store_llm_summary` overwrites the canonical session |

Both write to the same `compacted_summary` column. The runtime expects the LLM compactor to win (kernel triggers it explicitly), but if it never runs, the simpler text-concat compactor still keeps the canonical session bounded.

Canonical session readback (`crates/openfang-memory/src/session.rs:481-491`):

```rust
pub fn canonical_context(&self, agent_id, window_size) -> (Option<String>, Vec<Message>) {
    let canonical = self.load_canonical(agent_id)?;
    let window = window_size.unwrap_or(DEFAULT_CANONICAL_WINDOW); // 50
    let start = canonical.messages.len().saturating_sub(window);
    let recent = canonical.messages[start..].to_vec();
    Ok((canonical.compacted_summary.clone(), recent))
}
```

This (summary, recent_messages) pair is then injected as the **first user message** of the LLM call (`agent_loop.rs:407-417`) — not the system prompt, to keep the system prompt stable for provider prompt caching.

### 4.7 LLM-based compaction (`compactor.rs`)

A serious effort, with three escalating fallbacks:

1. **Stage 1 — single-pass full summarization** of all but the most recent `keep_recent` (default 10) messages, with retry up to 3 times.
2. **Stage 2 — adaptive chunked summarization**: split into chunks (`base_chunk_ratio = 0.4`, `min_chunk_ratio = 0.15`, adapted to average message length), summarize each independently, then merge with one final LLM call. Even if one chunk fails, others still contribute partial summaries.
3. **Stage 3 — minimal text fallback**: write a placeholder note (`[Session compacted: N messages removed. Recent K messages preserved. Summarization was unavailable.]`).

`CompactionConfig` defaults (`compactor.rs:49-65`):
```text
threshold:                   30        // message-count trigger (less used)
keep_recent:                 10        // messages preserved verbatim
max_summary_tokens:          1024
base_chunk_ratio:            0.4
min_chunk_ratio:             0.15
safety_margin:               1.2
summarization_overhead_tokens: 4096
max_chunk_chars:             80_000
max_retries:                 3
token_threshold_ratio:       0.7       // PRIMARY trigger: 70% of context_window
context_window_tokens:       200_000   // default fallback if unknown
```

The token-based trigger is what runs in production (`kernel.rs:2270-2285`), not the message-count one.

**Pre-compaction conversation rendering** (`compactor.rs:327-415`) is interesting:
- Tool-use blocks are rendered `[Used tool '<name>' with params: <preview>]`.
- Tool-result blocks are rendered `[Tool result (OK|ERROR): <preview>]`, after stripping base64 blobs and injection markers via `session_repair::strip_tool_result_details`.
- Oversized single messages are individually truncated.
- Image blocks become `[Image: <mime>]`.
- Thinking blocks are dropped entirely from the summary input.

**Tool-pair safety** (`compactor.rs:614-648`): the split index is *adjusted backwards* if it would land between an Assistant ToolUse turn and its corresponding User ToolResult turn — both stay in the kept-verbatim window so the LLM never sees an orphan ToolUse. This is a real correctness bug that AGH should learn from: when summarizing chat history, **never split a tool-call/tool-result pair**.

---

## 5. Knowledge Graph — what's actually there

The wiki overstated the KG. Source-of-truth:

### 5.1 Schema (entities + relations only)

```sql
CREATE TABLE entities (
  id TEXT PRIMARY KEY,
  entity_type TEXT,
  name TEXT,
  properties TEXT DEFAULT '{}',     -- JSON
  created_at TEXT, updated_at TEXT
);
CREATE TABLE relations (
  id TEXT PRIMARY KEY,
  source_entity TEXT,                -- references entities.id
  relation_type TEXT,                -- JSON-encoded RelationType enum
  target_entity TEXT,
  properties TEXT DEFAULT '{}',
  confidence REAL DEFAULT 1.0,
  created_at TEXT
);
```

There is no `agent_id` on either table — the KG is a **single global graph per database**. There are no foreign-key constraints (`relations.source_entity` is not enforced as a foreign key — see `migration.rs:152-172`).

### 5.2 Typed enums

```rust
// crates/openfang-types/src/memory.rs:133-199
pub enum EntityType { Person, Organization, Project, Concept, Event, Location, Document, Tool, Custom(String) }
pub enum RelationType {
    WorksAt, KnowsAbout, RelatedTo, DependsOn, OwnedBy, CreatedBy,
    LocatedIn, PartOf, Uses, Produces, Custom(String),
}
```

So the KG ontology is a **closed, opinionated 8-entity / 10-relation set with `Custom(String)` escape hatches**. The tool runner accepts free strings and maps them via case-insensitive `parse_entity_type` / `parse_relation_type` — typos go to `Custom("unknown_type")` rather than fail.

### 5.3 Query semantics — flat single-hop join

```rust
// crates/openfang-memory/src/knowledge.rs:83-188
pub fn query_graph(&self, pattern: GraphPattern) -> Vec<GraphMatch> {
    let mut sql = "SELECT s.*, r.*, t.* FROM relations r
                   JOIN entities s ON r.source_entity = s.id
                   JOIN entities t ON r.target_entity = t.id
                   WHERE 1=1";
    if pattern.source.is_some() { sql += " AND (s.id = ? OR s.name = ?)"; }
    if pattern.relation.is_some() { sql += " AND r.relation_type = ?"; }
    if pattern.target.is_some() { sql += " AND (t.id = ? OR t.name = ?)"; }
    sql += " LIMIT 100";
    // ... maps rows to Vec<GraphMatch{ source, relation, target }>
}
```

`GraphPattern.max_depth: u32` is **read but not used** (see `knowledge.rs:83-188` — `max_depth` is in the struct but the SQL is single-hop). The wiki's promise of "BFS up to N hops with cycle detection" is fiction.

`source` and `target` accept either entity ID **or** entity name — a usability concession (LLM tools can pass names without first resolving IDs) that has no uniqueness guarantee (you can have two `Person` entities both named "John Smith"). On insert, `add_entity` does `ON CONFLICT(id) DO UPDATE` so passing the same `id` updates the entity, but the runtime tool always passes empty `id` so a new UUID is generated every time — meaning the LLM cannot upsert by name.

### 5.4 Confidence as a `relations`-only attribute

`relations.confidence REAL DEFAULT 1.0` is set at insert (default 1.0; LLM can pass via the `knowledge_add_relation` tool, default 1.0). It is **not consulted by `query_graph`** — there's no minimum-confidence filter. It only shows up when the runtime renders results to the LLM (`tool_runner.rs:1956-1964`):

```rust
output.push_str(&format!(
    "\n  {} ({:?}) --[{:?} ({:.0}%)]--> {} ({:?})",
    m.source.name, m.source.entity_type,
    m.relation.relation, m.relation.confidence * 100.0,
    m.target.name, m.target.entity_type,
));
```

So the LLM *sees* confidence as a percentage but cannot filter by it without doing post-processing.

### 5.5 LLM-facing tools

Three tools, all wired through `KernelHandle::knowledge_*` which delegates to `Memory::add_entity / add_relation / query_graph`:

| Tool | Schema (excerpt) | Backed by |
|---|---|---|
| `knowledge_add_entity` | `{name, entity_type, properties?}` | `KnowledgeStore::add_entity` |
| `knowledge_add_relation` | `{source, relation, target, confidence?, properties?}` | `KnowledgeStore::add_relation` |
| `knowledge_query` | `{source?, relation?, target?, max_depth?}` (max_depth ignored) | `KnowledgeStore::query_graph`, output capped at 100 rows |

There are **no** delete or update tools. The KG is append-only from the LLM perspective. Operators can `sqlite3 openfang.db` directly.

### 5.6 No exports / imports

`Memory::export` returns empty bytes; `Memory::import` returns `"Import not yet implemented in Phase 1"` (`substrate.rs:716-728`). The `ExportFormat::{Json, MessagePack}` variants are placeholder. Cross-agent KG sharing, RDF/Turtle round-tripping, Neo4j export — **none of it exists in source**.

---

## 6. The `memory_store` / `memory_recall` Surprise

This is the most consequential gotcha for any AGH design that takes inspiration from OpenFang.

### 6.1 The shared-memory UUID

```rust
// crates/openfang-kernel/src/kernel.rs:6471-6478
/// A well-known agent ID used for shared memory operations across agents.
/// This is a fixed UUID so all agents read/write to the same namespace.
pub fn shared_memory_agent_id() -> AgentId {
    AgentId(uuid::Uuid::from_bytes([
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
    ]))
}

// crates/openfang-kernel/src/kernel.rs:6748-6760
fn memory_store(&self, key: &str, value: serde_json::Value) -> Result<(), String> {
    let agent_id = shared_memory_agent_id();
    self.memory.structured_set(agent_id, key, value).map_err(...)
}

fn memory_recall(&self, key: &str) -> Result<Option<serde_json::Value>, String> {
    let agent_id = shared_memory_agent_id();
    self.memory.structured_get(agent_id, key).map_err(...)
}
```

**Implication.** When the BOOTSTRAP.md template tells the agent "Use the `memory_store` tool with key `user_name` and the user's name", that name is written to the **shared global namespace**, not to the agent's own KV. Every agent on the daemon reads the same `user_name`. The `prompt_builder` (`kernel.rs:2003-2008`) then reads from the same shared UUID:

```rust
let user_name = self.memory
    .structured_get(shared_id, "user_name")     // shared, not agent-specific
    .ok().flatten()
    .and_then(|v| v.as_str().map(String::from));
```

The HTTP API `/api/memory/agents/:id/kv*` and the CLI `openfang memory list/get/set <agent>` both **silently ignore the `:id`/`<agent>` argument** and operate on the shared UUID:

```rust
// crates/openfang-api/src/routes.rs:3267-3289
pub async fn get_agent_kv(State(state): State<Arc<AppState>>, Path(_id): Path<String>) -> ... {
    let agent_id = openfang_kernel::kernel::shared_memory_agent_id();  // <-- _id ignored
    match state.kernel.memory.list_kv(agent_id) { ... }
}

// Note: memory_store tool writes to a shared namespace, so we read from that
// same namespace regardless of which agent ID is in the URL.
```

**Why this matters for AGH design.**

This pattern is the *opposite* of what AGH should do. The convenience of "every agent sees the same `user_name`" is at war with capability isolation, multi-tenancy, and the AGH security model. Two design lessons:

1. **Cross-agent KV is genuinely useful** (sharing user identity, common configuration). But it should be an **explicit `shared/`** namespace (e.g., `agent:share/user_name`), not a hidden global default.
2. **CLI/REST must mean what they say.** `openfang memory list <agent>` returning shared KV when called with an agent ID is a dishonest UI — exactly the "truthful UI" antipattern called out in `DESIGN.md`. AGH should reject any tool/endpoint design that pretends to scope to an agent but secretly collapses to a global.

### 6.2 Per-agent KV is reachable

The `Memory::get/set/delete` trait *does* take `agent_id` and the `kv_store` table *is* keyed on `(agent_id, key)`. So per-agent KV is fully functional in the substrate — it's only the LLM-facing `memory_store/recall` tools and the operator-facing CLI/REST that route through the shared UUID. Any AGH design that allows "shared memory" must keep the per-agent KV path first-class and the shared path opt-in.

---

## 7. Embeddings & Vector Search

### 7.1 Storage format

Embeddings are stored as raw little-endian f32 BLOBs in the `memories.embedding` column (added in v3 migration). `crates/openfang-memory/src/semantic.rs:492-506`:

```rust
fn embedding_to_bytes(embedding: &[f32]) -> Vec<u8> {
    let mut bytes = Vec::with_capacity(embedding.len() * 4);
    for &val in embedding { bytes.extend_from_slice(&val.to_le_bytes()); }
    bytes
}

fn embedding_from_bytes(bytes: &[u8]) -> Vec<f32> {
    bytes.chunks_exact(4)
        .map(|chunk| f32::from_le_bytes([chunk[0], chunk[1], chunk[2], chunk[3]]))
        .collect()
}
```

No dimensionality column — dimensions inferred from blob length. No model name column either; if you switch models, old vectors silently coexist with new ones in the same table. **Recall does not check that query and stored embeddings have matching dims** — `cosine_similarity` returns `0.0` on mismatch (`semantic.rs:472-489`), so cross-dim mixing is functionally a no-op rather than a panic.

### 7.2 Recall algorithm

Two paths, with explicit re-ranking (`semantic.rs:186-378`):

1. **No query embedding** (`recall` or `recall_with_embedding(.., None)`): SQL `LIKE %query%` over `content`, ordered by `accessed_at DESC, access_count DESC`, hard `LIMIT $limit`.
2. **Query embedding present**: SQL fetches `(limit * 10).max(100)` candidates without LIKE filter (`fetch_limit`), in-memory sorts by cosine similarity descending, truncates to `limit`. Memories without stored embeddings are sorted last (cosine similarity defaults to `-1.0`).

Filters honored on both paths: `agent_id`, `scope`, `min_confidence`, `source`. Time filters (`after` / `before` on `MemoryFilter`) are **declared but not honored** (`semantic.rs:232-254` — see absence of `before/after` SQL).

After the result is returned, `accessed_at` and `access_count` are bumped (`semantic.rs:370-376`):

```rust
for frag in &fragments {
    let _ = conn.execute(
        "UPDATE memories SET access_count = access_count + 1, accessed_at = ?1 WHERE id = ?2",
        rusqlite::params![Utc::now().to_rfc3339(), frag.id.0.to_string()],
    );
}
```

This *also* fires the bump for the `LIKE` path, so heavy users of LIKE-recall accumulate `access_count` even though they didn't get a smart match. The decay engine uses `accessed_at` to identify stale memories (§9.1), so any read counts as "touched" and resets the staleness clock.

### 7.3 Optional HTTP backend (memory-api gateway)

`crates/openfang-memory/src/http_client.rs` — opt-in via `[memory] backend = "http"` config. Offloads `remember` and `recall` to a separate `memory-api` service (PostgreSQL + pgvector + Jina AI embeddings, per the wire contract in `http_client.rs:32-86`):

```rust
// HTTP store: POST /memory/store {content, category, agentId, source, importance, tags} → {id, deduplicated}
// HTTP search: POST /memory/search {query, limit, category} → {results, count}
```

Failure mode: the HTTP backend is a strict overlay — `remember` falls back to local SQLite on HTTP error (warn-logged), `recall` falls back to local SQLite on HTTP error, `forget` is local-only with a warning ("memory-api doesn't support delete yet"). KG / KV / sessions are **always local SQLite** even when `backend = "http"`.

This is a clean pattern: the *gateway* runs the heavy index, but the local DB is the authoritative fallback. Any AGH multi-tenant or networked-memory design should mirror this "remote optional, local authoritative" structure.

### 7.4 Embedding driver (separate crate)

Separately in `crates/openfang-runtime/src/embedding.rs` — an OpenAI-compatible HTTP driver (also works against Ollama). Inferred dimensions table (`embedding.rs:107-116`): `text-embedding-3-small=1536`, `text-embedding-3-large=3072`, `text-embedding-ada-002=1536`, `all-MiniLM-L6-v2=384`. Default model in `MemoryConfig::default()` is `"all-MiniLM-L6-v2"` — i.e., 384-dim — suggesting Ollama-on-device is the preferred default.

---

## 8. Agent / Session Persistence

### 8.1 `agents` table

Stores the AgentManifest (rmp_serde-encoded named-fields blob), state (JSON enum), and identity (JSON). Has lenient deserialization with auto-repair (`structured.rs:262-414`):

- Missing columns fall back through cascading `prepare()` attempts.
- Failed manifest decodes are *skipped* (warn-logged, not deleted).
- Successfully decoded manifests are **re-serialized and written back** if their bytes don't match the round-trip — schema upgrades happen lazily on read.
- Duplicate names are deduplicated (first wins, lowercase comparison).

This is a "self-healing read path" pattern that AGH explicitly bans (greenfield-delete, no schema fallbacks). Mentioned here only because it is the kind of accumulated technical debt the AGH design rules forbid.

### 8.2 Session persistence

Sessions are `Vec<Message>` blobs serialized via rmp_serde. Per-channel sessions live in `sessions`; cross-channel canonical lives in `canonical_sessions` (one row per agent). Both can be deleted independently.

`sessions.label` (added v6) lets operators give sessions human names. `find_session_by_label(agent_id, label)` looks them up.

### 8.3 JSONL session mirror

`crates/openfang-memory/src/session.rs:533-617`:

Each line is a JSON object:
```json
{ "timestamp": "...", "role": "user|assistant|system",
  "content": "<concatenated text>",
  "tool_use": [ {"type":"tool_use","id":"...","name":"...","input":{...}}, ... ] }
```

Tool-use and tool-result blocks both go to `tool_use[]` (despite the name). Image blocks become `[image: <mime>]` text. Thinking blocks become `[thinking: <preview>]` text.

Side-effect of `kernel.process_turn`: every successful turn writes the entire session to JSONL. Idempotent (overwrite each turn). This is the migration path for OpenClaw users (which used JSONL transcripts as the primary store).

---

## 9. Maintenance — Consolidation, Decay, GC

### 9.1 Confidence decay (the only consolidation that ships)

```rust
// crates/openfang-memory/src/consolidation.rs:27-53
pub fn consolidate(&self) -> ConsolidationReport {
    // Decay confidence of memories not accessed in the last 7 days
    let cutoff = (Utc::now() - chrono::Duration::days(7)).to_rfc3339();
    let decay_factor = 1.0 - self.decay_rate as f64;

    let decayed = conn.execute(
        "UPDATE memories SET confidence = MAX(0.1, confidence * ?1)
         WHERE deleted = 0 AND accessed_at < ?2 AND confidence > 0.1",
        rusqlite::params![decay_factor, cutoff],
    )?;
    Ok(ConsolidationReport {
        memories_merged: 0,        // <-- HARD-CODED ZERO
        memories_decayed: decayed,
        duration_ms: ...,
    })
}
```

Decay rule:
- Only `memories` (the semantic store) is touched. KV / KG / sessions are not consolidated.
- 7-day inactivity threshold; not configurable.
- Multiplicative decay (`confidence *= 1 - decay_rate`).
- Floor at `0.1` (memories never decay below).
- **Merging is not implemented.** The wiki promises "duplicate detection / merging" — `memories_merged` is hard-coded zero.

### 9.2 Consolidation scheduling

```rust
// crates/openfang-kernel/src/kernel.rs:4327-4361
let interval_hours = self.config.memory.consolidation_interval_hours; // default 24h
if interval_hours > 0 {
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(interval_hours * 3600));
        interval.tick().await; // Skip first immediate tick
        loop {
            interval.tick().await;
            if kernel.supervisor.is_shutting_down() { break; }
            match kernel.memory.consolidate().await { ... }
        }
    });
}
```

Background task, default every 24 hours, no work-stealing or distributed consolidation. `0` disables.

### 9.3 Usage-event cleanup

`UsageStore::cleanup_old(days)` deletes `usage_events` older than N days. Wired in via metering at `kernel.rs:4314` (with a 90-day retention).

### 9.4 No GC for sessions, KV, KG

There is no scheduled cleanup for:
- Old sessions (`sessions` grow unboundedly; operators must `openfang sessions delete` manually).
- Stale KV entries (no expiration).
- Orphan entities (an entity whose every relation has been deleted is not garbage-collected).
- Fully decayed memories (confidence floor `0.1` keeps them around forever).

This is consistent with OpenFang's "operator-managed retention" stance (the KG wiki says "operators run bulk SQL through `openfang memory prune` or custom scripts. No built-in retention policy exists" — **a CLI command that doesn't actually exist**).

---

## 10. Hooks, Skills, Sub-agents — Memory Interactions

### 10.1 No memory-specific hooks

The hook event enum (used at `agent_loop.rs:340-353` and elsewhere) covers `BeforePromptBuild`, `AgentLoopEnd` — but **no `BeforeRecall` / `AfterRecall` / `BeforeRemember` / `AfterRemember` hooks**. Memory operations are not observable to extensions other than via the host_functions FFI. Extensions can call `host_kv_get/set` directly but cannot intercept the runtime's automatic recall/remember.

### 10.2 Capability-gated extension memory

```rust
// crates/openfang-types/src/capability.rs:47-49
MemoryRead(String),    // pattern (e.g., "user.*")
MemoryWrite(String),
```

Pattern matching is glob-style (`crates/openfang-types/src/capability.rs:131-134`). Extensions declare capabilities in their manifest; `host_kv_get/set` enforces them per-key (`crates/openfang-runtime/src/host_functions.rs:329-333, 355-359`).

LLM tools `memory_store` / `memory_recall` **do not enforce capabilities** — they go straight through `KernelHandle` which collapses to the shared UUID. Capability gates are an extension-only safety net.

### 10.3 Skills

Skills (`crates/openfang-skills`) declare a `memory_key: String` (`crates/openfang-hands/src/lib.rs:142-143`) — a hint that the runtime can pre-load a structured KV value for the skill. Otherwise skills don't talk to memory directly; they ship as tool collections + system-prompt fragments.

### 10.4 Sub-agents

The prompt builder treats sub-agents specially (`prompt_builder.rs:88-99, 124-200`): it skips `AGENTS.md`, persona files, `BOOTSTRAP.md`, peer-agent awareness, and channel awareness — keeping the sub-agent prompt minimal. Sub-agents share the same `MemorySubstrate` instance with their parent (no isolation), but with the same per-agent KV scoping.

There is no parent→child memory inheritance, no "spawn with memory snapshot" — a spawned sub-agent has an empty per-agent KV / semantic store unless the parent explicitly writes for it.

---

## 11. CLI & REST Surface (truthful inventory)

**LLM tools (5 total):**

| Tool | Purpose | Backed by |
|---|---|---|
| `memory_store(key, value)` | Cross-agent shared KV write | `KernelHandle::memory_store` → shared UUID `kv_store` |
| `memory_recall(key)` | Cross-agent shared KV read | `KernelHandle::memory_recall` → shared UUID `kv_store` |
| `knowledge_add_entity(name, entity_type, properties?)` | Append entity | `KernelHandle::knowledge_add_entity` → `entities` |
| `knowledge_add_relation(source, relation, target, confidence?, properties?)` | Append relation | `KernelHandle::knowledge_add_relation` → `relations` |
| `knowledge_query(source?, relation?, target?, max_depth?)` | Single-hop pattern match | `KernelHandle::knowledge_query` → flat JOIN, max_depth ignored |

There is **no LLM tool** to delete, list, or update KV; **no LLM tool** for the semantic memory (`remember/recall/forget`) — semantic operations happen *automatically* in the agent loop, not LLM-driven; **no LLM tool** for KG delete/update.

**REST endpoints (memory-related):**

| Method | Path | Behavior |
|---|---|---|
| GET | `/api/memory/agents/:id/kv` | List KV — silently uses shared UUID |
| GET | `/api/memory/agents/:id/kv/:key` | Get KV value — silently uses shared UUID |
| PUT | `/api/memory/agents/:id/kv/:key` | Set KV value — silently uses shared UUID |
| DELETE | `/api/memory/agents/:id/kv/:key` | Delete KV value — silently uses shared UUID |

Plus the substrate methods `list_kv`, `structured_get`, `structured_set`, `structured_delete` are all reachable through other internal endpoints — but the `/api/memory/...` path is the explicit surface and it is the dishonest one.

There is no `/api/memory/agents/:id/recall`, no `/api/memory/agents/:id/remember`, no `/api/memory/knowledge/...` route — the entire semantic store and KG are runtime-only / SQLite-only from an external API standpoint.

**CLI:**

```text
openfang memory list <agent> [--json]    # → list_kv on shared UUID
openfang memory get <agent> <key> [--json]
openfang memory set <agent> <key> <value>
openfang memory delete <agent> <key>
```

Same surprise: `<agent>` is the parameter but ignored.

---

## 12. Failure Modes, TODOs, and Code-Level Smells

Catalogued from inline source comments and the absence of features that wiki/README claim:

1. **`Memory::export` returns empty bytes** — `substrate.rs:716-719`:
   ```rust
   async fn export(&self, format: ExportFormat) -> OpenFangResult<Vec<u8>> {
       let _ = format;
       Ok(Vec::new())
   }
   ```
2. **`Memory::import` is a stub** — `substrate.rs:721-728` returns `errors: vec!["Import not yet implemented in Phase 1".to_string()]`.
3. **`memories_merged` hard-coded to 0** — `consolidation.rs:48-52`. Wiki claims duplicate-merging; source has decay only.
4. **`GraphPattern.max_depth` ignored** — `knowledge.rs:83-188`. Wiki promises BFS traversal; source is single-hop JOIN.
5. **MemoryFilter `before` / `after`** unused — `semantic.rs:232-254`. Time-window queries don't actually filter by time.
6. **HTTP `forget()` is local-only** — `semantic.rs:385-401` warns `"forget() not supported via HTTP backend, local-only soft-delete"`.
7. **Auto-repair on `agents` table** is `let _ = conn.execute(... ALTER TABLE ... ADD COLUMN ...)` at every `save_agent` — `structured.rs:128-136`. Schema drift glued together at runtime.
8. **`UsageRecord.tool_calls` from `iterations.saturating_sub(1)`** — `kernel.rs:2261`. Tool-call count is approximated by loop iterations minus 1, not the actual number of tool invocations.
9. **JSONL mirror writes the whole session every turn** — `kernel.rs:2227-2236`. No append; full overwrite. Cost grows quadratically with session length.
10. **Recall errors silently dropped** — `agent_loop.rs:294-338` uses `.unwrap_or_default()` — recall failure becomes "no memories" with no upward signal.
11. **Per-agent KV available but the LLM cannot use it** — the `memory_store/recall` tools always go to shared UUID. No tool wires through `Memory::set(session.agent_id, ...)`.
12. **`access_count` bump fires on LIKE-recall too** — `semantic.rs:370-376`. Even a typo-triggered substring match will reset the staleness clock for all returned memories.
13. **`memory.consolidation_threshold = 10_000`** in config but **not consulted anywhere** in source — declared in `MemoryConfig` (`config.rs:1628`) and never read. Dead config.
14. **CLI `openfang memory prune` referenced in wiki — does not exist** in source (`crates/openfang-cli/src/main.rs:737-772` defines only `list/get/set/delete`).
15. **`evidence.rs` test for `test_consolidation_decays_old_memories`** writes a literal `'\"conversation\"'` (escaped JSON string) as a `MemorySource` — the JSON escape pattern reveals that `MemorySource` is serialized as `"conversation"` (with JSON quoting) and stored that way, not as a clean enum string.

---

## 13. Genuinely Novel / Notably Good Patterns

Despite the gaps, OpenFang has some patterns AGH should consider:

### 13.1 Per-turn `context.md` file with mid-write resilience

Already detailed in §4.4. The key insight: **agents need a stable, file-based way for external systems (cron, integrations, OS hooks) to push fresh context into the LLM** without the runtime having to know about every external producer. `context.md` is a write-by-anyone, read-once-per-turn convention. The mid-write fallback (return last good content on read failure) is a thoughtful resilience touch.

### 13.2 Two-tier compaction (text-concat + LLM fallback)

§4.6 / §4.7. The text-concat compactor keeps canonical sessions bounded *even when the LLM compactor is unavailable* — a free defence against unbounded message accumulation. Cheap to compute, no LLM dependency, runs in the same `append_canonical` write transaction.

### 13.3 Tool-pair-aware split adjustment

§4.7. `adjust_split_for_tool_pairs` (`compactor.rs:614-648`) prevents compaction from leaving an orphan ToolUse without its ToolResult. Every chat-history compactor in any agent system must do this; OpenFang's implementation is small, focused, and correct.

### 13.4 Workspace identity files as prompt building blocks

§3.2. The clean separation of concerns:
- `SOUL.md` = persona.
- `IDENTITY.md` = visual/personality frontmatter + free body.
- `USER.md` = user knowledge.
- `MEMORY.md` = curated long-term notes (the file the LLM is supposed to maintain).
- `AGENTS.md` = behavioral guidelines (operator-authored).
- `BOOTSTRAP.md` = first-run protocol (only injected when `user_name` unknown — clean conditional).
- `HEARTBEAT.md` = autonomous-mode-only checklist.
- `context.md` = per-turn external state.

Each gets its own ordered prompt section and its own size cap. The `WorkspaceContext` mtime-cached reader (`workspace_context.rs:99-124`) keeps re-reads cheap. AGH's existing `.compozy/`-based memory model could borrow this taxonomy.

### 13.5 Subagent prompt minimization

`prompt_builder.rs:75-202` skips ~6 sections when `is_subagent = true` — keeping the spawned agent's context window focused. Specifically: `AGENTS.md`, persona files, `BOOTSTRAP.md`, `HEARTBEAT.md`, channel awareness, peer-agent awareness, sender identity, safety section. AGH should mirror this when sub-agents launch.

### 13.6 Truncation everywhere

Every file load, every memory fragment, every recall candidate has a hard size cap (32 KB for identity files; 500 chars per recall fragment in the prompt section; 4000 chars for canonical compaction summary; 1 MB for daily memory log; 80 KB for compactor input chunks). UTF-8-safe truncation is implemented in `crates/openfang-types/src/lib.rs` (`truncate_str`) and threaded through. **Bounded prompts as a system property, not an afterthought.**

### 13.7 HTTP backend as opt-in overlay

§7.3. The fallback semantics — local SQLite is always authoritative, HTTP is best-effort — is exactly the pattern AGH would need for any future shared-memory or networked-memory feature without making local development require a Postgres/Jina deployment.

---

## 14. What AGH Should NOT Copy

1. **The shared-UUID KV pattern.** This is OpenFang's worst design decision in the memory space — it makes "per-agent KV" a lie at the surface level. AGH's autonomy design assumes per-agent isolation; an opt-in `shared/` namespace is the right answer, not a hidden global default that pretends otherwise.

2. **One-mutex-on-one-connection across all stores.** Acceptable in OpenFang's single-tenant single-process design; will become a serialization bottleneck as soon as multiple agents are concurrently active. AGH already faces SQLite concurrency challenges (see lessons-learned around `BEGIN IMMEDIATE` and migration discipline). Per-store `Arc<Connection>` (or at least separating the high-frequency stores like `memories` and `usage_events` from the high-latency stores like `sessions`) would scale better.

3. **Auto-repair `ALTER TABLE` on every `save_agent`.** Schema drift glued at runtime is the antithesis of the AGH numbered-migration discipline.

4. **Wiki/marketing as spec.** Several features (4 memory tools, traversal API, KG consolidation, memory prune CLI, RDF export) are documented but not implemented. Truthful UI extends to truthful docs.

5. **Hard-coded constants in the consolidation engine.** 7-day decay window, 0.1 confidence floor — operators cannot tune them. Configurable defaults are cheap; baked-in hardcodes are technical debt.

6. **Per-turn full JSONL session rewrite.** Quadratic write cost. Use append-only logs, not full-file overwrites.

7. **`unwrap_or_default()` on recall failures.** Silent recall failure without a structured signal up the stack means observability gaps. AGH should at least log + emit a structured event when memory operations fail.

---

## 15. Open Questions / Things Worth Verifying Live

If AGH wants to validate any of these against a running OpenFang daemon:

1. Does `memory_store("user_name", "Pedro")` from agent A's tool call really make `user_name` visible to agent B's first prompt without B ever having interacted with the user? (Source says yes.)
2. Does `openfang memory list <agent_X>` return the same data as `openfang memory list <agent_Y>`? (Source says yes — they both hit shared UUID.)
3. Does `MemoryFilter { before: Some(<yesterday>), .. }` actually filter? (Source says no — the SQL doesn't honor it.)
4. What happens on schema upgrade when the agent manifest serde struct changes? (Source path: lenient decode → re-serialize → write back. Production behavior may surprise.)
5. Does `consolidation_threshold = 10_000` ever produce visible behavior change? (Source: never read → no.)

These would be quick test cases for an AGH "runtime truth vs. spec" sweep when comparing competitor systems.

---

## 16. Summary of Implications for AGH mem-v2

OpenFang's memory system is **a competent v0 substrate with two real architectural innovations** (file-based per-turn `context.md` + tool-pair-aware compaction) **wrapped in a marketing layer that overstates its semantic capabilities by ~30%**. The actual implementation is:

- One SQLite DB, one mutex, six logical stores, ~2200 lines of memory-crate code total.
- A semantic memory layer that's primarily LIKE-search with optional cosine re-ranking on raw f32 BLOBs.
- A KG that's a flat global triple store with no traversal and no per-agent scoping.
- Cross-channel persistence via a one-row-per-agent canonical session with bounded text-concat compaction *plus* an LLM-backed compactor on top.
- Auto-recall (top-5, embedding-aware-with-fallback) and auto-remember (every assistant turn) — no LLM signal needed for either.
- Identity files as first-class prompt sections, with per-turn `context.md` reload as the external-writer entry point.
- KV tools that pretend to be per-agent but are globally shared — the worst gotcha.

For AGH mem-v2, the design takeaways are:

1. **Keep auto-recall and auto-remember**, but make them observable (hooks) and bound the cost (single embedding per turn, fail-loud on errors).
2. **Adopt the file-based per-turn refresh pattern** for one well-known file (whatever AGH names it — the `context.md` convention is reasonable). Mid-write fallback to last-good is worth the implementation cost.
3. **Two-tier compaction** is correct: cheap text-concat keeps the bound, LLM compactor improves quality when reachable.
4. **Tool-pair split adjustment** is a must-have for any chat-history summarizer.
5. **Per-agent isolation is non-negotiable.** A `shared/` namespace can exist but must be explicit at the tool, CLI, and REST layers.
6. **Numbered migrations only.** Reject any "ensure-column-exists at write time" fallback.
7. **Truthful UI extends to docs.** Don't promise traversal, merging, export, or graph-database semantics until they ship.
8. **Bounded prompts as a system property.** Cap every file load, every recall fragment, every summary — and document the caps.

OpenFang sets the *shape* of a competent agent-runtime memory system. AGH should adopt the shape, fix the gotchas, and ship the features OpenFang only documented.
