# AGH Extensibility System: Deep Research Analysis

**Date:** 2026-04-09
**Scope:** Multi-language plugin/extension architecture for AGH Agent OS
**Status:** Research complete

---

## Executive Summary

AGH needs an extension system that lets third-party developers (who may not know Go) create plugins that extend the daemon's capabilities -- from custom agent drivers and memory backends to new API endpoints and session lifecycle hooks. After analyzing the four dominant approaches (Go native plugins, HashiCorp-style gRPC subprocess plugins, WebAssembly sandbox plugins, and JSON-RPC stdio plugins), this research recommends a **layered hybrid architecture**:

1. **Primary layer: JSON-RPC over stdio** for agent-class extensions (already proven by AGH's ACP driver and aligned with MCP/LSP ecosystems)
2. **Secondary layer: WebAssembly (Extism/wazero)** for sandboxed, lightweight, in-process extensions (hooks, transformers, validators)
3. **Go-native interfaces** for first-party and high-performance extensions compiled into the binary

This hybrid gives AGH the broadest language support (any language for JSON-RPC; Rust/Go/TypeScript/C for Wasm; Go for native), strong security isolation at every tier, and a developer experience that meets extension authors where they are. The architecture aligns with AGH's existing subprocess-based ACP pattern, avoids introducing external dependencies like NATS or Kafka, and keeps the daemon as a single binary.

Key finding: The industry has converged on **two winning patterns** for 2025-2026 agent extensibility -- JSON-RPC stdio (MCP/LSP lineage) for rich, stateful, out-of-process extensions, and WebAssembly (WASI Preview 2 + Component Model) for sandboxed, portable, in-process plugins. AGH should adopt both rather than choosing one, because they serve complementary use cases.

---

## Approach Comparison Matrix

| Dimension | Go Native Plugins | HashiCorp go-plugin (gRPC) | WebAssembly (Extism/wazero) | JSON-RPC over stdio |
|---|---|---|---|---|
| **Multi-language support** | Go only | Any language via gRPC (practical: Go, Python, Ruby) | 16+ languages compile to Wasm (Rust, Go, C, TS via AssemblyScript, JS via Javy) | Any language with JSON + stdin/stdout |
| **Performance** | Fastest (in-process, shared memory) | ~50-100us per RPC call over UDS | ~1-10us per call (in-process Wasm sandbox) | ~100-500us per call (JSON serialize + IPC) |
| **Developer experience** | Poor for non-Go devs; strict build env matching | Good Go SDK; gRPC boilerplate for other langs; protobuf required | Good (Extism PDKs for 7+ langs); single `.wasm` binary output | Excellent (any lang, simple JSON protocol, stdin/stdout) |
| **Security isolation** | None (shared process memory) | Strong (OS process isolation + optional TLS + checksums) | Very strong (Wasm linear memory sandbox, capability-based, deny-by-default) | Strong (OS process isolation) |
| **Crash safety** | Plugin crash kills host | Plugin crash isolated to subprocess | Wasm trap handled by runtime; host unaffected | Plugin crash isolated to subprocess |
| **Maturity** | Experimental (no Windows, no unloading, CGO required) | Battle-tested since 2012 (Terraform, Vault, Consul, Packer) | Maturing rapidly (Extism v1 GA 2024; WASI 0.2 stable; wazero 1.x stable) | Proven (LSP since 2016; MCP since 2024; 10+ years of JSON-RPC) |
| **Plugin distribution** | Platform-specific binaries; exact Go version match | Platform-specific binaries | Single portable `.wasm` file (runs anywhere) | Platform-specific binaries or scripts |
| **Binary size impact** | Minimal (shared libs) | Minimal (plugins are separate binaries) | ~5-10MB for wazero runtime in host binary | Minimal |
| **Bidirectional calls** | N/A (same process) | Yes (gRPC streams) | Yes (host functions) | Yes (JSON-RPC notifications) |
| **Hot reload** | Impossible (no plugin unloading) | Restart subprocess | Re-instantiate Wasm module (milliseconds) | Restart subprocess |
| **AGH alignment** | Low (contradicts single-binary goal) | Medium (proven but heavy for simple extensions) | High (in-process, portable, sandboxed) | Highest (already using JSON-RPC stdio for ACP) |

### Verdict

- **Go native plugins: Reject.** Platform limitations (no Windows), mandatory CGO, impossible hot-reload, no security isolation, and exact build-environment coupling make this unsuitable for a third-party extension ecosystem. Every major Go project (Terraform, Vault, Grafana, Consul) has moved away from or never adopted native plugins.

- **HashiCorp go-plugin (gRPC): Use as reference, not as dependency.** The architecture is sound and battle-tested, but adopting it directly adds a dependency on protobuf toolchains and gRPC machinery that may be heavier than needed. AGH already has JSON-RPC stdio infrastructure via ACP. The gRPC subprocess model is worth emulating for heavyweight extensions, but JSON-RPC stdio achieves the same isolation with a simpler protocol already present in the codebase.

- **WebAssembly (Extism + wazero): Adopt for in-process sandboxed extensions.** The zero-CGO wazero runtime fits AGH's single-binary constraint perfectly. Extism provides the highest-level SDK with 16+ host languages and 7+ guest PDKs. Wasm's deny-by-default security model is ideal for running untrusted third-party code. Use this for fast-path, synchronous extension points (hooks, transformers, validators, custom functions).

- **JSON-RPC over stdio: Adopt as the primary extension protocol.** AGH already speaks JSON-RPC stdio with ACP agents. This is the same pattern used by LSP (language servers) and MCP (model context protocol). It provides excellent multi-language support (literally any language), process isolation, and aligns with the broader AI agent ecosystem. Use this for stateful, long-running, rich extensions (custom agent drivers, memory backends, API extensions).

---

## Recommended Architecture

### Three-Tier Extension Model

```
+------------------------------------------------------------------+
|                        AGH Daemon (Go)                           |
|                                                                  |
|  +------------------+  +-------------------+  +----------------+ |
|  | Go-Native Layer  |  | Wasm Sandbox Layer|  | Subprocess     | |
|  | (compiled-in)    |  | (in-process)      |  | Layer          | |
|  |                  |  |                   |  | (out-of-process)| |
|  | - Core providers |  | - Hook handlers   |  |                | |
|  | - Built-in agents|  | - Validators      |  | - Agent drivers| |
|  | - Store backends |  | - Transformers    |  | - Memory backends|
|  |                  |  | - Custom functions |  | - API extensions||
|  | Interface:       |  |                   |  |                | |
|  | Go interfaces    |  | Interface:        |  | Interface:     | |
|  | (compile-time)   |  | Extism Host SDK   |  | JSON-RPC stdio | |
|  |                  |  | (WIT contracts)   |  | (protocol spec)| |
|  +------------------+  +-------------------+  +----------------+ |
+------------------------------------------------------------------+
```

### Layer 1: Go-Native Extensions (Compiled-In)

**Purpose:** First-party, high-performance extensions that ship with the AGH binary.

**When to use:** Core functionality, bundled agent support, default store implementations. Only for code authored by the AGH team or trusted contributors whose code is reviewed and merged.

**Pattern:** Standard Go interfaces with compile-time verification:
```go
// Defined in consuming package (Go-style)
type AgentDriver interface {
    Start(ctx context.Context, cfg AgentConfig) (*AgentProcess, error)
    Prompt(ctx context.Context, msg PromptMessage) (<-chan Event, error)
    Stop(ctx context.Context) error
}

// Compile-time check
var _ AgentDriver = (*ACPDriver)(nil)
```

**This is what AGH already does** with `internal/acp`, `internal/session`, `internal/store`. No changes needed -- just formalize the interfaces as the extension contract that external tiers must also satisfy.

### Layer 2: WebAssembly Sandbox Extensions (In-Process)

**Purpose:** Lightweight, sandboxed, portable extensions for synchronous operations.

**When to use:** Event hooks (pre/post session creation, message filtering), content validators, data transformers, custom functions, skill preprocessors. Anywhere you need fast (<1ms), safe, portable code execution without granting filesystem/network access.

**Runtime:** wazero (pure Go, zero CGO, zero dependencies) via the Extism Go Host SDK.

**Contract definition:** WIT (WebAssembly Interface Types) files define the extension points:

```wit
// extension-hooks.wit
package agh:hooks@0.1.0;

interface session-hooks {
    record session-context {
        session-id: string,
        agent-name: string,
        workspace: string,
    }

    record hook-result {
        allow: bool,
        message: option<string>,
        modified-context: option<session-context>,
    }

    on-session-creating: func(ctx: session-context) -> hook-result;
    on-session-created: func(ctx: session-context) -> hook-result;
    on-message-received: func(ctx: session-context, content: string) -> hook-result;
}
```

**Security model:**
- Wasm linear memory isolation (plugin cannot access host memory)
- No filesystem access by default (grant via explicit capability)
- No network access by default (grant via host functions)
- CPU/memory limits via wazero runtime configuration
- Deterministic execution (no threads, no random unless granted)

**Plugin distribution:** Single `.wasm` file, portable across OS/arch. Can be stored in a registry, downloaded by `agh plugin install`, and loaded at runtime.

### Layer 3: Subprocess Extensions (Out-of-Process)

**Purpose:** Rich, stateful, long-running extensions that need full system access.

**When to use:** Custom agent drivers (wrapping new AI CLIs), custom memory/knowledge backends (connecting to vector DBs, graph DBs), API extensions (adding new HTTP endpoints), complex integrations requiring network/filesystem access.

**Protocol:** JSON-RPC 2.0 over stdio (stdin/stdout), with an MCP-inspired capability negotiation handshake.

**Lifecycle:**
1. AGH daemon launches extension binary as subprocess
2. Extension writes a handshake line to stdout: `{"jsonrpc":"2.0","method":"initialize","params":{...}}`
3. AGH responds with capabilities it supports
4. Normal JSON-RPC message exchange begins
5. AGH sends `shutdown` request; extension exits cleanly
6. If extension crashes, AGH logs the failure and optionally restarts

**This mirrors exactly how AGH already talks to ACP agents.** The extension protocol is a generalization of the ACP client pattern.

### Extension Point Catalog

| Extension Point | Layer | Direction | Description |
|---|---|---|---|
| `agent.driver` | Subprocess (L3) | Bidirectional | Custom agent driver (like ACP but for non-ACP agents) |
| `memory.backend` | Subprocess (L3) | Request/Response | Custom memory storage (vector DB, graph DB) |
| `api.route` | Subprocess (L3) | Request/Response | Add HTTP/SSE routes to the daemon API |
| `session.hook.pre_create` | Wasm (L2) | Sync call | Validate/modify session before creation |
| `session.hook.post_create` | Wasm (L2) | Sync call | React to session creation |
| `session.hook.pre_prompt` | Wasm (L2) | Sync call | Filter/transform prompt before sending to agent |
| `session.hook.post_event` | Wasm (L2) | Sync call | Transform/filter events before persistence |
| `message.validator` | Wasm (L2) | Sync call | Content safety, policy enforcement |
| `message.transformer` | Wasm (L2) | Sync call | Content rewriting, enrichment |
| `skill.preprocessor` | Wasm (L2) | Sync call | Transform skill content before injection |
| `observe.exporter` | Subprocess (L3) | Push | Export metrics/events to external systems |
| `config.provider` | Go-native (L1) | Sync call | Custom config sources (compiled in) |

---

## Go-Specific Implementation Patterns

### Extension Manager (Composition Root Integration)

The extension system should be wired through `internal/daemon` as the composition root, following AGH's existing architectural principle that `daemon/` is the sole place that imports all other packages.

```go
// internal/extension/manager.go
package extension

import (
    "context"
    "log/slog"
    "sync"
)

// Manager orchestrates extension lifecycle across all tiers.
type Manager struct {
    mu          sync.RWMutex
    wasmRuntime *WasmRuntime     // Layer 2: in-process Wasm
    subprocesses map[string]*Subprocess // Layer 3: out-of-process
    registry    *Registry
    logger      *slog.Logger
}

type Option func(*Manager)

func WithLogger(logger *slog.Logger) Option {
    return func(m *Manager) { m.logger = logger }
}

func NewManager(registry *Registry, opts ...Option) *Manager {
    m := &Manager{
        subprocesses: make(map[string]*Subprocess),
        registry:     registry,
        logger:       slog.Default(),
    }
    for _, opt := range opts {
        opt(m)
    }
    return m
}

// Start initializes all configured extensions.
func (m *Manager) Start(ctx context.Context) error {
    // 1. Initialize wazero runtime
    // 2. Load Wasm extensions from registry
    // 3. Launch subprocess extensions
    // 4. Run capability negotiation handshakes
    return nil
}

// Stop gracefully shuts down all extensions.
func (m *Manager) Stop(ctx context.Context) error {
    // 1. Send shutdown to all subprocesses
    // 2. Wait with timeout
    // 3. SIGKILL stragglers
    // 4. Close wazero runtime
    return nil
}
```

### Wasm Runtime Integration (Layer 2)

Using Extism Go SDK with wazero:

```go
// internal/extension/wasm_runtime.go
package extension

import (
    "context"
    "fmt"
    "sync"

    extism "github.com/extism/go-sdk"
)

// WasmRuntime manages compiled Wasm plugins.
type WasmRuntime struct {
    mu      sync.RWMutex
    plugins map[string]*extism.Plugin
}

// LoadPlugin compiles and instantiates a Wasm extension.
func (wr *WasmRuntime) LoadPlugin(name string, wasmBytes []byte, config PluginConfig) error {
    manifest := extism.Manifest{
        Wasm: []extism.Wasm{
            extism.WasmData{Data: wasmBytes},
        },
    }

    // Configure capabilities (deny-by-default)
    pluginConfig := extism.PluginConfig{
        EnableWasi: config.EnableWASI,
    }

    plugin, err := extism.NewPlugin(context.Background(), manifest, pluginConfig, nil)
    if err != nil {
        return fmt.Errorf("extension: compile wasm plugin %q: %w", name, err)
    }

    wr.mu.Lock()
    wr.plugins[name] = plugin
    wr.mu.Unlock()
    return nil
}

// CallHook invokes a named function on a Wasm plugin.
func (wr *WasmRuntime) CallHook(name, function string, input []byte) ([]byte, error) {
    wr.mu.RLock()
    plugin, ok := wr.plugins[name]
    wr.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("extension: wasm plugin %q not found", name)
    }

    _, output, err := plugin.Call(function, input)
    if err != nil {
        return nil, fmt.Errorf("extension: call %s.%s: %w", name, function, err)
    }
    return output, nil
}
```

### Subprocess Extension Protocol (Layer 3)

Building on AGH's existing ACP driver pattern:

```go
// internal/extension/subprocess.go
package extension

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "sync"
)

// Subprocess manages a single out-of-process extension.
type Subprocess struct {
    cmd    *exec.Cmd
    stdin  *json.Encoder
    stdout *bufio.Scanner
    mu     sync.Mutex
    caps   Capabilities
}

// Capabilities declared by an extension during handshake.
type Capabilities struct {
    ExtensionPoints []string `json:"extension_points"`
    Version         string   `json:"version"`
    Name            string   `json:"name"`
}

// Start launches the extension subprocess and performs handshake.
func (s *Subprocess) Start(ctx context.Context, binary string, args []string) error {
    s.cmd = exec.CommandContext(ctx, binary, args...)
    // Wire stdin/stdout for JSON-RPC
    // Perform initialize handshake
    // Validate capabilities
    return nil
}
```

### Hook Chain Pattern

AGH's existing `HookRunner` pattern should be extended to support both subprocess hooks (existing) and Wasm hooks (new):

```go
// internal/extension/hook_chain.go
package extension

import "context"

// HookChain dispatches hooks across both Wasm and subprocess extensions.
type HookChain struct {
    wasmRuntime  *WasmRuntime
    subprocesses map[string]*Subprocess
}

// RunPreSessionCreate runs all registered pre-session-create hooks.
// Wasm hooks run first (fast, synchronous), then subprocess hooks.
// Any hook returning allow=false short-circuits the chain.
func (hc *HookChain) RunPreSessionCreate(ctx context.Context, session SessionContext) (HookResult, error) {
    // 1. Run Wasm hooks (fast path, <1ms each)
    // 2. If any deny, return immediately
    // 3. Run subprocess hooks (slower, parallel with timeout)
    // 4. Merge results
    return HookResult{Allow: true}, nil
}
```

### Interface-Based Extension Points

Following AGH's principle of "interfaces defined where consumed":

```go
// internal/session/driver.go (already exists conceptually)
// The AgentDriver interface is the extension contract for Layer 3 agent extensions.
// Any subprocess that implements JSON-RPC methods matching this interface
// can serve as an agent driver.

// internal/memory/backend.go (extension contract)
type Backend interface {
    Store(ctx context.Context, key string, entry MemoryEntry) error
    Recall(ctx context.Context, query RecallQuery) ([]MemoryEntry, error)
    Forget(ctx context.Context, key string) error
}

// internal/observe/exporter.go (extension contract)
type Exporter interface {
    Export(ctx context.Context, events []Event) error
    Flush(ctx context.Context) error
}
```

### Extension Manifest

Every extension (Wasm or subprocess) is described by a manifest file:

```toml
# extension.toml
[extension]
name = "custom-memory-pgvector"
version = "0.2.1"
description = "PostgreSQL pgvector memory backend for AGH"
type = "subprocess"       # "subprocess" | "wasm"
min_agh_version = "0.3.0"

[extension.subprocess]
command = "agh-ext-pgvector"
args = ["--config", "{{config_dir}}/pgvector.toml"]

[extension.capabilities]
extension_points = ["memory.backend"]

[extension.permissions]
network = true
filesystem = false
```

---

## TypeScript Extension SDK Design

### Strategy: Two Paths for TypeScript Authors

TypeScript extension authors should have two pathways depending on their use case:

**Path A: JSON-RPC Subprocess Extensions (full power)**
For rich, stateful extensions that need filesystem/network access. The TypeScript extension runs as a Node.js process, communicating with AGH over stdin/stdout JSON-RPC.

**Path B: Wasm Extensions (sandboxed, portable)**
For lightweight hooks and transformers. TypeScript is compiled to Wasm via AssemblyScript or Javy (QuickJS-in-Wasm), producing a portable `.wasm` binary.

### Path A: TypeScript JSON-RPC SDK

```typescript
// @agh/extension-sdk (npm package)

import { Extension, ExtensionContext } from '@agh/extension-sdk';

// Define a memory backend extension
const ext = new Extension({
  name: 'pgvector-memory',
  version: '0.2.1',
  extensionPoints: ['memory.backend'],
});

// Register handlers for the memory.backend contract
ext.handle('memory/store', async (ctx: ExtensionContext, params: StoreParams) => {
  // Store to pgvector
  await pgPool.query('INSERT INTO memories ...', [params.key, params.embedding]);
  return { success: true };
});

ext.handle('memory/recall', async (ctx: ExtensionContext, params: RecallParams) => {
  const rows = await pgPool.query('SELECT * FROM memories WHERE ...', [params.query]);
  return { entries: rows.map(toMemoryEntry) };
});

// Start the extension (reads stdin, writes stdout)
ext.start();
```

**SDK internals:**
```typescript
// @agh/extension-sdk/src/extension.ts

export class Extension {
  private handlers = new Map<string, Handler>();
  private transport: StdioTransport;

  constructor(private manifest: ExtensionManifest) {
    this.transport = new StdioTransport();
  }

  handle(method: string, handler: Handler): void {
    this.handlers.set(method, handler);
  }

  async start(): Promise<void> {
    // 1. Perform JSON-RPC initialize handshake
    await this.transport.sendRequest('initialize', {
      name: this.manifest.name,
      version: this.manifest.version,
      extensionPoints: this.manifest.extensionPoints,
    });

    // 2. Listen for incoming JSON-RPC requests
    for await (const message of this.transport.messages()) {
      if (message.method && this.handlers.has(message.method)) {
        const result = await this.handlers.get(message.method)!(
          this.createContext(message),
          message.params,
        );
        await this.transport.sendResponse(message.id, result);
      }
    }
  }
}
```

**SDK structure:**
```
@agh/extension-sdk/
  src/
    extension.ts          # Main Extension class
    transport.ts          # StdioTransport (JSON-RPC over stdin/stdout)
    types.ts              # Generated types from AGH extension contracts
    contracts/
      memory-backend.ts   # Type definitions for memory.backend
      agent-driver.ts     # Type definitions for agent.driver
      session-hooks.ts    # Type definitions for session hooks
    testing/
      mock-transport.ts   # In-memory transport for unit tests
      harness.ts          # Test harness that simulates AGH host
  bin/
    create-extension.ts   # CLI scaffolding tool
  templates/
    memory-backend/       # Starter template
    agent-driver/         # Starter template
    hook-extension/       # Starter template
```

### Path B: TypeScript-to-Wasm SDK (AssemblyScript)

For lightweight, sandboxed extensions:

```typescript
// Written in AssemblyScript (TypeScript-like syntax that compiles to Wasm)
// @agh/wasm-pdk (AssemblyScript PDK)

import { Host, JSON } from '@agh/wasm-pdk';

export function on_session_creating(): i32 {
  const input = Host.inputString();
  const ctx = JSON.parse<SessionContext>(input);

  // Validate session
  if (ctx.agentName === 'blocked-agent') {
    const result: HookResult = {
      allow: false,
      message: 'This agent is not permitted',
    };
    Host.outputString(JSON.stringify(result));
    return 0;
  }

  Host.outputString(JSON.stringify({ allow: true }));
  return 0;
}
```

Compile: `asc src/index.ts --target release --outFile dist/hook.wasm`

### Contract Generation Pipeline

AGH extension contracts should be defined once and generated for multiple languages:

```
extension-contracts/
  proto/
    memory_backend.proto     # Protobuf definitions
    agent_driver.proto
    session_hooks.proto
  wit/
    memory-backend.wit       # WIT definitions (for Wasm)
    session-hooks.wit
  generated/
    go/                      # Generated Go types
    typescript/              # Generated TypeScript types
    rust/                    # Generated Rust types (for Wasm PDK)
```

**Why both Protobuf and WIT?**
- Protobuf: For JSON-RPC subprocess extensions. Used to generate typed request/response structures. The actual wire format is JSON (not protobuf binary) -- protobuf serves as the schema definition language.
- WIT: For Wasm extensions. WIT is the native interface description language for the WebAssembly Component Model.

Both are generated from a single source of truth (the `.proto` files), with a custom codegen step that produces WIT from proto.

### Developer Experience Priorities

1. **`agh extension init`** -- Scaffolding CLI that generates a complete extension project with the right SDK, manifest, and tests
2. **`agh extension dev`** -- Development mode that watches for changes, rebuilds, and hot-reloads the extension into a running AGH daemon
3. **`agh extension test`** -- Runs extension tests using a mock AGH host (provided by the SDK)
4. **`agh extension build`** -- Produces the distributable artifact (binary for subprocess, `.wasm` for Wasm)
5. **`agh extension publish`** -- Publishes to the AGH extension registry

---

## Distribution and Registry

### Extension Registry Architecture

```
+------------------+     +------------------+     +------------------+
|  Extension Dev   |     |  AGH Registry    |     |   AGH Daemon     |
|                  |     |  (GitHub-based)  |     |                  |
|  agh ext publish |---->|  registry.json   |---->|  agh ext install |
|                  |     |  + GitHub Releases|     |                  |
+------------------+     +------------------+     +------------------+
```

### Registry Design (GitHub-Native)

Following the pattern proven by Terraform Registry and Homebrew:

1. **Registry Index**: A Git repository containing `registry.json` with metadata for all published extensions
2. **Extension Artifacts**: Stored as GitHub Releases on the extension author's repository
3. **Manifest Verification**: SHA-256 checksums in the registry index, verified at install time

```json
{
  "extensions": {
    "pgvector-memory": {
      "name": "pgvector-memory",
      "description": "PostgreSQL pgvector memory backend",
      "author": "agh-community",
      "repository": "github.com/agh-community/ext-pgvector-memory",
      "type": "subprocess",
      "extension_points": ["memory.backend"],
      "versions": {
        "0.2.1": {
          "min_agh_version": "0.3.0",
          "artifacts": {
            "darwin-arm64": {
              "url": "https://github.com/.../releases/download/v0.2.1/ext-pgvector-memory-darwin-arm64.tar.gz",
              "sha256": "a1b2c3..."
            },
            "linux-amd64": {
              "url": "https://github.com/.../releases/download/v0.2.1/ext-pgvector-memory-linux-amd64.tar.gz",
              "sha256": "d4e5f6..."
            },
            "wasm": {
              "url": "https://github.com/.../releases/download/v0.2.1/ext-pgvector-memory.wasm",
              "sha256": "789abc..."
            }
          }
        }
      }
    }
  }
}
```

### Local Extension Storage

```
~/.agh/
  extensions/
    installed/
      pgvector-memory/
        extension.toml          # Manifest
        bin/
          agh-ext-pgvector      # Binary (subprocess type)
        provenance.json         # Install metadata + SHA verification
      content-filter/
        extension.toml
        plugin.wasm             # Wasm binary (wasm type)
        provenance.json
    registry-cache.json         # Cached registry index
```

### Version Management

- **Semantic Versioning (SemVer)** for all extensions
- **`min_agh_version`** in manifests to declare compatibility
- **Lockfile** (`~/.agh/extensions/lock.json`) pinning exact versions
- **`agh extension update`** respects SemVer ranges
- **Automatic compatibility checking** at daemon startup (skip incompatible extensions with warning)

### Signing and Verification

Phase 1 (alpha): SHA-256 checksum verification only (matches AGH's current skill provenance system)
Phase 2 (beta): Cosign-based signature verification for published extensions
Phase 3 (GA): Mandatory signing for registry-published extensions; unsigned extensions require explicit `--allow-unsigned` flag

---

## Security Considerations

### Threat Model

| Threat | Layer 1 (Go) | Layer 2 (Wasm) | Layer 3 (Subprocess) |
|---|---|---|---|
| **Malicious code execution** | N/A (compiled in) | Mitigated: Wasm sandbox, no syscalls | Mitigated: process isolation |
| **Memory corruption** | Risk: shared address space | Mitigated: linear memory isolation | Mitigated: separate process |
| **Filesystem access** | Full access | Denied by default; explicit capability grants | Full access (constrain via permissions) |
| **Network access** | Full access | Denied by default; explicit host function grants | Full access (constrain via permissions) |
| **Resource exhaustion (CPU)** | Risk: shared process | Mitigated: wazero fuel metering / timeouts | Mitigated: cgroups / process limits |
| **Resource exhaustion (memory)** | Risk: shared heap | Mitigated: wazero memory limits | Mitigated: process memory limits |
| **Supply chain attack** | Mitigated: code review | Mitigated: checksum verification + sandbox | Mitigated: checksum verification |
| **Host crash** | Risk: panic kills daemon | Safe: Wasm trap handled by runtime | Safe: subprocess crash isolated |

### Security Controls by Extension Type

**Wasm Extensions (Layer 2):**
- Linear memory sandbox (cannot read/write host memory)
- No filesystem access unless explicitly granted via host functions
- No network access unless explicitly granted via host functions
- Fuel metering to prevent infinite loops (wazero supports this)
- Memory limits per plugin instance
- Deterministic execution (no threads, no random without capability)

**Subprocess Extensions (Layer 3):**
- OS-level process isolation
- Restricted environment variables (allowlist, matching AGH's existing `hookEnvAllowlist` pattern)
- JSON-RPC protocol boundary (extension only sees what the host sends)
- Startup timeout + heartbeat monitoring
- Graceful shutdown with escalation (SIGTERM -> wait -> SIGKILL, matching AGH's existing pattern)
- Optional: run in containers/namespaces for additional isolation (future)

**All Extensions:**
- SHA-256 checksum verification at install time (matching AGH's existing `Provenance` system)
- Manifest declares required permissions; daemon enforces at load time
- Marketplace trust tiers: bundled > user-local > marketplace (matching AGH's existing `SkillSource` precedence)
- Allowed-marketplace allowlist for MCP/hooks (matching AGH's existing `AllowedMarketplaceHooks` pattern)

### Supply Chain Security Lessons

The OpenClaw "ClawHavoc" incident (early 2026) -- where 341 malicious skills were uploaded to ClawHub, compromising thousands of instances -- underscores critical lessons:

1. **No default trust for marketplace content.** AGH's existing pattern of requiring explicit allowlisting (`AllowedMarketplaceHooks`, `AllowedMarketplaceMCP`) is correct. Extend this to all extension types.
2. **Content verification is mandatory.** AGH's existing `VerifyHash` and `VerifyContent` patterns should be applied to all extension artifacts.
3. **Sandboxing is not optional.** Wasm extensions get this for free. Subprocess extensions should run with minimal privileges.
4. **Code signing (future).** Move toward mandatory cryptographic signatures for marketplace-published extensions.

---

## Case Studies

### Terraform (HashiCorp)

**Architecture:** gRPC subprocess plugins via `hashicorp/go-plugin`. Each provider is a separate Go binary communicating over gRPC with protocol buffers.

**What AGH should learn:**
- The **provider plugin framework** (`terraform-plugin-framework`) abstracts away gRPC boilerplate, giving provider authors a request/response API that feels like writing a Go HTTP handler. AGH should provide similarly high-level abstractions over JSON-RPC.
- **Terraform Registry** uses a Git-based index with artifacts on GitHub Releases -- simple, scalable, and avoids operating a custom artifact server. AGH should follow the same model.
- **Muxing** allows incremental migration: old SDK plugins and new framework plugins can coexist in the same provider. AGH should design for forward compatibility from day one.
- **Limitation:** Terraform providers are overwhelmingly Go-only despite gRPC's language agnosticism. The protobuf/gRPC toolchain barrier is real. AGH's choice of JSON-RPC (simpler toolchain) for subprocess extensions should lower this barrier.

### VS Code

**Architecture:** Extensions run in a separate Node.js "Extension Host" process, communicating with the main Electron renderer via IPC. Extensions declare capabilities in `package.json` (contribution points) and activate lazily based on events.

**What AGH should learn:**
- **Lazy activation** is critical. VS Code loads extensions only when their activation events fire. AGH should similarly load Wasm extensions on first invocation rather than at daemon startup.
- **Contribution points** (static declarations in `package.json`) provide a declarative way to extend UI and behavior without running code. AGH's `extension.toml` manifest serves the same purpose.
- **No DOM access** -- extensions interact through a well-defined API, never directly with the UI layer. AGH should similarly ensure extensions interact only through defined contracts, never directly with internal data structures.
- **AI extensibility (2025-2026):** VS Code added Language Model Tools, MCP integration, and Chat Participants. This three-tier approach (tools for agent mode, MCP for external services, chat participants for custom experts) mirrors AGH's three-layer architecture.

### Grafana

**Architecture:** Dual-layer plugin system. Frontend plugins (TypeScript/React) loaded via SystemJS. Backend plugins (Go) launched as subprocesses via `hashicorp/go-plugin` with gRPC communication.

**What AGH should learn:**
- **Pipeline-based plugin loading** (Discovery -> Bootstrap -> Validation -> Initialization) is clean and testable. AGH should follow a similar staged loading pipeline.
- **Health checks** -- Grafana exposes plugin health via HTTP API, allowing external monitoring. AGH should expose extension health status through its existing HTTP API.
- **Instance management** -- Grafana passes all configuration in each request, allowing stateless plugin operation. AGH should consider a similar model for Wasm extensions (pass context per call rather than requiring plugins to maintain state).
- **Plugin SDK versioning** -- Grafana's SDK is versioned independently of Grafana itself, with the protocol considered stable. AGH should version its extension protocol separately from the daemon version.

### Neovim

**Architecture:** Multi-layer: in-process Lua (LuaJIT) for fast plugins, msgpack-RPC for remote/out-of-process plugins, provider system for delegating external capabilities.

**What AGH should learn:**
- **Two-speed plugin model** -- fast in-process (Lua/LuaJIT) for common operations, slower out-of-process (RPC) for heavy lifting. Directly analogous to AGH's Wasm (fast, in-process) + subprocess (full-power) split.
- **No manifest registration** -- plugins are discovered by filesystem convention. AGH's skill system already follows this pattern (scan `.agh/skills/` for `SKILL.md` files).
- **Built-in subsystems as extension points** -- Neovim's LSP client, Treesitter, and diagnostics framework provide rich hook points for plugins. AGH should similarly expose session management, memory, and observation as hookable subsystems.
- **Provider pattern** -- Neovim delegates clipboard, Python, Ruby to external "providers." AGH can use the same pattern: the daemon provides the contract, external extensions provide the implementation.

### Claude Code

**Architecture:** Four-tier extension: Skills (markdown-based instruction injection), Hooks (shell commands at lifecycle events), MCP servers (external tool access), Plugins (bundled distribution).

**What AGH should learn:**
- **Progressive disclosure** -- Skills load lazily; only names and descriptions are visible until invoked. This is critical for context window efficiency in AI agents. AGH should adopt the same approach: extension metadata is always available, but extension bodies/code load on demand.
- **Skill-as-markdown** -- Claude Code's skills are just markdown files with YAML frontmatter, not compiled code. AGH already follows this for skills; the extension system should complement (not replace) this pattern.
- **Hooks as shell commands** -- Simple, language-agnostic, zero-dependency. AGH already implements this in `HookRunner`. The extension system should extend this to Wasm hooks for sandboxed execution.
- **Trust tiers** -- Bundled > user-local > marketplace, with explicit allowlisting for marketplace extensions. AGH already implements this exact pattern in `marketplaceSkillAllowed`.

### MCP Ecosystem

**Architecture:** JSON-RPC 2.0 over stdio (local) or Streamable HTTP (remote). Capability negotiation during `initialize` handshake. Tools, resources, and prompts as standardized extension primitives.

**What AGH should learn:**
- **Protocol-first design** -- MCP defines the protocol, then SDKs implement it in multiple languages. AGH should define its extension protocol spec first, then build SDKs.
- **Capability negotiation** -- The `initialize` handshake where both sides declare what they support is elegant and forward-compatible. AGH should use the same pattern for subprocess extensions.
- **Transport agnosticism** -- The protocol works identically over stdio, HTTP+SSE, or WebSockets. AGH should similarly keep its extension protocol transport-agnostic, starting with stdio but designed to work over other transports.
- **Ecosystem scale** -- MCP grew from zero to thousands of servers in under a year by making it trivially easy to write a server in any language. AGH should target the same simplicity for extension authoring.

---

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-4)

1. Define extension protocol specification (JSON-RPC methods, capability negotiation, error codes)
2. Create `internal/extension` package with `Manager`, `Registry` types
3. Implement subprocess extension lifecycle (launch, handshake, shutdown) -- reuse patterns from `internal/acp`
4. Define first extension point: `session.hook` (pre/post session creation)
5. Build `agh extension list` CLI command

### Phase 2: Wasm Runtime (Weeks 5-8)

1. Integrate wazero via Extism Go SDK
2. Define WIT contracts for hook extension points
3. Implement Wasm hook chain (pre-session, pre-prompt, post-event)
4. Build `agh extension install` (local `.wasm` files)
5. Create AssemblyScript PDK for TypeScript extension authors

### Phase 3: SDK and Distribution (Weeks 9-12)

1. Publish `@agh/extension-sdk` npm package (TypeScript subprocess SDK)
2. Publish `@agh/wasm-pdk` npm package (AssemblyScript Wasm PDK)
3. Create extension scaffolding CLI (`agh extension init`)
4. Set up GitHub-based extension registry
5. Implement `agh extension publish` with checksum verification

### Phase 4: Ecosystem (Weeks 13-16)

1. Add extension points: `memory.backend`, `observe.exporter`
2. Build reference extensions (pgvector memory, OpenTelemetry exporter)
3. Extension development documentation and tutorials
4. `agh extension dev` hot-reload development mode
5. Health monitoring and observability for running extensions

---

## Appendix: Key Technology Choices

| Decision | Choice | Rationale |
|---|---|---|
| Wasm runtime | wazero (via Extism) | Pure Go, zero CGO, zero dependencies -- fits AGH's single-binary constraint. Extism adds high-level SDK. |
| Subprocess protocol | JSON-RPC 2.0 over stdio | Already used by AGH (ACP), aligned with MCP/LSP ecosystems, trivial multi-language support. |
| Contract definition | Protobuf (for types) + WIT (for Wasm) | Protobuf gives typed code generation for all languages; WIT is native to Wasm Component Model. |
| TypeScript SDK transport | stdin/stdout JSON-RPC | Node.js subprocess managed by AGH daemon, identical lifecycle to ACP agents. |
| Extension registry | Git-based index + GitHub Releases | Proven by Terraform/Homebrew; no custom infrastructure to operate. |
| Checksum verification | SHA-256 | Matches AGH's existing `Provenance` system; simple, well-understood. |
| TypeScript-to-Wasm | AssemblyScript (primary), Javy (alternative) | AssemblyScript: TypeScript-like syntax, direct Wasm compilation. Javy: full JS support via QuickJS-in-Wasm. |

---

## Sources

- [HashiCorp go-plugin (GitHub)](https://github.com/hashicorp/go-plugin)
- [HashiCorp go-plugin Tutorial](https://github.com/hashicorp/go-plugin/blob/main/docs/extensive-go-plugin-tutorial.md)
- [Eli Bendersky: RPC-based plugins in Go](https://eli.thegreenplace.net/2023/rpc-based-plugins-in-go/)
- [HashiCorp Plugin System Design (Medium)](https://zerofruit-web3.medium.com/hashicorp-plugin-system-design-and-implementation-5f939f09e3b3)
- [Extism Official Site](https://extism.org/)
- [Extism Go SDK (GitHub)](https://github.com/extism/go-sdk)
- [Extism v1 Announcement (The New Stack)](https://thenewstack.io/extism-v1-run-webassembly-in-your-app/)
- [wazero Official Site](https://wazero.io/)
- [wazero GitHub](https://github.com/wazero/wazero)
- [Arcjet: Lessons from running Wasm in production with Go & Wazero](https://blog.arcjet.com/lessons-from-running-webassembly-in-production-with-go-wazero/)
- [Wasmtime vs Wazero Comparison 2026](https://wasmruntime.com/en/compare/wasmtime-vs-wazero)
- [WASI and the WebAssembly Component Model: Current Status (eunomia)](https://eunomia.dev/blog/2025/02/16/wasi-and-the-webassembly-component-model-current-status/)
- [WASI 2.0 Components: Portable, Fast Plugins (Medium)](https://medium.com/@hadiyolworld007/wasi-2-0-components-portable-fast-plugins-58c24d891584)
- [WebAssembly Component Model Building a Plugin System (DEV)](https://dev.to/topheman/webassembly-component-model-building-a-plugin-system-58o0)
- [WebAssembly Component Model Introduction](https://component-model.bytecodealliance.org/)
- [MCP Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)
- [MCP Transport Future (Blog)](https://blog.modelcontextprotocol.io/posts/2025-12-19-mcp-transport-future/)
- [JSON-RPC Protocol in MCP Guide](https://mcpcat.io/guides/understanding-json-rpc-protocol-mcp/)
- [Terraform Provider Design Principles](https://developer.hashicorp.com/terraform/plugin/best-practices/hashicorp-provider-design-principles)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [terraform-plugin-framework (GitHub)](https://github.com/hashicorp/terraform-plugin-framework)
- [VS Code Extension API](https://code.visualstudio.com/api)
- [VS Code Extensibility Principles and Patterns](https://vscode-docs1.readthedocs.io/en/latest/extensionAPI/patterns-and-principles/)
- [VS Code AI Extensibility](https://code.visualstudio.com/api/extension-guides/ai/ai-extensibility-overview)
- [VS Code Architecture Guide](https://thedeveloperspace.com/vs-code-architecture-guide/)
- [Grafana Plugin System (DeepWiki)](https://deepwiki.com/grafana/grafana/11-plugin-system)
- [Grafana Backend Plugins](https://grafana.com/developers/plugin-tools/key-concepts/backend-plugins/)
- [Grafana Plugin SDK for Go (GitHub)](https://github.com/grafana/grafana-plugin-sdk-go)
- [Neovim Extension and Plugin System (DeepWiki)](https://deepwiki.com/neovim/neovim/4-api-and-extensions)
- [Neovim Lua Plugin Docs](https://neovim.io/doc/user/lua-plugin/)
- [Neovim Plugin Management and Providers (DeepWiki)](https://deepwiki.com/neovim/neovim/4.6-plugin-management-and-providers)
- [Claude Code Skills Documentation](https://code.claude.com/docs/en/skills)
- [Understanding Claude Code's Full Stack (alexop.dev)](https://alexop.dev/posts/understanding-claude-code-full-stack/)
- [Claude Code Extensions Guide 2026 (Morph)](https://www.morphllm.com/claude-code-extensions)
- [Claude Code Extensibility Analysis (DeepWiki)](https://deepwiki.com/liuup/claude-code-analysis/5-extensibility)
- [OpenClaw GitHub](https://github.com/openclaw/openclaw)
- [OpenClaw Guide 2026 (AiCybr)](https://aicybr.com/blog/openclaw-guide)
- [OpenClaw Ecosystem 2026](https://openclawnews.online/article/openclaw-ecosystem-2026)
- [waPC Specification](https://wapc.io/docs/spec/)
- [waPC Go Host (GitHub)](https://github.com/wapc/wapc-go)
- [knqyf263/go-plugin: Go Plugin System over WebAssembly (GitHub)](https://github.com/knqyf263/go-plugin)
- [Go Security Best Practices (Official)](https://go.dev/doc/security/best-practices)
- [Go Plugin Package Documentation](https://pkg.go.dev/plugin)
- [Plugin Architecture: Versioning, Distribution, and Ecosystem](https://oninebx.github.io/blog/architecture/plugin-architecture-in-practice-part-4-versioning-distribution-and-ecosystem/)
- [gRPC with TypeScript in 2025](https://caisy.io/blog/grpc-typescript)
- [How to Generate gRPC Code from Proto Files in Multiple Languages](https://oneuptime.com/blog/post/2026-01-08-grpc-code-generation-multiple-languages/view)
- [TypeScript in WebAssembly (Fermyon)](https://developer.fermyon.com/wasm-languages/typescript)
- [WebAssembly Ecosystem 2026](https://reintech.io/blog/webassembly-ecosystem-2026-tools-frameworks-runtimes)
- [Wasmnizer-ts: TypeScript to WasmGC (GitHub)](https://github.com/web-devkits/Wasmnizer-ts)
- [QJS: JavaScript in Go via Wasm (InfoQ)](https://www.infoq.com/news/2025/12/javascript-golang-wasm/)
- [Awesome Code Sandboxing for AI (GitHub)](https://github.com/restyler/awesome-sandbox)
- [Go Sandbox Library (GitHub)](https://github.com/criyle/go-sandbox)
- [Middleware Patterns in Go](https://drstearns.github.io/tutorials/gomiddleware/)
- [Watermill: Event-Driven Go Library](https://threedots.tech/event-driven/)
