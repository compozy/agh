# mem-v2 Integration Decision Matrix

**Source corpus:** the eight per-source forensics under `.compozy/tasks/mem-v2/analysis/`
(`analysis_ai-memory.md`, `analysis_ai-harness.md`, `analysis_hermes.md`,
`analysis_openclaw.md`, `analysis_codex.md`, `analysis_openfang.md`,
`analysis_claude-code.md`, `analysis_goclaw-paperclip-multica.md`).
**Current AGH shape (grounding):** `internal/memory/types.go`, `internal/memory/store.go`,
`internal/CLAUDE.md` (Memory & Skills RFC section).
**Posture:** opinionated. Greenfield alpha — no compat shims, no aliases. Date 2026-05-04.

---

## 1. TL;DR

The corpus collapses to a small set of load-bearing decisions for AGH. Memory is **not a
vector store** but a **layered system** (transcript / session / curated / procedural /
provider) with its own write-side controller, lifecycle hooks, and consolidation worker.
Across all eight analyses the same patterns dominate: (a) **frozen-snapshot system-prompt
injection** of curated memory (Hermes §3.2; Codex `memory_summary.md`; Claude Code MEMORY.md
top-of-prompt; OpenClaw `prompt-section.ts`); (b) **closed memory taxonomy** with explicit
`WHAT_NOT_TO_SAVE` (Claude Code `memoryTypes.ts:183-194` — the single most-cited gate); (c)
**MEMORY.md as a thin index** with pointer entries plus topic files (Claude Code; OpenClaw;
Codex `MEMORY.md`); (d) **ADD/UPDATE/DELETE/NOOP controller discipline** and async background
writes (Mem0; goclaw `dreamingWorker`); (e) **pre-compaction memory flush** as a silent
agentic turn (OpenClaw `agent-runner-memory.ts`); (f) **subagent extractor with sandboxed
tool surface** (Claude Code `createAutoMemCanUseTool`; Codex two-phase memory pipeline); (g)
**SQLite + FTS5 + optional vector** is the right local-first primitive (Hermes,
OpenClaw, OpenFang); (h) **agent-callable read tools, agents do not write directly to the
durable curated layer** (goclaw — agents consume; consolidation pipeline writes); (i)
**dreaming worker with quantified recall scoring** is the most novel pattern in the corpus
and the one that makes the difference between a transcript dump and an evolving memory
(goclaw `scoring.go`; Letta `/sleeptime`); (j) **sub-agent forks share parent prompt cache
but are sandboxed by `isAutoMemPath`** (Claude Code §10) — read-only by default.

For AGH this implies a v2 design centered on: SQLite-first dual store
(curated markdown files + derived FTS/vec catalog), a typed memory controller
(`internal/memory/controller`), a `consolidation/dreaming` worker driven by the existing
hooks taxonomy, an explicit pre-compaction flush hook, an extractor sub-agent that runs
with a narrow tool whitelist, and a tight extension contract that agents reach via
CLI/HTTP/UDS — never via direct table writes. Hard cuts: delete the current
`internal/memory/types.go` "future" surfaces (`ContextRefResolver`, `ProviderHookRunner`)
and re-author them as v2 contracts; rename existing `MemoryTypeUser/...` constants to align
with the new closed taxonomy; drop the `EnsureSchema`-style table reconciliation that
mirrors OpenFang/OpenClaw's anti-pattern.

---

## 2. Design pillars (load-bearing invariants)

### P1. Layered memory, never one store
Five-tier model (transcript / session-state / curated / procedural / external-provider) is
the de-facto standard across `analysis_ai-memory.md` §2, `analysis_ai-harness.md` §1,
Hermes §1, Codex §2. Each layer has its own write protocol, lifecycle, and reader. AGH must
model these as orthogonal axes, not collapse them. The current AGH `Backend` interface only
covers the curated layer. v2 must split clearly: `EventStore` (transcript, already
`internal/store/sessiondb`), `SessionState`, `Curated`, `Procedural` (skills),
`Provider` (extension-supplied).

### P2. Frozen system-prompt snapshot for cache stability
Every harness that injects memory into the system prompt freezes it at session start
(Hermes `tools/memory_tool.py:_system_prompt_snapshot`; Codex injects `memory_summary.md`
once; Claude Code top-of-prompt `MEMORY.md`). Mid-session writes update disk but do **not**
mutate the prompt — they surface in the *next* session. This is what keeps Anthropic prompt
caching alive across an entire session. v2 MUST enforce this — the curated block is bit-stable
until the next session boot.

### P3. Closed taxonomy + WHAT_NOT_TO_SAVE
Claude Code `memoryTypes.ts:14-19` defines exactly four types
(`user|feedback|project|reference`) and explicitly forbids saving code patterns, git
history, debugging fixes, ephemeral state, or anything already in `CLAUDE.md` — even when
the user asks. AGH already adopted this taxonomy in `internal/memory/types.go`. v2 must add
a runtime-injected `WHAT_NOT_TO_SAVE` block to every extractor / write tool, plus an
"explicit user override is *not* a bypass" rule. Without this, memory devolves into a
transcript dump in two weeks.

### P4. MEMORY.md is a thin index, topic files carry content
Claude Code `MEMORY.md` is at most 200 lines / 25 KB; entries are one-line pointers
(`- [Title](file.md) — one-line hook`); silent truncation past line 201 is a documented
failure mode. Codex separates `memory_summary.md` (injected) from the rest of the
handbook (read on demand). OpenClaw's `MEMORY.md` is auto-loaded; daily logs and DREAMS.md
are not. v2 must keep injected content small (≤25 KB), recall full topic files via a
relevance side-query, never inject all topic files.

### P5. Controller-mediated writes (ADD/UPDATE/DELETE/NOOP)
Mem0 paper, `analysis_ai-memory.md` §5.1: every write must traverse a controller that
chooses among ADD / UPDATE / DELETE / NOOP. NOOP is "the strongest indicator of a mature
memory system." The current AGH Backend has `Write/Delete` but no
contradiction-detection layer. v2 needs `internal/memory/controller` that dedups against
the K most-similar existing entries before persisting and emits a typed
`MemoryWriteDecision` event.

### P6. Pre-compaction memory flush (silent agentic turn)
OpenClaw `agent-runner-memory.ts:runMemoryFlushIfNeeded` — when a session approaches the
context window limit, the system injects a silent agentic turn with prompt "store durable
memories now to memory/YYYY-MM-DD.md or reply NO_REPLY," runs it on the same model, then
proceeds to compaction. This reframes compaction as a *checkpoint* rather than a loss.
Highly cited (`analysis_ai-memory.md` §5.4; `analysis_ai-harness.md` §4). AGH must add a
`pre_compaction_flush` hook in the existing hooks taxonomy and dispatch it from the session
manager *before* compaction triggers.

### P7. Two-phase consolidation (episodic → semantic, async)
goclaw is the strongest reference: `episodicWorker → semanticWorker → dedupWorker →
dreamingWorker` driven by an internal event bus. Letta runs reflection sub-agents on
step-count/compaction-event triggers. Mem0 v1.0.0 made async writes the default. AGH already
has `internal/memory/consolidation` (per `internal/CLAUDE.md`); v2 must move it from "Time →
Sessions → Lock cascade" to a worker pipeline driven by typed hooks plus per-agent
debounce and quantified recall scoring.

### P8. Agents never write directly to the durable curated layer (read-only by default)
goclaw §1.4 is the cleanest model: agents call `memory_search` / `memory_get` /
`memory_expand`; consolidation workers do all writes. Claude Code uses a forked Sonnet
extractor with `createAutoMemCanUseTool` denying everything except read tools and `Edit`
inside `isAutoMemPath`. v2 must lock down: tools the LLM can call are read-only against
curated memory; writes are always mediated by the controller (extractor sub-agent or
explicit `/memory add` command from the operator). This stops poisoning and instruction
injection (cf. `analysis_ai-memory.md` §9.1).

### P9. Sonnet ranker recall (precision > recall)
Claude Code `findRelevantMemories.ts:18-24` runs a forked Sonnet side-query that
returns at most 5 filenames against a manifest of frontmatter descriptions. System prompt
explicitly says "if unsure, do not include." Hermes does FTS5 → LLM summarisation in two
stages (`analysis_hermes.md` §2.5). v2 must adopt: cheap candidate generator (FTS + vec) →
cheap ranker (small-model side-query OR weighted score) → top-K (≤5) packaged into
`<system-reminder>` blocks with stable headers (so prompt cache still hits between turns
where the memory list is identical).

### P10. Dreaming consolidation worker with recall feedback
goclaw `dreamingWorker` + `scoring.go` is the corpus's most novel piece: 4-component recall
score (`0.30·frequency + 0.35·relevance + 0.20·recency + 0.15·freshness`), per-agent
debounce (default 10 min), threshold gates (≥5 unpromoted, MinRecallCount=2), LLM synthesis
into `_system/dreaming/YYYYMMDD-consolidated.md`, `MarkPromoted` to close the loop. This is
goclaw's analogue of Letta's `/sleeptime`. AGH should adopt this in
`internal/memory/consolidation/dreaming.go` with the same scoring formula and explicit
freshness-for-cold-start.

### P11. File-based curated memory with hierarchical precedence
Codex AGENTS.md, Claude Code CLAUDE.md, OpenClaw `<workspace>/memory/`. Same pattern: walk
ancestors from `cwd` to the project root, concatenate root → leaf, deeper-overrides-shallower
encoded in the prompt (Codex `hierarchical_agents_message.md`). AGH already has the
agent-local override + workspace + global precedence chain in
`internal/CLAUDE.md` Memory & Skills section; v2 must keep it and add per-directory
`AGENTS.md` discovery (start/end markers per directory à la Codex
`user_instructions.rs:11-16`).

### P12. Sub-agent forks are read-only against the parent's memory
Claude Code `createAutoMemCanUseTool` denies all writes except inside the memory dir for the
extractor; AGH already has the "subagents are read-only" rule in repo CLAUDE.md. v2 must
make this an enforced runtime invariant in the memory layer: a sub-agent's session inherits
the parent's curated snapshot but cannot write through the controller.

---

## 3. Decision matrix

Verdict legend: **ADOPT** (copy directly), **ADAPT** (copy idea, AGH-shape it),
**REJECT** (do not import), **DEFER** (out of v2 scope, parking lot).

| # | Pattern | Source | Verdict | Why | AGH integration point | Risk |
|---|---|---|---|---|---|---|
| 1 | Closed 4-type taxonomy (`user/feedback/project/reference`) | `analysis_claude-code.md` §2 | ADOPT | already aligned; canonical taxonomy | `internal/memory/types.go` (kept) | low |
| 2 | `WHAT_NOT_TO_SAVE` runtime block + "save-even-if-asked" denylist | `analysis_claude-code.md` §2 | ADOPT | prevents transcript-dump rot; eval-validated | extractor prompt assembly + write-tool guard in `internal/memory/controller` | low |
| 3 | Frozen system-prompt snapshot of curated block | `analysis_hermes.md` §3.2 | ADOPT | mandatory for prompt caching | `internal/situation/service.go` snapshot at session boot | medium — must invalidate on `/memory reload` |
| 4 | MEMORY.md hard caps (200 lines / 25 KB) + truncation banner | `analysis_claude-code.md` §`memdir.ts` | ADOPT | tested at p100; banner makes silent failure visible | `internal/memory/store.go` `LoadPromptIndex` | low |
| 5 | Two-write paths (main agent + forked extractor) with cursor dedup | `analysis_claude-code.md` §3 | ADAPT | AGH wants extractor as a daemon-side hook, not a forked LLM call from the agent — use `on_post_response` hook handler | new `internal/memory/extractor` package | medium — needs sub-agent execution path |
| 6 | Sandboxed `isAutoMemPath` tool surface for extractor | `analysis_claude-code.md` §3 | ADOPT | the only sane way to run an LLM-driven writer | `internal/tools` policy gate keyed off the memory dir | medium — symlink hardening |
| 7 | Sonnet ranker recall (precision-over-recall side-query) | `analysis_claude-code.md` §`findRelevantMemories.ts` | ADAPT | use cheaper local model when configured; fall back to weighted score | `internal/memory/recall` package | medium — model dependency |
| 8 | Pre-compaction memory flush (silent agentic turn) | `analysis_openclaw.md` §1.5 | ADOPT | reframes compaction as checkpoint | new hook event `pre_compaction_flush` in `internal/hooks` + dispatch from `internal/session/manager_lifecycle.go` | medium — must not loop |
| 9 | Independent compaction model selection | `analysis_openclaw.md` §4 | ADOPT | lets ops use Haiku for compaction while Opus runs main | `config.toml` `agent.compaction.model` | low |
| 10 | Compaction-as-handoff framing (Codex `summary_prefix.md`) | `analysis_codex.md` §3 | ADOPT | tells receiving model to trust summary as evidence | compaction prompt template | low |
| 11 | `Compacted.replacement_history` as resume checkpoint | `analysis_codex.md` §3.5 | ADOPT | makes resume O(checkpoint-tail) | `internal/store/sessiondb` event variant + replay pruning | medium — schema change |
| 12 | Two-phase memory pipeline (Phase 1 per-rollout extract, Phase 2 sandboxed agent edits) | `analysis_codex.md` §1, §10 | ADAPT | goclaw-shape: replace Phase 2 sandboxed agent with the dreaming worker | `internal/memory/consolidation/{episodic,dreaming}.go` | medium — large surface |
| 13 | Dreaming worker with 4-component recall score | `analysis_goclaw-paperclip-multica.md` §1.6 | ADOPT | most novel piece in the corpus | `internal/memory/consolidation/dreaming.go` | medium — tuning constants |
| 14 | Recall feedback loop (`RecordRecall` updates `recall_score`) | goclaw `tools/memory.go:194-222` | ADOPT | closes consolidation loop | `internal/memory/store.go` after each `Search` | low |
| 15 | Per-agent debounce on consolidation triggers | goclaw §1.6 | ADOPT | prevents thundering-herd on event bursts | `internal/memory/consolidation` | low |
| 16 | Hybrid FTS5 + vector search with weighted merge | Hermes §2.5; goclaw §1.5 | ADOPT | beats either alone | `internal/store/globaldb` (catalog) — `sqlite-vec` + FTS5 | medium — `sqlite-vec` cgo footprint |
| 17 | Reciprocal Rank Fusion (RRF, k=60) | `analysis_ai-memory.md` §4.2 | ADAPT | use weighted-merge until eval shows RRF wins | recall package | low |
| 18 | A-MEM-style spreading activation / multi-attribute graph | `analysis_ai-memory.md` §4.4 | DEFER | overkill for v2; revisit when KG lands | — | high if adopted now |
| 19 | KG triple store (entities/relations) | OpenFang §2.2; goclaw §1.2 | DEFER | meaningful KG needs LLM extractor + dedup; not worth it for v2 | parking lot | high — schema, dedup, prompt cost |
| 20 | Per-channel + canonical session split | OpenFang §2.1, Hermes | REJECT | AGH already canonicalises per-session and per-workspace; redundant | n/a | n/a |
| 21 | Single-mutex SQLite (one `Arc<Mutex<Connection>>`) | OpenFang §2.1 | REJECT | blocks reads under contention; AGH already uses WAL + jittered helper pattern | n/a | n/a |
| 22 | Hermes-style `_execute_write` with WAL + IMMEDIATE + jitter retry | `analysis_hermes.md` §2.3 | ADOPT | proven pattern; AGH already has WAL but not standardised retry helper | `internal/store` shared writer helper | low |
| 23 | Periodic WAL checkpoint every N writes | Hermes | ADOPT | bounds WAL size | `internal/store` | low |
| 24 | Two FTS5 indexes (unicode61 + trigram) for CJK | `analysis_hermes.md` §2.1 | ADOPT | AGH must work for non-English operators | catalog DDL | low |
| 25 | Frontmatter-only relevance manifest (one line per file) | `analysis_claude-code.md` §`memoryScan.ts` | ADOPT | cheap input for ranker | `internal/memory/store.go::List` | low |
| 26 | YAML frontmatter (`name`, `description`, `type`) | Claude Code | ADOPT | AGH already does this | `internal/frontmatter` (existing) | low |
| 27 | Memory drift / freshness banner ("This memory is N days old") | Claude Code `memoryAge.ts` | ADOPT | makes staleness load-bearing for the model | recall packaging step | low |
| 28 | "Action-cue" header phrasing (eval-validated 3/3 vs 0/3) | Claude Code §`memoryTypes.ts` | ADOPT | tiny effort, big impact | recall prompt template | low |
| 29 | Stable cached header at attachment-creation time | Claude Code §4 | ADOPT | otherwise prompt-cache busts every turn | recall packaging | low |
| 30 | `<memory-context>` scrubber on streaming output | Hermes §3.4 | ADOPT | prevents the model from leaking its own memory fence | `internal/sse` streaming filter | low |
| 31 | Filesystem-as-mutex for cross-process consolidation | Claude Code §`Memory and Session Persistence.md:262-300` | ADAPT | use SQLite advisory lock instead of `.consolidate-lock` | consolidation package | low |
| 32 | "Memory files in DB, not on disk" tool-hint interceptor | goclaw `exec_memory_hints.go` | ADOPT | prevents agents from `cat MEMORY.md` and burning a turn | tool runtime hint surface | low |
| 33 | Agent-callable tools: `memory_search`, `memory_get`, `memory_expand` (read-only) | goclaw §1.4 | ADOPT | enforces "agents do not write directly" | `internal/tools` | low |
| 34 | Operator-only `memory add/edit/delete` via CLI/UDS | AGH posture | ADOPT | matches "agents read; operators (or sub-agent extractor) write" | `internal/cli/memory*.go` + UDS | low |
| 35 | External provider plugin (Mem0/Honcho/Hindsight as plugin) | Hermes §1, OpenClaw §1.4 | ADAPT | wire as an `internal/extension` capability, single active provider, not native | extension contract | medium |
| 36 | Lifecycle hooks: `initialize / system_prompt_block / prefetch / sync_turn / on_session_end / on_pre_compress / on_memory_write / on_session_switch / shutdown` | Hermes §1, Letta | ADOPT | minus the Hermes provider-runtime extras AGH does not need | `internal/extension/contract` | medium |
| 37 | `appendSystemPrompt` tail-injected reminder for high-priority recall | Claude Code §4 | ADOPT | "Before recommending from memory" header | situation package | low |
| 38 | Bi-temporal validity (Zep) | `analysis_ai-memory.md` §8.3 | DEFER | requires schema overhaul; revisit with KG | parking lot | high |
| 39 | Provenance link semantic→episodic | `analysis_ai-memory.md` §8.1 | ADAPT | when consolidation lands; encode as `source_session_ids` JSON column on curated rows | catalog DDL | medium |
| 40 | Append-only memory event log as audit | Agno; OpenFang §2.2 v8 | ADOPT | AGH already has `Operation` enum in `internal/memory/types.go`; back it with a real table | new migration in `internal/store/globaldb` | low |
| 41 | DSAR / GDPR right-to-erasure soft-delete + hard-delete log | `analysis_ai-memory.md` §13.5 | DEFER | hold until v2 ships; track in security review | parking lot | medium |
| 42 | Per-user 1.2× score boost for personal memory | goclaw §1.5 | DEFER | AGH does not have a strong per-user model yet | parking lot | low |
| 43 | Daily-log memory `memory/YYYY-MM-DD.md` | OpenClaw, OpenFang | ADOPT | cheap, agent-readable, git-versionable | curated layout | low |
| 44 | Session-end snapshot on `/new` / `/reset` | OpenClaw `session-memory/HOOK.md` | ADOPT | matches AGH's `on_session_stopped` hook | hook handler | low |
| 45 | LLM-generated session slug for snapshot filename | OpenClaw | ADOPT | makes snapshots browsable | hook handler | low |
| 46 | Sub-agent extractor with sandboxed tool whitelist | Claude Code §3 | ADOPT | locked down via existing tool runtime | `internal/memory/extractor` | medium |
| 47 | Single-slot pending context coalescing (drop intermediate) | Claude Code §3 | ADOPT | "message 2's extraction window silently lost" is acceptable | extractor package | low |
| 48 | Periodic extraction throttle ("every N eligible turns") | Claude Code `tengu_bramble_lintel` | ADOPT | sane default 1; configurable | config | low |
| 49 | LLM-as-write-gate for memory poisoning detection | Hermes `_scan_memory_content` | ADOPT | regex stop-list for invisible Unicode + injection prompts is cheap | controller pre-write | low |
| 50 | Atomic write via `mkstemp` + `fsync` + `rename` | Hermes `_write_file` | ADOPT | AGH must never half-write a curated file | `internal/fileutil` helper | low |
| 51 | File lock per curated file (`fcntl.flock`) | Hermes | ADAPT | AGH is single-daemon; SQLite advisory lock + in-process mutex is sufficient | controller | low |
| 52 | Agent-mention scoping (`@agent-debugger` searches that agent's memory only) | Claude Code §4 | ADOPT | matches `agent_name` field already in `internal/memory/types.go::Header` | recall scope filter | low |
| 53 | KG entity LLM extraction with strongly-typed prompt | goclaw §1.6 | DEFER | KG itself deferred | parking lot | high |
| 54 | Letta MemFS git-backed memory dir | Letta | DEFER | nice-to-have but not v2 | parking lot | medium |
| 55 | Markdown chunking + embedding cache by content hash | OpenClaw §3.2; goclaw §1.2 | ADOPT | makes re-embedding cheap | catalog DDL | medium — sqlite-vec |
| 56 | `EnsureSchema` boot reconciliation for column changes | OpenFang `structured.rs:128-136`; OpenClaw `ensureColumn` | REJECT | violates `agh-schema-migration` skill | n/a | n/a |
| 57 | Single shared cross-agent UUID for tool-driven KV | OpenFang `kernel.rs:6471-6478` | REJECT | undocumented surprise; conflates agents | n/a | n/a |
| 58 | Pre-reasoning hook ("unified pre-reasoning hook" — ReMe) | `analysis_ai-memory.md` §11.6 | ADOPT | one place to assemble curated block + recall + skills | hook event `on_turn_start` | low |
| 59 | `parent_session_id` chain for compaction lineage | Hermes §2.4 | ADOPT | already partially in AGH | `internal/store/sessiondb` extension | low |
| 60 | Memory write event variant in transcript (`Compacted` with replacement) | Codex §3.2 | ADOPT | needed for resume + audit | sessiondb migration | medium |

---

## 4. Layer-by-layer integration sketch

### 4.1 Persistence (SQLite, numbered migrations only)

- Curated layer = **markdown files on disk** (existing AGH layout: global +
  workspace + agent-local), authoritative.
- Derived catalog in `agh.db` (`internal/store/globaldb`):
  - `memory_files (id, scope, workspace, agent_name, filename, type, name,
    description, content_hash, mtime_ms, indexed_at)` — one row per markdown file.
  - `memory_chunks (id, file_id, start_line, end_line, content, content_hash, embed_model)`
  - `memory_chunks_fts` FTS5 virtual table over `content` (unicode61).
  - `memory_chunks_fts_trigram` FTS5 virtual table (trigram) for CJK.
  - `memory_chunks_vec` `vec0` (sqlite-vec) virtual table for embeddings (gated by
    config; fallback to in-process cosine if cgo not available).
  - `memory_recall_signals (file_id, recall_count, recall_score, last_recalled_at)`
    — feeds dreaming worker.
  - `memory_operations (id, operation, scope, workspace, filename, agent_name, summary,
    created_at)` — backs `OperationRecord` (already declared).
  - `memory_consolidations (id, agent_id, last_run_at, last_promotion_at, debounce_until)`.
- Migrations: numbered files in `internal/store/migrations/00NN_memory_v2_*.sql`, applied
  by the existing migration registry. **No `EnsureSchema` reconciliation.**
- All SQL writers go through one helper (`store.ExecuteWrite(ctx, fn)`) that wraps
  `BEGIN IMMEDIATE` + jittered retry + periodic WAL checkpoint, mirroring Hermes
  `_execute_write`.

### 4.2 Read path

- `internal/memory/recall` package, single entry point:
  `Recall(ctx, query, opts) (Packaged, error)`.
- Pipeline:
  1. Trivial-message gate (stop-list, ≥3 meaningful tokens — goclaw `trivial_filter.go`).
  2. Context-aware query rewrite (rune-safe `Context: …\nQuery: …`).
  3. Candidate generation: hybrid FTS (unicode61 + trigram) + vector cosine;
     weighted merge (default 0.7 text + 0.3 vec; per-agent tunable).
  4. Re-rank: cheap small-model side-query if configured, else
     `0.5·cosine + 0.3·recency + 0.2·recall_score`.
  5. Apply `alreadySurfaced` filter (paths surfaced earlier in the same session).
  6. Top-K (≤5).
  7. Package as `<system-reminder>` blocks with stable cached headers + freshness
     banner from `memoryAge`.
- Records `RecordRecall(id, score)` in a fire-and-forget goroutine.
- `LoadPromptIndex(scope)` (already in `internal/memory/types.go::Backend`) returns
  the truncated-to-25 KB MEMORY.md content for the system-prompt block.

### 4.3 Write path (consolidation pipeline)

- New package: `internal/memory/controller`. Single entry point
  `Decide(ctx, candidate) (MemoryWriteDecision, error)` returning `ADD/UPDATE/DELETE/NOOP`.
- Decision input: extracted candidate fact + K most-similar existing entries (top-5 from
  catalog) + agent identity + scope.
- Pre-write gate: `_scan_memory_content` (Hermes §3.4) — invisible Unicode, prompt-injection
  patterns, exfiltration commands, persistence hooks.
- Persistence: atomic file write (`mkstemp` + `fsync` + `rename`); index update on commit;
  emit `memory.write` operation row.
- Source of writes:
  1. **Operator** via CLI/UDS (`agh memory add|edit|delete`).
  2. **Extractor sub-agent** invoked from `on_post_response` hook with sandboxed tool
     surface (Claude Code-style).
  3. **Pre-compaction flush** invoked from `pre_compaction_flush` hook (OpenClaw-style).
  4. **Dreaming worker** writing `_system/dreaming/YYYYMMDD-consolidated.md` after
     `episodic.created` events.
- Consolidation workers (`internal/memory/consolidation/`):
  - `episodic.go` — `on_session_stopped` → episodic summary into catalog.
  - `dreaming.go` — debounce + 4-component recall scoring + LLM synthesis +
    `MarkPromoted`.
  - `dedup.go` — Jaro-Winkler entity dedup (deferred until KG lands; stub for now).
- Async by default; never block the agent loop.

### 4.4 File-based memory layout

```
$AGH_HOME/memory/                        global scope
  MEMORY.md                              ≤200 lines / ≤25 KB index
  <type>_<slug>.md                       topic file with YAML frontmatter
  <type>_<slug>.md
  daily/YYYY-MM-DD.md                    short-term episodic log
  _system/dreaming/YYYYMMDD-*.md         consolidation output
$WORKSPACE/.agh/memory/                  workspace scope
  MEMORY.md                              workspace index
  <type>_<slug>.md
$AGH_HOME/agents/<agent>/memory/         agent-local override
```

- Hierarchical precedence: agent-local → workspace → global (matches existing
  `internal/CLAUDE.md` Memory & Skills RFC five-layer model).
- AGENTS.md per directory (Codex pattern), discovered by walking `cwd` → project-root
  marker (default `.git`). Concatenated root → leaf with explicit `# AGENTS.md
  instructions for <directory>` start markers and `</INSTRUCTIONS>` end markers.

### 4.5 Compaction

- Compaction is owned by the session manager; memory wires in via two hooks:
  - `pre_compaction_flush` — fired before compaction starts; the bundled handler runs
    a silent agentic turn that asks the model to flush durable facts.
  - `post_compaction` — fired after; runs the consolidation worker for the touched session.
- Compaction summary is stored as a `Compacted` event variant in `events.db` with an
  optional `replacement_history_ref` pointing at a sliced copy of the rebuilt history,
  so resume can short-circuit.
- Independent model selection via `agent.compaction.model` config key (OpenClaw pattern).
- Compaction prompt framed as inter-LLM handoff (Codex `summary_prefix.md`).

### 4.6 Hooks (event names AGH needs)

Add to `internal/hooks` taxonomy (already typed):

- `on_turn_start` (renamed from current `on_session_started` if it overlaps) — pre-reasoning
  packaging point; consolidates `MEMORY.md` snapshot + recall + skills.
- `on_post_response` — extractor entry point.
- `pre_compaction_flush` — silent agentic flush turn.
- `post_compaction` — consolidation trigger.
- `on_session_stopped` (existing) — snapshot to `daily/YYYY-MM-DD.md`.
- `on_memory_write` — for extension provider sync.
- `on_session_switch` — provider notification on split.
- `on_idle` — sleep-time consolidation trigger (for dreaming worker tick).

All hooks are typed dispatch (per `internal/CLAUDE.md`), never event bus.

### 4.7 Sessions (resume + subagent rules)

- Resume: walk events newest→oldest; stop at the latest `Compacted` checkpoint with
  `replacement_history_ref`; replay suffix forward (Codex pattern).
- Sub-agent sessions inherit the parent's curated snapshot but receive a controller
  configured with `Mode = ReadOnly` so write tools are denied at the controller layer
  (defense in depth; tool runtime also denies).
- Curated snapshot is captured at session boot from `LoadPromptIndex(scope)` and frozen
  in `internal/situation/service.go`. `agh memory reload` invalidates the snapshot for
  the next turn.

### 4.8 Extension contract surface

`internal/extension/contract` adds:

- `MemoryProvider` interface mirroring Hermes' lifecycle:
  - `Initialize(ctx, ProviderInit) error`
  - `SystemPromptBlock(ctx, SnapshotRequest) (string, error)`
  - `Recall(ctx, RecallRequest) (RecallResult, error)` — optional
  - `Prefetch(ctx, PrefetchRequest) error`
  - `SyncTurn(ctx, TurnRecord) error`
  - `OnMemoryWrite(ctx, WriteRecord) error`
  - `OnPreCompress(ctx, PreCompressRequest) (Hint, error)`
  - `OnSessionEnd(ctx, SessionEnd) error`
  - `OnSessionSwitch(ctx, SwitchRecord) error`
  - `Shutdown(ctx) error`
- Single active provider per workspace (Hermes rule); plugin slot in config.
- Provider tools dedup against built-in tool names (Hermes `agent_runner-memory.ts`).
- The existing `ProviderHookRunner` / `ProviderHookRequest` / `ProviderHookResult` in
  `internal/memory/types.go` get re-shaped to match this contract. Existing
  `ProviderHookEvent` enum gets extended with the new event names.

### 4.9 Agent-operable CLI/HTTP/UDS surface

- CLI:
  - `agh memory list [--scope] [--workspace] [--type] [--json]`
  - `agh memory show <filename>`
  - `agh memory add --type <t> --name <n> --scope <s> --content @file.md`
  - `agh memory edit <filename> --content @file.md`
  - `agh memory delete <filename>`
  - `agh memory search <query> [--scope] [--limit] [--json]`
  - `agh memory reindex [--scope] [--workspace]`
  - `agh memory history [--operation] [--since] [--json]`
  - `agh memory health [--json]`
  - `agh memory dream` (manual dreaming worker trigger; respects debounce)
  - `agh memory provider list|enable|disable <name>`
- HTTP/UDS parity for every CLI verb (per `internal/CLAUDE.md` "Agent-operable by default").
- All return JSON with stable schema (already typed in `internal/memory/types.go`).

### 4.10 Config keys

Under `[memory]` in `config.toml`:

- `enabled = true` (default)
- `default_scope = "workspace"` (default; per-agent override via `memory.scope`)
- `prompt_index_max_lines = 200`, `prompt_index_max_bytes = 25000`
- `extractor.enabled = true`, `extractor.model = "haiku"`, `extractor.every_n_turns = 1`,
  `extractor.max_turns = 5`
- `recall.top_k = 5`, `recall.text_weight = 0.7`, `recall.vector_weight = 0.3`,
  `recall.ranker_model = "haiku"`, `recall.use_vec = true`
- `consolidation.enabled = true`, `consolidation.debounce = "10m"`,
  `consolidation.min_unpromoted = 5`, `consolidation.recall.min_count = 2`,
  `consolidation.recall.min_score = 0.2`, `consolidation.dreaming.enabled = true`
- `compaction.model = "haiku"`, `compaction.warn_threshold_buffer = 20000`,
  `compaction.auto_threshold_buffer = 13000`
- `flush.pre_compaction = true`, `flush.snapshot_on_session_end = true`
- `freshness.banner_after_days = 1`
- `provider.name = ""` (one of `mem0|honcho|hindsight|...`), `provider.config = {}`
- `vector.backend = "sqlite-vec"|"in-process"` — fallback when cgo unavailable.

---

## 5. Delete list (hard cuts)

Greenfield-delete; no aliases or shims. Each is a single-commit cut alongside the v2 wiring.

- `internal/memory/types.go` — keep file, but **delete** the speculative
  `ContextRefResolver`, `ContextRef`, `ContextRefKind`, `ResolvedContext`,
  `ResolvedContextItem`, `TokenBudget`, `ProviderHookRunner`, `ProviderHookRequest`,
  `ProviderHookResult`, `ProviderHookEvent` declarations. Re-author them as v2
  contracts under the new packages (`internal/memory/recall`,
  `internal/extension/contract`). The current shapes are placeholders — they cannot
  survive the v2 contract.
- `internal/memory/store.go` — current `Backend` interface stays as the API contract,
  but the implementing store is fully rewritten to use the catalog (current
  implementation is presumably file-walk-only; see §4.1).
- Any `EnsureSchema`/boot-reconciliation pattern in `internal/memory/*` (if present —
  flagged from the OpenClaw/OpenFang anti-pattern). All schema lives in the migrations
  registry.
- Current `internal/memory/consolidation/*.go` "Time → Sessions → Lock cascade"
  implementation — replaced by the worker pipeline. Migration is a hard cut.
- Any in-process MEMORY.md cache that mutates mid-session — replaced by the frozen
  snapshot.
- Any tool surface that lets the LLM `Write` to curated memory directly — only the
  extractor sub-agent and operator CLI are valid writers. Delete tool registrations
  matching that pattern in `internal/tools/` (audit needed).
- Any per-agent ad-hoc `ALTER TABLE` (the OpenFang `agents` table hot-patch pattern) —
  none should exist; if one is found in `internal/memory/*`, delete it and add the
  column to the migrations registry.
- `internal/memory/types.go::OperationRecord.AgentName` redaction rule must explicitly
  drop raw provider tokens — verify against `internal/CLAUDE.md` security invariant
  ("`claim_token` redaction is non-negotiable").

---

## 6. Open questions for the TechSpec

1. Is `sqlite-vec` an acceptable cgo dependency, or do we want a pure-Go fallback at
   parity (in-process cosine over JSON-stored embeddings, Mem0-style)? Decision affects
   build matrix and CI.
2. Which embedding provider is the v2 default — local Ollama, OpenAI text-embedding-3-small,
   or a configurable adapter pattern (OpenClaw `engine-embeddings.ts` covers six)?
   Default must work with no API key.
3. The dreaming worker writes `_system/dreaming/YYYYMMDD-*.md` into the curated dir —
   should those files participate in `MEMORY.md` indexing, or live in a parallel
   `_system/` namespace excluded from prompt injection? (Otherwise model risks reading
   its own synthesis as "user instruction.")
4. Is the extractor sub-agent a forked LLM call from the daemon (Claude Code style) or
   a separate ACP-spawned session (Hermes-style curator agent)? The first is cheaper;
   the second composes with AGH's autonomy kernel.
5. AGENTS.md per-directory discovery — how deep do we walk, what is the default
   project-root marker, and do we expose `agh memory agents-md show` for debugging the
   resolved order?
6. Sub-agent isolation: does a sub-agent's curated snapshot equal the parent's, or do we
   intersect with the sub-agent's role-scoped allowlist?
7. Provider plugin lifecycle — does the daemon spawn the provider (process lifetime) or
   does the extension SDK speak HTTP/MCP to an external provider? (Hermes does Python
   in-process; AGH cannot.)
8. Recall ranker model — small local model only, or any configured chat model? If any
   model, is it always the same as the agent's, or independent (`recall.ranker_model`)?
9. Schema: do we store embeddings inline in `memory_chunks.embedding` (BLOB) or in a
   separate `memory_chunks_vec` virtual table? The latter is cleaner for migrations;
   the former is what OpenClaw/goclaw both did.
10. KG / graph memory — DEFER is the right call for v2, but the TechSpec must state
    explicitly *what* is deferred and *what evidence* would re-open it (e.g., LoCoMo eval
    showing recall-only stack <60 % accuracy on multi-hop queries).

---

## 7. Risk register

| # | Risk | Mitigation |
|---|---|---|
| R1 | sqlite-vec cgo footprint breaks Windows / pure-Go builds | Ship pure-Go fallback (`vector.backend = "in-process"`); CI matrix adds non-cgo build. |
| R2 | Frozen snapshot drifts from disk reality across long-running sessions | Expose `agh memory reload` UDS verb + auto-reload on `on_idle` past N hours; snapshot includes `mtime_ms` so divergence is observable. |
| R3 | Dreaming worker hot-loops on empty filter | Stamp `lastRun` even on empty-filter skips (goclaw P10.1 lesson). |
| R4 | Extractor sub-agent loses extraction window during burst | Single-slot pending context (Claude Code) — accept the silent loss; surface in observe events for ops. |
| R5 | `WHAT_NOT_TO_SAVE` block degrades over time as model drifts | Eval-validate periodically; freeze prompt text in `internal/memory/extractor/prompts.go` with version tag. |
| R6 | Memory poisoning via tool output stored as fact | Pre-write `_scan_memory_content` regex + invisible-Unicode reject (Hermes). |
| R7 | sqlite-vec migration drift between major versions | All schema in numbered registry; no `ensureColumn` patterns; CI gate `make codegen-check`. |
| R8 | Ranker side-query latency on hot path | Async with timeout (200 ms); fall back to weighted-score ranking if exceeded; record in `MetricRetrieval`-style row for tuning. |

---

*End of integration map. ~860 lines, decisions opinionated, deferrals explicit.*
