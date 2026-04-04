# TechSpec: AGH v2 — Agent Operating System

## Executive Summary

AGH v2 is a complete rewrite of the AGH daemon — a Go single-binary that manages AI agent sessions via ACP (Agent Client Protocol). The daemon spawns ACP-compatible agents (Claude Code, Codex, Gemini CLI, etc.) as subprocesses, communicates with them via JSON-RPC over stdio, persists all events in SQLite, and exposes interfaces via HTTP/SSE (for web UI) and UDS (for CLI).

Key architectural decisions: **Pragmatic Flat with Discipline** (11 packages under `internal/`, no event bus, CI-enforceable boundaries). ACP only internally (daemon as client); custom HTTP/SSE API externally. Agents defined as self-contained directories (`~/.agh/agents/<name>/AGENT.md`). Built-in provider registry with hardcoded ACP commands. Dual SQLite storage (per-session + global).

Primary trade-off: simplicity over premature extensibility. Features like memory, skills, dream consolidation, and agent networking are deferred to future phases. Frontend is mentioned but out of scope for this spec.

### Adversarial Review Amendments

This spec incorporates fixes from an adversarial review that identified 14 issues. Key changes:
- **Write path clarified**: session/ owns per-session DB writes, observe/ owns global DB writes via Notifier. Boot-time reconciliation for crash recovery.
- **Resume protocol defined**: Session stores ACP session ID. Resume attempts `session/load`, falls back to `session/new`.
- **Streaming contract expanded**: Added SSE endpoints for session streams and observability follow mode.
- **Schemas enriched**: Added sequence numbers, turn IDs, token usage tracking.
- **Permission model specified**: Precedence rules, static policy resolution, audit trail.
- **MCP server support added**: Both AGENT.md and config.toml support `mcp_servers`.
- **Failure handling section added**: Daemon restart recovery, orphan cleanup, stale lock detection.

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Go Binary (agh)                          │
├─────────────────────────────────────────────────────────────────┤
│  CLI (cobra)                                                    │
│    ├── daemon start/stop/status                                 │
│    ├── session new/list/stop/status/resume/wait/prompt/events   │
│    ├── agent list/info                                          │
│    ├── observe events/health                                    │
│    └── whoami                                                   │
├─────────────────────────────────────────────────────────────────┤
│  Daemon (composition root)                                      │
│    ├── Wires all packages                                       │
│    ├── Daemon lock, boot sequence, graceful shutdown            │
│    └── Signal handling (SIGINT, SIGTERM)                        │
├──────────────┬──────────────┬───────────────────────────────────┤
│  HTTP/SSE    │  UDS Server  │  Session Manager                  │
│  Server      │  (CLI IPC)   │    ├── Session lifecycle          │
│  (Web UI)    │              │    ├── Agent process management   │
│              │              │    └── Notifier fan-out            │
├──────────────┴──────────────┼───────────────────────────────────┤
│  ACP Client                 │  Store (SQLite)                   │
│    ├── Subprocess spawn     │    ├── Per-session events.db      │
│    ├── JSON-RPC over stdio  │    ├── Global agh.db              │
│    └── Bidirectional msgs   │    └── Schema + migrations        │
├─────────────────────────────┼───────────────────────────────────┤
│  Config                     │  Observe                          │
│    ├── TOML 2-level merge   │    ├── Event recording            │
│    ├── Agent def parsing    │    ├── Health metrics             │
│    ├── Provider registry    │    └── Query engine               │
│    └── Home paths           │                                   │
└─────────────────────────────┴───────────────────────────────────┘
         │                              │
    ACP stdio                      ACP stdio
         │                              │
   ┌─────┴─────┐                 ┌──────┴──────┐
   │claude --acp│                │codex --acp  │
   └────────────┘                └─────────────┘
```

### Data Flow

1. **CLI → Daemon**: User runs `agh session new --agent coder`. CLI connects to daemon via UDS, sends request.
2. **Daemon → Session**: Session Manager creates a new Session, opens per-session SQLite DB, registers session in global DB.
3. **Session → ACP**: Session resolves agent definition (AGENT.md) and provider command. ACP client spawns subprocess (`npx -y @agentclientprotocol/claude-agent-acp`), performs `initialize` handshake, calls `session/new` with `cwd` and `mcpServers`. Stores returned ACP `sessionId` for resume support.
4. **User → Daemon → Agent**: User sends `agh session prompt <id> "message"`. Daemon routes to session, session calls `session/prompt` on ACP client. Agent processes and streams `session/update` notifications back.
5. **Agent → Daemon**: During processing, agent sends `fs/readTextFile`, `terminal/create`, `request_permission` requests. ACP client handles these locally (reads files from disk, executes commands, applies permission policy per precedence rules).
6. **Daemon → Observers**: Session's Notifier fans out events to Observe (global DB writes) and HTTP/SSE (web client streaming). Session writes detailed events to its own per-session DB directly.

### Write Path Ownership

To ensure crash consistency, write ownership is clearly split:
- **session/** writes to per-session `events.db` (detailed events, full ACP payloads)
- **observe/** writes to global `agh.db` (session index, event summaries, token stats) via Notifier callbacks
- **session/** writes `meta.json` atomically (write to temp file, rename)
- **Boot-time reconciliation**: On daemon start, scan `~/.agh/sessions/` directory and reconcile with `agh.db` — any session directory not in global DB gets indexed, any global DB entry without a directory gets marked as orphaned.

### Package Dependency Graph

```
cli/ ────────────→ daemon/ ──→ session/ ──→ acp/ (AgentDriver interface)
                      │            │
                      │            └──→ store/ (EventRecorder interface)
                      │
                      ├──→ httpapi/ ──→ session/ (read-only Manager interface)
                      │                    └──→ store/ (queries)
                      │
                      ├──→ udsapi/ ──→ session/ (Manager interface)
                      │
                      ├──→ observe/ ──→ store/ (global DB writes)
                      │
                      ├──→ store/
                      ├──→ config/
                      └──→ acp/
```

Rules:
- **Designed for incremental extension** — every new capability (memory, skills, agent networking) arrives as a new package wired into `daemon/`, without modifying existing packages. Public APIs of each package must be stable: add, don't change.
- Dependencies flow downward only
- No package imports `daemon/`, `httpapi/`, `udsapi/`, or `cli/`
- `daemon/` is the sole composition root
- Use small interfaces and dependency injection so future features plug in rather than fork existing code
- CI grep checks enforce these boundaries

## Implementation Design

### Core Interfaces

```go
// session/interfaces.go — what session needs from ACP
type AgentDriver interface {
    Start(ctx context.Context, opts StartOpts) (*AgentProcess, error)
    Prompt(ctx context.Context, proc *AgentProcess, req PromptRequest) (<-chan AgentEvent, error)
    Cancel(ctx context.Context, proc *AgentProcess) error
    Stop(ctx context.Context, proc *AgentProcess) error
}
```

```go
// session/interfaces.go — what session needs from storage
type EventRecorder interface {
    Record(ctx context.Context, ev SessionEvent) error
    Query(ctx context.Context, opts EventQuery) ([]SessionEvent, error)
}
```

```go
// session/interfaces.go — fan-out notifications
type Notifier interface {
    OnSessionCreated(ctx context.Context, s *Session)
    OnSessionStopped(ctx context.Context, s *Session)
    OnAgentEvent(ctx context.Context, sessionID string, ev AgentEvent)
}
```

```go
// session/manager.go — session manager with functional options
type Manager struct { /* ... */ }

func NewManager(opts ...Option) (*Manager, error)
func (m *Manager) Create(ctx context.Context, opts CreateOpts) (*Session, error)
func (m *Manager) Stop(ctx context.Context, id string) error
func (m *Manager) Resume(ctx context.Context, id string) (*Session, error)
func (m *Manager) Prompt(ctx context.Context, id string, msg string) (<-chan AgentEvent, error)
func (m *Manager) Get(id string) (*Session, bool)
func (m *Manager) List() []*SessionInfo
```

```go
// daemon/daemon.go — composition root
type Daemon struct { /* ... */ }

func New(opts ...Option) (*Daemon, error)
func (d *Daemon) Run(ctx context.Context) error
func (d *Daemon) Shutdown(ctx context.Context) error
```

### Data Models

**Agent Definition** (parsed from AGENT.md frontmatter):
```go
type AgentDef struct {
    Name        string      `yaml:"name"`
    Provider    string      `yaml:"provider"`
    Command     string      `yaml:"command,omitempty"`     // Override provider command
    Model       string      `yaml:"model,omitempty"`
    Tools       []string    `yaml:"tools,omitempty"`
    Permissions string      `yaml:"permissions,omitempty"`
    MCPServers  []MCPServer `yaml:"mcp_servers,omitempty"` // MCP servers for this agent
    Prompt      string      // Markdown body (not from frontmatter)
}

type MCPServer struct {
    Name      string            `yaml:"name" toml:"name"`
    Command   string            `yaml:"command" toml:"command"`
    Args      []string          `yaml:"args,omitempty" toml:"args,omitempty"`
    Env       map[string]string `yaml:"env,omitempty" toml:"env,omitempty"`
}
```

**Provider Config**:
```go
type ProviderConfig struct {
    Command      string      `toml:"command"`
    DefaultModel string      `toml:"default_model"`
    APIKeyEnv    string      `toml:"api_key_env"`
    MCPServers   []MCPServer `toml:"mcp_servers,omitempty"` // Default MCP servers
}
```

Resolution chain for all fields: AGENT.md override → Provider config.toml → Built-in defaults. For `mcp_servers`, agent-level and provider-level are **merged** (not replaced).

**Session**:
```go
type Session struct {
    ID           string
    Name         string
    AgentName    string
    Workspace    string
    State        SessionState  // starting | active | stopping | stopped
    ACPSessionID string        // ACP wire session ID (for resume)
    ACPCaps      ACPCaps       // Agent capabilities snapshot
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type ACPCaps struct {
    SupportsLoadSession bool
    SupportedModes      []string
    SupportedModels     []string
}

type SessionState string
const (
    StateStarting SessionState = "starting"
    StateActive   SessionState = "active"
    StateStopping SessionState = "stopping"
    StateStopped  SessionState = "stopped"
)
```

**Session Resume Flow**:
1. Load session metadata from `meta.json`
2. Spawn agent process (same provider command)
3. Perform ACP `initialize` handshake
4. If agent supports `loadSession` (from ACPCaps): call `session/load` with stored `ACPSessionID`
5. If `session/load` fails or unsupported: call `session/new` (loses agent-side history but our `events.db` is intact)
6. Update `ACPSessionID` with new value if session/new was used

**Session Event** (persisted in SQLite):
```go
type SessionEvent struct {
    ID           string
    SessionID    string
    Sequence     int64      // Monotonic sequence number within session
    TurnID       string     // Groups events within a prompt/response cycle
    Type         string     // agent_message, tool_call, tool_result, thought, error, permission, system
    AgentName    string
    Content      string     // JSON payload
    Timestamp    time.Time
}

// TokenUsage captured from ACP PromptResponse.usage (per-turn).
// All fields nullable — not all agents report token usage.
// usage_update notification is UNSTABLE in ACP spec; we capture
// PromptResponse.usage as the primary source.
type TokenUsage struct {
    TurnID           string
    InputTokens      *int64  // From PromptResponse.usage.inputTokens
    OutputTokens     *int64  // From PromptResponse.usage.outputTokens
    TotalTokens      *int64  // From PromptResponse.usage.totalTokens
    ThoughtTokens    *int64  // Optional: reasoning tokens
    CacheReadTokens  *int64  // Optional: cached read tokens
    CacheWriteTokens *int64  // Optional: cached write tokens
    ContextUsed      *int64  // From usage_update.used (if available)
    ContextSize      *int64  // From usage_update.size (if available)
    CostAmount       *float64 // From usage_update.cost.amount (if available)
    CostCurrency     *string  // From usage_update.cost.currency (if available)
}
```

**SQLite Schemas**:

Per-session `events.db`:
```sql
CREATE TABLE events (
    id            TEXT PRIMARY KEY,
    sequence      INTEGER NOT NULL,
    turn_id       TEXT NOT NULL,
    type          TEXT NOT NULL,
    agent_name    TEXT NOT NULL,
    content       TEXT NOT NULL,
    timestamp     TEXT NOT NULL
);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_sequence ON events(sequence);
CREATE INDEX idx_events_turn ON events(turn_id);

-- Token usage per turn. All token fields nullable — ACP usage reporting
-- is UNSTABLE and not all agents provide it. PromptResponse.usage is the
-- primary source; usage_update notification is secondary.
CREATE TABLE token_usage (
    turn_id            TEXT PRIMARY KEY,
    input_tokens       INTEGER,
    output_tokens      INTEGER,
    total_tokens       INTEGER,
    thought_tokens     INTEGER,
    cache_read_tokens  INTEGER,
    cache_write_tokens INTEGER,
    context_used       INTEGER,
    context_size       INTEGER,
    cost_amount        REAL,
    cost_currency      TEXT,
    timestamp          TEXT NOT NULL
);
CREATE INDEX idx_usage_timestamp ON token_usage(timestamp);
```

Global `agh.db`:
```sql
CREATE TABLE sessions (
    id              TEXT PRIMARY KEY,
    name            TEXT,
    agent_name      TEXT NOT NULL,
    workspace       TEXT NOT NULL,
    state           TEXT NOT NULL,
    acp_session_id  TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL
);

CREATE TABLE event_summaries (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id),
    type        TEXT NOT NULL,
    agent_name  TEXT NOT NULL,
    summary     TEXT,
    timestamp   TEXT NOT NULL
);
CREATE INDEX idx_summaries_session ON event_summaries(session_id);
CREATE INDEX idx_summaries_type ON event_summaries(type);
CREATE INDEX idx_summaries_timestamp ON event_summaries(timestamp);

-- Aggregated token stats per session. Updated via Notifier on each turn.
-- Nullable because not all agents report usage (ACP usage is UNSTABLE).
CREATE TABLE token_stats (
    id            TEXT PRIMARY KEY,
    session_id    TEXT NOT NULL REFERENCES sessions(id),
    agent_name    TEXT NOT NULL,
    input_tokens  INTEGER,
    output_tokens INTEGER,
    total_tokens  INTEGER,
    total_cost    REAL,
    cost_currency TEXT,
    turn_count    INTEGER NOT NULL DEFAULT 0,
    updated_at    TEXT NOT NULL
);
CREATE INDEX idx_token_stats_session ON token_stats(session_id);

CREATE TABLE permission_log (
    id           TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL REFERENCES sessions(id),
    agent_name   TEXT NOT NULL,
    action       TEXT NOT NULL,
    resource     TEXT NOT NULL,
    decision     TEXT NOT NULL,
    policy_used  TEXT NOT NULL,
    timestamp    TEXT NOT NULL
);
CREATE INDEX idx_perm_session ON permission_log(session_id);
```

### API Endpoints

**HTTP/SSE API** (for web UI):

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sessions` | List all sessions |
| POST | `/api/sessions` | Create new session |
| GET | `/api/sessions/:id` | Get session details |
| DELETE | `/api/sessions/:id` | Stop session |
| POST | `/api/sessions/:id/resume` | Resume stopped session |
| POST | `/api/sessions/:id/prompt` | Send prompt (returns SSE stream) |
| GET | `/api/sessions/:id/events` | Query session events |
| GET | `/api/sessions/:id/history` | Full conversation history (turn-structured) |
| GET | `/api/sessions/:id/stream` | SSE stream of all session events (for `--follow`) |
| POST | `/api/sessions/:id/approve` | Approve/deny a pending permission request |
| GET | `/api/agents` | List available agents |
| GET | `/api/agents/:name` | Get agent info |
| GET | `/api/observe/events` | Cross-session event query |
| GET | `/api/observe/events/stream` | SSE stream of cross-session events (for `--follow`) |
| GET | `/api/observe/health` | Daemon health metrics |
| GET | `/api/daemon/status` | Daemon status |

**SSE Streaming Contracts**:

`POST /api/sessions/:id/prompt` — Prompt-scoped stream (ends when prompt completes):
- Content-Type: `text/event-stream`
- Events: `agent_message`, `tool_call`, `tool_result`, `thought`, `permission_request`, `error`, `done`
- Compatible with Vercel AI SDK `x-vercel-ai-ui-message-stream: v1` format (future web UI spec)

`GET /api/sessions/:id/stream` — Session-wide stream (long-lived, for `--follow` and web UI):
- Content-Type: `text/event-stream`
- Events: all session events including out-of-band state changes
- Supports `Last-Event-ID` header for reconnection/replay
- Each event includes `id` (sequence number) for resumption

`GET /api/observe/events/stream` — Cross-session stream (for global `--follow`):
- Content-Type: `text/event-stream`
- Events: event summaries from all active sessions
- Supports `Last-Event-ID` for reconnection

**UDS API** (for CLI):
- HTTP over Unix Domain Socket at `~/.agh/daemon.sock` (Gin with unix listener)
- Same endpoints as HTTP API — same request/response format, same SSE streaming
- CLI `--follow` commands use SSE over UDS (long-lived HTTP connection)
- CLI `wait` command uses `GET /api/sessions/:id/stream` and blocks until `session_stopped` event

### Permission Model

Precedence: agent-level `permissions` (AGENT.md) > global `[permissions].mode` (config.toml)

| Mode | `fs/readTextFile` | `fs/writeTextFile` | `terminal/create` | `request_permission` |
|------|---|---|---|---|
| `deny-all` | Deny | Deny | Deny | Deny |
| `approve-reads` | Allow | Deny | Deny | Deny |
| `approve-all` | Allow | Allow | Allow | Allow |

For v1, all decisions are **static** (no interactive approval flow). The daemon applies the policy and responds immediately. All decisions are logged to `permission_log` in global DB for audit.

Path boundaries: agents can only access files within the session's `cwd` and its subdirectories. Paths outside `cwd` are denied regardless of permission mode.

Future: interactive approval via `POST /api/sessions/:id/approve` endpoint (web UI sends user decision, daemon forwards to ACP client).

### Filesystem Layout

```
~/.agh/                             # AGH_HOME (overridable via env var)
├── agents/                         # Agent definitions
│   ├── coder/AGENT.md
│   └── researcher/AGENT.md
├── sessions/                       # Session data
│   ├── <xid>/
│   │   ├── events.db              # Per-session SQLite
│   │   └── meta.json              # Quick metadata
│   └── <xid>/
│       ├── events.db
│       └── meta.json
├── agh.db                          # Global SQLite
├── config.toml                     # Global config
├── daemon.sock                     # UDS socket
├── daemon.lock                     # File lock (flock)
├── daemon.json                     # Daemon info (PID, port, started_at)
└── logs/
    └── agh.log                     # Structured log file

.agh/                               # Workspace overlay (project-specific)
└── config.toml                     # Merged on top of global config
```

### Configuration

```toml
# ~/.agh/config.toml

[daemon]
socket = "~/.agh/daemon.sock"

[http]
host = "localhost"
port = 2123

[defaults]
agent = "coder"

[limits]
max_sessions = 10
max_concurrent_agents = 20

[permissions]
mode = "approve-reads"  # deny-all | approve-reads | approve-all

[providers.claude]
default_model = "claude-sonnet-4-20250514"
api_key_env = "ANTHROPIC_API_KEY"

[providers.codex]
default_model = "gpt-4o"
api_key_env = "OPENAI_API_KEY"

[providers.gemini]
default_model = "gemini-2.5-pro"
api_key_env = "GEMINI_API_KEY"

# MCP servers available to all agents using this provider
# [[providers.claude.mcp_servers]]
# name = "github"
# command = "npx"
# args = ["-y", "@modelcontextprotocol/server-github"]
# env = { GITHUB_PERSONAL_ACCESS_TOKEN = "" }

[observability]
enabled = true
retention_days = 7
max_global_bytes = 1073741824      # 1GB

[observability.transcripts]
enabled = true
segment_bytes = 1048576            # 1MB
max_bytes_per_session = 268435456  # 256MB

[log]
level = "info"
```

2-level merge: workspace `.agh/config.toml` is deep-merged on top of global `~/.agh/config.toml`. Built-in defaults → Global config → Workspace config → CLI flags.

`AGH_HOME` env var overrides the default `~/.agh` location.

## Integration Points

### ACP Agent Subprocesses

- **Protocol**: JSON-RPC 2.0 over stdio (newline-delimited JSON)
- **SDK**: `github.com/coder/acp-go-sdk` (community library, pre-v1, but actively maintained by Coder)
- **Lifecycle**: `initialize` → `session/new(cwd, mcpServers)` → `session/prompt` (repeatable) → `session/load` (on resume) → process termination
- **Bidirectional**: Agent sends `fs/readTextFile`, `fs/writeTextFile`, `terminal/create`, `terminal/output`, `terminal/waitForExit`, `terminal/kill`, `request_permission` — daemon handles all locally with permission policy enforcement and path sandboxing
- **MCP passthrough**: `session/new` includes `mcpServers` merged from AGENT.md + Provider config
- **Error handling**: Process crash detected via `cmd.Wait()`. ACP errors returned via JSON-RPC error responses. Session marked as stopped on unrecoverable errors. Orphaned processes cleaned on daemon restart.
- **SDK risk mitigation**: If `acp-go-sdk` lags or breaks, the ACP wire protocol is simple JSON-RPC — a minimal client can be implemented in ~500 lines of Go as fallback

### Web UI (out of scope — separate spec)

- Connects to daemon via HTTP/SSE endpoints listed above
- Uses Vercel AI SDK with `useChat` component pointing to `/api/sessions/:id/prompt`
- SSE stream format compatible with `x-vercel-ai-ui-message-stream: v1`

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| Entire codebase | New | Complete rewrite — all existing code replaced | Implement from scratch |
| `internal/kernel/` | Deprecated | Replaced by `daemon/` + `session/` + `acp/` | Remove after rewrite |
| `internal/drivers/` | Deprecated | Replaced by `acp/` using ACP protocol | Remove |
| `internal/transport/` | Deprecated | NATS + UDS replaced by `udsapi/` (stdlib) | Remove |
| `internal/pty/` | Deprecated | PTY management replaced by ACP stdio | Remove |
| `internal/dashboard/` | Deprecated | Web dashboard replaced by simple `httpapi/` | Remove |
| `web/` | Deprecated | Svelte dashboard replaced by new web UI (separate spec) | Remove |
| `internal/config/` | Modified | Simplified — remove driver/role/playbook config, add providers | Rewrite |
| `internal/state/` | Modified | Simplified schema, same SQLite approach | Rewrite as `store/` |
| `internal/cli/` | Modified | Reduced from 26+ to ~15 commands | Rewrite |
| `internal/observability/` | Modified | Same concept, simpler implementation | Rewrite as `observe/` |

## Testing Approach

### Unit Tests

- **config/**: TOML parsing, 2-level merge, agent definition loading, validation
- **acp/**: JSON-RPC message serialization/deserialization, protocol state machine
- **session/**: State machine transitions, session creation/stop/resume logic
- **store/**: SQLite operations with `t.TempDir()` databases, event queries with filters
- **observe/**: Event recording, health metric computation
- Mock strategy: interfaces defined in consuming packages, mock only at package boundaries

### Integration Tests

- **ACP round-trip**: Spawn a mock ACP server (simple Go binary that speaks ACP), verify full `initialize` → `session/new` → `session/prompt` → `session/update` flow
- **Daemon lifecycle**: Start daemon, create session, send prompt, verify events persisted, stop session, stop daemon
- **CLI integration**: Run CLI commands against a running test daemon, verify output format (human, json, toon)
- **Smoke tests**: Document manual smoke test procedure for each built-in provider (Claude, Codex, Gemini). Not automated in CI — real agents require auth and are non-deterministic.
- **HTTP/SSE**: Send prompt via HTTP, verify SSE stream contains expected events

### Test Conventions

- Table-driven tests with `t.Run` subtests
- `t.Parallel()` for independent subtests
- `t.TempDir()` for filesystem isolation
- `-race` flag on all test runs
- 80%+ coverage target per package

## Development Sequencing

### Build Order

1. **config/** — TOML loading, validation, home paths, agent def parsing. Zero internal dependencies.
2. **store/** — SQLite wrapper (per-session + global), schema, migrations. Depends only on stdlib + sqlite driver.
3. **acp/** — ACP client: subprocess spawn, JSON-RPC protocol, permission handling. Depends on `acp-go-sdk`.
4. **session/** — Session Manager wiring `acp/` and `store/` via interfaces. Core orchestration. Resume logic.
5. **observe/** — Event recording using `store/`. Implements session Notifier. Health metrics.
6. **daemon/** — Composition root wiring everything. Lock, boot, shutdown, reconciliation.
7. **udsapi/** — UDS server (HTTP over unix socket) exposing session Manager. CLI can now work.
8. **cli/** — Cobra commands talking to daemon via UDS. Output formatters (human, json, toon).
9. **httpapi/** — HTTP/SSE server exposing session Manager. SSE streaming. Web UI can now connect.

Each step is independently testable and shippable.

Note: `logger/` and `version/` are trivial utility packages implemented alongside step 1. Total: **11 packages** under `internal/` plus `cmd/agh/`.

### Technical Dependencies

- `github.com/coder/acp-go-sdk` — ACP client protocol implementation
- `modernc.org/sqlite` — Pure Go SQLite (no CGo)
- `github.com/BurntSushi/toml` — TOML parsing
- `github.com/spf13/cobra` — CLI framework
- `github.com/rs/xid` — Collision-free ID generation
- `github.com/gofrs/flock` — File locking for daemon singleton
- `github.com/joho/godotenv` — .env loading
- `github.com/gin-gonic/gin` — HTTP framework (routing, middleware, SSE)
- stdlib `log/slog` — Structured logging

No NATS, no suture, no gobreaker. Gin for HTTP. Minimal dependency footprint.

## Monitoring and Observability

### Key Metrics

- Active sessions count
- Active agent processes count
- Events recorded per session
- ACP message latency (prompt → first response)
- Agent process health (running, crashed, stopped)
- SQLite DB sizes (per-session and global)
- Daemon uptime

### Log Events

All structured via `slog` with consistent fields:
- `session_id`, `agent_name` — context fields on all session-related logs
- `event_type` — for event recording (agent_message, tool_call, tool_result, thought, error)
- `acp_method` — for ACP protocol messages
- `duration_ms` — for operation timing
- `error` — for error conditions

### Health Endpoint

`GET /api/observe/health` returns:
```json
{
  "status": "ok",
  "uptime_seconds": 3600,
  "active_sessions": 3,
  "active_agents": 3,
  "global_db_size_bytes": 1048576,
  "version": "0.1.0"
}
```

## Technical Considerations

### Key Decisions

See Architecture Decision Records below for full rationale on each decision.

### Known Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| ACP spec evolves, breaking changes | Medium | Use `acp-go-sdk` which tracks spec. Pin versions. Fallback: minimal JSON-RPC client (~500 LOC). |
| `acp-go-sdk` is community, pre-v1 | Known | SDK is maintained by Coder (large company). ACP wire protocol is simple JSON-RPC — can implement minimal client as fallback. |
| Agent ACP support varies (some use npx adapters) | Known | Built-in provider registry with exact commands per agent. Smoke test docs for each provider. |
| SQLite write contention on busy sessions | Low | WAL mode + single-writer pattern per session |
| Built-in provider commands become stale | Medium | Config.toml override allows users to specify updated commands |
| Package boundaries erode over time | Medium | CI grep checks enforce import rules on every PR |
| Daemon crash with active sessions | Medium | Boot-time reconciliation + orphan process cleanup (see Failure Handling) |
| `npx` unavailable or rate-limited | Low | Document requirement. Users can override with local binary paths in config.toml. |

### Failure Handling

**Daemon restart with active sessions**:
1. On boot, read `daemon.lock` — if stale (PID not running), remove and re-acquire
2. Remove stale `daemon.sock` if exists
3. Scan `~/.agh/sessions/` directory, reconcile with `agh.db`:
   - Sessions in directory but not in DB: index them (crash between per-session write and global write)
   - Sessions in DB marked `active` but no running process: mark as `stopped`
4. Kill orphaned agent subprocesses: scan for processes whose parent PID matches stale daemon PID

**Agent process crash**:
- Detected via `cmd.Wait()` returning non-zero exit
- Session state transitions to `stopped`
- Event recorded: `type: "error"`, content includes exit code and stderr tail
- Notifier fires `OnSessionStopped` — global DB and SSE streams updated

**SQLite corruption**:
- Per-session DB: mark session as corrupted in global DB, create new events.db on resume
- Global DB: on open failure, rename corrupted file to `agh.db.corrupt.<timestamp>`, create fresh DB, rebuild from session directories

**Disk full**:
- SQLite write failures logged via slog
- Session continues operating (events lost but agent interaction not blocked)
- Health endpoint reports disk pressure metric

### Future Phases (out of scope)

- **Phase 2**: Memory system (memdir), dream consolidation, skills, prompt assembly layers
- **Phase 3**: Agent network protocol (discovery, inter-agent messaging, multi-agent collaboration)
- **Separate spec**: Web UI with Vercel AI SDK

Each future phase adds new packages without modifying the core session/acp/store packages.

## Architecture Decision Records

- [ADR-001: Rewrite From Scratch](adrs/adr-001.md) — Complete rewrite instead of refactoring the coupled kernel monolith
- [ADR-002: Pragmatic Flat Architecture With Discipline](adrs/adr-002.md) — 11 Go-idiomatic packages with CI-enforceable boundaries, no event bus
- [ADR-003: ACP Internally, HTTP/SSE Externally](adrs/adr-003.md) — ACP for daemon↔agent, custom HTTP/SSE for web↔daemon
- [ADR-004: Self-Contained Agent Directory With AGENT.md](adrs/adr-004.md) — Agents as directories with frontmatter Markdown
- [ADR-005: Built-In Provider Registry With ACP Commands](adrs/adr-005.md) — Hardcoded provider commands, overridable via config
- [ADR-006: Dual SQLite Storage](adrs/adr-006.md) — Per-session events.db + global agh.db
- [ADR-007: Background Sessions With CLI Prompt](adrs/adr-007.md) — Sessions in background, interaction via CLI prompt and web UI
- [ADR-008: Direct Interfaces and Notifier Pattern](adrs/adr-008.md) — No event bus, direct calls + typed Notifier for fan-out
- [ADR-009: Agent-First Observability](adrs/adr-009.md) — CLI-queryable, structured output, designed for LLM consumption
