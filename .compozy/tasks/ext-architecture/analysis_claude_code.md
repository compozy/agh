# Claude Code Extension Architecture Analysis

## Overview

Claude Code implements a multi-layered extensibility system composed of four interlocking subsystems:

1. **Plugin System** -- distributable bundles of capabilities (tools, MCP servers, hooks, skills, slash commands) discovered through marketplaces and installed to an on-disk cache.
2. **Skill System** -- Markdown-based agentic workflows (`SKILL.md` files with YAML frontmatter) that inject procedures into the agent's context on activation.
3. **Hook System** -- event-driven lifecycle interceptors that can verify, modify, or block actions at 25+ lifecycle points.
4. **MCP Integration** -- Model Context Protocol client that connects to external servers (stdio, SSE, HTTP, WebSocket, claude-ai proxy, in-process) and exposes their tools/resources/prompts as first-class citizens.

The key architectural insight is the separation of concerns: **plugins package capabilities**, **skills package procedures**, **hooks intercept lifecycle events**, and **MCP bridges external servers**. These four systems compose orthogonally -- a plugin can bundle skills, hooks, and MCP servers together as a single installable unit.

### Three-Layer State Model

The plugin system uses a three-layer reconciliation model that is the core design decision for robustness:

| Layer | Storage | Reconciler |
|-------|---------|-----------|
| **Intent** (settings) | `.claude/settings.json` at user/project/local/managed scope | `pluginStartupCheck.ts` |
| **Materialization** (disk) | `~/.claude/plugins/cache/<marketplace>/<plugin>/<version>/` | `reconcileMarketplaces()` |
| **Activation** (runtime) | `AppState.plugins`, tool registry, command registry, hook registry | `refreshActivePlugins()` |

This means a plugin can be configured but not installed, installed but not enabled, or enabled but not loaded -- each state is reconciled independently, making the system resilient to partial failures.

---

## Extension Loading & Discovery

### Plugin Discovery Flow

```
Marketplace (GitHub/URL) --> DiscoverPlugins (UI) --> installPluginFromMarketplace
                                                             |
                                                             v
                                          ~/.claude/plugins/cache/
                                          installed_plugins.json
                                                             |
                                                             v
                                          refreshActivePlugins()
                                            clearAllCaches()
                                            loadAllPlugins()
                                            extract commands/agents/hooks
                                            bump mcp.pluginReconnectKey
                                            re-init LSP
                                                             |
                                                             v
                                          AppState.plugins
                                          Commander Registry
                                          MCPConnectionManager
```

**Source**: `src/services/plugins/pluginOperations.ts` defines five core lifecycle operations as pure library functions: `installPluginOp`, `uninstallPluginOp`, `updatePluginOp`, `enablePluginOp`, `disablePluginOp`. These are consumed by both CLI commands and the interactive UI.

### Plugin Installation Scopes

Plugins can be scoped to four levels with strict precedence:

| Scope | Storage | Availability |
|-------|---------|-------------|
| **Managed** | `managed-settings.json` (MDM-deployed) | Enforced by org policy, cannot be overridden |
| **User** | `~/.claude/settings.json` | Global for current user |
| **Project** | `<repo>/.claude/settings.json` (committed) | This repository only |
| **Local** | `<repo>/.claude/settings.local.json` (gitignored) | This checkout on this machine |

**Source**: `src/services/plugins/pluginOperations.ts` lines 72-84 define `VALID_INSTALLABLE_SCOPES = ['user', 'project', 'local']` and `VALID_UPDATE_SCOPES` which adds 'managed'.

### Built-in Plugin Registry

**Source**: `src/plugins/builtinPlugins.ts`

Built-in plugins ship with the CLI binary and are registered via `registerBuiltinPlugin()`. They differ from bundled skills in that:
- They appear in the `/plugin` UI under a "Built-in" section
- Users can enable/disable them (persisted to user settings via `enabledPlugins`)
- They can provide multiple components (skills, hooks, MCP servers)

Plugin IDs use the format `{name}@builtin` to distinguish from marketplace plugins (`{name}@{marketplace}`).

```typescript
export type BuiltinPluginDefinition = {
  name: string
  description: string
  version: string
  defaultEnabled?: boolean
  isAvailable?: () => boolean
  skills?: BundledSkillDefinition[]
  hooks?: HooksSettings
  mcpServers?: Record<string, ScopedMcpServerConfig>
}
```

The `getBuiltinPlugins()` function splits registered plugins into enabled/disabled based on user settings with `defaultEnabled` as fallback.

### Skill Discovery

**Source**: `src/skills/loadSkillsDir.ts`

Skills are discovered from multiple sources in parallel:

1. **Managed skills**: `<MDM-path>/.claude/skills/` (policy-enforced)
2. **User skills**: `~/.claude/skills/`
3. **Project skills**: `.claude/skills/` at every directory from cwd up to home
4. **Additional directory skills**: `--add-dir` paths
5. **Legacy commands**: `.claude/commands/` directories (deprecated format)
6. **Bundled skills**: Compiled into the CLI binary (`src/skills/bundled/`)
7. **Dynamic skills**: Discovered at runtime as the agent reads/writes files
8. **Conditional skills**: Skills with `paths` frontmatter, activated when matching files are touched

The `SKILL.md` file format requires directory structure: `skill-name/SKILL.md`. Skills are deduplicated by resolved filesystem path (via `realpath`) to handle symlinks.

### Dynamic Skill Discovery

When the agent touches files (Read/Write/Edit), `discoverSkillDirsForPaths()` walks up from the file path to cwd, looking for `.claude/skills/` directories. Newly found directories are loaded via `addSkillDirectories()` and merged into the `dynamicSkills` map. This enables monorepo patterns where sub-packages have their own skill definitions.

### Conditional Skills (Path-Filtered)

Skills can declare a `paths` frontmatter field using gitignore-style patterns. These skills are held in a `conditionalSkills` map and only activated when the agent operates on files matching those patterns. Once activated, they move to `dynamicSkills` and become available to the model. This is a token-budget optimization -- skills near rarely-touched code don't consume context.

---

## Hook System

### Architecture

The hook system provides 25+ lifecycle events with five execution engine types. Hooks can verify, modify, or block actions before they happen, transform results after completion, or inject context into messages.

**Source**: `src/utils/hooks.ts` (main orchestrator), `src/utils/hooks/` (sub-modules)

### Lifecycle Events

**Session Events:**
- `SessionStart` -- fires once at REPL bootstrap, before any user message
- `SessionEnd` / `Stop` -- fires on graceful shutdown (tight 1.5s default timeout)
- `Setup` -- fires during initial setup

**Tool Lifecycle Events:**
- `PreToolUse` -- fires before each tool call; can block, modify input, or inject context
- `PostToolUse` -- fires after successful tool call; can transform results
- `PostToolUseFailure` -- fires after failed tool call

**User Interaction Events:**
- `UserPromptSubmit` -- fires when user submits message; can inject context or rewrite prompt
- `PermissionRequest` -- participates in multi-resolver permission race

**Context Events:**
- `PreCompact` / `PostCompact` -- fires around context compaction

**Agent Events:**
- `SubagentStart` / `SubagentStop` -- fires for subagent lifecycle
- `TaskCreated` / `TaskCompleted` -- fires for task lifecycle
- `TeammateIdle` -- fires when teammate becomes idle

**Other Events:**
- `ConfigChange` -- fires when configuration changes
- `CwdChanged` -- fires when working directory changes
- `FileChanged` -- fires when files change on disk
- `InstructionsLoaded` -- fires when CLAUDE.md instructions load
- `Elicitation` / `ElicitationResult` -- fires for MCP elicitation flows
- `PermissionDenied` -- fires when permission is denied
- `Notification` -- fires for system notifications

### Hook Types

**Source**: `src/utils/hooks.ts` lines 166-175 for timeouts; `src/types/hooks.ts` for type definitions

| Type | Execution | Typical Use |
|------|-----------|-------------|
| `command` | `child_process.spawn()` subprocess | Shell commands (lint, format, tests) |
| `prompt` | Side-query to Claude API | LLM-based verification or rewriting |
| `agent` | Forked sub-agent via `queryLoop()` | Multi-step verification with tool access |
| `http` | HTTP POST to endpoint | Remote automation, approval gates |
| `function` | In-process callback (SDK only) | Programmatic hooks via Agent SDK |

### Hook Configuration Schema

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "type": "command",
        "command": "prettier --check \"$file\"",
        "timeout": 30,
        "statusMessage": "Checking format..."
      }
    ]
  }
}
```

Fields: `matcher` (regex/glob filter), `type` (execution engine), `command`/`prompt` (payload), `timeout` (seconds), `statusMessage` (UI spinner text).

### Hook Sources

Hooks are discovered from four sources, merged at startup and on hot-reload:

1. **settings.json** -- five-tier settings cascade (policy/user/project/local/session)
2. **Plugins** -- plugin manifests declare hooks, loaded by `loadPluginHooks`
3. **Agent frontmatter** -- agents with markdown frontmatter can declare scoped hooks
4. **SDK callbacks** -- `function`-type hooks registered via Agent SDK

### Execution Lifecycle

```
Event occurs in query loop
      |
getRegisteredHooks(eventName) --> List<HookDef>
      |
Filter by matcher (matchHook)
      |
For each matching hook IN PARALLEL:
  command  --> child_process.spawn(), write stdin, read stdout
  prompt   --> side-query to Claude API
  agent    --> fork sub-agent loop
  http     --> POST event payload
  function --> invoke registered callback
      |
Each hook returns HookJSONOutput (or times out)
      |
Aggregate results:
  - If any returned continue=false --> raise HookBlockedError
  - Merge hookSpecificOutput in declaration order
  - Record duration (telemetry)
      |
Resume query loop
```

Hooks for the same event run **concurrently**. The orchestrator waits for all to complete (or timeout) before proceeding, because a single `continue: false` must block downstream execution.

### Hook Output Protocol

```json
{
  "continue": true,
  "stopReason": "(only if continue=false)",
  "hookSpecificOutput": {
    "permissionDecision": "allow | deny | ask",
    "updatedInput": { "...tool input overrides..." },
    "additionalContext": "(text to inject)",
    "transformedResult": "(replacement result)"
  }
}
```

**PreToolUse** can: allow (pass through), block (`continue: false`), or modify (`updatedInput`).
**PostToolUse** can: transform results via `transformedResult`.
**UserPromptSubmit** can: inject context via `additionalContext`.
**PermissionRequest** can: pre-empt dialog via `permissionDecision`.

### Timeouts

- **Tool hook timeout**: 10 minutes default (`TOOL_HOOK_EXECUTION_TIMEOUT_MS = 10 * 60 * 1000`)
- **Session end timeout**: 1.5 seconds default (`SESSION_END_HOOK_TIMEOUT_MS_DEFAULT = 1500`), configurable via `CLAUDE_CODE_SESSIONEND_HOOKS_TIMEOUT_MS`
- **Per-hook timeout**: overrides default via the `timeout` field

### MDM Policy Enforcement

- `shouldAllowManagedHooksOnly()` -- when true, only MDM-managed hooks execute; user/project hooks are ignored
- `shouldDisableAllHooksIncludingManaged()` -- kills the hook system entirely

---

## Tool Registration

### The Tool Interface

**Source**: `src/Tool.ts`

Every tool implements a single `Tool<Input, Output, Progress>` interface built via the `buildTool()` factory:

```typescript
type Tool<Input, Output, Progress> = {
  name: string
  aliases?: string[]
  searchHint?: string
  inputSchema: ZodSchema
  outputSchema?: ZodSchema

  // Execution
  call(args, context, canUseTool, parentMessage, onProgress?): Promise<ToolResult>
  isConcurrencySafe(): boolean
  isReadOnly(): boolean
  isDestructive?(): boolean

  // Permissions
  checkPermissions(input, context): Promise<PermissionResult>
  validateInput?(input, context): Promise<ValidationResult>

  // UI Rendering (React components)
  prompt(): Promise<string>  // System prompt text teaching the model
  renderToolUseMessage(): ReactNode
  renderToolResultMessage?(): ReactNode
  renderToolUseProgressMessage?(): ReactNode

  // Budget management
  maxResultSizeChars: number
  shouldDefer?: boolean      // Deferred loading via ToolSearch
  alwaysLoad?: boolean       // Never deferred
}
```

Key properties that drive orchestration:
- `isReadOnly()` -- determines write permission need
- `isConcurrencySafe()` -- determines if tool can run in parallel batch
- `maxResultSizeChars` -- triggers disk persistence for large results
- `shouldDefer` -- lazy schema loading via ToolSearch tool

### Tool Registration (Central Registry)

**Source**: `src/tools.ts`

The central registry in `tools.ts` imports tools statically or conditionally behind feature flags. `getAllBaseTools()` returns the complete tool list for the current environment. Tools are assembled into the final pool via `assembleToolPool()`:

1. Get built-in tools via `getTools()` (respects mode filtering, deny rules)
2. Filter MCP tools by deny rules
3. Deduplicate by name (built-in tools take precedence)
4. Sort each partition for prompt-cache stability (built-ins as contiguous prefix)

### Tool Execution Pipeline

**Source**: `toolOrchestration.ts`

```
Claude returns assistant message with tool_use block
      |
queryLoop calls runTools(toolUseBlocks)
      |
Partition: read-only --> parallel batch, write --> serial batch
      |
For each tool:
  1. validateInput()         --> Zod schema validation
  2. canUseTool()            --> multi-layer permission decision
  3. PreToolUse hooks        --> may block/modify
  4. tool.call(input, ctx)   --> execute, yield progress
  5. PostToolUse hooks       --> may transform result
  6. Result truncation       --> enforce maxResultSizeChars
      |
yield tool_result block back to queryLoop
```

**Smart concurrency**: Read-only tools (`FileRead`, `Glob`, `Grep`, `WebFetch`) execute as parallel batch. Mutating tools (`FileEdit`, `FileWrite`, `Bash`) execute serially.

### Context-Aware Tool Availability

The tool registry is filtered per execution context:

| Context | Allowed | Disallowed |
|---------|---------|------------|
| Async Agent | FileRead, Grep, Bash, FileEdit | Agent, TaskStop, ExitPlanMode |
| Coordinator | Agent, TaskStop, SendMessage | Most filesystem/shell tools |

### ToolSearch (Deferred Loading)

Tools can be marked `shouldDefer: true` to keep them out of the initial prompt. The `ToolSearchTool` lets the model query the deferred registry at runtime, fetching full schemas on demand. This keeps the per-turn schema payload small while preserving access to the long tail of MCP tools.

---

## MCP Integration

### Host Architecture

**Source**: `src/services/mcp/client.ts`, `src/services/mcp/types.ts`

Claude Code acts as an MCP **host** managing multiple MCP **clients** (one per configured server). The architecture layers:

```
Agent Query Loop (natural language)
      |
MCPTool / ReadMcpResourceTool / ListMcpResourcesTool (adapter layer)
      |
connectToServer + ensureConnectedClient (client dispatch)
MCPConnectionManager
      |
StdioClientTransport | SSE | HTTP | WS | InProc | ClaudeAI-proxy (transport)
      |
Local subprocess | Remote server | Claude.ai (external services)
```

### Transport Types

**Source**: `src/services/mcp/types.ts`

| Transport | Config Schema | Use Case |
|-----------|--------------|----------|
| `stdio` | `McpStdioServerConfigSchema` | Local subprocess (most common) |
| `sse` | `McpSSEServerConfigSchema` | Remote HTTP with Server-Sent Events |
| `http` | `McpHTTPServerConfigSchema` | REST-style via `StreamableHTTPClientTransport` |
| `ws` | `McpWebSocketServerConfigSchema` | Persistent bidirectional |
| `claudeai-proxy` | `McpClaudeAIProxyServerConfigSchema` | Anthropic-hosted servers |
| `InProcessTransport` | Internal only | Built-in capabilities through MCP interface |

### Tool Namespacing

**Source**: `src/services/mcp/mcpStringUtils.ts`

MCP tools are namespaced via `buildMcpToolName(serverName, toolName)` producing `mcp__<server>__<tool>`:

```
Server "github-mcp" exposing "search_repos", "read_file":
  mcp__github-mcp__search_repos
  mcp__github-mcp__read_file

Server "postgres" exposing "query":
  mcp__postgres__query
```

This namespace is the contract for permission rules, hook matchers, and UI display.

### Connection Lifecycle

1. `connectToServer(config)` -- resolves transport, creates `McpClient`, completes JSON-RPC initialize handshake
2. `ensureConnectedClient(serverName)` -- cache-aware lookup, lazy reconnect on first tool use
3. Connection states: `pending` -> `connected` | `disabled` | `failed`
4. Session recovery: `isMcpSessionNotFoundError` detects dead sessions (HTTP 404, JSON-RPC -32001), triggers transparent reconnect

### Output Size Management

`truncateMcpContentIfNeeded` checks results against configured thresholds. Oversized results are persisted to temp files via `persistBinaryContent`, and the agent receives `getLargeOutputInstructions` with a file reference instead. This prevents context bloat.

### Authentication

- Static credentials (API keys in config)
- Interactive OAuth via browser (`handleOAuth401Error` -> `performMCPOAuthFlow`)
- Per-server token management (independent auth state per server)
- `ElicitRequest` messages for runtime credential gathering

### Server Approval

**Source**: `src/services/mcpServerApproval.tsx`

Before a newly discovered server can connect, explicit user approval is required via `MCPServerApprovalDialog`. This prevents malicious workspace configs from silently launching subprocesses.

### MCP-to-Skill Bridge

**Source**: `src/skills/mcpSkillBuilders.ts`

MCP servers can also expose skills (prompt templates). The `registerMCPSkillBuilders` pattern provides a cycle-free bridge between the MCP client and the skill loader -- MCP prompts become first-class slash commands alongside local skills.

---

## Security Model

### Multi-Layer Permission Cascade

Tool execution goes through a multi-layer permission cascade:
1. Static rules from settings.json (allow/deny/ask lists)
2. Tool's own `checkPermissions()` method
3. Current permission mode (default/plan/bypass)
4. Hook-based classification or user prompt

### Plugin Policy Enforcement

- `isPluginBlockedByPolicy(pluginId)` -- checks against org allowlist/blocklist
- Enforcement at two points: **discovery time** (filtered from marketplace UI) and **load time** (rejected at startup)
- `getManagedPluginNames()` -- lists plugins that cannot be uninstalled (MDM-enforced)

### Hook Security

- `shouldAllowManagedHooksOnly()` -- restricts to MDM-managed hooks only
- `shouldDisableAllHooksIncludingManaged()` -- kills hooks entirely
- `ManagedSettingsSecurityDialog` -- blocks startup when MDM pushes hooks/env/permissions, requiring user trust
- Schema validation via Zod at load time

### Skill Security

- `allowed-tools` frontmatter field scopes what tools a skill can use
- MCP skills are marked untrusted -- inline shell commands (`!...`) in their markdown body are never executed
- Path validation prevents traversal attacks in bundled skill file extraction

### Plugin Telemetry Privacy

"Twin-column" pattern: official Anthropic plugins log real names; third-party plugins use `plugin_id_hash` to preserve anonymity.

---

## Key Patterns for AGH

### 1. Three-Layer State Reconciliation

The intent/materialization/activation separation is directly applicable to AGH's extension system. AGH should maintain:
- **Intent**: TOML config declaring desired extensions
- **Materialization**: On-disk cache of downloaded/compiled extensions
- **Activation**: Runtime-loaded extension instances in the daemon

This makes the system resilient to partial failures and supports offline operation.

### 2. Progressive Disclosure for Token Budget

Skills use a `description` + `when_to_use` metadata pattern where only metadata appears in the baseline prompt. Full skill content materializes only when activated. This is critical for AGH's multi-agent sessions where context budget is shared. AGH should adopt this pattern for its skills/instructions system.

### 3. Hook Output Protocol (Structured JSON)

The hook output protocol (`continue`, `stopReason`, `hookSpecificOutput`) is clean and extensible. AGH should adopt a similar structured protocol for its hook system, with event-specific output fields.

### 4. Uniform Tool Interface

The `Tool<Input, Output, Progress>` interface with `buildTool()` factory means adding a tool never requires changes to the query loop, permission system, or UI. AGH's ACP driver already has a tool concept, but making MCP tools indistinguishable from built-ins is a pattern worth adopting.

### 5. Plugin = Capability Bundle, Skill = Procedure

The clean separation between distributable capability bundles (plugins with tools, hooks, MCP servers) and procedural workflows (SKILL.md files) avoids conflation. AGH should maintain this distinction -- the `internal/skills/` package handles procedures, while a future plugin system handles capability bundles.

### 6. Namespace Convention for MCP Tools

The `mcp__<server>__<tool>` naming convention is simple and effective for disambiguation. AGH should adopt a similar namespacing for tools from different ACP agents.

### 7. Scoped Configuration Cascade

The four-scope cascade (managed > user > project > local) with MDM override is essential for enterprise deployment. AGH's TOML config already supports merge, but it should add explicit scope precedence.

### 8. Dynamic Skill Discovery

Walking the filesystem from touched files upward to cwd to find `.claude/skills/` directories is a clever pattern for monorepos. AGH could apply this to workspace-scoped memory and skills.

### 9. Concurrent Hook Execution with Blocking Semantics

Hooks fire in parallel but a single `continue: false` blocks the operation. This balances throughput (parallel) with safety (any hook can veto). The timeout hierarchy (10min for tool hooks, 1.5s for session-end) prevents hangs.

### 10. Deferred Tool Loading (ToolSearch)

For AGH sessions with many connected agents/MCP servers, deferred tool loading keeps the initial prompt lean. The model uses a search tool to fetch schemas on demand.

---

## Code References

### Core Type Definitions
- `src/Tool.ts` -- `Tool<Input, Output, Progress>` interface, `buildTool()` factory, `ToolUseContext`, `ToolPermissionContext`
- `src/types/hooks.ts` -- `HookJSONOutput`, `HookCallback`, `PromptRequest`/`PromptResponse`
- `src/entrypoints/agentSdkTypes.ts` -- `HookEvent` union, all hook input types
- `src/types/plugin.ts` -- `LoadedPlugin`, `PluginManifest`, `BuiltinPluginDefinition`
- `src/types/command.ts` -- `Command`, `PromptCommand` (skill representation)

### Tool Registration
- `src/tools.ts` -- central registry (`getAllBaseTools()`, `getTools()`, `assembleToolPool()`)
- `src/tools/MCPTool/MCPTool.ts` -- MCP tool wrapper (overridden per-instance in `client.ts`)
- `src/tools/SkillTool/SkillTool.ts` -- skill invocation tool
- `src/tools/ToolSearchTool/ToolSearchTool.ts` -- deferred tool search

### Plugin System
- `src/plugins/builtinPlugins.ts` -- built-in plugin registry and `registerBuiltinPlugin()`
- `src/plugins/bundled/index.ts` -- `initBuiltinPlugins()` scaffold
- `src/services/plugins/pluginOperations.ts` -- install/uninstall/enable/disable/update operations
- `src/services/plugins/PluginInstallationManager.ts` -- background installation with marketplace reconciliation
- `src/utils/plugins/pluginLoader.ts` -- `loadAllPlugins()`, `cachePlugin()`, version management
- `src/utils/plugins/pluginPolicy.ts` -- `isPluginBlockedByPolicy()`
- `src/utils/plugins/pluginStartupCheck.ts` -- startup validation
- `src/utils/plugins/refresh.ts` -- `refreshActivePlugins()` runtime reload

### Skill System
- `src/skills/bundledSkills.ts` -- `registerBundledSkill()`, `BundledSkillDefinition` type
- `src/skills/loadSkillsDir.ts` -- skill discovery, loading, deduplication, dynamic/conditional activation
- `src/skills/mcpSkillBuilders.ts` -- cycle-free bridge for MCP-to-skill integration
- `src/skills/bundled/` -- built-in skills (skillify, verifyContent, updateConfig, keybindings, claudeApi, etc.)

### Hook System
- `src/utils/hooks.ts` -- main orchestrator, hook matching, execution, timeouts
- `src/utils/hooks/hookEvents.ts` -- hook event broadcasting system
- `src/utils/hooks/hooksConfigManager.ts` -- hook configuration management
- `src/utils/hooks/hooksConfigSnapshot.ts` -- `captureHooksConfigSnapshot()`, MDM enforcement
- `src/utils/hooks/sessionHooks.ts` -- session-scoped hook registration
- `src/utils/hooks/execPromptHook.ts` -- LLM-based hook execution
- `src/utils/hooks/execAgentHook.ts` -- sub-agent hook execution
- `src/utils/hooks/execHttpHook.ts` -- HTTP webhook execution
- `src/utils/hooks/registerFrontmatterHooks.ts` -- hooks from agent/skill frontmatter
- `src/utils/hooks/registerSkillHooks.ts` -- hooks from skill definitions
- `src/utils/hooks/skillImprovement.ts` -- background skill co-evolution

### MCP Integration
- `src/services/mcp/client.ts` -- `connectToServer()`, `ensureConnectedClient()`, `callMCPTool()`, auth handling
- `src/services/mcp/types.ts` -- transport schemas, `MCPServerConnection`, `ConfigScope`
- `src/services/mcp/MCPConnectionManager.tsx` -- React context for MCP connection lifecycle
- `src/services/mcp/mcpStringUtils.ts` -- `buildMcpToolName()`, `mcpInfoFromString()`, namespace utilities
- `src/services/mcp/normalization.ts` -- name normalization for MCP identifiers
- `src/services/mcp/config.ts` -- `getAllMcpConfigs()`, `isMcpServerDisabled()`
- `src/services/mcp/auth.ts` -- `ClaudeAuthProvider`, OAuth flow
- `src/services/mcp/InProcessTransport.ts` -- in-process MCP transport for built-in capabilities
- `src/services/mcpServerApproval.tsx` -- server approval security dialog

### Settings & Configuration
- `src/utils/settings/settings.ts` -- settings cascade, per-source access
- `src/utils/settings/types.ts` -- `SettingsSchema`, `HooksSchema`, `HooksSettings`
- `src/utils/settings/managedPath.ts` -- MDM settings path resolution
