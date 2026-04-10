# Extension Libraries & Frameworks Research

> Research date: 2026-04-10
> Target: AGH Agent Operating System -- three-tier extension architecture

---

## Wasm Runtime: wazero

### Overview

[wazero](https://github.com/tetratelabs/wazero) is a WebAssembly Core Specification 1.0 and 2.0 compliant runtime written in pure Go with **zero dependencies** and no CGO requirement. This is the strongest differentiator -- it preserves Go's cross-compilation story and adds minimal binary size overhead.

### Latest Stable Version

**v1.10.1** (latest as of early 2026)

- v1.10.0 was the first release under the new `wazero/wazero` GitHub org (previously `tetratelabs/wazero`)
- Experimental features: concurrent Wasm compilation, tail-call proposal
- Requires Go 1.23+ (floor version)
- v1.7 introduced an optimizing compiler with 30-40% average performance improvements
- 692+ known importers on pkg.go.dev

### Key Features for AGH

| Feature | Status | Notes |
|---|---|---|
| Context cancellation/timeout | Supported | `WithCloseOnContextDone(true)` -- essential for sandboxing untrusted code |
| Goroutine safety | Supported | 1:1 goroutine mapping; share `CompiledModule` across goroutines |
| Compilation cache | Supported | Pre-compile once, instantiate many times |
| WASI Preview 1 | Supported | Full wasip1 support |
| WASI Preview 2 | **Not supported** | Open issue [#2289](https://github.com/tetratelabs/wazero/issues/2289) |
| Fuel/gas metering | **Not supported** | No native fuel API; contrast with Wasmtime |
| Interpreter mode | Supported | Useful for debugging |
| Compiler mode | Supported | Production performance |

### Resource Limiting

wazero does **not** have built-in fuel metering like Wasmtime. The available mechanisms are:

1. **`context.WithTimeout`** + `WithCloseOnContextDone(true)` -- time-based execution limits
2. **Memory limits** -- configurable per-module memory caps
3. **`CompiledModule` sharing** -- compile once, instantiate cheaply per request

The `WithCloseOnContextDone` option inserts periodic cancellation checks in the interpreter/compiler, with a small performance cost (disabled by default).

### Gotchas

- No fuel metering means you can only limit by wall-clock time, not instruction count
- WASI Preview 2 absence means no Component Model support through wazero alone
- The `wazero/wazero` org migration may cause import path confusion -- the canonical import remains `github.com/tetratelabs/wazero`

### Links

- GitHub: https://github.com/tetratelabs/wazero
- Docs: https://wazero.io/
- pkg.go.dev: https://pkg.go.dev/github.com/tetratelabs/wazero
- Specs: https://wazero.io/specs/

---

## Wasm Plugin Framework: Extism

### Overview

[Extism](https://extism.org/) is a cross-language WebAssembly plugin framework that provides a higher-level abstraction over raw Wasm runtimes. The Go SDK (`extism/go-sdk`) uses wazero under the hood, providing plugin lifecycle management, host functions, memory management, and security sandboxing.

### Latest Versions

| Component | Version | Date |
|---|---|---|
| Extism Go SDK (`extism/go-sdk`) | **v1.3.0** | ~March 2025 |
| Extism Core Runtime (`extism/extism`) | **v1.12.0** | 2025 |
| Extism Go PDK (`extism/go-pdk`) | Latest commit Jan 2026 | Published Mar 2025 |
| Extism CLI | v1.5.2 | References go-sdk v1.3.0 |

### Go SDK API Surface

```go
// Plugin creation
plugin, err := extism.NewPlugin(ctx, manifest, config, hostFunctions)

// Compiled plugins for concurrent use
compiled, err := extism.NewCompiledPlugin(ctx, manifest, config, hostFunctions)
instance, err := compiled.Instance(ctx, extism.PluginInstanceConfig{})

// Calling exports
exitCode, output, err := plugin.Call("function_name", input)
exitCode, output, err := plugin.CallWithContext(ctx, "function_name", input)
```

### Key Features for AGH

| Feature | Status | Notes |
|---|---|---|
| Timeout | Supported | `Manifest.Timeout` field (uint64 ms), 30s default |
| Fuel metering | Supported | Plugins can be initialized with fuel limits |
| Host functions | Supported | Inject Go functions callable from Wasm |
| Module-scope variables | Supported | Persistent state between calls |
| HTTP control | Supported | Host-controlled HTTP without WASI |
| Memory limits | Supported | `MaxVarBytes`, `MaxHttpResponseBytes` |
| Allowed hosts/paths | Supported | Filesystem and network sandboxing |
| Compilation cache | Supported | Via wazero's compilation cache |
| Multi-language PDKs | Supported | Rust, Go, JS, Python, C/C++, AssemblyScript, Zig, Haskell |

### Plugin Development Kit (PDK) for Go

The Go PDK supports both TinyGo and the standard Go toolchain (Go 1.24+):

```go
// Plugin code (compiled to Wasm)
//go:wasmexport greet
func greet() int32 {
    name := pdk.InputString()
    pdk.OutputString("Hello, " + name)
    return 0
}
```

**Build options:**
- TinyGo: `tinygo build -target wasip1 -o plugin.wasm` (smaller output, ~5x smaller)
- Go native: `GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm -buildmode=c-shared`

TinyGo 0.37.0 is required for Go 1.24 compatibility and `//go:wasmexport` support.

### Gotchas

- The Go SDK wraps wazero's `Module` type -- if you need raw wazero access, you go through Extism's abstraction
- Go PDK with standard toolchain produces larger Wasm binaries than TinyGo
- API stability: the `extism/extism` package once warned "APIs may change until v1.0" -- the newer `extism/go-sdk` appears more stable
- Limited production case studies publicly available; community primarily in Discord
- 38 known importers on pkg.go.dev (relatively small ecosystem)

### Links

- Go SDK: https://github.com/extism/go-sdk
- Go PDK: https://github.com/extism/go-pdk
- Docs: https://extism.org/docs/
- Host Functions: https://extism.org/docs/concepts/host-functions/

---

## Extism Alternatives

### 1. knqyf263/go-plugin (Go Plugin System over WebAssembly)

**Version: v0.9.0** (March 12, 2025)

A Go plugin system that auto-generates type-safe Go SDKs from Protocol Buffers definitions. Uses wazero under the hood. Inspired by HashiCorp's go-plugin but communicates in-memory via Wasm instead of over RPC.

**Strengths:**
- Protobuf-based interface definition (familiar to gRPC users)
- Auto-generated Go SDK hides raw Wasm APIs from plugin authors
- Supports native Go plugins with WASI (wasip1)
- Memory-safe, sandboxed, portable
- Used in production by CNCF's Node Resource Interface (NRI)

**Weaknesses:**
- Depends on TinyGo for protobuf compatibility (well-known types reimplemented)
- v0.9.0 -- not yet v1.0 stable
- Smaller community than Extism

**Key difference from Extism:** go-plugin is Go-centric with protobuf contracts; Extism is language-agnostic with a universal ABI.

**Link:** https://github.com/knqyf263/go-plugin

### 2. HashiCorp go-plugin (RPC-based, not Wasm)

**Version: v1.6.3** (August 2025)

The battle-tested plugin system used by Terraform, Vault, Nomad, Boundary, and Waypoint. Launches plugins as subprocesses communicating over gRPC or net/rpc.

**Strengths:**
- 3,714 known importers -- massive production usage
- Supports plugins in any language (via gRPC)
- Process isolation (crash safety)
- Multiplexed gRPC connections
- Unix socket support with configurable permissions
- Health checking, version negotiation, secure handshake

**Weaknesses:**
- Subprocess overhead (not in-process like Wasm)
- MPL-2.0 license
- gRPC dependency adds binary size
- Not Wasm-based -- different security model

**Relevance to AGH:** Already similar to AGH's tier 3 (JSON-RPC subprocess). Could be used as-is for the subprocess tier, or its patterns could inform AGH's own implementation.

**Link:** https://github.com/hashicorp/go-plugin

### 3. Go 1.24+ Native Wasm Support

Go 1.24 added `//go:wasmexport` and WASI reactor build mode, making it possible to build Wasm plugins using the standard Go toolchain without TinyGo or Extism.

**Strengths:**
- Zero external dependencies
- Standard toolchain support
- Growing ecosystem

**Weaknesses:**
- No plugin lifecycle management (raw Wasm only)
- Larger binary output than TinyGo
- No host function framework -- you build it yourself
- Type restrictions on `wasmexport`/`wasmimport` (no pointer passing due to 32/64-bit mismatch)

### 4. Go's Native `plugin` Package

The standard library `plugin` package loads compiled `.so` shared objects at runtime.

**Strengths:**
- Part of Go standard library
- No serialization overhead

**Weaknesses:**
- Linux and macOS only (no Windows)
- Requires matching Go versions between host and plugin
- No sandboxing
- Not suitable for untrusted code

**Verdict:** Not recommended for AGH's use case.

### Comparison Matrix

| Framework | In-Process | Cross-Language | Type-Safe | Sandboxed | Production Maturity |
|---|---|---|---|---|---|
| Extism | Yes (Wasm) | Yes (universal ABI) | Via host functions | Yes | Medium |
| knqyf263/go-plugin | Yes (Wasm) | Go-centric | Protobuf-generated | Yes | Low-Medium |
| HashiCorp go-plugin | No (subprocess) | Yes (gRPC) | Protobuf-generated | Process isolation | Very High |
| Go 1.24 native Wasm | Yes (Wasm) | Build-your-own | Manual | Wasm sandbox | Low |
| Go `plugin` package | Yes (.so) | Go only | Go interfaces | None | Low |

---

## JSON-RPC Libraries

### Recommended Libraries for AGH's Tier 3 (Subprocess Extensions)

#### 1. sourcegraph/jsonrpc2 -- Best for Bidirectional Stdio

**Import:** `github.com/sourcegraph/jsonrpc2`
**Published:** February 2025, MIT license
**Importers:** 419 packages

The most battle-tested option for bidirectional JSON-RPC 2.0 over stdio. Used extensively in LSP implementations.

```go
// Symmetric connection -- both client and server
conn := jsonrpc2.NewConn(ctx, jsonrpc2.NewPlainObjectStream(rwc), handler)
```

**Features:**
- Bidirectional (symmetric client/server on same connection)
- Works with any `io.ReadWriteCloser` (stdio, TCP, etc.)
- Request/response correlation
- Notification support
- Handler interface for incoming requests

**Best fit for AGH:** Already the pattern used by LSP servers and ACP-compatible agents. Minimal abstraction, maximum control.

#### 2. golang.org/x/exp/jsonrpc2 -- Official Experimental

**Import:** `golang.org/x/exp/jsonrpc2`
**Published:** January 2026, BSD-3-Clause
**Importers:** 21 packages

The publicly importable version of the internal `golang.org/x/tools` jsonrpc2 implementation used by gopls.

**Features:**
- Bidirectional `Connection` type
- Pluggable `Framer` (HeaderFramer for LSP, RawFramer for raw JSON)
- `Dial` function with configurable binder
- Preempter and Handler patterns

**Caveat:** Under `x/exp` -- API stability not guaranteed.

#### 3. viant/jsonrpc -- Best Stdio Client

**Import:** `github.com/viant/jsonrpc`

Purpose-built for launching subprocesses and communicating via JSON-RPC 2.0 over stdin/stdout. Also provides streamable HTTP transport.

```go
client := stdio.New("my_service", stdio.WithArguments("--flag"), stdio.WithEnvironment("KEY=value"))
response, err := client.Send(ctx, request)
```

**Features:**
- Dedicated stdio client transport
- Process execution with configurable args/env
- Request, notification, and batch call support
- Also supports streamable HTTP with session management
- Used as transport layer for Viant's MCP implementation

#### 4. modelcontextprotocol/go-sdk -- Official MCP SDK

**Import:** `github.com/modelcontextprotocol/go-sdk`
**Published:** March 31, 2026, Apache-2.0
**Importers:** 1,443 packages

The official Go SDK for Model Context Protocol, maintained by Google and Anthropic. Built on JSON-RPC 2.0 with stdio and streamable HTTP transports.

**Relevance to AGH:** Since AGH already speaks ACP (which is MCP-adjacent), this SDK's transport layer and JSON-RPC patterns are directly relevant. The `jsonrpc` sub-package can be used independently for custom transports.

### JSON-RPC Library Comparison

| Library | Stdio | Bidirectional | LSP-Compatible | Maturity |
|---|---|---|---|---|
| `sourcegraph/jsonrpc2` | Via ReadWriteCloser | Yes | Yes | High (419 importers) |
| `golang.org/x/exp/jsonrpc2` | Via Framer | Yes | Yes (HeaderFramer) | Medium (21 importers) |
| `viant/jsonrpc` | Dedicated transport | Client-side | N/A | Medium |
| `modelcontextprotocol/go-sdk` | Built-in | Yes | N/A (MCP-specific) | High (1,443 importers) |

---

## WebAssembly Component Model in Go

### Current Status (April 2026)

The Component Model is **not yet natively supported** by the standard Go compiler. The situation is evolving rapidly:

### Go Compiler Support

| Capability | Status |
|---|---|
| GOOS=js GOARCH=wasm | Stable (Go 1.21+) |
| GOOS=wasip1 GOARCH=wasm | Stable (Go 1.21+) |
| `//go:wasmexport` | Stable (Go 1.24+) |
| GOOS=wasip2 | **Not supported** |
| GOOS=wasip3 | **Proposed** ([issue #77141](https://github.com/golang/go/issues/77141)) |
| Component Model output | **Not supported** (requires TinyGo or wit-bindgen-go) |

### WASIp3 Proposal

A proposal has been filed to add `wasip3/wasm` as a new Go port. WASIp3 is expected in early 2026 and integrates Component Model concurrency primitives (cooperative threads) that map well to Go's goroutine scheduler. However, this is still in proposal stage.

### Bytecode Alliance Tooling

The [bytecodealliance/go-modules](https://github.com/bytecodealliance/go-modules) project provides `wit-bindgen-go` to generate Go bindings from WIT (WebAssembly Interface Type) files. This is the primary path for Go developers wanting to use the Component Model today.

### WASI Roadmap

| Milestone | Expected Timeline |
|---|---|
| WASI Preview 2 (wasip2) | Released 2025 |
| WASI 0.3 (async I/O) | RC in late 2025, stabilizing 2026 |
| WASI 1.0 (stable) | Late 2026 / early 2027 |

### Implications for AGH

- **Short-term (2026):** The Component Model is not practical for AGH's plugin system via Go. Extism + wazero with WASI Preview 1 is the pragmatic choice.
- **Medium-term (2027):** Once WASI 1.0 lands and Go adds wasip3 support, the Component Model could replace Extism's custom ABI for cross-language plugins.
- **Risk:** Building on the Component Model now requires TinyGo or external tooling, adding complexity without clear benefit over Extism's proven approach.

---

## Modern Go Plugin Patterns

### Pattern 1: Interface-Driven Contracts (Tier 1 -- Go-native)

The dominant Go pattern. Define interfaces where consumed, implement in separate packages.

```go
// Consumed by session package
type AgentDriver interface {
    Start(ctx context.Context, cfg AgentConfig) error
    Send(ctx context.Context, msg Message) error
    Stop(ctx context.Context) error
}
```

AGH already uses this pattern. For first-party extensions, this is the right approach -- zero overhead, type-safe, compile-time verified.

### Pattern 2: Functional Options + Registry (Tier 1)

```go
type ExtensionRegistry struct {
    drivers map[string]AgentDriverFactory
    hooks   map[string][]HookHandler
}

func NewRegistry(opts ...RegistryOption) *ExtensionRegistry
func WithDriver(name string, factory AgentDriverFactory) RegistryOption
```

Small registries (maps for <10 items) are preferred over complex registry interfaces per AGH's architecture principles.

### Pattern 3: Interface Extension (Progressive Capability)

Used by go-mysql-server and others. Base interface is required; additional interfaces unlock optional capabilities.

```go
type Extension interface {
    Name() string
    Init(ctx context.Context) error
}

// Optional capabilities
type WithHealthCheck interface {
    HealthCheck(ctx context.Context) error
}

type WithMetrics interface {
    Metrics() []Metric
}
```

Caveat: `runtime.assertI2I` (type assertions) can consume ~13% CPU in tight loops. Use for initialization/configuration, not hot paths.

### Pattern 4: Wasm Sandbox (Tier 2)

Using Extism or raw wazero for in-process sandboxed execution of untrusted code.

### Pattern 5: RPC Subprocess (Tier 3)

HashiCorp go-plugin pattern: launch subprocess, negotiate protocol, communicate over gRPC/JSON-RPC. AGH's existing ACP driver model is already this pattern.

---

## Recommended Stack

### Tier 1: Go-Native Interfaces (First-Party)

**Approach:** Interface-driven contracts with functional options and a small registry.

- No additional dependencies needed
- AGH already implements this via `session.AgentDriver` and similar interfaces
- Use interface extension pattern for progressive capability discovery

### Tier 2: WebAssembly Sandbox (Lightweight Hooks/Validators)

**Recommended:** Extism Go SDK (`github.com/extism/go-sdk`) v1.3.0+

**Rationale:**
- Built on wazero (pure Go, no CGO) -- preserves AGH's single-binary story
- Provides timeout + fuel metering out of the box -- critical for untrusted code
- Host functions enable injecting AGH capabilities into plugins
- Multi-language PDKs let extension authors use Rust, Go, JS, etc.
- Compilation cache + `CompiledPlugin` for concurrent plugin instances

**Alternative considered:** `knqyf263/go-plugin` (v0.9.0) is appealing for its protobuf-based type safety but is less mature and Go-centric. Extism's universal ABI and multi-language support better fit AGH's agent ecosystem.

**Risk mitigation:**
- Extism's relatively small Go ecosystem (38 importers) is a concern. Mitigate by keeping a thin adapter layer so the host-side code could be rewritten against raw wazero if needed.
- Wrap Extism types behind AGH-owned interfaces (already an AGH architecture principle).

### Tier 3: JSON-RPC Subprocess (Rich Extensions/Agent Drivers)

**Recommended:** `github.com/sourcegraph/jsonrpc2` for the transport layer

**Rationale:**
- 419 importers, MIT license, actively maintained
- Bidirectional symmetric connection -- both host and extension can send requests
- Works with any `io.ReadWriteCloser` -- trivial to use with subprocess stdio
- Battle-tested in LSP implementations (same protocol pattern as ACP)

**Also consider:** `github.com/modelcontextprotocol/go-sdk` if AGH wants to adopt MCP-compatible extension protocol directly. The official SDK's `jsonrpc` sub-package provides a clean transport abstraction.

**Pattern:** Follow HashiCorp go-plugin's lifecycle model (subprocess launch, protocol negotiation, health checking, graceful shutdown) but use JSON-RPC 2.0 instead of gRPC to stay aligned with ACP.

---

## Version Matrix

| Dependency | Version | Go Minimum | License | Import Path |
|---|---|---|---|---|
| wazero | v1.10.1 | Go 1.23 | Apache-2.0 | `github.com/tetratelabs/wazero` |
| Extism Go SDK | v1.3.0 | (inherits wazero) | BSD-3-Clause | `github.com/extism/go-sdk` |
| Extism Go PDK | ~v1.1.0 | Go 1.24 (native) or TinyGo 0.37.0 | BSD-3-Clause | `github.com/extism/go-pdk` |
| Extism Core Runtime | v1.12.0 | N/A (Rust) | BSD-3-Clause | N/A |
| sourcegraph/jsonrpc2 | latest (Feb 2025) | ~Go 1.21 | MIT | `github.com/sourcegraph/jsonrpc2` |
| golang.org/x/exp/jsonrpc2 | latest (Jan 2026) | ~Go 1.21 | BSD-3-Clause | `golang.org/x/exp/jsonrpc2` |
| viant/jsonrpc | latest | ~Go 1.21 | Apache-2.0 | `github.com/viant/jsonrpc` |
| knqyf263/go-plugin | v0.9.0 | Go 1.21+ | MIT | `github.com/knqyf263/go-plugin` |
| HashiCorp go-plugin | v1.6.3 | Go 1.21+ | MPL-2.0 | `github.com/hashicorp/go-plugin` |
| MCP Go SDK (official) | latest (Mar 2026) | Go 1.22+ | Apache-2.0 | `github.com/modelcontextprotocol/go-sdk` |
| TinyGo | 0.37.0 | Go 1.24.0 | BSD-3-Clause | N/A (compiler) |
| Go (std toolchain) | 1.24+ | N/A | BSD-3-Clause | N/A |

### Key Compatibility Notes

1. **wazero v1.10.x requires Go 1.23+** -- AGH should ensure its Go version floor accommodates this
2. **Extism Go SDK depends on wazero** -- version coupling means Extism updates may lag wazero releases
3. **Go PDK with standard toolchain requires Go 1.24+** for `//go:wasmexport`; TinyGo 0.37.0 for TinyGo path
4. **WASI Preview 2 is NOT available** through wazero or Extism -- only WASI Preview 1 (wasip1)
5. **Component Model** requires TinyGo + `wit-bindgen-go` today; native Go support expected ~2027
6. **sourcegraph/jsonrpc2 is MIT** -- no license compatibility concerns with AGH

---

## Sources

- [wazero GitHub](https://github.com/tetratelabs/wazero)
- [wazero Docs](https://wazero.io/)
- [wazero vs CGO 2026](https://wasmruntime.com/en/blog/wazero-vs-cgo-2026)
- [wazero WASI Preview 2 Issue #2289](https://github.com/tetratelabs/wazero/issues/2289)
- [Extism Go SDK](https://github.com/extism/go-sdk)
- [Extism Go PDK](https://github.com/extism/go-pdk)
- [Extism Docs](https://extism.org/)
- [Extism Host Functions](https://extism.org/docs/concepts/host-functions/)
- [knqyf263/go-plugin](https://github.com/knqyf263/go-plugin)
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)
- [sourcegraph/jsonrpc2](https://github.com/sourcegraph/jsonrpc2)
- [golang.org/x/exp/jsonrpc2](https://pkg.go.dev/golang.org/x/exp/jsonrpc2)
- [viant/jsonrpc](https://github.com/viant/jsonrpc)
- [MCP Official Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Go 1.24 Wasm Blog Post](https://go.dev/blog/wasmexport)
- [Google Cloud: Go 1.24 Wasm](https://cloud.google.com/blog/products/application-development/go-1-24-expands-support-for-wasm)
- [Bytecode Alliance go-modules](https://github.com/bytecodealliance/go-modules)
- [WASIp3 Go Proposal #77141](https://github.com/golang/go/issues/77141)
- [State of WebAssembly 2025-2026](https://platform.uno/blog/the-state-of-webassembly-2025-2026/)
- [WASI Component Model Status](https://eunomia.dev/blog/2025/02/16/wasi-and-the-webassembly-component-model-current-status/)
- [WebAssembly Ecosystem 2026](https://reintech.io/blog/webassembly-ecosystem-2026-tools-frameworks-runtimes)
- [DoltHub Interface Extension Pattern](https://www.dolthub.com/blog/2022-09-12-golang-interface-extension/)
- [Go Plugin System with plugin Package](https://oneuptime.com/blog/post/2026-01-25-plugin-system-go-plugin-package/view)
- [Eli Bendersky: Plugins in Go](https://eli.thegreenplace.net/2021/plugins-in-go/)
- [trpc-mcp-go](https://pkg.go.dev/trpc.group/trpc-go/trpc-mcp-go)
- [Navidrome Plugins (Extism example)](https://github.com/navidrome/navidrome/blob/master/plugins/README.md)
