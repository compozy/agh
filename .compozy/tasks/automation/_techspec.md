# TechSpec: Automation System — Schedules and Triggers

## Executive Summary

AGH gains a built-in automation system that enables both time-based scheduling (cron, interval, one-shot) and event-driven triggers (session events, webhook, memory consolidation, hook events). The system lives in `internal/automation/`, boots as a daemon `Server`, and shares a unified dispatch mechanism that creates agent sessions with configured prompts. Jobs are defined via TOML config (declarative, version-controlled) and API/CLI (dynamic, agent-managed), both persisted in SQLite. gocron v2 drives the in-process scheduling runtime. Extensions can observe and manage the automation engine via Host API methods once the extension system (P1) is complete.

**Primary trade-off**: Building schedules and triggers together in one package increases initial scope but prevents duplication of storage, API, CLI, and UI layers that both subsystems share. The unified `Dispatcher` ensures both activation paths produce identical session creation behavior.

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        daemon/                              │
│                                                             │
│  ┌──────────────┐   ┌──────────────┐   ┌───────────────┐   │
│  │   config/     │   │   session/   │   │   observe/    │   │
│  │ Automation    │──▶│   Manager    │◀──│   Observer    │   │
│  │ Config (TOML) │   │              │   │               │   │
│  └──────┬───────┘   └──────▲───────┘   └───────▲───────┘   │
│         │                  │                    │           │
│  ┌──────▼──────────────────┼────────────────────┼───────┐   │
│  │              internal/automation/                      │   │
│  │                                                       │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  │   │
│  │  │  Scheduler   │  │  Trigger     │  │  Dispatcher  │  │   │
│  │  │  (gocron v2) │  │  Engine      │  │  (sessions)  │  │   │
│  │  │              │  │              │  │              │  │   │
│  │  │ cron/interval│  │ session evts │  │ Create sess  │  │   │
│  │  │ one-shot     │  │ webhook      │  │ Track runs   │  │   │
│  │  │              │  │ memory evts  │  │ Record hist  │  │   │
│  │  │              │  │ hook evts    │  │              │  │   │
│  │  └──────┬───────┘  └──────┬───────┘  └──────▲───────┘  │   │
│  │         │                 │                 │          │   │
│  │         └────── dispatch ─┴─────── dispatch ┘          │   │
│  │                                                       │   │
│  │  ┌─────────────┐  ┌──────────────┐                    │   │
│  │  │  Store       │  │  Manager     │                    │   │
│  │  │  (SQLite)    │  │  (Server)    │                    │   │
│  │  │              │  │  Boot/Stop   │                    │   │
│  │  │ jobs, runs   │  │  Sync TOML   │                    │   │
│  │  │ triggers     │  │  Wire deps   │                    │   │
│  │  └──────────────┘  └──────────────┘                    │   │
│  └───────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌──────────────┐   ┌──────────────┐   ┌───────────────┐   │
│  │  httpapi/     │   │  udsapi/     │   │   cli/        │   │
│  │  REST + SSE   │   │  UDS IPC     │   │   Cobra       │   │
│  └──────────────┘   └──────────────┘   └───────────────┘   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                     web/ (React SPA)                  │   │
│  │  /automation — list + detail + create/edit            │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Data flow**:
1. Daemon boots → loads TOML automation config → syncs to SQLite → registers with gocron / trigger engine
2. User creates job via CLI/API → persists to SQLite → registers with gocron / trigger engine
3. Schedule fires (time) or trigger activates (event) → Dispatcher creates session via `session.Manager`
4. Session lifecycle events flow back through Observer → automation store records run history
5. Web UI queries automation API for job/trigger list, run history, next-run times

## Implementation Design

### Core Interfaces

```go
// internal/automation/automation.go

// Manager is the automation system entry point. Implements daemon.Server.
type Manager struct {
    scheduler  *Scheduler
    triggers   *TriggerEngine
    dispatcher *Dispatcher
    store      *Store
    logger     *slog.Logger
}

func New(opts ...Option) (*Manager, error)
func (m *Manager) Start(ctx context.Context) error
func (m *Manager) Shutdown(ctx context.Context) error
func (m *Manager) Jobs(ctx context.Context) ([]Job, error)
func (m *Manager) Triggers(ctx context.Context) ([]Trigger, error)
func (m *Manager) CreateJob(ctx context.Context, req CreateJobRequest) (*Job, error)
func (m *Manager) UpdateJob(ctx context.Context, id string, req UpdateJobRequest) (*Job, error)
func (m *Manager) DeleteJob(ctx context.Context, id string) error
func (m *Manager) TriggerJob(ctx context.Context, id string, payload map[string]any) (*Run, error)
func (m *Manager) CreateTrigger(ctx context.Context, req CreateTriggerRequest) (*Trigger, error)
func (m *Manager) DeleteTrigger(ctx context.Context, id string) error
func (m *Manager) Runs(ctx context.Context, query RunQuery) ([]Run, error)
```

```go
// internal/automation/dispatch.go

// Dispatcher creates agent sessions from automation jobs.
type Dispatcher struct {
    sessions SessionCreator
    store    *Store
    logger   *slog.Logger
}

// SessionCreator is the subset of session.Manager the dispatcher needs.
type SessionCreator interface {
    Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
    Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
}

func (d *Dispatcher) Dispatch(ctx context.Context, req DispatchRequest) (*Run, error)
```

```go
// internal/automation/trigger.go

// TriggerSource represents a kind of event that can activate a trigger.
type TriggerSource interface {
    Kind() string
    Subscribe(ctx context.Context, ch chan<- Activation) error
    Unsubscribe(ctx context.Context) error
}

// Built-in sources: SessionEventSource, WebhookSource,
// MemoryEventSource, HookEventSource
```

### Data Models

```go
// internal/automation/job.go

type Job struct {
    ID          string          `json:"id"`
    Name        string          `json:"name"`
    AgentName   string          `json:"agent_name"`
    WorkspaceID string          `json:"workspace_id"`
    Prompt      string          `json:"prompt"`
    Schedule    *ScheduleSpec   `json:"schedule,omitempty"`
    Enabled     bool            `json:"enabled"`
    Retry       RetryConfig     `json:"retry"`
    FireLimit   FireLimitConfig `json:"fire_limit"`
    Source      JobSource       `json:"source"` // "config" or "dynamic"
    CreatedAt   time.Time       `json:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at"`
}

type ScheduleSpec struct {
    Mode     string `json:"mode"` // "cron", "every", "at"
    Expr     string `json:"expr,omitempty"`     // cron: "0 9 * * *"
    Interval string `json:"interval,omitempty"` // every: "30m"
    Time     string `json:"time,omitempty"`     // at: "2026-04-15T15:00:00Z"
}

type Trigger struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    AgentName   string            `json:"agent_name"`
    WorkspaceID string            `json:"workspace_id"`
    Prompt      string            `json:"prompt"` // supports {{.payload}} template
    Event       string            `json:"event"`  // "session.stopped", "webhook", etc.
    Filter      map[string]string `json:"filter,omitempty"` // e.g. {"agent": "researcher", "stop_reason": "completed"}
    Enabled     bool              `json:"enabled"`
    Retry       RetryConfig       `json:"retry"`
    FireLimit   FireLimitConfig   `json:"fire_limit"`
    Source      JobSource         `json:"source"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type RetryConfig struct {
    Strategy   string `json:"strategy" toml:"strategy"`       // "none" (default), "backoff"
    MaxRetries int    `json:"max_retries" toml:"max_retries"` // default: 3
    BaseDelay  string `json:"base_delay" toml:"base_delay"`   // default: "2s"
}

type FireLimitConfig struct {
    Max    int    `json:"max" toml:"max"`       // default: 12
    Window string `json:"window" toml:"window"` // default: "1h"
}

type Run struct {
    ID        string     `json:"id"`
    JobID     string     `json:"job_id,omitempty"`
    TriggerID string     `json:"trigger_id,omitempty"`
    SessionID string     `json:"session_id,omitempty"`
    Status    RunStatus  `json:"status"` // scheduled, running, completed, failed, cancelled
    Attempt   int        `json:"attempt"`
    StartedAt *time.Time `json:"started_at,omitempty"`
    EndedAt   *time.Time `json:"ended_at,omitempty"`
    Error     string     `json:"error,omitempty"`
}

type RunStatus string
const (
    RunScheduled RunStatus = "scheduled"
    RunRunning   RunStatus = "running"
    RunCompleted RunStatus = "completed"
    RunFailed    RunStatus = "failed"
    RunCancelled RunStatus = "cancelled"
)

type JobSource string
const (
    JobSourceConfig  JobSource = "config"
    JobSourceDynamic JobSource = "dynamic"
)
```

### Database Schema

```sql
CREATE TABLE automation_jobs (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL UNIQUE,
    agent_name   TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    prompt       TEXT NOT NULL,
    schedule     TEXT,          -- JSON: ScheduleSpec
    enabled      BOOLEAN NOT NULL DEFAULT 1,
    retry        TEXT NOT NULL,  -- JSON: RetryConfig
    fire_limit   TEXT NOT NULL,  -- JSON: FireLimitConfig
    source       TEXT NOT NULL DEFAULT 'dynamic',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE automation_triggers (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL UNIQUE,
    agent_name   TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    prompt       TEXT NOT NULL,
    event        TEXT NOT NULL,
    filter       TEXT,          -- JSON: map[string]string
    enabled      BOOLEAN NOT NULL DEFAULT 1,
    retry        TEXT NOT NULL,
    fire_limit   TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'dynamic',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE TABLE automation_runs (
    id         TEXT PRIMARY KEY,
    job_id     TEXT,
    trigger_id TEXT,
    session_id TEXT,
    status     TEXT NOT NULL,
    attempt    INTEGER NOT NULL DEFAULT 1,
    started_at TEXT,
    ended_at   TEXT,
    error      TEXT,
    FOREIGN KEY(job_id) REFERENCES automation_jobs(id) ON DELETE SET NULL,
    FOREIGN KEY(trigger_id) REFERENCES automation_triggers(id) ON DELETE SET NULL
);

CREATE INDEX idx_automation_jobs_enabled ON automation_jobs(enabled);
CREATE INDEX idx_automation_triggers_enabled ON automation_triggers(enabled);
CREATE INDEX idx_automation_triggers_event ON automation_triggers(event);
CREATE INDEX idx_automation_runs_job ON automation_runs(job_id);
CREATE INDEX idx_automation_runs_trigger ON automation_runs(trigger_id);
CREATE INDEX idx_automation_runs_status ON automation_runs(status);
CREATE INDEX idx_automation_runs_started ON automation_runs(started_at);
```

### TOML Config Schema

```toml
[automation]
enabled = true
timezone = "UTC"                    # default timezone for cron expressions
max_concurrent_jobs = 5             # global concurrent execution limit
default_fire_limit = { max = 12, window = "1h" }

[[automation.jobs]]
name = "daily-report"
schedule = { mode = "cron", expr = "0 9 * * *" }
agent = "researcher"
workspace = "/home/user/project"
prompt = "Generate daily AI news summary"
retry = { strategy = "none" }

[[automation.jobs]]
name = "health-check"
schedule = { mode = "every", interval = "30m" }
agent = "monitor"
prompt = "Check system health and report anomalies"
retry = { strategy = "backoff", max_retries = 3, base_delay = "2s" }

[[automation.triggers]]
name = "post-research"
event = "session.stopped"
filter = { agent = "researcher", stop_reason = "completed" }
agent = "summarizer"
prompt = "Summarize findings from session {{.session_id}}"

[[automation.triggers]]
name = "on-deploy"
event = "webhook"
agent = "deploy-reviewer"
prompt = "Review deployment: {{.payload}}"
```

### API Endpoints

#### Jobs (Scheduled)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/automation/jobs` | List all jobs |
| `POST` | `/api/automation/jobs` | Create a new job |
| `GET` | `/api/automation/jobs/:id` | Get job details + next run time |
| `PATCH` | `/api/automation/jobs/:id` | Update job (name, schedule, enabled, retry, etc.) |
| `DELETE` | `/api/automation/jobs/:id` | Delete a job (config-sourced jobs cannot be deleted, only disabled) |
| `POST` | `/api/automation/jobs/:id/trigger` | Force immediate execution |
| `GET` | `/api/automation/jobs/:id/runs` | List execution history for job |

#### Triggers (Event-Driven)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/automation/triggers` | List all triggers |
| `POST` | `/api/automation/triggers` | Create a new trigger |
| `GET` | `/api/automation/triggers/:id` | Get trigger details |
| `PATCH` | `/api/automation/triggers/:id` | Update trigger |
| `DELETE` | `/api/automation/triggers/:id` | Delete a trigger |
| `GET` | `/api/automation/triggers/:id/runs` | List execution history for trigger |

#### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/webhooks/:trigger-name` | External webhook delivery endpoint |

#### Runs (Shared)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/automation/runs` | List all runs (filterable by job_id, trigger_id, status) |
| `GET` | `/api/automation/runs/:id` | Get run details |

### CLI Commands

```
agh automation jobs                     # List all scheduled jobs
agh automation jobs create              # Create a job (interactive or flags)
  --name <name>
  --schedule <cron-expr|every:30m|at:2026-04-15T15:00>
  --agent <agent-name>
  --workspace <path>
  --prompt <prompt>
  --retry <none|backoff:3:2s>
agh automation jobs get <id>            # Get job details
agh automation jobs update <id>         # Update job
agh automation jobs delete <id>         # Delete job
agh automation jobs trigger <id>        # Force immediate run
agh automation jobs history <id>        # Show execution history

agh automation triggers                 # List all triggers
agh automation triggers create          # Create a trigger
  --name <name>
  --event <session.stopped|webhook|memory.consolidated|hook.*>
  --filter agent=researcher,stop_reason=completed
  --agent <agent-name>
  --prompt <prompt-template>
agh automation triggers get <id>        # Get trigger details
agh automation triggers delete <id>     # Delete trigger
agh automation triggers history <id>    # Show execution history

agh automation runs                     # List recent runs
agh automation runs get <id>            # Get run details
```

## Integration Points

### Daemon Boot Integration

New boot phase `bootAutomation` after `bootHooks`, before `bootServers`:

```go
func (d *Daemon) bootAutomation(ctx context.Context, state *bootState) error {
    if !state.cfg.Automation.Enabled {
        d.logger.Info("automation disabled")
        return nil
    }

    store, err := automation.NewStore(state.registry.DB())
    if err != nil {
        return fmt.Errorf("automation store: %w", err)
    }

    mgr, err := automation.New(
        automation.WithStore(store),
        automation.WithSessions(state.sessions),
        automation.WithObserver(state.observer),
        automation.WithHookRunner(state.hookRunner),
        automation.WithConfig(state.cfg.Automation),
        automation.WithLogger(d.logger.With("component", "automation")),
    )
    if err != nil {
        return fmt.Errorf("automation manager: %w", err)
    }

    state.automation = mgr
    state.cleanup.add(func(ctx context.Context) error {
        return mgr.Shutdown(ctx)
    })

    return mgr.Start(ctx)
}
```

### Session Notifier Integration

The trigger engine subscribes to session lifecycle events via `session.Notifier`:

- `OnSessionCreated` → matches triggers with `event = "session.created"`
- `OnSessionStopped` → matches triggers with `event = "session.stopped"` + filter by agent, stop_reason

### Memory Consolidation Integration

Subscribe to dream consolidation events via callback on `memory.Store` or `consolidation.Runtime`:

- `OnConsolidationComplete` → matches triggers with `event = "memory.consolidated"`

### Hook System Integration

Subscribe to hook completion events:

- After any hook runs, emit an internal event that triggers matching `event = "hook.*.completed"` triggers

### Webhook HTTP Integration

Register webhook routes in `httpapi/routes.go`:

- `POST /api/webhooks/:trigger-name` — validates trigger exists, enabled, event type is "webhook", then dispatches

### Observer Integration

Record automation events as `EventSummary` entries:

- `automation.job_fired` — schedule fired, dispatching session
- `automation.trigger_fired` — trigger activated, dispatching session
- `automation.run_completed` — run finished (success)
- `automation.run_failed` — run finished (failure)

### Extension Integration (Future P1)

When the extension Host API is complete, expose automation methods:

- `automation/jobs` — CRUD operations on jobs
- `automation/triggers` — CRUD operations on triggers
- `automation/runs` — query run history

Extensions can observe automation events via hook events already emitted.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/automation/` | New | Core automation package | Implement from scratch |
| `internal/config/` | Modified | Add `AutomationConfig` struct and validation | Low risk — additive |
| `internal/daemon/boot.go` | Modified | Add `bootAutomation` phase | Low risk — new phase between hooks and servers |
| `internal/daemon/daemon.go` | Modified | Add automation field to `RuntimeDeps` | Low risk — additive |
| `internal/store/globaldb/` | Modified | Add automation tables to schema | Low risk — new tables |
| `internal/api/contract/` | Modified | Add automation request/response types | Low risk — additive |
| `internal/api/httpapi/` | Modified | Add automation + webhook route handlers | Medium risk — new route groups |
| `internal/api/core/` | Modified | Add `AutomationManager` interface | Low risk — additive |
| `internal/cli/` | Modified | Add `automation` command group | Low risk — new subcommand |
| `internal/session/` | None | No changes — dispatcher uses existing `Manager.Create` | No action |
| `internal/observe/` | Minor | Record automation event summaries | Low risk — new event types |
| `web/` | Modified | New `/automation` page, sidebar entry, components | Medium risk — new feature page |
| `openapi/agh.json` | Modified | Add automation endpoints to OpenAPI spec | Low risk — additive |
| `go.mod` | Modified | Add `go-co-op/gocron/v2` dependency | Low risk |

## Testing Approach

### Unit Tests

- **Scheduler**: Register cron/interval/one-shot jobs, verify next-run calculation using `clockwork.FakeClock`, verify singleton mode prevents overlap, verify fire limits, verify retry strategies
- **Trigger engine**: Emit mock session/memory/hook events, verify trigger matching with filters, verify fire limits, verify prompt template rendering with `{{.payload}}`
- **Dispatcher**: Mock `SessionCreator`, verify session creation with correct `CreateOpts`, verify run recording, verify retry on failure
- **Store**: Table-driven CRUD tests with `t.TempDir()` SQLite, verify constraint enforcement (unique names), verify query filtering
- **Config**: Parse TOML automation section, validate schedule expressions, validate retry config, verify config-to-SQLite sync logic

### Integration Tests

- **Daemon boot → schedule fires → session created**: Full lifecycle test with real scheduler (fast cron expression like `@every 1s`), verify session appears in session list
- **Event → trigger → session**: Create a trigger for `session.stopped`, complete a session, verify trigger dispatches new session
- **Webhook → trigger → session**: POST to webhook endpoint, verify trigger fires and session is created
- **TOML sync**: Boot with TOML jobs, verify they appear in SQLite, modify via API, reboot, verify TOML jobs re-synced
- **Graceful shutdown**: Start jobs, initiate shutdown, verify running jobs receive context cancellation

## Development Sequencing

### Build Order

1. **Config + Store** (no dependencies) — `AutomationConfig` struct, TOML parsing, validation, SQLite schema, CRUD queries
2. **Job + Trigger types** (depends on 1) — domain types, `ScheduleSpec`, `RetryConfig`, `FireLimitConfig`, serialization
3. **Dispatcher** (depends on 2) — `SessionCreator` interface, dispatch logic, run recording, retry engine, fire limit tracking
4. **Scheduler** (depends on 2, 3) — gocron v2 wrapper, register/unregister jobs, lifecycle hooks wiring to dispatcher
5. **Trigger engine** (depends on 2, 3) — `TriggerSource` interface, session/webhook/memory/hook source implementations, event matching, filter evaluation
6. **Manager** (depends on 3, 4, 5) — compose scheduler + trigger engine + dispatcher + store, TOML sync, `Server` interface
7. **Daemon integration** (depends on 6) — `bootAutomation` phase, wire `RuntimeDeps`, shutdown ordering
8. **API contract + handlers** (depends on 6) — request/response types, HTTP handlers, webhook endpoint, route registration
9. **CLI commands** (depends on 6) — `agh automation` subcommand tree, output formats
10. **Web UI** (depends on 8) — `/automation` route, list/detail components, create/edit forms, run history
11. **OpenAPI spec** (depends on 8) — generate types, update `agh.json`

### Technical Dependencies

- **gocron v2**: `go get github.com/go-co-op/gocron/v2` — no blocking issues
- **Session Manager**: existing, no changes needed
- **Global DB**: existing, additive schema changes only
- **Observer**: existing, additive event types only

## Monitoring and Observability

### Metrics (via Observer event summaries)

| Metric | Type | Description |
|--------|------|-------------|
| `automation.jobs.total` | Gauge | Total registered jobs (enabled/disabled) |
| `automation.triggers.total` | Gauge | Total registered triggers (enabled/disabled) |
| `automation.runs.total` | Counter | Total runs by status (completed/failed/cancelled) |
| `automation.runs.duration_ms` | Histogram | Run duration from dispatch to session completion |
| `automation.fire_limit.rejected` | Counter | Dispatch attempts rejected by fire limits |
| `automation.retry.attempts` | Counter | Retry attempts by job/trigger |

### Log Events

| Event | Level | Fields |
|-------|-------|--------|
| Job fired | INFO | `job_id`, `job_name`, `agent`, `schedule` |
| Trigger activated | INFO | `trigger_id`, `trigger_name`, `event`, `agent` |
| Run completed | INFO | `run_id`, `job_id`/`trigger_id`, `session_id`, `duration_ms` |
| Run failed | WARN | `run_id`, `job_id`/`trigger_id`, `error`, `attempt` |
| Fire limit hit | WARN | `job_id`/`trigger_id`, `name`, `limit`, `window` |
| Retry scheduled | INFO | `run_id`, `attempt`, `delay` |
| TOML sync | INFO | `jobs_synced`, `triggers_synced`, `jobs_removed` |
| Webhook received | INFO | `trigger_name`, `remote_addr`, `payload_size` |
| Scheduler started | INFO | `jobs_loaded`, `triggers_loaded` |
| Scheduler shutdown | INFO | `running_jobs_cancelled`, `shutdown_duration_ms` |

### Health Integration

Add automation status to `GET /api/observe/health`:

```json
{
  "automation": {
    "enabled": true,
    "jobs": { "total": 5, "enabled": 4 },
    "triggers": { "total": 3, "enabled": 3 },
    "scheduler_running": true,
    "next_fire": "2026-04-11T09:00:00Z"
  }
}
```

## Technical Considerations

### Key Decisions

1. **gocron v2 over robfig/cron v3** — context propagation to jobs is a hard requirement; robfig/cron's `func()` signature cannot propagate cancellation. gocron wraps robfig's parser while adding the lifecycle management AGH needs. (ADR-003)

2. **Unified automation package** — schedules and triggers share dispatch, storage, API, and CLI layers. Separating them would duplicate ~60% of the code. (ADR-002)

3. **Built-in with extension hooks** — extension system (P1) is incomplete. Building in-process now with Host API exposure later avoids blocking on unfinished infrastructure. (ADR-001)

4. **Configurable retry per job** — agent sessions are expensive. Default `none` prevents cost amplification. Jobs with transient failure modes opt into `backoff`. Fire limits provide a global safety net. (ADR-004)

5. **No missed-job backfill** — if the daemon is down when a cron fires, the job is skipped. Running stale jobs hours late is usually wrong for LLM-powered agents. Record the miss, let the user decide. (Aligned with OpenFang's deliberate choice.)

6. **TOML jobs are source-of-truth** — on daemon boot, TOML-defined jobs sync to SQLite (create if missing, update if changed). Dynamic jobs (API/CLI) coexist in SQLite. Config-sourced jobs cannot be deleted via API (only disabled). This prevents config drift.

7. **Prompt templates** — trigger prompts support Go `text/template` syntax for injecting event payload data (e.g., `{{.session_id}}`, `{{.payload}}`). Schedule prompts are static strings.

### Known Risks

1. **gocron v2 maintenance** — if abandoned, the library is thin enough to fork or replace. AGH wraps it behind `internal/automation/schedule.go`, isolating the dependency. Likelihood: low.

2. **Trigger fan-out performance** — many triggers with complex filters on high-frequency events (e.g., `session.created`) could cause latency. Mitigation: fire limits, efficient filter matching (exact string match, not regex in v1), async dispatch.

3. **TOML ↔ SQLite sync conflicts** — user modifies a TOML job via API, then reboots. TOML wins (source-of-truth), overwriting API changes. Mitigation: log a warning on sync when TOML overrides a modified dynamic state. Config-sourced jobs are marked `source: "config"` to make this visible.

4. **Webhook security** — unauthenticated webhook endpoints could be abused. Mitigation: fire limits prevent runaway execution. Future: add HMAC signature verification for webhook payloads (not in v1).

## Architecture Decision Records

- [ADR-001: Built-In Daemon Component with Extension Integration Points](adrs/adr-001.md) — Automation lives in `internal/automation/` as a daemon Server, not as an extension, with Host API exposure planned for when P1 completes.
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Single package handles both time-based scheduling and event-driven triggers through a shared dispatch mechanism.
- [ADR-003: gocron v2 as In-Process Scheduling Runtime](adrs/adr-003.md) — gocron v2 chosen over robfig/cron v3 for context propagation, lifecycle hooks, singleton mode, and active maintenance.
- [ADR-004: Configurable Per-Job Retry with Fire Limits](adrs/adr-004.md) — Per-job retry strategy (none/backoff) with global fire limits to prevent runaway execution.
