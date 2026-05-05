# Analysis: AI Memory Architectures for the AGH Memory System Redesign

**Source:** `~/dev/knowledge/ai-memory/` (15 wiki concepts, 138 articles, 13 framework READMEs).
**Scope:** Memory taxonomies, persistence, retrieval, write-side design, forgetting, scoping, trust/provenance, failure modes, concrete competitors, API/SDK patterns, integration with hooks/tools/skills/RAG/CLI.
**Posture:** Read-only research. No code or external file modified.

---

## TL;DR (≈200 words)

The 2024–2026 literature converges on a small set of load-bearing claims: (1) memory is a **layered system**, not a vector store — short-term/working, episodic, semantic, and procedural memory each have different lifecycles, retrieval primitives, and update rules; (2) **maintenance, not retrieval, is the hard part** — production systems require explicit `ADD/UPDATE/DELETE/NOOP` operations and a memory **controller** that gates writes (Mem0 productized this; A‑MEM, MemoryOS, ReMe extend it); (3) **hybrid retrieval beats top-k vector search** — production stacks fuse BM25 + dense + graph traversal with reranking and metadata filters in a multi-stage pipeline (candidate → expand → prioritize → package → inject); (4) **persistence is tiered** — hot in-context, warm Redis-class session state, cold Postgres/pgvector or vector DB, archival blob, with explicit promotion/demotion and TTLs; (5) **scoping is four-axis** — `user_id`, `agent_id`, `session_id/run_id`, `app_id/org_id` with namespace+RLS isolation; (6) **temporal validity** (Zep/Graphiti bi-temporal model) and **provenance** (Hindsight's Opinion Network, episode→fact links) are emerging as first-class; (7) **file-first markdown memory with lifecycle hooks** (OpenClaw, Letta MemFS, Claude Code) is a credible production pattern when paired with a pre-compaction flush; (8) **privacy is by-architecture** (OpenMemory, federated, DP-RAG, soft-delete + provenance graph) — parametric memory cannot satisfy GDPR Article 17 alone.

---

## 1. Source corpus inventory

`/Users/pedronauck/dev/knowledge/ai-memory/`

- `topic.yaml` — `slug: ai-memory, qmd_collection: ai-memory`.
- `CLAUDE.md` — schema doc; declares 15 wiki articles + 138 raw sources.
- 15 wiki concepts (`wiki/concepts/`) — fully synthesized, citation-rich.
- 138 raw articles (`raw/articles/`) — papers, blog posts, vendor docs, surveys.
- 13 GitHub READMEs (`raw/github/`): `letta`, `mem0`, `zep`, `cognee`, `a-mem`, `agno-memory`, `memos`, `motorhead`, `openmemory`, `qdrant`, `chromadb`, `supermemory`, `reme`.
- `raw/codebase/` — empty (no source code mirrored locally for these systems; only READMEs/docs).

**Notable gaps**: no local clone of `letta-ai/letta`, `mem0ai/mem0`, `getzep/zep`, or `WujiangXu/A-mem` source — so claims below are grounded in their docs/READMEs and the surveys, not source-line citations.

---

## 2. Memory taxonomies in active use

Three converging taxonomies appear across the corpus:

### 2.1 CoALA — the canonical four-type model

CoALA (Sumers et al., 2023, TMLR) is treated as the de-facto standard taxonomy across `wiki/concepts/Cognitive Architectures for Language Agents.md` lines 38–58, `wiki/concepts/Memory Techniques for Agents and LLMs.md`, and adoption notes in IBM, MongoDB, LangChain, Letta, and Mem0 docs.

| CoALA type | Cognitive analog | Concrete substrate |
|---|---|---|
| Working (in-context) | Baddeley working memory | Active context window / KV cache |
| Episodic (external) | Tulving episodic / hippocampus | Vector DB or relational rows with timestamps |
| Semantic (external) | Tulving semantic / neocortex | KG, fact embeddings, structured profile |
| Procedural (in-weights + code) | Implicit procedural learning | Model weights + skill library / prompts |

CoALA also classifies *actions* into memory / execution / reasoning / communication — making "write to memory" a first-class action equivalent to a tool call (`Cognitive Architectures for Language Agents.md` lines 60–68).

### 2.2 Operational tier model (OS-inspired)

The dominant production pattern, repeated across MemGPT/Letta, Mem0, the "memory engineering" Medium post (`raw/articles/memory-engineering-agents.md` lines 161–227), and the LangGraph docs:

- **Tier 0 (in-context working set)** — system policies, current objective, plan, last N turns, retrieved memories, compressed observations. Must stay small and stable.
- **Tier 1 (short-term session state)** — tool outputs, step counters, intermediate variables, checkpoints. Redis or DynamoDB.
- **Tier 2 (long-term semantic memory)** — extracted facts, preferences, heuristics, summaries. Vector DB / hybrid retrieval.
- **Tier 3 (long-term structured memory)** — canonical user profiles, permissions, org policies, durable entities. SQL/NoSQL or graph DB. Retrieved by schema, not similarity.
- **Tier 4 (artifact storage)** — PDFs, HTML, logs, transcripts. Pointers in prompt; reversible relocation, not destruction.

Quoted directly from `memory-engineering-agents.md` line 165–227. The same five tiers reappear in `State Persistence Strategies for Long-Running AI Agents.md` lines 192–201.

### 2.3 Letta's three-tier OS analogy

Letta/MemGPT canonical tiers (`wiki/concepts/AI Agent Memory Frameworks Compared.md` lines 56–82, `raw/articles/letta-stateful-agents.md` lines 17–55):

- **Core memory** — pinned blocks, always in context, agent-editable via function calls. ~2,000 chars per block.
- **Recall memory** — searchable conversation history outside the context window.
- **Archival memory** — vector store for long-term episodic/semantic facts, queried via tools.

**Newest evolution — Letta MemFS (`letta-memory.md` lines 169–224):** memory is a git-backed directory of markdown files, edited via bash tools, versioned by git, with parallel subagents using git worktrees. `system/` directory is pinned to context; everything else is visible only as a tree, with content omitted until read. Reflection subagents fire on step-count / compaction-event triggers; defrag subagents reorganize stale memory.

### 2.4 Newer fine-grained taxonomies

- **AdaMem (Tsinghua/Tencent, March 2026)** — four types: working, episodic, persona, graph. Uses **question-conditioned retrieval** to decide which type to activate per query (`Episodic and Semantic Memory in AI Agents.md` lines 80–84).
- **Hindsight** — four "networks": World (objective facts), Experience (first-person actions), Opinion (subjective beliefs with confidence scores), Entity/Observation (synthesized profiles). The **separation of evidence from inference** is the key insight — beliefs live separately from facts (`AI Agent Memory Frameworks Compared.md` lines 199–204).
- **MAGMA (ICLR 2026)** — every memory is a node in **four orthogonal graphs simultaneously**: semantic, temporal, causal, entity. Policy-guided traversal picks the right graph for the query (`Episodic and Semantic Memory in AI Agents.md` lines 68–73). LoCoMo F1 = 0.70 vs. 0.481 full-context.
- **Synapse (arXiv:2601.02744)** — unified episodic-semantic graph with three edge types (temporal, abstraction, association) + **spreading activation** + **lateral inhibition** + **fan effect** + **PageRank prior**. F1 = 40.5 on LoCoMo; **96.6 F1 on adversarial** thanks to uncertainty-aware rejection.

### 2.5 The episodic/semantic split is the single most-cited claim

Pink et al. (Feb 2025), "Episodic Memory is the Missing Piece for Long-Term LLM Agents" (arXiv:2502.06975, `raw/articles/episodic-memory-missing-piece.md`), argues the five properties episodic memory must support: rapid encoding of single experiences, context-specific retrieval, temporal ordering, interference resistance, and consolidation into durable knowledge. Most production agents treat retrieval as static; "the agent has the episodes but never distilled the pattern" (`Episodic and Semantic Memory in AI Agents.md` line 60).

**Decision rule** (AdaMem-style, repeated across Mem0/Zep docs):
- "What happened / when" → episodic store
- "What is / preference" → semantic store
- "How are X and Y related" → graph / multi-hop
- Default → ranked merge of episodic + semantic

---

## 3. Persistence backends — chosen substrates and why

### 3.1 The standard hybrid stack

`Production Memory Architecture Patterns.md` lines 27–41 codifies the production reference:

| Tier | Backend | Latency | Retention |
|---|---|---|---|
| Hot | In-context / in-process cache | sub-100 ms | per-call |
| Warm | Redis / RediSearch HNSW | <5 ms | minutes–hours |
| Cold | PostgreSQL + pgvector | 5–50 ms | indefinite |
| Archival | S3 / object storage | seconds | indefinite |

Promotion and demotion policies (access-count-based promotion, TTL-based eviction, archive after N days zero access) are configurable per memory type, not global.

### 3.2 Backend choice cheat sheet

From `State Persistence Strategies for Long-Running AI Agents.md` lines 130–139 + `Vector Databases for Agent Memory.md`:

| Backend | Strengths | Weaknesses | Best For |
|---|---|---|---|
| **PostgreSQL + pgvector** | ACID, schema, joins, RLS | slower ANN at extreme scale | agent state, user data, audit logs, mid-scale memory |
| **Redis + RediSearch** | sub-ms reads, TTL, pub/sub, HNSW | weak consistency, memory-bound | session context, working memory, coordination |
| **Qdrant / Weaviate / Pinecone** | optimized ANN, payload filtering | no joins, no transactions | pure semantic retrieval, very large stores |
| **Neo4j / Kuzu / Neptune** | multi-hop, provenance, temporal edges | higher complexity, slower writes | KG memory, AriGraph-style world models |
| **S3 / blob** | cheap, immutable | not queryable | artifacts, transcripts, PDFs |
| **Markdown files** | readable, debuggable, git-friendly, zero-infra | no concurrent access, no semantic search | local agents, dev tools, OpenClaw / Letta MemFS |

**Anti-pattern (documented case, `State Persistence Strategies for Long-Running AI Agents.md` line 148):** dumping conversation logs, prefs, knowledge docs, and workflow state into one vector DB. At 20 M embeddings, latency went from 80 ms to ~2 s and the system collapsed under concurrent load. Fix: split into Postgres (history), Redis (session), vector DB (knowledge only).

### 3.3 The file-first pattern

The "How AI Agents Remember Things" article (`raw/articles/how-ai-agents-remember.md`, Damian Galarza, Feb 2026) and Letta's MemFS docs both argue that **storage is the easy part**. The OpenClaw model uses three files:

- `MEMORY.md` — semantic memory, ≤200 lines, **injected into every prompt** (not retrieved).
- `~/.openclaw/workspace/memory/YYYY-MM-DD.md` — episodic daily logs, append-only; today's + yesterday's auto-loaded at session start.
- `~/.openclaw/workspace/memory/YYYY-MM-DD-<topic>.md` — session snapshots, captured on `/new` from the last 15 meaningful messages (filters tool calls, system messages).

This pattern works because: (a) markdown is auditable and git-versionable; (b) the system has **lifecycle hooks** that fire at the right moments; (c) compaction is reframed as a *checkpoint* via the **pre-compaction flush** (see §5.4).

Letta's MemFS (`raw/articles/letta-memory.md` lines 169–224) operationalizes the same idea at production grade: a git-backed `~/.letta/agents/<id>/memory` directory; agent commits-and-pushes to "save"; reflection subagents run via git worktrees; `system/` files are pinned to context; everything else appears in the memory tree but contents omitted until read.

### 3.4 Append-only / immutable variants

- **Memvid** — single `.mv2` file; everything (data, embeddings, search index, metadata) packed in append-only "Smart Frames"; 0.025 ms P50 retrieval. Limitation: no concurrent writes, no fine-grained deletes — must rebuild file to update.
- **Event-sourced memory** (`Multi-Agent Memory Coordination.md` lines 222–225) — all writes are events on an append-only log; current state is a derived view; recovery, audit, and explicit conflict surfacing fall out for free.

---

## 4. Retrieval primitives

### 4.1 Retrieval is a pipeline, not top-k

Production retrieval is uniformly described (Mem0 docs, `memory-engineering-agents.md` lines 326–367, `Production Memory Architecture Patterns.md` §8) as a five-stage pipeline:

1. **Candidate generation** — combine **dense similarity** (embeddings), **sparse match** (BM25/keyword), and **metadata filtering** (`user_id`, project, time window, trust level). Cheap, broad, scoped.
2. **Expansion** — retrieve linked neighbors / cluster members. A-MEM "seed + neighborhood" pattern.
3. **Prioritization** — re-rank by relevance, recency decay, importance, trust, diversity.
4. **Packaging** — structure retrieved memory for the model. **"Packaging often dominates ranking in impact."** Stable layout, key facts explicit, separated from noisy traces.
5. **Injection** — place in a consistent prompt location so the model implicitly learns to expect it.

### 4.2 Hybrid search — the dominant production primitive

Three modalities + Reciprocal Rank Fusion (RRF, k=60) to merge:

- **BM25** — exact terms, names, SKUs, dates, technical phrases.
- **Dense semantic** — conceptual relevance, paraphrase tolerance.
- **Graph traversal** — multi-hop, relational, "user's manager's email."

Concrete pgvector example (`Production Memory Architecture Patterns.md` lines 137–148):

```sql
SELECT id, content,
       (embedding <=> $1::vector) AS semantic_dist,
       ts_rank(to_tsvector(content), plainto_tsquery($2)) AS bm25_rank
FROM agent_memories
WHERE user_id = $3
  AND (ttl_expires IS NULL OR ttl_expires > NOW())
ORDER BY (embedding <=> $1::vector) * 0.7 + (1 - ts_rank(...)) * 0.3
LIMIT 10;
```

Both Redis (RediSearch) and Weaviate support RRF natively; pgvector via tsvector + HNSW.

### 4.3 The memory router

For complex deployments, a lightweight classifier (or the LLM itself) routes queries to the appropriate modality (BM25 for "find SKU X-200", dense for "tell me about my preferences", graph for "who recommended this"). One LLM call cost, but avoids running all three on every request (`Production Memory Architecture Patterns.md` line 189; AdaMem question-conditioned retrieval).

### 4.4 Spreading activation (Synapse)

`Episodic and Semantic Memory in AI Agents.md` lines 92–151 — Synapse beats pure top-k by:

- **Dual anchor selection** — BM25 (exact entity matches) ∪ dense (conceptual). Activation injected only into anchors.
- **Propagation with fan effect** — activation spreads along edges, diluted by source out-degree (prevents hub flooding). Edge weights = temporal decay (ρ=0.01) + semantic similarity.
- **Lateral inhibition** — top-M=7 nodes compete; rest inhibited. Sparsity guarantee.
- **Sigmoid activation** — converges in T=3 iterations.
- **Triple-signal final score** — `0.5·cosine + 0.3·activation + 0.2·PageRank`. Top-30 retrieved, topologically reordered.

Result: 814 tokens/query (95% reduction vs full-context), 1.9 s avg latency, 167.3 F1-per-dollar (32% better than next-best MemoryOS at 126.8). Ablation: removing fan effect drops F1 from 40.5 → 36.1; removing decay drops 50.1 → 14.2 on temporal; removing spreading activation entirely drops 40.5 → 30.5.

### 4.5 Reranking

Mem0 v1.0.0 (`raw/articles/state-of-ai-agent-memory-2026.md` line 297) added an explicit reranker layer (Cohere, ZeroEntropy, HuggingFace, Sentence Transformers, LLM-based). Vector similarity returns a candidate set; reranker re-scores. This is now standard: "vector similarity returns a candidate set, but the ordering of that candidate set is often wrong."

### 4.6 Multi-factor retrieval scoring

Generative Agents (Park et al., 2023) established the canonical 3-signal score:

```
score = α_recency · recency + α_importance · importance + α_relevance · relevance
```

Mem0's composite scoring weights: 60% semantic similarity + 20% extraction confidence + 10% recency + 10% access frequency (`Episodic and Semantic Memory in AI Agents.md` line 78).

---

## 5. Write-side design — the actual hard problem

### 5.1 The core ADD/UPDATE/DELETE/NOOP discipline

This is the single most repeated pattern across the corpus (Mem0 paper arXiv:2504.19413, `Memory Techniques`, `State Persistence Strategies`, `memory-engineering-agents.md`). Direct quote (`memory-engineering-agents.md` lines 273–325):

- **ADD** — durable signal, no conflict. "User prefers email follow-up." Should be selective; over-use → bloat.
- **UPDATE** — revision of an existing fact. "User moved from Mumbai to Bangalore" → delete old city, add new. Without this: contradictory truth.
- **DELETE / invalidate** — fact has become wrong; constraint expired; access revoked. Hardest because contradiction detection is non-trivial.
- **NOOP** — "the most undervalued operation and arguably the strongest indicator of a mature memory system." Default action of a robust controller is conservative.

Mem0's pipeline: extract candidate from latest exchange + rolling summary + recent messages → compare against K most-similar existing memories → LLM picks ADD/UPDATE/DELETE/NOOP.

### 5.2 The memory controller as a load-bearing component

`memory-engineering-agents.md` lines 229–238: "Most memory discussions focus heavily on storage backends. The component that is consistently missing is the memory **controller**." Responsibilities:

1. What should be stored from a given interaction.
2. Where it belongs (semantic / structured / artifact).
3. Which operation applies (ADD/UPDATE/DELETE/NOOP).
4. What should be retrieved next time.
5. How to prevent poisoning and contradictions.

Without a controller, memory becomes either a transcript dump or an unmaintained vector store — both degrade over time. **Memory writes should be treated as privileged operations.**

### 5.3 Hot-path vs. background writes

Two modes (Memory Techniques, LangGraph docs, Mem0 docs):

- **Hot-path** — synchronous, before final response. Memory immediately available; adds latency to every turn.
- **Background** — async, after response sent. Eliminates write latency but introduces a window of unavailability.

Production hybrid: hot-path for **critical user preferences and direct corrections**; background for **synthesis, cleanup, reflection, clustering, consolidation**.

Mem0 v1.0.0 made `async_mode=True` the default (`raw/articles/state-of-ai-agent-memory-2026.md` line 295) "because memory writes that block the response pipeline add latency the user feels."

### 5.4 The pre-compaction flush — write-ahead log for memory

OpenClaw's most clever pattern (`raw/articles/how-ai-agents-remember.md` lines 116–125):

```
"Pre-compaction memory flush. Store durable memories now (use memory/YYYY-MM-DD.md;
create memory/ if needed). If nothing to store, reply with NO_REPLY."
```

When the session nears context window limit, the system injects a silent agentic turn instructing the agent to write durable memories before compaction destroys them, then reply `NO_REPLY` so it never surfaces in the conversation. **Compaction as a checkpoint, not a loss.**

### 5.5 Reflection / consolidation / sleep-time compute

Letta's reflection subagents (`raw/articles/letta-memory.md` lines 143–168):

- **Step count trigger** — every N user messages.
- **Compaction event trigger (recommended)** — when the context window is summarized.
- Subagent runs in background, modifies memory git repo via worktrees, notifies main agent on completion.

Generative Agents reflection: when accumulated importance exceeds threshold, the agent examines recent stream, identifies patterns, generates higher-level insights ("Klaus seems to be planning something social"), stores them as high-importance entries. **This is the consolidation bridge from episodic to semantic.**

A-MAC (Adaptive Memory Admission Control, MemAgents 2026) — five interpretable factors for what to consolidate:
1. Future utility
2. Factual confidence
3. Semantic novelty
4. Temporal recency
5. Content type prior

### 5.6 The skill library / procedural acquisition pipeline

For procedural memory (Voyager, JARVIS-1, SAGE, ExpeL):

```
propose → generate → execute → critique → verify → store
```

Verification gate is the most important architectural decision. Options:

- **Self-verification** (Voyager) — fast, relies on LLM self-assessment.
- **Reward-based** (SAGE/GRPO) — objective signal `R = R_outcome + λ₁·R_skill_generation + λ₂·R_skill_utilization`.
- **Human-in-the-loop** — highest reliability, lowest throughput.
- **Canary testing** — held-out test cases, deterministic for code skills.

Voyager skills are **JavaScript code with compositional reuse** — `craft_iron_pickaxe` calls `mine_iron_ore` calls `locate_cave`. Library grows monotonically; new skill never overwrites old. Result: 3.3× more unique items, 15.3× faster milestones than non-library baselines.

ExpeL retains **failures as well as successes** to extract anti-patterns ("what not to do"). LEGOMem demonstrates that **orchestrator memory > agent memory** for hierarchical multi-agent — high-level decomposition strategies dominate low-level execution heuristics.

---

## 6. Forgetting / decay / TTL strategies

### 6.1 Decay families

| Approach | Where used | Mechanism |
|---|---|---|
| **Exponential decay (Ebbinghaus)** | MemoryBank | `R = e^(-t/S)`; S increases on access; below threshold → prune |
| **Linear decay** | Aura | `relevance_score`: 1.0 → 0.5% per day; floor 0.01; half-life ≈ 138 days |
| **Recency in scoring** | Mem0 (10%), Generative Agents | weighted in retrieval score, no actual deletion |
| **TTL-based** | Redis, pgvector schemas | per-tier TTL: working 1h, session 24h, episodic 7–30d, prefs no TTL |
| **Validity windows** | Zep / Graphiti | bi-temporal: `valid_from / valid_to` + `tx_time`; old facts marked invalid, not deleted |
| **Append-only with supersedence** | Aura cosine-similarity merge nightly (>0.95 → merge, +10% boost; duplicate soft-deleted) | structural |

### 6.2 Aura's full schema (Postgres-native, `AI Agent Memory Frameworks Compared.md` lines 222–238)

```sql
CREATE TABLE memories (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  content         TEXT NOT NULL,
  type            memory_type NOT NULL,  -- fact|decision|personal|relationship|sentiment|open_thread
  source_message_id UUID REFERENCES messages(id),
  source_channel_type channel_type NOT NULL,
  related_user_ids TEXT[] NOT NULL DEFAULT '{}',
  embedding       VECTOR(1536),
  relevance_score REAL NOT NULL DEFAULT 1.0,
  shareable       INTEGER NOT NULL DEFAULT 0,
  search_vector   TEXT GENERATED ALWAYS AS (
    to_tsvector('english', coalesce(content, ''))
  ) STORED,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Daily cron decays `relevance_score` by 0.5%; nightly consolidation merges cosine ≥ 0.95 pairs. **Six concrete memory types**: fact, decision, personal, relationship, sentiment, open_thread.

### 6.3 The hard staleness problem

Recurring open problem (`AI Agent Memory Frameworks Compared.md` line 290–292, `state-of-ai-agent-memory-2026.md` line 317): a high-relevance memory ("user works at Acme") is highly retrieved and highly relevant **until the day it isn't** — at which point it becomes confidently wrong. Decay handles low-salience gradual obsolescence; **no current framework handles discrete state changes for high-relevance memories**.

Mitigations in the wild:
1. Timestamp every memory + age-based decay.
2. Allow explicit user updates / deletions.
3. Periodically prompt the user to confirm or update.
4. Use temporal validity windows (Zep) instead of point-in-time.

### 6.4 Compression as forgetting

Recursive summarization (`Memory Consolidation and Forgetting.md` lines 170–186): `S_t = LLM(S_{t-1}, D_t)`. Always far shorter than full history; allows infinite-length conversations within fixed context. Cost: minor inaccuracies compound; verbatim recall lost; quality bounded by summarization ability of base LLM. **+3% BLEU on text-davinci-003 across LLMs without task-specific training.**

### 6.5 Three-tier deletion decision framework

`Memory Consolidation and Forgetting.md` lines 196–210:

- **Keep in episodic store** — recent N sessions, explicit user corrections, turns containing facts with known downstream relevance.
- **Compress to semantic store** — repeated patterns across episodes, recurring stable preferences, general world knowledge that surfaced.
- **Delete with audit trail** — sensitive (right-to-erasure), outdated/superseded, low-salience filler.

---

## 7. Scoping — namespace and identity model

### 7.1 Mem0's four-axis model is the de facto standard

Every memory write carries at least one of:
- `user_id` — cross-session per user
- `agent_id` — agent-instance memory
- `run_id` / `session_id` — single-conversation scope
- `app_id` / `org_id` — shared org context

**They compose**: query can scope to user-within-run, or all memories for user across runs (`AI Agent Memory Frameworks Compared.md` line 180). Metadata filtering (added v1.0.0) allows structured attributes queryable independently of semantic content.

### 7.2 LangGraph BaseStore

Hierarchical namespace tuple (`raw/articles/langgraph-memory-production.md` lines 51–69):

```python
namespace = ("user_123", "preferences")  # tuple
key = "dietary_preferences"
value = {"vegetarian": True, "allergies": ["nuts"]}
store.put(namespace, key, value)
store.search(namespace, query="food preferences")
```

Three operations: `put / get / search`. Backends pluggable: `InMemoryStore`, `SqliteStore`, `PostgresStore` (with pgvector), `RedisStore`, `Mem0Store`, `ZepStore`. **GDPR cleanup is namespace prefix delete.**

### 7.3 Letta scoping

Per-agent scope by default. Memory blocks can be **shared** by attaching the same block to multiple agents — a deliberate primitive for multi-agent coordination (`raw/articles/letta-stateful-agents.md` line 41).

### 7.4 Multi-tenant isolation patterns

`Production Memory Architecture Patterns.md` lines 207–217:

- **Per-tenant collections** — own index/partition per tenant. Operationally expensive; hard isolation; enables per-tenant tuning.
- **Shared collection + namespace filter** — single index, filtered by `user_id`. Simpler ops; requires careful query construction; **pgvector + RLS policies enforce isolation at DB layer** even if the app omits the filter.

### 7.5 Cross-session identity resolution — open problem

(`AI Agent Memory Frameworks Compared.md` lines 302–304, `state-of-ai-agent-memory-2026.md` line 315): all current frameworks assume a stable `user_id`. Multi-device, anonymous-vs-authenticated, and identity-stitching is bespoke per app.

### 7.6 Actor-aware memory in multi-agent systems

Mem0 Group-Chat v2 (June 2025, `state-of-ai-agent-memory-2026.md` lines 261–268): tag every stored memory with its source actor. At retrieval, a planning agent can filter "what the user actually said" vs. "what another agent inferred" — preventing a downstream agent from treating an inference as ground truth.

---

## 8. Trust, provenance, verification

### 8.1 Provenance links from semantic to episodic

Standard pattern (Mem0g, MAGMA, dual-store consolidation):

```typescript
interface SemanticMemory {
  id: string;
  fact: string;
  confidence: number;
  sourceEpisodeIds: string[];   // Provenance
  lastUpdated: Date;
  embedding: number[];
}
```

Episodes are **marked consolidated, not deleted** — provenance preserved for audit trail and "when did this start?" queries (`Episodic and Semantic Memory in AI Agents.md` lines 282–303).

### 8.2 Hindsight's evidence/inference split

Single most architecturally distinctive idea in the corpus. Four networks (`AI Agent Memory Frameworks Compared.md` lines 199–204):

- **World Network** — objective external facts.
- **Experience Network** — first-person action history.
- **Opinion Network** — subjective beliefs **with confidence scores** that update as evidence accumulates.
- **Entity/Observation Network** — synthesized profiles of people/companies/topics.

When the agent forms a belief like "this user prefers concise answers," it lives in the Opinion Network with a confidence score — not mixed with factual memory "user's name is Priya." The belief can be updated, challenged, and explained.

### 8.3 Bi-temporal validity (Zep / Graphiti)

`Knowledge Graph Memory Systems.md` lines 119–135:

- **Valid time** — when the fact was true in the world ("user preferred tea Jan–Oct 2024").
- **Transaction time** — when recorded ("recorded Oct 15, 2024").

Old facts are not deleted on supersedence — marked with `valid_to` end timestamp. The graph accumulates a complete temporal history. Enables: "what did the user prefer at the time of the last contract renewal?", "how have priorities changed?", "what was our recorded understanding at time T?"

### 8.4 A-MEM's evolving notes

Memory entries are **mutable** in the sense that adding a new note can update related historical notes' contextual representation. The network reorganizes itself as knowledge accumulates. Static memory systems treat each stored entry as immutable; A-MEM does not (`Cognitive Architectures for Language Agents.md` lines 226–243).

### 8.5 Conflict detection at write time

Mem0g — entity extractor + relations generator + **conflict detector** that flags contradictions before writes (`Episodic and Semantic Memory in AI Agents.md` line 213). Schema validation in MetaGPT — downstream agents reject malformed PRDs rather than propagate hallucinated assumptions (`Multi-Agent Memory Coordination.md` line 113).

### 8.6 Audit logs as a first-class feature (Agno)

Agno (`raw/github/agno-memory.md`): "All agent data stored in your own database. Full auditability. Audit logs as first-class features. Per-user isolation enforced at framework level." This is the production posture: a memory event log is itself a queryable artifact.

### 8.7 Memory consistency protocols (multi-agent)

`Multi-Agent Memory Coordination.md` lines 178–197:

- **Read-your-writes** — agent sees its own write on subsequent read. Trivial-sounding; fails when writes go through async pipelines.
- **Monotonic read** — once a value V is read at time T, subsequent reads ≥ V (no time-travel).
- **Causal consistency** — practical target. Causally-related writes appear in causal order to all agents.
- **Sequential consistency** — strongest, most expensive; only feasible for small co-located deployments.

Conflict resolution patterns: last-write-wins (LWW), merge functions (LLM-mediated), version vectors (CRDT-style; conflicts surfaced explicitly).

---

## 9. Failure modes documented

### 9.1 Production failure mode taxonomy

(`memory-engineering-agents.md` lines 437–443, `State Persistence Strategies.md` lines 220–245):

1. **Memory bloat** — uncontrolled accumulation; retrieval degenerates into noise.
2. **Contradictory truth** — multiple conflicting memories coexist; agent arbitrates inside prompt, often wrong, often inconsistent across turns. "It 'remembers' something one day and 'forgets' it the next."
3. **Memory poisoning / instruction injection** — untrusted tool outputs or user-provided content stored, later retrieved as authoritative instruction.
4. **Episodic imitation drift** — agent blindly mimics prior action patterns because they appear in context.
5. **Hallucinated retrieval** — agent retrieves wrong node/summary; error feels grounded even when fabricated. **Synapse's uncertainty-aware rejection scores 96.6 F1 on adversarial vs. A-MEM 50.0**.
6. **Lost-in-the-middle** — long context windows have non-uniform attention; relevant facts in middle are missed. 200k-token models become unreliable around 130k.
7. **Skill graveyard** — stale skills retrieved, fail because preconditions no longer hold. No deprecation policy → drift.
8. **Retrieval distribution shift** — embedding similarity proxy breaks under domain shift; skills retrieve well, execute poorly.
9. **Synchronous-pipeline tail latency** — multi-stage retrieval on hot path consumes the latency budget.
10. **Vector-DB-as-everything** — pushing transactional/structured workloads into vector store; performance collapse around 20 M embeddings.

### 9.2 Privacy failure modes

`Privacy and Memory Rights in AI Systems.md` lines 27–35, 141–151:

- **Data leakage through generation** — synthesized output reveals more than any single stored entry.
- **Unauthorized cross-user access** — improperly namespaced embeddings retrievable by other users; prompt injection surfaces another user's memories.
- **Membership inference** — attacker probes whether a fact is in training/storage by observing response distributions.
- **Adversarial memory poisoning** — attacker inserts false facts directly or via crafted conversational inputs.
- **Reconstruction attacks** — targeted completion queries reconstruct private training data.
- **Prompt injection for memory exfiltration** — malicious instructions embedded in content the agent later reads.

### 9.3 The parametric gap

(`Memory Consolidation and Forgetting.md` lines 219–220, `Privacy and Memory Rights.md` lines 41–46):

> Operational memory (vector stores, databases) can be precisely deleted, but parametric memory (LLM weights) cannot — not with current techniques. A user may successfully delete their information from a vector store while the underlying LLM still "knows" it from pre-training. **True right-to-be-forgotten compliance is impossible for systems built on frontier LLMs without model-level unlearning.**

WikiMem (Staufer, XKDD 2025) provides a memorization metric: 5,000+ canaries × 243 Wikidata properties × 200 individuals at varying notability. Memorization correlates with web presence; scales with model size. SeUL (AAAI 2025) does token-level selective unlearning; ROME/MEMIT do locate-and-edit on FFN layers.

---

## 10. Concrete competitor systems — architectures grounded in their docs

### 10.1 Mem0 — memory as a service

**Source:** `raw/github/mem0-readme.md`, `raw/articles/mem0-production-memory.md`, `raw/articles/state-of-ai-agent-memory-2026.md`.

- **License:** Apache 2.0. **Status:** Y Combinator S24, 14 M downloads, 41k GitHub stars, **AWS exclusive memory provider for Agent SDK**.
- **Pipeline:** extract candidate from (latest exchange + rolling summary + recent messages) → compare against K most-similar → LLM picks ADD/UPDATE/DELETE/NOOP.
- **API surface:** `memory.add(messages, user_id)` / `memory.search(query, user_id)` / `memory.update(id, data)` / `memory.delete(id)`.
- **Scopes:** four-axis (`user_id`, `agent_id`, `session_id`, `app_id/org_id`), composable.
- **Storage:** 19 vector backends (Qdrant, Pinecone, Chroma, Weaviate, pgvector, Milvus, Cassandra, FAISS, Valkey, Elasticsearch, Azure AI Search, S3 Vectors, Mongo, etc.) + 3 graph backends (Neo4j, Kuzu, Neptune).
- **Mem0g (graph variant):** entity extractor → relations generator → **conflict detector** → directed labeled KG alongside vector store. +2% LoCoMo accuracy; **+36% temporal-reasoning accuracy** vs. OpenAI Memory.
- **Benchmarks:** LoCoMo 66.9% (Mem0) / 68.4% (Mem0g) vs. 72.9% full-context. **91% lower p95 latency**, **90% token savings**. p95 latency 1.44 s (Mem0) / 2.59 s (Mem0g) vs. 17.12 s full-context.
- **21 framework integrations** (LangChain, LangGraph, LlamaIndex, CrewAI, AutoGen, Mastra, Vercel AI SDK, OpenAI Agents SDK, Google ADK, ElevenLabs, LiveKit, Pipecat, ...).
- **Procedural memory:** v1.0.0 added `memory_type="procedural_memory"` — different extraction prompt focused on workflows, not facts.
- **Reranking:** v1.0.0 added Cohere/ZeroEntropy/HF/Sentence Transformers/LLM-based.
- **Async default:** `async_mode=True` since v1.0.0.
- **Actor-aware memory:** Group-Chat v2 (June 2025) tags writes with source actor.
- **Inclusion/exclusion prompts, memory depth, usecase config:** v1.0.3 — per-project tuning of extraction.
- **Consistency:** eventually consistent (100–500 ms async writes).

### 10.2 Letta (formerly MemGPT) — memory as a runtime

**Source:** `raw/github/letta-readme.md`, `raw/articles/letta-memory.md`, `raw/articles/letta-stateful-agents.md`, `wiki/concepts/Cognitive Architectures for Language Agents.md` lines 135–155.

- **License:** Apache-style; full open source. **Letta Code** = local CLI; **Letta API** = SDKs.
- **Architecture:** complete agent runtime. Three tiers: core memory (in-context, ≤2k chars/block, agent-editable), recall memory (searchable history), archival memory (vector store, function-call retrieval).
- **MemFS (v0.15+)** — git-backed markdown directory `~/.letta/agents/<id>/memory`. `system/` files pinned to prompt; everything else visible as tree but content omitted until read. **Bash-tool editing**, git commits-and-pushes to "save", reflection/defrag subagents via **git worktrees** for parallel modification.
- **Memory blocks (legacy):** `human` (facts about user), `persona` (agent identity). Server-side `memory` / `memory_apply_patch` tools.
- **Reflection trigger options:** Off / Step count / Compaction event (recommended). Subagent runs in background, notifies main agent on completion.
- **Defrag command:** explicit subagent that refactors large redundant blocks into smaller nested structure.
- **Personality presets:** /personality switches `system/persona.md` + `system/human.md`, requires clean repo, prompts `/clear` or `/new`.
- **Sleep-time compute:** background memory consolidation between active sessions.
- **Consistency:** transactional (10–50 ms immediate write/read).
- **Drawback:** Letta is a *replacement runtime*, not a drop-in layer. Doesn't integrate with LangGraph/CrewAI/AutoGen/LlamaIndex.
- **Benchmarks:** ~83.2% LoCoMo (independent), 93.4% MemGPT DMR; 94.8% Zep DMR.

### 10.3 Zep (Graphiti) — temporal knowledge graph

**Source:** `raw/github/zep-readme.md`, `wiki/concepts/Knowledge Graph Memory Systems.md` lines 119–155, `raw/articles/zep-temporal-knowledge-graph.md` (referenced).

- **License:** Apache 2.0 for Graphiti engine. Community Edition deprecated (legacy/).
- **Architecture:** memory = **temporal knowledge graph** built by Graphiti.
- **Three-tier subgraph hierarchy:**
  - **Episode subgraph** — raw conversation exchanges, max granularity.
  - **Semantic entity subgraph** — extracted entities + typed relationships, **temporal validity on edges**.
  - **Community subgraph** — higher-level thematic clusters.
- **Bi-temporal model** — every fact has `valid_time` (when true in world) and `transaction_time` (when recorded). Old facts marked `valid_to`, never deleted.
- **Retrieval:** bi-encoder (queries + nodes encoded independently, dot product), avoiding LLM-as-retriever latency. Hybrid: semantic + BM25 + graph traversal + cross-encoder rerank.
- **Latency:** 90% lower vs. MemGPT iterative LLM retrieval. P95 < 300 ms.
- **Benchmarks:** 94.8% on Deep Memory Retrieval (vs. MemGPT 93.4%), up to 18.5% improvement on LongMemEval, ~75% on LoCoMo (corrected).
- **Cost concern:** 600,000+ tokens/conversation memory footprint vs. Mem0's ~1,800.
- **Real-time concern:** immediate post-ingestion retrieval often fails — correct answers appear hours later after background graph processing.
- **Enterprise integration:** ingests structured business data alongside conversations (CRM, product catalogs, customer histories) — single graph spans both.

### 10.4 A-MEM — Zettelkasten agentic memory

**Source:** `raw/github/a-mem-readme.md`, `wiki/concepts/Cognitive Architectures for Language Agents.md` lines 226–243.

- **License:** MIT. NeurIPS 2025 (Xu et al., arXiv:2502.12110).
- Each memory = structured note with: contextual description, keywords, categorical tags, **explicit links to related existing memories**.
- Four operations: note creation, **link discovery**, **memory evolution** (new info updates contextual representation of related historical notes), adaptive multi-attribute graph traversal retrieval.
- **The memory evolution operation is the key innovation** — static memory treats each entry immutable; A-MEM allows new info to update related prior entries' representations.
- Tested across 6 foundation models (GPT-4, Claude, etc.) on LoCoMo. Backends: OpenAI, vLLM, Ollama.

### 10.5 Hindsight — structured belief networks

**Source:** `wiki/concepts/AI Agent Memory Frameworks Compared.md` lines 199–204.

- Four memory networks: World, Experience, Opinion (with confidence scores), Entity/Observation.
- **91.4% LongMemEval overall** (top score in published comparisons; 21.1% → 79.7% improvement on multi-session).
- Ships with MCP server, Docker deployment.

### 10.6 Supermemory — production hybrid stack

**Source:** `raw/github/supermemory-readme.md`.

- #1 LongMemEval, #1 LoCoMo, #1 ConvoMem (claimed).
- Five layers: (1) connectors auto-sync (Drive, Gmail, Notion, OneDrive, GitHub); (2) multi-modal extractors (PDFs, images OCR, videos transcript, code AST-aware chunking); (3) Super-RAG hybrid; (4) memory graphs; (5) user profiles (stable + recent activity, ~50 ms).
- API: `client.add()` / `client.profile()` / `client.search.memories()`.
- MCP install: `npx -y install-mcp@latest https://mcp.supermemory.ai/mcp --client claude --oauth=yes`.

### 10.7 MemOS — memory operating system

**Source:** `raw/github/memos-readme.md`.

- License: Apache 2.0. Released May 2025.
- **+43.70% accuracy vs OpenAI Memory; -35.24% memory tokens.**
- Memory as inspectable **graph structure**, not opaque embeddings.
- Multi-modal: text, images, tool traces, personas.
- "Memory cubes" — composable knowledge bases with isolation and controlled cross-project sharing.
- **MemScheduler** — Redis Streams background processing for minimal hot-path latency.
- Natural-language feedback to refine/correct memories; precise deletion.
- v2.0 ships: comprehensive KB (doc/URL parsing), tool memory for agent planning, MCP upgrade.

### 10.8 Cognee — knowledge engineering pipeline

**Source:** `raw/github/cognee-readme.md`.

- Three pillars: vector search + graph DBs + cognitive-science-inspired organization.
- Pipeline framing: ingest any format → extract structured knowledge → store in combined graph+vector → surface context for queries.
- Local execution; ontology grounding; Modal/Railway/Fly.io/Render/Daytona deployment.

### 10.9 Agno — bring-your-own-DB

**Source:** `raw/github/agno-memory.md`.

- Per-user/per-session isolation enforced at framework level; **all data in user's own DB**.
- Configurable: `add_history_to_context=True, num_history_runs=3`.
- SQLite quickstart; full data sovereignty; **audit logs as first-class**.
- Example agents: Pal (personal preferences), Gcode (post-IDE coding agent), Dash (six-layer self-learning data agent).

### 10.10 ReMe — file-based + vector hybrid

**Source:** `raw/github/reme-readme.md`.

- **ReMeLight** = "memory as files" — markdown, BM25 + vector, daily journals + persistent prefs.
- Three vector memory types: personal, procedural, tool-specific.
- Seven core components: context checking, conversation compaction, persistent summarization, tool result management, semantic search, in-session memory, **unified pre-reasoning hook** (orchestrates everything).
- **86.23% LoCoMo** (vs. 81.55% prior best); 94.06% HaluMem; +3.46% AppWorld; +6.22% BFCL-V3.
- Backends: in-memory, Chroma, Qdrant, Elasticsearch.

### 10.11 OpenMemory — local-first MCP

**Source:** `raw/github/openmemory-readme.md`, `wiki/concepts/Privacy and Memory Rights in AI Systems.md` lines 116–126.

- All storage in local Docker volume; nothing transmitted externally.
- MCP install: `npx @openmemory/install local http://localhost:8765/mcp/<client>/sse/<user-id> --client <client>`.
- Four MCP operations: `add_memories`, `search_memory`, `list_memories`, `delete_all_memories`.
- Ollama backend for fully offline.
- **Privacy by architecture, not policy** — GDPR trivially satisfied.

### 10.12 Memvid — single-file append-only

`AI Agent Memory Frameworks Compared.md` lines 207–210.

- Single `.mv2` file with embeddings + index + metadata in append-only Smart Frames.
- 0.025 ms P50 retrieval; 1,372× higher throughput vs. standard RAG.
- No concurrent writes, no fine-grained deletes (rebuild file). Read-mostly, edge/offline.

### 10.13 Generative Agents (Park et al., 2023, ACM UIST)

`Cognitive Architectures for Language Agents.md` lines 107–134.

- **Memory stream** — append-only DB of all observations as natural-language records with timestamp + LLM-assigned importance score.
- Triple retrieval: `recency · α_r + importance · α_i + relevance · α_rel`.
- **Reflection** — when accumulated importance crosses threshold, agent examines stream, identifies patterns, generates abstract insights stored back in stream as high-importance entries. **The most cognitively interesting consolidation mechanism implemented.**
- Hierarchical planning: daily → hour-by-hour → minute-by-minute.

### 10.14 LangGraph — composable memory primitives

`raw/articles/langgraph-memory-production.md`, `raw/articles/langgraph-memory-docs.md`.

- **Two-layer model:** thread-scoped (checkpoints) + cross-thread (BaseStore).
- BaseStore: `put / get / search` over `(namespace_tuple, key, value)`.
- Backends: InMemory, SQLite, Postgres+pgvector, Redis, Mem0, Zep.
- Checkpoint backends: `RedisSaver`, `PostgresSaver`. **Critical for long-running agents — without checkpointing, 90% higher rate of total task failure for runs >4h.**
- Two write patterns: hot-path (sync) and background (async). Hybrid in production.
- Cross-thread sharing via shared namespaces; LWW on conflicts.

### 10.15 Motorhead — historical reference (deprecated)

`raw/github/motorhead-readme.md`. Pioneer of memory-as-microservice. Get/post/delete/search endpoints; `MAX_WINDOW_SIZE=12` default; long-term via Redisearch. Architecturally established the dedicated memory server pattern.

### 10.16 Vector / KG infrastructure summary

| Backend | Index | Hybrid | Multi-tenant | Best for |
|---|---|---|---|---|
| **Chroma** | HNSW | yes | basic | prototyping, single-process Python |
| **Qdrant** | HNSW + ACORN filter | yes (BM25 + dense) | yes | production scale, heavy filtering |
| **Weaviate** | HNSW | RRF native | tenant isolation | hybrid + multi-modal |
| **pgvector** | HNSW / IVFFlat | tsvector + RRF | RLS | already-Postgres shops, ACID |
| **Redis (RediSearch)** | HNSW / FLAT | yes | keyspace | warm tier, sub-ms |
| **Pinecone** | proprietary | metadata | yes | managed scale, no ops |
| **Milvus / Cassandra / FAISS** | various | partial | varies | specific scale/cost profiles |
| **Neo4j / Kuzu / Neptune** | graph | n/a | varies | KG memory |

---

## 11. API / SDK patterns — how agents read/write memory

### 11.1 Direct CRUD APIs

**Mem0:**
```python
memory.add(messages, user_id="customer_123", agent_id="support_v2",
           metadata={"context": "healthcare"})
memory.search(query="What does customer_123 need?", user_id="customer_123")
memory.update(memory_id, data, timestamp=...)
memory.delete(memory_id)
```

**Supermemory:**
```typescript
client.add({ content, user_id })
client.profile({ user_id })           // ~50ms
client.search.memories({ query, user_id })
```

**LangGraph:**
```python
store.put(("user_123", "preferences"), "diet", {"vegetarian": True})
store.get(("user_123", "preferences"), "diet")
store.search(("user_123",), query="food preferences", limit=5)
```

### 11.2 Agent-as-controller (Letta, MemGPT, SCM)

The agent issues **explicit function calls** to read/write memory. The LLM is its own memory controller; memory ops are tools:

```python
agent.core_memory.edit("Customer prefers email over phone")
agent.archival_memory_search("similar billing dispute history")
```

In Letta MemFS, this is taken further — the agent edits memory **with the same bash tools used to write code** (`raw/articles/letta-memory.md` line 226).

### 11.3 Conversational `/remember` / explicit user signals

OpenClaw and Letta both expose `/remember` so the user can directly direct memory updates:

```
> /remember not to make that mistake again
```

The agent infers intent, decides target file (semantic vs episodic), writes accordingly. Letta also exposes `/init`, `/personality`, `/sleeptime`, `/memory`, `/doctor`, `/memfs enable`, `/agents`, `/pin`, `/rename`.

### 11.4 MCP — protocol-level memory

Standard MCP four operations (OpenMemory, Supermemory): `add_memories`, `search_memory`, `list_memories`, `delete_all_memories`. CA-MCP (Jayanti & Han, 2026) adds a Shared Context Store across MCP servers (`Multi-Agent Memory Coordination.md` lines 153–175).

### 11.5 Tool-mediated shared memory (AutoGen pattern)

For frameworks without native shared state (AutoGen, base MCP): implement `store_memory` / `retrieve_memory` as tools backed by Redis/Postgres/vector store. Any agent calling these joins the shared memory pool.

### 11.6 Pre-reasoning hooks (ReMe, OpenClaw)

ReMe's "unified pre-reasoning hook" orchestrates all memory operations before each agent step. OpenClaw's bootstrap loading: `MEMORY.md` auto-injected by the system into every prompt; daily logs loaded by the agent following its instructions. **The agent doesn't search for context — it's already there.**

### 11.7 Lifecycle hook surface (canonical list)

Combining OpenClaw + Letta + ReMe + LangGraph:

- **Session start / agent boot** — load semantic memory, load recent episodic, hydrate working set.
- **Pre-reasoning** — context check, retrieve relevant memory, package into prompt.
- **Post-tool-call** — capture tool outputs, decide ADD/UPDATE/DELETE/NOOP.
- **Post-response** — async memory extraction; reflection (Generative Agents).
- **Pre-compaction** — silent flush turn writes durable memories (OpenClaw).
- **Compaction event** — trigger reflection subagent (Letta).
- **Step-count threshold** — periodic reflection (Letta `/sleeptime`).
- **Session end (`/new`, `/reset`)** — snapshot last meaningful messages (OpenClaw).
- **Idle / sleep-time** — consolidation, deduplication, defragmentation.
- **User-explicit `/remember`** — direct write trigger.
- **Memory write / commit** — git push (Letta MemFS).

---

## 12. Integration with hooks, tools, skills, RAG, CLI

### 12.1 The natural integration points

The corpus consistently treats memory as a **cross-cutting concern wired through hook surfaces and tool registration**:

- **As tools** — `memory_search`, `memory_edit`, `archival_memory_search`. Mem0 exposes `Mem0-memorize` and `Mem0-remember` Mastra tools.
- **As hooks** — pre-prompt injection (MEMORY.md), post-response extraction, pre-compaction flush, session-end snapshot.
- **As skills/agentskills** — memory-management skills documented in Letta Code (`initializing-memory` skill loaded by `/init`).
- **As RAG layer** — Supermemory's Super-RAG, Cognee's vector+graph hybrid, LangGraph + Mem0/Zep store integrations.
- **As CLI commands** — `/init`, `/memory`, `/remember`, `/sleeptime`, `/personality`, `/doctor`, `/memfs enable`, `/new`, `/reset`.
- **As MCP servers** — OpenMemory, Supermemory, MemOS, Hindsight all ship MCP servers.

### 12.2 Skill library + memory

Procedural memory is implemented as a **skill library** (`Procedural Memory and Skill Learning.md`):

```
skill_library/
  ├── description (NL)
  ├── code or plan
  └── embedding
```

Acquisition pipeline: propose → generate → execute → critique → verify → store. Verification gate types: self-verification (Voyager), reward-based (SAGE/GRPO), human, canary. Compositional reuse — `craft_iron_pickaxe` calls `mine_iron_ore`. **Skills retrieved as in-context examples** for the planner.

### 12.3 RAG vs. memory — complementary, not interchangeable

`raw/articles/state-of-ai-agent-memory-2026.md` line 230 + Supermemory README:

> Vector memory tells you "this user mentioned Python." Graph memory tells you "this user works with Python, specifically for data pipelines, using pandas, at a company that uses dbt, migrating from Spark."

Mem0 paper, RAG@61.0% LoCoMo vs Mem0@66.9% — RAG retrieves uniformly across all chunks; **memory extracts and tracks facts about individual users over time**, understands updates supersede previous data, manages temporal validity. Hybrid Super-RAG = personalized + RAG in single query.

### 12.4 Connectors — auto-ingested memory

Supermemory: real-time sync from Drive, Gmail, Notion, OneDrive, GitHub. MemOS v2: doc/URL parsing, cross-project sharing. The connector layer turns external corpora into memory.

### 12.5 CLI surface (Letta canonical example)

`/agents`, `/resume`, `/new`, `/init`, `/personality`, `/remember`, `/sleeptime`, `/memory`, `/doctor`, `/memfs enable`, `/pin`, `/rename`, `/clear`. Combined with `letta` (resume default), `letta --new`, `letta --agent <id>`, `letta agents create --personality <name>`.

OpenClaw: `/new`, `/reset`, plus implicit "remember this" detection.

---

## 13. Design patterns reverse-engineered from the corpus

### 13.1 The retrieval pipeline canon

```
Query
  ↓
Memory Router (LLM or classifier — picks modality)
  ↓
Candidate Generation: dense + BM25 + metadata filter
  ↓
Expansion: linked neighbors / cluster members (A-MEM)
  ↓
Prioritization: relevance · recency · importance · trust · diversity
  ↓
Reranker (Cohere / cross-encoder / LLM)
  ↓
Packaging: stable layout, key facts explicit
  ↓
Injection: consistent prompt section
```

### 13.2 The write pipeline canon

```
Conversation turn / observation
  ↓
Extract candidate facts (LLM, possibly fast/cheap)
  ↓
Compare against K most-similar existing memories
  ↓
Memory Controller decides: ADD / UPDATE / DELETE / NOOP
  ↓ (default: NOOP)
Tag with: user_id, agent_id, session_id, app_id, source_actor,
          purpose, data_categories, consent_ts, valid_from, expires_at
  ↓
Write to appropriate tier (warm/cold/archival) with provenance link
  ↓
(Optional) emit event on coordination channel (Redis pub/sub)
  ↓
(Background) consolidation: dedupe (cosine ≥ 0.95), merge, decay,
              episodic→semantic promotion
```

### 13.3 The session lifecycle canon

```
Boot:
  - Load pinned semantic blocks (MEMORY.md / system/) into context
  - Load recent episodic (today + yesterday) into context
  - Initialize working tier

Active turns:
  - Pre-reasoning hook: route + retrieve + package + inject
  - Generate response (model attends to packaged memory + history)
  - Post-tool-call: capture observations
  - Hot-path write: critical user prefs / corrections (sync)
  - Background write: episodic log, low-confidence facts (async)

Trigger events:
  - Step count N    → reflection subagent
  - Compaction      → pre-compaction flush + reflection
  - Idle T          → background consolidation
  - User /remember  → direct write
  - User /new       → session snapshot

End:
  - Snapshot last meaningful messages (filter tool calls / system)
  - Background defrag if needed
  - Persist all uncommitted memory (git push for Letta MemFS)
```

### 13.4 The scoping canon

Composable axes (Mem0 model):
```
{ user_id, agent_id, session_id, run_id, app_id, org_id,
  purpose, source_actor, namespace_tuple }
```

Multi-tenant isolation: shared collection + RLS or per-tenant collection + per-tenant index tuning. Cross-thread sharing via shared namespace prefix.

### 13.5 The privacy-engineering canon

`Privacy and Memory Rights.md` §11–§13:

- Tag every memory entry: `user_id`, `purpose`, `consent_ts`, `source`, `data_categories`, `expires_at`, `created_at`, `deleted_at`.
- Per-purpose namespaces: `/agent/scheduling`, `/agent/medical`, `/agent/preferences`. Enforce purpose binding at retrieval.
- Tiered retention: ephemeral (session), short (7d), medium (90d), persistent (consent). Special-category data → consent gate before write.
- Soft-delete + scheduled hard-delete + deletion log (proves removal).
- Deletion propagation handler registry — every storage layer registers a handler called by DSAR pipeline.
- Per-user DP budget for sensitive-token generation (DP-RAG, Exponential Mechanism + RDP accountant).
- Federated architecture for personal memory (OpenMemory pattern) — privacy-by-architecture.

### 13.6 The episodic-semantic consolidation canon

```
Episode: "Called March 5  — billing dispute, frustrated"
Episode: "Called March 9  — billing error, requested email follow-up"
Episode: "Called March 14 — refund request, billing issue again"
                              |
                A-MAC scoring + consolidation
                              |
                              v
Semantic: "Customer has recurring billing issues"   confidence 0.88
Semantic: "Prefers email follow-up"                 confidence 0.95
Semantic: "Escalation risk — frustration trend"     confidence 0.71

(All linked back to source episodes via sourceEpisodeIds[] for audit.)
```

Episodes are **marked consolidated, not deleted** — provenance retained.

---

## 14. Architectural patterns vs. AGH-relevant constraints

The AGH runtime is a Go single-binary daemon with SQLite event persistence, ACP subprocess hosting, and CLI/HTTP/UDS/web surfaces (`internal/CLAUDE.md`). The corpus suggests several decisions are particularly load-bearing for an AGH memory redesign:

### 14.1 SQLite is the right primitive — but augment it

Aura demonstrates a **production memory system can run entirely on Postgres with pgvector**, no specialized vector DB. AGH's SQLite model can plausibly do the same with `sqlite-vec` + FTS5 + JSON1. Multi-tenant isolation can be RLS-equivalent via per-`user_id`/`agent_id`/`session_id` filters and dedicated tables per tier.

But: 20 M embeddings is the documented inflection where naive single-store vector approaches collapse. AGH should plan for explicit tier separation from day one (hot in-memory, warm SQLite session table, cold SQLite memory table with vec index, archival blob).

### 14.2 File-first is credible

OpenClaw's pattern (3 markdown files + 4 lifecycle hooks) is production-proven and is what Claude Code itself adopted. Letta MemFS extends this to git-versioned per-agent directories with worktree-based parallel reflection. This dovetails with AGH's "agents manipulate via CLI + REST" goal — markdown is naturally agent-readable and writable; git provides audit/rollback.

### 14.3 Hooks are the integration surface

Every reviewed system that ships in production uses **lifecycle hooks**, not pure embedding-and-retrieve:

- `pre_prompt_injection`, `post_tool_call`, `pre_compaction_flush`, `session_start`, `session_end`, `idle_consolidation`, `step_count_threshold`, `user_remember_command`.

These map directly onto AGH's existing hook architecture (`internal/daemon/hooks_bridge.go`). The memory controller belongs at this layer.

### 14.4 The four-axis scope plus actor-aware tags is the de-facto API

`(user_id, agent_id, session_id, run_id, app_id, org_id, purpose, source_actor)` — directly informs the AGH memory write API surface.

### 14.5 Two-touch anti-bloat: ADD/UPDATE/DELETE/NOOP must be explicit

Every wiki article calls out append-only as a primary failure mode. AGH should not provide an append-only memory primitive without an UPDATE/DELETE/NOOP discipline gated by a controller.

### 14.6 Truthful UI extends to truthful memory

Synapse's uncertainty-aware rejection (96.6 F1 adversarial) and Hindsight's Opinion Network with confidence scores both argue for: **expose confidence and provenance to the agent, not just the content.** Truthful memory → memories surface their evidence chain.

### 14.7 Open-network compatibility (RFC 002 alignment)

Mem0 ships as 21-framework drop-in. Letta is closed-runtime. Zep is open-source-Graphiti + managed. **AGH's "runtime moat, not protocol moat" stance favors a Mem0-style open contract** with a documented memory protocol other runtimes could implement, rather than a Letta-style runtime lock-in.

---

## 15. Open problems documented in the corpus (frontier of the field)

Distilled from `Memory Techniques`, `AI Agent Memory Frameworks Compared`, `Episodic and Semantic Memory in AI Agents`, `state-of-ai-agent-memory-2026.md`, `survey-memory-mechanism-llm-agents-2404.md`:

1. **Memory staleness for high-relevance facts** — discrete state changes (job change, address change) make confidently-retrieved memories actively harmful. No framework handles this gracefully at scale.
2. **Cross-session identity resolution** — multi-device, anonymous-vs-authenticated stitching.
3. **Application-level memory evaluation** — LoCoMo and LongMemEval measure general recall; application-specific quality (coding assistant vs. healthcare) is bespoke and manual.
4. **Epistemic governance** — when two memories contradict, which wins? Confidence calibration. Hindsight's Opinion Network is the most explicit attempt; everywhere else, application-layer.
5. **Memory poisoning defenses** — provenance tracking, write authentication, anomaly detection on writes — all immature.
6. **The parametric gap** — vector store can be wiped; LLM weights cannot. True GDPR Article 17 compliance impossible for systems on frontier LLMs without unlearning.
7. **Cross-agent memory standards** — no standard protocol for agents in different frameworks to share memory. CA-MCP is a step but MCP-specific.
8. **KV cache sharing across agents** — per-agent KV caches when multiple agents process overlapping contexts is a documented inefficiency.
9. **Group-configuration memory** — no current system persists "which agent compositions worked for which task types."
10. **Procedural memory at runtime** — almost no system updates in-weights procedural memory at runtime (LoRA-based continual learning is experimental).
11. **Memory consistency rigor** — most production systems use eventual consistency without formally reasoning about failure modes.
12. **Voice agents at real-time latency** — fastest-growing integration category; <300 ms ceiling.

---

## 16. Curated bibliography (with file paths + claim density)

### Wiki concepts (synthesized; primary references)

- `wiki/concepts/Memory Techniques for Agents and LLMs.md` — taxonomy + operations + structural patterns + tradeoffs.
- `wiki/concepts/AI Agent Memory Frameworks Compared.md` — Mem0 vs Letta vs Zep vs Hindsight vs Memvid vs Supermemory vs Aura, benchmarks, decision framework.
- `wiki/concepts/State Persistence Strategies for Long-Running AI Agents.md` — 7 strategies, DB selection, lifecycle, anti-patterns.
- `wiki/concepts/Production Memory Architecture Patterns.md` — three-tier reference, pgvector schema, hybrid search, build-vs-buy.
- `wiki/concepts/Episodic and Semantic Memory in AI Agents.md` — ICLR 2026 research (MAGMA, Mem0, AdaMem, E-mem, MEM-alpha), Synapse spreading activation, consolidation bridge.
- `wiki/concepts/Cognitive Architectures for Language Agents.md` — CoALA + Generative Agents + MemGPT + Hopfield/FFN-as-KV + A-MEM.
- `wiki/concepts/Memory Consolidation and Forgetting in LLMs.md` — CLS theory, EWC/LoRA, recursive summarization, machine unlearning (SeUL/ROME/MEMIT).
- `wiki/concepts/Privacy and Memory Rights in AI Systems.md` — GDPR/RTBF, DP-RAG, federated, OpenMemory pattern, threat models.
- `wiki/concepts/Procedural Memory and Skill Learning.md` — Voyager / JARVIS-1 / SAGE / LEGOMem / ExpeL / HuggingGPT.
- `wiki/concepts/Knowledge Graph Memory Systems.md` — HippoRAG / Zep+Graphiti / AriGraph / GraphRAG / LightRAG / ChatDB.
- `wiki/concepts/Multi-Agent Memory Coordination.md` — blackboard, CAMEL/MetaGPT/AutoGen/AgentVerse/CA-MCP, consistency protocols.
- `wiki/concepts/Vector Databases for Agent Memory.md` — Chroma/Qdrant/Weaviate/pgvector/Redis comparison, HNSW vs IVF.
- `wiki/concepts/Working Memory and KV Cache Management.md` — StreamingLLM, H2O, TOVA, LOCRET, PagedAttention.
- `wiki/concepts/Memory Evaluation Benchmarks.md` — LongMemEval, LoCoMo, MUSIQUE, SORT, Reflection-Bench, AgentBench.
- `wiki/concepts/Memory for Conversational AI Systems.md` — MemoryBank, MemoChat, THEANINE, Think-in-Memory, SCM.

### Vendor READMEs

- `raw/github/letta-readme.md`, `raw/github/mem0-readme.md`, `raw/github/zep-readme.md`, `raw/github/a-mem-readme.md`, `raw/github/cognee-readme.md`, `raw/github/agno-memory.md`, `raw/github/memos-readme.md`, `raw/github/openmemory-readme.md`, `raw/github/motorhead-readme.md`, `raw/github/qdrant-readme.md`, `raw/github/chromadb-readme.md`, `raw/github/supermemory-readme.md`, `raw/github/reme-readme.md`.

### Production-pattern articles (highest signal)

- `raw/articles/how-ai-agents-remember.md` (Galarza, Feb 2026) — OpenClaw markdown + 4-mechanism pattern.
- `raw/articles/memory-engineering-agents.md` (Mjgmario, Jan 2026) — constraint triangle, memory controller, ADD/UPDATE/DELETE/NOOP.
- `raw/articles/letta-memory.md` — MemFS git-backed memory.
- `raw/articles/mem0-production-memory.md` — Mem0 paper architecture.
- `raw/articles/state-of-ai-agent-memory-2026.md` — Mem0's state-of-field with integration ecosystem.
- `raw/articles/langgraph-memory-production.md` — LangGraph BaseStore, namespaces, hot/background patterns.
- `raw/articles/episodic-memory-missing-piece.md` (Pink et al., Feb 2025) — five episodic properties.
- `raw/articles/synapse-episodic-semantic-memory.md` — spreading activation architecture.
- `raw/articles/locomo-long-term-conversational-memory.md` — 35-session benchmark.
- `raw/articles/longmemeval-benchmark-long-term-memory.md` — 500-question, 5-type benchmark.
- `raw/articles/redis-agent-memory-stateful-systems.md` — Redis warm-tier + LangGraph checkpointing.
- `raw/articles/pgvector-semantic-memory.md` — production HNSW tuning.

---

## 17. Final synthesis — what would the next-phase design probably look like

A senior engineer designing AGH's memory v2 from this corpus would likely converge on:

1. **A four-type taxonomy** (working / episodic / semantic / procedural) with file-first storage for the latter three (markdown-in-SQLite-content or git-backed dir) and SQLite + sqlite-vec + FTS5 for indexed retrieval.
2. **A four-axis scope** (`user_id`, `agent_id`, `session_id`, `app_id/org_id`) plus `source_actor` and `purpose`, composable, namespace-prefixed.
3. **An explicit memory controller** at the hook layer enforcing ADD/UPDATE/DELETE/NOOP with NOOP as default, gated by an A-MAC-style scoring function (future utility, factual confidence, semantic novelty, recency, content-type prior).
4. **A multi-stage retrieval pipeline** (route → candidate → expand → prioritize → rerank → package → inject) with hybrid BM25 + dense + optional graph traversal and Reciprocal Rank Fusion.
5. **Explicit lifecycle hooks** at session_start, pre_reasoning, post_tool_call, pre_compaction (silent flush), step_count_threshold, idle_consolidation, session_end, user_remember.
6. **Bi-temporal validity** (`valid_from`, `valid_to`, `tx_time`) on at least the semantic tier; never delete on supersedence — mark and link.
7. **Provenance links** from semantic facts back to source episodes (`sourceEpisodeIds[]`); separate facts from beliefs (Hindsight Opinion Network with confidence scores).
8. **Tiered persistence + TTL** (in-context Tier 0; SQLite session_state Tier 1 with TTL; SQLite memory Tier 2; SQLite structured Tier 3; blob Tier 4) with promotion/demotion policies per memory type.
9. **Privacy-by-architecture** — local-first by default (per AGH's local-first runtime stance), purpose binding at write time, soft-delete + audit log + scheduled hard-delete, deletion handler registry.
10. **Agent-manageable surfaces** — full CRUD via CLI (`/remember`, `/memory`, `/init`, `/sleeptime`, `/doctor`), HTTP/UDS REST, MCP server. Every memory operation must be agent-driveable, not just web-UI-driveable.
11. **Reflection / consolidation as first-class subagent jobs** — step-count and compaction-event triggers; run in background via dedicated subagent (Letta MemFS git-worktree pattern adapts to AGH's session model).
12. **Skill library separately** for procedural memory — `(description, code, embedding)` triples with verification gate (canary tests for code skills; LLM critique with structured rubric for prompt skills) and compositional reuse.

**The single most undervalued operation is NOOP**, and the single hardest problem is **maintenance over time** — the corpus is unanimous that storage choice barely matters compared to write discipline and consolidation policy.

---

*End of analysis. Read-only — no code modified, no git operations performed.*
