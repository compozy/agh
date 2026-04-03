# Our System: CLI, Skills & Features

## Current Architecture

### High-Level Overview

AGH is a single-binary, local-first Go agent orchestration framework. It runs as a daemon process that manages sessions, each containing a tree of workgroups with spawned AI coding agents. The binary (`cmd/agh/main.go`) uses **Cobra** for CLI dispatch and communicates with the running daemon over a **Unix Domain Socket (UDS)** using JSON-over-HTTP.

```
User CLI (cobra) --> UDS HTTP Client --> Kernel Daemon
                                           |
                            +--------------+------------------+
                            |              |                  |
                       SessionManager   NATS Bus       Dashboard HTTP
                            |              |                  |
                    Sessions (1..N)   Transport      Gin Engine (API + Web)
                            |
               +------------+----------------+
               |            |                |
          Workgroups    AgentRegistry    State Store (SQLite)
               |
          Agent Processes (PTY-managed, driver-abstracted)
```

### Entry Point

`cmd/agh/main.go` -- Minimal main that creates a `cli.NewRootCommand()` cobra tree and executes it with `context.Background()`. Exit code is 0/1 based on error return.

### CLI Layer (`internal/cli/`)

The CLI is organized around a single `daemonCommandDeps` dependency injection struct that threads through all subcommands. This struct provides factory functions for home path resolution, kernel construction, daemon discovery, client creation, skill registry loading, and environment variable access. Every command that needs to talk to the daemon goes through `daemonClientFromDeps()` or `runtimeDaemonClientFromDeps()`.

#### Command Tree

| Command                                                 | File           | Description                                        |
| ------------------------------------------------------- | -------------- | -------------------------------------------------- |
| `agh version`                                           | `root.go`      | Print version string                               |
| `agh start [--foreground]`                              | `daemon.go`    | Start kernel daemon (detached or foreground)       |
| `agh status`                                            | `daemon.go`    | Show kernel status                                 |
| `agh stop`                                              | `daemon.go`    | Shutdown kernel                                    |
| `agh session start/list/stop/status/resume`             | `session.go`   | Full session lifecycle management                  |
| `agh workgroup create/list/destroy`                     | `workgroup.go` | Workgroup CRUD within a session                    |
| `agh topology`                                          | `workgroup.go` | Display full workgroup/agent tree                  |
| `agh spawn --role --name --workgroup`                   | `runtime.go`   | Spawn agent into workgroup                         |
| `agh kill <agent-id>`                                   | `runtime.go`   | Terminate agent                                    |
| `agh ps [--verbose]`                                    | `runtime.go`   | List agents (with optional PID/PTY/buffer details) |
| `agh whoami`                                            | `runtime.go`   | Show current agent identity (env-based)            |
| `agh attach <agent-id>`                                 | `attach.go`    | Read-only PTY streaming from agent                 |
| `agh send <agent-id> <msg>`                             | `messaging.go` | Direct message to agent                            |
| `agh broadcast <msg>`                                   | `messaging.go` | Broadcast to workgroup                             |
| `agh escalate <msg>`                                    | `messaging.go` | Escalate to parent workgroup master                |
| `agh state read/append`                                 | `state.go`     | Read/write blackboard entries                      |
| `agh context`                                           | `state.go`     | Aggregated workgroup context view                  |
| `agh agent-status <task-or-agent>`                      | `state.go`     | Read or update agent status                        |
| `agh events`                                            | `state.go`     | Read event log                                     |
| `agh wait <agent\|workgroup>`                           | `lifecycle.go` | Block until terminal state                         |
| `agh done <reason>`                                     | `lifecycle.go` | Mark current agent as done                         |
| `agh dashboard`                                         | `dashboard.go` | Display dashboard URL and status                   |
| `agh roles list/get/create/approve`                     | `roles.go`     | Role management with draft/approve workflow        |
| `agh playbooks list/get/save/approve`                   | `playbooks.go` | Playbook management with draft/approve workflow    |
| `agh skill list/view/info/create/install/remove/search` | `skill.go`     | Full skill lifecycle management                    |
| `agh install`                                           | `install.go`   | Install bundled roles to global home               |
| `agh hook-event` (hidden)                               | `hooks.go`     | Forward raw hook payload to kernel via NATS        |

#### Dual Output System (`output.go`)

The CLI supports two output modes via the global `--output` / `-o` flag:

- **`human`** (default for interactive terminals) -- Styled lipgloss tables and key-value sections via `internal/cli/human/`
- **`toon`** (default when `COLLAB_AGENT` or `AGI_AGENT` env var is set) -- Structured TOON format via `internal/toon/`

Resolution order: explicit `--output` flag > agent env vars > human default.

Every command's `RunE` calls `writeOutput(cmd, humanFn, toonFn)` which selects the appropriate renderer. If a human renderer is not provided, it falls back to TOON. This architecture ensures every command works for both humans and AI agents.

#### Human Output Layer (`internal/cli/human/`)

Three files:

- **`styles.go`** -- Lipgloss style constants: `HeaderStyle`, `LabelStyle`, `ValueStyle`, `SuccessStyle`, `ErrorStyle`, `WarnStyle`, `InfoStyle`, `DimStyle`
- **`renderer.go`** -- Render functions for every view type: `RenderKernelStatus`, `RenderSessionSummary`, `RenderSessionList`, `RenderSessionDetail`, `RenderAgents`, `RenderVerboseAgents`, `RenderAgentIdentity`, `RenderAgentRuntime`, `RenderMessage`, `RenderStatus`, `RenderBlackboard`, `RenderEvents`, `RenderWorkgroups`, `RenderTopologyTree`, `RenderRoleList`, `RenderRoleDetail`, `RenderPlaybookList`, `RenderPlaybookDetail`, `RenderDashboard`, `RenderContext`
- **`renderer_test.go`** -- Tests

Each renderer uses the same two primitives:

- `renderKeyValueSection(title, rows)` for single-entity views
- `renderTable(title, headers, rows)` using lipgloss table for lists

State values are styled via `styledState()` which colorizes based on keywords (error/dead=red, done/ready/running=green, stopping/pending=yellow).

### Daemon Architecture (`daemon.go`)

The daemon supports two modes:

1. **Detached** (default `agh start`) -- Spawns itself as a child process with `--internal-child`, redirects stdout/stderr to log file, detaches process group. Parent polls via `waitForDaemonStartup()` with 100ms tick until kernel responds on UDS or 15s timeout.
2. **Foreground** (`agh start --foreground`) -- Runs kernel in-process, mirrors logs to stderr.

Daemon discovery uses a multi-layer approach:

1. Check `~/.agh/daemon.lock` (flock-based)
2. Read `~/.agh/daemon.json` for PID and socket path
3. Verify PID is alive via `kill -0`
4. Probe socket with HTTP health check
5. Fallback: probe dashboard HTTP endpoint for kernel status

Stale artifacts are automatically cleaned up when the daemon is discovered to not be running.

#### UDS HTTP Client (`udsHTTPDaemonClient`)

All CLI-to-daemon communication flows through this client which uses `http.Transport` with a custom `DialContext` that connects to the Unix socket. It implements the full `daemonAPIClient` interface with methods for every kernel API endpoint. Request timeout defaults to 30s, with special 3-minute timeout for session start (to allow bootstrap agent readiness).

### Kernel (`internal/kernel/`)

The `Kernel` struct owns all global shared infrastructure:

- **Config** -- Loaded from `~/.agh/config.toml` (TOML format)
- **NATS** -- Embedded NATS server for inter-agent messaging
- **UDS** -- Unix Domain Socket bridge for CLI-to-daemon RPC
- **HTTP** -- Gin-based HTTP server for dashboard and REST API
- **RoleCatalog** -- Loaded roles from `~/.agh/roles/`
- **DriverRegistry** -- Registered agent drivers (claude, codex, opencode, pi)
- **SkillRegistry** -- Loaded skills from bundled + user + workspace directories
- **PromptTemplates** -- Pre-rendered prompt templates for each agent type
- **SessionManager** -- Manages all session lifecycles
- **DaemonLock** -- flock-based exclusive daemon lock
- **DaemonInfo** -- Metadata about running daemon (PID, socket, version, start time)

Boot sequence (13 steps): acquire lock -> write daemon info -> load config -> init logger -> start NATS -> start UDS -> load roles -> init drivers -> load skills -> load prompt templates -> init session manager -> start HTTP -> start signal handler.

Lifecycle is managed via `oklog/run` group with signal handler and HTTP server error channels. Graceful shutdown closes UDS, stops all sessions, shuts down HTTP, closes NATS, releases lock.

#### Session Model

Each session contains:

- **ID** (xid), **Name**, **Goal**, **State** (starting/active/stopping/stopped)
- **Workspace** path
- **Config** override, **RoleCatalog**, **Store** (SQLite state)
- **AgentRegistry**, **WorkgroupRegistry**
- **PtyManager** for terminal management
- **Supervisor** (suture) for agent process supervision
- **WsHub** for WebSocket dashboard streaming
- **NATS subscriptions**, circuit breakers, ring buffers, channels
- **ScopeValidator**, **HealthChecker**, **WorkgroupManager**, **ResilienceManager**

Session state machine: starting -> active -> stopping -> stopped.

#### Agent Drivers

Four drivers implemented in `internal/drivers/`:

- **claude** -- Claude Code CLI
- **codex** -- OpenAI Codex CLI
- **opencode** -- OpenCode CLI (with mode support)
- **pi** -- Pi CLI

Each implements the `AgentDriver` interface:

```go
type AgentDriver interface {
    Name() string
    Start(ctx, StartOpts) (*AgentProcess, error)
    SendMessage(ctx, proc, msg) error
    Stop(ctx, proc) error
    BuildHookConfig(agentName, hookEndpoint) (*HookConfig, error)
    ParseHookEvent(rawPayload) (*HookEvent, error)
    HealthCheck(ctx, proc) (AgentHealth, error)
    DetectReady(ctx, proc) error
}
```

#### Agent Environment

When an agent runs inside the AGH network, it receives environment variables:

- `COLLAB_AGENT` / `AGI_AGENT` -- Agent ID
- `COLLAB_AGENT_NAME` / `AGI_AGENT_NAME` -- Agent name
- `COLLAB_SESSION` / `AGI_SESSION` -- Session name
- `COLLAB_SESSION_ID` / `AGI_SESSION_ID` -- Session ID
- `COLLAB_WORKGROUP` / `AGI_WORKGROUP` -- Workgroup ID
- `COLLAB_WORKGROUP_NAME` / `AGI_WORKGROUP_NAME` -- Workgroup name
- `COLLAB_SOCKET` / `AGI_SOCKET` -- Daemon socket path
- `COLLAB_WORKSPACE` / `AGI_WORKSPACE` -- Workspace path

The `COLLAB_*` and `AGI_*` prefixes are interchangeable (both checked).

### Skills System (`internal/skills/`)

#### Types

```go
type SkillMeta struct {
    Name, Description, License, Compatibility, AllowedTools string
    Metadata map[string]string
}

type Skill struct {
    Meta     SkillMeta
    Content  string           // Markdown body after frontmatter
    Source   SkillSource      // bundled/user/.agents/workspace/clawhub
    Dir      string           // Filesystem directory
    FilePath string           // Path to SKILL.md
    Enabled  bool
    Warnings []Warning        // Parser/security findings
}

type SkillSnapshot struct {
    Skills  []*Skill
    Catalog string            // XML catalog for system prompt injection
    Version int
}
```

#### Source Hierarchy (4 levels, later overrides earlier)

1. **Bundled** -- Embedded in binary via `internal/skills/bundled/skills/` (currently only `code-review`)
2. **User** -- `~/.agh/skills/` and `~/.agents/skills/`
3. **.agents** -- `<workspace>/.agents/skills/`
4. **Workspace** -- `<workspace>/.agh/skills/`

#### Loader (`loader.go`)

Parses SKILL.md files with YAML frontmatter:

```
---
name: skill-name
description: When to use this skill.
license: MIT
compatibility: macOS, Linux
allowed-tools: grep, find
metadata:
  key: value
---
# Skill body (markdown)
```

Enforces:

- Max 256KB file size
- Required `name` and `description` fields
- Name validation (lowercase, hyphens, no consecutive hyphens, max 64 chars)
- YAML recovery for unquoted colons

#### Registry (`registry.go`)

Thread-safe `Registry` with:

- `LoadAll(LoadConfig)` -- Refresh from all 4 source levels
- `Get(name)` / `List()` -- Defensive copies
- `Snapshot(filter)` -- Immutable filtered view with XML catalog, cached by version+filter key
- `Freeze()` -- Prevents further loading after boot

#### Verification (`verify.go`)

Scans skill content for prompt injection patterns:

- **Critical** (blocks loading): "ignore previous instructions", "disregard rules", "forget instructions", "you are now" (role hijack), "new instructions:", "system prompt override"
- **Warning**: "do not tell the user", "output system prompt", HTML comment instructions, suspicious localhost URLs
- **Info**: pipe-to-shell, system XML tags

#### Eligibility (`eligibility.go`)

Filters skills by:

- Enabled flag
- Disabled skills list
- Allowed skills whitelist
- OS compatibility (macOS/Linux/Windows from compatibility field)

#### ClawHub Marketplace (`clawhub.go`)

Full marketplace client with:

- Search API (`/api/v1/search`)
- Download API (`/api/v1/download`) with archive extraction (tar.gz, tar, zip, or raw SKILL.md)
- Retry with exponential backoff (1.5s initial, 30s max, 5 retries)
- Path traversal protection for archives
- Post-download verification (critical warnings block install)

#### Catalog (`catalog.go`)

Builds XML catalog injected into agent system prompts:

```xml
<available_skills>
  <instructions>When a task matches a skill's description, run `agh skill view <name>` to load the full instructions...</instructions>
  <skill>
    <name>code-review</name>
    <description>Review code changes...</description>
  </skill>
</available_skills>
```

### Prompt Assembly (`internal/prompt/assembler.go`)

The `Assemble()` function composes a final system prompt from layers:

1. Type template (master/worker/advisor/reviewer/researcher)
2. Role specialization (from role config)
3. Skills catalog (XML)
4. Roles catalog
5. Playbooks catalog
6. Session context (workgroup, agents, blackboard)
7. Additional sections

### Transport Layer (`internal/transport/`)

- **Embedded NATS** -- In-process NATS server for inter-agent messaging
- **UDS Bridge** -- Unix Domain Socket bridge connecting CLI to kernel via Gin engine
- **Scope Validator** -- Agent message scope enforcement

### State Layer (`internal/state/`)

SQLite-backed persistent state per session with:

- Blackboard entries (shared knowledge)
- Status entries (agent task/state)
- Event entries (audit log)
- Workgroup records

### Configuration (`internal/config/`)

TOML-based configuration with:

- **Limits**: max sessions (5), max agents/session (50), max workgroup depth (3), max agents/workgroup (10), max total agents (100)
- **Runtime**: default driver, supervisor/advisor bootstrap configs, driver configs
- **Dashboard**: port (2123), host, terminal dimensions, buffer size
- **Meta**: learning behavior flags
- **Roles**: Markdown frontmatter files with draft/approved workflow
- **Playbooks**: Markdown files with draft/approved workflow

## Key Types & Interfaces

### CLI Layer

```go
// Central dependency injection for all daemon commands
type daemonCommandDeps struct {
    resolveHomePaths          func() (config.HomePaths, error)
    newKernel                 func(home, logWriter, outputFormat) (kernelWaiter, error)
    spawnChild                func(home) (daemonChildProcess, error)
    discoverDashboardDaemon   func(home) (*kernel.DaemonInfo, error)
    readDaemonInfo            func(path) (*kernel.DaemonInfo, error)
    discoverDaemon            func(home) (*daemonDiscovery, error)
    newClient                 func(socketPath) (daemonAPIClient, error)
    newSkillRegistry          func(home, workspace) (skillRegistry, error)
    newSkillMarketplaceClient func() skillMarketplaceClient
    resolveWorkspace          func(string) (string, error)
    getenv                    func(string) string
    now                       func() time.Time
}

// Full daemon API surface (28 methods)
type daemonAPIClient interface {
    KernelStatus(ctx) (KernelStatusResponse, error)
    ShutdownKernel(ctx) (KernelStatusResponse, error)
    StartSession(ctx, SessionStartRequest) (SessionSummaryResponse, error)
    ListSessions(ctx, includeHistorical) ([]SessionSummaryResponse, error)
    SessionStatus(ctx, ref) (SessionDetailResponse, error)
    StopSession(ctx, ref) (SessionSummaryResponse, error)
    ResumeSession(ctx, ref) (SessionSummaryResponse, error)
    CreateWorkgroup(ctx, sessionRef, WorkgroupCreateRequest) (WorkgroupResponse, error)
    ListWorkgroups(ctx, sessionRef) ([]WorkgroupResponse, error)
    DestroyWorkgroup(ctx, sessionRef, workgroupRef) (WorkgroupResponse, error)
    Topology(ctx, sessionRef) (TopologySnapshot, error)
    SpawnAgent(ctx, sessionRef, AgentSpawnRequest) (SessionAgentResponse, error)
    KillAgent(ctx, sessionRef, agentRef) (SessionAgentResponse, error)
    AttachAgent(ctx, sessionRef, agentRef) (io.ReadCloser, error)
    SendMessage / BroadcastMessage / EscalateMessage(ctx, sessionRef, MessageCommandRequest) (MessageDeliveryResponse, error)
    ReadBlackboard / AppendBlackboard(ctx, sessionRef, ...) (...)
    Context(ctx, sessionRef, callerAgent) (ContextResponse, error)
    ReadAgentStatus / UpdateAgentStatus(ctx, sessionRef, ...) (StatusEntry, error)
    CompleteAgent(ctx, sessionRef, DoneCommandRequest) (StatusEntry, error)
    ReadEvents(ctx, sessionRef, EventsReadOptions) ([]EventEntry, error)
}

// Agent identity from environment
type agentEnvironment struct {
    AgentID, AgentName, Session, SessionID    string
    Workgroup, WorkgroupID, SocketPath, Workspace string
}
```

### Kernel Layer

```go
type Kernel struct {
    Config          *config.Config
    NATS            *transport.EmbeddedNATS
    UDS             *transport.UDSBridge
    HTTP            *http.Server
    HomePaths       config.HomePaths
    RoleCatalog     RoleCatalogStore
    DriverRegistry  DriverRegistryStore
    PromptTemplates map[string]string
    skillRegistry   *skills.Registry
    SessionManager  *SessionManager
    Logger          *slog.Logger
    DaemonLock      *flock.Flock
    DaemonInfo      *DaemonInfo
    DashboardAddr   string
    // ... lifecycle fields
}

type Session struct {
    ID, Name, Goal    string
    State             SessionState
    Workspace         string
    Config            config.Config
    RoleCatalog       RoleCatalogStore
    Store             *state.Store
    AgentRegistry     AgentRegistryStore
    WorkgroupRegistry WorkgroupRegistryStore
    PtyManager        *internalpty.Manager
    Supervisor        *suture.Supervisor
    WsHub             *WsHub
    WorkgroupManager  *WorkgroupManager
    ResilienceManager *ResilienceManager
    HealthChecker     *HealthChecker
    // ... sync fields
}

type AgentDriver interface {
    Name() string
    Start(ctx, StartOpts) (*AgentProcess, error)
    SendMessage(ctx, proc, msg) error
    Stop(ctx, proc) error
    BuildHookConfig(agentName, hookEndpoint) (*HookConfig, error)
    ParseHookEvent(rawPayload) (*HookEvent, error)
    HealthCheck(ctx, proc) (AgentHealth, error)
    DetectReady(ctx, proc) error
}

type AgentInfo struct {
    ID, Name, Role, Type, Workgroup, Model, State string
    Driver  AgentDriver
    Process *AgentProcess
    Channel chan Message
    Breaker *gobreaker.CircuitBreaker
}
```

### Skills Layer

```go
type Skill struct {
    Meta     SkillMeta       // name, description, license, compatibility, allowed-tools, metadata
    Content  string          // Markdown body
    Source   SkillSource     // bundled/user/.agents/workspace/clawhub
    Dir      string
    FilePath string
    Enabled  bool
    Warnings []Warning
}

type Registry struct {
    // Thread-safe, freezable, version-tracked
    LoadAll(LoadConfig) error
    Get(name) (*Skill, bool)
    List() []*Skill
    Snapshot(SnapshotFilter) *SkillSnapshot
    Freeze()
}

type Client struct {
    Search(ctx, query) ([]SkillListing, error)
    Install(ctx, slug, targetDir) error
}
```

## Gaps & Opportunities

### 1. Human Output Coverage Is Incomplete

Several commands lack a human renderer and fall through to TOON:

- `agh install` -- Uses `renderInstallResults()` which only returns TOON; passes `nil` for humanFn
- `agh skill list/info/create/install/remove/search` -- All pass `nil` for humanFn
- `agh playbooks approve` -- Passes `nil` for humanFn
- `agh roles approve` -- Passes `nil` for humanFn

**Opportunity**: Implement `human.Render*` functions for all skill commands to give interactive users styled tables and key-value views instead of raw TOON format.

### 2. Only One Bundled Skill

The bundled skills directory (`internal/skills/bundled/skills/`) contains only `code-review`. Given the infrastructure supports a full catalog with XML injection into system prompts, this is underutilized.

**Opportunity**: Add more bundled skills covering common agent workflows (e.g., task decomposition, test writing, architecture review, PR creation, debugging).

### 3. No Skill Versioning or Update Mechanism

Skills loaded from ClawHub or user directories have no version tracking. There is no `agh skill update` command, and no way to check if a newer version is available.

**Opportunity**: Add version fields to SKILL.md frontmatter, implement `agh skill update` and `agh skill outdated`.

### 4. Hook Command Is Hidden and Uses Separate Transport

`agh hook-event` uses a different transport mechanism (NATS via `transport.UDSClient`) rather than the standard HTTP-over-UDS used by all other commands. It also has its own `hookCommandDeps` injection struct separate from `daemonCommandDeps`.

**Opportunity**: Evaluate whether hooks could use the standard HTTP API for consistency, or document why the separate NATS path is necessary.

### 5. No Interactive/TUI Mode

The CLI is purely non-interactive. There is no watch mode, no live dashboard in the terminal, and no interactive session management. The `attach` command streams raw PTY output but has no interactive input.

**Opportunity**: Consider a `agh watch` or `agh tui` command using bubbletea (already a dependency via lipgloss) for real-time monitoring.

### 6. Session Auto-Resolution Could Be Smarter

The `resolveRuntimeSession()` function only auto-selects when exactly one session is running. With zero or 2+ sessions, it returns an error requiring the `--session` flag.

**Opportunity**: Support a "default session" concept or workspace-based session affinity so agents don't always need explicit session references.

### 7. No Config Init/Edit Command

There is no `agh config init` or `agh config edit` command. Users must manually create/edit `~/.agh/config.toml`.

**Opportunity**: Add `agh config init`, `agh config show`, `agh config set <key> <value>` commands.

### 8. No Health/Diagnostics Command

While the kernel has health checking infrastructure, there is no `agh doctor` or `agh health` command to diagnose common issues (missing drivers, stale lock files, configuration errors).

**Opportunity**: Add a diagnostics command that checks driver availability, config validity, daemon health, and connectivity.

### 9. Workspace Discovery Happens at CLI Level

Both the CLI (`defaultSkillLoadConfig`) and kernel (`skillLoadConfig`) have nearly identical skill loading logic. The workspace root resolution also happens in both layers.

**Opportunity**: Deduplicate the skill loading config construction into a shared function.

### 10. Error Messages Are CLI-Aware but Not User-Friendly

Errors are wrapped with `userCommandError()` which just prepends "error: " to the message. There is no structured error rendering, no suggestions for next steps, and no colorized error output.

**Opportunity**: Create a richer error rendering system that:

- Colorizes error messages in human mode
- Suggests next commands (e.g., "Run 'agh start' first")
- Distinguishes between user errors and system errors

### 11. No Completion/Autocomplete Support

There are no shell completion generators despite Cobra having built-in support via `cobra.Command.GenBashCompletion()` etc.

**Opportunity**: Add `agh completion bash/zsh/fish` command.

### 12. ClawHub API Is Hardcoded

The ClawHub base URL is hardcoded to `https://clawhub.ai/api/v1`. There is no configuration option to change it, which limits testing and self-hosting.

**Opportunity**: Make the ClawHub base URL configurable via config file or environment variable.

### 13. Skill Content Verification Is Regex-Based Only

The prompt injection detection in `verify.go` uses simple regex patterns. While effective for obvious attacks, it could miss sophisticated obfuscation techniques.

**Opportunity**: Consider supplementing regex checks with structural analysis (e.g., checking for encoded content, unicode homoglyphs, invisible characters).

### 14. No Skill Dependencies or Composition

Skills are independent units with no way to declare dependencies on other skills or compose multiple skills into workflows.

**Opportunity**: Add a `depends` or `includes` field to SKILL.md frontmatter for skill composition.

### 15. Dashboard Command Is Read-Only

`agh dashboard` only displays the URL and status. It does not open the browser or provide any management capabilities.

**Opportunity**: Add `--open` flag to launch the default browser, or integrate dashboard management features.

### 16. No Log Streaming Command

While logs are written to `~/.agh/agh.log`, there is no `agh logs` command to tail daemon logs from the CLI. Users must manually `tail -f` the log file.

**Opportunity**: Add `agh logs [--follow]` command.

### 17. Role Type Validation Is Scattered

Valid role types ("master", "worker", "advisor", "reviewer", "researcher") are defined in both `internal/config/config.go` (via `validRoleTypes` map) and `internal/cli/roles.go` (via `ensureValidRoleType` switch). The kernel's `types.go` also references these implicitly.

**Opportunity**: Centralize role type validation in the config package and reference it from all layers.

### 18. Prompt Templates Are Hard-Coded at Boot

The kernel loads prompt templates for all five agent types at boot (`loadPromptTemplates()`). These cannot be customized or overridden per-session or per-workspace.

**Opportunity**: Support workspace-level prompt template overrides in `.agh/prompts/`.
