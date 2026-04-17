# AGH Project Feature Analysis

## 1. Executive Summary

AGH is an **Agent Operating System** -- a Go single-binary daemon that manages AI agent sessions via the Agent Client Protocol (ACP). It spawns ACP-compatible agents (Claude Code, Codex, Gemini CLI, Copilot, Cursor, Kiro, Pi, and others) as subprocesses, communicates via JSON-RPC over stdio, persists events in SQLite, and exposes interfaces through HTTP/SSE (web UI) and UDS (CLI IPC).

### Core Value Propositions

1. **Unified Agent Management**: One daemon to manage multiple AI agent sessions across different providers
2. **Agent Network Protocol**: A novel peer-to-peer protocol for agent-to-agent communication (AGH's key differentiator)
3. **Persistent Memory**: Dual-scope (global + workspace) memory that survives across sessions with dream consolidation
4. **Full Observability**: Event recording, health metrics, transcript replay, and session history
5. **Extensible Automation**: Scheduled jobs, event-driven triggers, webhook endpoints
6. **Bridge Adapters**: Connect agent sessions to external messaging platforms (Slack, Discord, etc.)
7. **Skills System**: Reusable agent capability modules with a marketplace registry

### Architecture Highlights

- Single-binary, local-first daemon running as a background process
- Pragmatic flat package layout under `internal/` with `daemon/` as the sole composition root
- SQLite for persistence (global catalog + per-session event stores)
- Embedded NATS server for agent network protocol transport
- React 19 SPA web UI (Vite, TanStack Router/Query, Tailwind, shadcn/ui)
- Cobra CLI for all operations, communicating with daemon over Unix Domain Socket

---

## 2. Feature Catalog

### 2.1 Session Management

**Package**: `internal/session/`

Sessions are the core runtime unit. Each session wraps a running ACP agent subprocess.

**Key Concepts**:

- **Session Lifecycle**: `starting` -> `active` -> `stopping` -> `stopped` (strict state machine)
- **Session Types**: `user` (interactive), `dream` (memory consolidation), `system` (internal)
- **Turn Sources**: `user` (human-initiated prompt) or `network` (peer-agent initiated prompt)
- **Stop Causes**: `completed`, `user_requested`, `process_exited`, `failed`, `shutdown`, `hook_denied`
- **Stop Reasons**: Classified post-mortem as `completed`, `user_stopped`, `agent_crashed`, `daemon_shutdown`, etc.

**Capabilities**:

- Create new sessions with agent selection, workspace binding, and optional network channel
- Resume stopped sessions from persisted metadata and event history (ACP `session/load`)
- Prompt active sessions and stream back events in real-time
- Permission approval for interactive tool-use requests
- Max concurrent session limits (configurable, default 10)
- Process watchdog: automatic crash detection with error recording
- Network peer lifecycle: auto-join/leave channels on session start/stop
- Hook dispatch at every lifecycle point (pre-create, post-create, pre-resume, post-resume, pre-stop, post-stop)

**Session Manager Options** (functional options pattern):

- `WithDriver` -- ACP runtime driver
- `WithStore` -- per-session event recorder opener
- `WithPromptAssembler` -- startup prompt assembly
- `WithNotifier` -- async event fan-out
- `WithHookSet` -- grouped hook dispatch domains
- `WithSkillRegistry` / `WithMCPResolver` -- skill-driven MCP server injection
- `WithWorkspaceResolver` -- workspace resolution
- `WithMaxSessions` -- configurable limit
- `WithLifecycleContext` -- daemon shutdown propagation

### 2.2 ACP (Agent Client Protocol) Client

**Package**: `internal/acp/`

The ACP client spawns agent subprocesses and brokers JSON-RPC over stdio.

**Key Concepts**:

- **ACP Driver**: Launches subprocesses, performs ACP initialization handshake, creates/loads sessions
- **Agent Process**: Runtime handle wrapping a managed subprocess with stdin/stdout JSON-RPC connection
- **Permission Policy**: Static permission modes applied by the daemon

**Event Types** emitted during prompt turns:
| Event | Description |
|-------|-------------|
| `user_message` | Agent echoes user message chunks |
| `agent_message` | Assistant message chunks |
| `thought` | Assistant thought/reasoning chunks |
| `tool_call` | Tool call start or in-flight update |
| `tool_result` | Tool call completion |
| `plan` | Plan updates |
| `permission` | Permission decision applied |
| `usage` | Token usage metadata |
| `system` | System-level ACP updates |
| `done` | Prompt turn finished |
| `error` | Prompt processing failure |

**ACP Capabilities** reported per session:

- `SupportsLoadSession` -- whether the agent supports session resume
- `SupportedModes` -- available session modes (e.g., `full-access`, `read-only`, `plan`)
- `SupportedModels` -- available models

**Token Usage Tracking**:

- Input/output/total/thought tokens
- Cache read/write tokens
- Context used/size
- Cost amount and currency

### 2.3 Configuration System

**Package**: `internal/config/`

Three-layer TOML configuration with merge semantics.

**Configuration Layers** (applied in order):

1. Built-in defaults
2. Global AGH home config: `~/.agh/config.toml`
3. Workspace overlay: `<workspace>/.agh/config.toml`

**MCP JSON Sidecar**: Both global and workspace directories support `mcp.json` for MCP server declarations alongside TOML config.

**Top-Level Config Sections**:

| Section         | Purpose                                                                      |
| --------------- | ---------------------------------------------------------------------------- |
| `daemon`        | UDS socket path                                                              |
| `http`          | HTTP server host and port (default `localhost:2123`)                         |
| `defaults`      | Default agent name, default provider                                         |
| `limits`        | Max sessions (default 10), max concurrent agents (default 20)                |
| `session`       | Per-session controls (timeout)                                               |
| `permissions`   | Global permission mode: `deny-all`, `approve-reads`, `approve-all` (default) |
| `mcp_servers`   | Global MCP server list                                                       |
| `providers`     | Provider config overrides                                                    |
| `observability` | Event retention, transcript capture settings                                 |
| `log`           | Log level: `debug`, `info`, `warn`, `error`                                  |
| `memory`        | Memory system settings including dream consolidation                         |
| `skills`        | Skill loading, poll interval, marketplace config                             |
| `extensions`    | Extension marketplace settings                                               |
| `automation`    | Automation jobs, triggers, scheduler settings                                |
| `hooks`         | Config-defined hook declarations                                             |
| `network`       | Embedded network runtime settings                                            |

**Built-in Providers** (pre-configured):

| Provider   | Command                                               | Default Model              |
| ---------- | ----------------------------------------------------- | -------------------------- |
| `claude`   | `npx -y @agentclientprotocol/claude-agent-acp@0.24.2` | `claude-sonnet-4-20250514` |
| `codex`    | `npx @zed-industries/codex-acp@0.10.0`                | `gpt-4o`                   |
| `gemini`   | `gemini --acp`                                        | `gemini-2.5-pro`           |
| `opencode` | `npx -y opencode-ai acp`                              | --                         |
| `copilot`  | `copilot --acp --stdio`                               | --                         |
| `cursor`   | `cursor-agent acp`                                    | --                         |
| `kiro`     | `kiro-cli-chat acp`                                   | --                         |
| `pi`       | `npx pi-acp@0.0.22`                                   | --                         |

**Agent Definitions** (`AGENT.md` files):

- Defined in `~/.agh/agents/<name>/AGENT.md` or `<workspace>/.agh/agents/<name>/AGENT.md`
- YAML frontmatter (or TOML) with fields: `name`, `provider`, `command`, `model`, `tools`, `permissions`, `mcp_servers`, `hooks`
- Markdown body becomes the system prompt
- Workspace agents override global agents by name

**Permission Modes**:

- `deny-all` -- deny all tool calls
- `approve-reads` -- allow read operations, block writes (maps to `read-only`/`plan` session modes)
- `approve-all` -- auto-approve everything (maps to `full-access`/`bypassPermissions` session modes)

### 2.4 Store / Persistence

**Package**: `internal/store/`, `internal/store/globaldb/`, `internal/store/sessiondb/`

Dual-database SQLite architecture:

**Global Catalog** (`agh.db`):

- Session metadata (ID, name, agent, workspace, state, stop reason, timestamps)
- Workspace registrations
- Automation jobs, triggers, runs
- Bridge instances and routes
- Network message audit log
- Task graph (tasks, dependencies, runs)
- Bundle activations
- Observability event aggregation

**Per-Session Event Store** (`events.db`):

- Event records with sequence numbers
- Turn grouping
- Hook run records
- Session-scoped event queries with type and time filtering

**Session Meta** (`meta.json`):

- Persisted alongside per-session DB for crash recovery
- Contains session state snapshot for resume

### 2.5 Observability

**Package**: `internal/observe/`

**Features**:

- Event recording and querying across sessions
- Health metrics aggregation
- Bridge health monitoring
- Task health optimization
- Reconciliation between in-memory and persisted state
- Global event stream with filtering

### 2.6 Memory System

**Package**: `internal/memory/`, `internal/memory/consolidation/`

Persistent dual-scope memory that survives across sessions.

**Memory Types** (closed taxonomy):
| Type | Default Scope | Purpose |
|------|--------------|---------|
| `user` | Global | User preferences, recurring facts |
| `feedback` | Global | Quality and review feedback |
| `project` | Workspace | Workspace-specific project knowledge |
| `reference` | Workspace | Workspace-specific external references |

**Memory Scopes**:

- **Global** (`~/.agh/memory/`) -- cross-workspace memories (user preferences, feedback)
- **Workspace** (`<workspace>/.agh/memory/`) -- project-specific memories

**Memory Files**:

- Markdown files with YAML frontmatter (`name`, `description`, `type`, `agent_name`)
- `MEMORY.md` index file per scope (auto-maintained, prompt-safe truncation)
- Atomic writes for crash safety
- Max 200 files scanned per scope

**Dream Consolidation**:

- Background service that periodically consolidates session learnings into memory
- **Time Gate**: Must wait `min_hours` (default 24h) since last consolidation
- **Session Gate**: Must have `min_sessions` (default 3) completed sessions since last consolidation
- **Lock-based concurrency**: File-based lock with rollback on failure
- Spawns a dedicated `dream` session type with a consolidation prompt
- Configurable check interval (default 30 minutes)
- Supports workspace-scoped consolidation

### 2.7 Skills System

**Package**: `internal/skills/`, `internal/skills/bundled/`

Skills are reusable agent capability modules.

**Skill Sources** (precedence order):

1. Workspace skills: `<workspace>/.agh/skills/<name>/SKILL.md`
2. Additional directory skills
3. Global skills: `~/.agh/skills/<name>/SKILL.md`
4. Bundled skills (embedded in binary)

**Bundled Skills**:

- `agh-agent-setup` -- Agent setup instructions
- `agh-memory-guide` -- Memory system usage guide
- `agh-network` -- Network protocol skill (injected when session has a channel)
- `agh-session-guide` -- Session management guide

**Skill Components** (defined in SKILL.md frontmatter):

- Prompt content (markdown body)
- MCP server declarations (sidecar)
- Hook declarations
- Metadata and provenance tracking

**Skill Registry**:

- Hot-reloading with configurable poll interval (default 3s)
- File watcher for workspace skill changes
- Snapshot-based cache per workspace
- Enable/disable individual skills
- Marketplace integration (ClaWHub registry)

**Skill Marketplace**:

- Install from registry: `agh skill install @org/skill-name`
- Remove: `agh skill remove @org/skill-name`
- Update: `agh skill update @org/skill-name`
- Search: `agh skill search <query>`
- Lock file: `skills-lock.json` for version pinning

### 2.8 Workspace Management

**Package**: `internal/workspace/`

**Key Concepts**:

- **Workspace**: A registered directory where agent sessions operate
- **Resolved Workspace**: Runtime representation with ID, root directory, additional directories, and merged config
- **Auto-registration**: Creating a session with `--cwd` auto-registers the workspace
- **Additional Directories**: Multi-root workspace support for monorepos

**Capabilities**:

- Register, list, edit, remove workspaces
- Name or ID-based lookup
- Default agent override per workspace
- Workspace-scoped config overlay
- Scanner for workspace discovery

### 2.9 Transcript Replay

**Package**: `internal/transcript/`

Canonical replay message assembly from persisted events. Reconstructs conversation history for session resume, allowing agents to continue from where they left off.

### 2.10 Hooks System

**Package**: `internal/hooks/`

Comprehensive event-driven hook system for extending AGH behavior.

**Hook Event Categories**:

| Category          | Events                                                                                                                            |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Session Lifecycle | `session.pre_create`, `session.post_create`, `session.pre_resume`, `session.post_resume`, `session.pre_stop`, `session.post_stop` |
| Agent Lifecycle   | `agent.pre_start`, `agent.spawned`, `agent.crashed`, `agent.stopped`                                                              |
| Prompt            | `input.pre_submit`, `prompt.post_assemble`                                                                                        |
| Events            | `event.pre_record`, `event.post_record`                                                                                           |
| Conversation      | `turn.start`, `turn.end`, `message.start`, `message.delta`, `message.end`                                                         |
| Compaction        | `context.pre_compact`, `context.post_compact`                                                                                     |

**Hook Sources**: Config-defined, agent-definition, skill-defined

**Hook Features**:

- Subprocess executor (shell commands)
- Native executor
- Priority ordering
- Conditional matchers (agent name, workspace, session type, tool name, event type, etc.)
- Required/optional hooks (required hooks block on failure)
- Timeout configuration
- Async dispatch for non-blocking hooks
- Pipeline composition
- Worker pool for concurrent execution
- Telemetry and introspection

### 2.11 Automation System

**Package**: `internal/automation/`

Scheduled jobs and event-driven triggers.

**Automation Jobs**:

- **Schedule Modes**: `cron` (cron expressions), `every` (fixed interval), `at` (one-shot timestamp)
- **Scopes**: `global` or `workspace`
- **Sources**: `config` (TOML-defined) or `dynamic` (API-created)
- **Fire Limits**: Rate limiting per window (e.g., "5 per 1h")
- **Retry Policy**: `none` or `backoff` with configurable max retries and base delay
- CRUD + manual trigger + run history

**Automation Triggers**:

- Event-driven activation (session events, webhook events)
- Exact-match filters for event payloads
- Prompt templates with variable interpolation
- Webhook support with HMAC secret verification, endpoint slugs
- Same scope, retry, and fire limit features as jobs

**Automation Runs**:

- Track execution with status (`pending`, `running`, `completed`, `failed`, `canceled`)
- Attempt counting for retries
- Session ID linking
- Time-ranged and status-filtered queries

### 2.12 Bridge Adapters

**Package**: `internal/bridges/`

Connect agent sessions to external messaging platforms.

**Key Concepts**:

- **Bridge Instance**: A configured connection to an external platform (Slack, Discord, etc.)
- **Platform**: The messaging platform identifier
- **Extension**: The bridge implementation (installed as an extension)
- **Routing Policy**: Dimensions for session routing (`peer`, `thread`, `group`)
- **Delivery**: Typed outbound message delivery with mode (`direct-send`, `reply`)

**Bridge Lifecycle**: `starting` -> `connected` -> `disabled` (with enable/disable/restart)

**Capabilities**:

- CRUD for bridge instances
- Routing key hashing for deterministic session binding
- Secret binding management (per-bridge credentials)
- Delivery broker with metrics
- Route inspection and test delivery
- Scoped: global or workspace-level bridges

### 2.13 Extensions System

**Package**: `internal/extension/`

Extensions are installable capability packages for AGH.

**Capabilities**:

- Install from marketplace (GitHub registry)
- Remove, update, list
- Extension-provided bridge adapters
- Extension-provided MCP servers

### 2.14 Task System

**Package**: `internal/task/`

Structured task management with dependency graphs.

**Key Concepts**:

- Tasks with parent-child relationships
- Dependency DAG with topological ordering
- Task runs with claim/start/complete/fail/cancel lifecycle
- Session attachment for task execution tracking
- Integration with automation jobs (task-mode jobs)

### 2.15 Bundles

**Package**: `internal/bundles/`

Pre-configured capability packages that activate multiple features at once.

**Capabilities**:

- Catalog listing
- Preview activation effects
- Activate/update/delete bundle activations
- Network settings for bundles

---

## 3. Network Protocol Deep-Dive

The AGH Network Protocol is the project's key differentiator -- a lightweight protocol for agent-to-agent communication.

### 3.1 Problem Statement

The agent ecosystem lacks a practical protocol for agent-to-agent networking that is transport-aware, artifact-aware, and operationally observable without becoming a workflow engine or telemetry infrastructure.

### 3.2 Protocol Versions

**v0** (RFC 003): Complete functional protocol without cryptographic identity verification
**v1** (RFC 004): Adds Ed25519+JCS baseline trust profile, verified identities, formal conformance levels

v0 is wire-compatible with v1 -- upgrading requires no format changes.

### 3.3 Architecture Layers

1. **AGH Network Core**: Envelope semantics, message kinds, lifecycle, discovery
2. **AGH Network over NATS**: Subject mapping, broadcast/direct routing, NATS-specific behavior

### 3.4 Envelope Schema

Every message is a UTF-8 JSON envelope with these canonical fields:

| Field            | Type    | Required | Purpose                              |
| ---------------- | ------- | -------- | ------------------------------------ |
| `protocol`       | string  | yes      | `agh-network/v0` or `agh-network/v1` |
| `id`             | string  | yes      | Collision-resistant message ID       |
| `kind`           | string  | yes      | Message kind                         |
| `channel`        | string  | yes      | Logical communication namespace      |
| `from`           | string  | yes      | Claimed sender identity (peer_id)    |
| `to`             | string  | no       | Target peer for directed messages    |
| `interaction_id` | string  | no       | Logical interaction container ID     |
| `reply_to`       | string  | no       | Message ID being replied to          |
| `trace_id`       | string  | no       | Distributed correlation ID           |
| `causation_id`   | string  | no       | Parent causal message ID             |
| `ts`             | integer | yes      | Unix epoch seconds                   |
| `expires_at`     | integer | no       | Sender-declared TTL boundary         |
| `body`           | object  | yes      | Kind-specific payload                |
| `proof`          | object  | no       | Reserved for v1 trust profile        |
| `ext`            | object  | no       | Implementation-specific extensions   |

### 3.5 Seven Message Kinds

| Kind      | Purpose                                     | Routing             |
| --------- | ------------------------------------------- | ------------------- |
| `greet`   | Peer presence and capability advertisement  | Broadcast           |
| `whois`   | Peer card lookup and capability retrieval   | Broadcast or Direct |
| `say`     | Chat-first, channel-scoped communication    | Broadcast           |
| `direct`  | Targeted interaction opener/continuation    | Direct              |
| `recipe`  | First-class reusable procedure artifact     | Broadcast or Direct |
| `receipt` | Acknowledge/reject protocol-level admission | Direct              |
| `trace`   | Progress or terminal outcome reporting      | Direct              |

### 3.6 Peer Card

Advertised via `greet` and `whois`:

- `peer_id` -- canonical identity
- `display_name` -- human-friendly label
- `profiles_supported` -- protocol profiles
- `capabilities` -- peer capabilities (opaque strings like `chat.translate`, `artifact.recipe.consume`)
- `artifacts_supported` -- artifact types understood
- `trust_modes_supported` -- e.g., `unverified`

### 3.7 Interaction Lifecycle

Interactions are lightweight logical containers for work progression:

```
[*] --> submitted --> working --> completed
                  --> needs_input --> working (resumed)
                  --> failed
                  --> canceled
```

- Opening `direct` implies `submitted`
- `receipt` acknowledges or rejects
- `trace` reports `working`, `needs_input`, `completed`, `failed`, `canceled`
- Terminal states are authoritative (no regression)
- Cancellation: `receipt(canceled)` = initiator-side, `trace(canceled)` = worker-side

### 3.8 Recipe Artifact

First-class protocol artifact for reusable procedures:

- `recipe_id`, `version`, `title`, `summary`
- `content_type` (e.g., `text/markdown`)
- `digest` (SHA-256)
- `uri` or `inline` content (at least one required)
- `inputs`, `outputs`, `requirements`

### 3.9 NATS Transport Binding

**Subject Prefix**: `agh.network.v0` (v0), `agh.network.v1` (v1)

**Subject Mapping**:

- Broadcast: `agh.network.v0.<channel>.broadcast`
- Direct: `agh.network.v0.<channel>.peer.<route_token>`

**Route Token**: First 32 lowercase hex chars of `SHA-256(peer_id)` (v0), or fingerprint suffix for verified peers (v1)

**Peer Identity Format**:

- v0: Any string matching `[a-z0-9][a-z0-9._-]{0,127}`
- v1 verified: `nickname@fingerprint` where fingerprint = first 32 hex of SHA-256(pubkey)

**Delivery Semantics**: Best-effort (may duplicate, may expire, may reorder, may fail silently)

**Presence**: Periodic `greet` (recommended 30s interval), peers expire at 2x interval (60s)

### 3.10 v1 Trust Profile (Ed25519 + JCS)

**Profile ID**: `agh-network.trust.ed25519-jcs/v1`

**Verification Steps**:

1. Confirm profile and algorithm
2. Decode public key from base64url
3. Compute SHA-256 of public key
4. Verify key_id matches digest
5. Verify `from` fingerprint matches first 32 hex
6. JCS canonicalize envelope (omit `proof.sig`)
7. Verify Ed25519 signature

**Proof-Stripping Defense**: Verified-format `from` without `proof` is `rejected` (not `unverified`)

**Conformance Levels**: Core Sender, Core Receiver, Core Peer, NATS Peer, Verified Peer

### 3.11 AGH Network Implementation

**Package**: `internal/network/`

- `envelope.go` -- Envelope construction and serialization
- `validate.go` -- Envelope validation
- `lifecycle.go` -- Interaction lifecycle state machine
- `peer.go` -- Peer management and Peer Card
- `router.go` -- Channel routing and message dispatch
- `transport.go` -- NATS transport integration
- `delivery.go` -- Message delivery pipeline
- `manager.go` -- Network runtime lifecycle
- `audit.go` -- Message audit logging
- `stats.go` -- Network metrics
- `tasks.go` -- Network-related task operations
- `rules/` -- Routing rules

**CLI Commands**:

- `agh network status` -- Show network runtime status and queue metrics
- `agh network peers [channel]` -- List visible local and remote peers
- `agh network channels` -- List active runtime channels
- `agh network send` -- Send one envelope through the network
- `agh network inbox` -- Show queued inbound messages for a session

**Network Config**:
| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `false` | Enable/disable network |
| `default_channel` | `default` | Default channel name |
| `port` | `-1` (auto) | NATS listener port |
| `max_payload` | 1 MB | Maximum envelope size |
| `greet_interval` | 30s | Heartbeat interval |
| `max_replay_age` | 300s | Message replay window |
| `max_queue_depth` | 100 | Per-session message queue |

---

## 4. Configuration Reference Summary

### 4.1 File Locations

| Path                                      | Purpose                                 |
| ----------------------------------------- | --------------------------------------- |
| `~/.agh/config.toml`                      | Global configuration                    |
| `~/.agh/mcp.json`                         | Global MCP server declarations          |
| `~/.agh/agents/<name>/AGENT.md`           | Global agent definitions                |
| `~/.agh/skills/<name>/SKILL.md`           | Global skill definitions                |
| `~/.agh/memory/`                          | Global memory files                     |
| `~/.agh/sessions/<id>/`                   | Per-session data (meta.json, events.db) |
| `~/.agh/agh.db`                           | Global catalog database                 |
| `~/.agh/daemon.sock`                      | Daemon UDS socket                       |
| `~/.agh/daemon.json`                      | Daemon process info                     |
| `~/.agh/daemon.log`                       | Daemon log file                         |
| `<workspace>/.agh/config.toml`            | Workspace config overlay                |
| `<workspace>/.agh/mcp.json`               | Workspace MCP server declarations       |
| `<workspace>/.agh/agents/<name>/AGENT.md` | Workspace agent definitions             |
| `<workspace>/.agh/skills/<name>/SKILL.md` | Workspace skill definitions             |
| `<workspace>/.agh/memory/`                | Workspace memory files                  |
| `<workspace>/.env`                        | Auto-loaded environment variables       |

### 4.2 Environment Variables

| Variable              | Purpose                             |
| --------------------- | ----------------------------------- |
| `AGH_SESSION_ID`      | Injected into agent process env     |
| `AGH_SESSION_CHANNEL` | Injected when session has a channel |
| `AGH_PEER_ID`         | Network peer ID for the session     |
| `AGH_BIN`             | Path to the AGH binary              |
| `ANTHROPIC_API_KEY`   | Claude provider API key             |
| `OPENAI_API_KEY`      | Codex provider API key              |
| `GEMINI_API_KEY`      | Gemini provider API key             |

### 4.3 Key Default Values

| Setting                    | Default       |
| -------------------------- | ------------- |
| HTTP port                  | 2123          |
| Max sessions               | 10            |
| Max concurrent agents      | 20            |
| Permission mode            | `approve-all` |
| Log level                  | `info`        |
| Observability retention    | 7 days        |
| Max global DB size         | 1 GB          |
| Transcript segment size    | 1 MB          |
| Max transcript per session | 256 MB        |
| Memory enabled             | true          |
| Dream enabled              | true          |
| Dream min hours            | 24            |
| Dream min sessions         | 3             |
| Dream check interval       | 30 min        |
| Skills enabled             | true          |
| Skills poll interval       | 3s            |
| Automation enabled         | true          |
| Automation max concurrent  | 5             |
| Network enabled            | false         |
| Network default channel    | `default`     |
| Network greet interval     | 30s           |
| Network max replay age     | 300s          |
| Network max queue depth    | 100           |

---

## 5. CLI Command Reference Summary

All commands support `--output` (`-o`) flag with values: `human`, `json`, `toon`.

### 5.1 Top-Level Commands

| Command       | Description                       |
| ------------- | --------------------------------- |
| `agh version` | Print version, commit, build date |
| `agh install` | Interactive setup wizard          |
| `agh whoami`  | Show current identity info        |

### 5.2 Daemon Management

| Command                         | Description                            |
| ------------------------------- | -------------------------------------- |
| `agh daemon start`              | Start the daemon (detached by default) |
| `agh daemon start --foreground` | Start in foreground mode               |
| `agh daemon stop`               | Stop the daemon                        |
| `agh daemon status`             | Show daemon status                     |

### 5.3 Session Management

| Command                             | Description                                            |
| ----------------------------------- | ------------------------------------------------------ |
| `agh session new`                   | Create a new session                                   |
| `agh session list`                  | List sessions (active by default, `--all` for stopped) |
| `agh session status <id>`           | Show session status                                    |
| `agh session stop <id>`             | Stop a session                                         |
| `agh session resume <id>`           | Resume a stopped session                               |
| `agh session wait <id>`             | Block until session stops                              |
| `agh session prompt <id> <message>` | Send a prompt to a session                             |
| `agh session events <id>`           | Read session events (with `--follow` for SSE)          |
| `agh session history <id>`          | Show session history grouped by turn                   |

**Session Create Flags**: `--agent`, `--workspace`, `--cwd`, `--name`, `--channel`
**Session Events Flags**: `--type`, `--last`, `--since`, `--follow`
**Session List Flags**: `--all`, `--workspace`

### 5.4 Workspace Management

| Command                             | Description                                               |
| ----------------------------------- | --------------------------------------------------------- |
| `agh workspace add <path>`          | Register a workspace                                      |
| `agh workspace list`                | List registered workspaces                                |
| `agh workspace info <name-or-id>`   | Show workspace details (sessions, agents, skills)         |
| `agh workspace edit <name-or-id>`   | Edit workspace (name, add-dir, remove-dir, default-agent) |
| `agh workspace remove <name-or-id>` | Remove workspace registration                             |

### 5.5 Agent Management

| Command                | Description                      |
| ---------------------- | -------------------------------- |
| `agh agent list`       | List available agent definitions |
| `agh agent get <name>` | Show agent definition details    |

### 5.6 Network Operations

| Command                       | Description                                                              |
| ----------------------------- | ------------------------------------------------------------------------ |
| `agh network status`          | Show network runtime status and metrics                                  |
| `agh network peers [channel]` | List visible peers                                                       |
| `agh network channels`        | List active channels                                                     |
| `agh network send`            | Send an envelope (requires `--session`, `--channel`, `--kind`, `--body`) |
| `agh network inbox`           | Show queued inbound messages (`--session` required)                      |

**Network Send Flags**: `--session`, `--channel`, `--kind`, `--to`, `--body`, `--interaction-id`, `--reply-to`, `--trace-id`, `--causation-id`, `--expires-at`, `--id`, `--ext`

### 5.7 Memory Management

| Command                        | Description                                                                           |
| ------------------------------ | ------------------------------------------------------------------------------------- |
| `agh memory list`              | List persistent memories                                                              |
| `agh memory read <filename>`   | Read a memory file                                                                    |
| `agh memory write <filename>`  | Write a memory (requires `--type`, `--description`, content via `--content` or stdin) |
| `agh memory delete <filename>` | Delete a memory file                                                                  |
| `agh memory consolidate`       | Trigger manual memory consolidation                                                   |

**Memory Flags**: `--scope` (global or workspace)
**Memory Write Types**: `user`, `feedback`, `project`, `reference`

### 5.8 Skill Management

| Command                    | Description              |
| -------------------------- | ------------------------ |
| `agh skill list`           | List skills              |
| `agh skill info <name>`    | Show skill details       |
| `agh skill view <name>`    | View skill content       |
| `agh skill create <name>`  | Create a new skill       |
| `agh skill enable <name>`  | Enable a skill           |
| `agh skill disable <name>` | Disable a skill          |
| `agh skill install <slug>` | Install from marketplace |
| `agh skill remove <slug>`  | Remove marketplace skill |
| `agh skill update <slug>`  | Update marketplace skill |
| `agh skill search <query>` | Search marketplace       |

### 5.9 Automation Management

| Command                                | Description              |
| -------------------------------------- | ------------------------ |
| `agh automation jobs`                  | List automation jobs     |
| `agh automation jobs create`           | Create a job             |
| `agh automation jobs get <id>`         | Show a job               |
| `agh automation jobs update <id>`      | Update a job             |
| `agh automation jobs delete <id>`      | Delete a job             |
| `agh automation jobs trigger <id>`     | Force immediate run      |
| `agh automation jobs history <id>`     | Show job run history     |
| `agh automation triggers`              | List triggers            |
| `agh automation triggers create`       | Create a trigger         |
| `agh automation triggers get <id>`     | Show a trigger           |
| `agh automation triggers update <id>`  | Update a trigger         |
| `agh automation triggers delete <id>`  | Delete a trigger         |
| `agh automation triggers history <id>` | Show trigger run history |
| `agh automation runs`                  | List all runs            |
| `agh automation runs get <id>`         | Show a run               |

### 5.10 Bridge Management

| Command                         | Description                       |
| ------------------------------- | --------------------------------- |
| `agh bridge list`               | List bridge instances             |
| `agh bridge get <id>`           | Show a bridge instance            |
| `agh bridge create`             | Create a bridge instance          |
| `agh bridge update <id>`        | Update mutable bridge fields      |
| `agh bridge enable <id>`        | Enable a bridge                   |
| `agh bridge disable <id>`       | Disable a bridge                  |
| `agh bridge restart <id>`       | Restart a bridge                  |
| `agh bridge routes <id>`        | Inspect routes                    |
| `agh bridge test-delivery <id>` | Test outbound delivery resolution |

### 5.11 Extension Management

| Command                 | Description               |
| ----------------------- | ------------------------- |
| `agh extension list`    | List installed extensions |
| `agh extension install` | Install from marketplace  |
| `agh extension remove`  | Remove an extension       |
| `agh extension update`  | Update an extension       |
| `agh extension search`  | Search marketplace        |

### 5.12 Task Management

| Command                | Description   |
| ---------------------- | ------------- |
| `agh task list`        | List tasks    |
| `agh task get <id>`    | Show a task   |
| `agh task create`      | Create a task |
| `agh task update <id>` | Update a task |
| `agh task cancel <id>` | Cancel a task |

### 5.13 Hooks Introspection

| Command             | Description                       |
| ------------------- | --------------------------------- |
| `agh hooks catalog` | List registered hook declarations |
| `agh hooks runs`    | Show hook execution history       |
| `agh hooks events`  | Show hook event types             |

### 5.14 Observability

| Command              | Description               |
| -------------------- | ------------------------- |
| `agh observe events` | Query global event stream |
| `agh observe health` | Show health metrics       |

---

## 6. API Surface Summary

### 6.1 HTTP API (Web UI + External Clients)

Base path: `/api`

**Sessions**:

- `GET /api/sessions` -- List sessions
- `POST /api/sessions` -- Create session
- `GET /api/sessions/:id` -- Get session
- `DELETE /api/sessions/:id` -- Stop session
- `POST /api/sessions/:id/resume` -- Resume session
- `POST /api/sessions/:id/prompt` -- Send prompt
- `GET /api/sessions/:id/events` -- Query events
- `GET /api/sessions/:id/history` -- Turn-grouped history
- `GET /api/sessions/:id/transcript` -- Full transcript
- `GET /api/sessions/:id/stream` -- SSE event stream
- `POST /api/sessions/:id/approve` -- Approve permission

**Workspaces**:

- `POST /api/workspaces` -- Create
- `GET /api/workspaces` -- List
- `GET /api/workspaces/:id` -- Get (with sessions, agents, skills)
- `PATCH /api/workspaces/:id` -- Update
- `DELETE /api/workspaces/:id` -- Delete
- `POST /api/workspaces/resolve` -- Resolve workspace from path

**Agents**:

- `GET /api/agents` -- List available agents
- `GET /api/agents/:name` -- Get agent details

**Network**:

- `GET /api/network/status` -- Network status and metrics
- `GET /api/network/peers` -- List peers
- `GET /api/network/peers/:peer_id` -- Get specific peer
- `GET /api/network/channels` -- List channels
- `POST /api/network/channels` -- Create channel
- `GET /api/network/channels/:channel` -- Get channel details
- `GET /api/network/channels/:channel/messages` -- Channel message history
- `POST /api/network/send` -- Send envelope
- `GET /api/network/inbox` -- Session inbox

**Memory**:

- `GET /api/memory` -- List memory files
- `GET /api/memory/:filename` -- Read file
- `PUT /api/memory/:filename` -- Write file
- `DELETE /api/memory/:filename` -- Delete file
- `POST /api/memory/consolidate` -- Trigger consolidation

**Skills**:

- `GET /api/skills` -- List skills
- `GET /api/skills/:name` -- Get skill info
- `GET /api/skills/:name/content` -- Get skill content
- `POST /api/skills/:name/enable` -- Enable skill
- `POST /api/skills/:name/disable` -- Disable skill

**Automation**:

- Jobs: `GET/POST /api/automation/jobs`, `GET/PATCH/DELETE /api/automation/jobs/:id`, `POST .../trigger`, `GET .../runs`
- Triggers: `GET/POST /api/automation/triggers`, `GET/PATCH/DELETE /api/automation/triggers/:id`, `GET .../runs`
- Runs: `GET /api/automation/runs`, `GET /api/automation/runs/:id`

**Tasks**:

- `POST /api/tasks` -- Create task
- `GET /api/tasks` -- List tasks
- `GET /api/tasks/:id` -- Get task
- `PATCH /api/tasks/:id` -- Update task
- `POST /api/tasks/:id/cancel` -- Cancel task
- `POST /api/tasks/:id/children` -- Create child task
- `POST /api/tasks/:id/dependencies` -- Add dependency
- `DELETE /api/tasks/:id/dependencies/:depends_on_id` -- Remove dependency
- `POST /api/tasks/:id/runs` -- Enqueue run
- `GET /api/tasks/:id/runs` -- List runs
- Task runs: `POST .../claim`, `.../start`, `.../attach-session`, `.../complete`, `.../fail`, `.../cancel`

**Bridges**:

- `GET/POST /api/bridges` -- List/Create
- `GET /api/bridges/providers` -- List available providers
- `GET/PATCH /api/bridges/:id` -- Get/Update
- `POST /api/bridges/:id/enable` -- Enable
- `POST /api/bridges/:id/disable` -- Disable
- `POST /api/bridges/:id/restart` -- Restart
- `GET /api/bridges/:id/routes` -- Inspect routes
- Secret bindings: `GET .../secret-bindings`, `PUT/DELETE .../secret-bindings/:binding_name`
- `POST /api/bridges/:id/test-delivery` -- Test delivery

**Bundles**:

- `GET /api/bundles/catalog` -- List catalog
- `POST /api/bundles/preview` -- Preview activation
- `GET/POST /api/bundles/activations` -- List/Activate
- `GET/PATCH/DELETE /api/bundles/activations/:id` -- Get/Update/Delete
- `GET /api/bundles/network/settings` -- Network settings

**Webhooks**:

- `POST /api/webhooks/global/:endpoint` -- Global webhook delivery
- `POST /api/webhooks/workspaces/:workspace_id/:endpoint` -- Workspace webhook delivery

**Observability**:

- `GET /api/observe/events` -- Query events
- `GET /api/observe/events/stream` -- SSE stream
- `GET /api/observe/health` -- Health check

**Hooks**:

- `GET /api/hooks/catalog` -- Hook catalog
- `GET /api/hooks/runs` -- Hook run history
- `GET /api/hooks/events` -- Hook event types

**Daemon**:

- `GET /api/daemon/status` -- Daemon status

### 6.2 UDS API (CLI IPC)

Mirrors the HTTP API routes for CLI-daemon communication over Unix Domain Socket. The CLI communicates with the daemon exclusively through UDS.

### 6.3 Web UI

**Technology**: React 19 SPA with Vite, TanStack Router/Query, Tailwind CSS, shadcn/ui

**Feature Systems** (under `web/src/systems/`):
| System | Purpose |
|--------|---------|
| `session` | Session management, prompt interaction, event streaming |
| `workspace` | Workspace management |
| `agent` | Agent definition browsing |
| `network` | Network status, peers, channels, message sending |
| `skill` | Skill catalog and management |
| `automation` | Automation job/trigger management |
| `bridges` | Bridge instance management |
| `daemon` | Daemon status |
| `knowledge` | Knowledge/memory management |

---

## 7. Key Concepts That Need Dedicated Doc Pages

### 7.1 Essential Concepts (must-have pages)

1. **Getting Started / Quick Start** -- Install, `agh install`, first session
2. **Agent Definitions** -- AGENT.md format, frontmatter schema, prompt authoring, multi-root discovery
3. **Sessions** -- Lifecycle, create/resume/stop, prompting, event streaming, permission approval
4. **Workspaces** -- Registration, config overlays, multi-root support, agent scoping
5. **Configuration** -- TOML reference, three-layer merge, MCP JSON sidecar, environment variables
6. **Providers** -- Built-in registry, custom providers, resolution chain
7. **Skills** -- SKILL.md format, bundled skills, marketplace, workspace-scoped skills, MCP sidecar
8. **Memory** -- Types, scopes, MEMORY.md index, file format, dream consolidation
9. **Network Protocol** -- Envelope format, message kinds, channels, peers, interactions, NATS transport
10. **Hooks** -- Event catalog, declaration format, matchers, executors, async dispatch, ordering
11. **Automation** -- Jobs (cron/every/at), triggers (events/webhooks), runs, retry/fire-limit policies
12. **Bridges** -- Platform adapters, routing policy, delivery targets, secret bindings
13. **Observability** -- Event recording, health, transcripts, session history

### 7.2 Architecture / Advanced (important for power users)

14. **Daemon Architecture** -- Boot, shutdown, composition root, process lifecycle
15. **ACP Protocol** -- JSON-RPC over stdio, handshake, session modes, capabilities
16. **Extensions** -- Marketplace, installation, bridge providers
17. **Tasks** -- DAG model, dependencies, run lifecycle, automation integration
18. **Bundles** -- Activation, preview, catalog
19. **MCP Servers** -- Global/workspace/agent/skill MCP server resolution chain
20. **Permissions** -- Mode hierarchy, session mode mapping, per-agent overrides

### 7.3 Reference Pages

21. **CLI Reference** -- Complete command reference with all flags
22. **API Reference** -- OpenAPI spec for HTTP endpoints
23. **Config Reference** -- Full TOML schema with all defaults
24. **Network Protocol Specification** -- v0 and v1 specs (could host RFC content)
25. **Hook Event Reference** -- All events with payload schemas
26. **Troubleshooting** -- Common issues, daemon logs, crash recovery

### 7.4 Guides / Tutorials

27. **Multi-Agent Workflows** -- Using network channels for agent collaboration
28. **Webhook Automation** -- Setting up GitHub/Slack webhook triggers
29. **Custom Agent Authoring** -- Creating specialized agents with skills and hooks
30. **Bridge Setup** -- Connecting Slack/Discord to agent sessions
31. **Memory Best Practices** -- Effective use of global vs workspace memory
