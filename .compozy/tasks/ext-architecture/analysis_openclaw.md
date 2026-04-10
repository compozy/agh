# OpenClaw Extension Architecture Analysis

## Overview

OpenClaw's extensibility is built on a **Plugin SDK boundary** pattern where all extensions -- bundled and third-party alike -- interact with core through a narrow, typed surface at `src/plugin-sdk/`. The system supports 70+ bundled extensions across four plugin types (channel, provider, tool, skill) and distributes third-party extensions through ClawHub (clawhub.ai). Native apps (macOS, iOS, Android) are distinct from plugins; they connect as WebSocket node clients rather than loading into the Gateway process.

The key architectural insight: **bundled extensions in `/extensions/` follow the exact same boundary rules as third-party plugins installed from ClawHub**. Core does not special-case bundled vs. external. This uniformity is enforced by convention (`extensions/AGENTS.md`), package boundaries (tsconfig isolation), and the Plugin SDK barrel structure.

OpenClaw is a TypeScript/Node.js system. Extensions are npm packages with a `package.json` containing an `openclaw` block and a companion `openclaw.plugin.json` manifest. The Gateway is the host process; all plugins run in-process.

## Extension Loading & Discovery

### Five-Phase Loading Sequence

Per `src/gateway/server-plugin-bootstrap.ts`, extensions load in a strict order:

```
1. Manifest discovery phase     -- scan /extensions/ + node_modules/@openclaw/*
                                   NO code execution
2. Manifest validation          -- parse openclaw.plugin.json, check JSON Schema,
                                   verify requirements (config keys, binaries)
3. Dependency ordering          -- providers before channels before skills
4. Code load phase              -- dynamic import of plugin entry points
5. Registration                 -- each plugin registers with core via Plugin API
6. Ready signal                 -- all plugins loaded -> Gateway binds WS server
```

The split between **discovery** (manifest inspection) and **load** (code execution) is deliberate. A plugin can declare itself, surface its requirements, and appear in `openclaw plugins status` without ever running code. Code runs only if the plugin is enabled.

### Discovery Locations

- **Bundled**: `/extensions/<name>/` directories in the OpenClaw repo
- **Third-party**: `node_modules/@openclaw/*` (installed via npm/ClawHub)

### Filtering

Plugins can be disabled via config:

```json5
{
  plugins: {
    allow: ["browser", "discord", "anthropic"],  // hard allowlist
    entries: {
      "<id>": { enabled: false }                 // per-plugin toggle
    }
  }
}
```

If `plugins.allow` is set, only listed plugins load. Everything else is skipped at discovery.

### Plugin Status States

| State      | Meaning                                    |
|------------|--------------------------------------------|
| `enabled`  | Loaded and registered                      |
| `disabled` | Present but config disables it             |
| `missing`  | Referenced in config, not installed        |

Missing plugins produce warnings at startup but do not prevent Gateway boot.

**Source files**: `extensions/AGENTS.md`, `extensions/CLAUDE.md`

## Extension Manifest Format

Every extension ships two declaration files:

### 1. `openclaw.plugin.json` -- Static Manifest

Declares metadata, capabilities, and configuration schema **without executing code**. Examples from source:

**Channel plugin (Discord)**:
```json
{
  "id": "discord",
  "channels": ["discord"],
  "channelEnvVars": { "discord": ["DISCORD_BOT_TOKEN"] },
  "configSchema": { "type": "object", "additionalProperties": false, "properties": {} }
}
```

**Provider plugin (Anthropic)**:
```json
{
  "id": "anthropic",
  "enabledByDefault": true,
  "providers": ["anthropic"],
  "modelSupport": { "modelPrefixes": ["claude-"] },
  "cliBackends": ["claude-cli"],
  "providerAuthEnvVars": { "anthropic": ["ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_API_KEY"] },
  "providerAuthChoices": [ ... ],
  "contracts": { "mediaUnderstandingProviders": ["anthropic"] },
  "configSchema": { ... }
}
```

**Tool plugin (Browser)**:
```json
{
  "id": "browser",
  "enabledByDefault": true,
  "configSchema": { "type": "object", "additionalProperties": false, "properties": {} }
}
```

**Memory plugin (memory-core)**:
```json
{
  "id": "memory-core",
  "kind": "memory",
  "uiHints": { ... },
  "configSchema": { ... complex schema with dreaming phases ... }
}
```

Key manifest fields:
- `id` -- unique identifier
- `enabledByDefault` -- whether the plugin loads without explicit config
- `channels` -- channel IDs this plugin provides
- `providers` -- provider IDs this plugin provides
- `modelSupport.modelPrefixes` -- which model ID prefixes this provider handles
- `providerAuthEnvVars` -- environment variables for auth
- `providerAuthChoices` -- onboarding auth flow options
- `contracts` -- capability contracts (e.g., media understanding, speech, image generation)
- `configSchema` -- JSON Schema for plugin-specific configuration
- `kind` -- plugin kind for specialized plugins (e.g., `"memory"`)
- `uiHints` -- UI rendering hints for config fields
- `channelEnvVars` -- required env vars per channel

### 2. `package.json` -- npm Metadata + OpenClaw Block

The `openclaw` block in `package.json` declares build, distribution, and runtime metadata:

```json
{
  "name": "@openclaw/discord",
  "version": "2026.4.10",
  "openclaw": {
    "extensions": ["./index.ts"],
    "setupEntry": "./setup-entry.ts",
    "channel": {
      "id": "discord",
      "label": "Discord",
      "selectionLabel": "Discord (Bot API)",
      "docsPath": "/channels/discord",
      "blurb": "very well supported right now.",
      "markdownCapable": true,
      "configuredState": {
        "specifier": "./configured-state",
        "exportName": "hasDiscordConfiguredState"
      }
    },
    "install": {
      "npmSpec": "@openclaw/discord",
      "defaultChoice": "npm",
      "minHostVersion": ">=2026.4.10"
    },
    "compat": { "pluginApi": ">=2026.4.10" },
    "release": {
      "publishToClawHub": true,
      "publishToNpm": true
    }
  }
}
```

Key `openclaw` block fields:
- `extensions` -- entry point file paths (array; supports multiple)
- `setupEntry` -- separate entry for onboarding/setup flows
- `channel` -- channel metadata (label, docs, capabilities)
- `install` -- npm spec and host version requirements
- `compat.pluginApi` -- minimum Plugin SDK version
- `release` -- distribution targets (ClawHub, npm)
- `bundle.stageRuntimeDependencies` -- build-time bundling hints

**Source files**: `extensions/discord/openclaw.plugin.json`, `extensions/discord/package.json`, `extensions/anthropic/openclaw.plugin.json`, `extensions/browser/openclaw.plugin.json`, `extensions/memory-core/openclaw.plugin.json`

## ClawHub Marketplace

**ClawHub** (https://clawhub.ai) is the public registry for skills and plugins:

### Distribution Model

- **Package format**: git repos or npm packages
- **Versioning**: semver tags on git refs
- **Namespacing**: `openclaw/weather`, `@username/custom-tool`
- **Metadata**: registry caches manifests, descriptions, homepage URLs

### CLI Operations

```bash
openclaw skills list                   # browse registry
openclaw skills info github            # inspect before install
openclaw skills install github         # install to workspace
openclaw skills install github@1.2.3   # pin version
openclaw skills update github          # upgrade
openclaw skills update --all           # upgrade all
openclaw skills uninstall github       # remove
openclaw skills search weather         # search index
```

### Distribution Config in package.json

```json
{
  "openclaw": {
    "release": {
      "publishToClawHub": true,
      "publishToNpm": true
    }
  }
}
```

ClawHub is optional -- users can point `skills.load.extraDirs` at any local directory and skip the registry entirely. This matters for air-gapped or sensitive deployments.

**Source**: Wiki article "Skills, ClawHub and Plugins"

## Hook System

Extensions hook into core through the **Plugin API** object (`OpenClawPluginApi`) passed to the `register()` function. The API provides typed registration methods:

### Entry Point Pattern

Every extension exports a default entry created by one of:
- `definePluginEntry()` -- general plugins (tools, providers, memory)
- `defineBundledChannelEntry()` -- channel plugins
- `defineBundledChannelSetupEntry()` -- channel setup/onboarding

**General plugin entry** (from `openclaw/plugin-sdk/plugin-entry`):
```typescript
export default definePluginEntry({
  id: "browser",
  name: "Browser",
  description: "Default browser tool plugin",
  reload: browserPluginReload,                          // hot-reload config
  nodeHostCommands: browserPluginNodeHostCommands,      // device commands
  securityAuditCollectors: [...collectors],              // security audit hooks
  register: registerBrowserPlugin,                       // main registration
});
```

**Channel plugin entry** (from `openclaw/plugin-sdk/channel-entry-contract`):
```typescript
export default defineBundledChannelEntry({
  id: "discord",
  name: "Discord",
  description: "Discord channel plugin",
  importMetaUrl: import.meta.url,
  plugin: { specifier: "./channel-plugin-api.js", exportName: "discordPlugin" },
  runtime: { specifier: "./runtime-api.js", exportName: "setDiscordRuntime" },
  registerFull(api) {
    api.on("subagent_spawning", async (event) => { ... });
    api.on("subagent_ended", async (event) => { ... });
    api.on("subagent_delivery_target", async (event) => { ... });
  },
});
```

### Plugin API Registration Methods

Observed from source code, `OpenClawPluginApi` provides:

| Method | Purpose | Example |
|--------|---------|---------|
| `api.registerTool(factory, opts?)` | Register agent-callable tool | Browser, memory_search |
| `api.registerProvider(provider)` | Register LLM/inference provider | OpenAI, Anthropic |
| `api.registerCliBackend(backend)` | Register CLI backend for agent control | codex-cli, claude-cli |
| `api.registerCli(fn, opts)` | Register CLI subcommands | `memory`, `browser` |
| `api.registerGatewayMethod(name, handler, opts)` | Register Gateway RPC method | `browser.request` |
| `api.registerService(service)` | Register long-running service | Browser plugin service |
| `api.registerHttpRoute(route)` | Register HTTP endpoint | Webhook routes |
| `api.registerMemoryCapability(cap)` | Register memory subsystem | memory-core |
| `api.registerImageGenerationProvider(p)` | Register image gen provider | OpenAI DALL-E |
| `api.registerRealtimeTranscriptionProvider(p)` | Register transcription | OpenAI Whisper |
| `api.registerRealtimeVoiceProvider(p)` | Register voice provider | OpenAI realtime |
| `api.registerSpeechProvider(p)` | Register TTS/STT | OpenAI speech |
| `api.registerMediaUnderstandingProvider(p)` | Register media understanding | Anthropic, OpenAI |
| `api.registerVideoGenerationProvider(p)` | Register video gen | OpenAI video |
| `api.on(event, handler)` | Subscribe to lifecycle events | subagent_spawning |

### Event Hooks

Channel plugins can subscribe to lifecycle events:
- `subagent_spawning` -- before a subagent starts
- `subagent_ended` -- after a subagent finishes
- `subagent_delivery_target` -- routing subagent output

### Deferred Loading

Extensions use dynamic `import()` for heavy modules, loading them lazily:
```typescript
let discordSubagentHooksPromise: Promise<Module> | null = null;
function loadDiscordSubagentHooksModule() {
  discordSubagentHooksPromise ??= import("./subagent-hooks-api.js");
  return discordSubagentHooksPromise;
}
```

### Plugin Config Access

Plugins access their configuration through `api.pluginConfig`, `api.config`, and `api.logger`.

**Source files**: `extensions/browser/index.ts`, `extensions/discord/index.ts`, `extensions/anthropic/index.ts`, `extensions/openai/index.ts`, `extensions/memory-core/index.ts`, `extensions/webhooks/index.ts`

## Tool Registration

Tools are registered via the Plugin API with a factory function pattern:

```typescript
api.registerTool(
  ((ctx: OpenClawPluginToolContext) =>
    createBrowserTool({
      sandboxBridgeUrl: ctx.browser?.sandboxBridgeUrl,
      allowHostControl: ctx.browser?.allowHostControl,
      agentSessionKey: ctx.sessionKey,
    })) as OpenClawPluginToolFactory,
);
```

The factory receives a context object (`OpenClawPluginToolContext`) containing:
- `ctx.sessionKey` -- current agent session
- `ctx.config` -- resolved configuration
- `ctx.browser` -- browser-specific context (for browser plugin)

Tools can also be registered with explicit names:
```typescript
api.registerTool(
  (ctx) => createMemorySearchTool({ config: ctx.config, agentSessionKey: ctx.sessionKey }),
  { names: ["memory_search"] },
);
```

### Tool Catalog

All registered tools appear in the agent's tool catalog via `tools.catalog` RPC. Each tool self-describes with a JSON Schema:

```json
{
  "name": "browser",
  "description": "Browser automation: navigate, click, type, snapshot",
  "schema": {
    "type": "object",
    "properties": {
      "action": { "type": "string", "enum": ["open", "click", "type", "snapshot"] },
      "url": { "type": "string" }
    },
    "required": ["action"]
  }
}
```

### Tool Profiles

Predefined bundles control which tools are available:

| Profile | Tools | Use |
|---------|-------|-----|
| `none` | (empty) | Text-only assistant |
| `research` | Browser, web search | Information gathering |
| `creative` | Canvas, image generation | Content creation |
| `coding` | Browser, exec, cron | Code work |
| `dangerous` | All tools | Full access (requires approvals) |

Composition rule: `deny` always wins; `alsoAllow` adds; `allow` replaces.

### Tool Dispatch Targets

| Target | Tools | How |
|--------|-------|-----|
| Gateway host | browser, cron, webhooks, api | In-process |
| Paired node | camera.snap, screen.record | WS to device |
| Sandbox container | exec (when sandboxed) | Docker spawn |
| Plugin code | Plugin-provided tools | In-process plugin |

### Tool Streaming Events

```
tool_call -> tool_start -> tool_progress -> tool_result|tool_error -> tool_end
```

**Source files**: `extensions/browser/plugin-registration.ts`, `extensions/memory-core/index.ts`, Wiki "Tool System and Approvals"

## Security Model

### Trust Boundaries

OpenClaw operates under a **single trusted operator** model. Three boundary layers:

1. **Trusted Operator Boundary**: Gateway config, state directory, plugins, skills, memory files, authenticated callers -- all full operator trust
2. **Untrusted Input Boundary**: Channel messages, tool execution results, webhook payloads
3. **Isolation Boundaries** (defense-in-depth): Docker sandbox, network mode, workspace-only filesystem

### Plugin Trust Level

**Plugins are trusted code**. They run in-process within the Gateway with full operator privileges. There is no per-plugin sandboxing or capability restriction -- the boundary is at the Plugin SDK contract level, not at a security isolation level.

### Tool Approval Flow

Critical tools require operator approval before execution:

```
1. Generate approval ID (UUID)
2. Emit exec.approval.request event
3. Broadcast to all connected operators
4. Wait for approval (block tool execution)
5. Timeout after configured duration (default 5 min) -> reject
6. On approval: execute; on denial: return error
```

### Sandbox Pipeline

Docker-based sandboxing for tool execution:

```
getBlockedBindReason()        -- check bind mounts against denylist
validateSandboxSecurity()     -- validate sandbox config
isDangerousNetworkMode()      -- check network isolation
resolveSandboxConfigForAgent() -- resolve per-agent policy
Docker/OpenShell/SSH Backend  -- launch container
```

Blocked bind mounts include: `~/.ssh`, `~/.aws`, `~/.gnupg`, `/etc`, `/var`, `~/.openclaw`.

### Dangerous Config Flags

Flags prefixed `dangerous` or `dangerouslyAllow` bypass safety defaults:
- `dangerousAllowUnsafeExec`
- `dangerouslyAllowAllTools`
- `dangerousBrowserControl`

These are flagged by the `openclaw security audit` command.

### Security Audit System

```bash
openclaw security audit
```

Checks: filesystem permissions, gateway auth exposure, sandbox config, channel DM policies, installed skills code safety, tool policy.

### Skill Security

Third-party skills are code. Defenses:
- Allowlists per agent
- Dangerous code scanner on install (`skills.dangerousCode.mode`: `warn`/`block`/`allow`)
- Optional Docker sandboxing for untrusted skill code

**Source files**: Wiki "Security Model and Trust Boundaries", `extensions/browser/plugin-registration.ts` (securityAuditCollectors)

## Native Apps vs Extensions

| Aspect | Plugins (Extensions) | Nodes (Native Apps) |
|--------|---------------------|---------------------|
| Where they run | Gateway process (in-process) | Separate OS process / device app |
| How they register | At Gateway startup via manifest | At connect time via WS handshake |
| What they contribute | Channels, providers, tools, skills | Device capabilities (camera, screen, etc.) |
| Isolation | In-process (Plugin SDK boundary) | OS-process (network boundary) |
| Trust model | Trusted if installed | Paired + capability-gated |
| Protocol | Plugin SDK + direct function calls | WebSocket RPC |

### Node Capability Advertisement

When a native app connects, it advertises capabilities:

```json
{
  "role": "node",
  "capabilities": ["camera", "canvas", "screen", "location", "voice"],
  "commands": ["camera.snap", "canvas.navigate", "screen.record", "location.get", "system.run"]
}
```

The Gateway indexes devices by capability and routes commands to the appropriate node. Multiple devices can advertise the same capability; the Gateway picks the most-recently-active or routes by explicit `nodeId`.

### Platform Capabilities

- **macOS**: Canvas, camera, screen recording, location, shell commands, notifications (all TCC-gated)
- **iOS**: Canvas, camera, screen recording, location, notifications
- **Android**: Canvas, camera/video, screen recording, location, SMS, notifications, calendar, motion sensors

**Source**: Wiki "Extensions and Native Apps"

## Key Patterns for AGH

### 1. Manifest-First Discovery (High Priority)

OpenClaw's split between manifest inspection and code execution is its strongest pattern. AGH should adopt this: extension manifests (TOML, not JSON) are parsed and validated before any Go code loads. This enables `agh extensions status` without loading extension binaries.

**AGH adaptation**: Use a TOML manifest (`agh.extension.toml`) with fields like `id`, `capabilities`, `config_schema`, `requires`. Parse at daemon startup before loading extension binaries.

### 2. Uniform Plugin SDK Boundary (High Priority)

The rule that bundled and third-party extensions follow identical contracts is critical for ecosystem growth. No "blessed" plugins with privileged access.

**AGH adaptation**: Define Go interfaces in a `pkg/extension` or `internal/extension` package that all extensions implement. Use compile-time interface verification.

### 3. Four Plugin Types (Adapt)

OpenClaw's four types (channel, provider, tool, skill) map partially to AGH:
- **Provider** -> AGH's `AgentDriver` (ACP client for Claude, Codex, Gemini)
- **Tool** -> AGH could expose tools through ACP
- **Skill** -> AGH's skills system (YAML+Markdown, already planned)
- **Channel** -> Less relevant for AGH (no multi-channel messaging)

### 4. Typed Registration API (High Priority)

The `api.registerTool()`, `api.registerProvider()` pattern gives extensions a clean, typed surface for declaring capabilities. AGH should provide equivalent Go interfaces:

```go
type ExtensionAPI interface {
    RegisterDriver(driver AgentDriver)
    RegisterTool(factory ToolFactory)
    RegisterSkill(skill SkillDefinition)
    RegisterCLICommand(cmd *cobra.Command)
    RegisterHook(event string, handler HookHandler)
}
```

### 5. Dependency Ordering (Medium Priority)

Loading providers before channels before skills prevents registration-order bugs. AGH should define a similar ordering for its extension types.

### 6. Plugin Config with JSON Schema (Medium Priority)

Each extension declares its config schema in the manifest. The core validates config against this schema before passing it to the extension. AGH should use JSON Schema (or a Go equivalent like `go-jsonschema`) for extension config validation.

### 7. Deferred/Lazy Loading (Low Priority for Go)

OpenClaw uses dynamic `import()` for heavy modules. In Go, this maps to plugin loading via `plugin.Open()` or subprocess-based extensions. Since AGH is a single binary, this pattern is less directly applicable but could matter for optional agent drivers.

### 8. Security: Trusted but Auditable (High Priority)

OpenClaw's model -- plugins are trusted code but auditable -- is pragmatic for a single-operator system. AGH should adopt the same: extensions run with daemon privileges but are subject to `agh security audit`. No multi-tenant isolation within one daemon.

### 9. Skills as Teaching Layer Above Tools (High Priority)

The separation of tools (raw capabilities) from skills (instructions teaching the agent to use tools) is a powerful pattern AGH should adopt. Skills are YAML+Markdown files; tools are code. Skills compose tools; plugins compose everything.

### 10. Allowlist-Based Tool Profiles (Medium Priority)

The profile system (`coding`, `research`, `dangerous`) with `allow`/`deny` composition gives operators coarse and fine-grained control. AGH should implement similar per-session or per-agent tool profiles.

### 11. Hot-Reload Config (Low Priority)

The `reload: { restartPrefixes: ["browser"] }` pattern allows extensions to declare which config changes require reloading. Useful for long-running daemons.

## Code References

### Extension Directory Structure (Typical)

```
extensions/<name>/
  openclaw.plugin.json     -- static manifest (capabilities, config schema)
  package.json             -- npm metadata + openclaw block (entry points, install, compat)
  index.ts                 -- main entry point (definePluginEntry / defineBundledChannelEntry)
  api.ts                   -- public barrel (exports for core/tests to consume)
  runtime-api.ts           -- runtime barrel
  setup-entry.ts           -- onboarding/setup entry (channel plugins)
  register.runtime.ts      -- runtime registration logic
  plugin-registration.ts   -- tool/CLI/gateway method registration
  src/                     -- private implementation
  tsconfig.json            -- extends package-boundary base
  *.test.ts                -- co-located tests
```

### Key SDK Imports

| Import Path | Purpose |
|-------------|---------|
| `openclaw/plugin-sdk/plugin-entry` | `definePluginEntry`, `OpenClawPluginApi`, `OpenClawPluginToolContext` |
| `openclaw/plugin-sdk/channel-entry-contract` | `defineBundledChannelEntry`, `defineBundledChannelSetupEntry` |
| `openclaw/plugin-sdk/provider-entry` | Provider plugin entry contract |
| `openclaw/plugin-sdk/core` | Core helpers (Session, Message, etc.) |
| `openclaw/plugin-sdk/channel-contract` | Channel interface (connect, disconnect, send, status) |
| `openclaw/plugin-sdk/provider-auth` | Auth patterns for providers |
| `openclaw/plugin-sdk/extension-shared` | Shared utilities (deferred, passive monitor, status) |
| `openclaw/plugin-sdk/channel-config-primitives` | Channel config validation helpers |

### Shared Extension Infrastructure

`extensions/shared/` re-exports SDK utilities:

| File | Re-exports |
|------|-----------|
| `runtime.ts` | `resolveLoggerBackedRuntime` |
| `deferred.ts` | `createDeferred` |
| `passive-monitor.ts` | `runStoppablePassiveMonitor` |
| `status-issues.ts` | `coerceStatusIssueAccountId`, `readStatusIssueFields` |
| `channel-status-summary.ts` | `buildPassiveChannelStatusSummary`, `buildTrafficStatusSummary` |
| `config-schema-helpers.ts` | `requireChannelOpenAllowFrom` |

### Package Boundary Enforcement

- `tsconfig.package-boundary.base.json` extends `tsconfig.package-boundary.paths.json`
- Each extension's `tsconfig.json` extends the base, restricting import paths
- Only `openclaw/plugin-sdk/*` and local barrels (`./api.ts`) are valid imports
- No `src/**`, no `../other-extension/**`, no `openclaw/plugin-sdk-internal/**`

### Source File Locations

- Wiki docs: `/Users/pedronauck/dev/knowledge/openclaw/wiki/concepts/`
- Extension source: `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/`
- Extension boundary rules: `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/AGENTS.md`
- Shared infrastructure: `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/shared/`
- Discord (channel): `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/discord/`
- Browser (tool): `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/browser/`
- Anthropic (provider): `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/anthropic/`
- OpenAI (provider): `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/openai/`
- Memory Core (memory): `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/memory-core/`
- Webhooks (tool/http): `/Users/pedronauck/dev/knowledge/.resources/openclaw/extensions/webhooks/`
