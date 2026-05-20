# AGH Extensibility: Final Analysis

**Date:** 2026-04-09
**Sources:** 8 parallel agent analyses across 6 agent harnesses (Claude Code, OpenClaw, OpenFang, GoClaw, Hermes, Pi-Mono), 3 cross-cutting knowledge bases (ai-harness, agent-networks, ai-memory), and deep web research on extensibility systems.
**Detailed per-project analyses:** `.compozy/tasks/extensability/analysis/analysis_*.md`

---

## 1. AGH's Current State (Verified)

What AGH already has and what is explicitly delegated to ACP agents:

| Capability                      | Status                         | Details                                                                                                                                                                             |
| ------------------------------- | ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Session lifecycle state machine | **Exists**                     | `StateStarting → StateActive → StateStopping → StateStopped` in `internal/session/session.go:23-28`                                                                                 |
| Approval/permissions            | **Exists (ACP passthrough)**   | `ApproveRequest`, `ResolvePermission()`, `PermissionMode` config. Flows through to ACP agents. Sufficient for now.                                                                  |
| Health endpoint                 | **Exists**                     | `GET /api/doctor` returns uptime, active sessions, DB sizes, version                                                                                                                |
| Context compaction              | **ACP agents handle it**       | Claude Code, Codex, Gemini CLI do their own compaction. Not AGH's concern.                                                                                                          |
| MCP server integration          | **By design: delegation**      | Skills declare MCP servers in frontmatter → `MCPResolver` collects them → passed to ACP agents at startup via `acp.StartOpts.MCPServers`. AGH is not an MCP client; the agents are. |
| Skills system                   | **Exists (5-tier precedence)** | `SourceBundled < SourceMarketplace < SourceUser < SourceAdditional < SourceWorkspace`. Registry with workspace cache, provenance verification, content safety checks.               |
| Hook system                     | **Exists (info-only)**         | `HookRunner` dispatches subprocess hooks for `on_session_created` and `on_session_stopped`. Cannot block or modify. Env allowlist isolation.                                        |
| Event recording                 | **Exists**                     | `internal/observe` with per-session SQLite event stores                                                                                                                             |
| Memory system                   | **Exists**                     | Dual-scope (global + workspace) with dream consolidation in `internal/memory/consolidation`                                                                                         |
| Workspace resolver              | **Exists**                     | Config merge, agent definition resolution, additional dirs, workspace-scoped skills                                                                                                 |

### What's Actually Missing

| Gap                               | Impact                                                                                                                                               | Evidence                                                                                                  |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **No extension architecture**     | Users can't create tools, hooks, or integrations without modifying Go source                                                                         | No `internal/extension` package. No plugin manifest. No subprocess or Wasm extension loading.             |
| **No tool registry**              | Tools are just `[]string` in `AgentPayload` (`internal/api/contract/contract.go:71`). No schema, no availability gating, no namespacing.             | No `Tool` interface or `ToolDriver` type anywhere in codebase.                                            |
| **Hooks can't block/modify**      | `HookRunner.RunHooks()` returns `[]HookResult` but nothing reads or acts on results. Only 2 events (`on_session_created`, `on_session_stopped`).     | `internal/skills/hooks.go:81-125` — results are captured but never used for decision-making.              |
| **No session stop reason**        | 4 binary states. No classification of _why_ a session stopped (user cancel, crash, timeout, budget, loop).                                           | `SessionInfo` has no `StopReason` field. `finalizeStopped()` records an error event but doesn't classify. |
| **No session repair on resume**   | `Resume()` re-resolves workspace/agent but doesn't validate stored state integrity.                                                                  | `manager_lifecycle.go:186-209` — reads meta, re-resolves, but no consistency checks.                      |
| **No loop/recursion guard**       | Zero depth limiting or cycle detection for tool/agent recursion.                                                                                     | No matches for `depth`, `recursion`, `cycle` in session/acp code.                                         |
| **Skills progressive disclosure** | **Resolved on 2026-04-09**. Skills are now metadata-only by default, with body content loaded explicitly on demand in the registry/API/CLI/web flow. | Implemented via metadata-only `Skill` objects plus explicit content loading endpoint/registry path.       |

---

## 2. What to Build (Priority Order)

### P0: Lifecycle Hook System with Blocking/Modification

**Why first:** Hooks define **where** extensibility happens in AGH's runtime. They are a core concept (Go interfaces, function-pointer fields on structs) that doesn't require the extension architecture to be useful. Every subsequent system depends on hook points existing: the extension architecture adds Wasm/subprocess executors to existing hooks; the tool registry runs tools through the hook pipeline (`pre_tool_call` can block, `post_tool_call` can transform).

**Current state:** `HookRunner` runs subprocess hooks for 2 events. Results captured but ignored.

**Target state:**

- Extend `HookRunner` to support **structured responses**: `{continue: bool, updatedInput?, transformedResult?, reason?}`
- Add hook points: `session.pre_create`, `session.pre_prompt`, `event.post_record`, `agent.spawned`, `agent.crashed`, `session.pre_stop`
- Support 2 hook executor types initially (3rd added by extension architecture later):
  - **Subprocess hooks** (existing `HookRunner`, extend with structured output parsing)
  - **Go-native hooks** (typed function-pointer fields on Manager structs, nil-checked — GoClaw pattern that aligns with AGH's "no event bus" principle)
  - _(Future, via extension architecture)_ **Wasm hooks** for sandboxed in-process execution
- Pre-hooks can **block** (return `continue: false`) or **modify** (return `updatedInput`)
- Post-hooks can **transform** (return `transformedResult`) or trigger side effects
- Hook ordering: Go-native first (fastest, in-process), then subprocess, short-circuit on any deny

**Depends on:** Nothing. Pure core Go work.

**Techspec needed:** `techspec-lifecycle-hooks.md`

---

### P1: Extension Architecture Foundation

**Why:** Once hook points exist as core interfaces, the extension architecture provides the mechanism for **external code** (non-Go, third-party) to plug into those hook points and add new capabilities.

**Recommended: Three-tier hybrid** (validated by Terraform, VS Code, Grafana, Neovim, Claude Code patterns):

| Tier                | Mechanism                           | Use Case                                              | Language Support                            | Performance          |
| ------------------- | ----------------------------------- | ----------------------------------------------------- | ------------------------------------------- | -------------------- |
| **L1: Go-native**   | Go interfaces compiled in           | First-party core functionality                        | Go only                                     | Fastest (in-process) |
| **L2: WebAssembly** | Extism + wazero (pure Go, zero CGO) | Hooks, validators, transformers                       | Rust, Go, TS (AssemblyScript), C, 16+ langs | ~1-10us/call         |
| **L3: Subprocess**  | JSON-RPC over stdio                 | Agent drivers, memory backends, API extensions, tools | Any language                                | ~100-500us/call      |

**Key insight:** L3 is a generalization of AGH's existing ACP subprocess pattern (`internal/acp`). The extension protocol reuses the same launch-binary, JSON-RPC-over-stdio, graceful-shutdown lifecycle.

**Deliverables:**

- `internal/extension` package: `Manager`, `Registry`, manifest loading
- Extension manifest format (`extension.toml`)
- Subprocess extension lifecycle (reuses `internal/acp` patterns)
- Wasm runtime integration via Extism/wazero — registers as additional hook executors on existing hook points
- `agh extension list/install` CLI commands
- TypeScript SDK (`@agh/extension-sdk` npm package) for non-Go developers

**Depends on:** Lifecycle hooks (P0) — extension executors plug into existing hook points.

**Techspec needed:** `techspec-extension-architecture.md`

---

### P2: Unified Tool Interface & Registry

**Why:** Tools are AGH's primary extension point for end users. Currently they're untyped string lists.

**Current state:** `AgentPayload.Tools` is `[]string`. No schema, no central registry, no availability checking.

**Target state:**

```go
type ToolDriver interface {
    Name() string
    Description() string
    InputSchema() json.RawMessage
    IsReadOnly() bool
    CheckAvailability(ctx context.Context) bool
    Call(ctx context.Context, input json.RawMessage) (ToolResult, error)
}
```

- Central `ToolRegistry` in `internal/tools` (new package)
- Tool sources: **built-in** (Go-native L1), **MCP-proxied** (from skill declarations), **extension-provided** (L2/L3), **dynamic** (config-defined shell commands)
- Namespacing: `mcp__{server}__{tool}` for MCP tools, `ext__{name}__{tool}` for extension tools
- Availability gating: tools hidden from agent when deps missing (`CheckAvailability` returns false)
- Toolset composition: named groups (`coding`, `research`, `full_stack`) with enable/disable per session
- Hybrid search: when >40 tools, switch to BM25 search + lazy activation (GoClaw pattern) to save context tokens
- Tool execution goes through hook pipeline: `pre_tool_call` → execute → `post_tool_call`

**Depends on:** Lifecycle hooks (P0) for hook pipeline. Extension architecture (P1) for extension-provided tools.

**Techspec needed:** `techspec-tool-registry.md`

---

### P3: Session Stop Reason Taxonomy

**Why:** Precise terminal state classification for observability, debugging, billing. Currently binary (stopped vs not).

**Current state:** 4 states: `Starting → Active → Stopping → Stopped`. No reason tracking.

**Target state:**

```go
type StopReason string
const (
    StopCompleted      StopReason = "completed"       // Agent finished naturally
    StopUserCanceled   StopReason = "user_canceled"    // User called Stop()
    StopMaxIterations  StopReason = "max_iterations"   // Hit turn/iteration limit
    StopLoopDetected   StopReason = "loop_detected"    // Cycle detection triggered
    StopTimeout        StopReason = "timeout"          // Wall-clock timeout
    StopBudgetExceeded StopReason = "budget_exceeded"  // Token/cost budget hit
    StopError          StopReason = "error"            // Unrecoverable error
    StopAgentCrashed   StopReason = "agent_crashed"    // Subprocess died unexpectedly
)
```

- Add `StopReason` field to `Session`, `SessionInfo`, `SessionMeta`
- Classify in `finalizeStopped()` based on `waitErr`, context cancellation, explicit stop
- Persist in meta JSON and session_stopped event
- Surface in API responses and observe events
- Expose as `StoppedReason` in the global catalog for historical queries

**Depends on:** Nothing. Can be built independently.

**Techspec needed:** `techspec-session-stop-reasons.md` (small, may be combined with P4)

---

### P4: Session Repair on Load

**Why:** After daemon crash or unclean shutdown, sessions can have inconsistent state. Currently Resume() just re-reads and hopes.

**Current state:** `Resume()` reads `SessionMeta`, re-resolves workspace/agent, starts agent. No validation.

**Target state:**

- **Workspace validation:** Check `resolvedWorkspace.RootDir` still exists and is accessible
- **Agent validation:** Verify agent definition still present in config. If renamed/removed, return descriptive error
- **State consistency:** If meta says `active` but process is dead, transition to `stopped` with `StopReason = "agent_crashed"`
- **Event store integrity:** Quick check — last event timestamp reasonable, no zero-length DB file
- **Graceful degradation:** Each check fails independently with clear error messages, not a single opaque failure

**Depends on:** P3 (stop reasons) for proper classification of crash-recovered sessions.

**Techspec needed:** Combined with P3 as `techspec-session-resilience.md`

---

### P5: Loop/Recursion Guard

**Why:** Without this, a misconfigured agent can loop forever or recurse infinitely, burning tokens and blocking the daemon.

**Current state:** Zero protection. No depth limiting, no cycle detection.

**Target state:**

- **Iteration budget:** Configurable per-agent `max_iterations` (default: 200). Tracked in session. When exceeded → stop with `StopReason = "max_iterations"`.
- **Cycle detection:** SHA256 fingerprint of last N tool-call sequences. If pattern repeats K times → stop with `StopReason = "loop_detected"`. Configurable sensitivity.
- **Delegation depth:** If AGH ever supports agent-to-agent delegation (Phase 3), enforce `MAX_DEPTH` (default: 5) via context propagation.
- **Cross-cutting:** Implemented in `internal/session` as a guard that wraps the event recording path — every tool-call event increments the counter and checks patterns.

**Depends on:** P3 (stop reasons) for `StopReason` values.

**Techspec needed:** Combined with P3/P4 as `techspec-session-resilience.md`

---

### P6: Progressive Disclosure for Skills

**Why:** With dozens of skills, eagerly loading full content wastes context tokens. Only metadata should be in context; full body loads on demand.

**Status:** Implemented on 2026-04-09.

**Previous state:** `ParseSkillFile()` read complete content into `Skill.Content`, and API responses included full content.

**Implemented state:**

- `skills.Skill` is metadata-only; skill bodies are no longer retained on the loaded registry object.
- Registry content loading is explicit via `Registry.LoadContent(...)`, covering filesystem and bundled skills.
- Skill list/detail API responses are metadata-only.
- API endpoint `GET /api/skills/:name/content` provides explicit content retrieval.
- CLI `agh skill view` loads full content explicitly instead of relying on eager preload.
- Web skill detail loads full content only after an explicit user action.

**Depends on:** Nothing. Can be built independently.

**Techspec needed:** None for implementation. Follow-up documentation can be added separately if desired.

---

## 3. Techspec Map

```
  techspec-lifecycle-hooks (P0)        techspec-session-resilience (P3+P4+P5)
  [hook points, protocol,              [stop reasons, repair, loop guard]
   Go-native executors]                [independent]
         |
         v
  techspec-extension-architecture (P1)
  [Wasm + subprocess executors,         techspec-skill-progressive-disclosure (P6)
   manifest, SDK]                       [independent]
         |
         v
  techspec-tool-registry (P2)
  [ToolDriver, registry, namespacing]
```

**Parallelism:** P0 (hooks) and P3-P5 (session resilience) can start simultaneously — zero cross-dependencies. P6 (skill disclosure) can also be done at any time.

| Techspec                                | Covers                                                                                                    | Dependencies                            | Size Estimate             |
| --------------------------------------- | --------------------------------------------------------------------------------------------------------- | --------------------------------------- | ------------------------- |
| `techspec-lifecycle-hooks`              | Hook taxonomy, structured protocol, Go-native executors, blocking/modification, subprocess output parsing | None                                    | Medium                    |
| `techspec-extension-architecture`       | Three-tier model, manifest, Manager, Registry, subprocess protocol, Wasm runtime, TypeScript SDK design   | Lifecycle hooks                         | Large                     |
| `techspec-tool-registry`                | ToolDriver interface, registry, namespacing, availability gating, toolset composition, hybrid search      | Lifecycle hooks, Extension architecture | Medium                    |
| `techspec-session-resilience`           | Stop reasons, session repair, loop/recursion guard, iteration budgets                                     | None                                    | Medium                    |
| `techspec-skill-progressive-disclosure` | Lazy content loading, API changes, context injection changes                                              | None                                    | Implemented on 2026-04-09 |

---

## 4. Deferred (Not Needed Now)

These are validated as valuable by the analysis but explicitly deferred per project priorities:

| Feature                                     | Rationale for Deferral                                                                             |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| Permission cascade (beyond ACP passthrough) | Current approval mode works. More sophisticated cascade when multi-user or enterprise needs arise. |
| Budget enforcement (token/cost limits)      | Important but not blocking. Can layer in after observe tracks usage.                               |
| FTS5 cross-session search                   | Powerful for recall but not needed for core extensibility. Phase 2 memory work.                    |
| Static/dynamic prompt split                 | Optimization. ACP agents manage their own prompts.                                                 |
| Cron scheduler + event triggers             | Extension candidate once extension architecture exists. Not core.                                  |
| Channel adapters                            | Extension once extension architecture exists. Define interface later.                              |
| Agent-to-agent networking (A2A)             | Phase 3. Define `AgentPeer` interface later.                                                       |
| Knowledge graph memory backend              | Extension on top of memory system.                                                                 |
| Workflow engine                             | Extension composing session primitives.                                                            |
| Extension marketplace/registry              | After extension ecosystem grows enough to need discovery.                                          |

---

## 5. Key Architectural Decisions (From Research)

### Why JSON-RPC stdio (not gRPC) for subprocess extensions

- AGH already uses JSON-RPC stdio for ACP — same pattern, same code paths
- No protobuf toolchain requirement — lower barrier for non-Go extension authors
- Aligned with MCP/LSP ecosystem convergence
- HashiCorp go-plugin (gRPC) is a good reference but heavier than needed

### Why Wasm via Extism + wazero (not Go native plugins)

- Go native plugins: no Windows, CGO required, no unloading, no security isolation, exact build-env matching. Every major Go project has rejected them.
- wazero: **pure Go, zero CGO, zero dependencies**. Fits single-binary constraint perfectly.
- Extism: high-level SDK with 16+ host languages and 7+ guest PDKs
- Deny-by-default security: Wasm plugins can't access filesystem/network unless explicitly granted
- Plugin crash doesn't kill host process (Wasm trap handled by runtime)
- Single portable `.wasm` file distribution — no platform-specific builds

### Why both Wasm AND subprocess (not one or the other)

- **Wasm** for fast-path synchronous operations: hooks, validators, transformers (<1ms latency, sandboxed)
- **Subprocess** for rich stateful extensions: agent drivers, memory backends, API routes (full system access, any language)
- Different use cases, different trade-offs. Forcing everything into one model over-constrains either power or safety.

### Why TypeScript as the first non-Go language

- Largest developer community for AI/agent tooling
- `@agh/extension-sdk` (npm) for subprocess extensions — natural for Node.js developers
- AssemblyScript for Wasm hooks — TypeScript-like syntax that compiles to Wasm
- Lowers the barrier for the majority of potential extension authors

---

## 6. Patterns Validated Across All Frameworks

These patterns appeared in 4+ of the 6 analyzed frameworks, confirming they are **not framework-specific opinions but industry convergence**:

| Pattern                                              | Frameworks                                      | AGH Status                  |
| ---------------------------------------------------- | ----------------------------------------------- | --------------------------- |
| Uniform tool interface with JSON Schema              | All 6                                           | Missing                     |
| Skills as markdown with YAML frontmatter             | Claude Code, GoClaw, Hermes, Pi-Mono, OpenClaw  | Exists                      |
| 5-tier skill precedence (workspace > user > bundled) | Claude Code, GoClaw, OpenClaw, Pi-Mono          | Exists                      |
| Lifecycle hooks at named points                      | All 6                                           | Exists (limited)            |
| Hooks can block/modify (not just observe)            | Claude Code, Pi-Mono, OpenClaw, Hermes          | Missing                     |
| Approval flow for dangerous operations               | Claude Code, Hermes, OpenClaw, OpenFang         | Exists (ACP)                |
| MCP tool integration                                 | Claude Code, GoClaw, Hermes, OpenFang, OpenClaw | Exists (delegation)         |
| Manifest-first plugin discovery                      | OpenClaw, Pi-Mono, OpenFang                     | Missing                     |
| Session compaction/context management                | All 6                                           | Delegated to ACP            |
| Non-blocking fan-out for events                      | Claude Code, GoClaw, OpenClaw                   | Exists (notifier)           |
| Subprocess environment isolation for hooks           | Claude Code, GoClaw, OpenFang                   | Exists (`hookEnvAllowlist`) |
| Tool namespacing to prevent collisions               | Claude Code, GoClaw, OpenFang                   | Missing                     |
| Progressive disclosure (lazy skill loading)          | Claude Code, Pi-Mono, GoClaw                    | Implemented on 2026-04-09   |
| Health reporting per subsystem                       | GoClaw, OpenClaw, OpenFang                      | Partial                     |
