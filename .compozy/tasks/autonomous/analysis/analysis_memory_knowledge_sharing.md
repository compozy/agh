# Memory & Shared Knowledge — What's Missing for Autonomous Agents

**Slice:** Memory and shared knowledge across agents and sessions.
**Premise:** Without persistent shared memory, autonomy collapses into rediscovery — every agent re-learns peers, channel norms, prior decisions, and capability outcomes from scratch every turn.

---

## 1. TL;DR

- AGH has a solid, **single-agent**, two-scope persistent memory: `global` (user/feedback) + `workspace` (project/reference), Markdown files with YAML frontmatter, FTS5 catalog, dream consolidation runtime, automatic recall on every prompt (`internal/memory/recall.go:22-59`, `internal/memory/store.go:36-46`, `internal/memory/catalog.go:34-87`).
- Autonomy is blocked at the **collective** layer: there is no peer-fact store, no per-channel blackboard, no skill/recipe outcome history, no episodic session-summary memory, no cross-agent shared file the dream pass can write to. Memory is "what the user told *one* agent about the project," not "what *we* (a swarm) have learned together."
- Writes are mostly user/operator-initiated. There is **no automatic per-turn extraction**, no pre-compaction flush, no fallback extractor (`docs/ideas/memory-gaps/README.md:74-98`). Comparable harnesses (claude-code `extractMemories`, hermes `MemoryProvider.sync_turn`) form memory continuously; AGH does not.
- The "agent" in `Header.AgentName` (`internal/memory/types.go:57`) is plumbed but never set by mutations or filtered on read — every memory effectively belongs to "daemon" (`internal/memory/catalog.go:31, 545-548`). Agents can write but can't be told *which agent wrote what*.
- Network primitives (`channel`, `peer_card`, `interaction_id`, `trace_id`) exist (`internal/store/globaldb/global_db_network_channels.go`, `internal/store/globaldb/global_db_network_messages.go`) but **never feed memory**. A channel is a transient pub/sub room with no durable shared knowledge layer, no peer-fact ledger, no echo/reputation digest.
- Recommended additions, in order of leverage: (a) **peer-fact store** scoped per-(observer, peer); (b) **channel/space blackboard** as a third memory scope; (c) **session episodic summaries** auto-generated at session end; (d) **skill/capability outcome history** (success rate, last failure, calling agent); (e) **automatic per-turn extractor** with `agent_name` provenance; (f) **session-start auto-injection** keyed on `(workspace, channel, peers, agent)`.

---

## 2. Current Memory Model

### 2.1 Scopes

Two scopes only — declared in `internal/memory/types.go:25-33`:

```go
const (
    ScopeGlobal    Scope = "global"
    ScopeWorkspace Scope = "workspace"
)
```

Default-scope-by-type rule (`internal/memory/types.go:236-248`):

| Type        | Default scope | Intent                              |
| ----------- | ------------- | ----------------------------------- |
| `user`      | global        | preferences, working style          |
| `feedback`  | global        | recurring corrections               |
| `project`   | workspace     | code/architecture decisions         |
| `reference` | workspace     | external docs/system pointers       |

There is **no `session` scope**, no `channel` scope, no `peer` scope, no `agent` scope, no `team` scope.

### 2.2 Storage layout

- Filesystem: `<global_dir>/*.md` and `<workspace_root>/.compozy/memory/*.md` (`internal/memory/store.go:1000-1007`, derivation in `:1105-1115`).
- Index: `MEMORY.md` per scope, regenerated after every mutation (`internal/memory/store.go:561-579`, max 200 lines / 25KB — constants at `:24-25`).
- Frontmatter schema: `name`, `description`, `type`, **`agent_name`** (`internal/memory/types.go:50-58`). `agent_name` is parsed and stored in the catalog row but **never set by `Write` callers and never used as a query filter** (no occurrences in `recall.go`, `assembler.go`, or HTTP handlers).
- Derived catalog: SQLite FTS5 in the global DB — table `memory_catalog_entries` with `(scope, workspace_root, filename)` PK + virtual `memory_catalog_fts` (`internal/memory/catalog.go:34-87`).
- Operation log: `memory_operation_log` (`:78-86`, schema migration v2 at `:96-101`) with `agent_name` column that is *always* hard-coded to `"daemon"` (`:31, 545-548`).

### 2.3 Write paths

| Surface            | Entry point                                    | Provenance carried |
| ------------------ | ---------------------------------------------- | ------------------ |
| HTTP/CLI write     | `Store.Write` → `pathFor` → `AtomicWriteFile` (`internal/memory/store.go:149-172`) | None — `agent_name` is whatever the file's frontmatter contains |
| Dream consolidation | `consolidation.spawnSession` runs a "dream" agent that uses normal write tools (`internal/memory/consolidation/runtime.go:410-464`) | Agent runs as the configured `cfg.Memory.Dream.Agent` (`:259-260`) |
| Automatic per-turn  | **None.** No hook fires `Store.Write` on session events. | — |
| Pre-compaction flush | **None.** | — |
| Network/channel join | **None.** | — |

The mutation path always logs an operation event with the literal string `"daemon"` (`internal/memory/catalog.go:545-548`) regardless of who actually called `Write`.

### 2.4 Read paths

- **Prompt assembly (always-on, both scopes):** `Assembler.PromptSection` injects the *full* `MEMORY.md` index for global plus workspace ahead of the agent's system prompt (`internal/memory/assembler.go:50-95`). Limited to ~200 lines / 25KB.
- **Per-turn recall (always-on, lexical):** `NewRecallAugmenter` runs an FTS5 search on the user message before every `driver.Prompt` call, prepends the top 3 hits (≤1500 chars) above the user message (`internal/memory/recall.go:22-59`). Uses workspace + global (no per-agent, per-peer, or per-channel filter).
- **CLI/HTTP search:** `Store.Search` → `catalog.search` (BM25) with fallback lexical scorer (`internal/memory/store.go:327-374`, `internal/memory/catalog.go:458-521`).
- **History:** `OperationHistoryQuery` filters `scope`, `workspace`, `operation`, `since` — but **not by `agent_name`** because all rows are "daemon" (`internal/memory/catalog.go:568-637`).

### 2.5 Lifecycle

- Mutations run a **synchronous** index regeneration + catalog `replaceScope` per mutation (`internal/memory/store.go:546-600`). No batching, no debounce. This is correct for one-writer-at-a-time but pessimistic if multiple agents mutate concurrently.
- Staleness: human-readable warning when `mtime` > 1 day (`internal/memory/staleness.go:28-40`). Used by `recall.go:78`. No structured confidence score, no auto-revalidation.
- Lock: file-based PID lock with mtime-as-timestamp for dream consolidation only (`internal/memory/lock.go:21-145`). Not used for write-conflict between user agents.

### 2.6 Consolidation (dream)

- Three gates evaluated cheapest-first (`internal/memory/dream.go:171-213`): time (default 24h), session count (default 3), lock acquisition.
- Spawns a session of the configured agent with `consolidationPromptTemplate` (`internal/memory/prompt.go:5-51`) — orient, gather, consolidate, prune.
- Operates **only on filesystem memory + completed session metadata files** (`internal/memory/dream.go:314-361`). It cannot ingest channel transcripts, peer interactions, recipe outcomes, or hook history because none of those produce memory writes.

---

## 3. What an Autonomous Agent Needs to Remember

For real autonomy across sessions, channels, and peers, agents need a richer taxonomy than `user/feedback/project/reference`. The taxonomy below is what I'd model — **none** of these have first-class storage in AGH today (most live in `network_timeline_log`, `session_events`, or nowhere).

| Class                       | Example                                                                | Today in AGH                                                                 | Gap                                              |
| --------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------ |
| **Peer facts**              | `bob@e5f6 speaks Portuguese, charges 0.10 USDC, last delivered ok`     | Visible only as one-shot `peer_card` in active channel (`contract.go:467`)   | No durable per-peer ledger, no "what I know about Bob" file |
| **Channel norms**           | `#fe-team replies in <1h, decisions need 2 +1s, reviewer is alice`     | Only `network_channels.purpose` text (`global_db_network_channels.go:30-47`) | No per-channel blackboard or ledger              |
| **Task / decision history** | `Decision 2026-04-12: chose pgvector over qdrant because <reason>`     | Buried in session transcript JSON                                             | No promotion path from session event → memory    |
| **Capability outcomes**     | `parse-nfe-ptbr@1.2 succeeded 9/10, fails on NFSe-SP`                  | None — `success_rate` is a network-spec aspiration only                      | No skill/recipe execution ledger                 |
| **Procedural memory**       | `When user asks "deploy", run X then Y then Z (worked 5x in a row)`    | None                                                                          | No trajectory → procedure promotion              |
| **Negative knowledge**      | `Tried approach X 3 times, always fails with <error>`                  | None                                                                          | Same anti-rediscovery gap                        |
| **Episodic summaries**      | `Session 2026-04-20 with alice: refactored auth, opened PR #42`        | Raw events in `sessiondb` only                                                | No autogenerated summary, no recall              |
| **Cross-agent agreements**  | `bob and I split the work: bob owns frontend, I own backend`           | Not modeled                                                                   | No shared agreement file in channel scope        |
| **Identity / authority**    | `alice is workspace owner, hooks must run as her`                      | Workspace exists; no notion of "owner" or "authority"                         | Memory has no per-identity scope                 |
| **Recall provenance**       | `This claim came from memory file X dated Y, observed by agent Z`      | Recall block lacks file-id / agent-id (`recall.go:72-76`)                    | Hard to audit or correct                         |

---

## 4. Gaps (concrete, with file refs)

### 4.1 No peer-fact memory

Peer cards (`internal/api/contract/contract.go:467-475`) live only in active session state. When a channel disconnects, what *this* agent learned about Bob — that he speaks Portuguese, charges 0.10 USDC, delivered correctly twice and once badly — is lost. There is no `memory/peers/<peer_fp>.md` and no peer-scoped catalog index. The hermes `MemoryProvider.on_delegation(task, result, child_session_id, ...)` hook (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/agent/memory_provider.py:175-186`) shows the contract that is missing here.

### 4.2 No channel blackboard / shared knowledge scope

`network_channels` is a metadata table (channel name, workspace, purpose, created_by — `global_db_network_channels.go:30-47`). There is no `memory/channels/<channel>/*.md`, no `ScopeChannel`, no way for agents in `#fe-team` to leave a durable note "we use feature flag X for Y" that the next agent joining the channel sees auto-injected. The Agora design treats spaces as pub/sub topics (`docs/ideas/network/agora-spec-v0.2.md:286-295`) and explicitly *defers* persistent state ("NOT a persistence layer — NATS core, no JetStream/KV" `:51`). AGH has the storage; it just hasn't wired channels into the memory store.

### 4.3 No skill / capability outcome ledger

The recipe outcome log (`trace` envelope in Agora `docs/ideas/network/agora-spec-v0.2.md:516-549`) and openclaw's `memory-core` short-term promotion (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/extensions/memory-core/src/short-term-promotion.ts`) both model the same idea: every capability execution leaves a structured trace, and durable patterns are promoted into procedural memory. AGH records hook runs (`hook_runs` queries via `contract.go:294-300`) and session events but never converts "this skill worked / failed" into memory. There is no `success_rate` per capability, no "last failure mode" file, no auto-disable of broken capabilities.

### 4.4 No automatic per-turn or pre-compaction extraction

`internal/memory/recall.go` augments the prompt before each turn (read path), but no hook augments the *write* side. Compare:

- claude-code `extractMemories` runs as a stop-hook subagent after every turn (`docs/ideas/from-claude-code/analysis_memory_autonomous.md:42-60`).
- hermes `MemoryProvider.sync_turn(user, assistant)` and `on_pre_compress(messages)` (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/agent/memory_provider.py:114-119, 163-173`).

AGH's `internal/session/manager_hooks.go` exists and dispatches lifecycle hooks but there is no `OnTurnEnd` → memory extractor wired in. The dream cycle's 24h cadence is the *only* automatic write path.

### 4.5 `agent_name` provenance plumbed but not used

Schema reserves `agent_name` everywhere (`internal/memory/types.go:57`, `internal/memory/catalog.go:80, 552`), but:

- `Store.Write` doesn't take an `agent_name` parameter — it parses whatever the frontmatter contains (`internal/memory/store.go:149-172`).
- `logCatalogEvent` always writes `"daemon"` (`internal/memory/catalog.go:31, 545-548`).
- `OperationHistoryQuery` has no `AgentName` filter (`internal/memory/types.go:95-101`).
- `Search` has no `agent_name` filter (`internal/memory/types.go:61-65`).

So even though the column exists, an agent can never ask "what did *I* learn?" or "what did the `code-reviewer` agent decide last time?". This is dead provenance.

### 4.6 No session-end episodic summary

Sessions are append-only event streams (`internal/store/sessiondb/session_db.go`), and `meta.json` is rewritten on state changes (described in `docs/ideas/orchestration/multi-agent-patterns-analysis.md:73-77`). When a session ends there is no automatic "what did this session accomplish, what was decided, what should the next session know" Markdown drop. The dream pass *might* synthesize one a day later, but only if the time + session-count gates pass and only via a heavyweight LLM call.

### 4.7 No session-start / channel-join auto-injection beyond scope-wide MEMORY.md

`Assembler.PromptSection` always injects the full global + workspace `MEMORY.md` index (`internal/memory/assembler.go:50-95`). This is good for "everything we know about this workspace" but doesn't filter by:

- Current `channel` (memory relevant to `#fe-team` shouldn't bleed into `#deploy`)
- Current `peers` in the room (per-peer fact recall when entering a channel)
- Current agent's identity (a `code-reviewer` agent doesn't need `user_role.md`)
- Current task type (debugging vs writing)

Compare claude-code's `findRelevantMemories.ts` which uses an LLM side-query to select 5 relevant memories per query (`docs/ideas/from-claude-code/analysis_memory_autonomous.md:24-30`).

### 4.8 No write-conflict policy across agents

The store regenerates `MEMORY.md` synchronously on every mutation and replaces the whole catalog scope (`internal/memory/store.go:546-600`). With one writer at a time this is fine; with two agents both writing `project_X.md` simultaneously they will silently overwrite each other (atomic file write at `:165` is per-file, not per-scope). There is no per-file optimistic concurrency, no last-writer-wins-with-merge, no append-only logs.

### 4.9 No identity / multi-tenant isolation

Memory is bound to filesystem path (global home dir, workspace dir). Two operators sharing a workspace share *one* memory store. There is no `user_id` or `peer_fp` namespace. The `memory-gaps/README.md:179-188` doc flags this explicitly: "global e workspace ajudam, mas ainda não cobrem bem cenários de multi-user, team/shared, tenant isolation".

### 4.10 No retrieval feedback loop

We log "memory.search" events (`internal/memory/types.go:39-47`, `internal/memory/store.go:349-360`) but nothing records *which* result was actually consumed by the model, *which* memory got cited, or which memory the model corrected. Compare hermes `on_memory_write` mirror hook (`memory_provider.py:30`) and openclaw `memory-events` (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/extensions/memory-core/src/memory-events.test.ts`). Without a retrieval-feedback loop, the dream pass cannot prune "memories nobody ever uses" or boost "memories that always help".

### 4.11 Recall block has no actionable provenance

The recall augmenter renders entries as `- <name> [<scope>]\n  Snippet: ...\n  Freshness: ...` (`internal/memory/recall.go:72-76`). Filename, agent, peer, channel are absent. The model has no way to point back at a memory file to update it. claude-code's MEMORY.md uses `- [Title](file.md) -- hook` pointers so the model can `Read(file.md)` to drill in (analysis at `docs/ideas/from-claude-code/analysis_memory_autonomous.md:11`). AGH's full MEMORY.md is good; the recall block isn't.

### 4.12 Network primitives never feed memory

`network_timeline_log` (`global_db_network_messages.go`) records every `say`/`direct`/`receipt`/`echo` envelope sent and received. None of these are observed by the memory pipeline. An `echo(positive, recipe_ref)` envelope from peer A about peer B is the canonical "Bob did good work" signal — and AGH stores it but never promotes it into a peer-fact memory file.

---

## 5. Reference Comparisons

### 5.1 claude-code (`docs/ideas/from-claude-code/analysis_memory_autonomous.md`)

- File-based memory + `MEMORY.md` index: AGH has this (parity).
- **Team memory** subdirectory (`memory/team/`) with per-type scoping rules (`user`=private, `project`=biases-team, `reference`=usually-team) — `analysis_memory_autonomous.md:17`. **Missing in AGH** (no team scope).
- **Daily logs** (`memory/logs/YYYY/MM/...md`) for KAIROS proactive mode, distilled nightly into topic files — `analysis_memory_autonomous.md:18`. **Missing in AGH** (no log scope).
- **`extractMemories` background subagent** that runs after every turn as a perfect-fork (`analysis_memory_autonomous.md:42-60`). **Missing in AGH** (no per-turn extractor).
- **`findRelevantMemories` LLM side-query** to select 5 relevant memories before each turn (`analysis_memory_autonomous.md:24-30`). **Missing in AGH** — recall is pure FTS5 lexical.
- **Mutual exclusion** between main agent and extractor + coalesced trailing run pattern — applies to AGH if/when extractor lands.
- Reference files: `/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/memdir/findRelevantMemories.ts`, `.../memdir/teamMemPaths.ts`, `.../services/teamMemorySync/`, `.../services/SessionMemory/sessionMemory.ts`, `.../utils/memory/types.ts`.

### 5.2 hermes (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/agent/memory_provider.py`)

- **Pluggable `MemoryProvider` interface** with builtin + at most one external (`memory_provider.py:42-86`, `memory_manager.py:84-142`). AGH has only one backend (`Backend` interface in `internal/memory/types.go:124-134`) but no provider plugin slot.
- **Lifecycle hook contract:** `initialize(session_id, hermes_home, platform, agent_context, agent_identity, agent_workspace, parent_session_id, user_id)` — `memory_provider.py:60-81`. **AGH has none of this context** in memory calls. Note explicit **`user_id`**, **`agent_identity`**, **`agent_workspace`**, and **`parent_session_id`** — exactly the dimensions AGH is missing.
- **`prefetch(query, session_id)` + `queue_prefetch(query, session_id)`** with background prefetching — `memory_provider.py:92-112`. AGH does it inline per turn.
- **`sync_turn(user, assistant, session_id)` after every turn** — `memory_provider.py:114-119`. **Missing in AGH.**
- **`on_pre_compress(messages) -> str`** to extract before context discard — `memory_provider.py:163-173`. **Missing in AGH** (no compaction integration).
- **`on_delegation(task, result, child_session_id)`** — parent observes subagent results — `memory_provider.py:175-186`. **Critical for the autonomy slice** because it's the canonical "I delegated to peer X, they returned Y, remember the outcome" hook. AGH has zero equivalent.
- **Context fencing** with `<memory-context>` tags + system note (`memory_manager.py:46-82`). AGH appends the recall block in plain text above the user message (`internal/memory/recall.go:53-58`). The model has no signal that the block is recalled context vs new user input.

### 5.3 openclaw memory-core (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/extensions/memory-core/src/`)

- **Distinct dreaming phases** as separate files: `dreaming-phases.ts`, `dreaming-narrative.ts`, `dreaming-repair.ts`, `dreaming-markdown.ts`, `dreaming-shared.ts`. AGH has one monolithic prompt (`internal/memory/prompt.go`). The phase split lets each phase get critique/repair iterations.
- **`short-term-promotion.ts`** — explicit promotion path from short-term observations to durable memory. AGH has no short-term tier; it's all durable from first write.
- **`session-search-visibility.ts`** — controls which sessions a search can see. AGH searches all of global + active workspace; can't restrict to "only sessions involving peer X".
- **Embedding backend** (`memory-lancedb`) as a separate plugin (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/extensions/memory-lancedb/`). AGH is FTS5-only — flagged as gap in `docs/ideas/memory-gaps/README.md:120-135`.
- **`active-memory` extension** — a separate concept of currently-active context vs durable corpus (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/extensions/active-memory/`). AGH has no equivalent (no session-scoped working memory).

### 5.4 Agora protocol (`docs/ideas/network/agora-spec-v0.2.md`)

- Agora explicitly defers shared persistent state (`:51`). The protocol carries **single events** (`echo`, `trace`, `recipe`); **the runtime is responsible for distilling them into knowledge**. AGH is the runtime — but currently doesn't.
- `echo` envelope (`:469-494`) — "reputational attestation about another agent". This is **the** wire-level peer-fact signal. AGH stores it in `network_timeline_log` (`global_db_network_messages.go`) and never promotes it to memory.
- `trace` envelope (`:516-549`) — verifiable execution log per recipe. **The** capability-outcome signal. Same fate: stored, not promoted.
- `recipe` artifact (`:553-732`) — content-addressed procedural knowledge. AGH has skills (`internal/skills/`) but no recipe-execution ledger linking recipe-id to outcome.

### 5.5 Multi-agent orchestration analysis (`docs/ideas/orchestration/multi-agent-patterns-analysis.md`)

- Recommends a `workflow_id` spanning sessions for cross-session correlation (`:308-320`). **Not implemented** — search confirms zero `workflow_id` references in `internal/store` or `internal/observe`. Without a workflow correlation handle, episodic memory can't be grouped by "this multi-agent task" — only by individual session.
- Recommends `EventTypeHandoff` event type for immutable inter-agent state handoff (`:90-101`). **Not implemented.** Without it, agents handing work to each other have no record of what state was transferred.

---

## 6. Concrete Proposals

These are stack-ranked for autonomy impact. Each proposal cites the file(s) it would touch and the new types/scopes it introduces.

### Proposal A — Add `ScopePeer` and `ScopeChannel` (MUST)

**Problem:** `Scope` enum (`internal/memory/types.go:25-33`) is binary global/workspace. Channels and peers have no durable knowledge layer.

**Change:**

```go
const (
    ScopeGlobal    Scope = "global"
    ScopeWorkspace Scope = "workspace"
    ScopePeer      Scope = "peer"      // NEW — keyed by (observer_peer_fp, subject_peer_fp)
    ScopeChannel   Scope = "channel"   // NEW — keyed by channel name (within workspace or global)
    ScopeSession   Scope = "session"   // NEW — episodic, keyed by session_id, lifecycle = consolidated then archived
)
```

- `ScopePeer` directories: `<global_dir>/peers/<peer_fp>/<topic>.md` — written by an observer agent about a subject peer. Read on `peer joined channel`.
- `ScopeChannel` directories: `<workspace>/.compozy/memory/channels/<channel>/*.md` plus a top-level `<global_dir>/channels/<channel>/*.md` for cross-workspace channels. Read on `channel join` and as part of every prompt while in the channel.
- `ScopeSession` directories: `<workspace>/.compozy/memory/sessions/<session_id>/episode.md` — written at session end by an episodic-summary hook (Proposal D), pruned by dream after promotion to durable memory.

**Code touchpoints:**

- `internal/memory/types.go:25-33` — extend enum + `Validate`.
- `internal/memory/types.go:236-248` — extend `DefaultScopeForType` (e.g. add `MemoryTypePeerFact` → `ScopePeer`, `MemoryTypeChannelNorm` → `ScopeChannel`, `MemoryTypeEpisode` → `ScopeSession`).
- `internal/memory/store.go:506-545` — add new dir resolvers; `Store.ForChannel(channel)`, `Store.ForPeer(observerFP, subjectFP)`, `Store.ForSession(sessionID)` methods.
- `internal/memory/catalog.go:34-51` — add `channel`, `peer_observer`, `peer_subject`, `session_id` columns to `memory_catalog_entries` (migration v3).

### Proposal B — Wire peer/channel/session scope into prompt assembly and recall (MUST)

**Problem:** `Assembler` and `RecallAugmenter` only know about `(global, workspace)`. They don't see `sess.Channel` or peer membership.

**Change:**

- `internal/memory/assembler.go:50-95` — extend `PromptSection` to also load `MEMORY.md` for the current channel (from `session.Info.Channel`, `internal/api/contract/contract.go:38`) and for each peer currently in the channel.
- `internal/memory/recall.go:22-59` — extend `NewRecallAugmenter` to filter `Search` by `(scope IN (global, workspace, channel-current, peers-in-channel, session-current))`.

**Token budget:** Today the recall block is bounded at 1500 chars (`recall.go:14`). With more scopes, introduce a per-scope sub-budget (e.g. 500 chars per scope) and a relevance-weighted fill, rather than a flat top-3.

### Proposal C — Make `agent_name` provenance real (MUST)

**Problem:** `agent_name` is a dead column.

**Change:**

- `internal/memory/types.go:124-134` — change `Backend.Write(scope, filename, content)` → `Write(ctx, WriteOpts{Scope, Filename, Content, AgentName, PeerFP, SessionID, Channel})`. (Keep a deprecated wrapper to stay compatible with the HTTP contract, but plumb `agent_name` from session context everywhere.)
- `internal/memory/store.go:149-172` — set frontmatter `AgentName` from opts if blank; reject if frontmatter and opts disagree.
- `internal/memory/catalog.go:545-548` — record actual `agent_name` from caller, not literal `"daemon"`.
- `internal/memory/types.go:95-101` — add `AgentName string` to `OperationHistoryQuery`.
- `internal/memory/types.go:61-65` — add `AgentName string` to `SearchOptions`.

This unlocks: "show me what *I* learned this week", "what does the `code-reviewer` agent know about this file?", "trust this fact — `architect-agent` wrote it".

### Proposal D — Auto-write hooks at session lifecycle boundaries (MUST)

**Problem:** No automatic write path other than dream-every-24h.

**Change:** Wire memory writes into existing session lifecycle hooks (`internal/session/manager_hooks.go` already exists per the diff in conversation context).

| Hook                | Write target                                                | Implementation                                                                               |
| ------------------- | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `OnTurnEnd`         | `ScopeSession`/`<sid>/scratch.md` (running session memory)  | Threshold-based (token+toolcall delta) à la claude-code's `SessionMemory/sessionMemory.ts`   |
| `OnPreCompact`      | `ScopeSession`/`<sid>/precompact.md`                        | Forced flush before context compaction — closes the loop hermes documents at `:163-173`      |
| `OnSessionEnd`      | `ScopeSession`/`<sid>/episode.md` (final episodic summary)  | Single LLM extractor (or fallback non-LLM extractor that just dumps tool/file diffs)         |
| `OnChannelJoin`     | (read-only) — auto-inject channel + peer scopes             | No write; triggers `Assembler` per Proposal B                                                |
| `OnReceiptReceived` | `ScopePeer`/peers/<their_fp>/outcomes.md`                   | Append: `bob delivered translation — receipt ok @ 2026-04-25`                                |
| `OnEchoReceived`    | `ScopePeer`/peers/<subject_fp>/reputation.md`               | Append: `+1 from alice for parse-nfe-ptbr@1.2 — "worked on 10 PDFs"`                         |

The `OnReceipt`/`OnEcho` hooks are the **bridge from the network protocol to durable memory**. They take the wire-level signals AGH already records in `network_timeline_log` and promote them into queryable knowledge.

### Proposal E — Skill / capability outcome ledger (SHOULD)

**Problem:** No success-rate or last-failure record per skill.

**Change:**

- New table `skill_outcomes` (in `globaldb`): `(skill_name, version, agent_name, session_id, outcome, duration_ms, error_excerpt, completed_at)`.
- Hook into the existing skill execution path (`internal/skills/`).
- Surface as a memory virtual scope: `ScopeSkill` reads aggregate `success_rate` and last 3 failures into the prompt when an agent considers using a skill.
- Auto-disable a skill (or warn) after N consecutive failures — implements the circuit-breaker pattern from `multi-agent-patterns-analysis.md:172-217` at the skill granularity.

### Proposal F — Session-start auto-injection key (SHOULD)

**Problem:** Memory is injected by scope only, not by relevance.

**Change:** New session-start hook calls `Store.RelevantOnInit(ctx, RelevantOnInitOpts{AgentName, Workspace, Channel, Peers, Goal})` which:

1. Always loads scope-wide MEMORY.md indexes (current behavior).
2. Runs an FTS5 search keyed on agent + channel + peers + goal text → top 5 by score.
3. Injects them inline as a "Likely-relevant memory for this session" block, with file pointers so the model can `Read(...)` to drill in.

Mirrors claude-code's `findRelevantMemories.ts` (`docs/ideas/from-claude-code/analysis_memory_autonomous.md:24-30`).

### Proposal G — Memory mutation surface for autonomy events (SHOULD)

**Problem:** Memory writes go through Markdown content. For machine-emitted facts (peer `echo`, capability outcome) the LLM round-trip is wasteful and noisy.

**Change:** Add a structured mutation API alongside Markdown writes:

```go
type FactWrite struct {
    Scope     Scope
    Subject   string  // peer_fp, channel, skill_name, session_id
    Predicate string  // "delivered", "rejected", "speaks", "reviews"
    Object    string
    Source    string  // wire envelope id, session event id
    Confidence float64
    AgentName string
}

func (s *Store) WriteFact(ctx context.Context, fact FactWrite) error
```

Stored as append-only rows in a `facts` table; rendered into auto-generated Markdown topic files at index time. The dream pass reads facts + Markdown together and produces consolidated narrative memory. This is how openclaw's `memory-core` separates short-term observation from long-term promotion (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/extensions/memory-core/src/short-term-promotion.ts`).

### Proposal H — Recall provenance + context fencing (SHOULD)

**Problem:** Recall block is plain text with no file pointer; model can't drill in or correct.

**Change:** `internal/memory/recall.go:61-101` — render entries with file pointer + agent + scope + score + freshness, all inside a fenced `<memory-context>` block (mirroring hermes `memory_manager.py:46-82`):

```
<memory-context source="agh-memory">
[System note: recalled memory context, NOT new user input. Treat as informational background.]

- [Bob speaks PT](peers/bob@e5f6/profile.md) — agent=alice, scope=peer, score=0.92, fresh
  Snippet: "bob handles greek→pt translation, charges 0.10 USDC"
- ...
</memory-context>
```

### Proposal I — `workflow_id` cross-session correlation (SHOULD)

**Problem:** Multi-agent workflows leave no trace that links sessions together.

**Change:** Per `multi-agent-patterns-analysis.md:308-320`. Adds `workflow_id` to `SessionEvent` content JSON (no schema change required) and enables a "workflow episodic memory" that records the chain of sessions and their outcomes as one unit.

### Proposal J — Channel-/peer-bound write conflict resolution (NICE)

**Problem:** Concurrent writes to the same channel/peer file silently overwrite.

**Change:** Adopt append-only logs for `ScopePeer` and `ScopeChannel` (one file = one append-log of facts), with a derived consolidated `summary.md` regenerated by the dream pass. This sidesteps the "two agents both rewriting `bob.md`" race. AGH's existing `memory_operation_log` (`internal/memory/catalog.go:78-86`) is a model for this pattern.

---

## 7. Open Questions

1. **Privacy across workspaces.** If agent A is in workspace X and learns "Bob speaks PT" via channel `#world`, should that fact also be visible to A in workspace Y? Per-workspace isolation says no; per-peer-globally says yes. The honest answer is: peer memory should live at `ScopeGlobal/peers/<fp>/...` (one fact per peer, regardless of workspace), but channel memory should be workspace-scoped unless explicitly created at global level. Needs a config flag.

2. **Write conflicts when many agents share a channel.** Two agents in `#fe-team` both decide to record "we use feature flag X for Y". File-level last-writer-wins is unsafe. Options: (a) append-only logs (Proposal J); (b) per-file optimistic concurrency with `If-Match` ETag; (c) channel-scope writes only by a designated "channel scribe" agent. (a) is simplest and matches the network's append-only log model.

3. **GC for transient scopes.** `ScopeSession/<sid>` should be archived/deleted once the dream pass has promoted its durable bits. `ScopeChannel/<channel>` should be deleted when a channel is deleted (`global_db_network_channels.go:141-154` already supports channel delete). `ScopePeer/<fp>` should never be deleted unless the peer revokes. Need explicit GC rules per scope.

4. **Embeddings vs structured.** `docs/ideas/memory-gaps/README.md:120-135` flags FTS5-only as a known gap. Two questions remain: (a) should embeddings be a *separate* index that augments FTS5 (hybrid BM25+vector — recommended), or replace it? (b) Where does the embedding model live — embedded in the daemon (slow, bloats binary) or via an MCP-style provider plugin (matches hermes `MemoryProvider` model)?

5. **Cost of LLM-side recall selection.** Proposal F's session-start relevant-memory selection wants a fast LLM call. AGH has multiple ACP-driver agents available — pick the cheapest model? Configure per-agent? Skip when cheap models unavailable? Agora explicitly worried about "LLM listening O(N×M)" cost (`docs/ideas/network/agora-council_round1.md:40-44`); same risk applies here at every prompt.

6. **What gets injected when the user is in a *busy* channel.** If `#fe-team` has 30 peers and the agent joins, naive auto-injection of every peer's memory file is a context-bomb. Need a "top-N peers I've actually interacted with, ranked by recency" filter. This is where the retrieval-feedback loop (Gap 4.10) becomes essential.

7. **Memory + skills + recipes — three knowledge stores or one?** AGH has `internal/skills/` (procedural how-to), `internal/memory/` (declarative facts), and the network protocol envisions `recipe` artifacts (`agora-spec-v0.2.md:553-732`). These three overlap. A single unified knowledge store is conceptually cleaner; three separate stores are operationally simpler. Decision impacts whether `ScopeSkill` (Proposal E) lives in `internal/memory/` or in a new `internal/knowledge/` package.

8. **Backwards compatibility (per CLAUDE.md "Greenfield Alpha — Zero Legacy Tolerance").** Proposals A–D break the on-disk schema and the `Backend` interface. Per project rules, that's acceptable — but the migration plumbing is real (e.g. existing operator-written Markdown files with `agent_name` blank need a default). Need to decide: rewrite everything in one PR (per CLAUDE rules), or stage scope additions before deprecating the old `Write` signature.

---

## Key Files Referenced

- `internal/memory/types.go` — types, scopes, taxonomy
- `internal/memory/store.go` — write/read/scan/index
- `internal/memory/catalog.go` — FTS5 catalog + operation log
- `internal/memory/recall.go` — per-turn recall augmenter
- `internal/memory/assembler.go` — prompt-section injector
- `internal/memory/dream.go` — consolidation gates + spawn
- `internal/memory/consolidation/runtime.go` — dream session lifecycle
- `internal/memory/prompt.go` — 4-phase consolidation prompt
- `internal/memory/staleness.go` — freshness calculation
- `internal/memory/lock.go` — dream lock
- `internal/store/globaldb/global_db_network_channels.go` — channel metadata table
- `internal/store/globaldb/global_db_network_messages.go` — `network_timeline_log` (echo, trace, etc. — never observed by memory)
- `internal/api/contract/contract.go` — memory + network + session payloads
- `docs/ideas/memory-gaps/README.md` — prior gap analysis (PT)
- `docs/ideas/from-claude-code/analysis_memory_autonomous.md` — claude-code memdir, extractMemories, autoDream
- `docs/ideas/orchestration/multi-agent-patterns-analysis.md` — workflow_id, immutable handoff
- `docs/ideas/network/agora-spec-v0.2.md` — echo/trace/recipe envelopes
- `.resources/hermes/agent/memory_provider.py` — pluggable provider interface, lifecycle hooks, on_delegation
- `.resources/hermes/agent/memory_manager.py` — context fencing, prefetch/sync orchestration
- `.resources/openclaw/extensions/memory-core/src/short-term-promotion.ts` — short→long promotion path
- `.resources/openclaw/extensions/memory-lancedb/` — embedding-backed memory plugin reference
- `.resources/claude-code/memdir/findRelevantMemories.ts` — LLM-side relevance selection
- `.resources/claude-code/services/teamMemorySync/` — team memory sync + secret guard
- `.resources/claude-code/services/SessionMemory/sessionMemory.ts` — running session memory
