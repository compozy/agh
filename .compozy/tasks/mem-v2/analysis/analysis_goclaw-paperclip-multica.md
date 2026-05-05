# goclaw + paperclip + multica Memory & Context Analysis

**Scope:** memory architecture in three additional ACP-compatible harnesses, contrasted with previously-analysed Claude-Code / Codex / Hermes / OpenClaw / OpenFang.
**Sources mined:**
- goclaw — `~/dev/knowledge/goclaw/wiki/concepts/Memory and Knowledge Graph.md`, `~/dev/knowledge/.resources/goclaw/internal/{memory,consolidation,knowledgegraph,store,store/pg,tools,agent,sessions}/`, `~/dev/knowledge/.resources/goclaw/migrations/`.
- paperclip — `~/dev/knowledge/.resources/paperclip/{server/src,packages/db/src/schema,packages/adapter-utils/src,skills/para-memory-files,server/src/onboarding-assets/ceo}` (no topic markdown KB present).
- multica — `~/dev/knowledge/.resources/multica/{server/migrations,server/internal/service,server/internal/handler,server/pkg/agent}` (no topic markdown KB present).
**Date:** 2026-05-04

---

## TL;DR (≈300 words)

Of the three, **only goclaw has an opinionated runtime memory subsystem** — and it is the most ambitious of every harness analysed so far. goclaw ships a **3-tier `v3` memory stack** in PostgreSQL+pgvector: (T1) `memory_documents`/`memory_chunks` for document RAG, (T2) `episodic_summaries` for session summaries with hybrid FTS+HNSW vector search and L0/L1/L2 progressive depth, (T3) `kg_entities`/`kg_relations` for an LLM-extracted, temporal (`valid_from`/`valid_until`), embedding-augmented knowledge graph with Jaro-Winkler dedup. Around it, goclaw runs an **event-driven async `consolidation` pipeline** with four workers — `episodicWorker` (session→summary), `semanticWorker` (summary→KG triples), `dedupWorker` (entity merging), and a genuinely novel **`dreamingWorker`** that periodically debounces, scores unpromoted summaries with a 4-component recall formula (frequency, relevance, recency, freshness), runs an LLM "REM-sleep" synthesis pass and writes consolidated long-term notes back into `memory_documents`. Memory is **agent-callable** via `memory_search` / `memory_get` / `memory_expand` tools (with explicit "L0/L1/L2 depth" parameter), and **auto-injected** at run-start via an L0 abstract block, gated by a trivial-message stop-list and rune-safe context-aware query rewriting.

**paperclip has no DB-resident memory layer at all.** It is a "control plane for AI-agent companies" that *delegates* memory to whichever ACP/CLI adapter the company uses (Claude Code, Codex, Hermes etc.) via session-resume IDs in `agent_runtime_state.session_id` and an `adapter-utils` `SessionCompactionPolicy` that turns *off* paperclip-side rotation for `nativeContextManagement="confirmed"` adapters. Persistent memory is instead a **filesystem PARA convention** (`$AGENT_HOME/life/`, daily notes, `MEMORY.md` tacit-knowledge) recalled with `qmd` (vector+BM25+rerank). Notable: explicit "no deletion, only supersede" + access-count decay.

**multica is a Linear-clone for AI agents.** It also has no real memory layer — it stores `agent_task_queue.session_id` + `chat_session.session_id` so the daemon can `--resume` Claude Code / Codex threads across runs. Workspace context is a single `workspace.context TEXT` and `issue.context_refs JSONB` field, fetched by the agent via the multica CLI rather than embedded in the task payload. Memory continuity is the underlying provider's problem.

---

## 1. goclaw — The "v3 Memory Evolution" stack (HIGH SIGNAL)

### 1.1 Architecture & file paths

goclaw deliberately rebuilt its memory layer as the "v3 Phase 3 — context tiering + consolidation" milestone (the comment is in source: `internal/memory/auto_injector.go:1-3`). The result is a textbook **3-tier memory + L0/L1/L2 progressive disclosure + event-driven consolidation** design.

Key directories:
- `internal/memory/` — embeddings provider, chunker, auto-injector, recall query rewriter, trivial-message filter.
  - `embeddings.go` — `ChunkText`, `OpenAIEmbeddingProvider`, `CosineSimilarity`, `ContentHash` (SHA-256 dedup key).
  - `auto_injector.go` / `auto_injector_impl.go` — L0 prompt-section builder.
  - `recall_query.go` — "Context: …\nQuery: …" rewriter, rune-safe.
  - `trivial_filter.go` — stop-list ("hi", "hello", "ok", "thanks"…) gating.
- `internal/consolidation/` — async pipeline workers and per-agent dreaming config.
  - `workers.go`, `episodic_worker.go`, `semantic_worker.go`, `dedup_worker.go`, `dreaming_worker.go`, `dreaming_config.go`, `scoring.go`, `l0_abstract.go`.
- `internal/knowledgegraph/` — LLM-based entity/relation extraction.
  - `extractor.go`, `extractor_prompt.go`, `similarity.go` (Jaro-Winkler).
- `internal/store/` — interface layer (`memory_store.go`, `episodic_store.go`, `knowledge_graph_store.go`, `knowledge_graph_temporal.go`).
- `internal/store/pg/` — Postgres+pgvector implementations.
- `internal/tools/memory.go`, `memory_expand.go`, `knowledge_graph.go`, `exec_memory_hints.go`, `memory_interceptor.go` — agent-callable tools.
- `internal/agent/loop_context.go` — propagates per-tenant/agent/user/team scopes via `context.Context` (`store.WithSharedMemory`, `store.WithSharedKG`).
- `internal/agent/extractive_memory.go` — regex fallback when LLM compaction fails.
- `internal/sessions/key.go` — canonical `agent:{key}:{channel}:…` session-key scheme.

The CLAUDE.md sums it up at `goclaw/CLAUDE.md:82`:
> **3-tier memory:** Working (conversation) → Episodic (session summaries) → Semantic (KG). Progressive loading L0/L1/L2 with auto-inject for L0.

### 1.2 Persistence backend & schema

**Postgres + pgvector + tsvector**, mostly HNSW-indexed (with one historical IVFFlat that the migration 040 superseded).

**Tier 1 — Document RAG** (`migrations/000001_init_schema.up.sql:163-209`):
```sql
CREATE TABLE memory_documents (id UUID, agent_id UUID, user_id VARCHAR(255), path …, content TEXT, hash VARCHAR(64));
CREATE UNIQUE INDEX idx_memdoc_unique ON memory_documents(agent_id, COALESCE(user_id, ''), path);
CREATE TABLE memory_chunks (id UUID, agent_id UUID, document_id UUID, user_id, path,
    start_line INT, end_line INT, text TEXT, embedding vector(1536),
    tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', text)) STORED);
CREATE INDEX idx_mem_global ON memory_chunks(agent_id) WHERE user_id IS NULL;  -- per-user vs global split
CREATE INDEX idx_mem_tsv ON memory_chunks USING GIN(tsv);
CREATE INDEX idx_mem_vec ON memory_chunks USING hnsw(embedding vector_cosine_ops);
CREATE TABLE embedding_cache (hash, provider, model, embedding vector(1536), …);  -- dedup by (hash, provider, model)
```

Note: the comment at `migrations/000001_init_schema.up.sql:189` calls out that `'simple'` (no stemming) is the right tsvector config for **Vietnamese / multi-language** stores — the codebase is locale-aware throughout (CJK rune-safe truncation appears in multiple places, e.g. `episodic_worker.go:155`, `recall_query.go:53`).

**Tier 2 — Episodic summaries** (migrations 037 → 045):
```sql
-- 000037: core
CREATE TABLE episodic_summaries (
    id UUID, tenant_id UUID, agent_id UUID, user_id VARCHAR(255), session_key TEXT,
    summary TEXT, l0_abstract TEXT, key_topics TEXT[], embedding vector(1536),
    source_type TEXT, source_id TEXT, turn_count INT, token_count INT,
    created_at TIMESTAMPTZ, expires_at TIMESTAMPTZ);
CREATE UNIQUE INDEX idx_episodic_source_dedup ON episodic_summaries(agent_id, user_id, source_id);
CREATE INDEX idx_episodic_tsv ON episodic_summaries USING GIN(to_tsvector('simple', summary));
CREATE INDEX idx_episodic_vec ON episodic_summaries USING hnsw(embedding vector_cosine_ops) WHERE embedding IS NOT NULL;
-- 000040: stored search_vector + HNSW(m=16, ef_construction=64)
ALTER TABLE … ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (to_tsvector('english', coalesce(summary,'') || ' ' || …key_topics…)) STORED;
-- 000041: dreaming-pipeline promotion flag
ALTER TABLE … ADD COLUMN promoted_at TIMESTAMPTZ;
CREATE INDEX idx_episodic_unpromoted ON episodic_summaries(agent_id, user_id, created_at) WHERE promoted_at IS NULL;
-- 000045: per-episode recall signals
ALTER TABLE … ADD COLUMN recall_count INT, recall_score DOUBLE PRECISION, last_recalled_at TIMESTAMPTZ;
CREATE INDEX idx_episodic_recall_unpromoted ON episodic_summaries(agent_id, user_id, recall_score DESC) WHERE promoted_at IS NULL;
```

The `immutable_array_to_string` helper at `migrations/000040_episodic_search_index.up.sql:8-10` is a small but real PG-quirk fix (PG marks `array_to_string` as STABLE; generated columns require IMMUTABLE).

**Tier 3 — Knowledge Graph** (migrations 013 → 037 → 031 → 025):
```sql
-- 000013
CREATE TABLE kg_entities (id UUID, agent_id UUID, user_id, external_id VARCHAR(255), name TEXT,
    entity_type VARCHAR(100), description, properties JSONB, source_id, confidence FLOAT,
    UNIQUE(agent_id, user_id, external_id));
CREATE TABLE kg_relations (id UUID, agent_id UUID, source_entity_id UUID, relation_type, target_entity_id UUID,
    confidence FLOAT, properties JSONB,
    UNIQUE(agent_id, user_id, source_entity_id, relation_type, target_entity_id));
-- 000025: pgvector entity embeddings (dedup + semantic match)
ALTER TABLE kg_entities ADD COLUMN embedding vector(1536);
CREATE INDEX … ON kg_entities USING hnsw(embedding vector_cosine_ops);
-- 000031: tsvector + dedup candidates table
ALTER TABLE kg_entities ADD COLUMN tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', name || ' ' || COALESCE(description, ''))) STORED;
CREATE TABLE kg_dedup_candidates (… entity_a_id, entity_b_id, similarity FLOAT, status …);
-- 000037 (v3): temporal validity windows
ALTER TABLE kg_entities ADD COLUMN valid_from TIMESTAMPTZ, valid_until TIMESTAMPTZ;
CREATE INDEX idx_kg_entities_current ON kg_entities(agent_id, user_id) WHERE valid_until IS NULL;
```

**Side tables of note:** `agent_evolution_metrics` and `agent_evolution_suggestions` (migration 037) — goclaw stores per-tool-call retrieval metrics (`MetricRetrieval` / `metric_key="memory_search"|"auto_inject"`) so a downstream "self-evolution" stage can propose system-prompt adjustments. This is genuinely past anything we've seen elsewhere — Hermes has Curated Memory but doesn't tie a quantified retrieval signal back into a suggestion table.

### 1.3 Memory taxonomy & scoping

Three orthogonal scope dimensions, all enforced via `context.Context` keys (see `internal/agent/loop_context.go:30-115`):
- **Tenant** — every store extracts `store.TenantIDFromContext(ctx)` and joins; missing tenant ⇒ refuse query (security invariant called out in `internal/store/episodic_store.go:53`).
- **Agent + user** — `(agent_id, user_id)` is the dominant index pair on every memory table. `user_id IS NULL` means "global per-agent" memory.
- **Sharing** — flags `WithSharedMemory(ctx)`, `WithSharedKG(ctx)`, `WithSharedSessions(ctx)` switch the search to "no user_id filter" so team agents see each other's memory (`internal/store/pg/memory_search.go:88-99`).

Per-agent JSONB config lives in `agents.kg_dedup_config` and (parsed) `MemoryConfig`; defaults at `internal/memory/auto_injector.go:64-73`:
```go
AutoInjectEnabled:    true
AutoInjectThreshold:  0.3
AutoInjectMaxTokens:  200
EpisodicTTLDays:      90
ConsolidationEnabled: true
```
Plus `KGConfig`: `DedupAutoThreshold=0.98`, `DedupFlagThreshold=0.90`, `ExtractionMinConf=0.75`, `EnableTemporal=true`.

### 1.4 Read/write API

Three concrete read tools agents can call (`internal/tools/memory.go`, `memory_expand.go`, `knowledge_graph.go`):

- `memory_search(query, maxResults?, minScore?, depth?)` — hybrid FTS+vector across **document chunks AND episodic summaries simultaneously**, returning a `tagged` result set with `Tier: "document"|"episodic"`. `depth` parameter (`l0|l1|l2`) only affects episodic. Includes a hint to also call `knowledge_graph_search` if the query implies entities.
- `memory_get(path, from?, lines?)` — line-range read of a memory document (handles per-user-then-global-then-leader-fallback at `tools/memory.go:325-340`).
- `memory_expand(id)` — load full episodic summary by ID. The "L2 retrieval" leg of L0/L1/L2.

Write paths are *not* exposed as agent tools — writes happen automatically via the consolidation pipeline (next section). The only write surface to **document memory** is internal: the dreaming worker calls `memoryStore.PutDocument(_, _, _, "_system/dreaming/<date>-consolidated.md", synthesis)` (`dreaming_worker.go:171-178`). Agents *consume* memory; they do not directly persist text. This is a deliberate choice contrary to Claude Code / Codex (which expose `add_memory`-style tools).

The tool description includes a **language-matching invariant** that's worth quoting (`tools/memory.go:51`):
> "Always query in the SAME language as the stored memory content. If the user speaks Vietnamese, search in Vietnamese. If memory was written in English, search in English. Matching the language dramatically improves search accuracy."

There is also a clever `tools/exec_memory_hints.go` interceptor that *detects shell commands trying to `cat MEMORY.md`* and emits a hint:
```go
"[HINT] Memory files are in the database, not on disk. Use memory_search or memory_get tool instead."
```
— prevents agents from defaulting to filesystem reads when memory is DB-resident. This is a small but **genuinely novel** ergonomic.

### 1.5 Retrieval semantics

**Hybrid FTS + vector with weighted merge + per-user "boost"** (`internal/store/pg/memory_search.go:215-275`):
1. Run tsvector `ts_rank()` query with `plainto_tsquery('simple', $query)` (FTS leg, fast).
2. Run cosine-distance query against `embedding <=> $query_vec` (vector leg, semantic).
3. Merge per `(path, start_line)` key. Default weights: `TextWeight≈0.7`, `VectorWeight≈0.3`. Per-user chunks (`user_id IS NOT NULL`) get a **1.2× score boost** so personal context beats global.
4. Apply `MinScore` cutoff and `PathPrefix` filter.

**Episodic search** is the same shape but on `episodic_summaries`, with the FTS leg using the stored `search_vector` column generated from `summary || ' ' || array_to_string(key_topics, ' ')` (`migrations/000040`).

**Auto-inject** (`internal/memory/auto_injector_impl.go:27-103`) is a separate, lighter-weight FTS-biased search that:
- Filters trivial messages first (`isTrivialMessage` — < 3 meaningful tokens after stop-list removal, `trivial_filter.go`).
- Builds a **context-aware query** by prepending up to 400 runes of recent conversation as `Context: …\nQuery: …` (`recall_query.go:28-47`). Rune-safe to handle Vietnamese / CJK.
- Defaults: `MaxEntries=5`, `Threshold=0.3`, `VectorWeight=0.3`, `TextWeight=0.7`.
- Output: `## Memory Context\n\nRelevant memories from past sessions (use memory_search for details):\n- <l0_abstract>\n…`
- Records `MetricRetrieval` rows in a 5s background goroutine, never blocking the hot path.

### 1.6 Auto-memory triggers — the consolidation pipeline

This is goclaw's most distinctive piece. `internal/consolidation/workers.go:36-93` wires four async workers to a domain event bus:

```
session.completed   → episodicWorker  → episodic.created
episodic.created    → semanticWorker  → entity.upserted
entity.upserted     → dedupWorker     → (terminal)
episodic.created    → dreamingWorker  → (terminal: writes _system/dreaming/<date>.md)
periodic 6h ticker → episodicStore.PruneExpired
```

**`episodicWorker`** (`episodic_worker.go`):
- Idempotency: builds `source_id = "{session_key}:{compaction_count}"` and calls `ExistsBySourceID` before insert.
- Source: prefers `payload.Summary` from compaction; falls back to LLM summarisation reading actual session messages with a UTF-8-safe rune slice at 500 chars per message (`episodic_worker.go:155`).
- Builds `l0_abstract` extractively (no LLM!) — first sentence ≥ 20 runes, capped at 200 (`l0_abstract.go:10-27`).
- Extracts `key_topics` extractively too — capitalized multi-word phrases (`l0_abstract.go:51-80`).
- TTL: `expires_at = now + 90 days` (matches `MemoryConfig.EpisodicTTLDays`).
- Publishes `episodic.created`.

**`semanticWorker`** (`semantic_worker.go`):
- Calls `Extractor.Extract()` (LLM with strict JSON-schema prompt, `internal/knowledgegraph/extractor_prompt.go`).
- The prompt is **strongly typed**: 10 entity types (`person|organization|project|product|technology|task|event|document|concept|location`) and a fixed relation vocabulary (`works_on, manages, reports_to, collaborates_with, …, related_to (LAST RESORT)`). Forces `external_id` to be a stable lowercase-hyphen slug so the same entity gets the same ID across extractions.
- Filters `confidence < minConfidence` (default 0.75); JSON-sanitiser fixes "0. 85" → "0.85" and trailing commas (`extractor.go:150-207`) — defensive against weak LLMs.
- For chunked extractions, dedup-merges by `external_id` keeping max confidence (`extractor.go:238-273`).
- Sets `valid_from = NOW()` on every entity/relation (temporal invariant).
- Publishes `entity.upserted`.

**`dedupWorker`** (`dedup_worker.go`):
- Calls `kgStore.DedupAfterExtraction(agentID, userID, newEntityIDs)` — auto-merge at >0.98 + name match (Jaro-Winkler, `knowledgegraph/similarity.go`), flag candidates above 0.90 for review.
- Bulk re-pointing of relations on merge happens via `MergeEntities`.

**`dreamingWorker`** ⭐ (`dreaming_worker.go`) — **the genuinely novel piece**:
- Subscribes to `episodic.created`. Per `(agent_id, user_id)` pair:
  - **Debounce** via `sync.Map` keyed `agent:user → time.Time`, default 10 minutes.
  - Skip if `CountUnpromoted < threshold` (default 5).
  - Fetch top `dreamingFetchLimit=10` unpromoted entries via `ListUnpromotedScored` (ORDER BY recall_score DESC, then created_at ASC).
  - **Filter by `recallThresholds` BEFORE LLM** (`scoring.go:109-128`): never-recalled entries pass (so cold-start agents synthesise their first sessions); recalled entries must clear both `MinRecallCount=2` and `MinScore=0.2`.
  - Format each entry with recall metadata (`[recalled %dx, last: Jan 2] %s`) so the LLM can weight frequently-recalled memories higher (`dreaming_worker.go:53-62`).
  - LLM synthesis call with hardcoded prompt asking for **User Preferences / Project Facts / Recurring Patterns / Key Decisions** sections (`dreaming_worker.go:225-235`).
  - Writes result to `_system/dreaming/YYYYMMDD-consolidated.md` in `memory_documents`, indexes it (creates chunks + embeddings), `MarkPromoted` on the source IDs.
  - Stores debounce cursor even when filter empties out the candidate set (with explicit comment about avoiding hot-loop re-evaluation, `dreaming_worker.go:148-152`).

**Recall scoring** (`scoring.go:47-87`) is the gem inside the gem:
```
score = 0.30 * frequency + 0.35 * relevance + 0.20 * recency + 0.15 * freshness
  frequency  = log1p(recall_count) / log1p(10)            # capped
  relevance  = clamp(recall_score, 0, 1)                  # running avg of search hit scores
  recency    = exp(-ln2/14d * (now - last_recalled_at))   # half-life 14d
  freshness  = exp(-ln2/14d * (now - created_at))         # cold-start protection
```

The freshness weight is explicitly there to keep brand-new agents alive: a fresh, never-searched memory still scores ~0.15 instead of zero, so the synthesis pipeline doesn't starve.

`memory_search` calls `episodicStore.RecordRecall(id, score)` in a fire-and-forget goroutine on every hit (`tools/memory.go:194-222`), feeding the running average into `recall_score`. **This closes the loop**: searching memory affects which memories get consolidated next time the agent "dreams".

### 1.7 Compaction / summarisation

Two layers:
1. **Per-session compaction** — handled by the standard agent loop (8-stage pipeline `context→history→prompt→think→act→observe→memory→summarize`). The compaction summary is what triggers `session.completed` events; if compaction returns nothing, `episodicWorker.summarizeSession` does its own LLM pass (`episodic_worker.go:131-183`) with a 30s timeout, max_tokens=1024, temperature=0.3.
2. **Cross-session "dreaming"** — the dreamingWorker described above, which is goclaw's analogue of Hermes's Curated Memory (but better-quantified and async-debounced).

Notable safety net: `internal/agent/extractive_memory.go` is a regex-based fallback that runs when LLM compaction returns NO_REPLY or empty output. It scrapes:
- Decisions: `(?i)decided to|let'?s go with|approved|agreed on|chose|we'?ll use`
- Preferences: `(?i)I prefer|don'?t do|always|never|I want|please remember`
- Tech facts, URLs, file paths, ISO dates.

This output gets injected as `## Extracted Context (auto-saved before compaction)` so context is *never* lost silently. None of the other harnesses we've analysed have this regex safety net.

### 1.8 Lifecycle (hydration, persistence, GC)

- **Hydration:** `injectContext` (`agent/loop_context.go`) propagates tenant/agent/user/team/sharing flags into ctx before any tool runs. Per-user workspace is `MkdirAll`-ed lazily (`loop_context.go:170-175`).
- **Persistence:** writes happen **after** the run via the event bus, never synchronously. The auto-inject metric record is also goroutine'd with 5s timeout.
- **GC:** `EpisodicStore.PruneExpired` runs every 6h via a `time.NewTicker(6*time.Hour)` in `consolidation.Register` (`workers.go:73-90`). KG has no GC; instead it has temporal supersession (`SupersedeEntity` sets `valid_until = NOW()`).
- **Embedding cache:** `embedding_cache(hash, provider, model)` table (init schema) acts as a content-addressed cache so re-uploaded documents don't pay the embedding cost twice.

### 1.9 Failure modes / TODOs surfaced in code

- **`hybridMerge` per-user boost still applies in shared mode** — explicit TODO at `pg/memory_search.go:213`: "when shared memory is active, the 1.2x personal boost still applies — consider removing it in shared mode if all docs should be treated equally."
- **JSON parse failures** in KG extraction are non-fatal; the worker logs `slog.Warn` and continues (`semantic_worker.go:43-46`). LLM hallucination is bounded only by `minConfidence` filter.
- **Truncation handling** — extractor retries once with shorter input on `finish_reason="length"` and gives up after that (`extractor.go:81-95`).
- **Recall thresholds tuned conservatively low** because goclaw boots with zero recall data (comment at `scoring.go:30-34`). Once production data accumulates, the constants will need re-tuning — the comment flags this explicitly.
- **Dreaming loop hot-skip risk** — fixed by stamping `lastRun` even on empty-filter skips (P10.1 review note inlined at `dreaming_worker.go:148-152`).
- **No tenant_id on KG tables** in the original 000013 migration — only `agent_id` + `user_id`. Tenant isolation rides on `agents.id ON DELETE CASCADE` instead. The dedup_candidates table did get an explicit `tenant_id` in 000031.

### 1.10 What's genuinely novel vs Hermes / Codex / OpenClaw / Claude-Code

| Feature | goclaw | Others |
|---|---|---|
| **3-tier (doc + episodic + KG) in one query path** | yes — `memory_search` returns `tagged` results across tiers | Codex/Claude-Code do per-source side-queries; Hermes has Curated Memory but no episodic+KG combo |
| **`dreamingWorker` REM-sleep consolidation with quantified recall scoring** | yes — 4-component formula with cold-start freshness term | Hermes has nudges; nothing has this scoring depth |
| **Recall-feedback loop** (search hits raise next-cycle synthesis priority) | yes — `RecordRecall` + `recall_score` running average | none |
| **Temporal KG with `valid_from`/`valid_until` + `SupersedeEntity`** | yes | OpenFang has supersession in RDF; goclaw is the only relational version |
| **Strongly-typed extraction prompt** (10 fixed entity types, fixed relation vocab) | yes | OpenFang's RDF extractor is more freeform |
| **Trivial-message filter for auto-inject** | yes — explicit stop-list | Claude-Code uses model judgement |
| **Context-aware recall query** (rune-safe context+query concat) | yes | nothing else does this |
| **Shell→memory tool hint interceptor** (`MaybeMemoryExecHint`) | yes | none |
| **Regex-based extractive fallback** when LLM compaction fails | yes | none |
| **Per-user 1.2× score boost** with deliberate "user copy wins" tiebreak | yes | none |
| **Agents are read-only consumers of memory** (no `add_memory` tool) | yes | Claude-Code / Codex expose write tools |
| **Embedding cache by content hash + (provider, model)** | yes | nothing — all others re-embed on every upload |

### 1.11 Notable good / bad

**Good:**
- The whole pipeline is event-driven and uses a worker pool with dedup+retry. Nothing blocks the agent loop.
- Memory tools are scoped via `context.Context` keys (`AgentID`, `TenantID`, `UserID`, `SharedMemory`, `LeaderAgentID`) consistently — the security model survives even with team delegation and leader fallback.
- Migrations are append-only and explicit. `idx_episodic_recall_unpromoted` is a partial index on `recall_score DESC WHERE promoted_at IS NULL` that exactly matches the dreaming worker's primary query shape.
- The L0 abstract is **extractive** (no LLM) so the cheapest tier of recall has zero hot-path LLM cost.
- `extractor.sanitizeJSON` is a state-machine that only fixes JSON outside string literals — defensive against the "model emits 0. 85" failure mode without breaking valid content.

**Bad / risky:**
- The dreaming pipeline is a lot of moving parts; a single LLM call failure during synthesis is logged as `slog.Warn` and the entries stay unpromoted forever unless something else triggers another `episodic.created` event — there's no retry queue, no DLQ.
- `kg_entities.UNIQUE(agent_id, user_id, external_id)` makes the LLM's `external_id` slugger load-bearing for correctness. If two extractions disagree on slug ("john-doe" vs "jdoe"), you get duplicate entities until the dedup worker runs Jaro-Winkler.
- Recall scoring constants are hand-tuned and the comments admit it. No A/B framework to validate.
- KG extraction prompt's 10 entity types are domain-tilted (project/product/technology heavy). Generic personal-life facts ("my dog is named Rex") will end up as `concept` or get dropped.
- pgvector + HNSW(`m=16, ef_construction=64`) is fine for ~1M vectors per tenant but recall vs latency tuning is silent at the schema level.

---

## 2. paperclip — control plane, no DB memory (LOW-MED SIGNAL)

### 2.1 What paperclip *is*

From `AGENTS.md:7-8`:
> Paperclip is a control plane for AI-agent companies.

It is **not** an ACP harness in the goclaw / OpenClaw / Hermes sense. It is a Drizzle-on-Postgres web app with:
- `companies` / `agents` / `issues` / `goals` / `documents` tables (Linear-meets-Asana for AI agent orgs).
- `agent_runtime_state` / `agent_task_sessions` / `heartbeat_runs` for tracking external agent processes.
- `packages/adapters/` containing wrappers for `claude-local`, `codex-local`, `cursor-local`, `gemini-local`, `opencode-local`, `pi-local`, `acpx-local`, plus an external Hermes adapter on the `feat/externalize-hermes-adapter` branch.

Memory is **owned by the underlying adapter**, not by paperclip.

### 2.2 What paperclip stores about the agent's memory

There are **no** `memory_*` / `episodic_*` / `kg_*` tables. The only memory-adjacent state lives on `agent_runtime_state` (`packages/db/src/schema/agent_runtime_state.ts`):
```ts
agentRuntimeState = pgTable("agent_runtime_state", {
  agentId: uuid().primaryKey(),
  companyId, adapterType,
  sessionId: text(),                    // ← THE memory pointer
  stateJson: jsonb().$type<Record<string, unknown>>().default({}),
  totalInputTokens, totalOutputTokens, totalCachedInputTokens, totalCostCents,
  …
});
```
Plus `agentTaskSessions` (`agent_task_sessions.ts`):
```ts
{ companyId, agentId, adapterType, taskKey,
  sessionParamsJson: jsonb(),           // adapter-specific resume hints
  sessionDisplayId: text(),
  lastRunId, lastError, … }
// UNIQUE(companyId, agentId, adapterType, taskKey)
```

Each `heartbeatRuns` row has `sessionIdBefore`/`sessionIdAfter` columns (`heartbeat_runs.ts:23-24`) — paperclip captures *what session ID the adapter used before and after this run*, but never tries to interpret it. `heartbeatRunEvents` is a JSONB event log (`bigserial id, run_id, seq, event_type, payload`). No vector search, no embeddings, no FTS.

### 2.3 Session compaction policy — the only memory logic in core

`packages/adapter-utils/src/session-compaction.ts` has the full memory policy paperclip cares about. It's a small but interesting piece:

```ts
const DEFAULT_SESSION_COMPACTION_POLICY = {
  enabled: true,
  maxSessionRuns: 200,
  maxRawInputTokens: 2_000_000,
  maxSessionAgeHours: 72,
};
const ADAPTER_MANAGED_SESSION_POLICY = {
  enabled: true,
  maxSessionRuns: 0,      // sentinel: disabled
  maxRawInputTokens: 0,
  maxSessionAgeHours: 0,
};
const ADAPTER_SESSION_MANAGEMENT = {
  acpx_local:    { supportsSessionResume: true, nativeContextManagement: "confirmed", defaultSessionCompaction: ADAPTER_MANAGED_SESSION_POLICY },
  claude_local:  { …, "confirmed", ADAPTER_MANAGED_SESSION_POLICY },
  codex_local:   { …, "confirmed", ADAPTER_MANAGED_SESSION_POLICY },
  hermes_local:  { …, "confirmed", ADAPTER_MANAGED_SESSION_POLICY },
  cursor:        { …, "unknown",   DEFAULT_SESSION_COMPACTION_POLICY },
  gemini_local:  { …, "unknown",   DEFAULT_SESSION_COMPACTION_POLICY },
  opencode_local:{ …, "unknown",   DEFAULT_SESSION_COMPACTION_POLICY },
  pi_local:      { …, "unknown",   DEFAULT_SESSION_COMPACTION_POLICY },
};
```

Translation: paperclip will rotate sessions itself for adapters whose native compaction is "unknown", and **defer entirely** to adapters with `nativeContextManagement: "confirmed"` (Claude, Codex, Hermes, ACPX). The `LEGACY_SESSIONED_ADAPTER_TYPES` set keeps a fallback for adapters that pre-date the table.

This is the cleanest articulation of "**don't fight the underlying agent's memory layer**" we've seen. The novel idea here is a tri-state enum (`confirmed | likely | unknown | none`) describing how much paperclip should trust the adapter's own compaction. AGH could borrow this directly.

### 2.4 The actual memory model — filesystem PARA

Memory in paperclip's onboarding-shipped CEO agent is not in the DB at all. It lives on disk under `$AGENT_HOME/`, organised by Tiago Forte's PARA method (`skills/para-memory-files/SKILL.md`). Three layers:

**Layer 1: Knowledge Graph as a folder tree**
```
$AGENT_HOME/life/
  projects/<name>/{summary.md, items.yaml}
  areas/people/<name>/, areas/companies/<name>/
  resources/<topic>/
  archives/
  index.md
```
Each entity has two tiers — `summary.md` (load first) and `items.yaml` (atomic facts, on demand).

**Atomic fact schema** (`skills/para-memory-files/references/schemas.md`):
```yaml
- id: entity-001
  fact: "The actual fact"
  category: relationship | milestone | status | preference
  timestamp: "YYYY-MM-DD"
  source: "YYYY-MM-DD"
  status: active            # active | superseded
  superseded_by: null       # e.g. entity-002
  related_entities: [companies/acme, people/jeff]
  last_accessed: "YYYY-MM-DD"
  access_count: 0
```

**Memory decay rules** are lifecycle policy on top of yaml:
- **Hot** (last 7 days): include in `summary.md`.
- **Warm** (8-30d): include at lower priority.
- **Cold** (30+d or never accessed): drop from `summary.md`, keep in `items.yaml`.
- High `access_count` resists decay.
- **No deletion** — only `status: superseded` + `superseded_by`.

**Layer 2: Daily notes** — `$AGENT_HOME/memory/YYYY-MM-DD.md` raw timeline.
**Layer 3: Tacit knowledge** — `$AGENT_HOME/MEMORY.md` ("how the user operates", not facts about the world).

**Recall** is via the `qmd` CLI — `qmd query "what happened at Christmas"` (semantic+rerank), `qmd search` (BM25), `qmd vsearch` (pure vector). `qmd index $AGENT_HOME` indexes the personal folder.

### 2.5 Heartbeat extraction loop

`server/src/onboarding-assets/ceo/HEARTBEAT.md:57-62`:
> ## 7. Fact Extraction
> 1. Check for new conversations since last extraction.
> 2. Extract durable facts to the relevant entity in `$AGENT_HOME/life/` (PARA).
> 3. Update `$AGENT_HOME/memory/YYYY-MM-DD.md` with timeline entries.
> 4. Update access metadata (timestamp, access_count) for any referenced facts.

So **the agent itself** runs the consolidation pass during its heartbeat — there is no goclaw-style async worker. Paperclip's responsibility ends at scheduling heartbeats and capturing `sessionIdBefore`/`sessionIdAfter`.

### 2.6 Genuinely novel vs the field

- **`nativeContextManagement` tri-state** as a per-adapter capability flag is the cleanest way I've seen anyone reason about "should the harness compact, or should it trust the adapter?". AGH should adopt this verbatim.
- **PARA folder structure** is more opinionated than the loose "memory/*.md" patterns we've seen. Whether it's better or just different is a values question — but the three-layer decomposition (KG / daily / tacit) maps cleanly onto goclaw's tiers without a DB.
- **No-deletion + supersession + access-count decay + Hot/Warm/Cold tiering** is a complete lifecycle policy that doesn't require any specialised storage.
- **`qmd` as a recall layer** is a reusable CLI primitive — paperclip ships zero memory infra and still gets vector+BM25+rerank by using a generic tool.

### 2.7 Notable good / bad

**Good:**
- The control-plane / runtime separation is *very* clean. Paperclip can adopt any new ACP adapter without DB schema changes.
- The session-compaction policy resolution (`resolveSessionCompactionPolicy`) is deterministic, has a precedence (`agent_override > adapter_default > legacy_fallback`), and degrades safely.
- `agentTaskSessions` keys on `(companyId, agentId, adapterType, taskKey)` so the same agent on different tasks gets *different* sessions — task-level memory isolation by construction.

**Bad / risky:**
- **No queries on the actual memory** — paperclip can't aggregate "what does this agent know about our customers?" without running a recall through the agent itself. The Linear-style audit log (`activity_log`, `heartbeat_run_events`) doesn't substitute for memory introspection.
- The PARA filesystem layer is *not enforced* — it's a skill prompt. An agent that ignores the heartbeat checklist accumulates no durable memory.
- `agentRuntimeState.stateJson: jsonb().default({})` is a classic "anything goes" escape valve. No schema validation, no migration story.
- Filesystem isolation across multiple companies / agents on one machine is implicit (`$AGENT_HOME` is whatever the operator sets). No tenant-level enforcement.

---

## 3. multica — Linear for AI agents, no memory (LOW SIGNAL)

### 3.1 What multica *is*

From `CLAUDE.md:5-7`:
> Multica is an AI-native task management platform — like Linear, but with AI agents as first-class citizens.

Go backend (`server/`, Chi + sqlc + pgvector/pg17) plus a Next.js + Electron frontend. **`assignee_type` is `'member' | 'agent'`** on every issue (`migrations/001_init.up.sql:61-62`), and the `agent_task_queue` is what dispatches work to local-daemon or cloud agents.

The CI workflow uses `pgvector/pgvector:pg17` (`CLAUDE.md:114`) but **no schema actually uses pgvector** in the multica server itself. The pgvector dependency is the multica-org standard image; the multica memory model itself is non-vector.

### 3.2 What multica stores about memory

Three columns total across the entire schema:
- `issue.context_refs JSONB DEFAULT '[]'` (`migrations/001_init.up.sql:67`) — list of issue/document references the assignee should pull. Pure list-of-pointers, no content.
- `workspace.context TEXT` (`migrations/006_workspace_context.up.sql:1`) — single workspace-level prose blob. The CLI exposes it via `multica workspace get --output json` (per `internal/daemon/execenv/runtime_config.go:113`).
- `agent_task_queue.context JSONB` (`migrations/003_task_context.up.sql:1-3`) — *added then mostly unused*. The comment at `internal/service/task.go:102-103` reads:
> "No context snapshot is stored — the agent fetches all data it needs at runtime via the multica CLI."

The `task.context` column is currently used **only** for "quick-create" tasks (`QuickCreateContextType="quick_create"`) where the agent receives a structured payload describing what kind of issue to create. Regular issue tasks leave `context` empty.

### 3.3 The actual memory layer — provider session-resume

**Migration 020** (`session_id` on `agent_task_queue`):
```sql
ALTER TABLE agent_task_queue ADD COLUMN session_id TEXT;
ALTER TABLE agent_task_queue ADD COLUMN work_dir TEXT;
-- Comment: "These enable resuming the same Claude Code session across multiple
-- tasks for the same (agent, issue) pair via --resume <session_id>."
```

**Migration 033** (`chat_session` for the agent-chat feature):
```sql
CREATE TABLE chat_session (
    id UUID PRIMARY KEY,
    workspace_id, agent_id, creator_id, title TEXT,
    session_id TEXT,            -- adapter session
    work_dir TEXT,
    status TEXT CHECK (status IN ('active', 'archived')),
    …);
ALTER TABLE agent_task_queue ADD COLUMN chat_session_id UUID REFERENCES chat_session(id);
```

When a task fails (`internal/service/task.go:746-760`):
> "sessionID/workDir are optional: when the agent established a real session before failing (e.g. crashed mid-conversation, was cancelled, or hit a tool error), the daemon should pass them so we can preserve the resume pointer on both the task row and the chat_session — otherwise the next chat turn would silently start a brand-new session and lose memory."

So multica's memory model is:
1. Issue-scoped: `(agent_id, issue_id) → session_id` enables resume.
2. Chat-scoped: `chat_session.session_id` enables resume across chat turns.
3. Workspace-scoped: a single `workspace.context` text blob.
4. Issue-scoped: `issue.context_refs` (list of pointers).

That's the entire memory layer.

### 3.4 Codex thread resume code (the cleanest example)

`server/pkg/agent/codex.go:341-384` implements `startOrResumeThread` — multica's contract is "give the adapter the prior `threadId`; if `thread/resume` fails for any reason (thread GC'd, schema drift, transport error), fall back to `thread/start` so the task still makes progress":
```go
if priorThreadID := opts.ResumeSessionID; priorThreadID != "" {
    resumeResult, err := c.request(ctx, "thread/resume", …)
    if err == nil { /* extract threadID, return resumed=true */ }
    logger.Warn("codex thread/resume failed; falling back to thread/start", …)
}
startResult, err := c.request(ctx, "thread/start", …)
```

The same pattern applies for every adapter (`server/pkg/agent/{codex,hermes,gemini,pi,copilot,kiro,…}.go`). multica's memory layer ends here.

### 3.5 Notable good / bad

**Good:**
- Brutal simplicity. No memory means no migration churn, no consolidation bugs, no dedup races.
- `forceFreshSession` flag on `CreateAgentTaskParams` (`internal/service/task.go:144`) gives the user an explicit "ignore the prior session" lever after a bad run — the alternative is a fork-or-resume guess.
- `chat_session.session_id` + `chat_session.runtime_id` move together (`internal/service/task.go:766-779`) so the next claim can apply a runtime-guard. No accidental cross-runtime resumes.
- Issue `assignee_type` is a tagged-union on `('member', 'agent')` — agents are first-class everywhere.

**Bad / risky:**
- Memory is entirely *the underlying provider's* problem. If Claude Code's session GCs or Codex's thread index drops, multica has no recovery.
- `workspace.context TEXT` is a single shared blob — no per-agent override, no scoping, no versioning.
- `issue.context_refs` is `JSONB DEFAULT '[]'` with no validation. The CLI fetches whatever pointers it gets.
- The `agent_task_queue.context` column was added in 003 then turned out to be only useful for quick-create tasks — minor schema sprawl.
- pgvector is loaded into the CI image but never used. Either dead weight or a placeholder for a future memory layer.

### 3.6 Genuinely novel vs the field

Honestly, very little. Multica is included in the comparison set because it's an ACP-adjacent runtime, but its memory thesis is "**the harness is the issue tracker; the agent owns its own memory via its provider's session-resume**". The only mildly novel thing is the dual-pointer pattern (`session_id` + `work_dir` recorded together) and the `forceFreshSession` user lever.

---

## 4. Cross-harness comparison (memory dimensions)

| Dimension | goclaw | paperclip | multica |
|---|---|---|---|
| **Has DB memory** | yes (3 tiers, pgvector+tsvector+KG) | no | no |
| **Memory tiers** | Working (live ctx) / Episodic / Semantic KG / "Dreaming" docs | filesystem PARA: KG folders / daily notes / tacit MD | none — only `session_id` + `workspace.context` + `issue.context_refs` |
| **Auto-inject at run-start** | yes — L0 abstracts, trivial-message gated, context-aware query | n/a — agent runs in own process | n/a |
| **Cross-session consolidation** | event-driven async workers (`dreamingWorker`) with quantified recall scoring | agent's own heartbeat runs `qmd index` + extraction | none |
| **Scoping** | tenant + agent + user + team + sharing flags via ctx keys | `$AGENT_HOME` filesystem isolation | workspace + agent + issue |
| **Dedup** | Jaro-Winkler on names, embedding cosine ≥0.98 auto-merge, ≥0.90 flag | no — supersession only | none |
| **TTL / GC** | `expires_at`, `PruneExpired` every 6h, KG temporal supersession | "no deletion, only supersede" + access-count decay | none |
| **Compaction** | provider-side + extractive regex fallback | per-adapter `nativeContextManagement` tri-state | per-adapter |
| **Embedding cache** | `embedding_cache(hash, provider, model)` | n/a | n/a |
| **Retrieval feedback loop** | yes — `RecordRecall` updates running avg → next dreaming cycle | access_count + last_accessed in yaml | none |
| **Strongly-typed extraction** | yes — 10 entity types, fixed relation vocab | no — yaml is loose | n/a |
| **Agent-callable tools** | `memory_search`, `memory_get`, `memory_expand`, `knowledge_graph_search` | none in core; agent uses `qmd` directly | none |
| **Localisation** | rune-safe everywhere (Vietnamese / CJK), `'simple'` tsvector | n/a | n/a |
| **Genuinely novel** | dreamingWorker recall scoring, regex extractive fallback, exec→memory hint, context-aware query | nativeContextManagement enum, PARA + access-count decay | dual session_id + work_dir pointer, `forceFreshSession` |

---

## 5. Recommendations for AGH mem-v2 (synthesised)

Drawn directly from this analysis only — to be reconciled with `analysis_ai-harness.md` / `analysis_ai-memory.md` / `analysis_codex.md` / `analysis_hermes.md` / `analysis_openclaw.md` upstream.

1. **Adopt goclaw's 3-tier shape (document RAG + episodic summaries + KG triples) but keep the storage backend pluggable.** AGH already uses SQLite for the daemon; pgvector is not a viable hard dep. SQLite + `sqlite-vec` or the FTS5 + `vec0` extensions cover the same use-cases with file-local persistence. The L0/L1/L2 progressive-disclosure interface (cheap → mid → full) is independent of the backend.
2. **Adopt the L0 abstract pattern (extractive, no LLM)** for cheap auto-inject. Defaults that worked for goclaw: max 5 entries, 200-token budget, 0.3 relevance threshold.
3. **Adopt the `dreamingWorker` design as the core consolidation motif** — debounced per `(agent, user)`, gated by a count threshold, scored on a 4-component formula (frequency / relevance / recency / freshness), with cold-start protection via the freshness term. The recall-feedback loop (`RecordRecall` on every search hit) is what closes the system.
4. **Borrow paperclip's `nativeContextManagement` tri-state** as the per-extension-type capability flag. AGH's extension manifest should declare `native_context_management: "confirmed" | "likely" | "unknown" | "none"`. When `confirmed`, AGH leaves session compaction to the underlying agent. When `unknown`, AGH applies a default rotation policy on top.
5. **Borrow goclaw's "memory is read-only to the agent"** discipline (or at least make writes structured/gated). The exec→memory hint interceptor is a small thing that pays off.
6. **Borrow goclaw's regex extractive fallback** as the safety net when LLM compaction returns empty/NO_REPLY. This is one of those features the field clearly should have but only goclaw shipped.
7. **Ditch goclaw's strongly-typed entity vocabulary** for AGH's first cut. The 10-type enum is too domain-specific. Start freeform with `entity_type` as a string, add a controlled vocabulary later if dedup analytics show fragmentation.
8. **Keep multica's "session_id resume" pattern as the *fallback* memory path** — when extensions opt out of AGH's mem-v2, the kernel still preserves continuity by handing back the prior session ID, and we never accidentally start fresh on the user.
9. **Borrow paperclip's no-deletion + supersession discipline** for the agent-managed write path. `status: active|superseded` + `superseded_by` is universally cheaper than building a delete-undo system later.
10. **Treat localisation as a first-class invariant.** goclaw's rune-safe truncation everywhere is one of the cheapest robustness wins available; doing it from day 1 is much easier than retrofitting.

---

## 6. Citations (paths only, anchors above)

- `~/dev/knowledge/goclaw/wiki/concepts/Memory and Knowledge Graph.md`
- `~/dev/knowledge/.resources/goclaw/internal/memory/{embeddings.go, auto_injector.go, auto_injector_impl.go, recall_query.go, trivial_filter.go}`
- `~/dev/knowledge/.resources/goclaw/internal/consolidation/{workers.go, episodic_worker.go, semantic_worker.go, dedup_worker.go, dreaming_worker.go, dreaming_config.go, scoring.go, l0_abstract.go, interfaces.go}`
- `~/dev/knowledge/.resources/goclaw/internal/knowledgegraph/{extractor.go, extractor_prompt.go, similarity.go}`
- `~/dev/knowledge/.resources/goclaw/internal/store/{episodic_store.go, memory_store.go, knowledge_graph_store.go, knowledge_graph_temporal.go}`
- `~/dev/knowledge/.resources/goclaw/internal/store/pg/{episodic_search.go, memory_search.go}`
- `~/dev/knowledge/.resources/goclaw/internal/agent/{loop_context.go, extractive_memory.go}`
- `~/dev/knowledge/.resources/goclaw/internal/sessions/key.go`
- `~/dev/knowledge/.resources/goclaw/internal/tools/{memory.go, memory_expand.go, exec_memory_hints.go}`
- `~/dev/knowledge/.resources/goclaw/migrations/{000001, 000013, 000025, 000031, 000037, 000039, 000040, 000041, 000045}_*.up.sql`
- `~/dev/knowledge/.resources/paperclip/AGENTS.md`
- `~/dev/knowledge/.resources/paperclip/packages/adapter-utils/src/session-compaction.ts`
- `~/dev/knowledge/.resources/paperclip/packages/db/src/schema/{agents.ts, agent_runtime_state.ts, agent_task_sessions.ts, heartbeat_runs.ts, heartbeat_run_events.ts, documents.ts}`
- `~/dev/knowledge/.resources/paperclip/skills/para-memory-files/{SKILL.md, references/schemas.md}`
- `~/dev/knowledge/.resources/paperclip/server/src/onboarding-assets/ceo/{AGENTS.md, HEARTBEAT.md}`
- `~/dev/knowledge/.resources/multica/{AGENTS.md, CLAUDE.md}`
- `~/dev/knowledge/.resources/multica/server/migrations/{001_init, 003_task_context, 004_agent_runtime_loop, 006_workspace_context, 020_task_session, 033_chat}*.up.sql`
- `~/dev/knowledge/.resources/multica/server/internal/service/task.go`
- `~/dev/knowledge/.resources/multica/server/pkg/agent/codex.go`
