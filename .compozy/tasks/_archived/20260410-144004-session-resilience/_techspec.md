# TechSpec: Session Resilience

## Executive Summary

This TechSpec adds two resilience capabilities to AGH's session lifecycle: a canonical stop reason taxonomy and infrastructure-level repair on resume. Together they close the gaps identified in the extensibility analysis (P3, P4) where AGH has no classification of why sessions stopped and no validation when resuming after a crash.

The implementation strategy adds a `StopReason` type in `internal/store` (avoiding import cycles), an explicit `StopCause` signal in the session lifecycle (so classification doesn't rely on context inference), and a validation pipeline in `Resume()`. These ship as two standalone PRs with **zero hooks dependency** — hook integration points are prepared as seams but don't block delivery.

Loop/recursion guards (P5) — including the LoopGuard with SHA-256 hashing, graduated verdicts, and iteration budgets — are **deferred to Phase 2** after the hooks platform is fully wired and real session data shows which loop patterns occur in practice. The hooks techspec already defines `tool.pre_call` as a sync-eligible deny-capable hook point, which is the correct intervention surface for pre-execution guards. Phase 2 will use that hook point, not the post-execution `tool.post_call` observation model originally proposed.

The primary trade-off is shipping safety incrementally: Phase 1 provides observability and crash recovery without loop protection. This is acceptable because ACP agents (Claude Code, Codex, Gemini CLI) have their own in-loop guards, and AGH's iteration budget can be added as a trivial `tool.pre_call` native hook during hooks task 10 (turn/tool dispatch integration) without a separate techspec.

## System Architecture

### Component Overview

- **`StopReason` type** — Lives in `internal/store` (same package as `SessionMeta`) to avoid import cycles between `session` and `store`. Constants for 10 canonical values. Validated at the store boundary.

- **`StopCause`** — New type in `internal/session` that explicitly signals WHY a stop was requested. Set by the caller of `Stop()` and by `handleProcessExit()`. Eliminates the need to infer `user_canceled` vs `shutdown` vs `completed` from `ctx.Err()`.

- **Stop reason classification** — Logic in `finalizeStopped()` that maps `StopCause` + `waitErr` + process exit status → `StopReason`. Deterministic, testable, no context inference.

- **Resume repair pipeline** — Validation functions in `Resume()` that check infrastructure state before starting the ACP agent. Classifies previous crashes from meta state. Hook dispatch points are prepared as no-op seams until the hooks platform wires them.

### Data Flow

```
Stop request (with StopCause)
    ↓
finalizeStopped(ctx, session, waitErr)
    ↓
classifyStopReason(cause, waitErr, processExitStatus)
    ↓
StopReason persisted to:
  → SessionMeta (JSON on disk)
  → Session.Info() (in-memory)
  → Global DB sessions table
  → session_stopped event
```

```
Resume(ctx, id)
    ↓
ReadSessionMeta()
    ↓
classifyPreviousStop(meta) → set StopReason if crashed
    ↓
validateInfrastructure(meta) → workspace, agent def, event store, meta fields
    ↓
[hook seam: session.pre_resume — no-op until hooks wired]
    ↓
resolveResumeWorkspace() → driver.Start() → activateAndWatch()
    ↓
[hook seam: session.post_resume — no-op until hooks wired]
```

## Implementation Design

### Core Interfaces

```go
// internal/store/types.go — StopReason lives here to avoid import cycles
type StopReason string

const (
    StopCompleted      StopReason = "completed"
    StopUserCanceled   StopReason = "user_canceled"
    StopMaxIterations  StopReason = "max_iterations"
    StopLoopDetected   StopReason = "loop_detected"
    StopTimeout        StopReason = "timeout"
    StopBudgetExceeded StopReason = "budget_exceeded"
    StopError          StopReason = "error"
    StopAgentCrashed   StopReason = "agent_crashed"
    StopHookStopped    StopReason = "hook_stopped"
    StopShutdown       StopReason = "shutdown"
)

func ValidStopReason(r StopReason) bool
```

```go
// internal/session/stop_cause.go — explicit stop request signal
type StopCause int

const (
    CauseNone         StopCause = iota // not yet stopped
    CauseCompleted                      // agent finished naturally
    CauseUserRequested                  // user called Stop()
    CauseShutdown                       // daemon shutting down
    CauseHookDenied                     // hook denied continuation
    CauseProcessExited                  // subprocess exited unexpectedly
)
```

### Data Models

**SessionMeta** (extended — `internal/store/types.go`):

```go
type SessionMeta struct {
    // ... existing fields ...
    StopReason *StopReason `json:"stop_reason,omitempty"`
    StopDetail string      `json:"stop_detail,omitempty"`
}
```

`Validate()` on `SessionMeta` checks `StopReason` membership via `ValidStopReason()` when non-nil.

**SessionInfo** (extended — `internal/session/session.go`):

```go
type SessionInfo struct {
    // ... existing fields ...
    StopReason store.StopReason
    StopDetail string
}
```

**Session** (extended — `internal/session/session.go`):

```go
type Session struct {
    // ... existing fields ...
    stopCause  StopCause // set by prepareStop() or handleProcessExit()
    stopReason store.StopReason
    stopDetail string
}
```

**Global DB sessions table** (extended — `internal/store/globaldb/`):

```sql
ALTER TABLE sessions ADD COLUMN stop_reason TEXT;
ALTER TABLE sessions ADD COLUMN stop_detail TEXT;
```

**SessionStateUpdate** (extended):

```go
type SessionStateUpdate struct {
    // ... existing fields ...
    StopReason *string
    StopDetail string
}
```

### Stop Cause Propagation

The `StopCause` is set explicitly at each stop initiation point:

| Call Site | StopCause | How |
|-----------|-----------|-----|
| `Manager.Stop(ctx, id)` | `CauseUserRequested` | New parameter or method variant |
| `daemon.Shutdown()` → `sessions.Stop()` | `CauseShutdown` | Set on session before calling Stop |
| `handleProcessExit()` — clean exit | `CauseCompleted` | `waitErr == nil` and `stopWasRequested()` is false |
| `handleProcessExit()` — unexpected exit | `CauseProcessExited` | `waitErr != nil` |
| Future: hook denied continuation | `CauseHookDenied` | Hook pipeline sets cause before stopping |
| Future: iteration budget exhausted | `CauseUserRequested` + detail | Guard calls `Manager.Stop()` with budget reason |

### Stop Reason Classification Logic

In `finalizeStopped(ctx, session, waitErr)`:

```go
func classifyStopReason(cause StopCause, waitErr error, detail string) (store.StopReason, string) {
    switch cause {
    case CauseShutdown:
        return store.StopShutdown, "daemon shutdown"
    case CauseHookDenied:
        return store.StopHookStopped, detail
    case CauseUserRequested:
        // Check if detail indicates a guard trigger
        if strings.Contains(detail, "max_iterations") {
            return store.StopMaxIterations, detail
        }
        if strings.Contains(detail, "loop_detected") {
            return store.StopLoopDetected, detail
        }
        if strings.Contains(detail, "budget_exceeded") {
            return store.StopBudgetExceeded, detail
        }
        return store.StopUserCanceled, detail
    case CauseProcessExited:
        if waitErr != nil {
            return store.StopAgentCrashed, waitErr.Error()
        }
        return store.StopError, "process exited unexpectedly"
    case CauseCompleted:
        return store.StopCompleted, ""
    default:
        if waitErr != nil {
            return store.StopError, waitErr.Error()
        }
        return store.StopCompleted, ""
    }
}
```

No context inference. No checking `ctx.Err()` to guess intent. The cause is explicit.

### Resume Repair Pipeline

Inserted into `Resume()` after `ReadSessionMeta()` and before `resolveResumeWorkspace()`:

```
1. ReadSessionMeta(metaPath)

2. classifyPreviousStop(meta):
   - meta.State == "active"   → StopReason = "agent_crashed", detail = "daemon crashed while session active"
   - meta.State == "stopping" → StopReason = "agent_crashed", detail = "stop did not complete"
   - meta.State == "starting" → StopReason = "error", detail = "start did not complete"
   - meta.State == "stopped"  → use existing StopReason from meta (already classified)

3. validateInfrastructure(meta) → returns []error (independent checks):
   a. os.Stat(workspace.RootDir) — exists and accessible?
   b. config.ResolveAgent(meta.AgentName) — agent definition still present?
   c. os.Stat(sessionDBPath) — event store file exists and size > 0?
   d. meta.ID, meta.AgentName, meta.WorkspaceID all non-empty?

4. IF meta was classified as crashed in step 2:
   - Update meta.StopReason and meta.StopDetail
   - WriteMeta to persist the classification
   - Log structured event: session.resume.crash_classified

5. [HOOK SEAM] — prepared for session.pre_resume dispatch
   When hooks platform wires this event, payload includes:
   session_id, agent_name, workspace_id, previous_stop_reason, previous_stop_detail

6-9. [existing] resolveResumeWorkspace() → ResolveAgent() → driver.Start() → activateAndWatch()

10. [HOOK SEAM] — prepared for session.post_resume dispatch
    Payload includes: session_id, agent_name, resume_from_crash (bool)
```

### API Endpoints

Existing endpoints extended, no new endpoints:

- `GET /api/sessions` — response includes `stop_reason` and `stop_detail` on stopped sessions
- `GET /api/sessions/:id` — response includes `stop_reason` and `stop_detail`
- Session SSE events include `stop_reason` in the `session_stopped` event payload

## Integration Points

- **Store** (`internal/store`) — `StopReason` type defined here. `SessionMeta` extended with new fields.
- **Global DB** (`internal/store/globaldb`) — New columns, updated queries for `RegisterSession()`, `UpdateSessionState()`, `ReconcileSessions()`, scan helpers.
- **Observer** (`internal/observe`) — `OnSessionStopped` passes StopReason to global DB update.
- **API contract** (`internal/api/contract`) — `SessionResponse` includes stop_reason/stop_detail. Conversion in `internal/api/core/conversions.go`.
- **Session query** (`internal/session/query.go`) — `sessionInfoFromMeta()` maps StopReason from meta.
- **Config** (`internal/config`) — New `[session.limits]` section with `timeout` field. `LoopGuardConfig` deferred to Phase 2.
- **Daemon** (`internal/daemon`) — `Shutdown()` sets `CauseShutdown` on sessions before stopping them. `boot.go` wiring unchanged in Phase 1.
- **Hooks** (`internal/hooks`) — Seams prepared for `session.pre_resume` and `session.post_resume`. Payloads extended with `PreviousStopReason` and `ResumeFromCrash` fields when hooks land.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/store/types.go` | modified | Add `StopReason` type, constants, validation. Add fields to `SessionMeta`. Low risk. | Define type, update Validate() |
| `internal/session/session.go` | modified | Add `StopCause`, `stopReason`, `stopDetail` fields. Update `Info()`, `Meta()`. Low risk. | Add fields, update snapshot methods |
| `internal/session/stop_cause.go` | new | `StopCause` enum. Low risk. | Simple type + constants |
| `internal/session/manager_lifecycle.go` | modified | `classifyStopReason()` in `finalizeStopped()`. Repair pipeline in `Resume()`. `Stop()` accepts/propagates cause. Medium risk — core lifecycle. | Classification logic, validation pipeline, cause propagation |
| `internal/store/globaldb/` | modified | Add columns, update `RegisterSession`, `UpdateSessionState`, `ReconcileSessions`, scan helpers, migration SQL. Low risk. | Schema + query updates |
| `internal/session/query.go` | modified | `sessionInfoFromMeta()` maps StopReason. Low risk. | Add field mapping |
| `internal/api/contract/contract.go` | modified | Add stop_reason/stop_detail to session response types. Low risk. | Add fields |
| `internal/api/core/conversions.go` | modified | Include StopReason in session info conversion. Low risk. | Update conversion |
| `internal/observe/observer.go` | modified | Pass StopReason in `OnSessionStopped` → global DB. Low risk. | Pass through fields |
| `internal/config/config.go` | modified | Add `SessionLimitsConfig` with `timeout`. Low risk. | Add struct, defaults, TOML parsing |
| `internal/config/merge.go` | modified | Merge session limits config. Low risk. | Add merge logic |
| `internal/daemon/daemon.go` | modified | Set `CauseShutdown` on sessions during shutdown. Low risk. | Propagate cause |

## Testing Approach

### Unit Tests

**StopReason validation** (`internal/store/types_test.go`):
- All 10 constants pass `ValidStopReason()`
- Empty string and arbitrary strings fail
- SessionMeta.Validate() rejects invalid StopReason values

**Stop classification** (`manager_lifecycle_test.go`):
- Table-driven: (StopCause, waitErr, detail) → expected (StopReason, StopDetail)
- `CauseShutdown` → always `StopShutdown` regardless of waitErr
- `CauseUserRequested` → `StopUserCanceled`
- `CauseUserRequested` + detail "max_iterations" → `StopMaxIterations`
- `CauseProcessExited` + waitErr → `StopAgentCrashed`
- `CauseProcessExited` + nil waitErr → `StopError`
- `CauseCompleted` → `StopCompleted`
- `CauseHookDenied` → `StopHookStopped`
- Precedence: shutdown wins over all other signals

**Resume repair** (`manager_lifecycle_test.go`):
- Crash classification: meta.State="active" → crashed, "stopping" → crashed, "starting" → error, "stopped" → preserved
- Missing workspace dir → descriptive error with path
- Missing agent definition → descriptive error with agent name
- Missing/zero-size event store → descriptive error
- Invalid meta fields → descriptive error per field
- Multiple failures: all checks run independently, all errors collected
- Crashed session: StopReason written to meta.json on disk

**StopCause propagation** (`manager_lifecycle_test.go`):
- `Stop()` sets `CauseUserRequested`
- Daemon shutdown sets `CauseShutdown`
- Process clean exit sets `CauseCompleted`
- Process unexpected exit sets `CauseProcessExited`

### Integration Tests

**Stop reason end-to-end** (`manager_integration_test.go`):
- Create → Stop explicitly → verify `StopUserCanceled` in meta JSON, global DB, API response
- Create → kill subprocess → verify `StopAgentCrashed` in meta, global DB, API
- Create → daemon shutdown → verify `StopShutdown`

**Resume after crash** (`manager_integration_test.go`):
- Create → write meta with State="active" (simulate crash) → Resume → verify crash classified, StopReason set in meta
- Create → delete workspace dir → Resume → verify descriptive error returned
- Create → remove agent from config → Resume → verify descriptive error
- Create → truncate event store to 0 bytes → Resume → verify descriptive error
- Create → crash → Resume → verify session activates successfully after classification

**API propagation** (`httpapi_integration_test.go`):
- Stop session → GET /api/sessions/:id → verify stop_reason and stop_detail in JSON response
- List sessions → verify stopped sessions include stop_reason

## Development Sequencing

### Build Order

1. **StopReason type in `internal/store`** — Define type, constants, `ValidStopReason()`. Add `StopReason` and `StopDetail` fields to `SessionMeta`. Update `Validate()`. — no dependencies

2. **StopCause type in `internal/session`** — Define `StopCause` enum. Add `stopCause`, `stopReason`, `stopDetail` fields to `Session`. Update `Info()`, `Meta()`. — depends on step 1

3. **Global DB schema** — Add columns to sessions table. Update `RegisterSession()`, `UpdateSessionState()`, `ReconcileSessions()`, scan helpers. Migration SQL. — depends on step 1

4. **Stop reason classification** — Implement `classifyStopReason()`. Wire into `finalizeStopped()`. Propagate `StopCause` through `Stop()`, `handleProcessExit()`, `daemon.Shutdown()`. — depends on steps 1, 2, 3

5. **Resume repair pipeline** — Implement `classifyPreviousStop()`, `validateInfrastructure()`. Insert into `Resume()`. Prepare hook seams (no-op functions). — depends on steps 1, 4

6. **API and query propagation** — Update `sessionInfoFromMeta()` in `query.go`. Update contract types. Update conversions. Update Observer. — depends on steps 3, 4

7. **Config extension** — Add `SessionLimitsConfig` with `timeout` field. TOML parsing, merge logic, defaults. — no dependencies (can parallel with steps 1-6)

8. **Full verification** — Integration tests, `make verify` — depends on all previous steps

### Technical Dependencies

- **No hooks dependency** — Phase 1 ships without the hooks platform. Hook seams are prepared as no-op function calls.
- **Existing session lifecycle** — `finalizeStopped()`, `Resume()`, `Stop()`, `Session`, `SessionMeta`
- **Existing global DB** — sessions table, registration/update/reconcile functions
- **Existing config** — TOML parsing, merge infrastructure

## Phase 2: Loop/Recursion Guards (Deferred)

Phase 2 is explicitly deferred until:
1. The hooks platform is fully wired (tasks 8-12 of hooks techspec)
2. Real session telemetry shows loop patterns and frequency
3. StopReason data from Phase 1 informs which guard types are needed

### Phase 2 Design Direction (from council + Codex review)

- **Hook point**: `tool.pre_call` (sync, deny-capable) — NOT `tool.post_call` (async, observation-only). The hooks platform already supports deny on `tool.pre_call`. This enables pre-execution blocking.
- **Architecture**: Decompose into sensor (evidence accumulation on `tool.post_call` async) + actuator (policy enforcement on `tool.pre_call` sync deny). Two components, not a monolithic LoopGuard.
- **Minimal guard first**: A simple `max_turns` counter as a native hook on `turn.start` (~30 lines) should ship as part of hooks task 10 (turn dispatch integration), before the full LoopGuard.
- **Full LoopGuard later**: SHA-256 hashing, same-args detection, outcome-aware detection, ping-pong patterns — only when production data justifies the complexity.
- **Package placement**: Separate package (`internal/guard` or similar) with interface-based injection, not embedded in `internal/session`.
- **Config**: Start global-only. Design structs so per-agent overrides can be added as a backward-compatible extension.

### Phase 2 Stop Reasons (Pre-wired)

`StopMaxIterations`, `StopLoopDetected`, and `StopBudgetExceeded` are defined in Phase 1's enum but not yet produced by any code path. They are reserved for Phase 2 guards to use. The classification logic in `classifyStopReason()` already handles them via the `detail` string on `CauseUserRequested`.

## Monitoring and Observability

- **Metrics**
  - `session.stop_reason` counter by reason — distribution of how sessions end
  - `session.resume.crash_recovered` counter — sessions resumed after crash
  - `session.resume.validation_failed` counter by check type

- **Structured logs**
  - `session.stopped` — includes `stop_reason`, `stop_detail`, `session_id`, `agent_name`, `duration`
  - `session.resume.crash_classified` — includes previous state, classified reason
  - `session.resume.validation_failed` — includes check name, error details
  - `session.resume.succeeded` — includes session_id, resume_from_crash

- **Alerting thresholds**
  - High crash rate (>10% of sessions stop with `agent_crashed` in a window)
  - Resume validation failure rate (>20% — indicates environment instability)

## Technical Considerations

### Key Decisions

- `StopReason` type lives in `internal/store`, not `internal/session`, to avoid import cycles between packages that both need the type. (ADR-001, updated)
- Stop classification uses an explicit `StopCause` signal, not `ctx.Err()` inference. Each stop initiation point sets the cause explicitly. (Council + Codex feedback)
- Resume repair validates infrastructure only (workspace, agent def, event store). ACP message repair is the agent's responsibility. (ADR-003)
- Loop/recursion guards deferred to Phase 2 pending hooks platform completion and production data. The correct hook point is `tool.pre_call` (sync deny), not `tool.post_call` (async observe). (Council + Codex review)
- All resilience config is global-only. Per-agent overrides deferred. (ADR-004)
- Phase 1 has zero hooks dependency. Hook seams prepared but not wired.

### Known Risks

- **Incomplete stop classification at alpha**: Some `StopCause` values (`CauseHookDenied`) won't be produced until hooks are wired. These paths will default to `StopError` until then. This is acceptable and self-correcting.
- **Schema migration**: Adding columns to global DB. Greenfield alpha — delete and recreate DB if needed.
- **No loop protection in Phase 1**: ACP agents provide their own guards. The gap is agents without built-in guards (early third-party ACP agents). Mitigation: Phase 2 minimal guard ships with hooks task 10.

## Architecture Decision Records

- [ADR-001: Canonical StopReason Enum on SessionMeta](adrs/adr-001.md) — Single ~10-value enum persisted on SessionMeta, classified in finalizeStopped(). Type lives in `internal/store` to avoid import cycles.
- [ADR-003: Infrastructure-Level Repair on Resume](adrs/adr-003.md) — Validation pipeline in Resume() checking workspace, agent def, event store, and meta consistency
- [ADR-004: Global-Only Configuration for Resilience Limits](adrs/adr-004.md) — All limits in agh.toml, no per-agent or per-session overrides
- [ADR-005: Defer Loop Guards to Phase 2](adrs/adr-005.md) — Full LoopGuard deferred until hooks platform complete and production data available. Minimal guard ships with hooks task 10 on `tool.pre_call` sync.
