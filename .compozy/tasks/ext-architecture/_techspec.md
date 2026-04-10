# TechSpec: Extension Architecture

## Executive Summary

AGH's extension architecture enables third-party developers to extend the daemon's capabilities through a **two-tier execution model**: Go-native interfaces (L1) for first-party compiled-in code, and JSON-RPC subprocess (L3) for all third-party extensions in **Go or TypeScript**. A WASM tier (L2) is designed as a future seam but deferred until hook latency or sandbox requirements justify it.

Extensions are modeled as **three-dimensional packages** that bundle **resources** (agents, skills, hooks, MCP configs), provide **capabilities** (agent drivers, memory backends, observe exporters), and perform **actions** via a bidirectional Host API (create sessions, manage memory, query events).

The architecture builds on AGH's existing infrastructure: the 27-event hook system with typed dispatch provides the extension dispatch layer, and the ACP subprocess pattern provides the L3 prototype. The primary technical trade-off is **power vs. security surface** — a rich bidirectional Host API enables extensions to drive complex workflows (channel adapters, scheduled tasks, memory enrichment) but requires capability-scoped security at the Host API boundary to prevent extensions from exceeding their declared privileges.

Key adjustments from council debate: capability-scoped security at the Host API boundary (not just process isolation), a minimal `Tool` struct to ground the existing hook tool dispatch, and daemon-context failure recovery for headless extension execution.

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
│  │  - Lifecycle mgmt │  │  - Executors:    │  │  - Lifecycle  │ │
│  │  - Capability     │  │    native,       │  │  - Events     │ │
│  │    enforcement    │  │    subprocess,   │  │              │ │
│  │  - Host API       │  │    wasm (stub)   │  │              │ │
│  └────────┬─────────┘  └────────┬─────────┘  └──────┬──────┘ │
│           │                      │                     │        │
│  ┌────────┴──────────────────────┴─────────────────────┴──────┐ │
│  │                   Extension Tiers                           │ │
│  │                                                             │ │
│  │  ┌─────────────┐  ┌─────────────────┐  ┌────────────────┐ │ │
│  │  │ L1: Go      │  │ L2: WASM        │  │ L3: Subprocess │ │ │
│  │  │ Native      │  │ (future seam)   │  │ (JSON-RPC)     │ │ │
│  │  │             │  │                 │  │                │ │ │
│  │  │ Compiled-in │  │ Stub exists.    │  │ Out-of-process │ │ │
│  │  │ interfaces  │  │ Implement when  │  │ bidirectional  │ │ │
│  │  │ [EXISTS]    │  │ needed.         │  │ Host API       │ │ │
│  │  └─────────────┘  └─────────────────┘  └────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Host API (bidirectional)                │   │
│  │  sessions/* │ memory/* │ skills/* │ observe/*             │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Install**: `agh extension install <path>` → parse manifest → validate capabilities → copy resources → register in extension registry
2. **Boot**: Daemon starts → Extension Manager loads enabled extensions → launch subprocesses → capability negotiation handshake
3. **Runtime**: Hook dispatched → Extension Manager routes to subprocess executor → capability check → execute → return result
4. **Host API call**: Extension sends JSON-RPC request → capability check → execute on daemon → return result
5. **Shutdown**: Daemon stops → Extension Manager sends shutdown to all subprocesses → wait with timeout → SIGKILL stragglers

---

## Implementation Design

### Core Interfaces

**Extension Manager** — the orchestrator that lives in `internal/extension/`:

```go
// internal/extension/manager.go
type Manager struct {
    mu          sync.RWMutex
    registry    *Registry
    subprocesses map[string]*subprocess.Process
    capChecker  *CapabilityChecker
    hostAPI     *HostAPIHandler
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
    Name         string             `toml:"name"`
    Version      string             `toml:"version"`
    Description  string             `toml:"description"`
    MinAGH       string             `toml:"min_agh_version"`
    Resources    ResourcesConfig    `toml:"resources"`
    Capabilities CapabilitiesConfig `toml:"capabilities"`
    Actions      ActionsConfig      `toml:"actions"`
    Subprocess   SubprocessConfig   `toml:"subprocess"`
    Security     SecurityConfig     `toml:"security"`
}
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
    grants map[string]CapabilityGrant
    tiers  map[ExtensionSource][]string
}

func (c *CapabilityChecker) Check(extName, capability string) error
func (c *CapabilityChecker) CheckHostAPI(extName, method string) error
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
min_agh_version = "0.5.0"

[resources]
skills = ["skills/"]
agents = ["agents/"]
hooks = []
mcp_servers = []

[capabilities]
provides = ["memory.backend"]

[actions]
requires = [
    "sessions/list",
    "sessions/events",
    "memory/store",
    "memory/recall",
]

[subprocess]
command = "agh-ext-pgvector"
args = ["--config", "{{config_dir}}/pgvector.toml"]
health_check_interval = "30s"
shutdown_timeout = "10s"

[subprocess.env]
PGVECTOR_URL = "{{env:PGVECTOR_URL}}"

[security]
capabilities = [
    "memory.read",
    "memory.write",
    "session.read",
]
```

**Extension Manifest** (JSON alternative):

```json
{
    "extension": {
        "name": "pgvector-memory",
        "version": "0.2.1",
        "description": "PostgreSQL pgvector memory backend for AGH",
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
        "requires": ["sessions/list", "sessions/events", "memory/store", "memory/recall"]
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

### Host API (Subprocess Extensions ↔ AGH)

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
| `observe/events` | `{session_id?, type?, since?, limit?}` | `[{type, timestamp, data}]` | `observe.read` |

**Skills Methods:**

| Method | Params | Result | Capability |
|---|---|---|---|
| `skills/list` | `{workspace?}` | `[{name, description, source}]` | `skills.read` |

**AGH → Extension Methods:**

| Method | Description |
|---|---|
| `initialize` | Capability negotiation handshake |
| `execute_hook` | Dispatch a hook event to the extension |
| `provide_tools` | Request tool definitions from extension |
| `health_check` | Liveness probe |
| `shutdown` | Graceful shutdown request |

### Extension Loading Pipeline

Six-phase pipeline (inspired by OpenClaw, validated across 5/6 harnesses):

```
1. DISCOVER    → Scan extension directories, find manifests
2. PARSE       → Read extension.toml/json, validate schema (no code execution)
3. VALIDATE    → Check version compatibility, verify checksums, validate capabilities
4. REGISTER    → Copy resources (skills, agents, hooks) into AGH registries
5. INITIALIZE  → Launch subprocesses, perform handshake, negotiate capabilities
6. ACTIVATE    → Extension is live, hooks dispatch to it, Host API available
```

Each phase can fail independently with clear error messages.

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

Extension subprocess hooks use the existing `SubprocessExecutor` — no new executor kind needed. The extension manifest declares hooks with `executor.kind = "subprocess"` pointing to the extension binary.

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
| `internal/acp/client.go` | Modified | Extract subprocess lifecycle into shared package | Medium risk — refactor existing working code |
| `internal/daemon/boot.go` | Modified | Add Extension Manager initialization phase | Low risk — additive phase in boot sequence |
| `internal/daemon/hooks_bridge.go` | Modified | Add extension declaration provider | Low risk — extends existing patterns |
| `internal/extension/` | New | Extension Manager, Registry, manifest loading, capability enforcement, Host API | New package — core of this techspec |
| `internal/subprocess/` | New | Shared subprocess lifecycle extracted from ACP | New package — refactor, not new functionality |
| `internal/tools/` | New | Minimal Tool struct and ToolProvider interface | New package — ~200 LOC |
| `internal/store/globaldb/` | Modified | Add extension registry table | Low risk — new table, no schema changes |
| `internal/cli/` | Modified | Add `agh extension list/install/enable/disable` commands | Additive CLI commands |

AGH's ACP layer uses `coder/acp-go-sdk` for JSON-RPC framing (ACP-specific). The extension protocol needs its own JSON-RPC framing — evaluate `sourcegraph/jsonrpc2` or a lightweight custom implementation. Only subprocess lifecycle (spawn, signals, health) is extractable from ACP.

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
    actions: { requires: ['sessions/list'] },
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

- **Extension Manager**: Mock subprocess launcher. Test lifecycle (start, stop, restart). Test capability enforcement (authorized vs unauthorized calls).
- **Manifest Parser**: Table-driven tests for TOML and JSON manifests. Test validation (missing fields, invalid versions, unknown capabilities). Test both formats produce identical `Manifest` structs.
- **Capability Checker**: Test all source-trust tier combinations. Test wildcard grants. Test unauthorized access returns typed errors.
- **Host API Handler**: Test each method with authorized and unauthorized callers. Test parameter validation. Test error responses.
- **Tool struct**: Test `ToolProvider` interface. Test tool serialization matches hook `ToolCallRef` payloads.
- **Subprocess lifecycle**: Test launch, handshake, health check, graceful shutdown, crash recovery.

### Integration Tests

- **End-to-end subprocess extension**: Install a test subprocess extension → daemon boots → handshake completes → extension calls Host API → verify results.
- **Extension lifecycle**: Install → enable → daemon restart → verify extension reloads → disable → verify extension stops → uninstall → verify cleanup.
- **Capability enforcement**: Install extension with limited capabilities → attempt unauthorized Host API call → verify rejection with typed error.
- **Resource registration**: Install extension with skills and agents → verify they appear in skills registry and agent definitions.
- **Host API bidirectional**: Extension creates session via Host API → session runs → extension reads events back.

---

## Development Sequencing

### Build Order

1. **`internal/tools/` — Minimal Tool struct + ToolProvider** — no dependencies. ~200 LOC. Grounds the hook tool dispatch that already exists.

2. **`internal/subprocess/` — Shared subprocess lifecycle** — no dependencies on step 1. Extract from `internal/acp/client.go`: process launch, JSON-RPC framing, handshake, graceful shutdown, health monitoring.

3. **`internal/extension/manifest.go` — Manifest parser** — no dependencies on prior steps. Parse `extension.toml` and `extension.json`. Validate schema. Produce `Manifest` struct.

4. **`internal/extension/capability.go` — Capability checker** — depends on step 3 (reads capabilities from manifest). Source-trust tier enforcement. Dispatch-time and Host API checks.

5. **`internal/extension/registry.go` — Extension registry** — depends on steps 3, 4. SQLite table in global DB. CRUD operations. Enabled/disabled state.

6. **`internal/extension/manager.go` — Extension Manager** — depends on steps 2, 4, 5. Orchestrates lifecycle: discover → parse → validate → register → initialize → activate.

7. **`internal/extension/host_api.go` — Host API handler** — depends on steps 4, 6. JSON-RPC method handlers for sessions/\*, memory/\*, observe/\*, skills/\*. Capability-checked.

8. **`internal/daemon/boot.go` — Daemon integration** — depends on step 6. Add Extension Manager phase to boot sequence. Wire declaration provider.

9. **`internal/cli/extension.go` — CLI commands** — depends on steps 5, 6. `agh extension list`, `install`, `enable`, `disable`.

10. **`@agh/extension-sdk` — TypeScript SDK** — depends on steps 6, 7. npm package with Extension class, StdioTransport, Host API client, test harness.

11. **Reference extensions** — depends on steps 6, 7. Two working examples: one Go subprocess extension, one TypeScript subprocess extension.

### Technical Dependencies

- **JSON-RPC library**: ACP uses `coder/acp-go-sdk` (ACP-specific). Extension subprocess protocol needs its own framing — evaluate `sourcegraph/jsonrpc2` (419 importers, MIT, bidirectional over any `io.ReadWriteCloser`) or lightweight custom implementation.
- **Node.js 18+**: Required for TypeScript SDK development and testing.

---

## Monitoring and Observability

### Key Metrics

- `agh_extensions_loaded{name, state}` — Gauge of loaded extensions by state
- `agh_extension_hook_duration_ms{extension, event}` — Histogram of hook execution time
- `agh_extension_host_api_calls{extension, method, status}` — Counter of Host API calls
- `agh_extension_subprocess_restarts{extension}` — Counter of subprocess restart events
- `agh_extension_capability_denied{extension, capability}` — Counter of denied capability checks

### Log Events

| Event | Level | Fields |
|---|---|---|
| Extension loaded | INFO | name, version, capabilities |
| Extension failed to load | ERROR | name, error, phase (discover/parse/validate/initialize) |
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
        "loaded": 2,
        "healthy": 2,
        "unhealthy": 0,
        "details": [
            {"name": "pgvector-memory", "state": "active", "uptime": "2h15m", "pid": 42891},
            {"name": "otel-exporter", "state": "active", "uptime": "2h14m", "pid": 42903}
        ]
    }
}
```

---

## Technical Considerations

### Key Decisions

1. **Two-tier model now, WASM later** (ADR-001): L1 Go-native + L3 subprocess covers Go and TypeScript. WASM stub remains as future seam — implement when hook latency is a measured bottleneck or sandbox is required for marketplace extensions.
2. **Capability-scoped security** (ADR-003): Per-extension capability grants enforced at Host API boundary, not just process isolation. Marketplace extensions restricted by default.
3. **Generalized ACP** (ADR-004): Shared subprocess lifecycle avoids code duplication between agents and extensions.
4. **Three-dimensional package model** (ADR-005): Resources + capabilities + actions maps to different security scopes and loading phases.
5. **Dual manifest format**: TOML primary (consistent with AGH config), JSON as fallback (for TypeScript/npm ecosystem). Loader tries `extension.toml` first.
6. **Minimal new Go dependencies**: Subprocess lifecycle extracted from ACP. JSON-RPC framing requires evaluation of `sourcegraph/jsonrpc2` or custom implementation since ACP uses the ACP-specific `coder/acp-go-sdk`.

### Known Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Host API contract changes break extensions | Medium | Version the protocol. Extensions declare `min_agh_version`. |
| Capability model too restrictive | Medium | Start with permissive defaults (`*` for bundled/user/workspace). Tighten based on real usage. |
| Subprocess hook latency accumulates | Low | Most hooks are async. Sync hooks can run in parallel. WASM seam exists for future optimization. |
| TypeScript SDK maintenance burden | Medium | Generate types from Go contracts. Minimize hand-written code. |
| ACP refactor breaks existing functionality | Medium | Extract incrementally. ACP integration tests must pass at every step. |

### Daemon-Context Failure Modes

Unlike CLI tools, AGH is a headless daemon. Extension failures must be recoverable without user intervention:

| Failure | Detection | Recovery |
|---|---|---|
| Subprocess crash | `waitForExit` goroutine detects exit | Auto-restart with exponential backoff (1s, 2s, 4s, 8s, max 60s). After 5 consecutive failures, disable extension and log ERROR. |
| Subprocess hangs | Health check timeout | SIGTERM → wait 10s → SIGKILL. Restart with backoff. |
| Subprocess Host API abuse | Rate limiting per extension | Return `rate_limited` error. Log WARN. |
| Extension install corruption | Checksum mismatch at load time | Refuse to load. Log ERROR with expected vs actual checksum. |
| Handshake failure | Timeout during initialize | Extension not activated. Log ERROR. Retry on next daemon boot. |

### Future Seams (Documented, Not Implemented)

| Seam | Trigger to Implement | Integration Point |
|---|---|---|
| **L2 WASM tier** (Extism) | Measured hook latency bottleneck or marketplace sandbox requirement | `internal/hooks/executor_wasm_stub.go` — fill existing stub |
| **Tool Registry** (BM25, namespacing) | Extension authors need tool registration | `internal/tools/` — extend minimal Tool struct |
| **Channel adapters** | Demand for Slack/Discord/Telegram integration | Extension capability + Host API `sessions/create` |
| **Cron scheduler** | Demand for scheduled agent runs | New `internal/cron/` package, exposed as extension capability |
| **API route extensions** | Extensions need custom HTTP endpoints | Dynamic route registration in Gin |
| **CLI command extensions** | Extensions need custom `agh` subcommands | Dynamic Cobra command registration |
| **Extension marketplace** | Ecosystem grows enough to need discovery | GitHub-based registry with checksums |

---

## Architecture Decision Records

- [ADR-001: Two-Tier Extension Model with Future WASM Seam](adrs/adr-001.md) — L1 Go-native + L3 subprocess now; L2 WASM deferred until measured need
- [ADR-002: Extism Go SDK for WASM Runtime (Deferred)](adrs/adr-002.md) — Extism chosen for future WASM tier; deferred until hook latency or sandbox justifies it
- [ADR-003: Capability-Scoped Security Model](adrs/adr-003.md) — Per-extension capability grants enforced at Host API boundary with source-trust tiers
- [ADR-004: Generalize ACP as Subprocess Extension Protocol](adrs/adr-004.md) — Shared subprocess lifecycle between ACP agents and extensions
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Resources (declarative) + capabilities (interfaces) + actions (Host API)
