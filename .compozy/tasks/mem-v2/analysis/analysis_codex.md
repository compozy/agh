# Codex CLI — Memory & Context Architecture (analysis for AGH mem-v2)

> **Source corpus:** `/Users/pedronauck/Dev/compozy/agh/.resources/codex/` (commit shipped with this snapshot of the OpenAI Codex CLI). Rust core lives in `codex-rs/`. All file paths in this report are absolute.

## TL;DR

Codex draws a hard line between **transcript** (durable JSONL) and **memory** (curated, hand-edited markdown the model reads through prompts and tools). Every session is recorded one-line-per-event into `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<thread_id>.jsonl` (`codex-rs/rollout/src/recorder.rs:1363-1393`). A SQLite DB at `~/.codex/state` mirrors thread metadata + Phase 1 memory rows + job leases (`codex-rs/state/migrations/`). The in-RAM `ContextManager` (`codex-rs/core/src/context_manager/history.rs`) holds the live transcript, normalizes it for the model, and tracks `reference_context_item` so settings diffs can be re-injected without bloating prompt prefix.

When tokens cross `model_auto_compact_token_limit`, Codex runs a **3-mode compaction** (pre-turn, mid-turn, model-downshift) that summarizes via the Responses API or a dedicated remote `responses_compact` endpoint, replaces history with `summary + last user messages + canonical context`, and writes a `Compacted` rollout item that doubles as the resume checkpoint (`codex-rs/core/src/compact.rs:447-511`, `compact_remote.rs`).

The "memory" workspace at `~/.codex/memories/` is generated **asynchronously at startup** by a 2-phase pipeline (`memories/README.md`, `memories/write/src/start.rs`): Phase 1 spawns a `gpt-5.4-mini`/Low-effort job per recent rollout that produces `raw_memory + rollout_summary + rollout_slug` (`memories/write/src/phase1.rs`); Phase 2 runs a single sandboxed sub-agent (`gpt-5.4`/Medium) that consumes the merged raw memories + git workspace diff and edits the on-disk handbook (`MEMORY.md`, `memory_summary.md`, `skills/`, `rollout_summaries/`). Read-time injection is a `memory_summary.md` slot in the developer prompt plus a hard contract that any reply that uses memory append a parseable `<oai-mem-citation>` block (`memories/read/templates/memories/read_path.md`).

AGENTS.md is loaded at session start by walking from cwd up to the nearest `.git` (or configured marker) and concatenating in root→leaf order with byte budget `project_doc_max_bytes` (`codex-rs/core/src/agents_md.rs:212-303`). Sub-agents do **not** trigger memory generation (`memories/README.md:31-37`, `memories/write/src/start.rs:33`).

The result is a fully separated three-tier model — *transcript on disk* → *handbook curated by a memory agent* → *prompt-time injection of a tiny summary plus a tool to read deeper* — that is observably opinionated, has explicit two-touch refinements, and is the most mature of any agent harness I have inspected so far.

---

## 1. Repo & crate layout (memory-relevant)

```
codex-rs/
├── core/                        # session, turn loop, ContextManager, AGENTS.md, compaction
│   └── src/
│       ├── agents_md.rs         # AGENTS.md discovery + project-doc concatenation
│       ├── compact.rs           # local "memento" Responses-API compaction
│       ├── compact_remote.rs    # remote /v1/responses_compact
│       ├── compact_remote_v2.rs # newer remote variant
│       ├── context_manager/     # in-RAM history (transcript) + diff/normalize
│       │   ├── history.rs       # ContextManager — the heart of context state
│       │   ├── normalize.rs     # call/output pairing + image stripping
│       │   └── updates.rs       # settings diff → developer/contextual user items
│       ├── session/
│       │   ├── session.rs       # Session struct + session boot
│       │   ├── mod.rs           # build_initial_context / settings injection
│       │   ├── turn.rs          # turn loop (auto-compact triggers live here)
│       │   ├── turn_context.rs  # per-turn snapshot (model, perms, cwd)
│       │   └── rollout_reconstruction.rs  # reverse replay → restore live state
│       ├── thread_rollout_truncation.rs   # rollback-aware turn boundaries
│       └── thread_manager.rs    # external manager wrapping sessions
├── rollout/                     # JSONL recorder + listing + state-DB sync
│   └── src/
│       ├── recorder.rs          # writer task; ~/.codex/sessions/YYYY/MM/DD/...jsonl
│       ├── policy.rs            # what RolloutItem variants are persisted
│       └── state_db.rs          # mirror to SQLite
├── thread-store/                # storage-neutral interface for threads
│   └── src/{store.rs,local/,remote/,types.rs,live_thread.rs}
├── memories/                    # the entire memory product
│   ├── README.md                # the canonical overview (quoted heavily here)
│   ├── read/                    # read-path: developer prompt + citations + usage metrics
│   │   └── src/{lib.rs,prompts.rs,citations.rs,usage.rs}
│   │   └── templates/memories/read_path.md  ← the prompt the model sees
│   ├── write/                   # write-path: the 2-phase pipeline
│   │   └── src/{start.rs,phase1.rs,phase2.rs,storage.rs,workspace.rs,prompts.rs,...}
│   │   └── templates/memories/{stage_one_system.md,stage_one_input.md,consolidation.md}
│   └── mcp/                     # tiny MCP server exposing list/read over the memories tree
├── state/                       # SQLite (state.db) — threads + stage1_outputs + jobs
│   └── migrations/0001..0029_*.sql
├── rollout-trace/               # opt-in raw evidence bundles (NOT memory; debug only)
├── external-agent-sessions/     # importer for non-Codex agent rollouts (Claude Code etc.)
├── skills/                      # built-in skill installer (~/.codex/skills/.system)
└── secrets/                     # redact_secrets() used in memory write path
```

Two crates are key for the discussion:

* **`codex-memories-write`** — writes; owns the pipeline (`codex-rs/memories/write/src/lib.rs:1-17`).
* **`codex-memories-read`** — reads; owns the prompt block + the `<oai-mem-citation>` parser (`codex-rs/memories/read/src/lib.rs:1-20`).

The split is enforced by Cargo: `read` does not depend on `write`, so the read-path is invariant under pipeline changes (`codex-rs/memories/read/src/lib.rs:1-7` — explicit comment).

---

## 2. The four "tiers" of context (this is the model)

Codex stitches four distinct context surfaces together. Internalize this taxonomy before reading anything else.

| Tier | Lifetime | Storage | Injection mechanism | Authoring |
|---|---|---|---|---|
| **Transcript** | per-thread | `~/.codex/sessions/.../rollout-*.jsonl` + in-RAM `ContextManager` | replayed verbatim into Responses API `input` | machine (turn loop) |
| **AGENTS.md / project-doc** | per-cwd | repo files | concatenated into `user_instructions` at session start | human (project-checked-in) |
| **Memory handbook** | global to user | `~/.codex/memories/{MEMORY.md, memory_summary.md, skills/*, rollout_summaries/*, raw_memories.md}` | `memory_summary.md` injected into developer prompt + filesystem tools to read the rest | mixed (Phase-1 model + Phase-2 sub-agent + ad-hoc human notes) |
| **Skills** | per-user, optional per-thread | `~/.codex/skills/{.system,<name>/SKILL.md}` and `memories/skills/<name>/` | listed in developer prompt; agent reads on demand | shipped + extension + memory-promoted |

Crucially, **transcript ≠ memory**. The transcript is immutable evidence. Memory is hand-curated knowledge derived from many transcripts. Codex's PRD-era constraint shows up directly in `stage_one_system.md`:

> Raw rollouts are immutable evidence. NEVER edit raw rollouts.
> (`codex-rs/memories/write/templates/memories/stage_one_system.md:21`)

This is the strongest single piece of architectural guidance in the codebase.

---

## 3. Transcript: the rollout JSONL file

### 3.1 Path layout

```
~/.codex/
├── sessions/<YYYY>/<MM>/<DD>/rollout-<YYYY-MM-DDThh-mm-ss>-<thread_uuid>.jsonl
├── archived_sessions/...     # archived equivalents
├── state                     # SQLite mirror (threads, stage1_outputs, jobs, ...)
├── memories/                 # the memory handbook (its own git repo)
│   ├── memory_summary.md
│   ├── MEMORY.md
│   ├── raw_memories.md
│   ├── rollout_summaries/<rollout_slug>.md
│   ├── skills/<skill_name>/SKILL.md
│   ├── extensions/ad_hoc/notes/<ts>-<slug>.md
│   └── .git/                 # baseline managed by codex-git-utils
├── skills/.system/           # baked-in system skills; refreshed via marker
└── ...
```

The rollout filename is computed at session-create time by `precompute_log_file_info` at `codex-rs/rollout/src/recorder.rs:1363-1393`:

```rust
dir.push(SESSIONS_SUBDIR);
dir.push(timestamp.year().to_string());
dir.push(format!("{:02}", u8::from(timestamp.month())));
dir.push(format!("{:02}", timestamp.day()));
let filename = format!("rollout-{date_str}-{conversation_id}.jsonl");
```

`SESSIONS_SUBDIR = "sessions"` and `ARCHIVED_SESSIONS_SUBDIR = "archived_sessions"` are constants in `codex-rs/rollout/src/lib.rs:21-22`. Year/month/day partitioning keeps directory listings tractable even for users with thousands of rollouts.

### 3.2 The on-the-wire schema

`RolloutItem` is a 5-variant tagged enum (`codex-rs/protocol/src/protocol.rs:2776-2789`):

```rust
pub enum RolloutItem {
    SessionMeta(SessionMetaLine),  // first line — identity, base_instructions, dynamic_tools, git
    ResponseItem(ResponseItem),    // every API-visible item: messages, reasoning, tool calls/outputs
    Compacted(CompactedItem),      // a compaction checkpoint (carries replacement_history!)
    TurnContext(TurnContextItem),  // turn-scoped settings snapshot (model, perms, cwd)
    EventMsg(EventMsg),            // protocol events (TurnStarted, ThreadRolledBack, etc.)
}

pub struct CompactedItem {
    pub message: String,                                 // human-readable summary
    pub replacement_history: Option<Vec<ResponseItem>>,  // exact rebuilt history at this point
}
```

`Compacted.replacement_history` is the **single most important field for AGH to copy**. It means a compaction checkpoint contains the *complete rebuilt history* at that moment. Resume can short-circuit and skip everything older.

### 3.3 The persistence policy is explicit (and selective)

Not every event is persisted. The policy is hand-coded in `codex-rs/rollout/src/policy.rs`:

```rust
pub fn should_persist_response_item(item: &ResponseItem) -> bool {
    match item {
        ResponseItem::Message { .. }
        | ResponseItem::Reasoning { .. }
        | ResponseItem::FunctionCall { .. }
        | ResponseItem::FunctionCallOutput { .. }
        | ResponseItem::CustomToolCall { .. }
        | ResponseItem::CustomToolCallOutput { .. }
        | ResponseItem::WebSearchCall { .. }
        | ResponseItem::ImageGenerationCall { .. }
        | ResponseItem::Compaction { .. }
        | ResponseItem::ContextCompaction { .. } => true,
        ResponseItem::Other => false,
    }
}
```

There are **two persistence levels** (`codex-rs/rollout/src/policy.rs:5-10, 70-89`):

* `Limited` — only durable replay-relevant events (default).
* `Extended` — adds rich UI/diagnostic events (e.g. `ExecCommandEnd` aggregated_output, McpToolCallEnd, etc.).

`Extended` mode further sanitizes large blobs before writing — `ExecCommandEnd.aggregated_output` is middle-truncated to 10 KiB, `stdout`/`stderr`/`formatted_output` are cleared (`codex-rs/rollout/src/recorder.rs:190-215`):

```rust
const PERSISTED_EXEC_AGGREGATED_OUTPUT_MAX_BYTES: usize = 10_000;
fn sanitize_rollout_item_for_persistence(item: RolloutItem, mode: EventPersistenceMode) -> RolloutItem {
    if mode != EventPersistenceMode::Extended { return item; }
    match item {
        RolloutItem::EventMsg(EventMsg::ExecCommandEnd(mut event)) => {
            event.aggregated_output = truncate_middle_chars(&event.aggregated_output, ...);
            event.stdout.clear(); event.stderr.clear(); event.formatted_output.clear();
            ...
```

There is also a **"persist for memories"** filter that is stricter still — it drops `Reasoning`, `ImageGenerationCall`, `Compaction`, and developer-role messages so that the Phase 1 extractor only sees the substantive turn (`codex-rs/rollout/src/policy.rs:48-65`).

### 3.4 Background writer

`RolloutRecorder` runs a single background `tokio` task fed by an `mpsc` of commands (`AddItems`, `Persist`, `Flush`, `Shutdown`) (`codex-rs/rollout/src/recorder.rs:78-113`). Writes are append-only and synced after each batch. Failures are sticky and exposed via `terminal_failure()` so subsequent recorder API calls can surface them (`codex-rs/rollout/src/recorder.rs:117-157`).

After every write, `sync_thread_state_after_write` upserts/touches the SQLite mirror so `state.db` always tracks `updated_at` even if `created_at`-affecting fields didn't change (`codex-rs/rollout/src/recorder.rs:1708-1750`).

### 3.5 Reverse replay = resume

Resuming a thread is a pure function of the rollout file. `Session::reconstruct_history_from_rollout` walks the items **newest→oldest** and stops as soon as it has the three things it needs (`codex-rs/core/src/session/rollout_reconstruction.rs:86-222`):

1. The newest surviving `Compacted.replacement_history` (= the base of rebuilt history; older items are obsolete).
2. The newest `previous_turn_settings` (model + realtime flag).
3. The newest `reference_context_item` (the diff baseline).

Then it replays the *suffix* forward to materialize history exactly. `ThreadRolledBack` markers are interpreted in reverse: a rollback of N turns becomes "skip the next N user-turn segments while scanning backward". The whole replay is purely deterministic from the JSONL — no separate state file, no DB read.

This is why `replacement_history` is so load-bearing: it lets resume run in O(checkpoint-tail) instead of O(rollout-length). For long sessions this is the difference between instant and slow resume.

---

## 4. AGENTS.md handling

### 4.1 Discovery

`AgentsMdManager::agents_md_paths` (`codex-rs/core/src/agents_md.rs:213-303`):

1. Canonicalize `cwd`.
2. Compute `project_root_markers` from non-project config layers (default `[".git"]`).
3. Walk ancestors of `cwd`; the first ancestor that contains any marker is the **project root**.
4. Build `search_dirs = [root, root/<sub>, ..., cwd]` (root→leaf).
5. For each dir, try `AGENTS.override.md` first, then `AGENTS.md`, then any configured fallback filenames.
6. Return all matching paths, **in root→leaf order**.

### 4.2 Concatenation + budget

`read_agents_md` concatenates the contents in that root→leaf order, with each file truncated to whatever budget is left under `project_doc_max_bytes` (`codex-rs/core/src/agents_md.rs:149-206`). Empty files are skipped. The remaining budget is decremented file-by-file and a `tracing::warn!` is emitted if a file would have overflowed:

```rust
let mut remaining: u64 = max_total as u64;
let mut parts: Vec<String> = Vec::new();
for p in paths {
    if remaining == 0 { break; }
    ...
    if size > remaining { data.truncate(remaining as usize); }
    ...
}
```

### 4.3 Hierarchical "agents message"

When the `child_agents_md` feature is on, Codex appends a fixed snippet (`codex-rs/core/hierarchical_agents_message.md:1-7`) after AGENTS.md content, regardless of whether any AGENTS.md exists:

> Each AGENTS.md governs the entire directory that contains it and every child directory beneath that point. Whenever you change a file, you have to comply with every AGENTS.md whose scope covers that file. … When two AGENTS.md files disagree, the one located deeper in the directory structure overrides the higher-level file, while instructions given directly in the prompt by the system, developer, or user outrank any AGENTS.md content.

This is the explicit precedence rule baked into the prompt, not enforced in code. There is **no merging** — Codex relies on the model to apply precedence at read time.

### 4.4 Global override and separator

There is also a global instruction file at `<codex_home>/AGENTS.md` (or `AGENTS.override.md`) loaded by `load_global_instructions` (`codex-rs/core/src/agents_md.rs:61-78`). It is concatenated with project AGENTS.md using a constant separator:

```rust
const AGENTS_MD_SEPARATOR: &str = "\n\n--- project-doc ---\n\n";
```

The combination order is: `config.user_instructions` (rare) → `AGENTS_MD_SEPARATOR` → concatenated AGENTS.md docs (root→leaf) → optional hierarchical message.

### 4.5 No hot-reload

There is no file-watcher for AGENTS.md inside the live session — once the session has been created, the captured `user_instructions` are immutable until next session start. `instruction_sources()` exists (`codex-rs/core/src/agents_md.rs:130-141`) but is read-only: it lists the files, it doesn't reload them.

The only "hot" doc is **skills**, which has a `SkillsWatcher` in core (file `codex-rs/core/src/skills_watcher.rs`), and emits `SkillsUpdateAvailable` events that the user/agent can consume to refresh.

---

## 5. ContextManager — the in-RAM transcript

`ContextManager` (`codex-rs/core/src/context_manager/history.rs:34-454`) is a small, very deliberate type. Fields (lines 34-51):

```rust
pub(crate) struct ContextManager {
    items: Vec<ResponseItem>,        // oldest→newest
    history_version: u64,            // bumped on rewrite (compaction or rollback)
    token_info: Option<TokenUsageInfo>,
    reference_context_item: Option<TurnContextItem>,  // baseline for next-turn diffs
}
```

### 5.1 Append discipline

`record_items` (line 99-113) only accepts items that pass `is_api_message` (line 478-495) — system messages are excluded; everything else (user/assistant/reasoning/tool-call/tool-output/compaction) is included. On insert, `process_item` (line 372-406) middle-truncates large `FunctionCallOutput`/`CustomToolCallOutput` bodies according to the per-turn `TruncationPolicy`, with a 1.2× "serialization budget" pad to soak up JSON envelope cost. **Codex truncates tool outputs at append time**, not at prompt time — this is a guarantee, not a heuristic.

### 5.2 Normalization invariants

`normalize_history` (line 361-370) enforces:

1. Every function/custom tool call has a corresponding output.
2. Every output has a corresponding call.
3. When the model doesn't support images, image content is stripped from messages and tool outputs.

These invariants are restored by `normalize::ensure_call_outputs_present` and `remove_orphan_outputs`, called every time history is sent to the model via `for_prompt(input_modalities)` (line 119-122). Any mid-history removal also calls `normalize::remove_corresponding_for` to keep call/output pairs intact (`codex-rs/core/src/context_manager/history.rs:160-180`).

### 5.3 Token estimation

`estimate_token_count` is byte-based with a hand-tuned image multiplier (line 135-158, 511-628). Reasoning items get a `length_estimate(encoded_len) = encoded_len * 3 / 4 - 650` heuristic to avoid over-counting `encrypted_content` blobs (line 497-503). Inline base64 image data URLs are replaced with a fixed `RESIZED_IMAGE_BYTES_ESTIMATE = 7373` (≈1844 tokens) for resized detail, or the actual patch count `(width/32)·(height/32)` capped at `ORIGINAL_IMAGE_MAX_PATCHES = 10000` for `detail: "original"`.

### 5.4 Turn-boundary helpers

`is_user_turn_boundary`, `is_codex_generated_item`, `is_model_generated_item`, `user_message_positions` (lines 679-728) drive **rollback** semantics: dropping the last N user turns (`drop_last_n_user_turns`, line 237-260) cuts the items vector at the chosen user-message index *and* trims trailing pre-turn context-update items (line 425-453), additionally clearing `reference_context_item` when a "mixed" pre-turn developer bundle gets cut so the next turn does a full re-injection rather than diff against stale state.

### 5.5 Settings diff (`updates.rs`)

`build_settings_update_items` (`codex-rs/core/src/context_manager/updates.rs:204-238`) is the diff function: given previous `TurnContextItem` + new `TurnContext`, it emits *only* the developer + contextual-user fragments needed to reflect changes in `cwd`, permissions, collaboration mode, realtime, personality, model. Comment at line 212-215:

> // TODO(ccunningham): build_settings_update_items still does not cover every model-visible item emitted by build_initial_context. Persist the remaining inputs or add explicit replay events so fork/resume can diff everything deterministically.

Even Codex admits this is the load-bearing weak spot.

---

## 6. Compaction — three flavors, one shape

### 6.1 Triggers

Auto-compact runs in three phases (search `codex-rs/core/src/session/turn.rs`):

1. **`PreTurn`** (`run_pre_sampling_compact`, line 718-753) — runs *before* sampling for the new user turn, two checks:
   * `maybe_run_previous_model_inline_compact` — if the model just changed and the new model has a smaller `auto_compact_token_limit`, compact under the *old* turn context first (`reason = ModelDownshift`).
   * Second: if `total_usage_tokens >= auto_compact_limit`, compact again (`reason = ContextLimit`, `phase = PreTurn`).
2. **`MidTurn`** (line 487-506) — after each sampling response, if `token_limit_reached && needs_follow_up`, compact and continue the loop. Uses `InitialContextInjection::BeforeLastUserMessage` so the compaction summary stays *last* but canonical context lives just above the latest real user message.
3. **`StandaloneTurn`** = manual `/compact` slash command (`run_compact_task` at `codex-rs/core/src/compact.rs:92-114`) — emits its own `TurnStarted` event and uses `InitialContextInjection::DoNotInject` (the next regular turn will fully re-inject context).

The threshold is `model_info.auto_compact_token_limit()` which falls back to `i64::MAX` if unset (`codex-rs/core/src/session/turn.rs:150`). Configurable via `model_auto_compact_token_limit` in `config.toml` (`codex-rs/core/src/config/mod.rs:413`).

### 6.2 Algorithm — "Memento" strategy

`run_compact_task_inner_impl` (`codex-rs/core/src/compact.rs:151-276`):

```text
1. Insert ContextCompactionItem (turn item) into history.
2. Append the synthesized SUMMARIZATION_PROMPT as a user message.
3. Stream a Responses API turn. On context-window-exceeded, drop oldest history item and retry — preserves cache prefix on retry.
4. Take the last assistant message produced; that becomes the post-compaction "summary_text".
5. Collect *all* prior user messages (filtered to drop existing summaries).
6. Build replacement history = [last K user messages truncated to 20 000 tokens total] + [a single user message containing summary_text prefixed with SUMMARY_PREFIX].
7. Optionally splice initial context (env, permissions, etc.) just before the last real user message.
8. Replace the in-RAM history; write a single Compacted rollout item with the full replacement_history; reset the WebSocket session; recompute token usage.
9. Emit a Warning telling the user "long threads are noisier; start a new thread when possible".
```

Constants:

* `COMPACT_USER_MESSAGE_MAX_TOKENS = 20_000` (`codex-rs/core/src/compact.rs:44`).
* `SUMMARIZATION_PROMPT` and `SUMMARY_PREFIX` are bundled markdown templates (`codex-rs/core/templates/compact/{prompt.md,summary_prefix.md}`).

The whole strategy is named `CompactionStrategy::Memento` in analytics — the design intent is *"forget specifics, remember the user's plan"*.

### 6.3 Remote variants

`compact_remote.rs` and `compact_remote_v2.rs` issue a single call to a provider-specific `/v1/responses_compact` endpoint that returns the compacted history server-side (eliminating the local model round-trip). The provider gates this via `ModelProviderInfo::supports_remote_compaction()` (`codex-rs/core/src/compact.rs:61-63`).

Pre-compaction history is trimmed by `trim_function_call_history_to_fit_context_window` so the **compact** endpoint itself doesn't OOM (`codex-rs/core/src/compact_remote.rs:131-143`). This is the only place where Codex deletes function-call history in advance for context-window reasons.

### 6.4 Compaction checkpoints in resume

Once written, `Compacted` is the resume base. `reconstruct_history_from_rollout` (`codex-rs/core/src/session/rollout_reconstruction.rs:110-129, 236-282`) prefers the newest `Compacted.replacement_history` and replays only items newer than that checkpoint. Legacy compactions without a `replacement_history` field force a "rebuild from scratch" path that drops `reference_context_item` so the next turn re-injects everything (line 256-272).

### 6.5 Compaction in compaction (Russian-doll)

The `Warning` emitted post-compact tells the user that *"Long threads and multiple compactions can cause the model to be less accurate"*. This is canonical Codex hygiene: compaction is treated as accuracy-eroding. The remedy is "start a new thread", not "compact harder". Internalizing this for AGH: **bottomless compaction is an anti-pattern; we should expose `/new` and `/resume` as visibly as `/compact`**.

---

## 7. Memory pipeline — Phase 1 + Phase 2

This is the most distinctive part of Codex's design. Re-read `codex-rs/memories/README.md` end-to-end if you internalize nothing else.

### 7.1 Trigger conditions

`start_memories_startup_task` (`codex-rs/memories/write/src/start.rs:22-75`) runs the whole pipeline asynchronously **at root-session start**, gated by:

```rust
if config.ephemeral
    || !config.features.enabled(Feature::MemoryTool)
    || source.is_non_root_agent() {
    return;
}
```

Plus inside the spawned task: `state_db.is_some()` and `guard::rate_limits_ok(...)` (the latter checks the user's Codex rate-limit windows have ≥`min_rate_limit_remaining_percent` available).

This is a **hard skip** for ephemeral sessions, sub-agents, and missing state DB. Memory is *only* generated for first-class root sessions where the user is using their personal account.

### 7.2 Defaults (`codex-rs/config/src/types.rs:45-50`)

```rust
DEFAULT_MEMORIES_MAX_ROLLOUTS_PER_STARTUP = 2;
DEFAULT_MEMORIES_MAX_ROLLOUT_AGE_DAYS = 10;
DEFAULT_MEMORIES_MIN_ROLLOUT_IDLE_HOURS = 6;
DEFAULT_MEMORIES_MIN_RATE_LIMIT_REMAINING_PERCENT = 25;
DEFAULT_MEMORIES_MAX_RAW_MEMORIES_FOR_CONSOLIDATION = 256;
DEFAULT_MEMORIES_MAX_UNUSED_DAYS = 30;
```

So per startup: max 2 rollouts processed, only if 6+ hours idle, only if not stale (>10 days), only if user has enough quota. Phase 2 is bounded to 256 raw memories considered for consolidation; never-used memories are pruned after 30 days.

### 7.3 Phase 1 — per-rollout extraction

`memories/write/src/phase1.rs:65-108` — strict 4-step flow:

1. **Claim**: `state_db.claim_stage1_jobs_for_startup` (with lease 3600 s, scan 5000 threads, allowed sources = `INTERACTIVE_SESSION_SOURCES = [Cli, VSCode, "atlas", "chatgpt"]` per `codex-rs/rollout/src/lib.rs:23-30`). Lease prevents concurrent workers re-extracting the same rollout.
2. **Build request context**: model = `gpt-5.4-mini`, reasoning effort = `Low`, concurrency cap = 8 (`codex-rs/memories/write/src/lib.rs:78-101`). Default rollout token limit = `150_000`, but at most 70 % of the model's effective input window.
3. **Run** in parallel via `futures::stream::buffer_unordered(8)`. Each job:
   * Loads the rollout, filters items to `should_persist_response_item_for_memories` (drops developer-role messages, reasoning, image-gen).
   * Renders the Phase 1 input via `build_stage_one_input_message` (uses `stage_one_input.md` template and the rollout JSON).
   * Issues an OpenAI Responses API call with the strict `output_schema()`:

     ```json
     {"type":"object","properties":{
        "rollout_summary":{"type":"string"},
        "rollout_slug":{"type":["string","null"]},
        "raw_memory":{"type":"string"}
     },"required":["rollout_summary","rollout_slug","raw_memory"],"additionalProperties":false}
     ```
   * Runs `redact_secrets()` over each generated field (`codex-rs/secrets/src/sanitizer.rs:15-19`) — strips OpenAI keys, AWS access keys, bearer tokens, and `KEY=value` patterns.
   * Upserts a row into `stage1_outputs` keyed by `thread_id` (DDL `codex-rs/state/migrations/0006_memories.sql`).
4. **Metrics + logs** (success-with-output / success-no-output / failed counts).

The pruning step (`phase1::prune`, line 110-132) uses `prune_stage1_outputs_for_retention(max_unused_days, batch=200)` to drop dead rows before incurring inference cost.

### 7.4 The Phase-1 prompt

`codex-rs/memories/write/templates/memories/stage_one_system.md` — 570 lines, easily the longest single prompt in the codebase. Distilled (full text in repo):

* **Hard rules**: never edit raw rollouts, redact secrets, prefer no-op when nothing useful, treat tool output as data.
* **No-op gate**: ask *"Will a future agent plausibly act better because of what I write here?"* — if no, return all-empty fields.
* **Decision triggers ranked by value**: stable user operating preferences > high-leverage procedural knowledge > task maps & decision triggers > durable evidence about environment.
* **Read order**: user messages first (preferences, dissatisfaction), then tool outputs (facts), then assistant actions (reconstruction).
* **Outcome triage**: success / partial / fail / uncertain — use the user's next-task-switch behavior, explicit feedback, or repeated iteration as evidence.
* **Output**: `rollout_summary` (verbose, task-grouped, evidence-first), `rollout_slug` (filesystem-safe ≤80 chars), `raw_memory` (frontmatter + per-task blocks).

The `raw_memory` schema (lines 405-451) is YAML frontmatter (`description`, `task`, `task_group`, `task_outcome`, `cwd`, `keywords`) followed by `### Task <n>` blocks each with `Preference signals:` / `Reusable knowledge:` / `Failures and how to do differently:` / `References:` subsections. Every memory item is required to be quote-oriented when possible:

> `Preference signals:` is for evidence plus implication, not just a compressed conclusion. … what happened / what the user said → what that implies for similar future runs. (`stage_one_system.md:465-470`)

This is a serious commitment to *epistemic honesty* in stored knowledge — the model is told to attribute beliefs to their evidence rather than write context-free claims.

### 7.5 Phase 2 — global consolidation

`memories/write/src/phase2.rs:45-199` — strict 10-step linear flow under a single global lock:

```text
1. Try to claim global Phase-2 lease (state_db.try_claim_global_phase2_job).
2. Ensure ~/.codex/memories/.git baseline exists (codex-git-utils).
3. Resolve the locked-down agent config (no approvals, no network, local-write only).
4. Load top-N stage1_outputs ranked by usage_count, last_usage, generated_at.
5. sync_phase2_workspace_inputs:
   - Mechanical merge of selected raw_memories into `raw_memories.md` (sorted by thread_id ASC).
   - Sync `rollout_summaries/<slug>.md` files exactly to the selected set (delete stale).
   - Prune memory extension resources older than 7 days.
6. Compute git diff against last successful baseline.
7. If empty diff → mark job success, exit (no agent run).
8. Else: write `phase2_workspace_diff.md` with the diff (≤4 MiB).
9. Spawn the consolidation sub-agent: model `gpt-5.4`, reasoning Medium, no approvals, no net,
   `local-write` sandbox, collab disabled (no recursive delegation), heartbeat lease every 90 s.
10. On success, reset the git baseline and remove the diff file before doing so (so deleted
    content does not stay in the prompt artifact).
```

The agent's job is to update the **on-disk handbook**:
* `MEMORY.md` — searchable handbook entries (the model greps this).
* `memory_summary.md` — capped at ~5000 tokens; injected into every future session prompt.
* `skills/<skill_name>/SKILL.md` — promoted reusable procedures.
* `rollout_summaries/<slug>.md` — kept verbatim, one per selected rollout.
* `raw_memories.md` — overwritten by the sync step (input only; agent typically doesn't edit).

The Phase 2 prompt is `codex-rs/memories/write/templates/memories/consolidation.md` — distinguishes INIT mode from INCREMENTAL UPDATE, mandates respect for the workspace diff, and tells the agent to prefer user-preference signal over routine procedural recap.

### 7.6 The two-phase split — *why*

`codex-rs/memories/README.md:154-158`:

> - Phase 1 scales across many rollouts and produces normalized per-rollout memory records.
> - Phase 2 serializes global consolidation so the shared memory artifacts are updated safely and consistently.

Translated: **Phase 1 is fan-out (parallel, tiny model, structured JSON), Phase 2 is fan-in (single agent, smart model, free-form file edits)**. Concurrency is impossible to share because the consolidation needs to serialize markdown edits — so it's protected by a global lock with cooldown + retry-backoff.

### 7.7 Watermarks and selection

Phase-2 uses git workspace-dirtiness, not DB watermarks, as the dirty signal. The DB still records a watermark for bookkeeping but it doesn't gate execution (`memories/README.md:140-150`). The selection function ranks by `usage_count` first then `last_usage` / `generated_at`. `usage_count`/`last_usage` columns were added in migration 0016 (`state/migrations/0016_memory_usage.sql`) — Phase-2 favors memories *the model has actually cited recently*.

### 7.8 Per-thread memory toggle

The `threads.memory_mode` column on `StoredThread` (`codex-rs/thread-store/src/types.rs:38-39`, populated by `ThreadMemoryMode::Enabled|Disabled`) controls per-thread memory extraction. Sessions opened with `generate_memories=false` mark the thread as `Disabled` so Phase 1 skips it (`codex-rs/core/src/session/session.rs:402-407`).

---

## 8. Read path — how memory shows up to the model

### 8.1 Developer-prompt injection

In `Session::build_initial_context` (`codex-rs/core/src/session/mod.rs:2591-2597`):

```rust
if turn_context.features.enabled(Feature::MemoryTool)
    && turn_context.config.memories.use_memories
    && let Some(memory_prompt) =
        build_memory_tool_developer_instructions(&turn_context.config.codex_home).await
{
    developer_sections.push(memory_prompt);
}
```

`build_memory_tool_developer_instructions` (`codex-rs/memories/read/src/prompts.rs:28-52`) reads `memory_summary.md`, truncates to `MEMORY_TOOL_DEVELOPER_INSTRUCTIONS_SUMMARY_TOKEN_LIMIT = 5_000` tokens (`codex-rs/memories/read/src/lib.rs:16`), and renders it into the `read_path.md` template with `{ base_path, memory_summary }`.

### 8.2 The read prompt — explicit decision boundary

`codex-rs/memories/read/templates/memories/read_path.md` is the most opinionated prompt in Codex. Key directives:

* **Decision boundary** (lines 7-17):
  > Skip memory ONLY when the request is clearly self-contained and does not need workspace history, conventions, or prior decisions. Hard skip examples: current time/date, simple translation, simple sentence rewrite, one-line shell command, trivial formatting. Use memory by default when … the query mentions workspace/repo/module/path/files in MEMORY_SUMMARY below; the user asks for prior context / consistency; the task is ambiguous; the ask is non-trivial.
* **Layout and quick pass** (lines 19-46): Skim summary → search MEMORY.md by keyword → open at most 1-2 rollout-summary or skill files. Quick-pass budget ≤ 4-6 search steps.
* **Verification rules** (lines 51-79): Decide based on drift risk vs. verification cost. If memory-derived without verification, *say so*. Offer to refresh.
* **Mandatory citation block** (lines 80-120):
  ```
  <oai-mem-citation>
  <citation_entries>
  MEMORY.md:234-236|note=[responsesapi citation extraction code pointer]
  rollout_summaries/2026-02-17T21-23-02-LN3m-weekly_memory_report_pivot_from_git_history.md:10-12|note=[weekly report format]
  </citation_entries>
  <rollout_ids>
  019c6e27-e55b-73d1-87d8-4e01f1f75043
  </rollout_ids>
  </oai-mem-citation>
  ```
  Required at the end of any reply that used memory. Parsed by `parse_memory_citation` (`codex-rs/memories/read/src/citations.rs:6-43`).
* **Mandatory mutation channel** (lines 122-128):
  > You can update the memories **only** when explicitly asked by the user. … Write your update in `{{ base_path }}/extensions/ad_hoc/notes/<timestamp>-<short slug>.md`. Each update must be one small file containing what you want to add/delete/update from the memories. Do not try to edit the memory files yourself.

The model never edits MEMORY.md directly. It writes a *delta note* and the next Phase-2 run integrates it. This is the core write/edit governance: humans/agents can leave breadcrumbs, but only the consolidation agent mutates the canonical handbook.

### 8.3 Memory access tools

The agent reads memory via existing filesystem tools (read, search/grep, ls). Two layers:

1. The agent has its normal sandboxed read access — the prompt tells it the layout and search strategy.
2. There is also a tiny **MCP server** at `codex-rs/memories/mcp/src/` exposing `list` and `read` over the memories tree (`schema.rs` for the JSON schemas; cursor-based pagination, line-offset reads). This is for *external* MCP-speaking clients, not the in-process agent.

### 8.4 Usage telemetry → Phase-2 selection feedback loop

`codex-rs/memories/read/src/usage.rs:7-57` defines `MemoriesUsageKind` (MemoryMd, MemorySummary, RawMemories, RolloutSummaries, Skills) and `memories_usage_kinds_from_command` parses safe-shell read/search commands to detect memory access. Detected reads emit the `MEMORIES_USAGE_METRIC` counter and increment `usage_count`/`last_usage` in `stage1_outputs` (migration 0016). Phase 2 then re-ranks selection by these counters — **the more a memory is read, the more likely it is to survive consolidation**. This is a beautiful feedback loop and the right behavior: the model votes with its searches.

### 8.5 The `<oai-mem-citation>` parser

`codex-rs/memories/read/src/citations.rs:53-81` is dead-simple regex-free string parsing:

```rust
let (location, note) = line.rsplit_once("|note=[")?;
let note = note.strip_suffix(']')?.trim().to_string();
let (path, line_range) = location.rsplit_once(':')?;
let (line_start, line_end) = line_range.split_once('-')?;
```

Then `thread_ids_from_memory_citation` extracts cited `rollout_ids` and the post-turn machinery touches `last_usage` for each. Entries also support `<thread_ids>` as an alias of `<rollout_ids>` (line 78-81) — backwards-compat residue.

---

## 9. Slash commands

`codex-rs/tui/src/slash_command.rs:12-138` lists all commands. Memory-relevant:

| Slash | Behavior |
|---|---|
| `/compact` | manual compaction (StandaloneTurn). Emits its own `TurnStarted`. |
| `/memories` | configure memory generation/use settings (TUI) |
| `/skills` | configure skills |
| `/init` | bootstrap an `AGENTS.md` file in cwd |
| `/resume` | reopen a saved chat from disk |
| `/fork` | fork the current chat (creates a new thread that inherits resumed history) |
| `/new` / `/clear` | start a fresh thread (different ergonomics, same effect — clear unloads everything; new keeps the daemon state) |
| `/rollout` | print the current rollout file path (debug) |
| `/debug-m-drop`, `/debug-m-update` | debugging memory state — explicitly marked "DO NOT USE" in description |
| `/agent`, `/subagents` | switch active agent thread (multi-agent v2) |

There is **no** `/memory` slash command in Codex — memory mutations go through `extensions/ad_hoc/notes/`, and configuration lives behind `/memories`. This is consistent with the "agent never edits the handbook" rule.

`available_during_task` (line 176-229) is interesting: `/compact`, `/memories`, `/clear`, `/init`, `/resume`, `/fork` are **not available during a task**. This prevents corrupting state mid-stream.

---

## 10. SQLite mirror (`state.db`)

The state DB is partitioned across 29 incremental migrations (`codex-rs/state/migrations/000{1..29}_*.sql`). The headline tables for memory:

* **`threads`** (migration 0001) — id, rollout_path, source, model_provider, cwd, title, sandbox_policy, approval_mode, tokens_used, archived, git_*. Indexes by created_at/updated_at/source/provider/archived. Later migrations add `cli_version`, `agent_nickname`, `model`, `reasoning_effort`, `agent_path`, `memory_mode`, etc.
* **`stage1_outputs`** (migration 0006) — `(thread_id PK, source_updated_at, raw_memory, rollout_summary, generated_at)` + index on `source_updated_at DESC`. Augmented with `selected_for_phase2` (0017), `usage_count`/`last_usage` (0016), `selected_for_phase2_source_updated_at` (0018), `rollout_slug` (0009), `cwd`/`git_branch` (0008/back-fill).
* **`jobs`** (migration 0001) — generic lease/retry table keyed by `(kind, job_key)`. Used by Phase 1 (per-thread) and Phase 2 (single global key). Fields: `status`, `worker_id`, `ownership_token`, `lease_until`, `retry_at`, `retry_remaining`, `last_error`, `input_watermark`, `last_success_watermark`. Critical: this is the same table for both phases — concurrency is by `kind`, not by separate tables.
* **`thread_spawn_edges`** (migration 0021) — parent/child links for sub-agents.
* **`backfill_state`** (migration 0008) — single-row table tracking the "import legacy rollouts into state.db" progress.

Worth noting: migration 0023 *drops* the legacy `logs` table; `STATE_DB_VERSION = 5` (`codex-rs/state/src/lib.rs:67`). Migration discipline mirrors AGH's own.

---

## 11. External-agent session import (Claude Code / etc.)

`codex-rs/external-agent-sessions/src/` adds the ability to import session histories from another agent harness (e.g. Claude Code) into Codex's rollout format:

* `detect.rs` — scans `<external_agent_home>/projects/*/...jsonl` for session files newer than 30 days, capped at 50.
* `records.rs` — parses external-agent records, summarizes them into a `SessionSummary` with cwd + title.
* `ledger.rs` — records imported source paths so re-runs don't double-import.
* `export.rs` — converts external rollout items into Codex `RolloutItem`s for ingestion.

This is a one-way migration tool — useful as a reference for how Codex models *foreign* histories and how the Phase 1 extractor would handle non-native sources. It **does not** create a sub-agent contract for cross-runtime memory sharing — there's no protocol for that.

---

## 12. Sub-agent context handoff

This is briefer than I expected. Sub-agents are first-class threads (multi-agent v2):

* `Session::new` distinguishes `is_subagent = session_configuration.session_source.is_non_root_agent()` (`codex-rs/core/src/session/session.rs:462`). Non-root agents:
  * Skip `history_metadata` lookup.
  * Skip the entire memory pipeline (`codex-rs/memories/write/src/start.rs:31-35`).
  * Their rollout joins the **same `ThreadTraceContext`** as the parent (`session.rs:540-550`) so trace bundles contain the whole rollout tree.
* The handoff payload is a normal task message — `codex-rs/core/src/session/multi_agents.rs:1-27` is small and just sets up the multi-agent module wiring.
* Spawned threads get a row in `thread_spawn_edges` (migration 0021) with `(parent_thread_id, child_thread_id, status)`.
* Reasoning items, tool calls, and outputs from the child are *not* surfaced into the parent's transcript directly — only the child's final assistant message comes back as a "subagent notification" `EventMsg::CollabAgentInteractionEnd`. The parent's history records the spawn/result edge only.
* The Phase-2 consolidation agent itself is spawned as a sub-agent with explicit collab=disabled (`memories/README.md:113-115`):
  > runs it with no approvals, no network, and local write access only; disables collab for that agent (to prevent recursive delegation)

This is the **right shape**: sub-agents have isolated context, parent gets a notification + the child's last message, and global memory is not generated from sub-agent runs (else it would over-fit to internal scaffolding turns).

---

## 13. Cache control

Codex relies on **OpenAI's prompt caching** via the Responses API rather than rolling its own cache_control breakpoints (the way Anthropic clients do).

`codex-rs/core/src/client.rs:885`:
```rust
let prompt_cache_key = Some(self.client.state.conversation_id.to_string());
```

So every request in the same thread shares the same `prompt_cache_key = thread_id`. The cached-input-tokens count is read back via `usage.cached_input_tokens` (`client.rs:1746`) and surfaced via the `codex.turn.token_usage.cached_input_tokens` / `non_cached_input_tokens` metrics (`tasks/mod.rs:376-377, 647-671`).

This means:
* Cache survives across turns within a thread (good).
* Cache survives compaction *only if* the cached prefix is still verbatim — but Memento compaction rewrites the prefix, so it busts the cache. The post-compaction `client_session.reset_websocket_session()` (`compact.rs:266`) is a deliberate acknowledgement of this cache-bust.
* No client-managed cache breakpoints; no Anthropic-style explicit `cache_control` markers anywhere in the codebase.

---

## 14. Observability & telemetry

* `codex_analytics::CodexCompactionEvent` — every compaction emits trigger/reason/strategy/status/before+after token counts/duration (`codex-rs/core/src/compact.rs:278-341`).
* `MEMORY_PHASE_ONE_E2E_MS`, `MEMORY_PHASE_ONE_JOBS`, `MEMORY_PHASE_ONE_OUTPUT`, `MEMORY_PHASE_ONE_TOKEN_USAGE`, `MEMORY_PHASE_TWO_*` (`codex-rs/memories/write/src/metrics.rs`).
* `MEMORIES_USAGE_METRIC` per-read (`codex-rs/memories/read/src/metrics.rs`).
* Rollout-trace bundles (`codex-rs/rollout-trace/`) — opt-in via `CODEX_ROLLOUT_TRACE_ROOT`. **NOT memory** — they're the local-only forensic record of model requests, payloads, terminal output, and tool dispatch ordering, with a separate `codex debug trace-reduce` reducer that walks `trace.jsonl` + `payloads/*.json` into a `state.json` graph (`codex-rs/rollout-trace/README.md`).

Relevant for AGH: rollout traces are a separate concern from memory and from the rollout JSONL. Codex correctly keeps "session history for resume" (`sessions/`), "knowledge for future agents" (`memories/`), and "raw evidence for debugging" (`rollout-trace bundles`) as three orthogonal stores.

---

## 15. Privacy & redaction

* `codex_secrets::redact_secrets` (`codex-rs/secrets/src/sanitizer.rs:15-19`) replaces OpenAI keys, AWS access keys, bearer tokens, and `KEY=VAL` patterns with `[REDACTED_SECRET]`. Called on Phase 1 output before persistence (`codex-rs/memories/write/src/phase1.rs:20`).
* `EventPersistenceMode::Extended` aggressively trims persisted exec output (`codex-rs/rollout/src/recorder.rs:190-215`).
* The Phase-1 prompt explicitly instructs:
  > Redact secrets: never store tokens/keys/passwords; replace with [REDACTED_SECRET].
* Rollout trace bundles carry an explicit privacy preamble:
  > Rollout tracing is not telemetry. Codex does **not** upload or report these traces; it writes local bundles only when `CODEX_ROLLOUT_TRACE_ROOT` is set. (`rollout-trace/README.md:3-7`)

There is **no** redaction of the rollout JSONL itself — sessions on disk contain everything the model saw. The user is implicitly trusted with their own `~/.codex` dir.

---

## 16. Things Codex got beautifully right

1. **Three orthogonal stores** — transcript, handbook, debug bundle — never confused.
2. **`Compacted.replacement_history`** as resume primitive — O(suffix) replay.
3. **Two-phase memory pipeline** with parallel tiny model + serialized smart agent.
4. **Citation contract** — `<oai-mem-citation>` makes "did the model use memory?" a first-class observable, and the rollout_ids feed the usage counter that ranks Phase-2 selection. Closed-loop.
5. **Mutation by delta note** — agents/users never edit MEMORY.md; they leave timestamped notes in `extensions/ad_hoc/notes/`. Phase 2 reconciles. This is the strongest design move.
6. **memories/.git** — the memories root is itself a git repo; consolidation diffing is structural, not heuristic.
7. **No-op gate** in Phase 1 — `{"rollout_summary":"","rollout_slug":"","raw_memory":""}` is a perfectly valid output, encouraged. Most rollouts produce no durable memory.
8. **Hard skip for sub-agent and ephemeral sessions** — the right default; nobody wants their multi-agent scaffolding turns becoming durable memory.
9. **Per-thread `memory_mode`** — granular opt-out without disabling the feature globally.
10. **Quick-pass budget** ≤ 4–6 search steps before main work — bounds the cost of memory in every reply.

## 17. Things Codex did awkwardly (or that are work-in-progress)

1. **No file-watcher for AGENTS.md** — once captured at session boot, the file is dead until next session. Skills have a watcher; AGENTS.md doesn't. Inconsistent.
2. **Settings update items don't cover the full initial-context bundle** — explicit TODO at `codex-rs/core/src/context_manager/updates.rs:212-215`. Means resume/fork still has subtle drift.
3. **Russian-doll compaction warning is just a Warning event** — there's no hard ceiling. A pathological session can compact endlessly.
4. **`/memory` doesn't exist** — slash commands `/debug-m-drop` and `/debug-m-update` exist but are explicitly marked "DO NOT USE". Power users have no way to inspect or surgically edit memory beyond writing an ad-hoc note.
5. **memory selection ties Phase-2 selection to safe-shell command parsing of file reads** — `MemoriesUsageKind` only counts when the agent uses a *known-safe* shell command on the memory tree. If the agent uses an MCP read or some custom tool, the usage counter doesn't tick. Brittle.
6. **No cross-session lock around the JSONL append** — the writer is a single tokio task per session, which is fine, but two simultaneous resumes of the same thread would race on append. There's no file lock — `LiveThread::resume` (in `codex-rs/thread-store/src/live_thread.rs`) presumably handles serialization at the thread-store layer.
7. **Memory pipeline is gated on Codex rate-limit headroom** (`min_rate_limit_remaining_percent`) — this couples memory generation to the user's quota. Sensible, but means heavy users may never accumulate memory.
8. **No partition / sharding for `stage1_outputs`** — single SQLite table; for users with thousands of threads this could become a Phase-2 selection bottleneck.

---

## 18. Hard-wired numbers worth quoting

| Constant | Value | Source |
|---|---|---|
| Phase 1 model | `gpt-5.4-mini` | `memories/write/src/lib.rs:79` |
| Phase 1 reasoning | `Low` | `lib.rs:80` |
| Phase 1 concurrency | 8 | `lib.rs:82` |
| Phase 1 lease | 3600 s | `lib.rs:83` |
| Phase 1 retry delay | 3600 s | `lib.rs:84` |
| Phase 1 thread scan | 5000 | `lib.rs:85` |
| Phase 1 prune batch | 200 | `lib.rs:86` |
| Phase 1 fallback rollout token cap | 150 000 | `lib.rs:93` |
| Phase 1 context window % | 70 | `lib.rs:100` |
| Phase 2 model | `gpt-5.4` | `lib.rs:104` |
| Phase 2 reasoning | `Medium` | `lib.rs:105` |
| Phase 2 lease | 3600 s | `lib.rs:107` |
| Phase 2 heartbeat | 90 s | `lib.rs:109` |
| Compact: max user-msg tokens kept | 20 000 | `compact.rs:44` |
| Compact: trim oldest on context-overflow during compact | yes | `compact.rs:204-213` |
| Persisted exec aggregated output cap | 10 000 bytes | `recorder.rs:190` |
| Memory summary token cap (read prompt) | 5 000 | `memories/read/src/lib.rs:16` |
| Phase-2 workspace diff cap | 4 MiB | `memories/write/src/lib.rs:115` |
| Phase-2 max raw memories | 256 | `config/types.rs:49` |
| Phase-2 max unused days | 30 | `config/types.rs:50` |
| Memory extension retention | 7 days | `memories/write/src/lib.rs:43` |
| Default rollouts per startup | 2 | `config/types.rs:45` |
| Default min idle hours | 6 | `config/types.rs:47` |
| Min rate-limit headroom % | 25 | `config/types.rs:48` |
| State DB version | 5 | `state/src/lib.rs:67` |
| External-session import max age | 30 days | `external-agent-sessions/src/detect.rs:12` |

---

## 19. Suggested mental model for AGH mem-v2

Codex implies (without ever stating it explicitly) the following layering. Steal it:

```
┌─────────────────────────────────────────────────────────────┐
│                     PROMPT-TIME CONTEXT                     │
│  base_instructions (system) ── model-specific, fixed        │
│  developer messages:                                        │
│    • AGENTS.md (project-doc) [session-immutable]            │
│    • memory_summary.md (≤5K tokens) [pipeline-mutable]      │
│    • permissions, skills, plugins, apps instructions        │
│    • hierarchical-AGENTS.md hint                            │
│  context (user contextual): EnvironmentContext, etc.        │
│  history: ContextManager.items (transcript)                 │
└─────────────────────────────────────────────────────────────┘
            ▲                                  │
            │ replay/diff                      │ append
            │                                  ▼
┌──────────────────────────┐  ┌──────────────────────────────┐
│  ROLLOUT JSONL           │  │  STATE.DB (SQLite mirror)    │
│  ~/.codex/sessions/.../  │◄─┤  threads, stage1_outputs,    │
│  rollout-*.jsonl         │  │  jobs, thread_spawn_edges,   │
│  + Compacted checkpoints │  │  backfill_state              │
└──────────────────────────┘  └──────────────────────────────┘
            │                                  │
            │ Phase-1 fan-out (gpt-5.4-mini)   │
            ▼                                  ▼
┌─────────────────────────────────────────────────────────────┐
│              MEMORY HANDBOOK (~/.codex/memories/.git)       │
│  raw_memories.md   ← machine-merged from stage1_outputs     │
│  rollout_summaries/<slug>.md   ← per-rollout               │
│  MEMORY.md, memory_summary.md, skills/<n>/SKILL.md          │
│      ← edited by the Phase-2 sub-agent (gpt-5.4 medium)     │
│  extensions/<src>/instructions.md, .../resources/*          │
│  extensions/ad_hoc/notes/<ts>-<slug>.md   ← human/agent     │
│      delta notes (only mutation channel for in-flight       │
│      sessions)                                               │
└─────────────────────────────────────────────────────────────┘
```

Five invariants hold across this picture. AGH should adopt them all:

1. **Transcript is immutable evidence.** Memory is curated knowledge. Never blur the line.
2. **Memory mutations go through delta notes.** The handbook is owned by the consolidation agent, not by the live agent.
3. **Compaction is a checkpoint, not just a summary.** `replacement_history` makes resume cheap and idempotent.
4. **Sub-agents are read-only with respect to global memory.** They do not generate memory; they don't inherit memory_summary in their prompt unless explicitly handed it.
5. **Read usage feeds write selection.** Cited memories survive consolidation; uncited memories age out.

---

## 20. Unanswered questions / things AGH should resolve before copying

1. **Promptless re-ranking** — Codex's Phase-2 selection over `usage_count + last_usage + generated_at` is a coarse rank; there is no semantic re-ranking, no embeddings. Worth checking how stable this is in practice.
2. **Per-cwd vs. per-user memory** — Codex's `cwd` field on `stage1_outputs` is purely metadata; the handbook is global. AGH may want stronger cwd-scoping.
3. **Cross-thread credential leakage** — `redact_secrets` is regex-based; sophisticated leaks (e.g. multi-line PEMs) could survive. AGH should consider a richer secret detector.
4. **Memory under organisation/team scope** — Codex is single-user. AGH multi-tenant story has to add an org boundary that doesn't currently exist.
5. **Compaction strategy hard-codes "Memento"** — no alternative strategies; no fallback for very long, low-signal sessions where summaries themselves are too large.
6. **No invalidation of stale memory** — once written, a memory only goes away by aging out (`max_unused_days`) or being explicitly removed. There is no "this memory is contradicted by later evidence" mechanism beyond the consolidation agent's discretion.
7. **AGENTS.md reload** — Codex chose session-immutable. AGH must decide whether daemon-resident sessions need hot-reload.
8. **`/memory` UX** — Codex doesn't expose direct memory inspection. AGH's "agents must be able to manipulate the runtime through CLI/HTTP/UDS" rule (CLAUDE.md) implies we should add an agent-operable `memory` surface.
