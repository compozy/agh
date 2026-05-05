# Session Ledger & Daily-Log Retention — Synthesis Analysis

> Two questions for AGH mem-v2:
> - **Q1**: how do competitors persist session-level memory? Filesystem? Event log? TTL?
> - **Q5**: competitors that ship `daily/YYYY-MM-DD.md` — how do they bound disk?
>
> Source corpus verified directly. Cites use `path:line`.

---

## TL;DR

For **Q1**, AGH should pick **Option C — hybrid event-log + on-disk JSONL ledger**. The DB (`memory_events`, `scope=session`) is the canonical truth (consistent with `analysis_layered-scope.md:110`). On `OnSessionEnd`, materialize a forensic `<workspace>/.agh/sessions/<id>/ledger.jsonl` snapshot (Codex `rollout-*.jsonl` shape, but as a *projection* of events.db, not authoritative). This matches Codex (`recorder.rs:1363-1393`), Claude Code (`<sessionId>.jsonl`), Hermes (`state.db` SQLite + `prune_sessions(90d)` at `hermes_state.py:2074-2127`), and the OpenClaw `sessions.json + per-session JSONL` split. Live-session ops use the DB; forensic browsing/replay/`agh session replay` uses the JSONL. No live-session JSONL writer task — too easy to dead-lock at scale.

For **Q5**, AGH should adopt **size+line cap with seq rotation** (1 MB, 5000 lines → `daily/YYYY-MM-DD.<seq>.md`), **dreaming reads last 7 days** (Hot), **cold-archive at 30 days** to `_system/archive/YYYY-MM/`, and **never hard-delete by default** (paperclip's "no deletion, only supersede" + Claude memdir's append-only-forever stance). Disk caps are configurable. OpenFang's silent-drop-at-1MB (`kernel.rs:451-475`) is the wrong default. OpenClaw's no-cap is the wrong default at the other extreme.

---

## Q1: Session Ledger — competitor evidence

### Codex — `rollout-*.jsonl` (filesystem authoritative + SQLite catalog)

- **Path**: `~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<thread_uuid>.jsonl`. Computed at session-create by `precompute_log_file_info` (`/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/rollout/src/recorder.rs:1363-1393`).
- **Structure**: 5-variant `RolloutItem` enum (`SessionMeta | ResponseItem | Compacted | TurnContext | EventMsg`). `Compacted.replacement_history` carries the rebuilt history at each compaction checkpoint — single most copy-able idea.
- **Persistence policy**: `policy.rs:14-44` whitelists `Message | Reasoning | LocalShellCall | FunctionCall* | ToolSearch* | CustomToolCall* | WebSearchCall | ImageGenerationCall | Compaction | ContextCompaction`. `EventPersistenceMode::{Limited, Extended}` (line 5-10) controls breadth; `Extended` middle-truncates `ExecCommandEnd.aggregated_output` to 10 KB (`recorder.rs:190-215`).
- **Background writer**: single-tokio-task per session fed by `mpsc<{AddItems, Persist, Flush, Shutdown}>` (`recorder.rs:78-113`). Append-only; sticky `terminal_failure()` propagates write errors.
- **DB mirror**: `~/.codex/state` (SQLite). `sync_thread_state_after_write` upserts `threads` + `stage1_outputs` (`recorder.rs:1708-1750`).
- **Retention**: **NONE for rollout files**. Verified — no `remove_file` / `prune` / `retent`-named symbols touch `~/.codex/sessions/`. Stage-1 memory pipeline uses `max_age_days: 30` (`/Users/pedronauck/Dev/compozy/agh/.resources/codex/codex-rs/state/src/runtime/memories.rs:128, 1641, 1741, 1811, 2015, 2049, 2103, 2136`) but that gates *which rollouts the memory extractor visits*, not deletion. Operator-level cleanup. External-agent imports cap at 30 days / 50 files (`external-agent-sessions/src/detect.rs:12`).
- **Dead path detection**: `state_db.rs:266-279` — when `state.db` lists a thread whose rollout file is gone (`tokio::fs::try_exists` false), `delete_thread(item.id)` is called. So files-deleted-out-from-under is handled, but Codex never deletes them itself.

**Verdict**: filesystem JSONL is authoritative; SQLite is the index for fast listing/search. No TTL. User cleans up.

### Claude Code — per-session JSONL under per-project dir

- **Path**: `~/.claude/projects/<sanitized-cwd>/<sessionId>.jsonl` (per `analysis_claude-code.md`).
- **Memdir** (auto-memory): `~/.claude/projects/<sanitized-git-root>/memory/MEMORY.md` + `*.md` topic files. **`MEMORY.md` capped at 200 lines / 25 KB** (`/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/memdir/memdir.ts:35-38`). When `feature('KAIROS')` is on, daily logs land at `<autoMemDir>/logs/YYYY/MM/YYYY-MM-DD.md` (`memdir.ts:335`).
- **Retention**: **NONE** in code. Daily logs are append-only forever; a separate nightly `/dream` skill distills (`memdir.ts:319-326`). Topic files persist indefinitely.

**Verdict**: per-session JSONL on filesystem; user-managed retention; index file (`MEMORY.md`) capped to protect prompt budget.

### Codex memory subsystem (parallel store, NOT session ledger)

`stage1_outputs` rows (one per rollout) are pruned by `prune_stage1_outputs_for_retention(max_unused_days, batch=200)` at startup (`memories/write/src/phase1.rs:110-132`). Default `DEFAULT_MEMORIES_MAX_UNUSED_DAYS = 30` (`config/types.rs:50`). Pruning targets the *derived memory rows*, not the rollouts.

### Hermes — single SQLite DB, parent_session lineage, time-based prune

- **Store**: `~/.hermes/state.db`, schema v11. Tables: `sessions`, `messages`, `state_meta`, FTS5 `messages_fts` (unicode61) + `messages_fts_trigram` (CJK). All chat turns from CLI, gateway adapters, ACP, cron, subagents land here (`hermes_state.py:36-101, 103-156`).
- **Lineage**: `sessions.parent_session_id` chains compaction-split sessions (`hermes_state.py:50, 71`).
- **Cleanup**:
  - `delete_session(session_id)` (`hermes_state.py:2040-2073`) — explicit single-session removal.
  - `prune_sessions(older_than_days=90)` (`hermes_state.py:2074-2127`):
    - Only prunes ended sessions (`ended_at IS NOT NULL`).
    - Orphans children (sets `parent_session_id = NULL`) instead of cascade-deleting.
    - Removes on-disk transcript files (`*.json`, `*.jsonl`, `request_dump_*`) outside the DB transaction.
  - `vacuum()` (`hermes_state.py:2153-2174`) — `PRAGMA wal_checkpoint(TRUNCATE)` then `VACUUM` to reclaim disk.
  - `maybe_auto_prune_and_vacuum(retention_days=90, min_interval_hours=24)` (`hermes_state.py:2176-2247`) — idempotent: stamps `state_meta.last_auto_prune` so re-runs within 24h skip; only `VACUUM`s when prune > 0 rows.

**Verdict**: SQLite-only ledger, time-based pruning **(default 90 days)**, idempotent auto-maintenance via `state_meta`. No filesystem-shadow ledger except legacy transcript files.

### OpenClaw — per-agent JSONL + per-store metadata index, age + count + disk caps

- **Paths** (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/src/config/sessions/paths.ts`):
  ```
  ~/.openclaw/agents/<agentId>/sessions/sessions.json                    ← metadata index
  ~/.openclaw/agents/<agentId>/sessions/<sessionId>.jsonl                ← transcript
  ~/.openclaw/agents/<agentId>/sessions/<sessionId>.reset.<ts>.jsonl     ← rotated
  ~/.openclaw/agents/<agentId>/sessions/<sessionId>-topic-<encTopic>.jsonl
  ```
- **Index lock**: per-store-path FIFO lock queue with `timeoutMs=10s, staleMs=30s` (analysis_openclaw.md §3.1).
- **Retention** (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/src/config/sessions/store-maintenance.ts:10-15`):
  ```ts
  DEFAULT_SESSION_PRUNE_AFTER_MS = 30 * 24 * 60 * 60 * 1000;  // 30 days
  DEFAULT_SESSION_MAX_ENTRIES = 500;                          // FIFO cap
  DEFAULT_SESSION_MAINTENANCE_MODE = "enforce";               // vs "warn"
  DEFAULT_SESSION_DISK_BUDGET_HIGH_WATER_RATIO = 0.8;
  ```
  Disk-budget enforcement at `disk-budget.ts:243-418` honors `maintenance.maxDiskBytes`. Pruned/capped entries archive their transcripts (analysis_openclaw.md §15: "archiveSessionTranscripts").

**Verdict**: filesystem JSONL is authoritative; per-store JSON index; **three caps stack** — age (30d) + entries (500) + disk bytes (configurable). `mode=warn` lets operators preview before enforcing.

### OpenFang — DB authoritative + filesystem mirror

- **Per-channel sessions** in SQLite (`sessions` rmp_serde blob; analysis_openfang.md §1, §2.1).
- **Canonical session** in same DB (one per agent, cross-channel).
- **JSONL mirror** at `<workspace>/sessions/<session-id>.jsonl` (auto-written; analysis_openfang.md TL;DR).
- **Retention**: **NONE** documented. analysis_openfang.md §13 row `TTL / GC | none` confirms.

**Verdict**: DB-first with on-disk projection; no GC.

### Summary table

| Harness | Authoritative store | Lineage | Filesystem mirror | Retention |
|---|---|---|---|---|
| Codex | filesystem JSONL | `Compacted.replacement_history` | n/a (FS *is* the store) | none (memory job uses 30d window) |
| Claude Code | filesystem JSONL | `<sessionId>.jsonl` per project | n/a | none |
| Hermes | SQLite | `parent_session_id` chains | legacy `*.jsonl` removed on prune | 90d default; idempotent + VACUUM |
| OpenClaw | filesystem JSONL | reset-rotation; topic-split | `sessions.json` index | 30d / 500 / disk-budget |
| OpenFang | SQLite | `canonical_sessions` | `<workspace>/sessions/<sid>.jsonl` | none |

---

## Q1 Verdict for AGH — Option C (hybrid)

**Pick Option C: live ops via `memory_events` (DB), forensic snapshot to `<workspace>/.agh/sessions/<id>/ledger.jsonl` on `OnSessionEnd`.**

### Why not Option A (event-log only)

- Already proposed in `analysis_layered-scope.md:111`: "Session-scope writes are events with `scope=session`; they project into ephemeral catalog rows that auto-purge on `OnSessionEnd` unless a controller decision promotes them."
- Failure mode: zero forensic surface. `agh session replay <id>` after the session ends has no ground truth. Codex resume (the strongest single feature in the corpus) depends on the JSONL being on disk and human-greppable. Pure DB throws away the "files-are-evidence" architecture that Codex's `stage_one_system.md:21` calls out as load-bearing: "Raw rollouts are immutable evidence."
- Hermes's pure-DB approach works because it's a Python harness with a curator, not a file-first agent runtime. AGH's premise (`CLAUDE.md`: "agent-first system, agents manipulate via CLI + REST") wants `cat ~/work/.agh/sessions/<id>/ledger.jsonl | jq …` to be a usable surgical tool.

### Why not Option B (live JSONL writer)

- The user's ledger habit ("I keep a ledger in my projects") naturally maps to JSONL. But making the JSONL **authoritative for live writes** doubles the lock surface during high-throughput sessions:
  - Codex needs a single tokio writer task + `mpsc` queue + sticky failure (`recorder.rs:78-157`). Adding the same on top of AGH's `events.db` is two write paths to keep consistent.
  - OpenClaw needed a per-store FIFO lock queue with stale-detection and Windows retry (analysis_openclaw.md §3.1). At AGH's scale (multiple sessions per workspace, daemon long-lived) this is two interlocking lock domains.
- Greenfield rule (`CLAUDE.md`: "Hard cuts, not bridges"): one source of truth.

### Option C (hybrid) — recommended shape

```
LIVE SESSION
├── events.db: memory_events (scope=session, session_id=<uuid>)
│   ← every turn, tool call, recall, hook, prompt rendered
└── ephemeral memory_catalog_entries (auto-purge on OnSessionEnd unless promoted)

OnSessionEnd hook fires
├── Materialize <workspace>/.agh/sessions/<id>/ledger.jsonl
│   (stream-write events.db rows in chronological order, one RolloutItem-shape line each)
├── Materialize <workspace>/.agh/sessions/<id>/manifest.json
│   (id, started_at, ended_at, end_reason, parent_session_id, message/tool/token totals)
└── Purge ephemeral catalog rows; persist promoted ones
```

This is **Codex's pattern, inverted**: AGH's events.db is the live writer, JSONL is the *snapshot*; Codex's JSONL is the live writer, state.db is the *index*. Both keep the JSONL as the human-grep surface; AGH gets the durability/concurrency benefits of the DB while live, plus the forensic affordance Codex's design earned.

### Concrete invariants

1. **events.db is canonical for live sessions.** `agh session show`, recall, hooks, the dreaming worker all read from events.db.
2. **JSONL ledger is canonical for ended sessions.** Once written it is immutable. `agh session replay <id>` reads the ledger.
3. **Compaction marker.** When compaction fires, write a `Compacted` event with the rebuilt history payload (Codex's `replacement_history`). Resume reads the newest `Compacted` event and replays the suffix — same O(suffix) win.
4. **Lineage.** `parent_session_id` column on the session row (Hermes pattern; `hermes_state.py:50`). On compaction-split, link old → new.
5. **Retention.**
   - JSONL ledgers: never auto-deleted (filesystem-first, like Codex, Claude). User opts in via `[memory.session.retention] hard_delete_after_days`.
   - events.db rows for ended sessions: tier into `memory_events_archive` after 90 days (Hermes default; `hermes_state.py:2178`). Promoted catalog rows survive the archive.
   - Auto-vacuum/prune cycle stamped in `meta` (Hermes `state_meta.last_auto_prune` pattern; `hermes_state.py:2205-2233`).
6. **Sub-agent ledgers belong to the parent thread tree.** Codex's `thread_spawn_edges` (migration 0021) is the cleanest model. AGH's session row already has fields for this; add `spawn_parent_id` if not there.

### Defense vs evidence

| Claim | Source |
|---|---|
| File-first ledgers are the dominant pattern | Codex `recorder.rs:1363`, Claude Code `<sessionId>.jsonl`, OpenClaw `<sessionId>.jsonl`, OpenFang `<workspace>/sessions/<sid>.jsonl` |
| Compaction-with-replacement-history is best-in-class resume | Codex `policy.rs:19-22`, `analysis_codex.md:140-146` |
| Pure DB works only with idempotent auto-maintenance | Hermes `hermes_state.py:2176-2247` |
| Live JSONL writer is a non-trivial concurrency build | Codex `recorder.rs:78-157`, OpenClaw `store.ts:576-647` lock queue |
| Forensic JSONL is the universal grep surface | Codex `stage_one_system.md:21`, Claude Code `transcriptSearch` (`memdir.ts:390-403`) |

---

## Q5: Daily-Log Retention — competitor evidence

### OpenClaw — `memory/YYYY-MM-DD.md`, append-only, NO file-level cap

- **Path**: `<workspaceDir>/memory/YYYY-MM-DD.md` (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/extensions/memory-core/src/flush-plan.ts:120`).
- **Write**: pre-compaction memory-flush turn appends; explicit prompt rule "APPEND new content only and do not overwrite existing entries" (`flush-plan.ts:14-16`).
- **Cap**: NONE on the daily file itself. The flush is *gated* by token thresholds (`DEFAULT_MEMORY_FLUSH_SOFT_TOKENS = 4000`, `DEFAULT_MEMORY_FLUSH_FORCE_TRANSCRIPT_BYTES = 2 MB`; `flush-plan.ts:10-11`) — i.e. how often the agent writes, not how big the file gets.
- **Distillation**: dreaming worker reads daily files via `daily-ingestion.json` checkpoint (`extensions/memory-core/src/dreaming-phases.ts:76`). Light/REM/deep phases consolidate; promoted items move to `MEMORY.md`.
- **Retention/rotation**: NONE. Daily files persist forever. Operator-managed.

### OpenFang — `memory/<YYYY-MM-DD>.md`, hard 1 MB cap, silent drop

```rust
// /Users/pedronauck/dev/knowledge/.resources/openfang/crates/openfang-kernel/src/kernel.rs:449-475
fn append_daily_memory_log(workspace: &Path, response: &str) {
    ...
    if let Ok(metadata) = std::fs::metadata(&log_path) {
        if metadata.len() > 1_048_576 {
            return;  // ← silent drop, no rotation, no warning
        }
    }
    let summary = openfang_types::truncate_str(trimmed, 500);
    ...
}
```

- 1 MB hard ceiling on the file. Once reached, all subsequent appends silently drop.
- Each entry truncated to 500 chars (UTF-8 safe).
- No rotation, no archive, no warning.
- **This is the wrong default** — operators discover the cap only when they `wc -l memory/*.md` and notice gaps.

### Claude Code memdir daily logs (KAIROS-gated)

- **Path**: `<autoMemDir>/logs/YYYY/MM/YYYY-MM-DD.md` (`/Users/pedronauck/Dev/compozy/agh/.resources/claude-code/memdir/memdir.ts:335`).
- **Cap**: NONE on the log file. `MEMORY.md` (the index, *not* the log) is capped at 200 lines / 25 KB.
- **Distillation**: separate nightly `/dream` skill (cited but external; `memdir.ts:319-326`).
- **Retention**: NONE.
- **Comment in source**: "A separate nightly process distills these logs into `MEMORY.md` and topic files." Logs intended to grow forever; distillation is the safety valve.

### Codex — no daily logs

Codex's parallel concept is per-rollout extraction (`stage1_outputs`), not daily logs. The Phase-2 consolidation reads stage1 rows ranked by `usage_count + last_usage`, not a time window. `DEFAULT_MEMORIES_MAX_ROLLOUT_AGE_DAYS = 10` (`config/types.rs:46`) caps how *fresh* a rollout has to be to enter Phase 1; old rollouts are skipped, not archived.

### paperclip PARA — access-count decay tiers (Hot/Warm/Cold)

From `analysis_goclaw-paperclip-multica.md:386-390`:
- **Hot** (last 7 days): include in `summary.md`.
- **Warm** (8-30d): include at lower priority.
- **Cold** (30+d or never accessed): drop from `summary.md`, keep in `items.yaml`.
- High `access_count` resists decay.
- "No deletion, only supersede" (line 413) — the canonical lifecycle policy.

`access_count` and `last_accessed` are tracked per-fact in YAML; tier transition is driven by whichever wins between time and access count. **No hard delete ever** — items move tiers, files relocate to `archives/` (analysis_goclaw-paperclip-multica.md §367).

### Summary table

| Harness | File cap | Rotation | Cold-storage threshold | Hard-delete | Distillation source |
|---|---|---|---|---|---|
| OpenClaw | none | none | none | never | dreaming worker (light/REM/deep) reads daily files |
| OpenFang | 1 MB | none (silent drop) | none | never | none |
| Claude Code | none on log; index 200 ln / 25 KB | none | none | never | nightly `/dream` skill |
| paperclip | none on file | tier-based summary inclusion | 30 days OR never-accessed | **never** | `summary.md` regenerated with Hot+Warm |

---

## Q5 Verdict for AGH

Adopt a **belt-and-suspenders** policy: a per-file cap + a Hot/Warm/Cold tiering + zero hard-delete by default. Concrete recommendation:

### Per-daily-file caps (size + lines, not just size)

| Cap | Default | Rationale |
|---|---|---|
| `max_bytes_per_file` | **1 MiB** (`1_048_576`) | Same ceiling as OpenFang but drives rotation, not silent drop |
| `max_lines_per_file` | **5000** | Catches long-line failure (Claude memdir lesson: "long lines are the failure mode the byte cap targets" — `memdir.ts:64-65`). 5000 lines × ~200 chars/line ≈ 1 MB, both caps fire roughly together. |
| `max_chars_per_entry` | **2000** | OpenFang truncates at 500. AGH ledger entries carry more structured payload (provenance, citations); 2000 is a comfortable middle. |

### Rotation policy

When a daily file would exceed either cap, rotate to `daily/YYYY-MM-DD.<seq>.md` (seq starts at 2; first overflow becomes `2026-05-04.2.md`, then `.3.md`). The bare `daily/YYYY-MM-DD.md` is always the *current* file the agent is told about; rotated files keep the same date for grep-ability.

This is what OpenFang lacks. Without rotation, hitting the cap is data loss. With rotation, the agent (and dreaming worker) just see "more daily files for today".

### Dreaming worker read window

**Last 7 days** (Hot, paperclip's window). Configurable. This is also the practical span over which compaction summaries reference recent context. Codex's stage-1 idle gate is 6 hours (`DEFAULT_MEMORIES_MIN_ROLLOUT_IDLE_HOURS = 6`, `config/types.rs:47`); 7 days for daily-log dreaming sits comfortably above.

### Cold-storage threshold

**30 days** (paperclip's Warm→Cold edge). Move `daily/YYYY-MM-DD*.md` to `_system/archive/YYYY-MM/` once the file's date is >30 days old AND the dreaming worker has marked it as ingested (i.e. `_system/extractor/<date>.jsonl` exists per `analysis_layered-scope.md:225`).

This matches:
- paperclip Hot/Warm/Cold transitions.
- OpenClaw's session retention default (`DEFAULT_SESSION_PRUNE_AFTER_MS = 30 days`; `store-maintenance.ts:10`).
- Hermes session prune offset (90d for the *DB*, but the curator-equivalent reads recent history only).

### Hard-delete threshold

**Never by default.** Configurable opt-in via `[memory.daily.retention] hard_delete_after_days = …`. Justification:
- paperclip's "no deletion, only supersede" rule (`analysis_goclaw-paperclip-multica.md:413`).
- Claude memdir is append-only forever (`memdir.ts:348`).
- OpenClaw and OpenFang both keep daily files indefinitely.
- Hermes deletes session DB rows but `analysis_hermes.md` shows it's gated on `ended_at IS NOT NULL` and orphans children — the closest precedent for AGH session DB rows, not for filesystem daily logs.

### Disk-cap safety valve

`max_archive_bytes = 1 GiB` (configurable). When `_system/archive/` exceeds this, oldest `_system/archive/YYYY-MM/` directories tier-list-deleted with a warning event. This mirrors OpenClaw's `maxDiskBytes` (`disk-budget.ts:243`) and Hermes's `vacuum()` post-prune sweep (`hermes_state.py:2153-2174`). Default is generous enough that most operators never hit it.

### Defense vs evidence

| Decision | Source |
|---|---|
| Per-file size cap with rotation | OpenFang `kernel.rs:451-475` (cap-without-rotation = data loss); Claude memdir `memdir.ts:35-38` (line+byte caps protect the prompt) |
| Hot read window 7 days | paperclip `analysis_goclaw-paperclip-multica.md:387` |
| Cold-archive 30 days | paperclip `:389` + OpenClaw `store-maintenance.ts:10` |
| Never hard-delete by default | paperclip `:413` + Claude memdir append-only stance + OpenClaw retains rotated `*.reset.<ts>.jsonl` permanently |
| Disk-cap safety valve | OpenClaw `disk-budget.ts` + Hermes `vacuum()` |

---

## AGH config keys (concrete)

Add to `config.toml`:

```toml
[memory.session]
# Where ended-session JSONL ledgers materialize. Path relative to workspace.
ledger_dir = ".agh/sessions"

# Whether to write the JSONL ledger snapshot on OnSessionEnd.
# false = events.db only. Default true.
write_ledger_on_end = true

# What to include in the ledger.jsonl. "limited" = Codex EventPersistenceMode::Limited
# equivalent (replay-relevant only). "extended" = adds tool I/O and diagnostics.
persistence_mode = "limited"

# Aggregated exec/tool output cap when persistence_mode = "extended" (bytes).
# Mirrors Codex PERSISTED_EXEC_AGGREGATED_OUTPUT_MAX_BYTES.
extended_output_cap_bytes = 10000

[memory.session.retention]
# Days after which ended-session events.db rows are tiered into
# memory_events_archive. Promoted catalog rows survive.
db_archive_after_days = 90

# Days after which ledger.jsonl files are hard-deleted from disk.
# 0 = never. Default 0.
hard_delete_after_days = 0

# How often the auto-maintenance sweep runs (hours). Mirrors Hermes
# state_meta.last_auto_prune cadence.
auto_maintenance_interval_hours = 24

# Whether to VACUUM events.db after a prune that freed rows.
vacuum_after_prune = true


[memory.daily]
# Directory for daily logs (relative to workspace).
dir = "memory/daily"

# Maximum bytes per daily file before rotation.
max_bytes_per_file = 1048576       # 1 MiB

# Maximum lines per daily file before rotation. Either cap triggers rotation.
max_lines_per_file = 5000

# Maximum characters per individual entry written by an agent or hook.
max_chars_per_entry = 2000

# Filename pattern for the rotated overflow. {date} = YYYY-MM-DD, {seq} = 2,3,...
rotation_pattern = "{date}.{seq}.md"

[memory.daily.dreaming]
# How many days of daily files the dreaming worker reads on each pass (Hot window).
read_window_days = 7

# Cron-style schedule for dreaming. Empty = disabled (dispatched by hooks instead).
schedule = ""

[memory.daily.archive]
# Move daily files older than this to archive_dir. 0 = never archive.
cold_archive_after_days = 30

# Archive sub-dir name (relative to memory.daily.dir parent).
archive_dir = "_system/archive"

# Group archived files by month: archive_dir/YYYY-MM/<file>.
group_by = "month"

[memory.daily.retention]
# Days after which archived daily files are hard-deleted. 0 = never.
hard_delete_after_days = 0

# Total disk budget across daily/ + archive/. When exceeded, oldest
# archive/YYYY-MM/ directories are tier-deleted with a memory_events
# audit row (op=archive_evicted). 0 = no cap.
max_archive_bytes = 1073741824     # 1 GiB
```

Defaults chosen so that the **out-of-the-box behavior is "never lose data"** (hard_delete = 0 everywhere), with rotation + tiering doing the work. Operators with disk pressure can dial down.

---

## Open sub-questions for the TechSpec

1. **Ledger-write back-pressure.** Materializing `ledger.jsonl` on `OnSessionEnd` is a synchronous fs write of potentially MB+. Is it inline in the lifecycle hook, or queued to a background goroutine with `WriteLedgerComplete` event? Codex uses a per-session tokio task (`recorder.rs:78-113`); AGH's hook dispatch is synchronous today. Recommend background goroutine + `memory_events op=ledger_materialized`.

2. **Ledger schema versioning.** Codex's `RolloutItem` enum is versioned via session-meta. AGH should declare a `LEDGER_VERSION` constant; bumping it is a hard cut (greenfield rule). Question: do we tag `ledger.jsonl` line 1 with `{type: "ledger_meta", version: 1, ...}` (Codex pattern) or carry version in `manifest.json`? Recommend line 1 `ledger_meta` so a single file is self-describing.

3. **Compacted event payload size.** Codex caps `replacement_history` implicitly via `COMPACT_USER_MESSAGE_MAX_TOKENS = 20_000` (`compact.rs:44`). AGH's `events.db` row has practical SQLite blob limits but no semantic cap. Define `MAX_COMPACTION_REPLACEMENT_BYTES` and middle-truncate. Recommend 256 KB.

4. **Rotation triggers — append vs flush.** Daily-file rotation should fire when a write *would* exceed the cap, not after. This means the writer needs to read current size pre-write. Question: is this a per-write `os.Stat` (cheap) or a tracked-in-process counter that drifts in multi-process setups? Recommend `os.Stat` per write — daily logs are low-frequency.

5. **Sub-agent ledgers.** Codex skips memory pipeline for sub-agents (`memories/write/src/start.rs:31-35`). Should sub-agent sessions write a ledger.jsonl, or only the parent's ledger captures the spawn edge + child's final assistant message? Recommend per-session ledger always, plus a `spawn_parent_id` field, so forensic replay of a sub-agent run is possible.

6. **Encryption / redaction.** Codex's `redact_secrets` (`secrets/sanitizer.rs:15-19`) runs on Phase-1 outputs only — *not* on the rollout JSONL itself. Should AGH's ledger.jsonl pass through `redact_secrets` on materialize? Recommend yes for ledger (filesystem grep surface), no for events.db (DB has its own access controls).

7. **Per-agent vs per-workspace daily logs.** OpenClaw daily logs are per-workspace; Codex memory is global. AGH today is per-workspace via `<workspace>/.agh/`. Confirm daily-log scope = workspace, not agent (else per-agent daily files multiply linearly with agent count and dreaming becomes O(agents × days)).

8. **Promotion gate from session events to durable scope.** When does `memory_events scope=session` get promoted to `scope=workspace` or `scope=global`? `analysis_layered-scope.md:111` says "unless a controller decision promotes them." TechSpec must define the controller (hook? slash command? auto-policy?). Recommend the dreaming worker is the only auto-promoter; humans/agents promote via explicit `memory promote --session <id>` UDS verb.

9. **Time-travel / replay correctness.** Codex's reverse-replay (`session/rollout_reconstruction.rs:86-222`) walks newest→oldest. AGH's forward-replay from `ledger.jsonl` is simpler but loses the `Compacted` short-circuit. Question: is replay performance important enough to copy reverse-replay, or is forward-replay-from-newest-Compacted-onwards good enough? Recommend forward-replay (simpler), with `Compacted` as the resume base.

10. **Disk-budget reaction.** When `max_archive_bytes` fires, OpenClaw's `mode=warn` previews. Should AGH copy this — `[memory.daily.retention] mode = "warn" | "enforce"`? Recommend yes, default `warn` for first cycle, `enforce` after operator acks.

---

## Closing summary

**Q1 answer**: hybrid event-log + on-end JSONL snapshot. DB live, filesystem forensic. Steal Codex's `Compacted.replacement_history` and Hermes's `parent_session_id` chain + idempotent auto-maintenance.

**Q5 answer**: 1 MiB / 5000-line per-file caps with seq rotation, 7-day Hot read window, 30-day cold-archive tier, no hard-delete by default, disk safety valve via `max_archive_bytes`. Steal paperclip's no-deletion+supersession lifecycle and Claude memdir's per-cap warning shape; do **not** copy OpenFang's silent-drop default.

Both answers preserve AGH's "files are evidence" stance while keeping live ops in the DB where concurrency is manageable.
