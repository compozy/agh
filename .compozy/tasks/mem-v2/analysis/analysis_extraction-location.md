# Fact Extraction Location — Analysis (mem-v2)

**Question.** Where does "transform a turn into memory candidates" run? When? In what context? Pick a verdict for AGH and defend it.

**Time-boxed verification of:** Claude Code (`extractMemories.ts`), OpenClaw (`agent-runner-memory.ts` + `memory-flush.ts`), Hermes (`tools/memory_tool.py`), Codex (`memories/write/{lib,start,phase1,phase2}.rs` + templates).

---

## 1. TL;DR — recommendation

**Pick D2 (hybrid A + B): a daemon-emitted `on_post_response` hook fires a forked extractor *every turn* (cheap, lightweight, throttled), and a *secondary* compaction-time silent flush runs *only* when the session's transcript hits the daemon-managed compaction watermark.** Extraction lives in a new `internal/memory/extractor` package, dispatched from the call site that owns the assistant-message persistence transition (per the daemon's "hooks dispatch at call site, never tail event tables" rule). The forked extractor runs as a *bounded* subagent (Claude Code shape — read-only sandbox + Edit/Write restricted to `~/.agh/memory/_inbox/`), single-slot in-progress + one-stashed trailing run, drained on shutdown. Codex-style next-session-boot (D) is rejected as the *primary* path: it forces 30-day backlog scans, doubles cold-start cost on resume, and contradicts AGH's "agents-first, latency-fair" runtime stance. Inside `Controller.Decide()` (C) is rejected because `analysis_write-controller.md` already established that extraction must run *outside* the controller — the controller decides ADD/UPDATE/DELETE/NOOP/REJECT against persisted neighbours; extraction is upstream of that decision. The **A→B fallback chain** matches AGH's gate cascade philosophy (Time → Sessions → Lock — see `internal/CLAUDE.md`): per-turn = Time, compaction-flush = Sessions/size, both feed a single bounded queue read by `Controller.Decide`.

---

## 2. Per-system mechanism (code-cited)

### 2.1 Claude Code — `on_post_response` forked sub-agent

- **Trigger.** "End of each complete query loop (when the model produces a final response with no tool calls) via `handleStopHooks` in `stopHooks.ts`" (`extractMemories.ts:1-7`). Public entry point `executeExtractMemories` is "called fire-and-forget from `handleStopHooks`" (`extractMemories.ts:594-602`).
- **Where it runs.** `runForkedAgent({ ... forkLabel: 'extract_memories', skipTranscript: true, maxTurns: 5 })` (`extractMemories.ts:415-427`). A *perfect fork* of the main conversation: shares the parent's prompt cache via `createCacheSafeParams(context)` (`extractMemories.ts:48-50, 372`); does **not** record to transcript ("can create race conditions with the main thread", `extractMemories.ts:421-423`). Hard cap of **5 turns** because "well-behaved extractions complete in 2-4 turns (read → write)" (`extractMemories.ts:424-427`).
- **Tool sandboxing.** `createAutoMemCanUseTool(memoryDir)` (`extractMemories.ts:171-222`) — Read/Grep/Glob unrestricted (`:184-191`); Bash only when `tool.isReadOnly(parsed.data)` (`:195-204`); Edit/Write only for paths inside `memoryDir` via `isAutoMemPath(filePath)` (`:206-215`); REPL allowed because primitive tools route through it (`:173-182`); everything else denied (`:217-220`).
- **Throttling.** `tengu_bramble_lintel` feature flag controls "every N eligible turns" (default 1 — i.e. every turn) (`extractMemories.ts:374-385`). Trailing runs from the stashed-context pattern *bypass* throttle because "they process already-committed work" (`:376-385`).
- **Sub-agent handling.** Hard skip: `if (context.toolUseContext.agentId) { return }` — "Only run for the main agent, not subagents" (`extractMemories.ts:531-534`). Sub-agent turns inherit nothing; only root-session Stop fires extraction.
- **Async vs sync.** Fire-and-forget from Stop (`:594-602`); the user-visible response has *already flushed* by then. `drainPendingExtraction(timeoutMs = 60_000)` is awaited by `print.ts` "after the response is flushed but before `gracefulShutdownSync`, so the forked agent completes before the 5s shutdown failsafe kills it" (`extractMemories.ts:605-614`, called at `cli/print.ts:968`).
- **Failure mode.** "Extraction is best-effort — log but don't notify on error" (`:497-502`). Cursor stays put on failure so messages are reconsidered next turn (`:429-435`).
- **Output destination.** Forked agent calls Edit/Write itself (sandboxed) into `~/.claude/projects/<path>/memory/` (`extractMemories.ts:1-3, 339`). Written paths returned to caller for `appendSystemMessage` notification (`:455-496`).
- **Single-slot pending.** When a new call arrives during in-flight extraction, **the stashed context is overwritten silently** — "overwrites any previously stashed context — only the latest matters since it has the most messages" (`:553-563`). One trailing run guaranteed via `finally` (`:506-522`). **This is the exact pattern Codex peer review §3.7 flagged as unsafe for AGH** (silent drop without observability).
- **Mutual exclusion with main agent writes.** `hasMemoryWritesSince(messages, lastMemoryMessageUuid)` short-circuits extraction when the main thread already wrote (`:121-148, 348-360`). Prevents double-write races.

### 2.2 OpenClaw — silent pre-compaction flush

- **Trigger.** Two AND-gated triggers, both checked inside the reply pipeline before the next turn runs:
  1. **Token threshold.** `tokenCount >= contextWindow - reserveTokensFloor - softThresholdTokens` (default `softThresholdTokens = 4_000`, `agent-runner-memory.ts:429, 656-657`; `memory-flush.ts:70-95`).
  2. **Transcript byte size override.** `transcriptByteSize >= forceFlushTranscriptBytes` (`agent-runner-memory.ts:678-698, 786-789`). Both AND-gated by `!isHeartbeat`, `!isCli`, `memoryFlushWritable` (sandbox=rw), and `!hasAlreadyFlushedForCurrentCompaction(entry)` (`memory-flush.ts:118-124`, dedupes within a single compaction cycle).
- **Where it runs.** `runEmbeddedPiAgent({ ... silentExpected: true, trigger: "memory", memoryFlushWritePath, prompt: activeMemoryFlushPlan.prompt, ... })` (`agent-runner-memory.ts:847-872`). A **silent embedded turn** in the same agent runtime, sharing `replyOperation.abortSignal`. Phase is set to `"memory_flushing"` first (`:799`), so the user-visible UI marks compaction.
- **Tool sandboxing.** Not restricted at the runner level — the silent agent runs with the same tool surface as the main agent. Defense is the system prompt (`activeMemoryFlushPlan.systemPrompt` + `flushSystemPrompt`, `:821-826`) which instructs the model to write only to `memoryFlushWritePath`. Plus `memoryFlushWritable` precondition: "if sandboxed, requires `workspaceAccess === 'rw'`" (`:616-629`).
- **Throttling.** `hasAlreadyFlushedForCurrentCompaction` — only one flush per `compactionCount` value (`memory-flush.ts:118-124`). Compaction cycle increments after a successful run (`agent-runner-memory.ts:889-921`).
- **Sub-agent handling.** N/A — flush is gated on the main session entry; embedded sub-agents emit `compaction` stream events but do not themselves trigger flushes.
- **Async vs sync.** **Synchronous**: blocks the next turn until flush completes. `replyOperation.setPhase("memory_flushing")` is observable on the wire (`:799`); the user sees a "memory flushing" badge during the silent turn. Tied to `replyOperation.abortSignal`, so user-cancel kills the flush.
- **Failure mode.** Outer `try/catch`: `logVerbose("memory flush run failed: …")` and continues (`:944-946`). Persistence-update failures logged separately (`:940-942`).
- **Output destination.** Flush plan's `relativePath` (e.g. `memory/<file>.md` per `loader.runtime-registry.test.ts:246`, `memory-state.test.ts:125`). The plan is registered by a memory plugin via `registerMemoryCapability(...)` (`memory-state.ts:235-244`). Markdown file the silent agent writes directly.
- **Important.** Compaction-count vs memory-flush-count is tracked separately — flush runs *before* compaction, then `incrementCompactionCount` runs only if the silent agent emitted `compaction.end` (`:884-921`). Couples extraction to compaction tightly.

### 2.3 Hermes — in-context tool only (no separate extractor)

- **Trigger.** **No autonomous extractor exists.** The main agent calls the `memory` tool itself when it decides a fact is worth saving. Schema description (`memory_tool.py:515-563`) tells the model *when* to save: "User corrects you", "User shares a preference", "You discover something about the environment", "You learn a convention", "You identify a stable fact" (`:521-527`).
- **Where it runs.** Inside the main agent's tool-call loop. `memory_tool(action, target, content?, old_text?)` is a deterministic dispatcher (`:465-503`) — no LLM in the runtime. Action is one of `add`/`replace`/`remove` (`:483-501`).
- **Tool sandboxing.** Not relevant — the main agent already has whatever sandbox the operator configured. The runtime guards are the threat-pattern regex, char-budget hard cap, and substring-match invariants (per `analysis_write-controller.md` §3, citing `memory_tool.py:67-122, 243-301`).
- **Throttling.** None — agent decides per-turn.
- **Sub-agent handling.** Sub-agents share the main store via the injected `store=kw.get("store")` (`:574-580`); they can write directly. No isolation.
- **Async vs sync.** Sync — runs inline as a tool call.
- **Failure mode.** Returns `tool_error("...", success=False)` JSON; the LLM sees the error and decides whether to retry (`:478, 481, 485, 491-498, 501`).
- **Output destination.** Two files: `MEMORY.md` and `USER.md` under `~/.hermes/` (per `analysis_hermes.md` §3 + `memory_tool.py:121-122` for char budgets).
- **Verdict.** No "extractor" — the model is the extractor. Works only because Hermes intentionally caps memory at 2200 + 1375 chars and snapshots it into the system prompt (frozen-snapshot pattern). Cannot scale past hundreds of entries; admitted in `analysis_hermes.md`.

### 2.4 Codex — deferred two-phase at next-session boot

- **Trigger.** **At root-session start**, *not* during the session being recorded. `start_memories_startup_task(thread_manager, auth_manager, thread_id, thread, config, source)` is called from session bootstrap (`memories/write/src/start.rs:22-75`). Skipped when `config.ephemeral || !features.enabled(MemoryTool) || source.is_non_root_agent()` (`:30-35`).
- **Where it runs.** `tokio::spawn` detached background task (`start.rs:51`). Inside the daemon process, **not** a forked sub-agent. Two phases:
  - **Phase 1 (extraction).** Per *eligible past rollout*, claim a `Stage1Job` from the state DB (`phase1.rs:148-183`), feed the rollout text to a dedicated `gpt-5.4-mini` (low reasoning effort) call with `stage_one_system.md` (`lib.rs:79-101` for constants; template at `templates/memories/stage_one_system.md`). Concurrency limit **8** (`lib.rs:82`). Output is JSON `{rollout_summary, rollout_slug, raw_memory}` (`phase1.rs:51-63`). `redact_secrets` applied to all three fields before persistence (`phase1.rs:313-315`).
  - **Phase 2 (consolidation).** Dedicated *spawned consolidation agent* under a sandboxed config (`phase2.rs:78-90`), reads workspace diff (`phase2_workspace_diff.md`, `lib.rs:113-115`), writes/edits `memories/*.md` files (`phase2.rs:114-125`). Model `gpt-5.4` Medium reasoning (`lib.rs:103-110`). Single global lock per claim (`phase2.rs:55-63`).
- **Tool sandboxing.** Phase 1 has no tools — pure JSON output from a structured-output model call (schema: `phase1.rs:135-146`). Phase 2 runs a real agent under `agent::get_config(...)` returning a *locked-down* sandbox (`phase2.rs:78-90`); details live in `agent.rs` but the comment is "locked-down config used by the consolidation agent".
- **Throttling.** `JOB_LEASE_SECONDS = 3_600`, `JOB_RETRY_DELAY_SECONDS = 3_600` (`lib.rs:83-84`). State DB tracks per-rollout claims so the same rollout never extracts twice. `THREAD_SCAN_LIMIT = 5_000` and `max_rollouts_per_startup` config cap (`phase1.rs:148-183`). Plus `min_rollout_idle_hours` (must be idle N hours before extraction, `phase1.rs:170`).
- **Sub-agent handling.** **`source.is_non_root_agent()` skip** (`start.rs:32`) — only root sessions trigger startup pipeline. Sub-agent rollouts are *eventually* picked up through the state-DB lease (any future root-session boot can claim them).
- **Async vs sync.** Fully async — `tokio::spawn` detaches from session boot (`start.rs:51-74`). Rate-limit guard checked first; if quota exhausted, increments `MEMORY_STARTUP[status=skipped_rate_limit]` counter and bails (`start.rs:61-68`).
- **Failure mode.** Per-job: `JobOutcome::Failed` aggregated into stats (`phase1.rs:35-48, 100-107`). Phase 2 lock-claim failure logs and returns; workspace prep failure marks job failed (`phase2.rs:55-90`). Idempotent retry via state-DB lease.
- **Output destination.** Phase 1 → `state-db.stage1_outputs` rows (durable, leased). Phase 2 → file edits in `~/.codex/memories/` (`memory_root` = `codex_home.join("memories")`, `lib.rs:118-120`); `raw_memories.md` and `rollout_summaries/` (`lib.rs:35-39, 122-128`).
- **Retention.** `DEFAULT_MEMORIES_MAX_UNUSED_DAYS: i64 = 30` (`config/src/types.rs:50`), used to prune dead stage-1 rows (`phase1.rs:111-132`) and to scope phase-2 input selection (`phase2.rs:54, 94`). Extension resources retained 7 days (`lib.rs:43`). The 30-day window is the *lookback* for new boots.

---

## 3. Comparative matrix

| System | Trigger | Async/Sync | Sandbox | Sub-agent rule | Single-slot policy | Latency impact | Infra cost |
|---|---|---|---|---|---|---|---|
| **Claude Code** | Stop hook (per turn, throttle `tengu_bramble_lintel`) | Async fire-and-forget; drained on shutdown | Forked sub-agent: Read/Grep/Glob unrestricted; Bash read-only; Edit/Write only inside `memoryDir`; deny everything else | **Skip**: `if (toolUseContext.agentId) return` — root only | **Silent overwrite** of stashed context (1 in-flight + 1 stashed) — peer-review issue | Zero on user response (already flushed); cost shows in shutdown drain (60s timeout) | Forked sub-agent inherits parent prompt cache — high cache hit rate, but a full Sonnet turn (~2-4 model calls) per Stop |
| **OpenClaw** | Token-threshold OR transcript-byte threshold, gated by `!hasAlreadyFlushedForCurrentCompaction` | **Sync** — blocks next turn, sets `phase: memory_flushing` | None at runner level; relies on system-prompt instructions + workspace=rw precondition | None | One flush per `compactionCount` (deterministic dedup) | High — silent turn happens *during* the next reply | One full agent turn on threshold; tied to compaction frequency (~ once per session for typical sessions) |
| **Hermes** | Main agent decides to call `memory(action, ...)` mid-turn | Sync (tool call) | Threat-regex + char-budget + substring-uniqueness + atomic write + flock | Sub-agents share store; no isolation | N/A (no extractor) | Zero extraction cost; the tool call *is* the extraction | Zero — no separate model run |
| **Codex** | Root-session boot via `start_memories_startup_task` (deferred) | Async `tokio::spawn` | Phase 1: structured-output only (no tools); Phase 2: locked-down sandbox agent | **Skip**: `source.is_non_root_agent()` — only root sessions | State-DB lease (`JOB_LEASE_SECONDS=3_600`) — durable, not silent | Cold-start cost on resume; zero during active session | High — `gpt-5.4-mini` ×8 concurrent per claimed rollout, plus a `gpt-5.4` Medium phase-2 agent per consolidation cycle |

---

## 4. AGH constraints (load-bearing)

1. **SD-010 — detached lifetime.** Long-running work (extractor model calls) MUST run under `context.WithoutCancel(parent)` with an explicit deadline re-attached. Tying extraction to the prompt-request context kills it the moment the user's HTTP/UDS request returns. (`internal/CLAUDE.md` Concurrency section, repeated in `docs/_memory/standing_directives.md`.)
2. **Subagents are read-only by default.** The subagent skill rule (`docs/_memory/standing_directives.md`, repeated in repo `CLAUDE.md`) says subagents "do analysis/research only; the paired agent is sole code author." For an *extractor* this matters because: the extractor must write to `~/.agh/memory/_inbox/` — but ONLY there, never the operator's project repo, never another session's transcripts. Same shape as Claude Code's `createAutoMemCanUseTool`.
3. **Hooks dispatch at call site, never tail event tables.** (`internal/CLAUDE.md` Architecture / Concurrency.) Extraction MUST be dispatched by the package that owns the assistant-message persistence transition (`internal/session/manager_lifecycle.go` or `internal/transcript`). Forbidden: a goroutine that polls `events.db` for `assistant_message` rows.
4. **Bounded queue with observable dropped windows** (Codex peer review §3.7, `_codex_peer_response.md`). A silent single-slot overwrite (Claude Code shape) is unacceptable. AGH must use a small bounded channel and emit a canonical event when an extraction is dropped/coalesced — or merge new turns into the stashed payload via a deterministic merge.
5. **Hot-path latency.** Extraction MUST NOT block the user-facing turn response. OpenClaw's sync flush is a non-starter for AGH's "agents-first runtime" stance. Even a 200 ms extractor-spawn delay on every Stop is unacceptable at the runtime tier where a single workspace can run ≥10 concurrent agent sessions.

---

## 5. Three candidates evaluated for AGH

(Dropping C — `analysis_write-controller.md` already established that extraction is upstream of `Decide`.)

### 5.A — `on_post_response` hook handler with forked extractor (Claude Code shape)

**Pros.**
- Per-turn cadence keeps the candidate stream fresh; no 30-day backlog at boot.
- Zero user-visible latency (fire-and-forget after assistant-message persistence).
- Aligns with AGH's existing typed hook taxonomy (`internal/hooks` already dispatches at call site).
- Forked subagent reuses parent prompt cache → cheap.
- Already battle-tested for cache stability + sub-agent skip.

**Cons.**
- Per-turn extractor cost adds up across many concurrent sessions. With 10 active sessions doing 3 turns/min each, that's 30 extractor turns/min — non-trivial.
- Silent-drop risk on overlapping calls (Claude Code's bug — must be replaced with a bounded-queue drop event).
- Doesn't catch *cross-rollout* patterns the way Codex's deferred consolidation does. (Mitigated by the existing `internal/memory/consolidation` package, which is the *consolidation* gate, not the extraction gate.)
- Forked-subagent cost requires either prompt-cache reuse (provider-dependent — some ACP drivers don't expose cache reads) OR a smaller dedicated extractor model (Codex shape).

### 5.B — Pre-compaction silent flush (OpenClaw shape)

**Pros.**
- Triggers exactly when the agent is about to lose recent context — natural "save it before it's gone" moment.
- Single-flush-per-compaction-cycle dedup is deterministic and observable.
- Couples cleanly to the daemon's existing compaction cascade.

**Cons.**
- Sync-blocks the next turn — directly violates AGH hot-path-latency rule.
- Depends on a working compactor (Slice 2 deliverable) — not available for the slice that ships extractors.
- Misses turns in *short* sessions that never hit compaction threshold (the most common operator session: short clarification chats).
- Tightly couples extraction policy to context-window math; tuning becomes provider-dependent.

### 5.D — Deferred two-phase at next-session boot (Codex shape)

**Pros.**
- Zero in-session cost — no impact on user-facing turn latency.
- Durable state-DB lease pattern resists session crashes.
- Per-rollout claims naturally serialize concurrent extractors.
- Catches `min_rollout_idle_hours` patterns — i.e. *only* extracts from sessions known to be done, never racing with active work.
- Redaction baked in (`redact_secrets` on all phase-1 outputs).

**Cons.**
- **Cold-start cost on resume.** When the operator restarts AGH after a heavy day, every root session pays a non-trivial extraction backlog. Codex pays this with a `tokio::spawn` and rate-limit guard, but for AGH's local-first single-binary runtime this is felt by the operator: high CPU on a fresh boot.
- 30-day window means a freshly-installed AGH has zero memory until the user has used it for at least one full session and *then* booted a second session.
- Two model calls per rollout (Phase 1 mini + Phase 2 medium) — high tokens-per-extraction.
- Recent learnings *aren't available* in the session that produced them — only in the *next* session. Hurts UX for "AGH should remember what we decided 5 minutes ago."

### 5.D2 — Hybrid A + B (RECOMMENDED)

**Pros.**
- A handles the dominant case: cheap, frequent, per-turn extraction → candidates available *during the same session* (matches operator UX).
- B handles the heavy case: when context is about to be lost, do a *stronger* consolidation pass that re-reads the full transcript window and produces denser candidates.
- Two complementary cadences map exactly to AGH's existing **gate cascade** (Time → Sessions → Lock):
  - A = Time gate (every-N-turns).
  - B = Sessions/size gate (compaction watermark).
  - A and B both write to the same `_inbox` queue, read by `Controller.Decide`.
- The pre-compaction flush *reuses* candidates A already produced — it doesn't redo the work, it consolidates.

**Cons.**
- Two code paths to maintain (mitigated: B is an explicit subset of A's machinery, called from the daemon's compaction trigger).
- Tuning surface is larger (per-turn N, plus token threshold, plus byte threshold).

---

## 6. Verdict for AGH

**Verdict: D2 (A + B) with A as the default, B optional in Slice 2.**

**Specifically:**

- **Where the extractor lives.** New package `internal/memory/extractor`. It exposes `Extractor` (interface) and `Forked` (concrete implementation that spawns a bounded ACP subagent). Wired into `internal/daemon` composition root. **Dispatched from** `internal/session/manager_lifecycle.go` at the call site that finalizes assistant-message persistence (so it follows "hooks dispatch at call site, never tail event tables" — `internal/CLAUDE.md`).

- **Hook event.** New typed event `hook.session.message_persisted` (or `on_post_response` if we prefer Claude Code's name) added to `internal/hooks` taxonomy. The extractor is registered as a built-in hook *consumer*; operators can also register their own hooks at the same event for telemetry. Event payload includes `session_id`, `root_session_id`, `agent_id`, `actor_kind`, `message_seq`, `parent_session_id` for proper sub-agent skip filtering.

- **Sandbox tool whitelist.** Read-only by default. Allow:
  - `read`, `grep`, `glob` unrestricted.
  - `bash` only for read-only commands (mirrors Claude Code's `tool.isReadOnly` check).
  - `edit`/`write` only for paths under `$AGH_HOME/memory/_inbox/` (and only when the candidate path passes `isAGHInboxPath` — the AGH analogue of `isAutoMemPath`).
  - REPL allowed *if* AGH ever ships REPL — same reason as Claude Code.
  - Everything else **denied**, with a `denyMemoryExtractorTool(tool, reason)` event so denials are observable.

- **Throttle rule.** `extractor.throttle_turns: 1` (default) — every turn. Configurable per-agent in `agent.memory.extractor.throttle_turns`. Optional token/byte threshold (`extractor.threshold.context_tokens`, `extractor.threshold.transcript_bytes`) for the secondary B-mode flush. Trailing runs from queued context bypass throttle.

- **Sub-agent rule.** **Skip extraction on sub-agent sessions.** Use `actor_kind != "agent_subagent"` AND `parent_session_id == nil` (root-only) — same shape as Codex's `source.is_non_root_agent()` and Claude Code's `agentId` check. The root agent *inherits* the right to extract on behalf of the entire spawn tree because root has full transcript visibility through the daemon's transcript projector. Sub-agent message-persisted events still dispatch the hook (for telemetry), but the extractor consumer no-ops.

- **Async semantics (SD-010).**
  - Fire-and-forget spawn: `go func() { ctx := context.WithoutCancel(parent); ctx, cancel := context.WithDeadline(ctx, time.Now().Add(extractor_deadline)); defer cancel(); ... }()`.
  - Default `extractor_deadline = 60s` (configurable).
  - Manager-owned `WaitGroup` tracks in-flight extractions; daemon shutdown calls `Drain(ctx, drain_timeout)` (default 60s, mirrors Claude Code).
  - Re-attach deadline because `context.WithoutCancel` does NOT preserve deadlines (`internal/CLAUDE.md`).

- **Failure mode.**
  - Best-effort: log + emit canonical event `memory.extractor.failed` (correlation keys: `session_id`, `root_session_id`, `agent_id`, `attempt`, `error_class`).
  - **Dead-letter queue at `$AGH_HOME/memory/_system/extractor/failures/<utc>-<session_id>.json`** — contains the turn payload + error. Replayable by `agh memory extractor replay --session <id>` (CLI parity is mandatory per "agent-manageable by default").
  - Cursor (`last_extracted_message_seq`) does NOT advance on failure, so the next turn re-processes the same range (Claude Code shape).

- **Single-slot pending policy (Codex peer-review §3.7 fix).**
  - Bounded `chan extractRequest` with capacity 1 (one in-flight + one queued) per session.
  - On overflow, the *queued* slot is **merged** with the new turn (concat message ranges) rather than silently overwritten, AND a canonical event `memory.extractor.coalesced` is emitted with `dropped_turn_count`. Operators have observability.
  - Hard ceiling: if coalesced count exceeds a threshold (default 16), emit `memory.extractor.dropped` and drop the oldest, retain newest — but the drop is *visible*.

- **Output destination.**
  - Extractor writes candidate JSONL files to `$AGH_HOME/memory/_inbox/<session_id>/<utc>-<message_seq>.jsonl`.
  - Each line is one `Candidate` (`{type, scope, content, evidence, source_seq_range}`).
  - Candidates are NOT memories yet — they are *proposals*. `Controller.Decide()` (separate package, per `analysis_write-controller.md`) reads from `_inbox`, runs deterministic prefilter + LLM tiebreaker, and either commits (ADD/UPDATE/DELETE) or rejects (NOOP/REJECT) into `internal/memory`'s authoritative store.
  - `_inbox` is a queue — `Controller.Decide` deletes processed lines.
  - Two-phase split lets us add Codex-style consolidation later without rearchitecting the extractor.

- **Mutual exclusion with explicit memory-tool calls.** If the *main agent* calls the `memory` tool in the same turn (Hermes shape — `memory_tool` action `add`/`update`/`delete` already exists at `internal/memory`), the extractor for that turn is a no-op (analogous to Claude Code's `hasMemoryWritesSince`). Prevents double-write race.

---

## 7. AGH-specific contract sketch

```go
// internal/memory/extractor/extractor.go
package extractor

// Extractor is the single point of "turn → candidates" transformation.
//
// Implementations MUST be safe to call concurrently across sessions but MUST
// serialize per-session via an internal bounded queue with at most 1 in-flight
// + 1 queued. Overflow merges the queued slot rather than silently dropping;
// a canonical hook.dispatch event is emitted on coalesce.
type Extractor interface {
    // Extract runs in a detached context (caller wraps with WithoutCancel +
    // WithDeadline). Returns candidates that the controller will evaluate
    // against the persisted store. Empty slice + nil err is a valid no-op
    // (minimum-signal gate, mirroring Codex stage_one_system.md NO-OP rules).
    Extract(ctx context.Context, turn TurnRecord) ([]Candidate, error)
}

// TurnRecord is the input to the extractor — the canonical assistant-message
// persistence event payload, enriched with the message-seq range covered.
type TurnRecord struct {
    SessionID         string
    RootSessionID     string
    ParentSessionID   string // empty for root sessions
    AgentID           string
    ActorKind         string // "agent_root" | "agent_subagent" | ...
    WorkspaceID       string
    SinceMessageSeq   int64  // exclusive
    UntilMessageSeq   int64  // inclusive
    Snapshot          TranscriptSnapshot // resolved by transcript package
    Trigger           Trigger // "post_message" | "compaction_flush"
}

type Trigger string
const (
    TriggerPostMessage     Trigger = "post_message"     // mode A
    TriggerCompactionFlush Trigger = "compaction_flush" // mode B
)

// Candidate is the unit produced by the extractor and consumed by the
// controller. The controller decides ADD/UPDATE/DELETE/NOOP/REJECT against
// the persisted store; the extractor never commits directly.
type Candidate struct {
    ID            string         // ULID assigned by extractor
    Type          memory.Type    // user | feedback | project | reference
    Scope         memory.Scope   // agent | workspace | global
    Content       string
    Evidence      Evidence       // turn provenance: message seqs, quoted spans
    Confidence    float32        // 0..1, extractor's self-rated confidence
    Tags          []string
}

type Evidence struct {
    SessionID         string
    RootSessionID     string
    AgentID           string
    SourceMessageSeqs []int64
    Quotes            []Quote
    GeneratedAt       time.Time
}
```

**Config keys** (`config.toml`, validated in `internal/config`):

```toml
[memory.extractor]
enabled = true
mode = "post_message"            # post_message | compaction_flush | hybrid (default hybrid in Slice 2)
throttle_turns = 1
deadline_seconds = 60
sandbox_inbox_only = true        # forced true; key reserved for future loosening
inbox_path = "$AGH_HOME/memory/_inbox"
dlq_path  = "$AGH_HOME/memory/_system/extractor/failures"

[memory.extractor.queue]
capacity = 1                     # in-flight + queued; coalesce on overflow
coalesce_max = 16                # hard drop threshold; emits memory.extractor.dropped

[memory.extractor.compaction_flush]   # mode B (Slice 2)
soft_threshold_tokens = 4000
reserve_tokens_floor = 8000
force_flush_transcript_bytes = "8MiB"
```

**CLI surface** (mandatory per "agent-manageable by default"):

```
agh memory extractor status [--session <id>] [-o json]
agh memory extractor list-pending [-o jsonl]
agh memory extractor replay --session <id> [--from-dlq]
agh memory extractor drain [--timeout 60s]
agh memory extractor disable [--session <id>]   # operator override
```

HTTP/UDS parity for each.

---

## 8. Open sub-questions (for the TechSpec)

1. **Forked subagent vs dedicated lightweight model.** Claude Code reuses a Sonnet fork (cache-friendly, expensive). Codex uses `gpt-5.4-mini` Low. Should AGH default to a forked main-model subagent (cache hit advantage on Anthropic ACP drivers), or a configurable `extractor.model = "haiku-class"` to keep cost bounded across all ACP drivers (some won't expose cache reads)? Decision affects the tool-sandbox shape — a dedicated model wouldn't share the parent prompt and could use a tighter, structured-output-only contract.
2. **Coalesce-merge semantics.** When the queue overflows and we merge instead of overwrite, *how* do we merge two `TurnRecord`s? Concatenate `[since, until]` ranges? Take the union? What if there's a gap (the extractor missed a turn)? Need a deterministic merge rule that the controller can reason about.
3. **Sub-agent inclusion policy for B-mode.** Codex skips sub-agents at extraction time but Phase 2 consolidation eventually folds them in via state-DB scan. For AGH's compaction-flush mode, do we extract sub-agent transcripts inline as part of the root's flush (since root's transcript projector has visibility), or do we run a separate per-sub-agent flush?
4. **Per-rollout claim vs per-turn claim.** Codex's lease pattern is per-rollout (a whole conversation is one job). Per-turn claims could be much smaller but would generate more state DB rows. Does AGH's `runtime.db` schema (per `internal/store`) want per-turn claim rows, or per-session?
5. **Cross-session deduplication of identical candidates.** Two parallel sessions saying "the user prefers gofumpt over gofmt" will both extract the same fact. Controller.Decide will NOOP the second one — but the extractor still pays the model-call cost. Worth a content-hash short-circuit before the model call (cheap dedup on the inbox), or leave dedup to the controller?

---

**Files referenced (for fast jump-back):**
- `/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/services/extractMemories/extractMemories.ts:1-7,121-148,171-222,329-577,594-614`
- `/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/src/auto-reply/reply/agent-runner-memory.ts:611-948`
- `/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/src/auto-reply/reply/memory-flush.ts:70-141`
- `/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/src/plugins/memory-state.ts:235-244`
- `/Users/pedronauck/Dev/compozy/agh/.resources/hermes/tools/memory_tool.py:121-122,243-301,465-580`
- `/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/memories/write/src/start.rs:22-75`
- `/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/memories/write/src/lib.rs:35-115`
- `/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/memories/write/src/phase1.rs:30-200,313-315`
- `/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/memories/write/src/phase2.rs:1-125`
- `/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/memories/write/templates/memories/stage_one_system.md`
- `/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/config/src/types.rs:50,248,272,288`
- `/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/state/src/runtime/memories.rs:128`
- `/Users/pedronauck/Dev/compozy/agh/.compozy/tasks/mem-v2/analysis/analysis_write-controller.md` (controller scope)
- `/Users/pedronauck/Dev/compozy/agh/internal/CLAUDE.md` (hook-call-site rule, SD-010, gate cascade)
- `/Users/pedronauck/Dev/compozy/agh/internal/hooks/dispatch_events.go:11-31` (dispatch phase taxonomy AGH already exposes)
