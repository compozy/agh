# OpenClaw Analysis for AGH Extensibility

## Overview

OpenClaw is a personal AI assistant that runs locally on a user's devices and fans out to 20+ messaging platforms through a single control plane. Its architecture follows a **hub-and-spoke control model**: a long-lived **Gateway** daemon owns every channel connection, session, tool invocation, and device pairing, while a separate **assistant runtime** (the Pi agent) performs inference and tool reasoning over WebSocket RPC.

The project is implemented in TypeScript/Node.js with 70+ bundled extensions, 20+ channel adapters, native apps for macOS/iOS/Android, and a Plugin SDK that isolates extensions from core through a narrow typed boundary. OpenClaw targets a single trusted operator who wants one assistant reachable from any device and any chat platform.

### Key Architectural Differences from AGH

| Aspect | OpenClaw | AGH |
|---|---|---|
| Language | TypeScript/Node.js | Go single-binary |
| Communication | WebSocket RPC between Gateway and Assistant | JSON-RPC over stdio between daemon and agents |
| Session persistence | JSONL files on disk | SQLite (globaldb + per-session eventdb) |
| Extension model | npm-based Plugin SDK with manifest discovery | Go interfaces with dependency injection |
| Channel surface | 20+ messaging platform adapters | HTTP/SSE (web UI) + UDS (CLI) |
| Scope | Personal assistant across many platforms | Agent session management via ACP |
| Assistant runtime | Stateless Pi agent over WS RPC | ACP-compatible agents (Claude Code, Codex, etc.) as subprocesses |

---

## Key Features Analysis

### Feature Classification Table

| Feature | OpenClaw Implementation | Classification for AGH | Rationale |
|---|---|---|---|
| **Gateway/Assistant Split** | Stateful Gateway control plane + stateless inference runtime communicating over WS RPC | **CORE** | AGH already has this via daemon + ACP subprocess model. The pattern of keeping all state in the daemon and treating agents as stateless between turns is foundational. |
| **Plugin SDK with Manifest-First Discovery** | Typed boundary (`plugin-sdk/`), `openclaw.plugin.json` manifests, discovery before code execution, four capability types (channel, provider, tool, skill) | **CORE** | AGH needs a plugin registration contract. Manifest-first discovery (inspect metadata without executing code) is a critical safety and performance pattern for any extensible system. |
| **Channel Adapter Matrix** | 20+ messaging adapters (WhatsApp, Telegram, Slack, Discord, etc.) behind a uniform `ChannelPlugin` interface with normalized `InboundMessage` shape | **EXTENSION** | Individual channel adapters are clearly extensions. But the normalized message contract and channel health monitoring model should inform AGH's API layer design. |
| **Skills System (AgentSkills format)** | YAML frontmatter + Markdown files, five-tier precedence (workspace > project > personal > managed > bundled), ClawHub registry, slash commands | **CORE (format) / EXTENSION (individual skills)** | AGH already has a skills package. The five-tier precedence model and the AgentSkills standard format are worth adopting as core. Individual skills and the registry (ClawHub equivalent) are extensions. |
| **Tool System with Catalog** | Self-describing JSON Schema tools, `tools.catalog` discovery, tool profiles (coding/research/creative/dangerous/none), allow/deny/alsoAllow composition | **CORE** | Tool catalog with self-describing schemas, profile-based defaults, and allow/deny composition rules should be core. The pattern of tools advertising their own contracts is essential for any agent-facing tool system. |
| **Approval Flow for High-Risk Operations** | Per-invocation approval with UUID tracking, broadcast to all operators, timeout + deny/approve, iOS push delivery | **CORE** | AGH must have an approval mechanism for dangerous tool invocations. The state machine (request > broadcast > wait > approve/deny/timeout) is simple and effective. |
| **DM Scope Policies** | Four policies (main, per-peer, per-channel-peer, per-account-channel-peer) preventing cross-user context leakage | **EXTENSION** | AGH is currently single-user/single-session focused. DM scope becomes relevant only if AGH exposes channel adapters or multi-user surfaces. The pattern is worth noting for future extension. |
| **Context Compaction** | LLM-powered summarization with proactive and reactive triggers, token estimation, tool-result stripping, identifier preservation, write-lock safety | **CORE** | AGH already has consolidation in its memory package. OpenClaw's approach (reactive on overflow + proactive guard, configurable compaction model, summarization that preserves identifiers) provides useful refinements. |
| **Device Pairing and Node Capabilities** | Fingerprint-pinned tokens, capability advertisement (`camera`, `canvas`, `screen`, `location`, `voice`), capability-based routing | **EXTENSION** | Device pairing and node capabilities are entirely about multi-device reach. Not relevant for AGH's current scope but a clean extension point if AGH adds device/node support. |
| **ACP Bridge for IDEs** | stdio-to-WS translator process (`openclaw acp`), session mapping, prompt/cancel/listSessions translation | **CORE** | AGH already has ACP as its primary protocol. OpenClaw's bridge pattern validates AGH's approach. The session-mapping strategy (per-client default, explicit override) is a good pattern. |
| **Canvas UI Rendering (A2UI)** | Agent-controlled HTML/CSS/JS workspace + structured A2UI v0.8 protocol, per-session file storage, deep-link scheme back to agent loop | **EXTENSION** | A rich visual surface is a powerful capability but not essential for AGH's core. Should be an extension that any agent can use if available. |
| **Voice and Speech Stack** | Wake-word detection, Talk Mode, STT/TTS provider matrix with fallback chains, global wake-word sync across devices | **EXTENSION** | Voice is a premium feature that adds complexity. Should be a cleanly separated extension with provider interfaces. |
| **Browser Automation** | Multi-profile CDP control, SSRF protection, node-host proxying, Chrome Extension Relay for user sessions, accessibility tree snapshots | **EXTENSION** | Browser automation is a powerful tool but clearly an extension. The SSRF protection pattern and profile-based isolation are worth noting. |
| **Model Provider System** | Auth profiles, auto-discovery (Ollama, Bedrock, Vertex), OAuth token management, auth profile rotation with cooldown, `models.json` pipeline, three-level parameter merge | **CORE (provider interface) / EXTENSION (individual providers)** | AGH needs a provider abstraction. The auth profile rotation with cooldown (don't thrash a rate-limited key) and the three-level parameter merge (global > model-specific > agent-specific) are patterns worth adopting in core. |
| **Cron/Webhooks (Proactive Agent)** | Cron scheduler for periodic agent jobs, webhook endpoints for HTTP-triggered runs | **EXTENSION** | Proactive agent triggers (scheduled jobs, external webhooks) are extensions on top of the core session model. Good extension candidates. |
| **Sandboxing (Docker)** | Three specialized images (generic, browser, common base), per-invocation container spawn, resource limits, network policy, nested sandboxing | **EXTENSION** | Docker-based sandboxing is an isolation strategy. AGH should define a sandboxing interface in core but let the Docker implementation be an extension. |
| **Security Audit System** | `openclaw security audit` CLI command, automated checks for filesystem permissions, gateway config, sandbox config, channel policies, skill code safety, tool policy | **CORE** | A security audit surface that validates configuration against best practices should be part of AGH's core. The pattern of automated security assessment at CLI time is valuable. |
| **Onboard Wizard** | Interactive six-step setup (model/auth, workspace, gateway, channels, daemon, health), non-interactive mode for CI, idempotent reconfiguration | **CORE** | AGH needs a clean first-boot experience. The pattern of wizard-writes-config (not hidden state) and idempotent `configure --section` reconfiguration is good. |
| **Session Middleware/Hooks** | `before_compaction`, `after_compaction`, `session.load`, `context.assemble` hooks with exec:// handlers | **CORE** | Lifecycle hooks at well-defined points (session load, context assembly, pre/post-compaction) enable extensions without core changes. AGH should define these hook points. |
| **Idempotency Keys** | Per-request UUID for side-effecting methods, retry-safe collapse of duplicate messages | **CORE** | Essential for any system where messages can be delivered at-least-once. AGH should adopt idempotency keys for state-mutating operations. |
| **Event Fan-Out / Broadcast** | Every agent event broadcasts to all authorized subscribers, enabling multi-client observation | **CORE** | AGH already has this via SSE. The pattern of every connected client seeing the same event stream is fundamental for observability. |
| **Sub-Agent System** | `sessions_spawn` tool, sub-agent registry with lifecycle tracking, thread-bound sessions, announcement/delivery pipeline with exactly-once semantics | **EXTENSION** | Multi-agent orchestration is an advanced feature. The `sessions_spawn` unified entry point and the sub-agent registry pattern are worth studying for AGH's future phases. |
| **Deployment Topologies** | Six deployment modes (local, Tailscale Serve, SSH tunnel, Tailscale Funnel, Docker, Fly.io) with explicit migration paths | **EXTENSION** | Each deployment topology beyond local is an extension concern. But the health endpoint contract (`/healthz`, `/readyz`) should be core. |

---

## Architectural Patterns Worth Adopting

### 1. Manifest-First Plugin Discovery

OpenClaw's strongest extensibility pattern is the split between **manifest discovery** (read metadata, no code execution) and **code loading** (dynamic import after validation). This means:

- `openclaw plugins status` can list all plugins without executing any plugin code
- Requirements can be checked, missing dependencies flagged, before any risk
- Disabled plugins are never loaded

**AGH recommendation**: Define a plugin manifest format (TOML or JSON) that AGH reads at daemon startup before loading any plugin Go code. This enables `agh plugins list` without importing plugin packages.

### 2. Four-Capability Plugin Model

OpenClaw defines exactly four plugin capabilities: **channels**, **providers**, **tools**, **skills**. Every extension implements one or more of these through typed contracts. This keeps the plugin surface finite and comprehensible.

**AGH recommendation**: Define AGH's plugin capabilities explicitly. Candidates:
- **AgentDriver** (already exists as an interface in `session/`)
- **Tool** (agent-callable capabilities with JSON Schema)
- **Skill** (YAML+Markdown instruction files)
- **Observer** (event consumers for observability/integrations)

### 3. Tool Profiles with Allow/Deny Composition

OpenClaw's tool profile system (`coding`, `research`, `creative`, `dangerous`, `none`) provides sensible defaults. The composition rule (`deny` always wins, `alsoAllow` adds, `allow` replaces) is simple and predictable.

**AGH recommendation**: Adopt this for AGH's tool configuration. It avoids the complexity of inheritance trees while giving users enough control. The rule "deny always wins" is the right safety default.

### 4. Skill Precedence Tiers

Five tiers (workspace > project-agents > personal-agents > managed > bundled) with higher tiers winning. This lets users override bundled behavior without forking.

**AGH recommendation**: AGH already has bundled skills. Adding workspace-level and personal-level tiers would let users customize without modifying the binary. The precedence model is simple: scan each tier, build a name-to-definition map, higher tiers overwrite.

### 5. Lifecycle Hooks at Defined Points

OpenClaw exposes hooks at `before_prompt_build`, `before_compaction`, `after_compaction`, `session.load`, and `context.assemble`. These are not a generic event bus -- they are specific, named lifecycle points where extensions can inject behavior.

**AGH recommendation**: Define AGH's lifecycle hook points explicitly in the daemon package. Candidates:
- `session.create` / `session.resume`
- `context.assemble` (before building the prompt for the agent)
- `event.record` (after an event is persisted)
- `agent.start` / `agent.done`
- `consolidation.before` / `consolidation.after`

### 6. Health Endpoints as Core Contract

Every OpenClaw deployment mode (local, Docker, Fly.io, Tailscale) uses the same `/healthz` and `/readyz` HTTP endpoints. Supervisors, load balancers, and health monitors all converge on these two URLs.

**AGH recommendation**: AGH should expose `/healthz` (liveness) and `/readyz` (readiness) on the HTTP API as a core contract. These are cheap to implement and universally useful.

### 7. Idempotency Keys for Side-Effecting Operations

OpenClaw requires idempotency keys on every state-mutating WS RPC method. This collapses duplicates from at-least-once delivery and makes retries safe.

**AGH recommendation**: Adopt idempotency keys for AGH's HTTP API endpoints that mutate state (session creation, event submission, config changes). Store recent keys in an LRU cache with TTL.

### 8. Normalized Message Shape

OpenClaw compresses 20+ platform-specific message formats into one `InboundMessage` type: `{senderId, channelId, accountId, threadId, groupId, text, timestamp, attachments}`. Every downstream consumer works with this single shape.

**AGH recommendation**: If AGH adds input channels beyond HTTP/UDS, define a canonical internal message type early. Even for HTTP/UDS, a normalized request shape simplifies the pipeline.

---

## Extension System Insights

### ClawHub (Skills Registry)

OpenClaw's ClawHub (clawhub.ai) is a centralized public registry for skills, modeled after npm:

- **Install**: `openclaw skills install github`
- **Update**: `openclaw skills update --all`
- **Search**: `openclaw skills search weather`
- **Version pinning**: `openclaw skills install github@1.2.3`

Skills are distributed as git repos or npm packages with semver tags. The registry is **optional** -- users can point `skills.load.extraDirs` at any local directory and skip ClawHub entirely.

**Insight for AGH**: A skill registry is a Phase 2-3 concern. For now, AGH should ensure its skills format is portable (the AgentSkills standard is shared across multiple agent frameworks). When a registry is needed, the npm-like CLI UX (`agh skills install/update/search`) is the right model. The critical design decision is making the registry optional -- air-gapped and enterprise deployments must work without it.

### Native Apps (Node Mode)

OpenClaw's native apps (macOS, iOS, Android) connect as **node-role WebSocket clients** that expose device capabilities back to the Gateway. They are NOT plugins (no in-process registration). Instead:

1. Connect to Gateway WS with `role: "node"`
2. Advertise capabilities: `["camera", "canvas", "screen", "location", "voice"]`
3. Gateway indexes capabilities by device
4. Agent calls `nodes.invoke({command: "camera.snap"})` and Gateway routes to the right device

The distinction between **plugins** (in-process, Plugin SDK boundary) and **nodes** (external process, WS protocol boundary) is clean and important.

**Insight for AGH**: If AGH adds device/node support, adopt this two-tier model:
- **Extensions/plugins**: Go interfaces, in-process, compiled into the binary or loaded at startup
- **Nodes/clients**: External processes connecting via HTTP/WS/UDS, advertising capabilities, receiving routed commands

### Channel Adapters

Each channel adapter implements five concerns:
1. **Transport** -- how it connects
2. **Normalization** -- platform-native to internal message type
3. **Send/receive** -- round-trip delivery
4. **Auth/accounts** -- credential management
5. **Health monitoring** -- state machine with reconnect backoff

The key insight is that every adapter follows the same interface, and core never special-cases bundled vs. third-party adapters.

**Insight for AGH**: AGH's HTTP/SSE and UDS "channels" already follow this pattern implicitly. If AGH adds more input surfaces (CLI stdio, WebSocket, platform-specific adapters), formalizing the adapter interface would be valuable. The five-concern decomposition is a good checklist.

### Plugin Configuration Pattern

OpenClaw separates plugin config into three layers:
- **`config`**: passed to plugin setup code, referenced in prompts (no raw secrets)
- **`env`**: injected as process environment at tool invocation time (for secrets)
- **`enabled`**: toggle without removing config

The split between `config` (agent-visible) and `env` (execution-only, never in prompt) prevents secret leakage through the LLM.

**Insight for AGH**: When AGH's skill/tool extensions need configuration, adopt this config/env split. Never let extension secrets appear in the context sent to the LLM.

### Extension Loading Order

OpenClaw enforces a deterministic loading order: providers before channels before skills. This prevents a channel from registering before its required provider is loaded.

**Insight for AGH**: AGH's daemon package (the composition root) should document and enforce an explicit initialization order for extensions. Go's `init()` functions are not sufficient -- explicit ordering through the daemon's boot sequence is needed.

---

## Patterns to Explicitly Avoid

### 1. WebSocket RPC Between Gateway and Assistant

OpenClaw uses WS RPC because the Gateway and assistant can be on different hosts. AGH uses stdio JSON-RPC because agents are subprocesses. AGH's approach is simpler and more appropriate for its single-binary model. Do not adopt OpenClaw's WS split.

### 2. In-Process Channel Adapters (at scale)

OpenClaw runs 20+ channel adapters inside the Gateway process. This is fine for Node.js's event-loop model but would be problematic in Go if each adapter needed goroutines with complex lifecycle management. If AGH adds channel adapters, consider subprocess isolation rather than in-process loading.

### 3. JSONL Session Persistence

OpenClaw uses append-only JSONL files for session transcripts. AGH already uses SQLite, which is strictly better for structured queries, concurrent access, and crash recovery. Do not regress to JSONL.

### 4. 70+ Bundled Extensions

OpenClaw ships 70+ extensions in its binary. AGH's philosophy is "robust minimal core" -- keep the binary lean, let extensions be separately compiled or loaded. Do not bundle everything.

---

## Summary of Recommendations

### Must-Adopt (Core)

1. **Manifest-first plugin discovery** -- read metadata before executing code
2. **Typed plugin capability model** -- enumerate the finite set of extension types
3. **Tool profiles with allow/deny composition** -- sensible defaults, predictable overrides
4. **Lifecycle hooks at named points** -- not a generic bus, but specific extension points
5. **Approval flow for dangerous operations** -- per-invocation, with timeout
6. **Health endpoints** -- `/healthz` and `/readyz` as core contract
7. **Idempotency keys** -- for all state-mutating API operations
8. **Security audit CLI** -- automated configuration validation

### Should-Adopt (Near-term Extension Design)

1. **Five-tier skill precedence** -- workspace > project > personal > managed > bundled
2. **Config/env split for extension secrets** -- never leak secrets into LLM context
3. **Deterministic extension loading order** -- enforce in daemon boot sequence
4. **Normalized internal message type** -- prepare for multiple input surfaces

### Worth-Studying (Future Phases)

1. **ClawHub-style registry** -- when AGH has enough extensions to warrant discovery
2. **Node capability advertisement** -- when AGH supports multi-device
3. **Sub-agent orchestration** -- `sessions_spawn` pattern for Phase 3 agent networks
4. **A2UI-style structured surfaces** -- if AGH adds visual output beyond web UI
5. **Channel adapter matrix** -- if AGH moves beyond HTTP/SSE + UDS
