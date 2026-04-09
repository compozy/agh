# Our System: Kernel, Prompt & Config

## Current Architecture

### High-Level Overview

AGH is a multi-agent orchestration system that runs as a single daemon process on the local machine. The kernel is the root infrastructure owner, managing sessions, drivers, messaging, and lifecycle. Each session is an isolated execution context with its own agent registry, workgroup hierarchy, state store, and resilience management.

The architecture follows this hierarchy:

```
Kernel (singleton daemon)
  -> SessionManager (owns all sessions)
     -> Session (isolated execution context)
        -> WorkgroupManager (hierarchical agent groups)
           -> AgentInfo (per-agent in-memory records)
        -> ResilienceManager (circuit breakers, message delivery, failure handling)
        -> HookRouter (NATS-based hook event ingestion and routing)
        -> HealthChecker (periodic PID liveness monitoring)
        -> Store (SQLite via internal/state)
```

Communication flows through three channels:

1. **UDS (Unix Domain Socket)** - CLI-to-kernel commands via a Gin HTTP engine over a Unix socket
2. **NATS (embedded)** - Inter-agent messaging, hook events, broadcast/escalate patterns
3. **HTTP (TCP)** - Dashboard web server serving the same Gin engine

### Boot Sequence

The kernel boots in a strict ordered sequence (13 steps):

1. Acquire daemon lock (file-based, singleton enforcement)
2. Write daemon.json metadata (PID, socket path, version, timestamp)
3. Load global config from `~/.agh/config.toml` (with defaults fallback)
4. Initialize structured logger (file + optional mirror writer)
5. Start embedded NATS server
6. Create Gin engine, register all HTTP routes, start UDS bridge
7. Load role catalog from `~/.agh/roles/` directory
8. Initialize driver registry (placeholder `UnimplementedDriver` for each configured driver)
9. Load skill registry (bundled + user + workspace skills)
10. Load prompt templates (master, worker, advisor, reviewer, researcher)
11. Initialize SessionManager
12. Start dashboard HTTP server (TCP)
13. Start signal handler (SIGINT, SIGTERM) and lifecycle goroutine

The lifecycle goroutine uses `oklog/run` to manage the signal context and HTTP server error channel, coordinating graceful shutdown.

### Session Lifecycle

Sessions go through: `starting -> active -> stopping -> stopped`

Session creation (`SessionManager.Create`) does:

1. Validate goal + resolve workspace path
2. Load session-scoped config (global config overlaid with workspace `.agh/config.toml`)
3. Load session-scoped role catalog (global roles merged with workspace roles)
4. Generate session ID (xid), create session directory under `~/.agh/sessions/<id>/`
5. Reserve the session name (auto-generated from goal if not provided)
6. Initialize session infrastructure:
   - Open SQLite state store (`session.db`)
   - Create agent and workgroup registries (in-memory + SQLite-backed)
   - Create PTY manager, WebSocket hub, NATS scope validator
   - Create suture supervisor (restart policy from config)
   - Create WorkgroupManager and ResilienceManager
   - Create root workgroup
   - Start HookRouter (subscribes to NATS hook and command subjects)
   - Bootstrap supervisor agent (master type) and advisor agent
   - Send kickoff message to supervisor
   - Start HealthChecker goroutine
7. Activate session (move from pending to active in SessionManager)
8. Persist session metadata (meta.json, goal.md, index.json)

Session resume reconstructs from persisted state, including historical context (last 10 blackboard entries + statuses).

### Agent Types and Driver Architecture

Five canonical agent types: `master`, `worker`, `advisor`, `reviewer`, `researcher`.

The `AgentDriver` interface abstracts agent runtime interaction:

```go
type AgentDriver interface {
    Name() string
    Start(ctx context.Context, opts StartOpts) (*AgentProcess, error)
    SendMessage(ctx context.Context, proc *AgentProcess, msg string) error
    Stop(ctx context.Context, proc *AgentProcess) error
    BuildHookConfig(agentName string, hookEndpoint string) (*HookConfig, error)
    ParseHookEvent(rawPayload []byte) (*HookEvent, error)
    HealthCheck(ctx context.Context, proc *AgentProcess) (AgentHealth, error)
    DetectReady(ctx context.Context, proc *AgentProcess) error
}
```

Configured drivers: `claude`, `codex`, `opencode`, `pi`. At boot, each is registered as an `UnimplementedDriver` placeholder that returns `ErrNotImplemented` for all operations. Real driver implementations are injected via `WithDriver()` option.

`StartOpts` carries: Name, Role, Model, SystemPrompt, WorkDir, TerminalSize, Tools, ExtraDirs, EnvVars.

`AgentProcess` is an alias for `internalpty.Process` -- agents run as PTY-backed subprocesses with ring buffer output capture.

### Workgroup Hierarchy

Workgroups form a tree structure with depth limits (default 3). Each workgroup has:

- A single master agent (required before workgroup becomes active)
- Worker/advisor/reviewer/researcher agents
- State transitions: `create -> active -> closing -> closed`

The `WorkgroupManager` enforces:

- Only master agents can be spawned before workgroup is active
- One master per workgroup
- Agent count limits per workgroup and per session
- Depth limits for nesting
- Destroy requires all children to be closed first
- Workgroup destroy snapshots state and notifies parent

### Messaging and State

Three messaging patterns via NATS:

1. **Direct** (`agh send`) - Agent-to-agent within same or cross workgroups
2. **Broadcast** (`agh broadcast`) - To all agents in caller's workgroup
3. **Escalate** (`agh escalate`) - To parent workgroup's master

State is maintained in SQLite through `internal/state`:

- **Blackboard** - Shared knowledge entries (type + content, scoped to workgroup)
- **Status** - Agent status updates (state + task description)
- **Events** - Audit log of all system events (messages, hooks, lifecycle)
- **Agent/Workgroup records** - Persisted registry snapshots

### Resilience Layer

The `ResilienceManager` provides:

- **Per-agent circuit breakers** (gobreaker): 3 consecutive failures trips the circuit, 30s open timeout
- **Per-agent message channels** (buffered, size 64)
- **Agent supervision** via suture (restart policy from config: max 3 attempts, 5s backoff)
- **Failure detection** through 5 causes: process exit, PTY EOF, health check, circuit open, message failure
- **Failure response**: mark agent dead, log event, notify workgroup master or parent master
- **Master death**: freeze entire workgroup (set to closing, idle all members), notify parent

The `HealthChecker` runs periodic PID liveness probes (default 30s interval), falling back to `syscall.Kill(pid, 0)` when driver doesn't implement `HealthCheck`.

### Hook System

The `HookRouter` handles tool-use events from agent processes:

1. Agent hook scripts publish raw JSON to a NATS command subject
2. Router resolves the source agent, validates workgroup membership
3. Driver parses raw payload into normalized `HookEvent`
4. Event is logged to SQLite and routed to workgroup master
5. If master isn't ready yet, events are queued (up to 1000 per workgroup)

### Dashboard

The kernel exposes a dashboard backend adapter that translates kernel domain objects into dashboard-specific view types. It supports:

- Session listing (active + historical from filesystem)
- Topology view (workgroup tree with agent details)
- Agent listing and detail
- Blackboard reading
- PTY output streaming (ring buffer subscription)
- Topology event streaming (WebSocket via WsHub)

---

## Key Types & Interfaces

### Kernel Core

```go
type Kernel struct {
    Config          *config.Config           // Global + session-merged config
    NATS            *transport.EmbeddedNATS  // Embedded NATS server
    UDS             *transport.UDSBridge     // Unix domain socket bridge
    HTTP            *http.Server             // Dashboard TCP server
    HomePaths       config.HomePaths         // ~/.agh/ layout
    RoleCatalog     RoleCatalogStore         // Global role definitions
    DriverRegistry  DriverRegistryStore      // Agent runtime drivers
    PromptTemplates map[string]string        // Type -> rendered template
    skillRegistry   *skills.Registry         // Bundled + user skills
    SessionManager  *SessionManager          // Session lifecycle
    Logger          *slog.Logger
    DaemonLock      *flock.Flock
    DaemonInfo      *DaemonInfo
    DashboardAddr   string
    // internal lifecycle channels and state...
}
```

### Session

```go
type Session struct {
    ID, Name, Goal, Workspace string
    State                     SessionState          // starting|active|stopping|stopped
    Config                    config.Config          // Merged config for this session
    ConfigOverridePath        string
    RoleCatalog               RoleCatalogStore
    Store                     *state.Store           // SQLite persistence
    AgentRegistry             AgentRegistryStore     // In-memory + SQLite agent records
    WorkgroupRegistry         WorkgroupRegistryStore // In-memory + SQLite workgroup records
    PtyManager                *internalpty.Manager
    Supervisor                *suture.Supervisor
    WsHub                     *WsHub
    NATSSubscriptions         []*nats.Subscription
    Breakers                  map[string]*gobreaker.CircuitBreaker
    Buffers                   map[string]*RingBuffer
    Channels                  map[string]chan Message
    ScopeValidator            *transport.ScopeValidator
    HealthChecker             *HealthChecker
    WorkgroupManager          *WorkgroupManager
    ResilienceManager         *ResilienceManager
    CreatedAt                 time.Time
    SessionDir                string
    // attach observers, shutdown function, lifecycle context...
}
```

### Agent & Workgroup

```go
type AgentInfo struct {
    ID, Name, Role, Type, Workgroup, Model, State string
    Driver  AgentDriver
    Process *AgentProcess          // PTY-backed subprocess handle
    Channel chan Message            // Buffered message delivery
    Breaker *gobreaker.CircuitBreaker
}

type WorkgroupInfo struct {
    ID, Name, Parent, Master, State string
    Agents   []string               // Member agent IDs
    Children []string               // Child workgroup IDs
}
```

### Key Interfaces

```go
type AgentDriver interface {
    Name() string
    Start(ctx, StartOpts) (*AgentProcess, error)
    SendMessage(ctx, *AgentProcess, string) error
    Stop(ctx, *AgentProcess) error
    BuildHookConfig(agentName, hookEndpoint string) (*HookConfig, error)
    ParseHookEvent([]byte) (*HookEvent, error)
    HealthCheck(ctx, *AgentProcess) (AgentHealth, error)
    DetectReady(ctx, *AgentProcess) error
}

type RoleCatalogStore interface {
    Lookup(name string) (*config.RoleConfig, bool)
    List() []*config.RoleConfig
}

type DriverRegistryStore interface {
    Lookup(name string) (AgentDriver, bool)
    Names() []string
}

type AgentRegistryStore interface {
    LookupAgent(id string) (*AgentInfo, bool)
    LookupByName(name string) (*AgentInfo, bool)
    List() []*AgentInfo
}

type WorkgroupRegistryStore interface {
    LookupWorkgroup(id string) (*WorkgroupInfo, bool)
    LookupByName(name string) (*WorkgroupInfo, bool)
    List() []*WorkgroupInfo
}
```

### Config

```go
type Config struct {
    Limits    LimitsConfig    // Session/agent/workgroup ceilings, restart policy, health intervals
    Runtime   RuntimeConfig   // Default driver, bootstrap agents, driver binary configs
    Meta      MetaConfig      // Meta-learning on/off, auto-approve
    Dashboard DashboardConfig // Host, port, ring buffer size, terminal dimensions
}

type RoleConfig struct {
    Name, Description, Type, Driver, Model, SystemPrompt string
    Status       ArtifactStatus  // approved|draft
    DraftVersion int
    Path         string
}

type Playbook struct {
    Name, Description, Domain, Content string
    Tags         []string
    Status       ArtifactStatus
    DraftVersion int
    Path         string
}
```

Config merging: `Default() -> global ~/.agh/config.toml overlay -> workspace .agh/config.toml overlay`. Uses TOML with pointer-based overlay structs to distinguish "not set" from zero values.

### Prompt Assembly

```go
type AssembleOptions struct {
    Type               string             // Agent type (master/worker/advisor/reviewer/researcher)
    Template           string             // Override template (default: load from embedded FS)
    RoleName           string
    Role               *config.RoleConfig // Role specialization
    SkillsCatalog      string             // XML-formatted available skills
    RolesCatalog       string             // "AVAILABLE ROLES:" section for masters
    PlaybooksCatalog   string             // "AVAILABLE PLAYBOOKS:" section for masters
    Context            Context            // Goal, agent identity, workgroup identity
    AdditionalSections []string           // Extra sections (workspace, historical context)
}

type Context struct {
    Goal, Domain, AgentID, WorkgroupID, WorkgroupName, AgentType, RoleName string
}
```

Prompt assembly order:

1. Base template (from embedded `internal/prompt/templates/<type>.md`)
2. Role specialization (`SPECIALIZATION:` + role.SystemPrompt)
3. Skills catalog (XML formatted)
4. Roles catalog (master-only, `AVAILABLE ROLES:` header)
5. Playbooks catalog (master-only, `AVAILABLE PLAYBOOKS:` header)
6. Context block (`CONTEXT:` with goal, agent identity, etc.)
7. Additional sections (workspace path, historical context)

Tool allowlists per type:

- **master**: read, grep, glob, list, bash
- **worker**: read, write, edit, bash, grep, glob, list
- **advisor**: read, grep, glob, list
- **reviewer**: read, bash, grep, glob, list
- **researcher**: read, grep, glob, list

### Skills System

Skills are loaded from multiple sources (in priority order):

1. Bundled skills (embedded filesystem via `internal/skills/bundled`)
2. User global skills (`~/.agh/skills/`)
3. User agents skills (`~/.agents/skills/`)
4. Workspace agent skills (`<workspace>/.agents/skills/`)
5. Workspace AGH skills (`<workspace>/.agh/skills/`)

The `skills.Registry` loads all, freezes (immutable), and can produce `SkillSnapshot` filtered by OS for injection into prompts. The `ClawHub` client (`internal/skills/clawhub.go`) provides marketplace integration for skill discovery, search, and installation.

---

## Gaps & Opportunities

### Architectural Strengths

1. **Clean separation of concerns**: kernel owns infra, sessions own execution, managers own domains.
2. **Functional options pattern**: `NewKernel(opts...)` is testable and composable.
3. **Defense in depth**: circuit breakers, health checks, supervisor restart policies, workgroup freezing.
4. **Config layering**: global defaults -> home config -> workspace config with overlay semantics.
5. **Comprehensive API surface**: the Gin router exposes every kernel operation as HTTP endpoints.
6. **Immutable cloning discipline**: agent/workgroup info is always cloned before returning to callers.
7. **Skills as first-class**: multi-source skill loading with marketplace integration.
8. **Session persistence**: meta.json, goal.md, session index, state store enable resume.

### Gaps and Improvement Opportunities

#### 1. Driver Implementation Gap

The `UnimplementedDriver` placeholder pattern means all four configured drivers (claude, codex, opencode, pi) return `ErrNotImplemented` at boot. Real driver implementations must be injected externally via `WithDriver()`. This is by design (task-gated), but means the system has no built-in runnable drivers yet.

#### 2. Prompt System Rigidity

- **Static tool allowlists**: `agentTypeTools` is a hardcoded map. No way to configure per-role or per-session tool restrictions.
- **Template loading is sync.Once**: templates are loaded once and cached globally. Hot-reloading or per-session template customization isn't supported.
- **No template variables/interpolation**: templates are raw Markdown strings concatenated together. There's no templating engine (Go templates, etc.) for conditional sections or variable substitution within templates.
- **Catalogs are master-only**: roles and playbooks catalogs are only injected for master agents (`buildPromptCatalogsForAgent` returns empty strings for non-masters).

#### 3. Session-to-Session Isolation vs. Sharing

Sessions are fully isolated. There's no mechanism for cross-session communication, shared knowledge bases, or session handoff. The session index only maps workspace -> session IDs for listing.

#### 4. Config Limitations

- **No hot-reload**: config is loaded once at session creation. Runtime config changes require session restart.
- **Driver config is minimal**: only `binary` and `mode` (opencode only). No support for environment variables, API keys, rate limits, or model-specific parameters in driver config.
- **No per-role config overrides**: roles define driver/model but can't override limits, dashboard settings, etc.
- **Validation is strict but not extensible**: known driver names are hardcoded in `isKnownDriver()`.

#### 5. State Store Coupling

The `state.Store` is SQLite-based. The kernel directly depends on concrete `state.Store` rather than an interface, limiting testability and alternative backend support. The session registries duplicate state between in-memory maps and SQLite.

#### 6. Error Handling in Messaging

Message delivery through `ResilienceManager.DeliverMessage` is synchronous and blocking. If an agent's driver `SendMessage` is slow, it blocks the caller. There's no async message queue beyond the per-agent channel buffer (size 64).

#### 7. Hook System Single Point of Failure

All hook events for a session flow through a single `HookRouter`. If NATS subscription processing falls behind, the 1000-event pending queue drops oldest events silently (with a warning log).

#### 8. No Agent Readiness Protocol

The kernel calls `driver.DetectReady()` via the interface, but the bootstrap flow currently just marks agents ready immediately after `attachProcess`. There's no polling loop waiting for agent readiness signals.

#### 9. Missing Observability

- No metrics/counters for agent spawns, message throughput, circuit breaker trips.
- No distributed tracing across agent interactions.
- Health check results are not persisted or exposed via API.

#### 10. Dashboard Backend is Adapter-Heavy

The `dashboardBackend` adapter has significant view-model translation code. The dashboard package defines its own `AgentView`, `WorkgroupView`, `TopologySnapshot`, etc. that mirror kernel types closely. This duplication is intentional (package boundary) but adds maintenance cost.

#### 11. Prompt Assembly Has No Caching

Every agent spawn calls `prompt.Assemble()` which re-loads templates, re-builds catalogs, and re-renders context. For sessions spawning many agents of the same type, this is redundant work.

#### 12. Workspace Resolution

`ResolveWorkspace` defaults to `os.Getwd()`. In a daemon context, the CWD is the daemon's CWD, not the user's. The workspace must always be passed explicitly for CLI commands, which is handled, but could be a source of bugs in automated scenarios.

#### 13. No Agent-to-Agent Direct Communication Channel

Despite having NATS subscriptions, agents don't subscribe to their own agent subjects. Message delivery goes through `driver.SendMessage()` (PTY stdin injection), not NATS pub/sub. NATS is used for hook events and topology notifications, not for the primary message delivery path.

#### 14. Skills Registry is Session-Scoped Reload

`kernel.skillSnapshot(workspace)` reloads the entire skill registry for every workspace-scoped call. The kernel-level registry (no workspace) is cached, but workspace-specific registries are rebuilt on each agent spawn.
