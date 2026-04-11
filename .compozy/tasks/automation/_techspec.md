# TechSpec: Automation System — Schedules and Triggers

## Executive Summary

AGH gains a built-in automation system that enables both time-based scheduling (cron, interval, one-shot) and event-driven triggers (session events, webhook, memory consolidation, hook events). The system lives in `internal/automation/`, boots as a daemon `Server`, and shares a unified dispatch mechanism that creates agent sessions with configured prompts. Automation definitions support both `global` and `workspace` scope. Jobs are defined via TOML config (declarative, version-controlled) and API/CLI (dynamic, agent-managed), both persisted in SQLite. Config-defined jobs and triggers may be overlaid at runtime only for `enabled/disabled` operational state. gocron v2 drives the in-process scheduling runtime. Internal trigger ingress reuses the daemon's existing `observer/hooks` boundary, and external webhook triggers are authenticated with per-trigger HMAC. Extensions can observe and manage the automation engine via Host API methods once the extension system (P1) is complete.

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
3. Schedule fires (time) or an internal/external event is normalized into an `ActivationEnvelope`
4. Trigger engine matches envelopes to triggers → Dispatcher applies the global concurrency gate and creates the session via `session.Manager`
5. Runs are recorded in SQLite and reused for run history, fire-limit evaluation, and restart-safe operational state
6. Web UI queries automation API for job/trigger list, run history, next-run times, and webhook endpoints

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

// ActivationEnvelope is the normalized trigger input regardless of source.
type ActivationEnvelope struct {
    Kind        string          `json:"kind"`
    Scope       AutomationScope `json:"scope"`
    WorkspaceID string          `json:"workspace_id,omitempty"`
    Source      string          `json:"source"` // "observer", "hook", "webhook", "extension"
    Data        map[string]any  `json:"data"`
}

// Trigger ingress adapters convert existing daemon/runtime events into
// ActivationEnvelope values. Built-in ingress paths:
// - session + memory lifecycle events via observer/hooks
// - authenticated webhook deliveries via HTTP
// - extension-triggered events via Host API
```

### Data Models

```go
// internal/automation/job.go

type Job struct {
    ID          string          `json:"id"`
    Scope       AutomationScope `json:"scope"`
    Name        string          `json:"name"`
    AgentName   string          `json:"agent_name"`
    WorkspaceID string          `json:"workspace_id,omitempty"`
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
    ID           string            `json:"id"`
    Scope        AutomationScope   `json:"scope"`
    Name         string            `json:"name"`
    AgentName    string            `json:"agent_name"`
    WorkspaceID  string            `json:"workspace_id,omitempty"`
    Prompt       string            `json:"prompt"` // supports strict text/template execution against ActivationEnvelope
    Event        string            `json:"event"`  // "session.stopped", "webhook", etc.
    Filter       map[string]string `json:"filter,omitempty"` // exact-match against allowed ActivationEnvelope field paths
    Enabled      bool              `json:"enabled"`
    Retry        RetryConfig       `json:"retry"`
    FireLimit    FireLimitConfig   `json:"fire_limit"`
    Source       JobSource         `json:"source"`
    WebhookID    string            `json:"webhook_id,omitempty"`
    EndpointSlug string            `json:"endpoint_slug,omitempty"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
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

type AutomationScope string

const (
    AutomationScopeGlobal    AutomationScope = "global"
    AutomationScopeWorkspace AutomationScope = "workspace"
)
```

### Database Schema

```sql
CREATE TABLE automation_jobs (
    id           TEXT PRIMARY KEY,
    scope        TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
    name         TEXT NOT NULL,
    agent_name   TEXT NOT NULL,
    workspace_id TEXT,
    prompt       TEXT NOT NULL,
    schedule     TEXT,          -- JSON: ScheduleSpec
    enabled      BOOLEAN NOT NULL DEFAULT 1, -- definition default / dynamic desired state
    retry        TEXT NOT NULL,  -- JSON: RetryConfig
    fire_limit   TEXT NOT NULL,  -- JSON: FireLimitConfig
    source       TEXT NOT NULL DEFAULT 'dynamic',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    CHECK (
        (scope = 'global' AND workspace_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL)
    )
);

CREATE TABLE automation_triggers (
    id           TEXT PRIMARY KEY,
    scope        TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
    name         TEXT NOT NULL,
    agent_name   TEXT NOT NULL,
    workspace_id TEXT,
    prompt       TEXT NOT NULL,
    event        TEXT NOT NULL,
    filter       TEXT,          -- JSON: map[string]string
    enabled      BOOLEAN NOT NULL DEFAULT 1, -- definition default / dynamic desired state
    retry        TEXT NOT NULL,
    fire_limit   TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT 'dynamic',
    webhook_id   TEXT,
    endpoint_slug TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    CHECK (
        (scope = 'global' AND workspace_id IS NULL) OR
        (scope = 'workspace' AND workspace_id IS NOT NULL)
    )
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

CREATE TABLE automation_job_overlays (
    job_id            TEXT PRIMARY KEY,
    enabled_override  BOOLEAN NOT NULL,
    updated_at        TEXT NOT NULL,
    FOREIGN KEY(job_id) REFERENCES automation_jobs(id) ON DELETE CASCADE
);

CREATE TABLE automation_trigger_overlays (
    trigger_id        TEXT PRIMARY KEY,
    enabled_override  BOOLEAN NOT NULL,
    updated_at        TEXT NOT NULL,
    FOREIGN KEY(trigger_id) REFERENCES automation_triggers(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX uq_automation_jobs_global_name
    ON automation_jobs(name) WHERE scope = 'global';
CREATE UNIQUE INDEX uq_automation_jobs_workspace_name
    ON automation_jobs(workspace_id, name) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uq_automation_triggers_global_name
    ON automation_triggers(name) WHERE scope = 'global';
CREATE UNIQUE INDEX uq_automation_triggers_workspace_name
    ON automation_triggers(workspace_id, name) WHERE scope = 'workspace';
CREATE UNIQUE INDEX uq_automation_triggers_webhook_id
    ON automation_triggers(webhook_id) WHERE webhook_id IS NOT NULL;
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
scope = "workspace"
name = "daily-report"
schedule = { mode = "cron", expr = "0 9 * * *" }
agent = "researcher"
workspace = "/home/user/project"
prompt = "Generate daily AI news summary"
retry = { strategy = "none" }

[[automation.jobs]]
scope = "global"
name = "health-check"
schedule = { mode = "every", interval = "30m" }
agent = "monitor"
prompt = "Check system health and report anomalies"
retry = { strategy = "backoff", max_retries = 3, base_delay = "2s" }

[[automation.triggers]]
scope = "workspace"
name = "post-research"
event = "session.stopped"
workspace = "/home/user/project"
filter = { "data.agent_name" = "researcher", "data.stop_reason" = "completed" }
agent = "summarizer"
prompt = "Summarize findings from session {{ index .Data \"session_id\" }}"

[[automation.triggers]]
scope = "global"
name = "on-deploy"
event = "webhook"
endpoint_slug = "deploy-review"
webhook_secret_env = "AGH_CONFIG_WEBHOOK_SECRET"
agent = "deploy-reviewer"
prompt = "Review deployment payload: {{ index .Data \"payload\" }}"
```

Config-backed webhook triggers must provide `webhook_secret_env` so the daemon can resolve a write-only runtime secret without exposing it through readable trigger definitions.

### API Endpoints

#### Jobs (Scheduled)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/automation/jobs` | List all jobs (filterable by `scope`, `workspace_id`, `source`) |
| `POST` | `/api/automation/jobs` | Create a new job |
| `GET` | `/api/automation/jobs/:id` | Get job details + next run time |
| `PATCH` | `/api/automation/jobs/:id` | Update job; config-sourced jobs accept only operational `enabled` overlay updates |
| `DELETE` | `/api/automation/jobs/:id` | Delete a job (config-sourced jobs cannot be deleted, only disabled) |
| `POST` | `/api/automation/jobs/:id/trigger` | Force immediate execution |
| `GET` | `/api/automation/jobs/:id/runs` | List execution history for job |

#### Triggers (Event-Driven)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/automation/triggers` | List all triggers (filterable by `scope`, `workspace_id`, `event`) |
| `POST` | `/api/automation/triggers` | Create a new trigger |
| `GET` | `/api/automation/triggers/:id` | Get trigger details |
| `PATCH` | `/api/automation/triggers/:id` | Update trigger; config-sourced triggers accept only operational `enabled` overlay updates |
| `DELETE` | `/api/automation/triggers/:id` | Delete a trigger (config-sourced triggers cannot be deleted, only disabled) |
| `GET` | `/api/automation/triggers/:id/runs` | List execution history for trigger |

#### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/webhooks/global/:endpoint` | External webhook delivery endpoint for global webhook triggers |
| `POST` | `/api/webhooks/workspaces/:workspace_id/:endpoint` | External webhook delivery endpoint for workspace-scoped webhook triggers |

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
  --scope <global|workspace>
  --schedule <cron-expr|every:30m|at:2026-04-15T15:00>
  --agent <agent-name>
  --workspace <path-or-id>              # required when --scope=workspace
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
  --scope <global|workspace>
  --event <session.stopped|webhook|memory.consolidated|hook.*>
  --workspace <path-or-id>              # required when --scope=workspace
  --filter data.agent_name=researcher,data.stop_reason=completed
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
func (d *Daemon) bootAutomation(ctx context.Context, state *bootState, cleanup *bootCleanup) error {
    if !state.cfg.Automation.Enabled {
        state.logger.Info("automation disabled")
        return nil
    }

    mgr, err := automation.New(
        automation.WithRegistry(state.registry),
        automation.WithSessions(state.sessions),
        automation.WithObserver(state.observer),
        automation.WithHooks(state.hooks),
        automation.WithConfig(state.cfg.Automation),
        automation.WithLogger(state.logger.With("component", "automation")),
    )
    if err != nil {
        return fmt.Errorf("automation manager: %w", err)
    }

    cleanup.add(func(ctx context.Context) error {
        return mgr.Shutdown(ctx)
    })

    return mgr.Start(ctx)
}
```

### Session Notifier Integration

The trigger engine does **not** add a second direct subscription mechanism to `session/`. Instead, session lifecycle events are consumed from the existing daemon `observer/hooks` boundary and normalized into `ActivationEnvelope` values:

- `session.created` → normalized to `ActivationEnvelope{Kind: "session.created", ...}`
- `session.stopped` → normalized to `ActivationEnvelope{Kind: "session.stopped", ...}` with `data.agent_name`, `data.stop_reason`

### Memory Consolidation Integration

Dream consolidation completion is exposed through the same observer/hooks-facing boundary rather than through a dedicated automation-only callback:

- `memory.consolidated` → normalized to `ActivationEnvelope{Kind: "memory.consolidated", ...}`

### Hook System Integration

Automation consumes hook-derived events from the existing hook runtime and also emits its own automation lifecycle hook points:

- Existing hook completions may be normalized to `ActivationEnvelope{Kind: "hook.<name>.completed", ...}`
- Automation emits `automation.job.pre_fire`, `automation.job.post_fire`, `automation.trigger.pre_fire`, `automation.trigger.post_fire`, `automation.run.completed`, and `automation.run.failed`

### Webhook HTTP Integration

Register webhook routes in `httpapi/routes.go`:

- `POST /api/webhooks/global/:endpoint` — resolves stable `webhook_id` from the endpoint suffix, validates HMAC + timestamp, then normalizes and dispatches
- `POST /api/webhooks/workspaces/:workspace_id/:endpoint` — same behavior for workspace-scoped webhook triggers

### Observer Integration

Record automation events as `EventSummary` entries:

- `automation.job_fired` — schedule fired, dispatching session
- `automation.trigger_fired` — trigger activated, dispatching session
- `automation.run_completed` — run finished (success)
- `automation.run_failed` — run finished (failure)

### Extension Integration

The automation system integrates with the extension architecture at three levels: **Host API methods** (extensions manage automation), **hook events** (extensions observe automation), and **custom trigger sources** (extensions extend automation).

#### 1. Host API Methods (JSON-RPC over stdio)

Extensions with the `automation.read` or `automation.write` security capabilities can call these methods on the daemon:

```json
// Extension → Daemon: List jobs
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "automation/jobs",
  "params": { "scope": "workspace", "workspace_id": "ws_123", "enabled": true }
}
// Response: { "result": [{ "id": "job_1", "name": "daily-report", ... }] }

// Extension → Daemon: Create a job
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "automation/jobs/create",
  "params": {
    "name": "ext-health-check",
    "scope": "workspace",
    "agent_name": "monitor",
    "workspace_id": "ws_123",
    "prompt": "Check extension health",
    "schedule": { "mode": "every", "interval": "5m" },
    "retry": { "strategy": "backoff", "max_retries": 2, "base_delay": "1s" }
  }
}
// Response: { "result": { "id": "job_abc", "name": "ext-health-check", ... } }

// Extension → Daemon: Trigger a job immediately
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "automation/jobs/trigger",
  "params": { "id": "job_abc", "payload": { "reason": "manual" } }
}
// Response: { "result": { "run_id": "run_xyz", "session_id": "sess_456" } }

// Extension → Daemon: Create a trigger
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "automation/triggers/create",
  "params": {
    "name": "on-memory-consolidated",
    "scope": "workspace",
    "agent_name": "knowledge-updater",
    "workspace_id": "ws_123",
    "event": "memory.consolidated",
    "prompt": "Update knowledge base after consolidation"
  }
}

// Extension → Daemon: Query runs
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "automation/runs",
  "params": { "job_id": "job_abc", "status": "failed", "limit": 10 }
}
```

**Full Host API method table:**

| Method | Params | Result | Security Capability |
|--------|--------|--------|---------------------|
| `automation/jobs` | `{scope?, workspace_id?, enabled?}` | `[Job]` | `automation.read` |
| `automation/jobs/get` | `{id}` | `Job` | `automation.read` |
| `automation/jobs/create` | `CreateJobRequest` | `Job` | `automation.write` |
| `automation/jobs/update` | `{id, ...fields}` | `Job` | `automation.write` |
| `automation/jobs/delete` | `{id}` | `{}` | `automation.write` |
| `automation/jobs/trigger` | `{id, payload?}` | `Run` | `automation.write` |
| `automation/jobs/runs` | `{id, limit?, status?}` | `[Run]` | `automation.read` |
| `automation/triggers` | `{scope?, workspace_id?, event?}` | `[Trigger]` | `automation.read` |
| `automation/triggers/get` | `{id}` | `Trigger` | `automation.read` |
| `automation/triggers/create` | `CreateTriggerRequest` | `Trigger` | `automation.write` |
| `automation/triggers/update` | `{id, ...fields}` | `Trigger` | `automation.write` |
| `automation/triggers/delete` | `{id}` | `{}` | `automation.write` |
| `automation/triggers/runs` | `{id, limit?, status?}` | `[Run]` | `automation.read` |
| `automation/runs` | `{job_id?, trigger_id?, status?, limit?}` | `[Run]` | `automation.read` |

**Extension manifest declaring automation capabilities:**

```toml
# extension.toml
[extension]
name = "smart-scheduler"
version = "1.0.0"
description = "Intelligent scheduling extension that optimizes job timing"

[actions]
requires = [
    "automation/jobs",
    "automation/jobs/create",
    "automation/jobs/update",
    "automation/triggers/create",
    "automation/runs",
]

[security]
capabilities = [
    "automation.read",
    "automation.write",
]
```

#### 2. Hook Events (Extensions Observe Automation)

The automation system emits hook events at every lifecycle point. Extensions with hook declarations can observe and optionally modify automation behavior.

**Hook events emitted by the automation system:**

| Hook Event | Mode | Payload | Patchable Fields |
|------------|------|---------|-----------------|
| `automation.job.pre_fire` | sync | `{job_id, job_name, agent, prompt, schedule, attempt}` | `prompt` (modify prompt before dispatch), `cancel: true` (skip this fire) |
| `automation.job.post_fire` | async | `{job_id, job_name, run_id, session_id}` | — |
| `automation.trigger.pre_fire` | sync | `{trigger_id, trigger_name, event, agent, prompt, payload}` | `prompt` (modify), `cancel: true` (skip) |
| `automation.trigger.post_fire` | async | `{trigger_id, trigger_name, run_id, session_id}` | — |
| `automation.run.completed` | async | `{run_id, job_id?, trigger_id?, session_id, duration_ms}` | — |
| `automation.run.failed` | async | `{run_id, job_id?, trigger_id?, error, attempt, will_retry}` | — |

**Example: Extension hook that enriches prompts before dispatch:**

```json
// Daemon → Extension: execute_hook
{
  "jsonrpc": "2.0",
  "id": 10,
  "method": "execute_hook",
  "params": {
    "invocation_id": "hook-01ABC",
    "hook": {
      "name": "enrich-automation-prompt",
      "event": "automation.job.pre_fire",
      "mode": "sync",
      "timeout_ms": 5000
    },
    "payload": {
      "job_id": "job_abc",
      "job_name": "daily-report",
      "agent": "researcher",
      "prompt": "Generate daily AI news summary",
      "schedule": { "mode": "cron", "expr": "0 9 * * *" },
      "attempt": 1
    }
  }
}

// Extension → Daemon: Response with modified prompt
{
  "jsonrpc": "2.0",
  "id": 10,
  "result": {
    "patch": {
      "prompt": "Generate daily AI news summary. Focus on: transformer architectures, agent frameworks, and reasoning models. Today is Thursday April 10, 2026."
    }
  }
}
```

**Example: Extension hook that cancels a fire based on conditions:**

```json
// Extension cancels fire if already ran today
{
  "jsonrpc": "2.0",
  "id": 11,
  "result": {
    "patch": {
      "cancel": true
    }
  }
}
```

**Hook declarations in extension.toml:**

```toml
[[resources.hooks]]
name = "enrich-automation-prompt"
event = "automation.job.pre_fire"
mode = "sync"
required = false
timeout_ms = 5000
```

#### 3. Custom Trigger Sources (Extensions Extend Automation)

Extensions can register custom trigger sources beyond the built-in ones (session, webhook, memory, hook events). This enables integration with external systems without modifying the daemon.

**Architecture:**

```
Extension subprocess ──JSON-RPC──▶ Daemon
                                    │
                      automation/triggers/fire
                                    │
                                    ▼
                            TriggerEngine
                                    │
                                    ▼
                              Dispatcher
                                    │
                                    ▼
                           session.Manager.Create()
```

An extension that wants to act as a trigger source:
1. Subscribes to an external event stream (Slack events, GitHub webhooks, Grafana alerts, etc.)
2. When an event arrives, calls `automation/triggers/fire` Host API method
3. The daemon's trigger engine matches it against registered triggers and dispatches

```json
// Extension → Daemon: Fire a trigger from external event
{
  "jsonrpc": "2.0",
  "id": 20,
  "method": "automation/triggers/fire",
  "params": {
    "event": "ext.github.push",
    "scope": "workspace",
    "workspace_id": "ws_123",
    "payload": {
      "repo": "acme/api",
      "branch": "main",
      "commit": "abc123",
      "author": "dev@acme.com",
      "message": "feat: add new endpoint"
    }
  }
}
// Daemon matches against triggers with event = "ext.github.push"
// and dispatches session with prompt template filled from payload
```

**Host API method for custom trigger firing:**

| Method | Params | Result | Security Capability |
|--------|--------|--------|---------------------|
| `automation/triggers/fire` | `{event: string, scope: string, workspace_id?: string, payload: object}` | `{matched: int, runs: [Run]}` | `automation.write` |

**Trigger configuration referencing extension events:**

```toml
[[automation.triggers]]
scope = "workspace"
name = "code-review-on-push"
event = "ext.github.push"
workspace = "/home/user/project"
filter = { "data.branch" = "main" }
agent = "code-reviewer"
prompt = "Review push to {{ .Data.repo }} by {{ .Data.author }}: {{ .Data.message }}"
```

**Convention**: Extension-provided events use the `ext.` prefix (e.g., `ext.github.push`, `ext.slack.message`, `ext.grafana.alert`). Built-in events use bare names (e.g., `session.stopped`, `webhook`, `memory.consolidated`).

#### Extension Integration Summary

| Integration Level | How | When Available |
|-------------------|-----|----------------|
| **Observe** (hook events) | Extensions subscribe to `automation.*` hook events via hook declarations | When P0 hooks + P1 extensions are complete |
| **Manage** (Host API CRUD) | Extensions call `automation/*` JSON-RPC methods to create/update/delete jobs and triggers | When P1 Host API is complete |
| **Extend** (custom triggers) | Extensions call `automation/triggers/fire` to inject external events as trigger activations | When P1 Host API is complete |
| **Modify** (pre_fire hooks) | Extensions patch prompts or cancel fires via sync hooks on `automation.job.pre_fire` / `automation.trigger.pre_fire` | When P0 hooks + P1 extensions are complete |

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `internal/automation/` | New | Core automation package | Implement from scratch |
| `internal/config/` | Modified | Add `AutomationConfig` struct and validation | Low risk — additive |
| `internal/daemon/boot.go` | Modified | Add `bootAutomation` phase | Low risk — new phase between hooks and servers |
| `internal/daemon/daemon.go` | Modified | Add automation field to `RuntimeDeps` | Low risk — additive |
| `internal/store/globaldb/` | Modified | Add scope-aware automation tables, overlays, and run queries | Low risk — new tables |
| `internal/api/contract/` | Modified | Add automation request/response types | Low risk — additive |
| `internal/api/httpapi/` | Modified | Add automation + webhook route handlers | Medium risk — new route groups |
| `internal/api/core/` | Modified | Add `AutomationManager` interface | Low risk — additive |
| `internal/cli/` | Modified | Add `automation` command group | Low risk — new subcommand |
| `internal/extension/host_api.go` | Modified | Add `automation/*` Host API method handlers | Low risk — additive, follows existing pattern |
| `internal/hooks/events.go` | Modified | Add `automation.*` hook event constants | Low risk — additive |
| `internal/session/` | None | No direct subscription changes — automation consumes observer/hooks outputs | No action |
| `internal/memory/consolidation/` | Minor | Emit a normalized observable completion event for automation ingress | Low risk — additive |
| `internal/observe/` | Minor | Record automation event summaries and normalized ingress metadata | Low risk — additive |
| `web/` | Modified | New `/automation` page, sidebar entry, components | Medium risk — new feature page |
| `openapi/agh.json` | Modified | Add automation endpoints to OpenAPI spec | Low risk — additive |
| `go.mod` | Modified | Add `go-co-op/gocron/v2` dependency | Low risk |

## Testing Approach

### Unit Tests

- **Scheduler**: Register cron/interval/one-shot jobs, verify next-run calculation using `clockwork.FakeClock`, verify singleton mode prevents overlap, verify fire limits, verify retry strategies
- **Trigger engine**: Emit mock normalized activation envelopes, verify exact-match filtering, verify strict prompt template validation, verify scope-aware matching
- **Dispatcher**: Mock `SessionCreator`, verify session creation with correct `CreateOpts`, verify run recording, verify retry on failure
- **Store**: Table-driven CRUD tests with `t.TempDir()` SQLite, verify scope-aware uniqueness, overlay persistence, and query filtering
- **Config**: Parse TOML automation section, validate schedule expressions, validate retry config, verify config-to-SQLite sync logic

### Integration Tests

- **Daemon boot → schedule fires → session created**: Full lifecycle test with real scheduler (fast cron expression like `@every 1s`), verify session appears in session list
- **Event → trigger → session**: Create a trigger for `session.stopped`, complete a session, verify the observer/hooks boundary produces a normalized activation and dispatches the new session
- **Webhook → trigger → session**: POST to webhook endpoint with valid HMAC, verify trigger fires and session is created
- **Webhook auth reject**: Invalid HMAC or stale timestamp is rejected before any dispatch
- **Scope-aware naming**: Global and workspace-scoped automations with the same human name coexist correctly
- **TOML sync**: Boot with TOML jobs, verify they appear in SQLite, toggle only `enabled` via overlay, reboot, and verify the overlay still applies while the definition remains TOML-owned
- **Fire limit persistence**: Restart daemon inside the active fire-limit window and verify the limit still applies
- **Graceful shutdown**: Start jobs, initiate shutdown, verify running jobs receive context cancellation

## Development Sequencing

### Build Order

1. **Config + Store** (no dependencies) — `AutomationConfig` struct, TOML parsing, validation, SQLite schema, CRUD queries
2. **Job + Trigger types** (depends on 1) — domain types, `ScheduleSpec`, `RetryConfig`, `FireLimitConfig`, serialization
3. **Dispatcher** (depends on 2) — `SessionCreator` interface, dispatch logic, run recording, global concurrency gate, restart-safe fire-limit evaluation
4. **Scheduler** (depends on 2, 3) — gocron v2 wrapper, register/unregister jobs, singleton protection, lifecycle hooks wiring to dispatcher
5. **Trigger engine** (depends on 2, 3) — activation-envelope ingress, observer/hooks-backed internal event normalization, authenticated webhook normalization, filter evaluation
6. **Manager** (depends on 3, 4, 5) — compose scheduler + trigger engine + dispatcher + store, TOML sync, overlay resolution, `Server` interface
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
| `automation.fire_limit.rejected` | Counter | Dispatch attempts rejected by restart-safe fire-limit evaluation |
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
| Webhook received | INFO | `webhook_id`, `trigger_name`, `remote_addr`, `payload_size` |
| Webhook rejected | WARN | `webhook_id`, `reason`, `remote_addr` |
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

4. **Configurable retry per job** — agent sessions are expensive. Default `none` prevents cost amplification. Jobs with transient failure modes opt into `backoff`. Fire limits provide a restart-safe safety net based on persisted runs. (ADR-004)

5. **No missed-job backfill** — if the daemon is down when a cron fires, the job is skipped. Running stale jobs hours late is usually wrong for LLM-powered agents. Record the miss, let the user decide. (Aligned with OpenFang's deliberate choice.)

6. **TOML jobs are source-of-truth** — on daemon boot, TOML-defined jobs sync to SQLite (create if missing, update if changed). Dynamic jobs (API/CLI) coexist in SQLite. Config-sourced jobs may persist only an `enabled/disabled` operational overlay; definition edits remain TOML-owned. This prevents config drift without removing runtime operational control.

7. **Prompt templates** — trigger prompts support Go `text/template` syntax against normalized activation envelopes, with strict validation and `missingkey=error`. Schedule prompts are static strings.

8. **Explicit scope model** — automation definitions are either `global` or `workspace` scoped. Name uniqueness and webhook routes are derived from that scope boundary.

9. **Observer/hooks as canonical ingress** — internal automation triggers consume normalized events from the daemon's existing observer/hooks boundary instead of adding new direct subscriptions to `session` or `memory/consolidation`.

### Known Risks

1. **gocron v2 maintenance** — if abandoned, the library is thin enough to fork or replace. AGH wraps it behind `internal/automation/schedule.go`, isolating the dependency. Likelihood: low.

2. **Trigger fan-out performance** — many triggers with complex filters on high-frequency events (e.g., `session.created`) could cause latency. Mitigation: fire limits, efficient filter matching (exact string match, not regex in v1), async dispatch.

3. **TOML ↔ SQLite sync clarity** — operators may expect broader mutation of config-backed definitions than v1 allows. Mitigation: explicit overlay model, clear API errors, and visible `source: "config"` semantics.

4. **Webhook secret management** — HMAC removes unauthenticated webhook abuse, but secret distribution and rotation become operator concerns. Mitigation: per-trigger secrets, explicit rotation workflow, and write-only secret handling in API/CLI surfaces.

5. **Fire-limit query cost** — evaluating fire limits from persisted runs introduces read pressure on hot automation paths. Mitigation: bounded-window queries plus optional in-memory cache, with SQLite remaining the source of truth.

## Architecture Decision Records

- [ADR-001: Built-In Daemon Component with Extension Integration Points](adrs/adr-001.md) — Automation lives in `internal/automation/` as a daemon Server, not as an extension, with Host API exposure planned for when P1 completes.
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Single package handles both time-based scheduling and event-driven triggers through a shared dispatch mechanism.
- [ADR-003: gocron v2 as In-Process Scheduling Runtime](adrs/adr-003.md) — gocron v2 chosen over robfig/cron v3 for context propagation, lifecycle hooks, singleton mode, and active maintenance.
- [ADR-004: Configurable Per-Job Retry with Fire Limits](adrs/adr-004.md) — Per-job retry strategy (none/backoff) with global fire limits to prevent runaway execution.
