# OpenFang Extension Architecture Analysis

Research date: 2026-04-10
Source: OpenFang v0.5.7 (Rust, 14-crate workspace)
Repository: `https://github.com/RightNow-AI/openfang`

## Overview

OpenFang is a Rust single-binary agent daemon with a **compile-time composition** philosophy. Its extension surface is organized into four distinct subsystems:

1. **53 Builtin Tools** -- Rust functions compiled into the binary (filesystem, web, shell, browser, media, inter-agent, memory, collaboration)
2. **MCP Integration** -- 25 bundled MCP server templates + arbitrary user-defined servers, connected at daemon boot via `rmcp` SDK
3. **40 Channel Adapters** -- messaging platform bridges (Telegram, Discord, Slack, etc.) compiled into the binary
4. **Skills + Hands** -- 60+ bundled skills (Python/WASM/Node/Shell/PromptOnly runtimes) and 7 preconfigured autonomous agent packages ("Hands")

All four subsystems merge into a **single unified tool catalog** that the LLM sees. The `ToolRunner` dispatches each call to the correct backend based on tool name prefix (`mcp_*`, `agent_*`, skill names, or builtin names). There is **no dynamic plugin loading** for core capabilities -- the binary ships with everything. The escape hatches for runtime extensibility are ClawHub skill marketplace downloads and MCP server connections.

### 14-Crate Workspace Structure

```
crates/
  openfang-types/        # Shared types, config, agent, capability, tool definitions
  openfang-runtime/      # Agent loop, tool runner, MCP client, sandbox, audit
  openfang-kernel/       # Composition root: kernel, capabilities, workflows, scheduler
  openfang-api/          # HTTP/SSE server, channel bridge adapter
  openfang-cli/          # CLI commands
  openfang-channels/     # 40 channel adapters, bridge manager, router
  openfang-skills/       # Skill system: registry, loader, ClawHub, verification
  openfang-hands/        # Hand definitions, registry, lifecycle
  openfang-extensions/   # Integration registry, credential vault, OAuth, health monitor
  openfang-memory/       # Memory substrate (SQLite)
  openfang-wire/         # OFP peer network, HMAC-SHA256 auth
  openfang-migrate/      # Database migrations
  openfang-desktop/      # Desktop integration
  xtask/                 # Build tasks
```

Key dependency flow: `openfang-kernel` is the composition root. It imports `openfang-runtime`, `openfang-memory`, `openfang-skills`, `openfang-hands`, `openfang-extensions`. The runtime defines the `KernelHandle` trait to avoid circular deps -- the kernel implements it.

## Tool ("Hands") System

### Builtin Tool Architecture

The `ToolRunner` in `crates/openfang-runtime/src/tool_runner.rs` is the central dispatch point. Its `execute_tool` function signature reveals the full dependency surface:

```rust
pub async fn execute_tool(
    tool_use_id: &str,
    tool_name: &str,
    input: &serde_json::Value,
    kernel: Option<&Arc<dyn KernelHandle>>,          // Inter-agent ops
    allowed_tools: Option<&[String]>,                 // Capability enforcement
    caller_agent_id: Option<&str>,
    skill_registry: Option<&SkillRegistry>,            // Skill dispatch
    mcp_connections: Option<&Mutex<Vec<McpConnection>>>, // MCP dispatch
    web_ctx: Option<&WebToolsContext>,
    browser_ctx: Option<&BrowserManager>,
    allowed_env_vars: Option<&[String]>,
    workspace_root: Option<&Path>,
    media_engine: Option<&MediaEngine>,
    exec_policy: Option<&ExecPolicy>,
    tts_engine: Option<&TtsEngine>,
    docker_config: Option<&DockerSandboxConfig>,
    process_manager: Option<&ProcessManager>,
) -> ToolResult
```

**Dispatch routing** (by tool name):
- `mcp_*` prefix -> routes to `McpConnection::call_tool` on the matching server
- `agent_*` names -> routes to `KernelHandle` inter-agent methods
- Skill tool names -> routes to `SkillRegistry` + `execute_skill_tool`
- Everything else -> direct Rust function call (builtin)

**Pre-dispatch checks** (every call):
1. Tool name normalization via `normalize_tool_name` (e.g. `fs-write` -> `file_write`)
2. Capability enforcement: reject if tool not in `allowed_tools` list
3. Approval gate: check if tool requires human approval (configurable per risk level)
4. Taint tracking: `check_taint_shell_exec` and `check_taint_net_fetch` sanitize arguments

### Tool Definition Model

```rust
// From openfang-types/src/tool.rs
pub struct ToolDefinition {
    pub name: String,
    pub description: String,
    pub input_schema: serde_json::Value, // JSON Schema
}

pub struct ToolResult {
    pub tool_use_id: String,
    pub content: String,
    pub is_error: bool,
}
```

### Tool Execution Matrix

| Tool Type    | Execution        | Sandbox                      | Timeout |
|-------------|------------------|------------------------------|---------|
| Builtin     | Direct Rust fn   | Kernel restrictions          | N/A     |
| WASM skill  | Wasmtime         | Fuel + epoch + watchdog      | 30s     |
| Python skill| Subprocess       | `env_clear()` + allowlist    | 120s    |
| Node skill  | Subprocess       | `env_clear()` + allowlist    | 120s    |
| Shell skill | Subprocess       | `env_clear()` + allowlist    | 120s    |
| MCP tool    | Remote call      | Transport isolation          | 60s     |
| Inter-agent | Kernel dispatch  | Recursion guard (depth 5)    | 600s    |

### Tool Filtering Per Agent

Agents declare tool subsets in their manifest via TOML config:

```toml
[[agents]]
name = "chat-assistant"
tools = ["web_search", "web_fetch", "memory_recall"]      # Whitelist

[[agents]]
name = "researcher"
tools_exclude = ["docker_run", "kill_process"]             # Blacklist
```

The `CapabilityManager` (in `openfang-kernel/src/capabilities.rs`) uses a `DashMap<AgentId, Vec<Capability>>` for concurrent-safe RBAC. Child agents inherit a subset of parent capabilities, preventing privilege escalation through delegation.

### Hands System

**Hands** are preconfigured autonomous agent packages combining:
- A `HAND.toml` manifest (tools, settings, dashboard metrics, requirements, guardrails)
- A system prompt (500+ words expert persona)
- A `SKILL.md` domain knowledge file
- A default cron schedule

Defined in `crates/openfang-hands/src/lib.rs`, the `HandDefinition` struct includes:

```rust
pub struct HandDefinition {
    pub id: String,
    pub name: String,
    pub description: String,
    pub category: HandCategory,
    pub tools: Vec<String>,               // Required tool names
    pub skills: Vec<String>,              // Skill allowlist
    pub mcp_servers: Vec<String>,         // MCP server allowlist
    pub requires: Vec<HandRequirement>,   // Binary/env/API key prereqs
    pub settings: Vec<HandSetting>,       // User-configurable settings
    pub agent: HandAgentConfig,           // LLM config + system prompt
    pub dashboard: HandDashboard,         // Metrics schema
    pub skill_content: Option<String>,    // Injected at load time
}
```

**Hand lifecycle states**: `Active -> Paused -> Error -> Inactive`

**7 Bundled Hands**: Researcher, Lead, Collector, Predictor, Clip, Twitter, Browser -- all compiled into the binary.

**Key pattern for AGH**: Hands are essentially a "meta-extension" that composes tools + skills + prompts + schedules into a named, configurable, marketplace-distributable agent personality. The `HandRequirement` system (binary checks, env var checks, API key checks) with platform-specific `HandInstallInfo` is particularly well-designed for UX.

## Channel Adapter Pattern

### Trait Architecture

The channel system uses two complementary traits defined in `crates/openfang-channels/`:

**Inbound** (`ChannelAdapter` in `types.rs`):
```rust
#[async_trait]
pub trait ChannelAdapter: Send + Sync {
    fn name(&self) -> &str;
    fn channel_type(&self) -> ChannelType;
    async fn start(&self) -> Result<Pin<Box<dyn Stream<Item = ChannelMessage> + Send>>, Box<dyn Error>>;
}
```

**Outbound** (`MessageAdapter`):
```rust
#[async_trait]
pub trait MessageAdapter {
    async fn send(&self, msg: OutboundMessage) -> Result<()>;
    async fn connect(&self) -> Result<()>;
    async fn disconnect(&self) -> Result<()>;
}
```

Each concrete adapter implements both traits. The `start()` method returns an async stream -- the bridge polls this continuously. Each adapter hides its own transport (WebSocket, long-polling, webhooks, SSE) behind the stream abstraction.

### Unified Message Envelope

```rust
pub struct ChannelMessage {
    pub channel: ChannelType,
    pub platform_message_id: String,
    pub sender: ChannelUser,
    pub content: ChannelContent,       // Text | Image | File | FileData | Voice | Location | Command
    pub target_agent: Option<AgentId>,
    pub timestamp: DateTime<Utc>,
    pub is_group: bool,
    pub thread_id: Option<String>,
    pub metadata: HashMap<String, serde_json::Value>,
}
```

### Bridge Manager and Routing

The `BridgeManager` in `bridge.rs` orchestrates adapter lifecycle:
1. Reads channel config from TOML
2. Instantiates each enabled adapter
3. Spawns a Tokio task per adapter to consume its message stream
4. Applies policies (DM policy, group policy, user allow/block lists)
5. Routes through `AgentRouter` with 5-level priority chain:
   - Bindings (most specific: user+channel -> agent)
   - Direct routes
   - User defaults (persisted preference)
   - Channel defaults
   - System default (global fallback)

### Channel Bridge Handle

The `ChannelBridgeHandle` trait (defined in channels crate, implemented in API crate) breaks the circular dependency between channels and kernel:

```rust
#[async_trait]
pub trait ChannelBridgeHandle: Send + Sync {
    async fn send_message(&self, agent_id: AgentId, message: &str) -> Result<String, String>;
    async fn send_message_with_blocks(&self, agent_id: AgentId, blocks: Vec<ContentBlock>) -> Result<String, String>;
    async fn find_agent_by_name(&self, name: &str) -> Result<Option<AgentId>, String>;
    async fn list_agents(&self) -> Result<Vec<(AgentId, String)>, String>;
    async fn spawn_agent_by_name(&self, manifest_name: &str) -> Result<AgentId, String>;
    // ... transcribe_audio, pending_approvals, uptime_info, etc.
}
```

### Adding a New Adapter (3 steps)

1. Implement `ChannelAdapter` + `MessageAdapter` in a new module under `crates/openfang-channels/src/`
2. Register in `crates/openfang-channels/src/lib.rs` factory + add `ChannelType` variant in `openfang-types`
3. Add config support for `[channels.<name>]` section

No changes to kernel, agent loop, or API server required. The trait is the sole integration point.

### Configuration and Policies

```toml
[channels.telegram]
bot_token = "${TELEGRAM_BOT_TOKEN}"
default_agent = "assistant"
model_override = "gpt-4"
rate_limit = 10
dm_policy = "allowed_only"      # Respond | AllowedOnly | Ignore
group_policy = "mention_only"   # All | MentionOnly | CommandsOnly | Ignore
output_format = "telegram_html" # Markdown | TelegramHtml | SlackMrkdwn | PlainText
allowed_users = ["@alice", "@bob"]
blocked_users = ["@spammer"]
```

Features: per-channel rate limiting, message splitting (platform size limits), user filtering, model override per channel, auto-reply templates, command prefix recognition, hot reload without daemon restart.

## MCP Integration

### Client Architecture

Implemented in `crates/openfang-runtime/src/mcp.rs` using the `rmcp` SDK.

**Three transports**:
```rust
pub enum McpTransport {
    Stdio { command: String, args: Vec<String> },  // Most common for bundled
    Sse { url: String },                           // Deprecated (2024-11-05)
    Http { url: String },                          // Current recommended (2025-03-26+)
}
```

**Connection lifecycle** (`McpConnection`):
1. **Connect** -- spawn subprocess or open HTTP stream, perform MCP `initialize` handshake
2. **Discover** -- call `tools/list`, convert tool schemas to internal `ToolDefinition` format
3. **Map** -- namespace tools as `mcp_{server}_{tool}`, store original names for reverse lookup
4. **Execute** -- `call_tool` sends JSON-RPC with 60s timeout

**Key implementation detail**: The `original_names: HashMap<String, String>` preserves server-side tool names because hyphens (e.g., `list-repos`) are normalized to underscores for LLM function-calling compatibility, but the MCP server expects the original name.

### Tool Namespacing

Every MCP tool is prefixed: `mcp_{server_name}_{tool_name}`. This prevents collisions across servers (two servers could both expose `search`). The prefix also makes tool origin transparent in logs. Provider-specific schema adaptation strips unsupported JSON Schema keys (`$schema`, `$defs`, `additionalProperties`, `title`) and inlines `$ref` references.

### 25 Bundled Integration Templates

Managed by `IntegrationRegistry` in `crates/openfang-extensions/src/registry.rs`. Each template is an `IntegrationTemplate` struct embedded at compile time via `bundled.rs`. The registry supports:

- `load_bundled()` -- parse compile-time TOML templates
- `load_installed()` -- merge with `~/.openfang/integrations.toml`
- `install()` / `uninstall()` -- manage installed state
- `to_mcp_configs()` -- convert to `McpServerConfig` for kernel consumption

**Installation flow** (`openfang add <name>`):
1. Lookup template in `IntegrationRegistry`
2. Prompt operator for credentials (or OAuth PKCE flow)
3. Encrypt credentials in vault (AES-256-GCM)
4. Write integration config to `integrations.toml`
5. Mark as `Ready` or `Setup` based on credential completeness

### MCP Server Mode

OpenFang can also operate **as** an MCP server, exposing its tools to external clients. This enables bidirectional MCP: one OpenFang instance connects to another's MCP server endpoint, discovers remote tools, and invokes them. The A2A protocol layers on top of this.

## Workflow Engine

Implemented in `crates/openfang-kernel/src/workflow.rs`.

### Data Model

```rust
pub struct Workflow {
    pub id: WorkflowId,
    pub name: String,
    pub description: String,
    pub steps: Vec<WorkflowStep>,
    pub created_at: DateTime<Utc>,
}

pub struct WorkflowStep {
    pub name: String,
    pub agent: StepAgent,            // ById { id } or ByName { name }
    pub prompt_template: String,     // Jinja-style: {{input}}, {{var_name}}
    pub mode: StepMode,
    pub timeout_secs: u64,
    pub error_mode: ErrorMode,       // Fail | Skip | Retry { max_retries }
    pub output_var: Option<String>,
}
```

### Five Step Modes

| Mode | Agent Invocations | Blocking | Use Case |
|------|-------------------|----------|----------|
| `Sequential` | 1 | Yes | Linear pipeline |
| `FanOut` | N (parallel) | Yes (all) | Multi-perspective analysis |
| `Collect` | 0 (merge only) | No | Aggregate parallel outputs |
| `Conditional { condition }` | 0 or 1 | Yes | Quality gates, branching |
| `Loop { max_iterations, until }` | 1 to N | Yes | Iterative refinement |

### Variable Interpolation

Steps reference variables via `{{var_name}}` syntax. Resolution order:
1. Step output variables (highest, most recent wins)
2. Global workflow variables
3. Initial input (`{{input}}`)

Missing variables leave the placeholder literal -- deliberate to prevent silent failures.

### Execution Flow

Each step invocation goes through the standard `run_agent_loop()`, respecting loop guards and metering. The workflow engine adds no execution capability of its own -- it only orchestrates when and how agent loops run. Fan-out uses `tokio::task::JoinSet` for structured concurrency. Each parallel agent gets a fresh session.

### Configuration

```toml
[[workflows]]
id = "research-pipeline"
name = "Research Pipeline"

  [[workflows.steps]]
  name = "research"
  agent_id = "researcher"
  prompt = "Research {{topic}}"
  mode = "sequential"
  timeout_secs = 300
  output_var = "findings"
  error_mode = "fail"
```

### API Surface

8 endpoints: CRUD for workflow definitions + run/list-runs/get-run. Also accessible via CLI (`openfang workflow list|create|run`).

## Plugin/Extension Model

OpenFang does **not** have a traditional plugin system with dynamic loading. Instead, it uses a **four-layer composition model**:

### Layer 1: Compile-Time (Builtins, Channel Adapters, Hands)
All 53 tools, 40 adapters, and 7 Hands are compiled into the binary. Benefits: auditability (single hash = known feature set), air-gap deployment, startup speed, integrity.

### Layer 2: Boot-Time (MCP Servers, Bundled Skills)
MCP server connections are established and skill registries loaded during daemon boot. The tool catalog is rebuilt at this point.

### Layer 3: Runtime (ClawHub Skills, User MCP Servers)
Skills can be installed from the ClawHub marketplace at runtime:
```bash
openfang skill install financial-analysis
# Downloads, validates Ed25519 signature, installs to ~/.openfang/skills/
```

Skills declare their runtime in `SKILL.toml`:
```rust
pub struct SkillManifest {
    pub skill: SkillMeta,            // name, version, description, tags
    pub runtime: SkillRuntimeConfig, // Python | Wasm | Node | Shell | Builtin | PromptOnly
    pub tools: SkillTools,           // Tool definitions
    pub requirements: SkillRequirements,
    pub prompt_context: Option<String>, // For PromptOnly skills
    pub source: Option<SkillSource>,    // Native | Bundled | OpenClaw | ClawHub
}
```

The `SkillRegistry` supports:
- `load_bundled()` -- compile-time embedded SKILL.md files
- `load_all()` -- scan `~/.openfang/skills/` directory
- `freeze()` -- lock registry in Stable mode (no new skills)
- Prompt injection scanning on all skills (even bundled, defense-in-depth)
- OpenClaw compatibility layer for cross-framework skill format

### Layer 4: Protocol-Level (A2A, MCP Server Mode)
Cross-instance extensibility via A2A protocol and MCP server mode.

### Skill Execution Model

The `SkillLoader` in `crates/openfang-skills/src/loader.rs` dispatches by runtime type:

```rust
match manifest.runtime.runtime_type {
    SkillRuntime::Python => execute_python(skill_dir, entry, tool_name, input),
    SkillRuntime::Node   => execute_node(skill_dir, entry, tool_name, input),
    SkillRuntime::Shell  => execute_shell(skill_dir, entry, tool_name, input),
    SkillRuntime::Wasm   => Err("not yet implemented"),
    SkillRuntime::Builtin => Err("handled by kernel"),
    SkillRuntime::PromptOnly => Ok("instructions are in system prompt"),
}
```

Skills communicate via **stdin/stdout JSON**: the loader spawns a subprocess, writes `{"tool": name, "input": input}` to stdin, reads JSON from stdout. Environment is cleared (`env_clear()`) with only PATH, HOME, and PYTHONIOENCODING allowed.

### Multi-Language Support

- **Python**: Subprocess with `env_clear()`, JSON over stdin/stdout
- **Node.js**: Subprocess (OpenClaw compatibility layer)
- **Shell/Bash**: Subprocess with sandbox
- **WASM**: Wasmtime sandbox (fuel + epoch metering) -- declared but not yet fully implemented
- **PromptOnly**: No code execution, markdown injected into system prompt
- **Builtin**: Direct Rust functions

## Security Model

OpenFang implements **16 interlocking security layers** distributed across every crate:

### Layers Most Relevant to Extensions

| # | Layer | Purpose for Extensions |
|---|-------|----------------------|
| 1 | WASM dual-metered sandbox | Sandboxes untrusted skills (fuel + epoch + watchdog timeout) |
| 3 | Taint tracking | Prevents credential leakage into LLM prompts via `Tainted<T>` newtype |
| 4 | Ed25519 signed manifests | Validates agent/skill manifests against supply-chain injection |
| 5 | SSRF protection | Blocks `web_fetch` from private IPs, metadata endpoints, DNS rebinding |
| 6 | Secret zeroization | `Zeroizing<String>` auto-wipes credentials on drop |
| 7 | OFP mutual auth | HMAC-SHA256 for A2A peer authentication |
| 8 | Capability gates | RBAC tool allowlists via `CapabilityManager` (deny-by-default) |
| 11 | Subprocess sandbox | `env_clear()` + allowlist for shell/Python/Node skills |
| 12 | Prompt injection scanner | Scans user messages and tool results before LLM prompt |
| 13 | Loop guard | SHA256 cycle detection + 50-iteration hard limit |
| 15 | Path traversal prevention | `std::fs::canonicalize()` + base directory check |

### Credential Resolution Chain

```rust
// Priority: vault -> .env -> env var -> interactive prompt
pub fn resolve_credential(&self, key: &str) -> Result<Zeroizing<String>> {
    // 1. AES-256-GCM encrypted vault (~/.openfang/vault.enc)
    // 2. Dotenv file (~/.openfang/.env)
    // 3. std::env::var
    // 4. Interactive prompt (CLI last resort)
}
```

Master key sourced from: OS keyring (preferred) -> `OPENFANG_VAULT_KEY` env var (CI) -> manual backup.

### Approval System

Tools classified by risk level:
- **Low** (auto-approve): `kg_query`, `list_files`
- **Medium** (auto-approve): `web_fetch`, `web_search`
- **High** (require approval): `shell_exec`, `write_file`
- **Critical** (require approval): `docker_run`, `delete_file`

Approval request has 60s timeout, auto-denies on timeout. 5 pending per agent, 100 recent in memory. `--yolo` flag disables all approval gates.

### Inter-Agent Recursion Guard

```rust
tokio::task_local! {
    static AGENT_CALL_DEPTH: std::cell::Cell<u32>;
}
const MAX_AGENT_CALL_DEPTH: u32 = 5;
```

Each `agent_send`/`agent_spawn` increments depth. Task-local scoping ensures concurrent agent calls maintain independent counters.

## Key Patterns for AGH

### 1. Unified Tool Catalog with Prefix-Based Routing

**Pattern**: All extension types (builtin, MCP, skill, inter-agent) present tools through the same `ToolDefinition` struct. The `ToolRunner` routes by name prefix. The LLM sees a single flat list.

**AGH relevance**: AGH already has a similar concept with `AgentDriver` implementations. The key insight is that tool namespacing (e.g., `mcp_github_search`) prevents collisions and makes provenance transparent without requiring a separate routing layer.

### 2. Trait-Based Extension Points with Crate Boundaries

**Pattern**: Extension interfaces are traits defined in downstream crates (`ChannelAdapter` in channels crate, `KernelHandle` in runtime crate). The kernel crate implements them. This avoids circular dependencies while allowing clean extension.

**AGH relevance**: AGH's `session/` defines `AgentDriver`, `acp/` implements it -- same pattern. Could extend this to `ChannelAdapter`-style traits for external integrations.

### 3. TOML Manifests for Everything

**Pattern**: Every extension type has a TOML manifest: `SKILL.toml` for skills, `HAND.toml` for hands, `config.toml` sections for channels, `integrations.toml` for MCP servers. Manifests are the declarative contract; code implements the behavior.

**AGH relevance**: AGH already uses TOML config. The skill manifest pattern (declaring runtime, tools, requirements) is directly applicable for an AGH extension system.

### 4. Compile-Time Composition with Runtime Escape Hatches

**Pattern**: Core capabilities are compiled in (auditability, air-gap, speed). Runtime extensibility exists but is sandboxed (WASM, subprocess isolation, MCP transport isolation).

**AGH relevance**: For AGH Phase 2/3, this suggests: compile critical skills into the binary, allow runtime skill installation with subprocess sandbox, and use MCP as the interop layer for third-party tools.

### 5. Multi-Runtime Skill Execution

**Pattern**: Skills declare their runtime type, and the loader dispatches to the appropriate executor. Communication is JSON over stdin/stdout for subprocess runtimes. Each runtime gets its own sandbox profile.

**AGH relevance**: If AGH adds a skill/extension system, the `SkillRuntime` enum pattern (Python, Shell, WASM, PromptOnly) with stdin/stdout JSON protocol is simple and proven. The `PromptOnly` runtime (inject markdown into system prompt) is particularly elegant for knowledge-only extensions.

### 6. Credential Vault with Zeroization

**Pattern**: AES-256-GCM encrypted vault with OS keyring integration, `Zeroizing<String>` wrapper that auto-wipes on drop, and a 4-tier resolution chain (vault -> .env -> env var -> interactive).

**AGH relevance**: AGH will need credential management for MCP servers and integrations. The vault pattern with zeroization is a strong security baseline.

### 7. Channel Adapter as Stream Abstraction

**Pattern**: Each channel adapter returns an async `Stream<Item = ChannelMessage>`. The bridge manager polls streams. This hides all transport complexity (WebSocket, long-polling, webhooks) behind a uniform interface.

**AGH relevance**: If AGH needs to receive messages from external platforms, the stream-based adapter pattern is the cleanest approach. The `ChannelBridgeHandle` trait pattern for breaking circular deps is also directly useful.

### 8. Hands as Meta-Extensions

**Pattern**: Hands compose tools + skills + prompts + schedules + requirements + settings into a named, distributable, marketplace-ready agent personality. The `HandRequirement` system with platform-specific install info provides excellent UX.

**AGH relevance**: This is essentially what AGH sessions could evolve into with configuration presets. The HAND.toml manifest pattern (declaring required tools, configurable settings, dashboard metrics) is directly applicable for AGH agent templates.

### 9. Workflow Engine as Orchestration Layer

**Pattern**: The workflow engine does not add execution capability -- it only orchestrates when and how existing agent loops run. Steps are pure data (prompt template + mode + error handling). Variable interpolation connects steps.

**AGH relevance**: If AGH adds multi-agent workflows, the step mode taxonomy (Sequential, FanOut, Collect, Conditional, Loop) covers the essential orchestration patterns. The "workflow adds no execution capability" principle keeps the system simple.

### 10. Defense-in-Depth with Composable Security Layers

**Pattern**: 16 security layers, each addressing a distinct attack surface, applied structurally (not optionally). Complete mediation: every tool call passes through capability check, approval check, taint check, and sandbox check.

**AGH relevance**: AGH should plan security layers early. The minimal set for extensions: capability gates (tool allowlists), subprocess sandbox (`env_clear` + allowlist), credential isolation, and an approval system for high-risk operations.

## Code References

### Core Extension Infrastructure

| File | Purpose |
|------|---------|
| `crates/openfang-runtime/src/tool_runner.rs` | Central tool dispatch, capability enforcement, taint checks |
| `crates/openfang-runtime/src/mcp.rs` | MCP client: `McpConnection`, `McpTransport`, tool namespacing |
| `crates/openfang-runtime/src/kernel_handle.rs` | `KernelHandle` trait: inter-agent operations, memory, tasks |
| `crates/openfang-kernel/src/kernel.rs` | `OpenFangKernel` struct: composition root with all subsystems |
| `crates/openfang-kernel/src/capabilities.rs` | `CapabilityManager`: RBAC tool allowlists |
| `crates/openfang-kernel/src/workflow.rs` | `WorkflowEngine`, `WorkflowStep`, `StepMode` |

### Skills and Hands

| File | Purpose |
|------|---------|
| `crates/openfang-skills/src/lib.rs` | `SkillManifest`, `SkillRuntime` enum, `SkillToolDef` |
| `crates/openfang-skills/src/registry.rs` | `SkillRegistry`: load/freeze/snapshot, bundled + filesystem skills |
| `crates/openfang-skills/src/loader.rs` | `execute_skill_tool`: dispatches to Python/Node/Shell/WASM runtimes |
| `crates/openfang-skills/src/verify.rs` | `SkillVerifier`: prompt injection scanning |
| `crates/openfang-hands/src/lib.rs` | `HandDefinition`, `HandInstance`, `HandSetting`, `HandRequirement` |
| `crates/openfang-hands/src/registry.rs` | `HandRegistry`: load, activate, deactivate hands |

### Channel Adapters

| File | Purpose |
|------|---------|
| `crates/openfang-channels/src/types.rs` | `ChannelAdapter` trait, `ChannelMessage`, `ChannelContent` |
| `crates/openfang-channels/src/bridge.rs` | `BridgeManager`, `ChannelBridgeHandle` trait, chat commands |
| `crates/openfang-channels/src/router.rs` | `AgentRouter`: 5-level priority routing |
| `crates/openfang-channels/src/telegram.rs` (etc.) | Individual adapter implementations |

### Extensions and Security

| File | Purpose |
|------|---------|
| `crates/openfang-extensions/src/lib.rs` | `IntegrationTemplate`, `McpTransportTemplate`, `RequiredEnvVar` |
| `crates/openfang-extensions/src/registry.rs` | `IntegrationRegistry`: bundled + installed MCP templates |
| `crates/openfang-extensions/src/credentials.rs` | `CredentialResolver`: vault -> .env -> env var chain |
| `crates/openfang-extensions/src/vault.rs` | `CredentialVault`: AES-256-GCM encrypted storage |
| `crates/openfang-extensions/src/installer.rs` | `install_integration`: one-click MCP server setup |
| `crates/openfang-extensions/src/oauth.rs` | OAuth2 PKCE flow for Google/GitHub/Slack |
| `crates/openfang-types/src/taint.rs` | `TaintedValue`, `TaintLabel`, `TaintSink` |
| `crates/openfang-types/src/manifest_signing.rs` | Ed25519 manifest signing/verification |
| `crates/openfang-runtime/src/subprocess_sandbox.rs` | `env_clear()` + allowlist subprocess isolation |
| `crates/openfang-runtime/src/sandbox.rs` | `WasmSandbox`: Wasmtime fuel + epoch metering |
| `crates/openfang-runtime/src/audit.rs` | `AuditLog`: Merkle hash-chain append-only log |

### Configuration

| File | Purpose |
|------|---------|
| `openfang.toml.example` | Example config showing all sections |
| `crates/openfang-types/src/config.rs` | `KernelConfig`, `ChannelOverrides`, `DmPolicy`, `GroupPolicy` |
| `crates/openfang-types/src/capability.rs` | `Capability` enum, `capability_matches` |

### Workspace

| File | Purpose |
|------|---------|
| `Cargo.toml` (root) | 14-crate workspace definition |
| `crates/openfang-wire/` | OFP peer network protocol, HMAC-SHA256 mutual auth |
