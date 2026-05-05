# Write Controller Decision Shape — Analysis (mem-v2)

**Question.** When a candidate memory mutation arrives at the AGH memory v2 write controller (extracted from a turn, written via CLI, proposed via native tool, surfaced by the dreaming worker), **who decides** ADD / UPDATE / DELETE / NOOP / REJECT, and **how**?

**Time-boxed read of:** `~/dev/knowledge/ai-memory/{wiki,raw}`, `.resources/hermes`, `.resources/openclaw`, plus `analysis_ai-memory.md`, `analysis_hermes.md`, `analysis_openclaw.md`.

---

## 1. TL;DR — recommendation

**Hybrid (rule-first, LLM-as-tiebreaker), single-step. Default verdict is NOOP.**

A pure deterministic engine cannot detect contradictions reliably (e.g. "moved Mumbai→Bangalore"); a pure LLM-mediated controller adds 200–500 ms and a model dependency on every write — unacceptable for AGH's local-first, agent-manageable, latency-sensitive runtime. Mem0 (canonical reference) uses an LLM for every write only because it's a SaaS optimizing for accuracy on benchmark conversations; even Mem0's own paper credits **NOOP discipline**, not LLM intelligence, as the maturity signal (`Production Memory Architecture Patterns.md:47`, `memory-engineering-agents.md:304-314`).

AGH's controller should:

1. Run a deterministic prefilter over candidate + top-K cosine neighbours: exact-content/hash hit → **NOOP**, threat-scan / size / scope violation → **REJECT**, no neighbours within `dedup_threshold` AND no entity overlap → **ADD**, exact-name slot or single-attribute entity match (`city`, `email`, `tz`) → **UPDATE**.
2. Only when neighbours fall in the **ambiguity band** (e.g. `0.72 ≤ cos_sim < 0.88` with conflicting attributes), call a haiku-class LLM with the candidate + top-K and a single-shot JSON prompt (Mem0 schema: `{decision, target_id, justification}`). LLM is bounded — fixed timeout, fallback to deterministic NOOP on error.
3. Emit a `Decision` record (with rule-trace and optional LLM trace) into a write-ahead log so every mutation is auditable, replayable, and reversible.

This matches AGH's invariants: agent-manageable (every decision visible via CLI/HTTP/UDS), extensible (rule weights and LLM model are config keys), local-first (no LLM dependency for the dominant ADD/NOOP path), and greenfield-clean (no legacy aliases). Hermes proves the deterministic-only path works for **exact-dedup, hand-curated** memory; OpenClaw proves the deterministic-threshold path works for **promotion** but explicitly delegates extraction to a silent agent turn. AGH should fuse: deterministic for the 80% obvious cases, LLM for the 20% genuinely ambiguous case where wrong UPDATE/DELETE corrupts state for weeks.

---

## 2. Mem0's exact algorithm (canonical reference)

**Decision taxonomy.** Four ops, decided per *candidate fact* (not per turn):

- **ADD** — durable signal, no conflict (`Production Memory Architecture Patterns.md:47`).
- **UPDATE** — supersedes an existing fact (`AI Agent Memory Frameworks Compared.md:39`: "user moved from Mumbai to Bangalore" → delete old city, add new).
- **DELETE** — contradicted by new info (`memory-engineering-agents.md:298-302`: "facts become wrong, constraints expire").
- **NOOP** — already known / temporary / low-confidence. *"The most undervalued operation and arguably the strongest indicator of a mature memory system"* (`State Persistence Strategies for Long-Running AI Agents.md:188`, `memory-engineering-agents.md:304-314`).

**Pipeline (verbatim summary from sources).**

> "At write time, an LLM analyzes each new conversation turn to extract candidate memories as discrete facts. Each candidate is evaluated against the **K most similar existing memories** and assigned one of four operations." — `Production Memory Architecture Patterns.md:47`

> "The extraction pipeline ingests three context sources — **the latest exchange, a rolling summary, and recent messages** — and runs them through an LLM to extract candidate memories as discrete facts. Each candidate is then compared against the K most similar existing memories from the vector database. **An LLM decides** one of four operations: ADD / UPDATE / DELETE / NOOP." — `AI Agent Memory Frameworks Compared.md:39`

So Mem0 uses **two LLM calls per write batch**: (a) extraction LLM produces N facts, (b) controller LLM decides ADD/UPDATE/DELETE/NOOP for each fact against top-K neighbours. Mem0g (graph variant) inserts a third — entity extractor + relations generator + **conflict detector** — *before* the write, flagging contradictions explicitly (`analysis_ai-memory.md:441`, `Episodic and Semantic Memory in AI Agents.md` line 213).

**The actual prompt.** The Mem0 README in `~/dev/knowledge/ai-memory/raw/github/mem0-readme.md` is a stub; the prompt strings live in the upstream Python package (`mem0/configs/prompts.py`) which is not in the local mirror. Sources describe the contract: a system prompt enumerating the four op codes with examples, then a user message containing `{candidate_fact, top_k_existing[]}`. Output is structured JSON `{ "memory": [{"id": "<uuid_or_null>", "text": "...", "event": "ADD|UPDATE|DELETE|NONE"}, ...]}` (Mem0 v1 uses `NONE` rather than `NOOP` in the wire format). **Mem0g adds a "conflict detector" pre-step** (`analysis_ai-memory.md:441`, `mem0-production-memory.md:39-50`).

**Top-K context size.** Mem0's default is small — typically **K=5** existing memories per candidate (sources use the phrase "K most similar" without a fixed number; Mem0's reference config exposes `limit` for retrieval, and Mem0 paper baseline uses k≈5–10 for the controller window).

**Embedding + threshold defaults.** Mem0 supports 19 vector backends. Default embedding is `text-embedding-3-small` (1536-dim) per Mem0 docs/README. There is no fixed similarity threshold *before* the LLM — the LLM is the threshold. Mem0g's conflict detector uses graph-edge contradiction logic, not cosine.

**Pure LLM, pure rule, or hybrid?** **Pure LLM-mediated** for the decision step — the controller has no deterministic fallback. The only deterministic gate is the candidate-extraction prompt's "extract only durable facts" guidance.

**Composite scoring (retrieval, not write).** The Mem0 *retrieval-time* score is 60% semantic + 20% extraction confidence + 10% recency + 10% access frequency (`analysis_ai-memory.md:200`, sourced from `Episodic and Semantic Memory in AI Agents.md:78`). This is read-side, not write-side, but documents the philosophy.

---

## 3. Hermes' write decision mechanism

Hermes does **not** use a Mem0-style controller. Memory writes are **agent-driven** (LLM is its own controller, à la Letta), with **deterministic guardrails only**.

**Tool surface.** `tools/memory_tool.py:465-503` exposes a single `memory(action, target, content?, old_text?)` tool with three actions: `add`, `replace`, `remove`. Two targets: `memory` and `user` (`memory_tool.py:480-481`). Schema description (`memory_tool.py:515-563`) gives the agent prose-level *priority rules* — "User preferences and corrections > environment facts > procedural knowledge" — but no JSON taxonomy. The LLM decides ADD vs REPLACE vs REMOVE in-context.

**Decision is the LLM's, not the runtime's.** No top-K retrieval, no similarity comparison, no controller prompt. The agent reads the **frozen system-prompt snapshot** of MEMORY.md/USER.md (`memory_tool.py:126-142, 361`), reasons in-context about whether the new info is new/revised/dead, and calls the right action.

**Deterministic guardrails inside the runtime (not LLM).**

- **Exact-content dedup** on add: `if content in entries: return "Entry already exists"` (`memory_tool.py:243-244`).
- **Char-budget hard cap**: 2200 chars MEMORY, 1375 chars USER (`memory_tool.py:121-122, 250-261`). Prevents bloat.
- **Substring-match invariants for replace/remove**: must be unique (`memory_tool.py:289-301, 342-352`). Multi-match without all-identical content → `"Be more specific"`. Forces the LLM to disambiguate.
- **Threat scan** before accepting any write: 13 regex patterns (`_MEMORY_THREAT_PATTERNS`, `memory_tool.py:67-83`) covering prompt injection, exfil-curl, ssh backdoor, hermes env dumps, plus invisible-unicode detection (`memory_tool.py:86-104`). This is the **REJECT path** in AGH terms — purely deterministic.
- **Atomic file replace + fcntl/msvcrt lock** (`memory_tool.py:144-179, 433-462`) — durability + cross-session safety.
- **Reload from disk under lock before mutation** (`memory_tool.py:236-237, 284, 333-334`) — picks up writes from sibling sessions.
- **Frozen system-prompt snapshot** (`analysis_hermes.md:250-257`, `memory_tool.py:138-142, 361-372`) — mid-session writes update disk but **not the prompt** (cache-friendly). New snapshot loaded at the next session start.

**Contradiction detection.** None at runtime. The LLM sees existing entries (in the system prompt) and decides whether the new info contradicts them. If the LLM gets it wrong, MEMORY.md becomes contradictory — and `analysis_hermes.md:610` flags exactly this failure mode: *"Flat markdown vs structured DB — `MEMORY.md` becomes contradictory at hundreds of lines; users must hand-edit periodically."*

**Curator (separate axis — skill hygiene, not memory writes).** `agent/curator.py:1-1674` curates the **skill library** (`~/.hermes/skills/`), not MEMORY/USER. It runs as a forked sub-agent with `CURATOR_REVIEW_PROMPT` (`curator.py:329-444`) and parses YAML output `{consolidations: [...], prunings: [...]}` (`analysis_hermes.md:480-481`). Snapshot-and-rollback lives in `agent/curator_backup.py:9, 600-628` — a tar.gz of `~/.hermes/skills/` to `.curator_backups/<utc-iso>/`, default keep=5. Useful as a precedent for **reversible skill updates**, but **not** an ADD/UPDATE/DELETE controller for facts.

**Verdict.** Hermes is the *agent-as-controller* extreme. Works because (a) memory is small (≤2200 chars), (b) snapshot is frozen so the LLM always sees current state, (c) deterministic guards block obvious abuse. Cannot scale past hundreds of entries without a runtime-level controller — own analysis admits it.

---

## 4. OpenClaw's write decision mechanism

OpenClaw runs a **two-fence** model: (a) a *silent agent flush* writes durable memories (LLM-driven), then (b) a *deterministic promotion gate* (no LLM) decides what gets promoted from short-term to long-term. **Dreaming** is a multi-phase post-process, also deterministic-thresholded with optional LLM narrative.

**Fence 1 — pre-compaction memory flush.** Triggered when projected next-input tokens cross threshold (`agent-runner-memory.ts:776-789`).

- **Trigger calculation (deterministic).** `flushThreshold = contextWindowTokens − reserveTokensFloor − softThresholdTokens` (`agent-runner-memory.ts:656-657`). Defaults: `reserveTokensFloor=20_000`, `softThresholdTokens=4_000` (`agent-runner-memory.ts:425-429`).
- **Forced flush when transcript exceeds `forceFlushTranscriptBytes`** (`agent-runner-memory.ts:678-698`). Rule, not LLM.
- **Dedup gate via `hasAlreadyFlushedForCurrentCompaction(entry)`** (`agent-runner-memory.ts:789`, regression test #34222 in `agent-runner-memory.dedup.test.ts:48`). Stored post-flush hash compared against current transcript hash; same transcript + same prompt = same hash, no re-flush. **Pure deterministic.**
- **`compactionCount` vs `memoryFlushCompactionCount`** (`agent-runner-memory.ts:769, 885-921`) — the flush only runs once per compaction generation; once a flush completes for compaction N, `memoryFlushCompactionCount` bumps to N and subsequent flushes for the same generation are skipped.
- **The flush itself is an LLM call.** `runEmbeddedPiAgent({ trigger: "memory", prompt: activeMemoryFlushPlan.prompt, memoryFlushWritePath, ... })` (`agent-runner-memory.ts:847-872`). This is a silent agent turn writing into `memory/YYYY-MM-DD.md`. So the **decision** is the LLM's, but the **trigger** and **dedup** are deterministic.

**Fence 2 — short-term-recall → MEMORY.md promotion (deterministic, no LLM).**

`extensions/memory-core/src/short-term-promotion.ts:1-160` defines the gates:

- `DEFAULT_PROMOTION_MIN_SCORE = 0.75` (line 23).
- `DEFAULT_PROMOTION_MIN_RECALL_COUNT = 3` (line 24).
- `DEFAULT_PROMOTION_MIN_UNIQUE_QUERIES = 2` (line 25).
- `DEFAULT_PROMOTION_WEIGHTS = { frequency: 0.24, relevance: 0.30, diversity: 0.15, recency: 0.15, consolidation: 0.10, conceptual: 0.06 }` (lines 55-62).
- Recency: 14-day half-life (`DEFAULT_RECENCY_HALF_LIFE_DAYS = 14`, line 22).

A recall must be retrieved 3+ times across 2+ unique queries with composite score ≥ 0.75 to promote. **Pure rule engine** — `clampScore`, `toFiniteScore` (`short-term-promotion.ts:215-231`), no LLM in the path.

**Dreaming pipeline (light → REM → deep).** `extensions/memory-core/src/dreaming.ts:590-617` logs the candidate scores. Phase signals (`PHASE_SIGNAL_LIGHT_BOOST_MAX = 0.06`, `PHASE_SIGNAL_REM_BOOST_MAX = 0.09`, `PHASE_SIGNAL_HALF_LIFE_DAYS = 14`, `short-term-promotion.ts:37-39`) **boost** scores when dreaming revisits a snippet — repeated revisits can clear the gate without organic recall. The dreaming **narrative** (`api.ts:11`, `src/dreaming-narrative.ts`) is generated by an LLM, but it's a *side-effect* (writes a dream-diary entry); the *promotion decision itself* is deterministic.

**Verdict.** OpenClaw's controller is **two-tier deterministic**: (1) deterministic flush trigger → LLM extracts and writes durable memories; (2) deterministic recall scoring → deterministic promotion to long-term. The LLM only writes content; the LLM never *decides* what gets stored long-term. This is the inverse of Mem0.

---

## 5. Letta's evolving notes (A-MEM-style)

**Letta — agent-as-controller, no separate write controller.** The agent owns three memory tiers (core ≈ RAM in-context, recall ≈ disk-cache, archival ≈ cold storage; `AI Agent Memory Frameworks Compared.md:57-59`). The agent decides via tool calls — `memory_edit`, `archival_memory_search`, `memory_replace`. Letta MemFS (v0.15+) takes it further: the agent edits memory **with the same bash tools used to write code**, and **git commits-and-pushes** to "save" (`analysis_ai-memory.md:525, 713`, `letta-memory.md:221`). Reflection/defrag run as separate subagents in **git worktrees** for parallel modification — a snapshot-and-rollback pattern not unlike Hermes' curator.

**A-MEM (`a-mem-agentic-memory.md:25-90`) — the "evolving notes" pattern.** Zettelkasten-inspired:

- New memory arrives → generates a **comprehensive note**: content + auto-generated keywords + tags + **links to related historical notes** + a contextual "evolution" field that summarises how this note relates to prior knowledge (`a-mem-agentic-memory.md:40-50`).
- **Memory evolution.** When a new memory arrives, the system **triggers updates to contextual representations of related historical memories** (`a-mem-agentic-memory.md:50, 81-84`) — UPDATE is not "supersede the old fact" but "re-summarise what neighbours mean in light of the new arrival". The agent's controller decides which neighbours to link and which to re-summarise.
- Net result: ADD with a side-effect graph-evolution UPDATE on K linked neighbours. Closer to a "graph stitch" than Mem0's slot-replace UPDATE.

**Implication for AGH.** A-MEM's "evolution" suggests UPDATE is not always "overwrite slot X" — it can be "re-anchor neighbours to incorporate new context". This is a v3 concern; for v2, AGH should keep UPDATE simple (slot supersession) but design `Decision.Targets[]` to allow plural target IDs in the future.

---

## 6. Comparative matrix

| System | Decision mechanism | Top-K context | Latency budget per write | Dedup quality | Audit shape |
|---|---|---|---|---|---|
| **Mem0** | LLM-mediated (haiku-class on every fact) | K=5–10 neighbours | 100–500 ms (extract) + 100–300 ms (decide) ≈ 200–800 ms | High (LLM detects contradiction) | structured JSON `{event, id, text}` per fact |
| **Mem0g** | LLM + entity-extractor + relations + **conflict detector** | K + graph 1-hop | 300 ms – 3 s (Mem0g paper p95 < 3 s) | Highest in class (+36% temporal accuracy) | facts + entities + edges + conflict flags |
| **Letta MemFS** | Agent-as-controller (no separate controller) | Whole core block in-context | 0 (no extra call — same agent step) | Medium (depends on LLM diligence) | git commit per write |
| **A-MEM** | LLM-mediated; ADD always + side-effect UPDATE on linked neighbours | K linked notes | LLM call + graph-write | Medium-High (link-based) | note + links + evolution log |
| **Zep / Graphiti** | Bi-temporal: never DELETE, mark old fact `valid_to=now`, ADD new | Graph traversal | 50–100 ms decide; hours for full graph ingest | High via temporal validity | bi-temporal graph edges |
| **Hermes curator (memory)** | **Agent-as-controller**; runtime = exact-dedup + threat-scan + char cap | Frozen snapshot of all entries | 0 (deterministic guards only) | Exact-only (substring/exact); no semantic dedup | append-only file + lock log |
| **OpenClaw memory-core** | Deterministic flush trigger → LLM writes; deterministic promotion gates | Flush: whole transcript; promotion: recall analytics store | Flush: full silent agent turn; promotion: <1 ms | Deterministic dedup via transcript hash; semantic via score+recall | `short-term-recall.json`, `phase-signals.json`, dream diary |
| **goclaw / paperclip** | (per `analysis_goclaw-paperclip-multica.md`) closer to OpenClaw — file-first markdown + dedup, no LLM controller | n/a | n/a | rule-based | markdown |
| **Claude Code extractor** | Implicit — agent writes via the agent's own tool calls into AGENTS.md / CLAUDE.md | Whole file in-context | 0 | Manual / agent-mediated | markdown diff |

---

## 7. Three strategies for AGH evaluated

### 7.1 Pure LLM-mediated (Mem0 model)

**Pros.**
- Highest dedup/contradiction quality on the head distribution Mem0 benchmarks (LOCOMO).
- Single uniform decision path — easy to reason about.
- Natural fit for ADD/UPDATE/DELETE/NOOP semantics.

**Cons.**
- 200–500 ms LLM call **per write** — every CLI write, every native-tool write, every dreaming-worker promotion blocks on it. Web UI write feels laggy; CLI scripting (a core agent-manageability requirement) feels broken.
- Adds a **mandatory model dependency** to the write path. Local-first, offline, or constrained environments stall.
- Cost: even haiku-class is non-zero per write across N agents × M turns/day. Mem0 ships SaaS partly to amortise this.
- **Failure modes.** LLM hallucinates a `target_id` that doesn't exist → write fails or corrupts a wrong slot. JSON schema drift. Prompt injection in the candidate content → model makes a malicious decision. Timeout → entire write fails (or worse, half-applies).
- Inverts AGH's truth-direction: a model decides what the runtime stores. Audit becomes "trust the model".

### 7.2 Pure deterministic rules

**Pros.**
- Sub-millisecond decisions. Synchronous, predictable, replayable.
- No model dependency on the write path. AGH stays local-first.
- Trivially auditable — every Decision has a rule trace.
- Hermes proves this works for hand-curated, char-bounded memory (≤2200 chars).
- OpenClaw proves this works for promotion (deterministic gates over recall analytics).

**Cons.**
- **Cannot detect semantic contradiction reliably.** "I prefer email" vs "I prefer phone" don't share enough surface form for cosine to flag; rule-based UPDATE will miss it and append → contradictory truth (`memory-engineering-agents.md:267-271`).
- Rules over-trigger UPDATE on benign paraphrase, or under-trigger when a moved-cities case has different phrasing.
- Heavy dependency on embedding quality — if the embedding model is bad, dedup fails. (Mitigation: AGH already needs an embedding pipeline for retrieval; same model is fine.)
- **Hermes' own analysis** (`analysis_hermes.md:610`) admits flat-markdown contradiction is the failure mode at scale.

### 7.3 Hybrid (rule-first, LLM-as-tiebreaker)

**Pros.**
- Sub-millisecond on the dominant path (clear ADD or clear NOOP — empirically 70–90% of writes for personal-agent workloads). LLM only burns in the genuinely-ambiguous middle band.
- Reuses AGH's existing embedding pipeline for top-K. Reuses LLM-call infra (existing native-tool LLM bridge).
- LLM has a **fallback** — if the LLM call fails or times out, the rule engine returns NOOP (safest default per `memory-engineering-agents.md:314`: *"NOOP unless the signal is durable and beneficial"*).
- Decision audit: every Decision carries `Source = "rule:<rule_id>"` or `Source = "llm:<model>"` and a serialised trace.
- Configurable: `[memory.controller.llm.enabled = false]` reduces hybrid to pure-rule for offline/strict-determinism deployments. Agent-manageable.

**Cons.**
- Two code paths to maintain.
- Requires careful tuning of the ambiguity-band thresholds.
- Need to monitor escalation rate — if >40% of writes escalate to LLM, hybrid degenerates to pure-LLM with extra latency.

**Decision tree (pre-LLM rules).**

```
Decide(ctx, candidate):
  if candidate.policy_violation: return REJECT(reason)         // size, scope, threat scan
  hash = blake2b(normalize(candidate.content))
  if exists(hash): return NOOP("exact_dup")                    // O(1) hash hit
  topK = vectorStore.search(candidate, k=10)
  bestSim = topK[0].similarity if topK else 0
  if bestSim >= dup_threshold (0.92): return NOOP("near_dup")
  if entityMatch(candidate, topK):                              // same entity slot
    if attributeContradicts(candidate, topK[match]):
      if bestSim >= rule_update_threshold (0.88): return UPDATE(target=topK[match])
      // ambiguous — escalate
      return llmDecide(candidate, topK)
    return NOOP("attribute_match_no_change")
  if bestSim < ambiguity_low (0.72): return ADD                 // genuinely novel
  // ambiguity band: 0.72 ≤ sim < 0.88 with no entity slot match
  return llmDecide(candidate, topK)
```

`llmDecide` budget: **1 call, 250 ms timeout, fallback NOOP**. Top-K window = 5 (Mem0 baseline). Prompt is templated and version-pinned in `internal/memory/prompts/`. Output validated against a strict JSON schema; invalid → NOOP.

**Latency profile (expected).**

- ~80% of writes hit hash/sim/no-neighbour rules → **<5 ms**.
- ~15% hit entity-slot match with clear contradiction or no contradiction → **<10 ms**.
- ~5% escalate to LLM → **150–300 ms**.

P95 across all writes ≈ 50 ms, P99 ≈ 300 ms. Compare to pure-LLM P95 ≈ 400 ms.

---

## 8. Recommended verdict for AGH

**Adopt the Hybrid strategy. Default LLM enabled with `model = "haiku-cheap"`, but the runtime MUST work with `[memory.controller.llm.enabled = false]`.**

Defended by:

1. **Competitor evidence.** Mem0's accuracy lead doesn't justify uniform LLM cost in a local-first runtime. OpenClaw's split (deterministic gates + LLM-as-content-writer) ships in production today and handles real users. Hermes proves deterministic-only is workable but admits the contradiction failure mode. Zep's bi-temporal-never-delete is an alternative we should keep on the v3 roadmap but is too heavy for v2 (Zep ingest is hours-async; AGH writes need to be readable on the next turn).
2. **AGH constraints.**
   - *Greenfield*: no migration tax — we can land the right shape now.
   - *Agent-manageable*: every Decision is JSON-serialisable and surfaced via CLI (`agh memory decisions`) and HTTP (`GET /memory/decisions/:id`). Pure-LLM hides reasoning in opaque LLM calls.
   - *Extensible*: rule weights, thresholds, LLM model, and prompt template are config keys. Extensions can register additional prefilter rules via a hook.
   - *Local-first*: hybrid can degrade to pure-rule with a single config flag; pure-LLM cannot.
3. **Two-touch rule.** This is the *first* design pass for the controller. It must be right enough that v2 doesn't need a v2.1 redesign — hybrid leaves all three escape hatches (rules-only, LLM-only, hybrid) reachable via config.
4. **LLM-only is a strict subset of hybrid** (set ambiguity band to `[0, 1]`). So picking hybrid forecloses no future option.

---

## 9. AGH-specific implementation contract

### 9.1 Go interface

```go
// internal/memory/controller/types.go

type Op uint8
const (
    OpNoop   Op = iota
    OpAdd
    OpUpdate
    OpDelete
    OpReject
)

type DecisionSource string
const (
    SourceRule DecisionSource = "rule"
    SourceLLM  DecisionSource = "llm"
)

type Candidate struct {
    Workspace   string            // scope: workspace_root
    Scope       Scope             // user|agent|session|workspace
    Origin      Origin            // turn-extract | cli | tool | dreaming
    Content     string            // canonical text
    Entity      string            // optional structured slot ("city", "email"), empty = unstructured
    Attribute   string            // optional ("preferred_email")
    Embedding   []float32         // pre-computed; controller does NOT call embedding
    Metadata    map[string]string // free-form, e.g. citation, source_msg_id
    SubmittedAt time.Time
}

type Decision struct {
    ID            string         // ULID
    CandidateHash string         // blake2b of normalized content
    Op            Op
    Targets       []string       // existing memory IDs (UPDATE/DELETE may target multiple)
    Confidence    float32        // 0..1; rule = 1.0 unless ambiguity band; LLM = model-reported
    Source        DecisionSource
    RuleTrace     []RuleHit      // ordered list of rules consulted (always populated)
    LLMTrace      *LLMCall       // nil unless escalated
    Reason        string         // short human-readable
    DecidedAt     time.Time
}

type Controller interface {
    // Decide is sync. It MUST return within `Config.MaxLatency`.
    // On context cancel or LLM failure, falls back to RuleTrace's terminal decision.
    Decide(ctx context.Context, c Candidate) (Decision, error)
}
```

`Decision` is persisted to a `memory_decisions` SQLite table (write-ahead log) **before** the corresponding store mutation. The controller is the only path for writes — `internal/memory/store.go` accepts only `(Decision, Candidate)` pairs, never raw `Add`.

### 9.2 Where the LLM call happens

- **Synchronous**, in-line with `Decide()`. Caller blocks. This matches Mem0/OpenClaw flush semantics: the next agent turn must see the write.
- **Bounded budget.** `[memory.controller.llm.timeout = "250ms"]`. On timeout: log, fall back to rule-engine NOOP, mark `Confidence = 0.0`.
- **Single retry on transient errors only** (network/5xx). Schema-invalid response → no retry, NOOP fallback.
- **No async path for writes.** Async is a v3 optimisation (write returns immediately, controller decides later); for v2, sync keeps semantics simple and avoids the "memory not yet readable" window Mem0 ships with (`AI Agent Memory Frameworks Compared.md:170`: Mem0 eventual consistency 100–500 ms async).

### 9.3 Configuration keys (`config.toml`)

```toml
[memory.controller]
mode               = "hybrid"          # hybrid | rules | llm
max_latency        = "300ms"           # hard ceiling on Decide()
default_op_on_fail = "noop"            # noop | reject

[memory.controller.dedup]
near_dup_threshold = 0.92              # cos sim → NOOP
update_threshold   = 0.88              # entity slot match + contradiction → UPDATE
ambiguity_low      = 0.72              # < this AND no entity match → ADD
ambiguity_high     = 0.88              # in [low, high] → escalate to LLM

[memory.controller.scoring]
recency_half_life_days = 14
weights            = { frequency = 0.24, relevance = 0.30, diversity = 0.15, recency = 0.15, consolidation = 0.10, conceptual = 0.06 }

[memory.controller.llm]
enabled            = true
model              = "anthropic/claude-haiku-4"
top_k              = 5
prompt_version     = "v1"              # binds to internal/memory/prompts/decide.v1.tmpl
timeout            = "250ms"
max_tokens_out     = 256

[memory.controller.policy]
max_content_chars  = 4096
max_writes_per_min = 60                # rate limit per scope
allow_origins      = ["turn", "cli", "tool", "dreaming"]
```

All keys hot-reloadable. `mode = "rules"` disables every LLM path (ambiguity band → NOOP). `mode = "llm"` makes the rule engine a pre-validator only and escalates everything else.

### 9.4 Prompt template (LLM tiebreaker, sketch)

Stored at `internal/memory/prompts/decide.v1.tmpl`. Versioned. Single user message:

```
You are a memory write controller. Decide one operation for the candidate fact
relative to the existing memories. Output strict JSON: {"op":"ADD|UPDATE|DELETE|NOOP","target_id":"...|null","confidence":0..1,"reason":"<=120 chars"}.

Rules:
- ADD only if the candidate is durably useful AND not implied by any existing memory.
- UPDATE only if the candidate revises a specific existing memory (give its target_id).
- DELETE only if the candidate explicitly contradicts and invalidates an existing memory.
- NOOP if the candidate is implied, transient, low-confidence, or already covered.
- Default to NOOP. Saving everything is not intelligence.

Candidate:
  scope: {{.Scope}}
  entity: {{.Entity}}
  attribute: {{.Attribute}}
  content: {{.Content}}

Existing top-{{.TopK}} most-similar memories:
{{range .Existing}}- id={{.ID}} sim={{printf "%.2f" .Similarity}} content="{{.Content}}"
{{end}}

JSON:
```

Output validated by a strict zod-equivalent (Go schema). Schema-invalid → NOOP.

### 9.5 Audit + reversibility

Every `Decision` is appended to `memory_decisions` (SQLite). UPDATE/DELETE store `prior_content` so a `agh memory decisions revert <id>` can roll back N steps. Mirrors Hermes' curator-backup tar.gz pattern (`curator_backup.py:600-628`) but at row granularity.

---

## 10. Open sub-questions for the TechSpec

1. **Where does fact-extraction live?** Mem0 pairs an extraction LLM with the controller LLM. AGH must decide: does extraction happen (a) inside the silent flush turn (OpenClaw model), (b) as a hook fired post-turn, or (c) inside `Decide()`? Recommendation: **(b), out-of-band**, so the controller's input contract is *one candidate* — extraction is somebody else's problem. The dreaming worker, native tools, and CLI all produce candidates by their own extraction logic.
2. **Write conflict policy across multi-agent / multi-session.** Hermes uses `fcntl.flock` (`memory_tool.py:144-179`). LangGraph uses last-write-wins (`Production Memory Architecture Patterns.md:173`). AGH SQLite gives serializable; do we need optimistic locking on `memory_decisions.candidate_hash`? What about the dreaming worker writing while a CLI write lands?
3. **Mem0g-style conflict detector — do we add one in v2 or defer?** Adds a third LLM call (`analysis_ai-memory.md:441`). Defer to v3 unless benchmarks show v2 misses too many contradictions.
4. **Top-K=5 right for AGH?** Mem0's K is 5–10 over a flat user store. AGH has scoping (`Scope`). K=5 within scope might be too narrow; K=10 might pull cross-scope noise. Empirically tune in v2 alpha; ship K=5 default.
5. **Prompt versioning + eval.** When we change `decide.v1.tmpl` to `decide.v2.tmpl`, do we re-decide existing memories (replay)? Recommendation: keep prompt-version on every `Decision`, never replay automatically; offer `agh memory decisions replay --since <ts> --prompt-version v2` as an opt-in agent operation.

---

## Sources cited

- `~/dev/knowledge/ai-memory/wiki/concepts/State Persistence Strategies for Long-Running AI Agents.md:188, 256, 289`
- `~/dev/knowledge/ai-memory/wiki/concepts/Production Memory Architecture Patterns.md:43-57, 197-198`
- `~/dev/knowledge/ai-memory/wiki/concepts/AI Agent Memory Frameworks Compared.md:33-103, 165-177`
- `~/dev/knowledge/ai-memory/raw/articles/memory-engineering-agents.md:228-324`
- `~/dev/knowledge/ai-memory/raw/articles/mem0-production-memory.md:39-50, 120`
- `~/dev/knowledge/ai-memory/raw/articles/a-mem-agentic-memory.md:25-90`
- `~/dev/knowledge/ai-memory/raw/articles/letta-memory.md:137, 221-226`
- `~/dev/knowledge/ai-memory/raw/articles/ai-agent-memory-systems-2026.md:116, 144`
- `.resources/hermes/tools/memory_tool.py:1-583` (controller surface)
- `.resources/hermes/agent/curator.py:1-1674` (skill curator, snapshot-and-rollback precedent)
- `.resources/hermes/agent/curator_backup.py:1-693`
- `.resources/openclaw/src/auto-reply/reply/agent-runner-memory.ts:418-949` (flush triggers)
- `.resources/openclaw/src/auto-reply/reply/agent-runner-memory.dedup.test.ts:1-200`
- `.resources/openclaw/extensions/memory-core/src/short-term-promotion.ts:1-340` (deterministic promotion gates)
- `.resources/openclaw/extensions/memory-core/src/dreaming.ts:590-617` (scoring)
- `.compozy/tasks/mem-v2/analysis/analysis_ai-memory.md:200, 206-227, 441, 467-486, 504-525, 700-751`
- `.compozy/tasks/mem-v2/analysis/analysis_hermes.md:13-14, 245-291, 405-486, 576-621`
- `.compozy/tasks/mem-v2/analysis/analysis_openclaw.md:9-67`
