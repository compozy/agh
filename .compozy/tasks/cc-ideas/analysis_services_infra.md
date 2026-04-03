# Services, State & Infrastructure Analysis

## How It Works (in Claude Code)

### Services Layer Architecture

Claude Code's `services/` directory contains ~30+ service modules organized as a flat collection of standalone services rather than a hierarchical dependency tree. Each service owns a well-defined domain boundary:

**Core Services:**
- `analytics/` - Event logging with a sink-pattern design. Events are queued pre-initialization and drained asynchronously once the sink is attached via `attachAnalyticsSink()`. This allows code to log events before the analytics backend is wired up.
- `api/` - Claude API client construction and request handling (the `claude.ts`, `client.ts` files).
- `claudeAiLimits.ts` - Rate limit management. Parses HTTP response headers (`anthropic-ratelimit-unified-*`) to track quota status, utilization percentages, and overage states. Emits status changes to a listener set.
- `compact/` - Conversation compaction subsystem. Contains 8+ files for different compaction strategies: `microCompact.ts` (token-level), `autoCompact.ts` (automatic triggers), `sessionMemoryCompact.ts` (memory-preserving), and `timeBasedMCConfig.ts` (cache-TTL-aware compaction).
- `mcp/` - Model Context Protocol client management. 23 files covering connection lifecycle, authentication (OAuth, XAA), server types (stdio, SSE, HTTP, WebSocket, SDK), normalization, and permission handling.
- `tokenEstimation.ts` - Multi-strategy token counting: exact API-based counting, Haiku-fallback counting, and rough estimation (bytes/4 or bytes/2 for JSON). Platform-aware (Bedrock, Vertex, 1P).
- `vcr.ts` - Test fixture recording/playback (Video Cassette Recorder pattern). Hashes API inputs, caches responses to disk, dehydrates/rehydrates environment-specific values (CWD, config paths).

**Lifecycle Services:**
- `preventSleep.ts` - macOS sleep prevention via `caffeinate` with reference counting (`startPreventSleep`/`stopPreventSleep`), self-healing timeouts, and cleanup registry integration.
- `notifier.ts` - Terminal notification abstraction. Auto-detects terminal type (iTerm2, Kitty, Ghostty) and dispatches notifications using the appropriate escape sequences.
- `diagnosticTracking.ts` - IDE diagnostic tracking service (singleton pattern). Captures baseline diagnostics before edits, computes diffs after, formats summaries. Communicates with IDE via MCP RPC calls.

**Intelligence Services:**
- `SessionMemory/` - Session memory persistence across compaction cycles.
- `PromptSuggestion/` - Speculative prompt suggestion system.
- `extractMemories/` - Extracts and persists learnings from conversations.
- `awaySummary.ts` - Generates "while you were away" recaps using a small model on the last 30 messages.
- `rateLimitMessages.ts` - Centralized rate limit message generation with tiered severity (error/warning), upsell text, and reset time formatting.

### State Management

Claude Code uses a **two-tier state architecture**:

**Tier 1: Bootstrap State (`bootstrap/state.ts`)** - A module-scoped singleton (`STATE`) with ~100+ getter/setter functions. This is "global process state" that must be accessible from anywhere without import cycles. It covers:
- Session identity (sessionId, parentSessionId, projectRoot, cwd)
- Cost tracking (totalCostUSD, totalAPIDuration, modelUsage by model)
- Telemetry state (OTel meter, counters, logger providers)
- Agent color assignments
- Feature flags and latches (afkModeHeaderLatched, fastModeHeaderLatched)
- Invoked skills tracking (Map keyed by `agentId:skillName`)
- Slow operation tracking for debugging

Key design constraint: The comment `DO NOT ADD MORE STATE HERE - BE JUDICIOUS WITH GLOBAL STATE` appears twice. The file uses a `getInitialState()` factory that returns the default, and `resetStateForTests()` re-initializes everything.

**Tier 2: App State (`state/store.ts` + `state/AppStateStore.ts`)** - A minimal reactive store:

```typescript
export type Store<T> = {
  getState: () => T
  setState: (updater: (prev: T) => T) => void
  subscribe: (listener: Listener) => () => void
}
```

The store is created with an optional `onChange` callback. `AppState` is a massive type (~450+ lines) covering:
- Settings, model selection, permission context
- MCP clients, tools, commands, resources
- Plugin state (enabled, disabled, errors, installation status)
- Task management (background agents, foreground tasks)
- Bridge/remote connection state
- UI state (expanded view, footer selection, overlays)
- Speculation state (active speculative execution)
- Team context (swarm members, inbox, worker permissions)

The `onChangeAppState` function acts as a side-effect dispatcher: when specific state slices change, it persists to config, notifies CCR/SDK, clears caches, or re-applies environment variables.

Selectors in `state/selectors.ts` derive computed state (e.g., `getViewedTeammateTask`, `getActiveAgentForInput`).

### Bridge/Communication System

The bridge system enables bidirectional communication between the CLI and claude.ai:

**Architecture:**
1. **Registration**: `bridgeApi.ts` registers an "environment" with the server, getting back an `environment_id` and `environment_secret`.
2. **Polling**: The bridge polls for work items (`pollForWork`) which can be sessions or healthchecks.
3. **WebSocket Transport**: Once a session is assigned, communication happens over WebSocket with SSE fallback.
4. **Message Flow**: Messages are filtered through `isEligibleBridgeMessage` (only user/assistant turns and slash-command system events), echo-deduplicated via `BoundedUUIDSet` (FIFO ring buffer), and routed to handlers.

**Control Protocol**: The bridge handles control requests (initialize, set_model, interrupt, set_permission_mode, set_max_thinking_tokens) with a request-response pattern. Unknown subtypes get error responses so the server doesn't hang.

**Security**: Path traversal prevention (`validateBridgeId`), OAuth retry on 401, trusted device tokens, outbound-only mode that rejects mutable requests.

**Spawn Modes**: `single-session`, `worktree` (git worktree per session), `same-dir` (shared CWD).

### Plugin System

**Plugin Types:**
1. **Built-in Plugins** (`plugins/builtinPlugins.ts`) - Ship with the CLI, togglable via `/plugin` UI. Identified by `{name}@builtin` format.
2. **Marketplace Plugins** - External plugins from git repositories. Identified by `{name}@{marketplace}`.
3. **Inline Plugins** - Session-only from `--plugin-dir` flag.

**Plugin Components**: A plugin can provide:
- Commands (slash commands)
- Agents (custom agent definitions)
- Skills (prompt templates)
- Hooks (lifecycle event handlers)
- Output styles
- MCP servers
- LSP servers

**Error Handling**: `PluginError` is a 22-variant discriminated union covering every failure mode (git auth, network, manifest parsing, MCP config, marketplace blocked by policy, dependency unsatisfied, etc.). Each variant carries contextual data, and `getPluginErrorMessage` formats them for display.

**Loading**: Plugins have `enabled`/`disabled` states. Installation status tracking supports `pending | installing | installed | failed` for background installation. A `needsRefresh` flag triggers reloading when plugin state changes on disk.

### Hooks System

Hooks are user-configurable lifecycle event handlers defined in `settings.json`. Four hook types:
- **Command** (`type: 'command'`) - Shell commands with optional `if` condition, shell type, timeout, async/once flags.
- **Prompt** (`type: 'prompt'`) - LLM evaluation with `$ARGUMENTS` placeholder.
- **Agent** (`type: 'agent'`) - Full agent invocation for verification.
- **HTTP** (`type: 'http'`) - Webhook POSTs with env-var interpolation in headers.

Each hook has a `matcher` (string pattern like tool names) and the hook event determines when it fires.

### Server Mode (Direct Connect)

The `server/` directory is minimal (3 files). `createDirectConnectSession` posts to `${serverUrl}/sessions` and returns a `DirectConnectConfig` with sessionId, wsUrl, and authToken. The `directConnectManager.ts` manages the lifecycle. Response validation uses Zod schemas.

### Remote Session Management

`remote/RemoteSessionManager.ts` manages WebSocket connections to remote sessions. It handles:
- SDKMessage routing (type guard filters control messages from data messages)
- Permission request/response bridging
- Reconnection and disconnection callbacks
- Viewer-only mode (no interrupt on Ctrl+C)

### Migrations

The `migrations/` directory contains 11 migration files, each a pure function that reads settings and conditionally rewrites them. Examples:
- `migrateFennecToOpus.ts` - Model alias migration (fennec-latest -> opus)
- `migrateSonnet45ToSonnet46.ts` - Model version bumps
- `resetProToOpusDefault.ts` - Default model changes
- `migrateAutoUpdatesToSettings.ts` - Config location changes

Migrations are idempotent (reading + writing the same source), touch only `userSettings` scope, and often include guards like `if (process.env.USER_TYPE !== 'ant') return`.

### Constants Organization

Constants are split by domain:
- `common.ts` - Date utilities with memoization for prompt-cache stability
- `system.ts` - System prompt prefixes, attribution headers
- `betas.ts` - Beta feature flags
- `apiLimits.ts` - API limit constants
- `errorIds.ts` - Error identifiers
- `files.ts` - File-related constants
- `tools.ts` / `toolLimits.ts` - Tool definitions and limits
- `prompts.ts` / `systemPromptSections.ts` - Prompt templates

---

## Key Patterns Worth Adopting

### 1. Sink Pattern for Analytics/Event Logging

Events are queued in memory until the backend sink is attached during app initialization. This decouples event producers from the transport layer.

```typescript
// From services/analytics/index.ts
const eventQueue: QueuedEvent[] = []
let sink: AnalyticsSink | null = null

export function attachAnalyticsSink(newSink: AnalyticsSink): void {
  if (sink !== null) return // Idempotent
  sink = newSink
  // Drain queued events via queueMicrotask (non-blocking)
}

export function logEvent(name: string, metadata: LogEventMetadata): void {
  if (!sink) { eventQueue.push({ eventName: name, metadata, async: false }); return }
  sink.logEvent(name, metadata)
}
```

**Why**: Eliminates initialization ordering issues. Any code can call `logEvent` at any time, even during bootstrap.

### 2. Minimal Reactive Store with Selector Pattern

A 34-line store implementation that supports the entire application's UI state.

```typescript
// From state/store.ts
export function createStore<T>(initialState: T, onChange?: OnChange<T>): Store<T> {
  let state = initialState
  const listeners = new Set<Listener>()
  return {
    getState: () => state,
    setState: (updater: (prev: T) => T) => {
      const prev = state
      const next = updater(prev)
      if (Object.is(next, prev)) return  // Skip no-ops
      state = next
      onChange?.({ newState: next, oldState: prev })
      for (const listener of listeners) listener()
    },
    subscribe: (listener: Listener) => {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
  }
}
```

**Why**: The `onChange` callback acts as a centralized side-effect dispatcher. State transitions are auditable in one place (`onChangeAppState`), making it easy to persist specific slices, sync with external systems, or clear caches.

### 3. Reference-Counted Resource Management

The `preventSleep.ts` pattern uses reference counting with self-healing cleanup.

```typescript
let refCount = 0
export function startPreventSleep(): void {
  refCount++
  if (refCount === 1) { spawnCaffeinate(); startRestartInterval() }
}
export function stopPreventSleep(): void {
  if (refCount > 0) refCount--
  if (refCount === 0) { stopRestartInterval(); killCaffeinate() }
}
export function forceStopPreventSleep(): void {
  refCount = 0; stopRestartInterval(); killCaffeinate()
}
```

The process auto-exits after 5 minutes (self-healing if Node is killed with SIGKILL), and is restarted every 4 minutes.

**Why**: Multiple callers can independently request sleep prevention without coordination. The cleanup registry ensures `forceStopPreventSleep` runs on exit.

### 4. BoundedUUIDSet for Echo Deduplication

A FIFO ring buffer backed by both a Set (O(1) lookup) and a circular array (bounded memory).

```typescript
// From bridge/bridgeMessaging.ts
export class BoundedUUIDSet {
  private readonly ring: (string | undefined)[]
  private readonly set = new Set<string>()
  private writeIdx = 0

  add(uuid: string): void {
    if (this.set.has(uuid)) return
    const evicted = this.ring[this.writeIdx]
    if (evicted !== undefined) this.set.delete(evicted)
    this.ring[this.writeIdx] = uuid
    this.set.add(uuid)
    this.writeIdx = (this.writeIdx + 1) % this.capacity
  }
  has(uuid: string): boolean { return this.set.has(uuid) }
}
```

**Why**: Constant-memory dedup for message echo filtering. Essential for bidirectional communication where messages may be reflected back.

### 5. VCR Test Fixture System

API responses are recorded to disk on first run and replayed from cache on subsequent runs. Values are dehydrated (CWD, config paths replaced with placeholders) for portability.

```typescript
// From services/vcr.ts
function dehydrateValue(s: unknown): unknown {
  if (typeof s !== 'string') return s
  return s
    .replaceAll(configHome, '[CONFIG_HOME]')
    .replaceAll(cwd, '[CWD]')
    .replace(/num_files="\d+"/g, 'num_files="[NUM]"')
    .replace(/duration_ms="\d+"/g, 'duration_ms="[DURATION]"')
}
```

**Why**: Tests against real API responses without network calls. Fixtures are committed to the repo. CI enforces `VCR_RECORD=1` for missing fixtures.

### 6. Discriminated Union Error Types

Plugin errors use a 22-variant discriminated union instead of string matching.

```typescript
// From types/plugin.ts
export type PluginError =
  | { type: 'git-auth-failed'; source: string; gitUrl: string; authType: 'ssh' | 'https' }
  | { type: 'manifest-parse-error'; source: string; manifestPath: string; parseError: string }
  | { type: 'marketplace-blocked-by-policy'; source: string; marketplace: string; blockedByBlocklist?: boolean }
  | { type: 'generic-error'; source: string; error: string }
  // ... 18 more variants
```

**Why**: Each error variant carries exactly the context needed for its display/handling. No string parsing, no lost information.

### 7. Two-Tier State Architecture (Bootstrap + App)

Bootstrap state (`bootstrap/state.ts`) is a module singleton with accessor functions, deliberately kept as a leaf in the import DAG. App state (`state/AppStateStore.ts`) is a reactive store for UI/runtime state.

**Why**: Bootstrap state avoids import cycles entirely. App state gets reactivity and selectors. The comment "DO NOT ADD MORE STATE HERE" enforces discipline.

### 8. Rate Limit State Machine from HTTP Headers

Rate limiting is driven entirely by response headers, with a clean state machine:

```typescript
type QuotaStatus = 'allowed' | 'allowed_warning' | 'rejected'
type ClaudeAILimits = {
  status: QuotaStatus
  resetsAt?: number
  rateLimitType?: RateLimitType
  utilization?: number      // 0-1 scale
  overageStatus?: QuotaStatus
  isUsingOverage?: boolean
}
```

The system computes warnings from both server-sent threshold headers AND client-side time-relative calculations (usage % vs time elapsed %).

### 9. Lazy Module Loading for Performance

Heavy dependencies are loaded only when needed:

```typescript
// From entrypoints/init.ts - OpenTelemetry deferred until telemetry init
const { initializeTelemetry } = await import('../utils/telemetry/instrumentation.js')

// From services/preventSleep.ts - plist only for Apple Terminal users
const plist = await import('plist')

// From commands.ts - Feature-gated command imports
const voiceCommand = feature('VOICE_MODE') ? require('./commands/voice/index.js').default : null
```

### 10. Idempotent Migrations with Scope Isolation

Migrations only touch `userSettings` scope (not project/local/policy), read and write the same source for idempotency, and use no completion flags.

```typescript
// From migrations/migrateFennecToOpus.ts
export function migrateFennecToOpus(): void {
  if (process.env.USER_TYPE !== 'ant') return
  const settings = getSettingsForSource('userSettings')  // Read ONLY userSettings
  const model = settings?.model
  if (typeof model === 'string' && model.startsWith('fennec-latest')) {
    updateSettingsForSource('userSettings', { model: 'opus' })  // Write ONLY userSettings
  }
}
```

---

## Ideas for Our System

### 1. Event Bus with Deferred Sink Attachment

Implement an event bus in Go that queues events until a sink is attached. This solves the common problem of logging during initialization before the logger is ready.

```go
type EventBus struct {
    mu     sync.Mutex
    queue  []Event
    sink   EventSink
}

func (b *EventBus) Emit(event Event) {
    b.mu.Lock()
    defer b.mu.Unlock()
    if b.sink != nil {
        b.sink.Handle(event)
        return
    }
    b.queue = append(b.queue, event)
}

func (b *EventBus) AttachSink(sink EventSink) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.sink = sink
    for _, e := range b.queue {
        sink.Handle(e)
    }
    b.queue = nil
}
```

### 2. Reactive State Store with onChange Side-Effects

Port the minimal store pattern to Go using generics. The `onChange` callback pattern is particularly powerful for syncing state to disk, notifying remote systems, or clearing caches.

```go
type Store[T any] struct {
    mu        sync.RWMutex
    state     T
    listeners map[int]func()
    onChange   func(old, new T)
    nextID    int
}

func (s *Store[T]) SetState(updater func(T) T) {
    s.mu.Lock()
    old := s.state
    s.state = updater(old)
    if s.onChange != nil {
        s.onChange(old, s.state)
    }
    for _, listener := range s.listeners {
        listener()
    }
    s.mu.Unlock()
}
```

### 3. Reference-Counted Resources with Self-Healing

Apply the `preventSleep` pattern to our daemon's resource management (e.g., preventing system sleep, holding file locks, maintaining connections):

- Reference count tracks concurrent users of a resource
- Self-healing timeout ensures cleanup even if the process crashes
- Cleanup registry integration for graceful shutdown

### 4. BoundedSet for Message Deduplication

Implement a bounded ring-buffer set for our bridge/IPC layer. This is critical for any system where messages can echo or be redelivered.

### 5. Discriminated Union Error Types for Plugin/Skill System

Our skills system should use typed error enums instead of generic errors:

```go
type SkillError interface {
    ErrorType() string
    ErrorMessage() string
}

type SkillNotFoundError struct {
    SkillName   string
    SearchPaths []string
}
func (e *SkillNotFoundError) ErrorType() string { return "skill_not_found" }
```

### 6. Two-Tier State: Bootstrap + Runtime

Adopt the two-tier state pattern:
- **Bootstrap state**: Singleton in `internal/bootstrap/state.go` with getter/setter functions, zero import dependencies. Houses session ID, CWD, costs, feature flags.
- **Runtime state**: Reactive store in `internal/state/store.go` for UI-relevant state that changes during operation.

### 7. VCR-Style Test Fixtures for API Calls

Build a fixture recording/playback system for our LLM API tests:
- Hash request parameters to generate fixture filenames
- Dehydrate environment-specific values (paths, timestamps, UUIDs)
- CI fails on missing fixtures, requires explicit recording

### 8. Rate Limit State Machine

Implement a clean rate limit tracker that:
- Parses limits from HTTP response headers
- Computes warnings using both server thresholds and client-side time-relative calculations
- Emits status changes to subscribers
- Provides centralized message formatting (error vs warning severity)

### 9. Idempotent Migration Framework

Build a migration system for settings/config:

```go
type Migration struct {
    Name    string
    Scope   SettingsScope  // Only touches one scope
    Apply   func(settings *Settings) *Settings
}
```

Migrations read and write the same scope for idempotency, require no completion flags, and are safe to re-run.

### 10. Plugin Architecture with Typed Component System

Design our plugin system with:
- Component types: Skills, Hooks, MCP Servers, Commands
- Error discriminated unions per component type
- Installation status tracking (pending/installing/installed/failed)
- `needsRefresh` flag for hot-reloading
- Built-in vs marketplace vs inline (session-only) plugin sources

### 11. Bridge Communication Pattern

For any remote control / web UI bridge:
- Register environment -> poll for work -> acknowledge work -> WebSocket for session
- Echo deduplication with BoundedUUIDSet
- Control request/response protocol with timeout safety (respond to unknown types with error)
- Outbound-only mode for read-only connections
- Path traversal prevention on server-provided IDs

### 12. Lazy Initialization Pattern

Defer heavy module loading:

```go
var heavyModule sync.Once
var heavyInstance *HeavyThing

func getHeavyThing() *HeavyThing {
    heavyModule.Do(func() {
        heavyInstance = initializeHeavyThing()
    })
    return heavyInstance
}
```

Apply to: telemetry, analytics exporters, rarely-used tools.

---

## Key Files Reference

### Services Layer
| File | Description |
|------|-------------|
| `services/analytics/index.ts` | Event logging with sink-pattern, event queuing, PII-safe marker types |
| `services/vcr.ts` | VCR test fixture recording/playback with dehydration/rehydration |
| `services/preventSleep.ts` | Ref-counted macOS sleep prevention with self-healing timeouts |
| `services/notifier.ts` | Terminal notification abstraction (iTerm2, Kitty, Ghostty, bell) |
| `services/diagnosticTracking.ts` | IDE diagnostic tracking singleton with baseline diffing |
| `services/tokenEstimation.ts` | Multi-strategy token counting (API, Haiku fallback, rough) |
| `services/claudeAiLimits.ts` | Rate limit state machine from HTTP headers, status change emission |
| `services/rateLimitMessages.ts` | Centralized rate limit message formatting with severity tiers |
| `services/awaySummary.ts` | "While you were away" recap generation using small model |
| `services/compact/` | 8-file conversation compaction subsystem |
| `services/mcp/` | 23-file MCP client management (auth, transport, config) |
| `services/mcp/types.ts` | MCP configuration schemas (Zod) for stdio, SSE, HTTP, WebSocket, SDK |

### State Management
| File | Description |
|------|-------------|
| `state/store.ts` | 34-line minimal reactive store (getState, setState, subscribe) |
| `state/AppStateStore.ts` | 570-line AppState type definition with defaults factory |
| `state/onChangeAppState.ts` | Side-effect dispatcher for state transitions (persist, sync, cache clear) |
| `state/selectors.ts` | Pure computed state derivation (viewed teammate, active agent routing) |
| `state/AppState.tsx` | React integration (useSyncExternalStore, selector-based subscriptions) |
| `bootstrap/state.ts` | 1758-line module singleton for process-level state (zero-dep leaf) |

### Bridge/Communication
| File | Description |
|------|-------------|
| `bridge/bridgeApi.ts` | HTTP API client for environment registration, work polling, heartbeat |
| `bridge/bridgeMessaging.ts` | Message routing, echo dedup (BoundedUUIDSet), control request handling |
| `bridge/types.ts` | Bridge protocol types (WorkResponse, SessionHandle, SpawnMode, BridgeConfig) |
| `bridge/bridgeConfig.ts` | Bridge configuration resolution |
| `bridge/replBridgeTransport.ts` | WebSocket/SSE transport abstraction |

### Plugin System
| File | Description |
|------|-------------|
| `plugins/builtinPlugins.ts` | Built-in plugin registry with enable/disable, skill-to-command conversion |
| `types/plugin.ts` | Plugin types: 22-variant PluginError union, LoadedPlugin, PluginManifest |

### Infrastructure
| File | Description |
|------|-------------|
| `commands.ts` | Top-level command registry with feature-gated lazy imports |
| `entrypoints/init.ts` | Application initialization (config, mTLS, proxy, telemetry, cleanup) |
| `constants/system.ts` | System prompt prefixes, attribution header construction |
| `constants/common.ts` | Memoized date utilities for prompt-cache stability |
| `schemas/hooks.ts` | Zod schemas for 4 hook types (command, prompt, agent, HTTP) |
| `types/permissions.ts` | Permission type definitions (modes, behaviors, rules, decisions) |
| `migrations/` | 11 idempotent setting migration files |
| `server/createDirectConnectSession.ts` | Direct-connect session creation with Zod response validation |
| `remote/RemoteSessionManager.ts` | Remote session WebSocket management with permission bridging |
