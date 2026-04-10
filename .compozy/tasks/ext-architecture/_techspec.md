# TechSpec: Extension Architecture

## Executive Summary

AGH's extension architecture enables third-party developers to extend the daemon's capabilities through a three-tier execution model: Go-native interfaces (L1), WebAssembly sandbox via Extism (L2), and JSON-RPC subprocess (L3). Extensions are modeled as three-dimensional packages that bundle **resources** (agents, skills, hooks, MCP configs), provide **capabilities** (agent drivers, memory backends, observe exporters), and perform **actions** via a bidirectional Host API (create sessions, manage memory, query events).

The architecture builds on AGH's existing infrastructure: the 27-event hook system with typed dispatch provides the extension dispatch layer, the ACP subprocess pattern provides the L3 prototype, and the WASM executor stub provides the L2 seam. The primary technical trade-off is surface area vs. power — a full three-tier model with rich Host API provides maximum extensibility but requires capability-scoped security at every boundary to prevent extensions from exceeding their declared privileges.

The council debate stress-tested all decisions. Key adjustments incorporated: capability-scoped security at the Host API boundary (not just process isolation), `ExecutorConfig` isolation to prevent `HookDecl` from becoming a God struct, a minimal `Tool` struct to ground the existing hook tool dispatch, and daemon-context failure recovery for headless extension execution.

---

## System Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                       AGH Daemon (Go)                           │
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐ │
│  │  Extension        │  │  Hook System     │  │  Session      │ │
│  │  Manager          │  │  (existing)      │  │  Manager      │ │
│  │                   │  │                  │  │  (existing)   │ │
│  │  - Registry       │  │  - 27 events     │  │              │ │
│  │  - Manifest load  │  │  - Typed dispatch│  │  - AgentDriver│ │
│  │  - Lifecycle mgmt │  │  - 3 executors   │  │  - Lifecycle  │ │
│  │  - Capability     │  │  - Hot-reload    │  │  - Events     │ │
│  │    enforcement    │  │                  │  │              │ │
│  └────────┬─────────┘  └────────┬─────────┘  └──────┬──────┘ │
│           │                      │                     │        │
│  ┌────────┴──────────────────────┴─────────────────────┴──────┐ │
│  │                   Extension Tiers                           │ │
│  │                                                             │ │
│  │  ┌─────────────┐  ┌─────────────────┐  ┌────────────────┐ │ │
│  │  │ L1: Go      │  │ L2: WASM        │  │ L3: Subprocess │ │ │
│  │  │ Native      │  │ (Extism/wazero) │  │ (JSON-RPC)     │ │ │
│  │  │             │  │                 │  │                │ │ │
│  │  │ Compiled-in │  │ In-process      │  │ Out-of-process │ │ │
│  │  │ interfaces  │  │ sandbox         │  │ bidirectional  │ │ │
│  │  │             │  │ fuel-metered    │  │ Host API       │ │ │
│  │  └─────────────┘  └─────────────────┘  └────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Host API                               │   │
│  │  sessions/* │ memory/* │ skills/* │ observe/* │ hooks/*   │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Install**: `agh extension install <path>` → parse manifest → validate capabilities → copy resources → register in extension registry
2. **Boot**: Daemon starts → Extension Manager loads enabled extensions → L2: compile WASM modules → L3: launch subprocesses → capability negotiation handshake
3. **Runtime**: Hook dispatched → Extension Manager routes to appropriate executor → capability check → execute → return result
4. **Host API call** (L3 only): Extension sends JSON-RPC request → capability check → execute on daemon → return result
5. **Shutdown**: Daemon stops → Extension Manager sends shutdown to all L3 subprocesses → wait with timeout → SIGKILL stragglers → close WASM runtime

---

## Implementation Design

### Core Interfaces

**Extension Manager** — the orchestrator that lives in `internal/extension/`:

```go
// internal/extension/manager.go
type Manager struct {
    mu          sync.RWMutex
    registry    *Registry
    wasm        *WasmRuntime
    subprocesses map[string]*Subprocess
    capChecker  *CapabilityChecker
    logger      *slog.Logger
}

func NewManager(registry *Registry, opts ...Option) *Manager
func (m *Manager) Start(ctx context.Context) error
func (m *Manager) Stop(ctx context.Context) error
func (m *Manager) Get(name string) (*Extension, error)
func (m *Manager) List() []ExtensionInfo
```

**Extension manifest** — parsed from `extension.toml` or `extension.json`:

```go
// internal/extension/manifest.go
type Manifest struct {
    Name        string            `toml:"name"`
    Version     string            `toml:"version"`
    Description string            `toml:"description"`
    MinAGH      string            `toml:"min_agh_version"`
    Type        ExtensionType     `toml:"type"` // "wasm" | "subprocess"
    Resources   ResourcesConfig   `toml:"resources"`
    Capabilities CapabilitiesConfig `toml:"capabilities"`
    Actions     ActionsConfig     `toml:"actions"`
    Subprocess  *SubprocessConfig `toml:"subprocess,omitempty"`
    Wasm        *WasmConfig       `toml:"wasm,omitempty"`
}
```

**WASM Executor** — fills the existing stub via Extism:

```go
// internal/hooks/executor_wasm.go
type WasmExecutor struct {
    plugin *extism.Plugin
    config WasmExecutorConfig
}

func (e *WasmExecutor) Kind() HookExecutorKind { return HookExecutorWASM }
func (e *WasmExecutor) Execute(ctx context.Context,
    hook RegisteredHook, payload []byte) ([]byte, error)
```

**Subprocess extension** — generalized from ACP:

```go
// internal/subprocess/process.go
type Process struct {
    cmd     *exec.Cmd
    rpc     *jsonrpc2.Conn
    caps    NegotiatedCapabilities
    health  HealthState
    logger  *slog.Logger
}

func Launch(ctx context.Context, cfg LaunchConfig) (*Process, error)
func (p *Process) Call(ctx context.Context, method string,
    params, result any) error
func (p *Process) Shutdown(ctx context.Context) error
```

**Capability Checker** — enforces ADR-003:

```go
// internal/extension/capability.go
type CapabilityChecker struct {
    grants map[string]CapabilityGrant // extension name → grants
    tiers  map[ExtensionSource][]string // source → max capabilities
}

func (c *CapabilityChecker) Check(extName string,
    capability string) error
func (c *CapabilityChecker) CheckHostAPI(extName string,
    method string) error
```

**Minimal Tool struct** — grounds the hook tool dispatch:

```go
// internal/tools/tool.go
type Tool struct {
    Name        string          `json:"name"`
    Description string          `json:"description"`
    InputSchema json.RawMessage `json:"input_schema"`
    ReadOnly    bool            `json:"read_only"`
    Source      ToolSource      `json:"source"`
}

type ToolProvider interface {
    Tools(ctx context.Context) ([]Tool, error)
}
```

### Data Models

**Extension Registry** — persisted in global DB:

| Field | Type | Description |
|---|---|---|
| `name` | TEXT PK | Extension unique identifier |
| `version` | TEXT | Semver version |
| `type` | TEXT | "wasm" \| "subprocess" |
| `source` | TEXT | "bundled" \| "user" \| "workspace" \| "marketplace" |
| `enabled` | BOOLEAN | Whether extension is active |
| `manifest_path` | TEXT | Path to manifest file |
| `installed_at` | TIMESTAMP | Installation time |
| `capabilities` | TEXT (JSON) | Declared capabilities |
| `actions` | TEXT (JSON) | Declared Host API actions |
| `checksum` | TEXT | SHA-256 of extension artifact |

**Extension Manifest** (TOML example):

```toml
[extension]
name = "pgvector-memory"
version = "0.2.1"
description = "PostgreSQL pgvector memory backend for AGH"
type = "subprocess"
min_agh_version = "0.5.0"

[resources]
skills = ["skills/"]           # Directory of SKILL.md files to register
agents = ["agents/"]           # Directory of AGENT.md files to register
hooks = []                     # Inline hook declarations (if any)
mcp_servers = []               # MCP server configs to register

[capabilities]
provides = ["memory.backend"]  # What this extension provides to AGH

[actions]
requires = [                   # Host API methods this extension needs
    "sessions.list",
    "sessions.events",
    "memory.store",
    "memory.recall",
]

[subprocess]
command = "agh-ext-pgvector"
args = ["--config", "{{config_dir}}/pgvector.toml"]
health_check_interval = "30s"
shutdown_timeout = "10s"

[subprocess.env]
PGVECTOR_URL = "{{env:PGVECTOR_URL}}"

[security]
capabilities = [               # Capability families (coarse-grained)
    "memory.read",
    "memory.write",
    "session.read",
]
```

**WASM Manifest** (TOML example):

```toml
[extension]
name = "content-filter"
version = "1.0.0"
description = "Content safety validator for agent output"
type = "wasm"
min_agh_version = "0.5.0"

[capabilities]
provides = ["content.validate", "message.transform"]

[wasm]
module = "content_filter.wasm"
fuel_limit = 100000              # Max Wasm instructions per call
memory_limit_pages = 16          # Max Wasm memory (16 × 64KB = 1MB)
timeout = "500ms"                # Per-call timeout
exports = [                      # Functions the host can call
    "validate_content",
    "transform_message",
]

[wasm.host_functions]            # AGH capabilities exposed to Wasm
allowed = ["log"]                # Deny-by-default; only logging allowed

[security]
capabilities = ["message.read", "message.write"]
```

### Extension Manifest (JSON alternative):

```json
{
    "extension": {
        "name": "pgvector-memory",
        "version": "0.2.1",
        "description": "PostgreSQL pgvector memory backend for AGH",
        "type": "subprocess",
        "min_agh_version": "0.5.0"
    },
    "resources": {
        "skills": ["skills/"],
        "agents": ["agents/"]
    },
    "capabilities": {
        "provides": ["memory.backend"]
    },
    "actions": {
        "requires": ["sessions.list", "memory.store", "memory.recall"]
    },
    "subprocess": {
        "command": "agh-ext-pgvector",
        "args": ["--config", "{{config_dir}}/pgvector.toml"]
    },
    "security": {
        "capabilities": ["memory.read", "memory.write", "session.read"]
    }
}
```

### Host API (L3 Subprocess Extensions → AGH)

Bidirectional JSON-RPC 2.0 over stdio. Extensions call these methods on the daemon:

**Session Methods:**

| Method | Params | Result | Capability |
|---|---|---|---|
| `sessions/list` | `{workspace?: string}` | `[{id, name, agent, state, created_at}]` | `session.read` |
| `sessions/create` | `{agent, prompt?, workspace?}` | `{session_id}` | `session.write` |
| `sessions/prompt` | `{session_id, message}` | `{turn_id}` | `session.write` |
| `sessions/stop` | `{session_id}` | `{}` | `session.write` |
| `sessions/status` | `{session_id}` | `{state, agent, started_at, ...}` | `session.read` |
| `sessions/events` | `{session_id, limit?, offset?}` | `[{type, timestamp, data}]` | `session.read` |

**Memory Methods:**

| Method | Params | Result | Capability |
|---|---|---|---|
| `memory/recall` | `{query, scope?, limit?}` | `[{key, content, score}]` | `memory.read` |
| `memory/store` | `{key, content, scope?, tags?}` | `{}` | `memory.write` |
| `memory/forget` | `{key, scope?}` | `{}` | `memory.write` |

**Observe Methods:**

| Method | Params | Result | Capability |
|---|---|---|---|
| `observe/health` | `{}` | `{uptime, sessions, extensions, ...}` | `observe.read` |
| `observe/events` | `{session_id?, type?, limit?}` | `[{type, timestamp, data}]` | `observe.read` |

**Skills Methods:**

| Method | Params | Result | Capability |
|---|---|---|---|
| `skills/list` | `{workspace?}` | `[{name, description, source}]` | `skills.read` |

**AGH → Extension Methods:**

| Method | Description | Used By |
|---|---|---|
| `initialize` | Capability negotiation handshake | L3 |
| `execute_hook` | Dispatch a hook event to the extension | L2, L3 |
| `provide_tools` | Request tool definitions from extension | L3 |
| `health_check` | Liveness probe | L3 |
| `shutdown` | Graceful shutdown request | L3 |

### Extension Loading Pipeline

Six-phase pipeline (inspired by OpenClaw, validated across 5/6 harnesses):

```
1. DISCOVER    → Scan extension directories, find manifests
2. PARSE       → Read extension.toml/json, validate schema (no code execution)
3. VALIDATE    → Check version compatibility, verify checksums, validate capabilities
4. REGISTER    → Copy resources (skills, agents, hooks) into AGH registries
5. INITIALIZE  → L2: compile WASM modules. L3: launch subprocesses, handshake
6. ACTIVATE    → Extension is live, hooks dispatch to it, Host API available
```

Each phase can fail independently with clear error messages. A corrupt WASM module does not prevent subprocess extensions from loading.

---

## Integration Points

### Daemon Composition Root (`internal/daemon/boot.go`)

The Extension Manager is initialized between the hooks system and the servers:

```
Phase 4: Skills Registry          (existing)
Phase 5: Global Registry          (existing)
Phase 8: Session Manager          (existing)
Phase 9: Hooks System             (existing)
  ↓
Phase 9.5: Extension Manager      (NEW)
  - Load extension registry from global DB
  - Discover installed extensions
  - Parse manifests
  - Register resources (skills, agents, hooks into existing registries)
  - Initialize WASM runtime (Extism)
  - Launch subprocess extensions
  - Capability negotiation
  - Inject extension-provided hook declarations into hooks.Rebuild()
  ↓
Phase 10: Skills Watcher          (existing)
Phase 11: Servers                 (existing)
```

### Hook System Integration

Extensions provide hook declarations through a new `DeclarationProvider`:

```go
// internal/daemon/hooks_bridge.go (extend existing)
func extensionDeclarationProvider(extMgr *extension.Manager) hooks.DeclarationProvider {
    return func(ctx context.Context) ([]hooks.HookDecl, error) {
        return extMgr.HookDeclarations(ctx)
    }
}
```

WASM hooks use the new `WasmExecutor` via the existing `ExecutorResolver`:

```go
func daemonExecutorResolver(extMgr *extension.Manager) hooks.ExecutorResolver {
    return func(decl hooks.HookDecl) (hooks.Executor, error) {
        switch decl.ExecutorKind {
        case hooks.HookExecutorWASM:
            return extMgr.WasmExecutorFor(decl)
        // ... existing native, subprocess cases
        }
    }
}
```

### ACP Integration

Extract shared subprocess lifecycle from `internal/acp/client.go` into `internal/subprocess/`:

```
internal/subprocess/
    process.go        — Launch, Call, Shutdown, health monitoring
    transport.go      — JSON-RPC framing over stdin/stdout
    handshake.go      — Initialize/capability negotiation
    signals.go        — Graceful shutdown with signal escalation

internal/acp/
    client.go         — ACP-specific methods (imports internal/subprocess)
    handlers.go       — ACP inbound handlers (existing)

internal/extension/
    subprocess.go     — Extension-specific methods (imports internal/subprocess)
    host_api.go       — Host API handler (inbound from extensions)
```

---

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|---|---|---|---|
| `internal/hooks/executor_wasm_stub.go` | Modified | Replace stub with Extism implementation | Fill executor, add `WasmExecutorConfig` |
| `internal/hooks/types.go` | Modified | Add `ExecutorConfig` field to avoid widening `HookDecl` | Low risk — additive field |
| `internal/acp/client.go` | Modified | Extract subprocess lifecycle into shared package | Medium risk — refactor existing working code |
| `internal/daemon/boot.go` | Modified | Add Extension Manager initialization phase | Low risk — additive phase in boot sequence |
| `internal/daemon/hooks_bridge.go` | Modified | Add extension declaration provider and executor resolver | Low risk — extends existing patterns |
| `internal/extension/` | New | Extension Manager, Registry, manifest loading, capability enforcement | New package — core of this techspec |
| `internal/subprocess/` | New | Shared subprocess lifecycle extracted from ACP | New package — refactor, not new functionality |
| `internal/tools/` | New | Minimal Tool struct and ToolProvider interface | New package — ~200 LOC |
| `internal/store/globaldb/` | Modified | Add extension registry table | Low risk — new table, no schema changes to existing |
| `internal/cli/` | Modified | Add `agh extension list/install/enable/disable` commands | Additive CLI commands |
| `go.mod` | Modified | Add `github.com/extism/go-sdk` dependency | New dependency — ~5-8MB binary size increase |

---

## TypeScript SDK

### Package Structure

```
@agh/extension-sdk/
    src/
        extension.ts           — Main Extension class
        transport.ts           — StdioTransport (JSON-RPC over stdin/stdout)
        host-api.ts            — Typed Host API client (sessions, memory, etc.)
        types.ts               — TypeScript types matching AGH contracts
        capabilities.ts        — Capability declaration helpers
        contracts/
            memory-backend.ts  — Type definitions for memory.backend
            agent-driver.ts    — Type definitions for agent.driver
            observe-exporter.ts — Type definitions for observe.exporter
            session-hooks.ts   — Type definitions for hook payloads
        testing/
            mock-transport.ts  — In-memory transport for unit tests
            harness.ts         — Test harness simulating AGH host
    bin/
        create-extension.ts    — CLI scaffolding: `npx @agh/create-extension`
    templates/
        hook-subprocess/       — Starter template for hook extension
        memory-backend/        — Starter template for memory backend
```

### Core API

```typescript
import { Extension, HostAPI } from '@agh/extension-sdk';

const ext = new Extension({
    name: 'my-memory-backend',
    version: '0.1.0',
    capabilities: { provides: ['memory.backend'] },
    actions: { requires: ['sessions.list'] },
});

// Handle daemon → extension calls
ext.handle('memory/store', async (ctx, params: StoreParams) => {
    await db.insert(params.key, params.content);
    return { success: true };
});

ext.handle('memory/recall', async (ctx, params: RecallParams) => {
    const results = await db.search(params.query, params.limit);
    return { entries: results };
});

// Call Host API (extension → daemon)
ext.onReady(async (host: HostAPI) => {
    const sessions = await host.sessions.list();
    console.error(`Connected. ${sessions.length} active sessions.`);
});

ext.start(); // Reads stdin, writes stdout
```

### WASM PDK (AssemblyScript)

```typescript
// Written in AssemblyScript, compiles to .wasm
import { Host, JSON } from '@agh/wasm-pdk';

export function validate_content(): i32 {
    const input = Host.inputString();
    const msg = JSON.parse<MessagePayload>(input);

    if (containsPII(msg.text)) {
        Host.outputString(JSON.stringify({
            allow: false,
            reason: "Content contains PII",
        }));
        return 0;
    }

    Host.outputString(JSON.stringify({ allow: true }));
    return 0;
}
```

### Test Harness

```typescript
import { TestHarness } from '@agh/extension-sdk/testing';

const harness = new TestHarness();
harness.mockHostAPI('sessions/list', () => [
    { id: 'sess-1', name: 'test', agent: 'claude', state: 'active' },
]);

const ext = harness.loadExtension('./my-extension');
const result = await harness.call('memory/store', {
    key: 'test', content: 'hello',
});
expect(result.success).toBe(true);
```

---

## Testing Approach

### Unit Tests

- **Extension Manager**: Mock WASM runtime and subprocess launcher. Test lifecycle (start, stop, restart). Test capability enforcement (authorized vs unauthorized calls).
- **WASM Executor**: Test with pre-compiled `.wasm` test fixtures. Test fuel metering limits. Test timeout enforcement. Test crash recovery (Wasm trap).
- **Manifest Parser**: Table-driven tests for TOML and JSON manifests. Test validation (missing fields, invalid versions, unknown capabilities). Test both formats produce identical `Manifest` structs.
- **Capability Checker**: Test all source-trust tier combinations. Test wildcard grants. Test unauthorized access returns typed errors.
- **Host API Handler**: Test each method with authorized and unauthorized callers. Test parameter validation. Test error responses.
- **Tool struct**: Test `ToolProvider` interface. Test tool serialization matches hook `ToolCallRef` payloads.

### Integration Tests

- **End-to-end WASM hook**: Install a test WASM extension → dispatch a hook → verify the extension receives the payload and returns a valid patch.
- **End-to-end subprocess extension**: Install a test subprocess extension → daemon boots → handshake completes → extension calls Host API → verify results.
- **Extension lifecycle**: Install → enable → daemon restart → verify extension reloads → disable → verify extension stops → uninstall → verify cleanup.
- **Capability enforcement**: Install extension with limited capabilities → attempt unauthorized Host API call → verify rejection with typed error.
- **Resource registration**: Install extension with skills and agents → verify they appear in skills registry and agent definitions.

---

## Development Sequencing

### Build Order

1. **`internal/tools/` — Minimal Tool struct + ToolProvider** — no dependencies. ~200 LOC. Grounds the hook tool dispatch that already exists.

2. **`internal/subprocess/` — Shared subprocess lifecycle** — depends on step 1 (none, actually, but logically follows). Extract from `internal/acp/client.go`: process launch, JSON-RPC framing, handshake, graceful shutdown, health monitoring.

3. **`internal/hooks/executor_wasm.go` — WASM executor via Extism** — depends on step 2 (no direct dep, but concurrent). Fill the existing stub. Add `WasmExecutorConfig`. Wire into `ExecutorResolver`.

4. **`internal/extension/manifest.go` — Manifest parser** — depends on step 3 (none). Parse `extension.toml` and `extension.json`. Validate schema. Produce `Manifest` struct.

5. **`internal/extension/capability.go` — Capability checker** — depends on step 4 (reads capabilities from manifest). Source-trust tier enforcement. Dispatch-time and Host API checks.

6. **`internal/extension/registry.go` — Extension registry** — depends on steps 4, 5. SQLite table in global DB. CRUD operations. Enabled/disabled state.

7. **`internal/extension/manager.go` — Extension Manager** — depends on steps 2, 3, 5, 6. Orchestrates lifecycle: discover → parse → validate → register → initialize → activate. Wires into daemon boot.

8. **`internal/extension/host_api.go` — Host API handler** — depends on steps 5, 7. JSON-RPC method handlers for sessions/*, memory/*, observe/*, skills/*. Capability-checked.

9. **`internal/daemon/boot.go` — Daemon integration** — depends on step 7. Add Extension Manager phase to boot sequence. Wire declaration provider and executor resolver.

10. **`internal/cli/extension.go` — CLI commands** — depends on steps 6, 7. `agh extension list`, `install`, `enable`, `disable`.

11. **`@agh/extension-sdk` — TypeScript SDK** — depends on steps 7, 8. npm package with Extension class, StdioTransport, Host API client, test harness.

12. **Reference extensions** — depends on steps 3, 7, 8. Three working examples: native hook, subprocess hook, WASM hook.

### Technical Dependencies

- **Extism Go SDK** (`github.com/extism/go-sdk` v1.3.0+): Must be added to `go.mod` via `go get`.
- **sourcegraph/jsonrpc2** or equivalent: Evaluate whether AGH's existing JSON-RPC code in `internal/acp/` is sufficient or if a library is needed for the shared subprocess package.
- **Node.js 18+**: Required for TypeScript SDK development and testing.

---

## Monitoring and Observability

### Key Metrics

- `agh_extensions_loaded{tier, name, state}` — Gauge of loaded extensions by tier and state
- `agh_extension_hook_duration_ms{extension, event, tier}` — Histogram of hook execution time
- `agh_extension_host_api_calls{extension, method, status}` — Counter of Host API calls
- `agh_extension_wasm_fuel_consumed{extension}` — Counter of WASM fuel consumed
- `agh_extension_subprocess_restarts{extension}` — Counter of subprocess restart events
- `agh_extension_capability_denied{extension, capability}` — Counter of denied capability checks

### Log Events

| Event | Level | Fields |
|---|---|---|
| Extension loaded | INFO | name, version, type, tier, capabilities |
| Extension failed to load | ERROR | name, error, phase (discover/parse/validate/initialize) |
| WASM fuel limit reached | WARN | name, function, fuel_consumed, fuel_limit |
| Subprocess crashed | ERROR | name, exit_code, stderr_tail, restart_count |
| Host API call | DEBUG | extension, method, duration_ms, status |
| Capability denied | WARN | extension, capability, method, source_tier |
| Extension handshake completed | INFO | name, negotiated_capabilities, latency_ms |
| Extension shutdown | INFO | name, reason (graceful/timeout/killed), uptime |

### Health Endpoint

Extend `GET /api/observe/health` to include extension status:

```json
{
    "extensions": {
        "loaded": 3,
        "wasm": 1,
        "subprocess": 2,
        "healthy": 3,
        "unhealthy": 0,
        "details": [
            {"name": "content-filter", "tier": "wasm", "state": "active", "uptime": "2h15m"},
            {"name": "pgvector-memory", "tier": "subprocess", "state": "active", "uptime": "2h15m"},
            {"name": "otel-exporter", "tier": "subprocess", "state": "active", "uptime": "2h14m"}
        ]
    }
}
```

---

## Technical Considerations

### Key Decisions

1. **Extism over raw wazero** (ADR-002): Fuel metering and multi-language PDKs justify the dependency. Wrapped behind `Executor` interface for swap-ability.
2. **Capability-scoped security** (ADR-003): Per-extension capability grants enforced at Host API boundary, not just process isolation. Marketplace extensions restricted by default.
3. **Generalized ACP** (ADR-004): Shared subprocess lifecycle avoids code duplication between agents and extensions.
4. **Three-dimensional package model** (ADR-005): Resources + capabilities + actions maps to different security scopes and loading phases.
5. **ExecutorConfig isolation**: WASM-specific config lives in an opaque `ExecutorConfig` on hook declarations, not as inline fields on `HookDecl`. Prevents the declaration struct from becoming a union type.
6. **Dual manifest format**: TOML primary (consistent with AGH config), JSON as fallback (for TypeScript/npm ecosystem). Loader tries `extension.toml` first.

### Known Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Extism goes unmaintained | Low-Medium | Wrapped behind `Executor` interface. Can swap to raw wazero. |
| Host API contract changes break extensions | Medium | Version the protocol. Extensions declare `min_agh_version`. |
| Capability model too restrictive | Medium | Start with permissive defaults (`*` for bundled/user/workspace). Tighten based on real usage. |
| Daemon-context failures (headless WASM hangs) | Medium | Fuel metering + per-call timeout. Subprocess health check + auto-restart with backoff. |
| Binary size increase from Extism/wazero | Low | ~5-8MB is acceptable for a daemon binary. Monitor in CI. |
| TypeScript SDK maintenance burden | Medium | Generate types from Go contracts. Minimize hand-written code. |

### Daemon-Context Failure Modes

Unlike CLI tools, AGH is a headless daemon. Extension failures must be recoverable without user intervention:

| Failure | Detection | Recovery |
|---|---|---|
| WASM hook hangs | Per-call timeout (from Extism) | Timeout error returned to dispatch pipeline. Hook marked unhealthy after N timeouts. |
| WASM fuel exhaustion | Fuel meter exception from Extism | Error returned. Extension logged as fuel-exceeded. |
| WASM module crash | Wasm trap caught by wazero runtime | Error returned. Module can be re-instantiated on next call. |
| Subprocess crash | `waitForExit` goroutine detects exit | Auto-restart with exponential backoff (1s, 2s, 4s, 8s, max 60s). After 5 consecutive failures, disable extension and log ERROR. |
| Subprocess hangs | Health check timeout | SIGTERM → wait 10s → SIGKILL. Restart with backoff. |
| Subprocess Host API abuse | Rate limiting per extension | Return `rate_limited` error. Log WARN. |
| Extension install corruption | Checksum mismatch at load time | Refuse to load. Log ERROR with expected vs actual checksum. |

---

## Architecture Decision Records

- [ADR-001: Three-Tier Extension Model](adrs/adr-001.md) — L1 Go-native, L2 WASM-Extism, L3 subprocess JSON-RPC with tier-appropriate security isolation
- [ADR-002: Extism Go SDK for WASM Runtime](adrs/adr-002.md) — Extism over raw wazero for fuel metering, host functions, and multi-language PDKs
- [ADR-003: Capability-Scoped Security Model](adrs/adr-003.md) — Per-extension capability grants enforced at Host API boundary with source-trust tiers
- [ADR-004: Generalize ACP as Subprocess Extension Protocol](adrs/adr-004.md) — Shared subprocess lifecycle between ACP agents and L3 extensions
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Resources (declarative) + capabilities (interfaces) + actions (Host API)
