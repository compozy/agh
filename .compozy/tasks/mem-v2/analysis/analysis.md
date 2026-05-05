# mem-v2 — overarching analysis

**Status:** research synthesis (read-only). Not a TechSpec; the input that feeds one.
**Date:** 2026-05-04. **Branch:** `qa-final`.
**Scope:** definitive memory system for AGH, sized to be at parity with — and
preferably better than — the strongest competitors in the corpus.

---

## 1. TL;DR

Across ten forensic analyses (eight competitor memory systems, one academic /
framework survey of the field, one honest audit of `internal/memory/`), the
shape of a definitive AGH memory system is no longer ambiguous.

**The architecture converges on a layered system, not a vector store**
(transcript / session-state / curated / procedural / external-provider), with a
**typed write controller** that mediates every mutation, a **frozen prompt
snapshot** for cache stability, **deterministic recall first** (FTS5 + trigram +
metadata), and an **append-only operation log** as the audit + observability
spine.

**Honest competitive position today.** AGH ships dreaming-consolidation
*scaffolding* (gates Time→Sessions→Lock + ticker runtime + spawner +
consolidation prompt — `internal/memory/dream.go` 456 lines + `consolidation/runtime.go`
464 lines), but **none of the worth-having parts**: no recall feedback loop
(`RecordRecall`), no quantified scoring (frequency/relevance/recency/freshness),
no `MarkPromoted` close-the-loop, no `_system/dreaming/YYYYMMDD-*.md` output
discipline, no dedicated dreaming agent with tool whitelist, no DLQ. AGH also
lacks Hermes' hard-won engineering wins (FTS5 trigram twin index, jittered
write retry helper, `<memory-context>` SSE scrubber, `parent_session_id`
lineage chain, atomic `mkstemp+fsync+rename`, pre-write
`_scan_memory_content`). And the `MemoryProvider` ABC — Hermes' real moat — is
declared in `internal/memory/types.go` but unconsumed. **AGH today ships
~20–25% of what Hermes ships.** Sequencing the redesign as "spine first,
everything advanced gradually graduates" puts AGH ~5 slices behind a system
that already exists in the wild.

**Where the integration map (`analysis_integration-map.md`) is right** — closed
taxonomy + `WHAT_NOT_TO_SAVE`, frozen system-prompt snapshot, MEMORY.md as a
thin index, controller-mediated writes, hybrid FTS + vec retrieval, lifecycle
hooks contract, sub-agent fork read-only memory.

**Where the integration map over-reaches and Codex's peer review corrects it**:
(a) it conflates `MEMORY.md` with source-of-truth (should be projection); (b)
it defers DSAR/observability/security threat-model to "later" when those are
architectural; (c) it borrows OpenClaw's pre-compaction flush before AGH has a
production compactor; (d) it imports goclaw's exact recall-scoring weights as
v2 invariants when the goclaw authors themselves flag those weights as
hand-tuned; (e) it reads "agents never write directly" as a ban, where AGH's
project rules require *every feature to be agent-manageable*; (f) it does not
separate **memory hierarchy** from **instruction hierarchy** (`AGENTS.md` is
the latter, not the former); (g) it told us to *delete* the `ProviderHookRunner`
declared types and re-author them — **wrong**: those are exactly the seam that
becomes the `MemoryProvider` ABC. Extend, do not delete.

**Where Codex's peer review under-shoots.** Codex correctly identified the
spine deficit but treated dreaming + provider ABC + Hermes engineering as
"slice 5" graduation candidates. That sequencing trades AGH's competitive
position for architectural purity. AGH already paid for the dreaming
scaffolding; the missing pieces (signals + scoring + promotion + dedicated
agent) are small additions, not a rewrite. The `MemoryProvider` ABC is mostly
contract work (10 lifecycle hooks against the existing typed-hooks dispatch).
The Hermes engineering wins are individually cheap. Treating these as
*extensions of slice 1* — not as deferred upgrades — is what shifts AGH from
"catching up" to "leading."

**The vertical slice that ships first is a *fat* Slice 1**: spine **plus**
dreaming v2 (extending the existing runtime to goclaw-grade signals + scoring +
promotion) **plus** the `MemoryProvider` ABC (10 Hermes lifecycle hooks against
the existing declared seam) **plus** Hermes engineering parity (twin FTS,
retry helper, scrubber, lineage, atomic writes). Sqlite-vec, real compaction,
pre-compaction flush, KG, external memory providers, and AGH Network
federation stay deferred — those genuinely require evidence or a compactor
that does not yet exist. Detail in §5.

---

## 2. Source corpus

Ten files in `.compozy/tasks/mem-v2/analysis/`:

| File | What it covers | Lines |
|---|---|---|
| `analysis_ai-memory.md` | Academic survey: Letta, Mem0, Zep, A-MEM, Cognee, Agno, MemOS, OpenMemory, Motorhead, Aura, Hindsight, Supermemory, ReMe + CoALA / OS-tier / three-tier taxonomies | 1037 |
| `analysis_ai-harness.md` | Cross-harness patterns: five-tier memory, file conventions, compaction cascades, hooks, sub-agent isolation, eval | 998 |
| `analysis_hermes.md` | Hermes runtime: 4 memory layers, `MemoryProvider` ABC, `ContextCompressor`, SQLite + FTS5 + trigram, holographic plugin, Honcho integration | 706 |
| `analysis_openclaw.md` | OpenClaw memory plugin: `MemoryPluginCapability`, dreaming cron (light→REM→deep), `memory/YYYY-MM-DD.md`, dreaming promotion weights | 763 |
| `analysis_codex.md` | Codex CLI: rollout JSONL, `Compacted.replacement_history`, two-phase consolidation pipeline (Phase 1 per-rollout, Phase 2 sandboxed agent), `memory_summary.md`, AGENTS.md resolver, three compaction triggers | 785 |
| `analysis_openfang.md` | OpenFang: single SQLite + 6 stores, source-vs-wiki drift, two-tier compaction, workspace identity files, KG warts | 1065 |
| `analysis_claude-code.md` | Claude Code memdir: closed 4-type taxonomy, `WHAT_NOT_TO_SAVE`, `findRelevantMemories` Sonnet ranker, forked extractor with `createAutoMemCanUseTool`, `/compact` algorithm, `.jsonl` sessions | 562 |
| `analysis_goclaw-paperclip-multica.md` | goclaw 3-tier consolidation + dreaming worker + 4-component recall scoring + recall feedback loop; paperclip `nativeContextManagement` capability flag; multica minimal-context | 578 |
| `analysis_agh-current.md` | Audit of AGH today: dual-scope markdown + derived FTS5 catalog, lock-rollback, Time→Sessions→Lock cascade; gaps: no `agent` scope, compaction is test-only, native tools read-only, extension capability declared but unconsumed | 922 |
| `analysis_integration-map.md` | Proposed v2 shape: 12 design pillars + 60-row decision matrix + layer-by-layer sketch + delete list + 10 open questions + 8 risks | 494 |

Plus `_codex_peer_response.md` — Codex (`gpt-5.5` `xhigh`) peer-pressure pass on
the integration map.

---

## 3. Convergent design pillars (after Codex correction)

The integration map's 12 pillars are mostly correct but several need precise
re-wording. The corrected list below is what the TechSpec must encode as
invariants.

### P1 — Layered memory, never one store
Five orthogonal layers: transcript, session-state, curated, procedural,
external-provider. Each has its own write protocol, lifecycle, and reader. AGH
must include the **`agent` scope** explicitly — current code only has
`global|workspace`, while `internal/CLAUDE.md` and the Memory & Skills RFC both
mandate `global|workspace|agent`. (`analysis_agh-current.md` §3, §16.)

### P2 — Frozen *pinned* snapshot + dynamic recall (corrected)
Pin the *curated index* at session boot for prompt-cache stability (Hermes,
Codex, Claude Code all do this). But recall remains dynamic per-turn, with a
**precomputed cache-stable header** (Claude Code `findRelevantMemories.ts`).
Don't conflate the two — frozen-everything kills recall freshness; dynamic-
everything kills prompt cache.

### P3 — Closed curated taxonomy with orthogonal axes (corrected)
The four-type curated taxonomy (`user|feedback|project|reference`) is one axis
*for curated memory only*. It does not stand in for AGH's full memory model. v2
adds three orthogonal axes: **source actor** (operator | extractor | dreaming |
provider | agent-proposed), **scope tuple** (`global|workspace|agent|session`,
plus stable namespace), **provenance** (source session/event IDs, confidence
class, supersession link). `WHAT_NOT_TO_SAVE` is non-negotiable
(Claude Code `memoryTypes.ts:183-194`) — including the "do not save even if the
user asks" rule.

### P4 — `MEMORY.md` is a *projection*, not co-source-of-truth (corrected)
`MEMORY.md` is rendered from a stronger source (event log or files). It is
hard-capped at 200 lines / 25 KB, with a visible truncation banner. It is never
both edited by humans/agents *and* edited by the runtime — that path is the
documented Claude Code drift mode (`analysis_claude-code.md` §12).

### P5 — Controller-mediated writes (ADD/UPDATE/DELETE/NOOP)
Every mutation traverses one typed controller that returns a
`MemoryWriteDecision`. NOOP is the "strongest indicator of a mature memory
system" (Mem0, `analysis_ai-memory.md` §5.1). Pre-write content scan
(`_scan_memory_content` regex from Hermes §3.4) is mandatory: invisible
Unicode, prompt-injection markers, exfiltration commands all reject. Atomic
writes via `mkstemp + fsync + rename`.

### P6 — Pre-compaction memory flush (DEFERRED, not P6 in slice one — corrected)
The pattern (silent agentic flush turn before compaction; OpenClaw
`agent-runner-memory.ts`) is excellent. But AGH does not yet have a production
compactor — `runContextCompaction` is only invoked from tests
(`analysis_agh-current.md` §8). Order: **first** define compaction checkpoints
+ resume semantics, **then** wire flush as a handler. Don't ship the flush
before there's a compactor to flush *into*.

### P7 — Async two-phase consolidation, but keep current lock semantics until v2 worker proves itself (corrected)
Episodic → semantic → dedup pipeline driven by typed events is the right
target shape (goclaw, Letta, Mem0). AGH's current `Time → Sessions → Lock`
cascade with hardlink-PID lock and rollback is one of the better-tested parts
of the current subsystem (`analysis_agh-current.md` §9, §16). Replace it
**after** the v2 worker proves idempotency and rollback, not before.

### P8 — Agents never *bypass* the controller (corrected from "agents never write")
"Agents never write directly" — as stated in the integration map — collides
head-on with AGH's project rule that every feature be agent-manageable. The
correct invariant is: **all writes flow through the controller**, regardless of
caller. Agents reach the controller through capability-gated, typed, auditable
surfaces (CLI / HTTP / UDS / native tools). Whether an agent's write is
*committed* or *proposed* is a policy decision, not a hard ban.
(`analysis_agh-current.md` §11, §15; project rule "agent-manageable by default".)

### P9 — Provider-neutral ranker (corrected from "Sonnet ranker")
Cheap candidate generator (FTS5 + trigram + metadata filters) → optional
re-rank with a small model when configured (timeout 200 ms, fall back to
weighted score) → top-K (≤5). AGH is provider-neutral; "Sonnet ranker" is a
Claude Code implementation detail, not the principle. Hermes proves FTS5 → LLM
summarisation works without a vector-first design (`analysis_hermes.md` §2.5).

### P10 — Dreaming worker ships in slice one, evidence tunes the weights (corrected after pushback)
Original wording said "dreaming graduates later." That was a misread of the
existing AGH state. AGH already ships ~40% of the dreaming engine: gates
Time→Sessions→Lock (`dream.go`), background ticker + queued check requests
(`consolidation/runtime.go`), workspace-aware spawner. What is missing is the
**signal layer** (`RecordRecall` after every search; per goclaw
`tools/memory.go:194-222`) and the **promotion layer** (4-component scoring +
`MarkPromoted` close-the-loop + `_system/dreaming/YYYYMMDD-*.md` output dir).
Both are small extensions on top of the existing scaffolding. v2 ships them in
slice one. The 4-component weights
(`0.30·frequency + 0.35·relevance + 0.20·recency + 0.15·freshness`) are
config keys, not invariants — defaults match goclaw, but eval data tunes them
post-ship. The dedicated dreaming agent gets a tool whitelist
(`createAutoMemCanUseTool` style), not the generic configured agent that
`dream.go` uses today.

### P11 — Memory hierarchy ≠ instruction hierarchy (corrected, restated)
File-based curated memory has a hierarchy (agent-local → workspace → global).
`AGENTS.md` per-directory is an *instruction resolver* (Codex). Don't merge
the two: instructions and memory have different lifecycles, different write
authorities, different consumers. Both must be visible to debugging
(`agh memory show-resolved`, `agh agents-md show`).

### P12 — Sub-agent forks are read-only against the parent's controller, but parent may extract from sub-agent traces
Sub-agent sessions inherit the parent's curated snapshot. The sub-agent's
controller is `Mode=ReadOnly` so write tools are denied at the controller layer
(defense in depth). Parent / session-end extraction *may* process the sub-
agent's final evidence through the controller (Codex partially does this).

### P13 (new — added by Codex peer review) — Memory observability is load-bearing
Every recall, write, skip, decision emits a structured event with: query,
scope filters, candidate IDs and scores, decision, provenance, redaction. AGH
already has the operation-log shape; v2 must extend it to recall + skip
events, and the agent-operable surface must let an operator query *why* a
memory was returned, *why* a write was a NOOP, *why* a recall returned empty.

### P14 (new) — ACP-native context management contract
Some providers (Claude Code adapter, Codex adapter, Hermes adapter, OpenClaw
adapter) already inject memory and run their own compaction. AGH must declare
a `native_context_management ∈ {confirmed | likely | unknown | none}` flag per
adapter (paperclip pattern, `analysis_goclaw-paperclip-multica.md` §2.3) and
default to **observe-and-pass-through** when the provider is `confirmed`.
Otherwise AGH double-injects against an already-managed context — a real,
documented failure mode.

---

## 4. Architecture overview

```
┌───────────────────────────────────────────────────────────────────────────┐
│                       AGH Memory v2 — overall shape                        │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  Layer 1: Transcript (events.db) — append-only, single source            │
│  Layer 2: Session-state — manager, lifecycle hooks                        │
│  Layer 3: Curated (markdown files + derived catalog) — frozen-snapshot   │
│           injected; recall on demand                                      │
│  Layer 4: Procedural (skills) — read into prompt at boot                 │
│  Layer 5: External provider (extension contract) — single active         │
│                                                                           │
│  Spine                                                                    │
│  ─────                                                                    │
│  • controller (typed write decisions: ADD/UPDATE/DELETE/NOOP/REJECT)     │
│  • operation log (append-only audit + observability)                     │
│  • scope identity (global|workspace|agent|session + namespace + actor)   │
│  • frozen prompt snapshot at session boot                                │
│  • deterministic recall (FTS5 + trigram + metadata, top-K)               │
│  • lifecycle hooks (typed dispatch, never bus)                           │
│                                                                           │
│  Layered upgrades (graduate after eval data, behind config flags)        │
│  ───────────────────────────────────────────────────────────────         │
│  • sqlite-vec embeddings (optional, fallback to in-process)              │
│  • dreaming consolidation worker (signals first, synthesis later)        │
│  • pre-compaction flush hook (after compactor is real)                   │
│  • KG / triple store (deferred until LoCoMo eval shows recall <60%)      │
│  • bi-temporal validity (deferred until KG)                              │
│  • DSAR / GDPR erasure (architectural — design now, ship as flag later)  │
│                                                                           │
└───────────────────────────────────────────────────────────────────────────┘
```

### Source-of-truth choice (the single biggest open question)

The integration map is silent on this. Codex peer review names it as the
single most under-explored fork. The candidates:

1. **Markdown files** authoritative; SQLite catalog is a derived index.
   (Current AGH; OpenClaw; goclaw partially.) Strength: human-editable,
   git-friendly, agent-readable. Weakness: split-brain risk on concurrent
   writes; index drift.

2. **Append-only event log** (`memory_events`) authoritative; Markdown files
   and SQLite catalog are projections. Strength: replay, rollback, audit,
   causal ordering, derived views for free. Aligns with AGH's posture of
   strong observability + operation log already in `internal/memory/types.go`.
   Weakness: human-edited Markdown becomes either an *input event* or a
   *projection* (not both) — needs explicit reconcile semantics; more
   projection code; harder to git-diff.

3. **Hybrid** (Markdown for curated, event log for state mutations) —
   defaults to today's behavior; risks split-brain we already see in Claude
   Code (`analysis_claude-code.md` §12).

**Recommendation for the TechSpec:** option 2 (event-sourced) is the
strongest fit for AGH's existing posture (operation log already exists; SD-009
about strong observability; the audit invariant from `internal/CLAUDE.md`).
Markdown files become **projections**: the runtime renders them from the event
log; human edits enter the system as `human_edit` events (committed via
`agh memory edit` or via git hook). This is the *biggest architectural call*
the TechSpec must make explicitly — defaulting to today's "files authoritative"
shape risks the same drift Claude Code documents.

---

## 5. Vertical slice — what to build first

**Reframe (after pushback):** Codex's peer review correctly identified the
spine deficit but proposed a 6-slice graduation that puts AGH ~5 slices behind
Hermes. The honest sequencing is a **fat Slice 1** that ships the spine **and**
extends what AGH already has into goclaw-grade dreaming **and** wires the
`MemoryProvider` ABC that is already declared in `internal/memory/types.go`
**and** ports the cheap-to-add Hermes engineering wins. After Slice 1, AGH
matches or exceeds Hermes everywhere except where a real compactor is
required. Subsequent slices add the genuinely deferrable pieces (compactor,
optional vector, external providers, federation).

### Slice 1 — Spine + Dreaming v2 + Provider ABC + Hermes engineering parity (TechSpec #1)

This is one TechSpec, four eixos, all merged together.

#### Eixo 1 — Spine (Codex's recommendation)

| Component | Required behavior |
|---|---|
| Scope identity | First-class `global|workspace|agent|session` scope; stable `workspace_id` (not filepath); explicit namespace + source actor on every memory record. Fixes documented drift (`analysis_agh-current.md` §3, §16). |
| Authoritative write spine | `memory_events` (append-only) as audit log: typed `ADD`/`UPDATE`/`DELETE`/`NOOP`/`REJECT`/`REINDEX`/`RECALL`. Events.db migration. Markdown files become projections from the event log. Human-edited Markdown enters the system as `human_edit` events via git hook or `agh memory edit`. |
| Controller | One package (`internal/memory/controller`). All CLI/HTTP/UDS/native-tool writes route through it. Frontmatter validation, `WHAT_NOT_TO_SAVE` scan, pre-write content security scan (`_scan_memory_content` — see Eixo 4), provenance tagging, dedup against top-K similar entries, contradiction detection. |
| Frozen prompt snapshot | At session boot, capture global+workspace+agent indexes as a frozen snapshot. Mid-session writes mutate disk and event log but do **not** mutate the system prompt. `agh memory reload` invalidates the snapshot for the next turn (not the current turn). |
| Deterministic recall | FTS5 (unicode61) + FTS5 (trigram for CJK) + metadata filters; top-K (≤5); stable cached header at attachment time; freshness banner from `memoryAge`; *why-recalled* metadata in observability events. **No vector. No LLM ranker.** Slice one is deterministic. |
| Agent-manageability parity | Every memory verb (`list`, `show`, `add`, `edit`, `delete`, `search`, `reindex`, `history`, `health`, `scope-show`, `signals show`, `dream trigger`, `dream status`, `provider list|enable|disable`) reachable via CLI **and** HTTP **and** UDS **and** at least one native tool. Read-only tools default open; `agh__memory_propose` is policy-gated commit-or-propose. |
| Observability | Every recall/write/skip emits structured events with: query, scope, filters, candidate IDs and scores, decision, provenance, redaction. Operation-log writes go to `events.db` (already mirrored in `globaldb` per `analysis_agh-current.md` §2). |

#### Eixo 2 — Dreaming v2 (extend, don't replace)

AGH today ships gates + ticker + spawner + consolidation prompt. v2 adds the
worth-having layers on top of that scaffolding — **no rewrite of `dream.go`
or `consolidation/runtime.go`, just extension**.

| Component | Required behavior |
|---|---|
| Recall feedback loop | New table `memory_recall_signals (file_id, recall_count, recall_score, last_recalled_at, freshness_started_at)`. Fire-and-forget `RecordRecall(id, score)` after every search (goclaw `tools/memory.go:194-222`). |
| 4-component scoring | Worker reads `memory_recall_signals` and applies weighted score (defaults match goclaw: `0.30·freq + 0.35·rel + 0.20·rec + 0.15·fresh`) — **weights are config keys**, not invariants. Cold-start protection via `freshness_started_at`. |
| Promotion gates + `MarkPromoted` | Threshold gates (≥5 unpromoted, MinRecallCount=2 — config); `MarkPromoted` close-the-loop after synthesis succeeds. |
| Output dir discipline | `<scope>/_system/dreaming/YYYYMMDD-<slug>.md` (LLM-generated session slug for browsability — OpenClaw pattern). Excluded from MEMORY.md prompt injection (open question §10 already raised this — answer is "yes excluded"). |
| Dedicated dreaming agent | New built-in agent role with tool whitelist (`createAutoMemCanUseTool`-style: read tools + `Edit` inside `_system/dreaming/` only). Replaces today's "any configured agent" approach. Prompt versioned in `internal/memory/prompts/`. |
| DLQ + idempotency | Stamp `lastRun` even on empty-filter skips (goclaw P10.1 lesson). On synthesis failure, write `memory_consolidations` row with `failure_reason`; manual retry via `agh memory dream retry <run_id>`. |
| Existing gates retained | Time → Sessions → Lock cascade stays (it works; SD-009 audit-friendly; lock-rollback semantics are tested). Add scoring as a **fourth gate** post-Time/Sessions/Lock: only run synthesis if at least N candidates pass score threshold. |

#### Eixo 3 — `MemoryProvider` ABC (extend declared seam, do **not** delete)

The integration map said to delete `ProviderHookRunner` / `ProviderHookRequest`
/ `ProviderHookResult` / `ProviderHookEvent`. Wrong — those are exactly the
seam that becomes the Hermes-style 10-hook ABC. Extend.

| Component | Required behavior |
|---|---|
| 10 lifecycle hooks | `Initialize` / `SystemPromptBlock` / `Recall` (optional) / `Prefetch` / `SyncTurn` / `OnSessionEnd` / `OnPreCompress` / `OnMemoryWrite` / `OnSessionSwitch` / `Shutdown`. Mirror Hermes shape verbatim — proven-in-production. |
| Single active provider per workspace | Hermes invariant. Plugin slot in `config.toml` `[memory.provider]`. |
| Bundled local provider | Reference implementation = the local SQLite + markdown store (existing `internal/memory/store.go`). Always present. Other providers (Mem0/Honcho/Hindsight) ship in slice 4 as extensions. |
| Provider tool dedup | Provider-supplied tools dedup against built-in tool names (Hermes pattern). Collisions log + reject at register-time. |
| Metadata-aware dispatch | Provider methods accept opts struct, not positional args (Hermes signature-introspection pattern in Go form). |
| Daemon delegation | Daemon **actually delegates** to enabled providers — fixes the SD-009 violation called out in `analysis_agh-current.md` §16 ("`memory.backend` extension capability is declared but unconsumed"). |

#### Eixo 4 — Hermes engineering parity (cheap wins)

Each is small in isolation but together separate "memory store" from
"memory platform."

| Component | Source | Effort | Required behavior |
|---|---|---|---|
| FTS5 trigram twin index | Hermes §2.1 | XS | Add trigram FTS5 virtual table beside unicode61. Recall pipeline queries both, merges. Fixes CJK regression. |
| Jittered write retry helper | Hermes `_execute_write` | XS | `store.ExecuteWrite(ctx, fn)` wrapping `BEGIN IMMEDIATE` + 15-retry loop with 20-150 ms jitter + periodic WAL checkpoint. All SQL writers go through it. |
| `<memory-context>` SSE scrubber | Hermes §3.4 | XS | `internal/sse` filter that neutralises memory fence markers in streaming output. Prevents the model from regurgitating its own memory injection. |
| `parent_session_id` lineage chain | Hermes §2.4 | S | Already partial in AGH; complete it. Resume walks lineage forward to message-bearing descendants. |
| Atomic write pattern | Hermes `_write_file` | XS | `internal/fileutil.AtomicWrite` (`mkstemp + fsync + rename`). All curated-file writes go through it. Half-write impossible. |
| Pre-write `_scan_memory_content` | Hermes §3.4 | S | Regex catalog: invisible Unicode, prompt-injection markers (`ignore previous`, `you are now`, etc.), exfiltration commands (`curl`, `nc`, `base64 -d`), persistence hooks (`launchctl`, `cron`, `systemd`), zero-width chars. Reject + log. |
| Char-bounded markdown discipline | Hermes; Claude Code `memdir.ts` | XS | 25 KB cap on `MEMORY.md` with visible truncation banner. Per-file caps configurable. |
| Action-cue header phrasing | Claude Code `memoryTypes.ts` | XS | Eval-validated 3/3 vs 0/3 effect — tiny effort, big behavior shift. Versioned prompt in `internal/memory/prompts/`. |

#### QA proofs (must pass before Slice 1 merges)

Twelve named scenarios — eight from Codex's spine list plus four for the
extended scope:

1. Write memory → next session sees snapshot.
2. Mid-session update does not mutate current snapshot.
3. `agh memory reload` invalidates snapshot for next turn (not current).
4. Agent-scoped memory does not leak into other-agent recall.
5. Workspace path move does not orphan rows (stable `workspace_id`).
6. Stale memory (older than `freshness.banner_after_days`) shows banner.
7. Delete/supersede removes from recall AND records audit event.
8. Operation history is replayable to reconstruct curated state.
9. **Recall signal recorded after every search; promotion fires only when threshold passes.**
10. **Dreaming run with synthesis failure leaves a `memory_consolidations` row + retryable.**
11. **Bundled provider + one mock external provider both fire all 10 lifecycle hooks in correct order during a session boot/end cycle.**
12. **Pre-write `_scan_memory_content` rejects invisible-Unicode + prompt-injection payloads; rejection emits an audit event.**

#### Explicit non-goals in Slice 1

- `sqlite-vec` embeddings (slice 3)
- Real compactor + `Compacted.replacement_history` resume (slice 2)
- Pre-compaction memory flush (slice 2 — depends on compactor)
- LLM-based recall ranker (slice 3)
- KG / triple store / bi-temporal validity (parking lot)
- External memory providers (Mem0/Honcho/Hindsight — slice 4)
- AGH Network memory federation (slice 5)
- Per-directory `AGENTS.md` resolver changes (separate TechSpec; instruction
  hierarchy ≠ memory hierarchy per P11)

### Slice 2 — Real compactor + pre-compaction flush

Current `runContextCompaction` is test-only (`analysis_agh-current.md` §8).
Slice 2 ships:

- Real compactor wired into the session manager (not test-only).
- `Compacted.replacement_history` event variant (Codex §3.5) for O(checkpoint-tail) resume.
- Pre-compaction flush hook (OpenClaw `pre_compaction_flush`) — now there is a compactor to flush *into*.
- Independent compaction model selection (`agent.compaction.model`).
- Compaction prompt framed as inter-LLM handoff (Codex `summary_prefix.md`).
- Five-section structured summary template (Hermes Active Task / Decisions / Identifiers / Progress / Pending — load-bearing for context preservation).
- Tool-pair sanitisation + JSON-safe argument truncation (Hermes §3).

### Slice 3 — Optional vector + optional LLM ranker

Both behind config flags; both with deterministic fallback.

- `sqlite-vec` virtual table behind `[memory] vector.backend = "sqlite-vec"` config flag; default off; pure-Go fallback (in-process cosine over JSON-stored embeddings) at parity for non-cgo builds.
- Hybrid retrieval: weighted-merge default; RRF-k60 alternative behind flag.
- Optional LLM ranker (provider-neutral; not Sonnet-specific) with strict 200 ms timeout; falls back to weighted score.

### Slice 4 — External memory providers

`MemoryProvider` extension API (already shipped in Slice 1) gets second-party
providers as extensions:

- Mem0 / Honcho / Hindsight as separate extension packages.
- Provider lifetime model: process / HTTP / MCP (TechSpec #4 chooses).
- `native_context_management` tri-state flag per provider (paperclip pattern, P14).

### Slice 5 — AGH Network memory federation

- Explicit namespace exchange protocol (never hidden global state — OpenFang's shared UUID is the anti-pattern).
- Opt-in only.
- Namespace audit + provenance preservation across peers.

### Slice 6 — KG / bi-temporal / advanced

- Re-open evaluation gate (LoCoMo / LongMemEval) showing recall-only stack <60% accuracy on multi-hop queries before this slice opens.
- Triple store (entities/relations) + LLM extraction pipeline.
- Bi-temporal validity (Zep pattern).
- Spreading activation / multi-attribute graph (A-MEM).

---

## 6. Decision matrix delta

The integration map proposed 60 verdicts. Codex peer review flips 8 of them.
The TechSpec must adopt the corrected verdicts.

| # | Pattern | Map verdict | **Corrected** | Reason |
|---|---|---|---|---|
| 1 | Closed 4-type curated taxonomy | ADOPT | **ADAPT** | Keep the four types but add orthogonal `source_actor` / `scope` / `provenance` axes. Curated taxonomy is one axis, not the full memory model. |
| 8 | Pre-compaction memory flush | ADOPT | **DEFER** | AGH has no production compactor. Define compaction first; flush is slice 3. |
| 9 | Independent compaction model selection | ADOPT | **DEFER** | Same: requires a compactor. |
| 13 | Dreaming worker formula (4-component) | ADOPT | **ADAPT (slice 1, weights as config)** | AGH already ships dreaming scaffolding; slice 1 adds signals + scoring + promotion + dedicated agent. Weights are config keys (defaults match goclaw), not invariants — eval data tunes them post-ship. |
| 16 | Hybrid FTS + vector | ADOPT | **ADAPT** | FTS5 + trigram is mandatory; vector is optional behind a config flag with a pure-Go fallback. Vector is *not* a v2 gate. |
| 34 | Operator-only memory CRUD | ADOPT | **ADAPT** | "Operator-only" violates AGH's agent-manageability rule. Correct shape: controller-gated writes through CLI/HTTP/UDS/native tools, with policy deciding whether agent writes are commit, propose, or deny. |
| 41 | DSAR / GDPR erasure | DEFER | **ADAPT** | Soft-delete + hard-delete log + deletion propagation are architectural. Design now; ship slice-by-slice. Don't pretend it's a compliance afterthought. |
| 47 | Single-slot pending context coalescing | ADOPT | **REJECT** | Silent extraction loss is too weak for AGH. Use a bounded queue with observable dropped windows, not Claude Code's silent-overwrite (`analysis_claude-code.md` §3 catalogs this as a degradation mode). |

---

## 7. Alternative architectures still in play

The integration map under-explored three structural alternatives. The TechSpec
must consider each and explicitly take or defer.

### Alt 1 — Event-sourced memory log as source of truth
**What it buys.** Audit, replay, rollback, causal ordering, derived views
(Markdown, FTS, vector, provider sync) for free. Matches AGH's posture
(strong observability + operation log already exists).
**What it costs.** More projection code, repair tooling, migration
discipline. Human-edited Markdown becomes either an input event or a
projection (not both) — needs explicit reconcile semantics.
**Recommendation.** **Adopt.** This is the biggest under-explored fork in the
integration map; defaulting away from it preserves the Claude Code drift mode.

### Alt 2 — Content-addressed memory / artifact store
**What it buys.** Cheap re-embedding (hash-keyed cache); dedup; immutable
provenance; stable citations; aligns with goclaw + OpenClaw which both
hash-key embeddings.
**What it costs.** Harder deletion / GC; less direct human editability; more
projection machinery for topic files.
**Recommendation.** Reconsider when v2 adds embeddings, large artifacts,
provider sync, or AGH Network federation. Not slice one.

### Alt 3 — Embedding-free retrieval MVP
**What it buys.** Deterministic, pure-local, prove-the-architecture-first.
Hermes shows FTS5 → LLM summarisation works; AGH already has lexical FTS.
**What it costs.** Worse paraphrase recall; some "conceptual" memory missed
until vector lands.
**Recommendation.** **Adopt for slice one.** Vector graduates to slice four
once eval data exists.

---

## 8. Blind spots in the corpus + Codex peer review

Eight blind spots where the corpus systematically under-delivered:

| Blind spot | Why it matters | TechSpec must answer |
|---|---|---|
| Memory observability | "Why was this recalled / written / skipped?" The integration map treats this as a side-effect. AGH's posture (SD-009) makes it load-bearing. | Structured event schema for every recall + write + skip + decision. |
| Versioning, rollback, time-travel | Codex has `Compacted.replacement_history`; Letta uses git dirs; Hermes checkpoints. The integration map names history but not rollback semantics. | `agh memory rollback <event_id>` UDS verb; supersession links on every record. |
| Memory as a security surface | The map treats security as gates, not threat model. Poisoning, prompt injection, secret-symlink escape, cross-user leakage, purpose binding all appear in the corpus. | Threat-model section; `_scan_memory_content` regex catalog; symlink hardening; tool-output handling rules. |
| AGH Network / cross-agent federation | The corpus covers within-runtime memory. AGH's network premise needs explicit namespace exchange — never hidden global state (OpenFang's cross-agent UUID is the anti-pattern). | Federation namespace protocol; opt-in only; namespace audit. |
| ACP-native context management | Some providers (Codex, Claude Code, Hermes, OpenClaw adapters) already inject memory and run their own compaction. Double-injection is real. | `native_context_management` tri-state flag in extension manifest (paperclip pattern). |
| Evaluation strategy | LoCoMo / LongMemEval are cited but no AGH-specific eval plan exists. | Acceptance tests: stale-fact correction, contradictory memory, scope isolation, recall precision, compaction continuity. |
| Prompt artifacts as versioned code | OpenClaw and Hermes show compaction/flush prompts are load-bearing runtime logic, but treated as inert text. | Prompts versioned in `internal/memory/prompts/` with version tags + eval-validation gate. |
| Procedural memory governance (skills) | The map treats skills as a layer; Hermes shows curation is a lifecycle (state, backup, c-uration, rollback). | Skill-curation lifecycle: backup before edit, rollback on failure, structured YAML reconciliation. |

---

## 9. Risk register (corpus + peer-review)

Top 8 structural risks. The integration map listed eight; Codex peer review
adds five more. The merged top 8 ranked by criticality:

| # | Risk | When it bites | Mitigation |
|---|---|---|---|
| R1 | Scope identity rot | Workspace path moves; two agents share a host; AGH Network introduces remote peers. AGH today has path-based workspace identity and *no* agent scope. | Canonical scope tuple `(user_id, workspace_id, agent_id, session_id, source_actor, namespace)` enforced at every CLI/HTTP/UDS/tool surface. Stable workspace_id, not filepath. |
| R2 | Source-of-truth split-brain | Topic file, `MEMORY.md`, SQLite catalog, operation log, and provider disagree after crash/edit/reindex. Documented in Claude Code (`§12`) and AGH today (`agh-current §16`). | Pick **one** authority (event log preferred). Everything else is projection with explicit repair / audit commands. |
| R3 | Dreaming before evidence | LLM consolidation promotes wrong abstractions or retries nothing after synthesis failed. goclaw has no DLQ; thresholds hand-tuned. | Slice one records signals only; expose candidate audit; dreaming starts manual/disabled and graduates with eval data. |
| R4 | Agent-manageability contradiction | "Agents cannot write" rule clashes with project rule that every feature be agent-manageable. | Expose agent write *requests* and policy-gated commits through native tools and HTTP/UDS — all routed through the controller. |
| R5 | Provider-memory double injection | Claude Code, Codex, Hermes, OpenClaw adapters already inject memory + compact + resume. AGH duplicates → stale or contradictory context. | `native_context_management` flag per adapter; default observe-and-pass-through when "confirmed". |
| R6 | sqlite-vec cgo footprint | Pure-Go / Windows builds break. | Pure-Go fallback at parity behind `[memory] vector.backend = "in-process"`; CI matrix adds non-cgo. |
| R7 | Memory poisoning via tool output | Untrusted tool output stored as fact. | Pre-write `_scan_memory_content` (Hermes regex catalog); invisible-Unicode reject; provider-output quarantine. |
| R8 | Frozen snapshot drift across long sessions | Disk state diverges from snapshot for hours. | `agh memory reload` UDS verb; auto-reload on `on_idle` past N hours; snapshot includes `mtime_ms` so divergence is observable. |

---

## 10. Open questions to resolve in the TechSpec

Ranked by criticality. These are the calls the spec author cannot punt.

1. **Source of truth.** Event log, Markdown files, or SQLite rows? If
   multiple, what is the deterministic reconciliation rule? (See §4. Codex
   peer review and the corpus both name this as the biggest fork.)
2. **Canonical scope tuple and precedence order.** `(global, workspace, agent,
   session, source_actor, namespace)` plus stable workspace identity — confirm
   exact shape and precedence, including Network federation hooks.
3. **Agent write authority.** Can an ACP agent commit, or only propose? If
   both modes exist, what config / capability decides, and how are
   contradictions / deletions authorized?
4. **Provider-native context management.** Concrete `native_context_management`
   contract for Claude Code, Codex, Hermes, OpenClaw adapters. Default policy
   when the flag is `unknown`.
5. **Compaction ownership.** Does AGH own compaction in v2, or only memory? If
   AGH owns it: checkpoint event schema, resume behavior, tool-pair
   invariants, whether to copy `Compacted.replacement_history`.
6. **Provenance model.** Source session/event IDs, source actor, confidence
   class, supersession link, fact-vs-opinion treatment — for every semantic
   memory record.
7. **Security / injection threat model.** Prompt-injection scanning, secret
   scanning, symlink/path safety, untrusted tool-output handling, provider
   boundaries, purpose/consent tags.
8. **Evaluation plan.** AGH-specific eval covering recall precision, stale-fact
   correction, contradictory memory, scope isolation, token budget, long-
   session continuity.
9. **Extension / provider contract.** One provider or many? Built-in
   precedence? Fail-open vs fail-closed? Hook namespace? Tool collision policy?
10. **Greenfield deletion list.** Hard cuts must be enumerated explicitly:
    interfaces, schema, config keys, CLI behavior, tests, docs, artifacts.
    Greenfield alpha means no aliases or shims.

---

## 11. Delete list (greenfield hard cuts)

Per `CLAUDE.md` greenfield rule, every breaking change must enumerate its
delete targets in a single commit alongside the v2 wiring. Initial list (the
TechSpec must finalize):

- **`internal/memory/types.go` — speculative context-resolution types only.**
  `ContextRefResolver`, `ContextRef`, `ContextRefKind`, `ResolvedContext`,
  `ResolvedContextItem`, `TokenBudget` get deleted; they are placeholder
  shapes that cannot survive the v2 contract. **`ProviderHookRunner`,
  `ProviderHookRequest`, `ProviderHookResult`, `ProviderHookEvent` are NOT
  deleted** — corrected from the integration map's original recommendation.
  These are the seam that becomes the `MemoryProvider` ABC (Eixo 3 of slice
  one). Extend the declared types into the 10-hook Hermes shape; do not
  re-author from scratch. Daemon delegation finally consumes them.
  (`analysis_agh-current.md` §16.)
- **`internal/memory/consolidation/*` Time→Sessions→Lock cascade — KEPT, NOT
  cut.** Corrected after pushback. The lock-rollback semantics are
  well-tested (`dream_test.go` 878 LOC, `lock_test.go` 373 LOC, `runtime_test.go`
  814 LOC). Slice 1 adds layers *on top* (signals + scoring + promotion +
  dedicated agent + DLQ); the existing gate cascade becomes the **first
  three gates** of a 4-gate pipeline. The integration map's original "hard
  cut, no aliases" was wrong — that throws away the most-tested part of the
  current subsystem.
- **`internal/memory/store.go::scoreMemoryRecall`** + **in-process
  fallback** — three independent search/scoring implementations is a
  two-touch candidate. Unify under `internal/memory/recall`.
  (`analysis_agh-current.md` §6, §16.)
- **Any `EnsureSchema` / `ensureColumn` boot reconciliation** — schema lives
  only in the numbered migrations registry. Forbidden by `agh-schema-migration`
  skill.
- **Any LLM-callable `Write` tool that bypasses the controller** — only the
  extractor sub-agent and operator CLI may write. Audit `internal/tools/*` and
  `internal/daemon/native_tools.go` for offenders.
- **`workspace_root` filepath as identity** — replace with stable
  `workspace_id` UUID stored in `globaldb`, mapped to current path. Otherwise
  moving a workspace orphans catalog rows. (`analysis_agh-current.md` §3.)
- **Doc drift in `internal/CLAUDE.md`** — current text says
  `consolidation.min_sessions=5`, code default is `3` (`dream.go:20`). Pick
  one and propagate. (`analysis_agh-current.md` §16.)
- **Declarative-only `memory.backend` extension capability** — daemon never
  delegates to extensions today (`analysis_agh-current.md` §12). Either ship
  the delegation in slice 6 *or* delete the capability declaration; declared-
  but-unconsumed surfaces are SD-009 violations.

---

## 12. Don't-lose-in-v2 list

What AGH already has that must survive (and in several cases be **extended**,
not replaced) in the redesign:

- **Lock-rollback semantics** in `internal/memory/lock.go` (273 LOC, 17
  tests) — hardlink-PID file with stale reclaim, mtime-as-`last_consolidated_at`.
  One of the better-tested parts of the current subsystem. Slice 1 keeps
  these; the new scoring layer becomes a fourth gate after Time/Sessions/Lock,
  not a replacement. (`analysis_agh-current.md` §9, §16.)
- **Dreaming runtime ticker + `EnqueueCheck`** in
  `consolidation/runtime.go` (464 LOC, 14 tests) — already supports
  hook-driven re-evaluation and ticker-based scheduling. Slice 1 extends it
  with signal-driven gating, not a rewrite.
- **Workspace-aware dream spawner** (`NewSessionSpawner` + `resolveWorkspaces`)
  — already picks recent workspaces from session metadata. Slice 1 keeps
  the resolver and adds the dedicated dreaming agent + tool whitelist.
- **`ProviderHookRunner` / `ProviderHookEvent` declared types** — corrected
  from "delete" to "extend." They become the seam for the `MemoryProvider`
  ABC. Daemon delegation finally consumes them in Slice 1 (Eixo 3).
- **Operation log integration with the observability spine** —
  `memory_operations` already mirrored in `globaldb` (`analysis_agh-current.md`
  §2). Extend with recall + skip events in Slice 1.
- **Markdown-source-of-truth + catalog-as-derived design** — half-
  implemented; FTS5 + AI/AD/AU triggers pattern works. The §4
  source-of-truth call (event-log authoritative) only re-orients the
  catalog/projection relationship — does not throw away the indexing work.
- **Closed-Type validation in `internal/memory/types.go::Header`** — keep,
  add the orthogonal source/scope/provenance axes from P3.
- **CLI/HTTP/UDS verb parity** — current implementation is a strength
  (`analysis_agh-current.md` §11). Slice 1 extends the verb list (signals,
  dream trigger/status, provider list/enable/disable, scope-show); existing
  verbs untouched.
- **`interfaces_test.go` future-seam guard test pattern** — forces explicit
  decisions about premature wiring. Keep and extend.
- **`memory_schema_migrations` registry pattern** — already in place. All
  Slice 1 schema changes go through numbered migrations. No `EnsureSchema`,
  no `ensureColumn`.

---

## 13. Next steps

1. **TechSpec #1** — `cy-create-techspec` for **Slice 1 (fat: spine +
   dreaming v2 + provider ABC + Hermes engineering parity)**. Driven by this
   analysis + `analysis_integration-map.md`, with the §5 reframe as the
   structural anchor. Must answer the 10 open questions in §10 before draft
   approval, with explicit delete-list per §11.
2. **Codex peer-review pass on TechSpec #1** — same `compozy exec --ide codex
   --model gpt-5.5 --reasoning-effort xhigh` pattern, after draft is approved
   and saved. Specifically pressure-test: (a) is Slice 1 too fat to ship in
   one cycle? (b) which Eixo is highest-risk under the AGH greenfield rule?
3. **Task generation for slice one** — `cy-create-tasks`, with the 12 QA
   proofs from §5 as the QA-execution acceptance criteria, plus
   `cy-tasks-tail-qa-pair` and `cy-web-docs-impact` per project rules. Tasks
   should be sliced by Eixo (1 → 4) to allow incremental review.
4. **Parallel design work**: agent-operable surface design (web, CLI, HTTP)
   and `agh-design` review for any UI surfaces (memory inspector, recall
   trace viewer, operation history browser, dreaming dashboard).
5. **Eval harness scaffold** — even before slice one ships, a minimal eval
   harness on synthetic transcripts for the 12 QA proofs, plus
   recall-precision baseline so Eixo 2 weight tuning has a control.
6. **Competitive parity check at slice 1 end** — explicit pass against
   `analysis_hermes.md` checklist: every Hermes capability either present in
   AGH, or explicitly marked deferred-with-reason. No silent gaps.

Slices 2–6 each get their own TechSpec, gated on slice one's QA passing AND
the competitive-parity check.

---

## 14. Reference map

- Source corpus: `.compozy/tasks/mem-v2/analysis/analysis_*.md` (10 files).
- Integration verdicts: `analysis_integration-map.md`.
- Codex peer review (raw, token-interleaved but legible):
  `_codex_peer_response.md`. Distilled corrections incorporated above.
- Project rules: `/CLAUDE.md`, `/internal/CLAUDE.md`,
  `/docs/_memory/standing_directives.md` (SD-009 observability,
  SD-010 composition-root, SD-011 extensible-and-agent-manageable).
- AGH today: `internal/memory/`, `internal/store/globaldb`,
  `internal/store/sessiondb`, `internal/hooks/`, `internal/situation/`,
  `internal/extension/contract`, `internal/cli/`, `internal/api/`.
- Glossary: `docs/_memory/glossary.md` (capability vs recipe; AGENT.md vs
  AGENTS.md).

---

*End of overarching analysis. ~600 lines. The next artifact is a TechSpec, not
another analysis.*
