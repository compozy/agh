# Hermes Extension Architecture Analysis

## Overview

Hermes is a Python-based AI agent harness built around a modular tool-calling architecture. Its extension system spans five major subsystems: a **self-registering tool registry**, a **skills pipeline** (markdown-based procedural memory), **MCP tool integration** (dynamic external tool servers), an **ACP adapter** (IDE integration protocol), and a **plugin system** (user/project/pip-distributed extensions). All extension paths converge on a single `ToolRegistry` singleton that the agent loop queries for schema assembly and dispatches through for tool execution.

The architecture is designed for graceful degradation: missing dependencies shrink the tool surface rather than blocking startup. Tools, skills, MCP servers, and plugins all fail soft -- import errors, missing API keys, and unavailable servers are logged and skipped.

### Source Root

All code references below are relative to:
```
/Users/pedronauck/dev/knowledge/.resources/hermes/
```

## Tool Registry & Dispatch

### Registry Singleton (`tools/registry.py`)

The entire tool system is built on a module-level singleton `ToolRegistry` class:

```python
class ToolRegistry:
    def __init__(self):
        self._tools: Dict[str, ToolEntry] = {}
        self._toolset_checks: Dict[str, Callable] = {}

registry = ToolRegistry()  # module-level singleton
```

Each tool is a `ToolEntry` with `__slots__`:
- `name` -- unique identifier
- `toolset` -- grouping key (e.g., "web", "terminal", "mcp-github")
- `schema` -- OpenAI-format function schema dict
- `handler` -- callable `(args_dict, **kwargs) -> str`
- `check_fn` -- optional availability predicate (returns bool)
- `requires_env` -- list of env var names (for `/doctor` diagnostics)
- `is_async` -- whether handler returns a coroutine
- `description`, `emoji` -- display metadata
- `max_result_size_chars` -- per-tool output cap

**File:** `tools/registry.py` (291 lines)

### Self-Registration Pattern

Every tool file imports the singleton and calls `registry.register()` at module scope:

```python
from tools.registry import registry

def web_search(query: str, task_id: str = None) -> str:
    ...

registry.register(
    name="web_search",
    toolset="web",
    schema={...},
    handler=lambda args, **kw: web_search(
        query=args.get("query", ""),
        task_id=kw.get("task_id")
    ),
    check_fn=check_requirements,
    requires_env=["FIRECRAWL_API_KEY"],
)
```

Key patterns:
1. **Handlers are lambdas that unpack args** -- each receives `(dict, **kwargs)` and unpacks into named args for the real function.
2. **All handlers return JSON strings** -- `json.dumps(dict)` is the universal return contract.
3. **`check_fn()` gates availability** -- if it returns False, the tool is withheld from the schema list; the model never sees it.
4. **Name collisions log a warning** -- second registration wins silently (with a log).

### Discovery Pipeline (`model_tools.py`)

Discovery is a one-shot `_discover_tools()` function that imports ~21 tool modules, triggering their side-effect registrations:

```python
def _discover_tools():
    _modules = [
        "tools.web_tools",
        "tools.terminal_tool",
        "tools.file_tools",
        # ... 18 more modules
    ]
    for mod_name in _modules:
        try:
            importlib.import_module(mod_name)
        except Exception as e:
            logger.warning("Could not import tool module %s: %s", mod_name, e)

_discover_tools()

# MCP servers (external)
from tools.mcp_tool import discover_mcp_tools
discover_mcp_tools()

# User plugins
from hermes_cli.plugins import discover_plugins
discover_plugins()
```

Registration failures are soft -- missing optional dependencies just shrink the toolbelt.

**File:** `model_tools.py` (578 lines)

### Dispatch Contract

The dispatch entry point is `handle_function_call()`:

```python
def handle_function_call(function_name, function_args, task_id=None, ...) -> str:
    function_args = coerce_tool_args(function_name, function_args)  # "42" -> 42
    # Plugin pre-hook
    invoke_hook("pre_tool_call", tool_name=function_name, args=function_args, ...)
    result = registry.dispatch(function_name, function_args, task_id=task_id, ...)
    # Plugin post-hook
    invoke_hook("post_tool_call", tool_name=function_name, result=result, ...)
    return result
```

The dispatch wraps exceptions as `{"error": "..."}` JSON strings -- the model can reason about errors and retry.

**Special tools** (`todo`, `memory`, `session_search`, `delegate_task`) are intercepted by the agent loop before reaching `handle_function_call()` because they need agent-level state.

### Toolsets (`toolsets.py`)

Toolsets group related tools for bulk enable/disable with recursive composition:

```python
TOOLSETS = {
    "web": {
        "description": "Web research and content extraction tools",
        "tools": ["web_search", "web_extract"],
        "includes": [],
    },
    "debugging": {
        "tools": ["terminal", "process"],
        "includes": ["web", "file"],
    },
    "hermes-cli": {
        "tools": _HERMES_CORE_TOOLS,  # ~40 tools
        "includes": [],
    },
}

def resolve_toolset(name: str, visited: Set[str] = None) -> List[str]:
    # Recursive resolution with cycle detection
```

Platform-specific toolsets (e.g., `hermes-telegram`, `hermes-discord`, `hermes-acp`) enumerate exactly which tools are available per interface. MCP tools are injected into `hermes-*` umbrella toolsets automatically.

The `get_tool_definitions()` function resolves toolsets, runs `check_fn()` for each tool, dynamically rebuilds schemas that reference other tools (e.g., `execute_code`'s sandbox tools), and stores the resolved tool names in `_last_resolved_tool_names` (a module global used by code execution and delegation).

**File:** `toolsets.py` (643 lines)

### Deregistration

The registry supports `deregister(name)` for dynamic tool removal (used by MCP when servers send `notifications/tools/list_changed`). It also cleans up the toolset check if the removed tool was the last in its toolset.

### Helper Functions

`tools/registry.py` exports `tool_error()` and `tool_result()` helper functions that eliminate boilerplate JSON serialization across all 50+ tool files.

## Skills Pipeline

### Architecture

Skills are markdown-based procedural memory stored as directory-structured packages under `~/.hermes/skills/`:

```
~/.hermes/skills/
  mlops/
    axolotl/
      SKILL.md           # Main instructions (required)
      references/        # Supporting docs
      templates/         # Output templates
      assets/            # Supplementary files (agentskills.io standard)
      scripts/           # Executable helpers
```

Each `SKILL.md` has YAML frontmatter (agentskills.io compatible):

```yaml
---
name: axolotl
description: "Fine-tuning LLMs with Axolotl"
version: 1.0.0
platforms: [macos, linux]       # OS filter
required_environment_variables:
  - name: HF_TOKEN
    prompt: "Enter Hugging Face token"
    help: "https://huggingface.co/settings/tokens"
metadata:
  hermes:
    tags: [fine-tuning, llm]
---
```

**File:** `tools/skills_tool.py` (1377 lines)

### Progressive Disclosure (3-Tier)

1. **`skills_list(category=)`** -- Returns name + description only (token-efficient).
2. **`skill_view(name)`** -- Returns full SKILL.md content + linked_files dict.
3. **`skill_view(name, file_path)`** -- Returns a specific linked file.

This tiered approach minimizes context window usage -- the model only loads full skill content when it determines a skill is relevant.

### Activation Semantics

Skill content is injected as a **user message**, not a system prompt modification:

```python
messages.append({
    "role": "user",
    "content": f"[Skill activated: {skill_name}]\n\n{loaded_content}"
})
```

This preserves Anthropic's prompt caching -- the system prompt stays constant, the cache stays warm. Only the new user message costs full token pricing.

### Skill Readiness

Skills declare environment requirements via `required_environment_variables` and `required_credential_files`. On `skill_view()`, the system:
1. Checks if each required env var is set (in `~/.hermes/.env` or `os.environ`)
2. If CLI is interactive, triggers a secret-capture callback for missing vars
3. If gateway, tells the user to configure via CLI
4. Reports `readiness_status`: `"available"`, `"setup_needed"`, or `"unsupported"`

### Platform Filtering

Skills can declare `platforms: [macos, linux]` to restrict availability by OS. `skill_matches_platform()` checks `sys.platform` against the declared list.

### Security

`skill_view()` includes:
- **Path traversal prevention** (`..` in file_path is rejected)
- **Resolved path boundary check** (must stay within skill directory)
- **Prompt injection detection** (scans for common injection patterns like "ignore previous instructions")
- **Trust boundary warning** (skills outside `~/.hermes/skills/` get logged)

### Skill Manager Tool

`tools/skill_manager_tool.py` provides `skill_manage(action, skill_name, content)` for:
- `propose` -- Agent auto-creates a skill from a completed task
- `edit` -- Modify an existing skill
- `delete` -- Remove a skill

### System Prompt Integration

Only the skill **index** (name + description) is injected into the system prompt via `build_skills_system_prompt()`. Individual skill bodies are loaded on demand. This keeps the system prompt compact and stable for caching.

### External Skills Dirs

Skills can also be loaded from directories configured via `skills.external_dirs` in config, allowing workspace-local skills that coexist with the global `~/.hermes/skills/`.

## MCP Tool Integration

### Architecture (`tools/mcp_tool.py`)

MCP integration uses a dedicated background event loop in a daemon thread to manage long-lived async connections:

```
Main Thread                     Background Thread (_mcp_loop)
  |                               |
  |-- discover_mcp_tools() -----> |-- MCPServerTask(github)
  |                               |     |-- stdio_client() -> subprocess
  |                               |     |-- ClientSession.initialize()
  |                               |     |-- list_tools() -> register in registry
  |                               |
  |-- registry.dispatch() ------> |-- session.call_tool()
  |   (via run_coroutine_threadsafe)
```

**File:** `tools/mcp_tool.py` (2187 lines -- the largest single tool file)

### Transport Types

- **Stdio** -- Launch subprocess (`npx -y @modelcontextprotocol/server-github`), communicate over stdin/stdout
- **HTTP/StreamableHTTP** -- Connect to remote endpoint with optional OAuth 2.1 PKCE

### Configuration

```yaml
mcp_servers:
  github:
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_TOKEN}"
    timeout: 120
    connect_timeout: 60
    tools:
      include: [create_issue, list_issues]  # whitelist
      exclude: [delete_repo]                # blacklist
      resources: true
      prompts: true
    sampling:
      enabled: true
      model: "gemini-3-flash"
      max_tokens_cap: 4096
```

`${VAR}` placeholders are resolved from `os.environ`.

### Schema Mapping

MCP tools are transformed before registry insertion:
1. **Prefixing** -- `create_issue` from `github` becomes `mcp_github_create_issue`
2. **Sanitization** -- Hyphens replaced with underscores for LLM compatibility
3. **Input schema normalization** -- Missing `properties` keys are filled in

### Dynamic Tool Discovery

When an MCP server sends `notifications/tools/list_changed`, Hermes:
1. Fetches the new tool list
2. Deregisters all old tools for that server
3. Re-registers with the fresh list
4. Updates `hermes-*` umbrella toolsets

This is gated on `_MCP_MESSAGE_HANDLER_SUPPORTED` (SDK version check).

### Include/Exclude Filtering

Per-server tool filtering via `tools.include` / `tools.exclude`:
- `include` is a whitelist -- only listed MCP tool names are registered
- `exclude` is a blacklist -- all except listed are registered
- `include` takes precedence over `exclude`

### Utility Tools

For each MCP server, Hermes also registers utility tools:
- `mcp_{server}_list_resources` / `mcp_{server}_read_resource`
- `mcp_{server}_list_prompts` / `mcp_{server}_get_prompt`

These are gated on both config (`tools.resources`, `tools.prompts`) and server capability (whether the session has the corresponding method).

### Sampling Support

MCP servers can request LLM completions via the sampling protocol. The `SamplingHandler` class:
- Rate-limits requests (sliding window, configurable max RPM)
- Resolves model (config override > server hint > default)
- Enforces model whitelist
- Caps max_tokens
- Limits tool loop rounds
- Converts MCP message format to OpenAI format
- Offloads sync LLM calls to thread via `asyncio.to_thread()`

### Security

- **Environment filtering** -- Only `_SAFE_ENV_KEYS` (PATH, HOME, USER, etc.) plus explicitly configured vars are passed to stdio subprocesses
- **Credential redaction** -- `_CREDENTIAL_PATTERN` regex strips `ghp_*`, `sk-*`, `Bearer`, etc. from error messages before returning to LLM
- **OSV malware check** -- Before spawning a stdio server, the package is checked against the OSV malware database
- **Auto-reconnection** -- Exponential backoff up to 5 retries
- **Collision guard** -- MCP tools that collide with built-in tools are skipped

### Toolset Injection

After registration, MCP tools are:
1. Added to a custom `mcp-{server_name}` toolset
2. Injected into all `hermes-*` umbrella toolsets
3. Made available as standalone toolset aliases (e.g., `--enabled github`)

## ACP Adapter

### Architecture (`acp_adapter/`)

The ACP adapter exposes Hermes as an Agent Client Protocol server for IDE integration (VS Code, Zed, JetBrains). It wraps `AIAgent` instances in a stateful session server:

```
IDE Extension
  | JSON-RPC over stdio/HTTP
  v
HermesACPAgent (acp_adapter/server.py)
  |
  v
SessionManager (acp_adapter/session.py)
  |-- SessionState { session_id, agent, cwd, model, history }
  |-- Persisted to ~/.hermes/state.db
  |
  v
AIAgent.run_conversation()
```

**Files:**
- `acp_adapter/server.py` -- `HermesACPAgent` class (extends `acp.Agent`)
- `acp_adapter/session.py` -- `SessionManager` and `SessionState`
- `acp_adapter/events.py` -- Callback factories for streaming events
- `acp_adapter/permissions.py` -- Approval callback for dangerous commands
- `acp_adapter/auth.py` -- Provider detection

### Protocol Methods

- `initialize` -- Client capabilities handshake, returns agent capabilities
- `tools/list` -- List available tools from the registry
- `tools/call` -- Execute a tool
- `completion/complete` -- Run an agent turn
- `resources/read` -- Fetch memory, skill, or session

### Session Lifecycle

Sessions are keyed by client. Each gets:
- An `AIAgent` instance with its own message history
- Code context accumulation (open files, cursor position from IDE)
- Task-specific CWD overrides for terminal tools
- Cancel events for interruption
- Persistence to `~/.hermes/state.db` for survive-restart and `session_search`

### Slash Commands

The ACP server advertises IDE-facing commands: `/help`, `/model`, `/tools`, `/context`, `/reset`, `/compact`, `/version`.

### MCP Server Registration

ACP sessions register per-session MCP servers passed by the client via `_register_session_mcp_servers()`, allowing IDE extensions to bring their own MCP servers.

## Security & Approval Model

### Multi-Layer Defense-in-Depth (`tools/approval.py`)

```
Ring 4 (Outermost): Gateway Authorization -- who can talk to the agent?
Ring 3:              Command Detection -- what requires approval?
Ring 2:              Execution Isolation -- containerized auto-approve
Ring 1 (Core):       File/Config Protection -- never bypassable
```

### Dangerous Command Detection

`detect_dangerous_command()` runs regex patterns against normalized commands:

```python
DANGEROUS_PATTERNS = [
    (r'\brm\s+(-[^\s]*\s+)*/', "delete in root path"),
    (r'\brm\s+-[^\s]*r', "recursive delete"),
    (r'\bchmod\s+(-[^\s]*\s+)*(777|666|...)', "world-writable permissions"),
    (r'\bmkfs\b', "format filesystem"),
    (r'\b(curl|wget)\b.*\|\s*(ba)?sh\b', "pipe remote content to shell"),
    # ... 25+ patterns total
]
```

Normalization pipeline: ANSI strip -> null byte removal -> Unicode NFKC normalization.

**File:** `tools/approval.py` (878 lines)

### Tirith Security Scanner

Binary scanner for content-level threats (homograph URLs, terminal injection, pipe-to-interpreter). Downloads on first use, verified via SHA-256 checksum. Exit codes: 0=allow, 1=block, 2=warn.

### Approval State Machine

Three scopes:
| Scope | Duration | Storage |
|-------|----------|---------|
| Once | Single execution | In-memory (discarded) |
| Session | Current session | `_session_approved[session_key]` |
| Permanent | Across sessions | `config.yaml` `command_allowlist` |

Thread-safe with `threading.Lock`. Per-session approval keys support legacy aliases for backward compatibility.

### Smart Approval

When `approvals.mode=smart`, an auxiliary LLM assesses risk before prompting the user:
- `APPROVE` -- Auto-approve, grant session-level approval
- `DENY` -- Block permanently
- `ESCALATE` -- Fall through to manual prompt

### Gateway Approval (Blocking Queue)

For messaging platforms, approval uses a blocking queue pattern:
1. Agent thread creates `_ApprovalEntry` with `threading.Event`
2. Gateway callback sends approval request to user in chat
3. Agent thread blocks on `entry.event.wait(timeout=300)`
4. User replies `/approve` or `/deny`
5. `resolve_gateway_approval()` sets `entry.result` and signals the event

Multiple parallel subagents can block concurrently -- each gets its own entry.

### Container Isolation Bypass

Commands in Docker, Singularity, Modal, Daytona backends auto-approve (no host damage possible).

### YOLO Mode

`HERMES_YOLO_MODE` bypasses all approval prompts. Intended for ephemeral containers, batch runs, and CI/CD.

### URL Safety (SSRF Protection)

`tools/url_safety.py` blocks access to private networks (RFC 1918), loopback, cloud metadata endpoints (169.254.169.254), and configurable domain blocklists.

### File Write Safety

Hard safety boundary (not bypassable by YOLO mode):
- System files: `/etc/passwd`, `/etc/shadow`, etc.
- Hermes internals: `~/.hermes/auth.json`, `~/.hermes/config.yaml`
- Shell configs: `~/.bashrc`, `~/.zshrc`

## Extension Patterns

### Plugin System (`hermes_cli/plugins.py`)

Three plugin sources:
1. **User plugins** -- `~/.hermes/plugins/<name>/` with `plugin.yaml` + `__init__.py`
2. **Project plugins** -- `./.hermes/plugins/<name>/` (opt-in via `HERMES_ENABLE_PROJECT_PLUGINS`)
3. **Pip plugins** -- packages exposing `hermes_agent.plugins` entry-point group

**File:** `hermes_cli/plugins.py` (612 lines)

#### Plugin Manifest (`plugin.yaml`)

```yaml
name: my-plugin
version: 1.0.0
description: "Example plugin"
author: "User"
requires_env: [MY_API_KEY]
provides_tools: [my_tool]
provides_hooks: [pre_tool_call, post_tool_call]
```

#### PluginContext API

Each plugin's `register(ctx)` receives a `PluginContext` with:

```python
ctx.register_tool(name, toolset, schema, handler, ...)   # -> tools.registry.register()
ctx.register_hook(hook_name, callback)                     # -> lifecycle hooks
ctx.register_cli_command(name, help, setup_fn)             # -> argparse subcommand
ctx.inject_message(content, role="user")                   # -> inject into conversation
```

#### Lifecycle Hooks

```python
VALID_HOOKS = {
    "pre_tool_call",       # Before tool dispatch
    "post_tool_call",      # After tool dispatch
    "pre_llm_call",        # Before LLM inference (can inject context)
    "post_llm_call",       # After LLM inference
    "pre_api_request",     # Before HTTP API call
    "post_api_request",    # After HTTP API call
    "on_session_start",    # Session created
    "on_session_end",      # Session ended
    "on_session_finalize", # Session finalized
    "on_session_reset",    # Session reset
}
```

Hooks are invoked via `invoke_hook(name, **kwargs)` -- each callback is wrapped in try/except so a misbehaving plugin cannot break the agent loop.

The `pre_llm_call` hook can return context to inject into the user message (preserving prompt cache).

#### Plugin Toolset Integration

Plugin-registered tools appear as their own toolsets. The `_get_plugin_toolset_names()` function discovers toolset names from the registry that don't exist in the static `TOOLSETS` dict. These are included in `get_all_toolsets()` and `resolve_toolset()` so plugin tools participate in the standard enable/disable flow.

### Code Execution Sandbox (PTC) (`tools/code_execution_tool.py`)

Programmatic Tool Calling -- collapses multi-step tool chains into single Python scripts:

```
Parent Process (Hermes)          Child Process (Sandbox)
  |                                |
  |-- Generate hermes_tools.py --->|
  |-- Open UDS, start RPC thread   |
  |-- Spawn child process -------->|
  |                                |-- import hermes_tools
  |                                |-- hermes_tools.web_search("query")
  |   <-- JSON-RPC over UDS -------|
  |-- Dispatch via registry         |
  |-- Return result over UDS ----->|
  |                                |-- process results
  |                                |-- print() to stdout
  |   <-- Capture stdout -----------|
  |-- Return stdout to LLM         |
```

Security boundaries:
- **Whitelisted tools only**: `SANDBOX_ALLOWED_TOOLS` = {web_search, web_extract, read_file, write_file, search_files, patch, terminal}
- **Execution timeout**: 300s
- **Call volume limit**: 50 RPC requests per script
- **Output capping**: 50KB stdout, 10KB stderr
- **Credential scrubbing**: Sensitive env vars stripped from child process
- **Filesystem isolation**: Runs in tempdir

Remote backends (Docker, SSH, Modal) use file-based RPC instead of UDS.

### Subagent Delegation (`tools/delegate_tool.py`)

`delegate_task` spawns isolated child agents:

```python
subagent = AIAgent(
    model=model or parent_agent.model,
    max_iterations=max_iterations,
    enabled_toolsets=parent_agent.enabled_toolsets,
    platform="subagent",
    session_id=str(uuid.uuid4()),
)

# CRITICAL: save/restore global tool names
saved_tool_names = _last_resolved_tool_names.copy()
try:
    result = subagent.run_conversation(task)
finally:
    _last_resolved_tool_names[:] = saved_tool_names
```

Properties: own thread (ThreadPoolExecutor), inherits parent's toolset/memory/skills, independent message history and session ID. Multiple subagents can run in parallel.

## Key Patterns for AGH

### 1. Self-Registering Singleton Registry

Hermes' most important pattern. Each tool file owns its registration -- no central manifest to maintain. Adding a tool is a single-file operation. The registry is the stable interface between schema assembly and dispatch.

**AGH consideration:** Go doesn't have import-time side effects. Consider `init()` functions in tool packages that register with a central registry, or a declarative approach where the daemon package wires tools.

### 2. Availability Gating via `check_fn()`

Tools that can't run are withheld from the model entirely. The model never hallucinates calls to unavailable tools. This is the single most important reliability property.

**AGH consideration:** Map to the `AgentDriver` interface -- drivers should report which tools they support, and session setup should filter accordingly.

### 3. Toolset Composition with Recursive Resolution

Toolsets group tools for bulk enable/disable with recursive includes and cycle detection. Platform-specific toolsets define exactly which tools each interface exposes.

**AGH consideration:** AGH already has agent definitions in TOML config. Extend with toolset grouping that supports composition (e.g., "research" includes "web" + "vision").

### 4. Graceful Degradation via Soft Failures

Every extension point (tool imports, MCP connections, plugin loading) is wrapped in try/except. Missing deps shrink capability, they don't block startup.

**AGH consideration:** Critical for a daemon that must stay running. Use the same pattern for ACP driver spawning -- if an agent binary is missing, the session should report the error rather than crashing the daemon.

### 5. Plugin Hooks at Tool Dispatch Points

Pre/post hooks at tool calls and LLM calls enable observability, context injection, and cross-cutting concerns without modifying core code.

**AGH consideration:** AGH's `Notifier` pattern already provides fan-out for observability. Extend with pre/post hooks on session events and tool dispatch for plugin-like extensibility.

### 6. MCP as First-Class Registry Citizens

MCP tools are registered in the same registry as built-in tools. The agent loop doesn't distinguish between them. This transparency enables MCP tools to participate in toolset filtering, subagent delegation, and code execution.

**AGH consideration:** AGH already has MCP support via the ACP protocol. Ensure MCP tools discovered by the agent subprocess are surfaced through the session's tool list.

### 7. User-Message Injection for Skills (Cache Preservation)

Skills are injected as user messages rather than system prompt modifications to preserve prompt caching. The system prompt stays constant across turns.

**AGH consideration:** When implementing skills/memory for AGH, inject context as conversation messages rather than modifying the system prompt, to preserve whatever caching the underlying agent supports.

### 8. Three-Scope Approval State Machine

Approvals at once/session/permanent granularity with thread-safe state. Smart approval via auxiliary LLM reduces human-in-the-loop friction.

**AGH consideration:** AGH's approval model should live in the daemon (not the agent), since the daemon mediates between the user and the agent. Store approval state per-session in SQLite.

### 9. Mutable Global State as Design Tension

`_last_resolved_tool_names` is a module-level mutable global that subagents must save/restore. This is explicitly called out as a design tension -- a cleaner design would thread it through function arguments.

**AGH consideration:** Avoid this pattern. Go's explicit argument passing makes it natural to thread session-scoped state through function calls rather than relying on globals.

### 10. Dynamic Tool Discovery and Hot-Reload

MCP servers can trigger tool list refreshes via `notifications/tools/list_changed`. The registry supports `deregister()` for nuke-and-repave updates.

**AGH consideration:** If AGH supports MCP servers that change their tool lists, the session manager needs to handle tool list invalidation and notify the agent.

## Code References

| Component | File | Lines | Description |
|-----------|------|-------|-------------|
| Tool Registry | `tools/registry.py` | 336 | Singleton registry, ToolEntry, dispatch, helpers |
| Model Tools | `model_tools.py` | 578 | Discovery pipeline, get_tool_definitions, handle_function_call |
| Toolsets | `toolsets.py` | 643 | TOOLSETS dict, resolve_toolset, composition |
| MCP Tool | `tools/mcp_tool.py` | 2187 | MCPServerTask, discovery, schema mapping, sampling |
| Plugins | `hermes_cli/plugins.py` | 612 | PluginManager, PluginContext, hooks, discovery |
| Skills Tool | `tools/skills_tool.py` | 1377 | skills_list, skill_view, readiness checks |
| Approval | `tools/approval.py` | 878 | Detection patterns, approval state machine, smart approval |
| ACP Server | `acp_adapter/server.py` | ~300 | HermesACPAgent, protocol methods |
| ACP Session | `acp_adapter/session.py` | ~200 | SessionManager, SessionState, persistence |
| Code Execution | `tools/code_execution_tool.py` | ~800 | PTC sandbox, UDS RPC, stub generation |
| Delegate Tool | `tools/delegate_tool.py` | ~250 | Subagent spawning, global state save/restore |
| Skill Manager | `tools/skill_manager_tool.py` | ~200 | propose/edit/delete skills |
| URL Safety | `tools/url_safety.py` | ~150 | SSRF protection, private network blocking |
| Tirith Security | `tools/tirith_security.py` | ~200 | Binary scanner integration |
| Skill Utils | `agent/skill_utils.py` | ~300 | Frontmatter parsing, platform matching |
| MCP Config | `hermes_cli/mcp_config.py` | ~100 | MCP server config loading |
| Plugin Commands | `hermes_cli/plugins_cmd.py` | ~100 | CLI for plugin management |

### Wiki Sources

| Document | Path |
|----------|------|
| Tool Registry and Dispatch | `/Users/pedronauck/dev/knowledge/hermes/wiki/concepts/Tool Registry and Dispatch.md` |
| Agent Skills Pipeline | `/Users/pedronauck/dev/knowledge/hermes/wiki/concepts/Agent Skills Pipeline.md` |
| Code Execution and MCP Tools | `/Users/pedronauck/dev/knowledge/hermes/wiki/concepts/Code Execution and MCP Tools.md` |
| ACP Adapter and Subagents | `/Users/pedronauck/dev/knowledge/hermes/wiki/concepts/ACP Adapter and Subagents.md` |
| Security and Command Approval | `/Users/pedronauck/dev/knowledge/hermes/wiki/concepts/Security and Command Approval.md` |
