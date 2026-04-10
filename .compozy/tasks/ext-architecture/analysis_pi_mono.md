# Pi-Mono Extension Architecture Analysis

## Overview

Pi-Mono is a TypeScript monorepo (`github.com/badlogic/pi-mono`) created by Mario Zechner for building AI coding agents. It comprises seven packages organized in three tiers: a foundation LLM API (`pi-ai`), an infrastructure agent runtime (`pi-agent-core`, `pi-tui`), and application-tier products (`pi-coding-agent`, `pi-web-ui`, `pi-mom`, `pi-pods`). All packages are published under `@mariozechner` npm scope with lockstep versioning.

The system's extension architecture is built on a philosophy of **aggressive extensibility**: a minimal core (4 tools, <1000-token system prompt) with deep hooks at every phase of the agent lifecycle. Extensions, skills, prompt templates, and themes form four customization axes, all distributable as "Pi Packages" via npm or git.

**Key architectural principle**: Pi does NOT use MCP, sub-agents, permission popups, plan mode, or built-in todos. These are all delegated to the extension system, proving the extension API's completeness.

---

## Extension & Customization System

### Extension Entry Point

Every extension is a TypeScript module that exports a single default function receiving an `ExtensionAPI` object:

```typescript
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  // all registration happens here
}
```

There is no plugin DSL, no YAML manifest for capabilities, and no restricted sandbox language. Extensions import the same packages the agent uses.

**Source**: `packages/coding-agent/src/core/extensions/types.ts` -- `ExtensionFactory` type (line 1273):
```typescript
export type ExtensionFactory = (pi: ExtensionAPI) => void | Promise<void>;
```

### ExtensionAPI Surface

The `ExtensionAPI` interface (types.ts lines 986-1211) is deliberately flat and exposes:

| Category | Methods |
|----------|---------|
| Event subscription | `pi.on(eventName, handler)` -- 25+ strongly-typed event types |
| Tool registration | `pi.registerTool(toolDef)` |
| Command registration | `pi.registerCommand(name, opts)` |
| Shortcut registration | `pi.registerShortcut(key, opts)` |
| Flag registration | `pi.registerFlag(name, opts)` / `pi.getFlag(name)` |
| Message rendering | `pi.registerMessageRenderer(customType, renderer)` |
| Provider registration | `pi.registerProvider(name, config)` / `pi.unregisterProvider(name)` |
| Actions | `sendMessage`, `sendUserMessage`, `appendEntry`, `exec` |
| Tool management | `getActiveTools`, `getAllTools`, `setActiveTools` |
| Model control | `setModel`, `getThinkingLevel`, `setThinkingLevel` |
| Session metadata | `setSessionName`, `getSessionName`, `setLabel` |
| Inter-extension comms | `pi.events` (shared EventBus) |

### Extension Loading & Discovery

Extensions auto-discover from four filesystem locations:

| Location | Scope |
|----------|-------|
| `.pi/extensions/*.ts` | Project-local |
| `.pi/extensions/*/index.ts` | Project-local (subdirectory) |
| `~/.pi/agent/extensions/*.ts` | Global (all projects) |
| `~/.pi/agent/extensions/*/index.ts` | Global (subdirectory) |

Additional paths via `settings.json` under `extensions` or `packages` arrays. The `-e ./path.ts` CLI flag loads a single extension for testing.

**Source**: `packages/coding-agent/src/core/extensions/loader.ts`

The loader (`discoverAndLoadExtensions()`, loader.ts line 511) resolves extension files, creates a `jiti` transpiler with virtual modules for bundled packages, and calls each extension's default export. Virtual modules ensure extensions can import core packages without installing them:

```typescript
const VIRTUAL_MODULES: Record<string, unknown> = {
  "@sinclair/typebox": _bundledTypebox,
  "@mariozechner/pi-agent-core": _bundledPiAgentCore,
  "@mariozechner/pi-tui": _bundledPiTui,
  "@mariozechner/pi-ai": _bundledPiAi,
  "@mariozechner/pi-ai/oauth": _bundledPiAiOauth,
  "@mariozechner/pi-coding-agent": _bundledPiCodingAgent,
};
```

### Extension Runtime Lifecycle

The runtime lifecycle proceeds in stages (managed by `ExtensionRunner` in runner.ts):

1. **Load phase**: Extensions loaded sequentially. `createExtensionRuntime()` creates a runtime with throwing stubs for action methods. Registration calls (tools, commands, etc.) work immediately. Action calls (sendMessage, etc.) throw.

2. **`bindCore(actions, contextActions)`**: Called by `AgentSession` after initialization. Flushes pending provider registrations and wires real action implementations into the runtime. All extension API objects reference the shared runtime, so wiring is automatic.

3. **`bindCommandContext(actions)`**: Wires navigation actions (`newSession`, `fork`, `navigateTree`, `switchSession`, `reload`). Only called when UI mode is present.

4. **Event dispatch**: Multiple specialized emit methods: `emit(event)`, `emitToolCall(event)`, `emitToolResult(event)`, `emitBeforeAgentStart(event)`, `emitContext(messages)`, `emitInput(text, images, source)`, `emitResourcesDiscover(cwd, reason)`.

### Event System

25+ strongly-typed events organized into categories:

- **Session lifecycle**: `session_start`, `session_before_switch`, `session_before_fork`, `session_before_compact`, `session_compact`, `session_shutdown`, `session_before_tree`, `session_tree`
- **Agent loop**: `before_agent_start`, `agent_start`, `agent_end`, `turn_start`, `turn_end`, `context`, `before_provider_request`
- **Messages**: `message_start`, `message_update`, `message_end`
- **Tools**: `tool_execution_start`, `tool_execution_update`, `tool_execution_end`, `tool_call` (can block!), `tool_result` (can modify, chains like middleware)
- **Input**: `input` (transform/handle user input before agent)
- **Resources**: `resources_discover` (contribute skill/prompt/theme paths)
- **Model**: `model_select`
- **Bash**: `user_bash`

The `tool_call` event is notable: returning `{ block: true, reason: "..." }` prevents execution. The `event.input` is mutable -- mutations propagate to subsequent handlers. No re-validation after mutation. This enables permission gates, path protection, and input rewriting.

The `tool_result` event chains like middleware: each handler returns patches for `content`, `details`, or `isError`, merged sequentially.

### Custom Tools

Tools are registered via `pi.registerTool()` with TypeBox schema validation:

```typescript
interface ToolDefinition<TParams, TDetails, TState> {
  name: string;
  label: string;
  description: string;
  parameters: TParams;           // TypeBox schema
  promptSnippet?: string;        // one-liner for system prompt
  promptGuidelines?: string[];   // bullets for Guidelines section
  prepareArguments?: fn;         // pre-validation shim
  execute: fn;                   // core logic
  renderCall?: fn;               // custom TUI for arguments
  renderResult?: fn;             // custom TUI for results
}
```

**Source**: types.ts lines 369-405

The `execute` function receives: `toolCallId`, validated params, `AbortSignal`, `onUpdate` callback (streaming progress), and `ExtensionContext`.

Extensions can **override built-in tools** by registering a tool with the same name. Rendering inheritance: if override omits `renderCall`/`renderResult`, built-in renderers are used.

Built-in tools also expose pluggable operations interfaces (`ReadOperations`, `WriteOperations`, `BashOperations`) for delegating execution to remote systems.

### Custom Commands

Commands are slash-prefixed (`/mycommand args`), user-initiated (not LLM-invoked). Handlers receive `ExtensionCommandContext` with session control methods: `waitForIdle()`, `newSession()`, `fork()`, `navigateTree()`, `switchSession()`, `reload()`, `shutdown()`.

Commands bypass skill expansion. If multiple extensions register same name, numeric suffixes assigned (`/review:1`, `/review:2`). Built-in commands always win.

### State Persistence

Two mechanisms:
1. **Custom entries**: `pi.appendEntry(customType, data)` -- persisted in session file, survives restarts, NOT sent to LLM
2. **Tool result details**: The `details` field in tool results is stored and replayed during restore/navigation

---

## Provider Plugin Pattern

### API Registry Architecture

**Source**: `packages/ai/src/api-registry.ts`

The API registry is a `Map<string, RegisteredApiProvider>` keyed by API type string. Each entry holds two streaming functions: `stream` (provider-specific options) and `streamSimple` (unified options). Runtime type checking enforces model's `api` field matches.

```typescript
export function registerApiProvider<TApi extends Api, TOptions extends StreamOptions>(
  provider: ApiProvider<TApi, TOptions>,
  sourceId?: string,
): void {
  apiProviderRegistry.set(provider.api, { provider: { ... }, sourceId });
}
```

Operations: `getApiProvider(api)`, `getApiProviders()`, `unregisterApiProviders(sourceId)`, `clearApiProviders()`.

### Lazy Provider Registration

**Source**: `packages/ai/src/providers/register-builtins.ts`

Each provider uses lazy loading via nullish assignment to avoid pulling in heavy SDKs at startup:

```typescript
function loadAnthropicProviderModule() {
  anthropicProviderModulePromise ||= import("./anthropic.js").then((module) => ({
    stream: module.streamAnthropic,
    streamSimple: module.streamSimpleAnthropic,
  }));
  return anthropicProviderModulePromise;
}
```

If dynamic import fails, the error is encoded in the stream (stopReason: "error") rather than thrown -- preserving the stream protocol contract.

`registerBuiltInApiProviders()` runs as a side effect at module load, registering all 10 built-in API types:
- `anthropic-messages`, `openai-completions`, `openai-responses`, `azure-openai-responses`, `openai-codex-responses`, `mistral-conversations`, `google-generative-ai`, `google-gemini-cli`, `google-vertex`, `bedrock-converse-stream`

### Extension-based Provider Registration

Extensions register providers at runtime via `pi.registerProvider(name, config)`:

```typescript
interface ProviderConfig {
  baseUrl?: string;
  apiKey?: string;
  api?: Api;
  streamSimple?: (model, context, options?) => AssistantMessageEventStream;
  headers?: Record<string, string>;
  authHeader?: boolean;
  models?: ProviderModelConfig[];
  oauth?: { name, login, refreshToken, getApiKey, modifyModels? };
}
```

**Source**: types.ts lines 1218-1270

Calls during load phase are queued in `pendingProviderRegistrations` and flushed by `bindCore()`. Post-initialization calls take effect immediately.

The `Api` type is `KnownApi | (string & {})` -- any string is valid at runtime, allowing custom API types without modifying core type definitions.

### Custom models.json

Users add custom providers/models via `~/.pi/agent/models.json`. Three modes:
- **Proxy mode**: just `baseUrl`, redirects existing models through proxy
- **Upsert mode**: `models` array alongside built-in provider, replaces matching IDs
- **Model overrides**: `modelOverrides` field for tweaking specific built-ins

Dynamic value resolution: `apiKey`/`headers` support shell commands (`!` prefix), env vars, or literals.

### OAuth Provider System

Five built-in OAuth providers implementing `OAuthProviderInterface`:
- Anthropic (PKCE + callback server, port 53692)
- OpenAI Codex (PKCE + callback server, port 1455)
- GitHub Copilot (device code flow)
- Gemini CLI (Google Cloud OAuth)
- Antigravity (specialized Google Cloud)

Extensions register custom OAuth via `pi.registerProvider()` with an `oauth` config object.

---

## Skills & Prompt Templates

### Skills (Agent Skills Standard)

Skills implement the [agentskills.io](https://agentskills.io/specification) standard. A skill is a `SKILL.md` file with YAML frontmatter and Markdown instructions.

**Source**: `packages/coding-agent/src/core/skills.ts`

Frontmatter schema:
```yaml
name: my-skill         # lowercase a-z, 0-9, hyphens. Max 64 chars.
description: ...       # Max 1024 chars. Determines auto-invocation.
disable-model-invocation: false  # Optional
```

Discovery locations (in precedence order):
1. `~/.pi/agent/skills/` -- global user
2. `~/.agents/skills/` -- cross-agent compatibility
3. `.pi/skills/` -- project-specific
4. `.agents/skills/` -- project agents (scanned up through parents)

Project resources override global ones on name collision.

**Progressive disclosure execution model**:
1. Only skill `name` and `description` injected into system prompt
2. Agent uses `read` tool to fetch full `SKILL.md` on demand
3. Users can force via `/skill:name [args]`

System prompt format (skills.ts `formatSkillsForPrompt`, line 339):
```xml
<available_skills>
  <skill>
    <name>my-skill</name>
    <description>...</description>
    <location>/path/to/SKILL.md</location>
  </skill>
</available_skills>
```

### Prompt Templates

Markdown files with bash-style variable substitution:
- `$1`, `$2`, ... -- positional arguments
- `$@` / `$ARGUMENTS` -- all arguments
- `${@:N}` -- arguments from index N
- `${@:N:L}` -- L arguments from index N

Discovery: `~/.pi/agent/prompts/` (global), `.pi/prompts/` (project). Project overrides global.

### Skills vs Extensions vs MCP (Design Philosophy)

| Capability | Skills | Extensions | MCP (rejected) |
|-----------|--------|------------|----------------|
| Teach agent procedures | Yes | No | N/A |
| Register custom tools | No | Yes | Yes |
| Requires TypeScript | No | Yes | Yes |
| Context cost | On-demand | Per-tool definition | Always loaded |
| Auto-invoked by agent | Yes | N/A | N/A |

Pi explicitly rejects MCP due to context bloat (13,700-18,000 tokens per server), composability limits (CLI pipes are more capable), and extensibility friction.

---

## Monorepo Package Architecture

### Three-Tier Layer Diagram

```
APPLICATIONS:  pi-coding-agent  |  pi-web-ui  |  pi-mom  |  pi-pods
                    |                  |             |           |
INFRASTRUCTURE: pi-agent-core  |  pi-tui (standalone)
                    |
FOUNDATION:     pi-ai (standalone)
```

### Package Inventory

| Package | npm name | Purpose |
|---------|----------|---------|
| `packages/ai` | `@mariozechner/pi-ai` | Unified LLM API: 15+ providers, 10 API protocols, streaming, model catalog, cost tracking, OAuth |
| `packages/agent` | `@mariozechner/pi-agent-core` | Minimal agent runtime: turn-based loop, tool execution (sequential/parallel), lifecycle hooks |
| `packages/coding-agent` | `@mariozechner/pi-coding-agent` | CLI (`pi`): 3 run modes (interactive/print/RPC), extension system, session management, compaction |
| `packages/tui` | `@mariozechner/pi-tui` | Terminal UI: differential rendering, components, editor, keyboard handling |
| `packages/web-ui` | `@mariozechner/pi-web-ui` | Web components: chat panel, messages, tool renderers |
| `packages/mom` | `@mariozechner/pi-mom` | Slack bot: delegates to coding agent, per-channel stores, sandbox execution |
| `packages/pods` | `@mariozechner/pi` | GPU pod management: vLLM deployment, SSH-based operations |

### Dependency Rules

- Foundation packages (`pi-ai`, `pi-tui`) have zero internal dependencies
- `pi-agent-core` depends only on `pi-ai`
- `pi-coding-agent` pulls all three lower layers
- Leaf packages depend on subsets -- consumers can use each layer independently
- All packages lockstep versioned (same version number)

### Build System

- npm workspaces (not Yarn/pnpm/Turborepo)
- Sequential build respecting dependency graph: `tui -> ai -> agent -> coding-agent -> mom -> web-ui -> pods`
- TypeScript compiled with `tsgo` (Go port of tsc); web-ui uses standard `tsc`
- Biome for linting/formatting
- Model catalog auto-generated from provider APIs at build time
- Vitest for testing (pi-ai, pi-agent-core, pi-coding-agent); Node test runner for pi-tui

### Module Export Patterns

`pi-ai` uses subpath exports for selective imports (e.g., `@mariozechner/pi-ai/anthropic`). This enables tree-shaking -- unused provider SDKs are eliminated from bundles.

`pi-coding-agent` exposes core logic and a hooks API for extension developers.

---

## Security Model

**There is no sandbox.** Extensions run with full trust in the same Node.js process. No capability-based sandbox, no permission manifest, no code review gate. This is a deliberate choice: pi targets developers already running arbitrary code.

Practical mitigations:
- **Scope isolation**: project-local extensions in `.pi/extensions/` visible only to that project
- **Permission gates via tool_call**: extensions CAN add confirmation prompts (but this is advisory, not a security boundary)
- **Package provenance**: npm audit, git commit history
- **Conflict resolution rules**:
  - Reserved keybindings cannot be overridden by extensions
  - Built-in commands always win over extension commands
  - Extension-vs-extension: first-registered-wins, numeric suffixes for duplicates
  - Provider overrides: `registerProvider()` may override built-ins; `unregisterProvider()` restores them

---

## Key Patterns for AGH

### 1. Single Entry Point Factory Pattern
Every extension exports one default function receiving a typed API object. No manifest files, no class inheritance, no interfaces to implement. This is the lowest-friction extension model possible.

**AGH implication**: Go equivalent could be a `func(api *ExtensionAPI)` entry point loaded via plugin or embedded scripting.

### 2. Typed Event Bus with Lifecycle Hooks
25+ strongly-typed events covering every phase: session start/stop, agent loop, tool execution, message streaming, context modification, input transformation. Events can block operations (`tool_call`), modify data in flight (`tool_result`, `context`, `input`), or inject content (`before_agent_start`).

**AGH implication**: The Notifier pattern in AGH already provides fan-out. Adding typed hook points at session/tool/agent boundaries would enable similar extensibility. Critical hooks to replicate:
- Before/after tool execution (with block/modify capability)
- Context modification before LLM call
- Input transformation
- Session lifecycle (start, compact, shutdown)

### 3. Deferred Binding / Two-Phase Initialization
Extensions load and register tools/commands/events synchronously, but action methods (sendMessage, etc.) throw until `bindCore()` is called. Provider registrations are queued during load, flushed at bind time.

**AGH implication**: For Go, this maps to a two-phase init: (1) collect registrations during plugin load, (2) wire in real implementations once daemon services are ready. Use interface stubs or channel-based deferred execution.

### 4. Virtual Module Injection
Core packages are bundled into the binary and provided as virtual modules to extensions, eliminating dependency installation for simple extensions.

**AGH implication**: If using embedded scripting (Lua, Starlark, etc.), provide standard library modules pre-loaded. If using Go plugins, provide a well-defined SDK package.

### 5. Progressive Disclosure for Skills
Only skill name+description are in the system prompt (few tokens). Full content loaded on-demand via the agent's own `read` tool. This preserves context window while enabling unlimited skills.

**AGH implication**: AGH's skills system should follow this pattern. Inject a compact skill index into the system prompt; let the agent load full skill content when needed.

### 6. Pi Package Distribution Model
Extensions, skills, prompts, and themes bundled in packages with a `package.json` `"pi"` key:
```json
{ "pi": { "extensions": ["./extensions"], "skills": ["./skills"], "prompts": ["./prompts"], "themes": ["./themes"] } }
```

Sources: npm (`npm:@scope/pkg`), git (`git:github.com/user/repo`), local paths. Pattern filtering with glob, `!` exclude, `+` force-include, `-` force-exclude.

**AGH implication**: AGH could use a similar manifest-based package format. TOML instead of JSON for consistency. Distribution via git (primary) and Go module paths.

### 7. Tool Override with Renderer Inheritance
Extensions can replace built-in tools by name. If the override omits custom renderers, built-in renderers are used. Built-in tools expose pluggable operations interfaces for delegation.

**AGH implication**: Allow extensions to wrap or replace ACP tool implementations while preserving observability/rendering.

### 8. No MCP -- CLI Tools + Skills Instead
Pi's alternative to MCP: teach the agent about CLI tools via skill files. Token cost is on-demand (only when relevant) vs MCP's always-loaded tool definitions. Shell composition (pipes, redirects) provides superior composability.

**AGH implication**: AGH already has skills. Consider making skills the primary extensibility mechanism for agent capabilities, with extensions reserved for runtime behavior modification.

### 9. Conflict Resolution is Deterministic
- Shortcuts: reserved cannot be overridden, non-reserved generate warning
- Commands: built-in wins, extension duplicates get numeric suffixes
- Tools: first-registered wins, warning for built-in conflicts
- Providers: explicit override/restore semantics

**AGH implication**: Define clear precedence rules for every registrable resource type.

### 10. Settings Cascade: Global + Project
Global settings (`~/.pi/agent/`), project settings (`.pi/`). Project overrides global. Team sharing via committed `.pi/settings.json`.

**AGH implication**: AGH already has workspace-scoped config. Ensure extension/skill discovery follows the same cascade with project taking precedence.

---

## Code References

### Extension System
- **Extension types & API**: `packages/coding-agent/src/core/extensions/types.ts` (1450 lines, comprehensive type definitions)
- **Extension loader**: `packages/coding-agent/src/core/extensions/loader.ts` (discovery, jiti transpiler, virtual modules)
- **Extension runner**: `packages/coding-agent/src/core/extensions/runner.ts` (lifecycle, event dispatch, conflict resolution)
- **Extension index**: `packages/coding-agent/src/core/extensions/index.ts` (public API exports)

### Provider System
- **API registry**: `packages/ai/src/api-registry.ts` (provider registration, type-safe dispatch)
- **Lazy registration**: `packages/ai/src/providers/register-builtins.ts` (lazy loading pattern, 10 built-in providers)
- **Provider modules**: `packages/ai/src/providers/*.ts` (one file per provider: anthropic.ts, google.ts, openai-completions.ts, etc.)

### Skills System
- **Skill loader**: `packages/coding-agent/src/core/skills.ts` (discovery, validation, prompt formatting)
- **Package manager**: `packages/coding-agent/src/core/package-manager.ts` (npm/git/local source resolution)
- **Resource loader**: `packages/coding-agent/src/core/resource-loader.ts` (aggregation pipeline)

### Agent Core
- **Agent types**: `packages/agent/src/types.ts` (StreamFn, tool execution modes, hook interfaces)
- **Agent loop**: `packages/agent/src/agent-loop.ts` (core turn-by-turn cycle)

### Examples (60+ extension examples)
- **Minimal tool**: `packages/coding-agent/examples/extensions/hello.ts`
- **Custom provider with OAuth**: `packages/coding-agent/examples/extensions/custom-provider-anthropic/` (full streaming implementation)
- **Permission gate**: `packages/coding-agent/examples/extensions/confirm-destructive.ts`
- **Git checkpoint**: `packages/coding-agent/examples/extensions/git-checkpoint.ts`
- **Custom compaction**: `packages/coding-agent/examples/extensions/custom-compaction.ts`
- **Dynamic resources**: `packages/coding-agent/examples/extensions/dynamic-resources/`

### Configuration
- **Root package.json**: `package.json` (workspace config, build scripts)
- **Pi package manifest**: `"pi"` key in `package.json` (extensions, skills, prompts, themes arrays)
- **Settings cascade**: `~/.pi/agent/settings.json` (global) + `.pi/settings.json` (project)

### Wiki Documentation
- `/Users/pedronauck/dev/knowledge/pi-mono/wiki/concepts/Extension and Customization System.md`
- `/Users/pedronauck/dev/knowledge/pi-mono/wiki/concepts/Design Philosophy and Extensibility.md`
- `/Users/pedronauck/dev/knowledge/pi-mono/wiki/concepts/Pi Skills and Prompt Templates.md`
- `/Users/pedronauck/dev/knowledge/pi-mono/wiki/concepts/Provider and Model System.md`
- `/Users/pedronauck/dev/knowledge/pi-mono/wiki/concepts/Pi Monorepo Architecture.md`
