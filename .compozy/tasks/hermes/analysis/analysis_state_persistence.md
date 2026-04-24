# Hermes vs AGH — State, Persistence & Recovery

Scope: state, persistence, and recovery only. Line numbers are approximate.

## Executive Summary

- AGH already has the SQLite fundamentals right: WAL + `synchronous=NORMAL`
  + `busy_timeout` + `foreign_keys=ON`, atomic `meta.json` writes with
  parent-dir fsync, and corrupt-file quarantine
  (`internal/store/sqlite.go:88-208`, `internal/fileutil/atomic.go:14-46`).
- **One P0 correctness risk**: no app-level retry on `SQLITE_BUSY`. Hermes
  learned the hard way (`hermes_state.py:123-137`) that pure driver-level
  `busy_timeout` causes convoy stalls under concurrent writers. AGH will hit
  the same once daemon + reconciler + UDS + HTTP + automation all write to
  `agh.db`.
- **Three P1 gaps before release**: (1) no `schema_version` table / numbered
  migration runner, (2) `observability.retention_days` is config-only with
  no sweeper, (3) no `agh backup` / `agh restore` command.
- **One P1 security call**: no per-session subprocess HOME. Spawned Claude /
  Codex CLIs share the operator's `~/.config`, `~/.claude`, `~/.npm`. Fine
  solo; unsafe the moment two AGH sessions or a shared user enter the room.
- **Do not port**: FTS5-over-messages, trajectory JSONL dumps, shadow-git
  checkpoint manager, Hermes' per-project migration chain (steal the pattern,
  not the table).

## Capability-by-Capability Gap Analysis

### 1. Write-contention strategy under multiple writers — P0
- **Hermes**: 1s timeout + `BEGIN IMMEDIATE` + 15-attempt retry with 20–150ms
  random jitter, explicitly to dodge SQLite's deterministic-backoff convoys
  (`hermes_state.py:132-214`).
- **AGH today**: driver-level `busy_timeout(5000)` only
  (`internal/store/sqlite.go:94-118`); mutators use default deferred
  transactions.
- **Gap**: no immediate locking, no jittered retry. Under daemon + CLI +
  reconciler + automation concurrency, busy waits serialize into a convoy.
- **Recommended adoption**: add `ExecWithRetry(ctx, db, fn)` in
  `internal/store/sql_helpers.go` that wraps `BEGIN IMMEDIATE` and retries on
  `SQLITE_BUSY`/`SQLITE_LOCKED` with jittered sleep. Use it across
  `internal/store/globaldb/` and `internal/store/sessiondb/`. After that
  lands, drop `busy_timeout` to ~1s.

### 2. Schema versioning & migration runner — P1
- **Hermes**: single-row `schema_version` table + explicit 1→6 numbered
  cascade (`hermes_state.py:252-341`).
- **AGH today**: migrations driven by `PRAGMA table_info` probing
  (`migrate_workspace.go:57-104, 560-579`). Works, but ordering is implicit
  and there's no "what DB am I?" read.
- **Recommended adoption**: add `schema_migrations(version INTEGER PRIMARY
  KEY, applied_at TEXT)` + a tiny runner in `internal/store/schema.go`; keep
  the column-probe helpers as safety nets. Log `schema_version` on boot.

### 3. Corruption detection — AGH leads
- **AGH** already matches known corruption markers and renames the bad file
  to `.corrupt.<ts>` alongside `-wal`/`-shm` before reopening
  (`internal/store/sqlite.go:37-54, 165-208`). Hermes has no equivalent.
- **Optional P2**: periodic `PRAGMA integrity_check` in the observer loop.

### 4. Atomic file writes (non-DB state) — P2 polish
- **Both** use temp + fsync + rename. Hermes additionally preserves original
  file mode (`utils.py:35-161`) so subsequent writes don't clobber an
  operator's `chmod`. AGH's `fileutil.AtomicWriteFile`
  (`internal/fileutil/atomic.go:14-46`) resets to the requested `perm`
  every time.
- **Recommended adoption**: add a "preserve existing mode if present"
  option to `AtomicWriteFile`.

### 5. WAL growth control — P2
- **Hermes**: passive `wal_checkpoint(PASSIVE)` every 50 writes + on close
  (`hermes_state.py:193-250`).
- **AGH today**: checkpoint only on close
  (`internal/store/globaldb/global_db.go:559-573`,
  `internal/store/sqlite.go:154-163`). Daemon runs for days; WAL can grow
  unbounded.
- **Recommended adoption**: a periodic ticker (5 min) inside `GlobalDB`
  calling a new passive variant of `store.Checkpoint`; keep TRUNCATE on close.

### 6. Backup & restore — P1
- **Hermes**: `hermes backup` / `hermes import` CLIs using SQLite's online
  backup for `.db` files (`hermes_cli/backup.py:75-98`), zipping the
  home dir minus transient files, traversal-safe extract.
- **AGH today**: nothing. Users who `cp agh.db` get an inconsistent
  WAL snapshot.
- **Recommended adoption**: add `agh backup` / `agh restore` in
  `internal/cli/`. For each `.db`, use `VACUUM INTO` (single statement,
  WAL-safe). Zip the AGH home minus socket/pid files. Port Hermes' path-
  traversal guard (`hermes_cli/backup.py:362-367`) on restore.

### 7. Session state transition graph — AGH leads
- **AGH** has a real state machine with `canTransition` gate
  (`internal/session/session.go:22-29, 599-698`). Hermes is open-ended
  (`end_reason` string). No action.

### 8. Unclean-shutdown recovery — AGH leads (verify boot path)
- **Hermes**: writes a JSON checkpoint of live PIDs, probes them on restart
  via `os.kill(pid, 0)`, marks survivors `detached=True`
  (`tools/process_registry.py:1003-1112`).
- **AGH today**: stronger semantically —
  `ClassifyInactiveMetaForRecovery` stalls sessions idle >2min,
  `RepairLegacyProvider` normalizes legacy meta before resume
  (`internal/session/liveness.go:13-113`,
  `internal/session/resume_repair.go:59-150`). `procutil.Alive(pid)` is
  the PID probe.
- **Gap**: confirm the daemon boot path always runs
  `observer.Reconcile()` before accepting traffic; log the adoption /
  stall counts.
- **Priority**: P2 (verify, then done).

### 9. Concurrent-process access — folded into §1
- Hermes explicitly supports multi-process writers via retry+jitter. AGH's
  daemon is the only writer today, but intra-process contention alone
  justifies §1's retry loop.

### 10. Hot vs cold state separation — AGH leads
- AGH splits global index (`agh.db`) from per-session events
  (`events.db`) (`internal/store/store.go:11-18`,
  `internal/store/sessiondb/session_db.go:22-67`). Deleting a session =
  `rm -rf <dir>`. Hermes crams everything into one `state.db`.

### 11. Retention / pruning — P1
- **Hermes**: `prune_sessions(older_than_days=90)`
  (`hermes_state.py:1404-1443`).
- **AGH today**: `observability.retention_days` exists in config
  (`internal/config/config.go:93-101, 828`) but nothing consumes it.
  `DeleteSession` is only invoked via API
  (`internal/api/core/handlers.go:272-275`). No scheduled sweep.
- **Gap**: shipping a config knob that does nothing is worse than not
  having it.
- **Recommended adoption**: daily sweep in `internal/observe/` that
  deletes sessions with `state='stopped' AND updated_at < now - retention`;
  log a `memory_operation_log` entry.

### 12. Subprocess HOME isolation — P1
- **Hermes**: `get_subprocess_home()` returns a per-profile HOME that gets
  injected into child env; the Python parent's own HOME is never mutated
  (`tests/test_subprocess_home_isolation.py:20-199`,
  `tools/environments/local._make_run_env`).
- **AGH today**: nothing. `grep` of `internal/acp/` / `internal/session/`
  shows no HOME/XDG override on subprocess spawn. ACP agents inherit the
  operator's full home — breaks for system-service users, for two sessions
  that want separate Claude auth, and for containerized deployments.
- **Recommended adoption**: add an `agents.isolate_home` toggle (default
  off to preserve solo UX). When on, create
  `~/.agh/sessions/<id>/home/` and inject as `HOME` + `XDG_*` when spawning
  in `internal/acp/`. Mirror Hermes' test matrix exactly (unset /
  missing-dir / A-vs-B / parent-unchanged).

### 13. Filesystem checkpoint manager — P2 (defer)
- Hermes' shadow-git `CheckpointManager` (`tools/checkpoint_manager.py`)
  exists because Hermes owns its tools' filesystem access. ACP puts
  mutations on the other side of a protocol boundary — we do not. Defer
  until AGH has native file-mutating tools.

### 14. Large tool-result persistence — N/A
- ACP owns the wire format; spillage is the agent's problem
  (`tools/tool_result_storage.py`).

## Patterns worth stealing (code-level)

1. `BEGIN IMMEDIATE` + jittered retry helper
   (`hermes_state.py:164-214` → `internal/store/sql_helpers.go`). Log a
   warning when retry count ≥ 3.
2. `schema_migrations` version table + linear runner. Keep the PRAGMA
   probing as a safety net, not as the gate.
3. Periodic `wal_checkpoint(PASSIVE)` — every ~5 min / N writes.
4. `VACUUM INTO` for online DB backup under WAL (Go equivalent of Hermes'
   `_safe_copy_db`).
5. Mode-preserving `AtomicWriteFile` — port Hermes' `_preserve_file_mode` /
   `_restore_file_mode` pair (`utils.py:35-161`).
6. Zip-traversal guard on restore (`hermes_cli/backup.py:362-367`).
7. Subprocess HOME isolation test suite, even if the feature ships off by
   default (`tests/test_subprocess_home_isolation.py:20-199`).

## Things Hermes does that AGH should explicitly NOT adopt

1. FTS5 over messages — duplicates `internal/transcript/`, couples search
   to schema.
2. Trajectory JSONL side-writes (`agent/trajectory.py`) — RL fine-tuning,
   not our domain.
3. Shadow-git checkpoint manager — see §13.
4. Hermes' 1→6 migration chain as a literal template — port the *shape*
   (version table + numbered upgrade fns), not the billing/reasoning
   columns.
5. Title resolution / compression-chain projection
   (`hermes_state.py:662-908`) — feature AGH doesn't have; comes with
   LIKE-escaping cruft.
6. Tool-result spill-to-disk — ACP boundary.
7. Python `GIT_CONFIG_GLOBAL=/dev/null` tricks — only relevant if we ever
   ship the shadow-repo feature; Go's `exec.Command` gives us env control
   for free.

---

**Release-blocking list:**
- **P0** — `BEGIN IMMEDIATE` + jittered retry
  (`internal/store/sql_helpers.go`).
- **P1** — `schema_migrations` table (`internal/store/schema.go`).
- **P1** — Implement or remove `observability.retention_days` sweep
  (`internal/observe/`).
- **P1** — `agh backup` / `agh restore` (`internal/cli/`).
- **P1** — Decide per-session HOME isolation (implement with flag, or
  document as post-release) (`internal/acp/`).
