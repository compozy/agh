# Improvements Report — internal/store

## Skill Invocation Log

| Skill                         | Status  | Evidence / Artifact Reference |
| ----------------------------- | ------- | ----------------------------- |
| refactoring-analysis          | run     | cyclomatic top-10 + file-size + duplication below |
| extreme-software-optimization | run     | `internal/store/globaldb/perf_bench_test.go`, `internal/store/sessiondb/perf_bench_test.go`, benchmark table below |
| ubs                           | not-run | No skill runner tool is available in this session; only local `SKILL.md` instructions are exposed. |
| deadlock-finder-and-fixer     | run     | goroutine/channel/mutex/select inventories below |
| security-review               | run     | threat model + attacker-input surface inventory below |

## Inventories

### Refactoring — Cyclomatic Top-10

| Complexity | Symbol | File:Line |
| ---------- | ------ | --------- |
| 18 | `(*GlobalDB).ReplaceBridgeInstances` | `internal/store/globaldb/global_db_bridge.go:231` |
| 15 | `(*SessionDB).QueryHookRuns` | `internal/store/sessiondb/session_db.go:241` |
| 15 | `(*GlobalDB).ReconcileSessions` | `internal/store/globaldb/global_db_session.go:113` |
| 13 | `reconcileLegacySessionMetaWorkspaceIDs` | `internal/store/globaldb/migrate_workspace.go:908` |
| 12 | `(NetworkAuditEntry).Validate` | `internal/store/types.go:363` |
| 12 | `validateAutomationRunRecord` | `internal/store/globaldb/global_db_automation.go:1362` |
| 12 | `migrateNetworkAuditTable` | `internal/store/globaldb/migrate_workspace.go:393` |
| 12 | `migrateGlobalSchema` | `internal/store/globaldb/migrate_workspace.go:56` |
| 12 | `migrateBridgeInstanceColumns` | `internal/store/globaldb/migrate_workspace.go:223` |
| 12 | `loadLegacySessions` | `internal/store/globaldb/migrate_workspace.go:472` |

### Refactoring — Files > 300 LOC

| File | LOC | Unit-smell summary |
| ---- | ---: | ------------------ |
| `internal/store/globaldb/global_db_automation.go` | 1627 | Job/trigger CRUD, overlay, run, and webhook-secret concerns are co-located and duplicated. |
| `internal/store/globaldb/global_db_bridge.go` | 1116 | Bridge instance CRUD, routes, secret bindings, ingest dedup, and scanners live in one file. |
| `internal/store/globaldb/global_db_task.go` | 1049 | Task record and task-run persistence/query concerns are densely packed. |
| `internal/store/globaldb/migrate_workspace.go` | 1008 | Migration orchestration and many schema-rewrite helpers are concentrated in a single file. |
| `internal/store/globaldb/global_db_task_aux.go` | 839 | Dependency, idempotency, and task-event helpers mix graph logic with persistence plumbing. |
| `internal/store/sessiondb/session_db.go` | 766 | Writer-loop lifecycle, writes, queries, scanners, and shutdown helpers share one file. |
| `internal/store/types.go` | 526 | Many unrelated persistence DTOs and validators are defined in one package-level types file. |
| `internal/store/globaldb/global_db_session.go` | 498 | Session registry, state updates, reconciliation, and environment scanning are bundled together. |
| `internal/store/globaldb/global_db.go` | 492 | Schema/bootstrap, lifecycle, and readiness helpers share the same file. |
| `internal/store/globaldb/global_db_workspace.go` | 378 | Workspace CRUD plus scan/normalization helpers are moderately oversized. |

### Refactoring — Duplication

- `internal/store/globaldb/global_db_automation.go:88-126` ↔ `internal/store/globaldb/global_db_automation.go:250-293`
- `internal/store/globaldb/global_db_automation.go:609-640` ↔ `internal/store/globaldb/global_db_bridge.go:193-228`
- `internal/store/globaldb/global_db_bridge.go:193-228` ↔ `internal/store/globaldb/global_db_workspace.go:170-201`
- `internal/store/globaldb/global_db_observe.go:160-198` ↔ `internal/store/globaldb/global_db_session.go:66-110`
- `internal/store/globaldb/global_db_task.go:125-154` ↔ `internal/store/globaldb/global_db_task.go:337-365`
- `internal/store/globaldb/migrate_workspace.go:96-120` ↔ `internal/store/globaldb/migrate_workspace.go:367-391`

### Optimization — Hot-Path Candidates

| Function | File:Line | Reasoning | Benchmark |
| -------- | --------- | --------- | --------- |
| `(*GlobalDB).ReplaceBridgeInstances` | `internal/store/globaldb/global_db_bridge.go:231` | Bridge projection reconcile path loops over every instance and currently re-normalizes payloads during each upsert. | `BenchmarkReplaceBridgeInstances` |
| `(*SessionDB).Query` | `internal/store/sessiondb/session_db.go:316` | Session event retrieval backs follow-style API reads and transcript assembly. | `BenchmarkSessionDBQuery` |
| `(*SessionDB).History` | `internal/store/sessiondb/session_db.go:371` | Transcript history groups queried events by turn on every session-history read. | `BenchmarkSessionDBHistory` |

### Optimization — Benchmark Results

Baseline command: `go test -bench=. -benchmem -count=5 ./internal/store/...`

| Benchmark | Before ns/op (median) | Before B/op (median) | After ns/op | After B/op | Decision |
| --------- | --------------------: | -------------------: | ----------: | ---------: | -------- |
| `BenchmarkReplaceBridgeInstances` | 3691889 | 559153 | 3393633 | 416201 | fixed-with-benchmark |
| `BenchmarkSessionDBQuery` | 347776 | 210314 | 341760 | 210314 | not-hot-confirmed-by-benchmark |
| `BenchmarkSessionDBHistory` | 363490 | 287059 | 362079 | 287059 | not-hot-confirmed-by-benchmark |

### UBS Invocation Output

`not-run — No skill runner tool is available in this session; only local SKILL.md instructions are exposed.`

### Concurrency — Goroutine Inventory

| File:Line | Owner | Shutdown mechanism | Notes |
| --------- | ----- | ------------------ | ----- |
| `internal/store/sessiondb/session_db.go:155` | `SessionDB` dedicated writer goroutine | `Close()` sends `shutdownCh`, waits on `writerDone`, and cancels `writerCtx` | Single writer serializes per-session SQLite writes. |

### Concurrency — Channel Inventory

| File:Line | Capacity | Owner | Closer | Readers | Notes |
| --------- | -------- | ----- | ------ | ------- | ----- |
| `internal/store/sessiondb/session_db.go:103` (`writeCh`) | 256 (`defaultWriteBufferSize`) | `SessionDB` | never explicitly closed | `writerLoop`, `drainWrites` | Request queue for event, token-usage, and hook-run writes. |
| `internal/store/sessiondb/session_db.go:104` (`shutdownCh`) | 1 | `SessionDB.Close` | never explicitly closed | `writerLoop` | Close handshake channel carrying drain context/result. |
| `internal/store/sessiondb/session_db.go:105` (`writerDone`) | unbuffered | writer goroutine | writer goroutine (`close(writerDone)`) | `Close`, `waitForWriterExit` | Completion signal after writer-loop exit. |

### Concurrency — Mutex Inventory

| File:Line | Read/Write | Protects | Notes |
| --------- | ---------- | -------- | ----- |
| `internal/store/sessiondb/session_db.go:109` (`acceptMu`) | read-heavy | enqueue-vs-close admission around `state`, `writeCh`, and `shutdownCh` | `RLock` guards normal enqueues; `Lock` serializes shutdown request injection. |

### Concurrency — Select Audit

- `internal/store/sessiondb/session_db.go:442` send-to-`writeCh` wait includes `ctx.Done()`
- `internal/store/sessiondb/session_db.go:448` result wait includes `ctx.Done()`
- `internal/store/sessiondb/session_db.go:458` writer loop includes `writerCtx.Done()`
- `internal/store/sessiondb/session_db.go:474` drain loop includes `ctx.Done()`
- `internal/store/sessiondb/session_db.go:714` shutdown-result wait includes `ctx.Done()`
- `internal/store/sessiondb/session_db.go:726` writer-exit wait includes `ctx.Done()`

All production `select` sites are cancellation-aware or use the writer-owned shutdown context.

### Security — Threat Model

- Trust boundaries:
  - Internal callers in `session`, `daemon`, `task`, `automation`, `bridges`, `api/core`, and `extensiontest` invoke `store`, `globaldb`, and `sessiondb` through direct Go interfaces.
  - `meta.go` and `sqlite.go` cross the filesystem boundary into per-session metadata files and SQLite database files.
  - SQLite query helpers construct limited SQL fragments from internal query-builder inputs.
- Attacker capabilities:
  - Indirectly influence persisted fields through CLI/HTTP/network-peer/session-agent/task payloads that higher layers pass into this package.
  - Cannot directly execute SQL or choose arbitrary DB/meta paths unless a higher layer already violates its own path derivation rules.
- In-scope assets:
  - Integrity of `agh.db`, per-session `events.db`, and session metadata files.
  - Integrity of persisted task, automation, bridge, session, audit, and network-log records.
- Out-of-scope:
  - A compromised local OS user with direct filesystem access.
  - Malicious dependencies or a compromised SQLite driver.
  - Higher layers bypassing this package’s validators and calling SQLite directly.

### Security — Attacker-Input Surface Inventory

| File:Line | Source | Sanitization | Sink | Verdict |
| --------- | ------ | ------------ | ---- | ------- |
| `internal/store/meta.go:15` | Session metadata path from session/daemon callers | `strings.TrimSpace` only; caller derives path | `os.ReadFile` + JSON decode | LOW — internal-only derived path; no evidence of attacker-controlled path traversal within this package. |
| `internal/store/meta.go:38` | Session metadata path + `SessionMeta` payload from manager/environment reconcile | `meta.Validate()` + atomic write | `fileutil.AtomicWriteFile` | LOW — payload is validated and persisted as data; path remains an internal caller concern. |
| `internal/store/sqlite.go:20` | SQLite database path from daemon/session home-path resolution | trimmed path + parent `MkdirAll` | `sql.Open` / DB file creation | LOW — filesystem access is internal-path driven; no attacker-controlled path join exists here. |
| `internal/store/sqlite.go:134` | Identifier strings used by query helpers | strict identifier allowlist | query-fragment generation in `sql_helpers.go` | LOW — allowlisted identifier characters block SQL injection through helper-built fragments. |
| `internal/store/sql_helpers.go:24` | Query filter columns/operators from internal list/query builders | identifier/operator allowlists, bound args | `BuildClauses` / `AppendWhere` / `AppendLimit` across `globaldb` and `sessiondb` | LOW — dynamic SQL is constrained to validated identifiers and placeholders. |
| `internal/store/sessiondb/session_db.go:180` | Session event content and metadata from agent/provider outputs | `SessionEvent.Validate()` and parameterized insert in `writeEvent` | `events` table | LOW — data is stored via bound parameters; no code execution or SQL injection path. |
| `internal/store/sessiondb/session_db.go:224` | Hook audit records and optional patch JSON from hook runtime | event/source/mode validation + bound parameters | `hook_runs` table | LOW — untrusted data is persisted as JSON/text only. |
| `internal/store/globaldb/global_db_task.go:21` | Task/task-run fields from API, automation, and task manager callers | normalization/validation + bound parameters | `tasks` / `task_runs` tables | LOW — inserts/updates are parameterized and scoped by explicit validators. |
| `internal/store/globaldb/global_db_automation.go:19` | Automation job/trigger specs from resources/API callers | normalization, JSON encoding, uniqueness constraints | automation tables | LOW — persisted as validated data; no dynamic execution in this package. |
| `internal/store/globaldb/global_db_network_messages.go:14` | Network text/intents from peer/runtime message flows | `NetworkMessageEntry.Validate()` + bound parameters | `network_message_log` | LOW — message payloads are stored only as text. |
| `internal/store/globaldb/global_db_network_audit.go:13` | Network audit metadata from runtime/network handlers | `NetworkAuditEntry.Validate()` + bound parameters | `network_audit_log` | LOW — validation plus parameterized SQL blocks injection; remaining risk is operational volume, not an exploit path proven here. |

## Findings

| ID | Skill | Severity | File:Line | Summary | Decision |
| -- | ----- | -------- | --------- | ------- | -------- |
| 01 | refactoring-analysis | medium | `internal/store/globaldb/global_db_session.go:392` | `scanSessionEnvironment` silently discarded malformed persisted `environment_last_sync_at` values instead of surfacing store corruption. | fixed |
| 02 | extreme-software-optimization | low | `internal/store/globaldb/global_db_bridge.go:231` | `ReplaceBridgeInstances` normalized and JSON-prepared every bridge instance twice per batch swap. | fixed |
| 03 | refactoring-analysis | medium | `internal/store/globaldb/global_db_automation.go:1` | Job/trigger CRUD and query logic remain duplicated inside a 1627-LOC file. | deferred — needs a broader package-local extraction pass than this focused improvements task. |
| 04 | refactoring-analysis | medium | `internal/store/globaldb/migrate_workspace.go:1` | The migration orchestrator still combines many schema-rewrite flows in a 1008-LOC file. | deferred — splitting migration epochs/tables is structural work that would widen this pass substantially. |

## Per-Skill Notes

### refactoring-analysis

- Fixed the inconsistent timestamp-parse behavior in `scanSessionEnvironment` so corrupted persisted `environment_last_sync_at` values now fail the scan instead of being silently dropped.
- Large-file and duplication pressure remains concentrated in `globaldb`, especially the paired job/trigger helpers in `global_db_automation.go` and the schema migration aggregator in `migrate_workspace.go`; both are deferred because they require broader structural extraction than this focused pass.

### extreme-software-optimization

- `ReplaceBridgeInstances` now reuses the normalized/encoded bridge payload prepared in the outer replacement loop instead of recomputing it inside every upsert.
- Median benchmark change for `BenchmarkReplaceBridgeInstances`: `3691889 ns/op -> 3393633 ns/op` and `559153 B/op -> 416201 B/op` with allocs dropping from `9372` to `6555`.
- `SessionDB.Query` and `History` remained effectively flat under the full-suite benchmark command, so they are recorded as not-hot rather than “optimized”.

### ubs

- `ubs` remains `not-run` because this session exposes only the local skill instructions and no callable skill runner.

### deadlock-finder-and-fixer

- The only production concurrency surface is the `SessionDB` writer loop; all admission/drain waits are bounded by context cancellation or explicit writer shutdown.
- No concrete deadlock, goroutine leak, channel-cycle, or mutex-ordering finding survived the inventory audit.

### security-review

- No high-confidence vulnerabilities identified so far.
- All attacker-controlled inputs inspected in this package currently terminate in validated filesystem operations or parameterized SQLite writes rather than dynamic execution.

## Deferred Items (carry forward)

- **03** — Split the duplicated job/trigger CRUD/query/update flows in `internal/store/globaldb/global_db_automation.go` into smaller focused helpers/files. This is package-local but large enough to deserve a dedicated refactor pass with broad regression coverage.
- **04** — Extract the migration steps in `internal/store/globaldb/migrate_workspace.go` into narrower units (for example by table family or migration epoch). This is structurally valuable but too wide to mix with the current correctness/perf changes.

## `make verify`

Command: `make verify`

Executed after the final `internal/store/` code changes using a captured log at `/tmp/task28-make-verify.LZpCcs`.

Excerpt:

```text
Found 0 warnings and 0 errors.
✓ built in 351ms
ld: warning: -bind_at_load is deprecated on macOS
0 issues.
✓  internal/store (cached)
✓  internal/store/sessiondb (1.182s)
✓  internal/store/globaldb (8.01s)

DONE 4500 tests in 11.272s
OK: all package boundaries respected
```

Non-blocking environment warnings observed earlier in the same run:

- Node repeatedly printed `Warning: The 'NO_COLOR' env is ignored due to the 'FORCE_COLOR' env being set.`
- The vendored `golangci-lint` build on macOS printed `ld: warning: -bind_at_load is deprecated on macOS`.
