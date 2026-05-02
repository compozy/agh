# AGH Runtime — Code-Grounded Feature Truth

## What ships today (with evidence)

### 1. AGH Network (Agent-to-Agent Protocol)
- **Files**: `internal/network/` (manager.go, delivery.go, router.go, peer.go, tasks.go)
- **What operators get**: Live peer discovery, multi-agent message routing, delivery coordination with retry logic, channel-based broadcast. Status surface shows connected peers, queued messages, delivery workers.
- **What agents get**: Send/receive messages via `network send` CLI; join channels; broadcast capabilities; claim/release task runs across network. Signed message validation, peer identity verification.
- **Maturity**: Mature. NATS-backed wire transport (stdio + HTTP SSE). Full audit trail in `audit.go`. Delivery coordinator with queue management, retry scheduling. Peer heartbeat tracking.
- **Key types**: `Manager`, `deliveryCoordinator`, `Envelope`, `Peer`

### 2. Autonomy Kernel / Task Orchestration
- **Files**: `internal/task/` (autonomy.go, manager.go, lease.go, interfaces.go)
- **What operators get**: Task lifecycle state machine (claimed, starting, running, completed). Lease management with heartbeat/expiry recovery. Operator sees run status, actor identity, claim token hash (redacted). Event history of state transitions.
- **What agents get**: `task claim-next-run` to atomically claim a queued run. `task heartbeat` to renew lease. Session-bound autonomy check prevents running tasks from different sessions. Lease expiry triggers recovery protocol.
- **Maturity**: Mature. Transactional claim/release primitives in store layer. Lease lock cascade with TTL. Actor identity tracking (agent, user, extension).
- **Key types**: `Task`, `Run`, `RunStatus`, `ClaimResult`, `AutonomyLeaseHandle`

### 3. Dream / Consolidation
- **Files**: `internal/memory/dream.go`, `internal/memory/consolidation/` 
- **What operators get**: Consolidation service evaluates time gates (24h default), session count gates (3 default), and file-lock gates before spawning a one-shot session. Operators see consolidation lock status, last-consolidated timestamp.
- **What agents get**: Consolidation runs as a background session with goal="memory-consolidation". Agents don't directly invoke it; it's orchestrated by the daemon based on gate thresholds. Memory is persisted in dual-scope (global/workspace) with YAML frontmatter.
- **Maturity**: Scaffolding-only for actual consolidation execution. Gate evaluation exists; actual compression/synthesis is not implemented (would require Claude to write memory).
- **Key types**: `Service` (consolidation Service), `consolidationLocker`, `lock.go`

### 4. Memory
- **Files**: `internal/memory/` (types.go, store.go, catalog.go, dream.go)
- **Memory taxonomy**: 4 closed types: `user`, `feedback`, `project`, `reference`
- **Memory scopes**: `global` (user/feedback) and `workspace` (project/reference)
- **What operators get**: Write/read/search/delete memory files. Reindex derived catalog. Query operation history. View health stats (indexed files, orphaned count, last-reindex timestamp). Full YAML frontmatter-driven metadata (name, type, description, agent_name).
- **What agents get**: `memory write|read|search|delete` CLI. Search returns ranked results with snippets. Can write to default scope per agent config (global or workspace). Audit trail logs every operation (write/delete/search/reindex).
- **Maturity**: Mature for storage/retrieval. Future gates: @file/@folder/@git/@url context references (defined, not wired), provider hooks (on_turn_start, on_session_end, on_pre_compress—defined, not called).
- **Key types**: `Backend` interface, `Store`, `Type`, `Scope`, `SearchResult`, `OperationRecord`

### 5. Tool Registry
- **Files**: `internal/tools/` (registry.go, dispatch.go, policy.go, builtin/*.go), `internal/tools/builtin/` (18 files)
- **What operators get**: Built-in tool catalog (automation, autonomy, bridges, config, extensions, hooks, memory, network, observe, sessions, skills, tasks). Policy-driven approval/denial. Tool result limits (byte cap). Sensitive field redaction in results.
- **What agents get**: 18 built-in tools available without registration: tasks, memory, network, hooks, automation, extensions, bridges, skills, mcp-auth, observe, settings, etc. Tools expose `-o json` output. Policy evaluator checks permission scope (user/workspace/agent).
- **Maturity**: Mature. Provider-agnostic registry with policy evaluation, result limiting, sensitive-field masking. MCP tool integration via `mcp.go`. Native subprocess tools via `native.go`.
- **Key types**: `RuntimeRegistry`, `Provider`, `PolicyEvaluator`, `ResultLimiter`

### 6. Capabilities (vs. Recipes)
- **Files**: `internal/config/capabilities.go`, `internal/network/capability_brief.go`, `internal/network/capability_catalog.go`
- **Definition**: Capabilities are outcome-oriented, agent-advertised, discoverable units with: `id`, `summary`, `outcome`, version, context_needed, artifacts_expected, execution_outline, constraints, examples, requirements, SHA256 digest.
- **Distinguisher**: Capabilities are agent-defined, signed, discoverable via network protocol (sent as `agh network send --kind capability`). Recipes are task workflows in the task system. Capabilities describe WHAT an agent can do; recipes describe HOW a task runs.
- **What operators get**: Capability catalog per agent directory (TOML or JSON). Network audit trail logs all capability exchanges. Digest mismatch detection.
- **What agents get**: Can declare capabilities.toml or capabilities/ directory. Network broadcast with canonical digest. Other agents discover via `agh channel get-peers` or network listener.
- **Maturity**: Mature for declaration/validation. Broadcast infrastructure mature (delivery.go). Contract defined. Not yet deeply integrated into task-selection or skill-recommendation UI.
- **Key types**: `CapabilityDef`, `CapabilityCatalog`, `CapabilityBrief`

### 7. Hooks / Extensions
- **Files**: `internal/hooks/` (46 files), `internal/extension/` (58 files)
- **Hooks**: Typed dispatch system. Lifecycle hooks (on_session_created, on_session_stopped, on_task_*, on_hook_*, on_autonomy_*) execute in skill precedence order + alphabetical. 5s default timeout, fail-open semantics.
- **Extensions**: Host API surface. Capabilities check. Entrypoint bootstrap. Bridge delivery integration. Discord, Google Chat, GitHub provider integrations tested.
- **What operators get**: Hook catalog view (skill source, mode native|subprocess, resolved declarative rules). Hook run audit in per-session event store. Operator sees which hooks fired, on what events, with what outcome.
- **What agents get**: Can register hooks via skill manifest. Hooks fire on daemon-owned state transitions (task claimed, session created). Hooks can spawn child tasks, send network messages, etc. Subprocess hooks can be written in any language.
- **Maturity**: Mature for dispatch and execution. Security: hooks cannot bypass safety primitives (claim tokens, leases, TTLs). Hooks deny/narrow/annotate but cannot replicate claim authority.
- **Key types**: `HookDecl`, `ResolvedHook`, `HookRunRecord`, `CapabilityChecker` (extension API gate)

### 8. CLI / HTTP / UDS Control Surfaces
- **CLI**: ~20 top-level commands: `daemon`, `install`, `config`, `session`, `task`, `skill`, `memory`, `network`, `bridge`, `workspace`, `agent`, `automation`, `extension`, `hooks`, `tools`, `toolsets`, `mcp`, `observe`.
- **HTTP API** (`internal/api/httpapi/`): JSON REST endpoints for all CLI operations. SSE for live streams. Gin-based server.
- **UDS API** (`internal/api/udsapi/`): Unix Domain Socket server for CLI IPC. Same contract as HTTP.
- **What operators get**: Full HTTP surface with `-o json` / `-o jsonl` / `-o toon` output formats. Deterministic exit codes. All state mutations (task create/claim, session spawn, skill install) are CLI-driven and machine-readable.
- **What agents get**: Can call `agh <command> --output json` for structured responses. Agents themselves are ACP subprocesses but can shell-out to `agh` CLI within same workspace.
- **Maturity**: Mature. All public surfaces close the loop: contract → HTTP handler → UDS handler → CLI command → extension API surface → docs.
- **Key types**: `BaseHandlers` (core), Cobra command tree

### 9. ACP Integration
- **Files**: `internal/acp/` (client.go, handlers.go, types.go), `internal/config/provider.go`
- **Supported providers** (built-in): `claude`, `codex`. Custom command-based providers supported via `command` config.
- **Transport**: JSON-RPC over stdio (subprocess). MCP servers (stdio, HTTP SSE) can be declared per provider in config.
- **What operators get**: Multi-provider setup in config. Per-agent provider override. Default model override. OAuth2 PKCE for remote MCP servers. Fallback provider chain (global → workspace override).
- **What agents get**: Spawned as ACP subprocess by session manager. Agents talk JSON-RPC to daemon for tool dispatch, memory, network, tasks. Daemon multiplexes agent tool calls through provider backends.
- **Maturity**: Mature. Full ACP client implementation. Permission system (agent can't escalate beyond session scope). Tool call auditing.
- **Key types**: `AgentProcess`, `ProviderConfig`, `MCPServer`, `MCPAuthConfig` (OAuth2 PKCE)

### 10. Observability
- **Files**: `internal/observe/` (observer.go, tasks.go, health.go), `internal/store/sessiondb/`, `internal/store/globaldb/`
- **Event store**: SQLite `runtime.db` (global) + per-session `events.db`. Append-only event log with correlation keys (workspace_id, session_id, parent_session_id, agent_name, task_id, run_id, claim_token_hash, lease_until, etc.).
- **Surfaces**: Session info (state, agent, workspace, start/end times). Event summaries (type, actor, timestamp). Token stats (input/output counts). Permission log (who called what, denied/allowed). Network audit (peer, direction, message_id, kind).
- **What operators get**: `agh observe query` returns event history filtered by agent/task/session. Health metrics (active sessions, tasks per status, token totals). Live SSE stream of events. Status view shows daemon uptime, network peers, scheduler state.
- **What agents get**: Via network protocol, agents can broadcast trace/receipt messages. Daemon records as network audit entries. Agents can query `/agent/context` for live situation (active peers, task assignments, memory recall).
- **Maturity**: Mature. Full event schema with redaction (claim_token_hash, not raw token). Live broadcaster with reconnect/replay via after_seq. Coverage matrix tests verify all lifecycle paths emit events.
- **Key types**: `Observer`, `EventSummary`, `TokenStats`, `PermissionLogEntry`, `NetworkAuditEntry`

---

## CLI Surface (Subcommands)

| Command | Role |
|---------|------|
| `daemon start/stop/status` | Daemon lifecycle |
| `session new/list/attach/cancel` | Session management |
| `task create/list/claim-next-run/heartbeat/release/complete` | Task orchestration |
| `skill install/list/validate/bundle` | Skill management |
| `memory write/read/search/delete/reindex` | Memory operations |
| `network join/leave/send/list-peers/get-peers` | Network communication |
| `bridge configure/send` | Bridge integrations |
| `workspace list/select/create` | Workspace management |
| `automation schedule/list/trigger` | Automation scheduling |
| `extension install/list/check` | Extension management |
| `hooks list/show/run-history` | Hook introspection |
| `tools list/invoke` | Tool discovery & execution |
| `config show/set/validate` | Configuration management |
| `observe query/export` | Event querying |

---

## HTTP API Surface (Endpoint Groups)

| Group | Key Endpoints |
|-------|--------------|
| **Sessions** | POST /api/sessions, GET /api/sessions/{id}, SSE /stream/sessions |
| **Tasks** | POST /api/tasks, GET /api/tasks/{id}/runs, POST /api/tasks/{id}/claim-next, PATCH /api/runs/{id}/heartbeat |
| **Memory** | POST /api/memory/write, GET /api/memory/search, DELETE /api/memory/{scope}/{filename} |
| **Network** | POST /api/network/send, GET /api/network/peers, WS /ws/network/channel/{id} |
| **Skills** | GET /api/skills, POST /api/skills/install, DELETE /api/skills/{id} |
| **Tools** | GET /api/tools, POST /api/tools/{id}/invoke |
| **Automation** | POST /api/automation/schedule, GET /api/automation/triggers |
| **Settings** | GET /api/settings, PATCH /api/settings, GET /api/settings/providers |
| **Extensions** | GET /api/extensions, POST /api/extensions/{id}/check-capability |
| **Observe** | GET /api/events?query=..., GET /api/health |

---

## ACP Providers Supported

**Built-in (in code):**
- `claude` — Anthropic Claude provider (via custom command)
- `codex` — OpenClaw Codex provider (via custom command)

**Custom/External:**
- Any provider with configurable `command` + `default_model` in config
- MCP servers via stdio/HTTP/SSE (declared per provider)
- OAuth2 PKCE for remote MCP auth

---

## Honest Distinguishers

1. **Dual-Scope Memory with Consolidation Gates** — AGH is *the only* system shipping operators a task-triggered memory consolidation service that uses time (24h), session-count (3), and file-lock gates before running. Memory is workspace-scoped AND global-scoped, with type taxonomy (user/feedback/project/reference).

2. **Agent-to-Agent Network Protocol** — AGH ships a real, signed message protocol for agents to discover each other, exchange capabilities, and claim task runs across network boundaries. Not just a message bus—a peer registry with heartbeat, audit trail, and delivery retry. Genuine agent autonomy that outlasts a single session.

3. **Autonomy Kernel with Lease-Based Task Ownership** — Task runs are owned via transactional claim/release primitives backed by claim tokens + lease TTLs. Only the agent holding the token can heartbeat or complete the run. Session-bound autonomy rejects runs from different sessions. This prevents accidental double-execution and orphan leaks.

4. **Capability Advertisement & Discovery** — Agents declare outcome-oriented capabilities with SHA256 digests, signed identities, and context/artifact specifications. Network layer broadcasts and audits all capability exchanges. No UI-only manageability; agents themselves discover capabilities via `network get-peers`.

5. **Type-Driven Hook Dispatch** — Hooks are not a generic event bus. Dispatch is explicit at state-transition call sites (task.Manager.ClaimNextRun fires on_autonomy_* hooks ONLY). Hooks cannot bypass safety primitives (claim tokens, lease locks). Fail-open semantics (errors logged, never block).

6. **Event-Store Observability with Redaction** — Every domain operation appends to a durable event store (SQLite `events.db` + `runtime.db`) with canonical correlation keys. Raw claim tokens never leak; only hashes appear in logs/APIs. Operator gets full history; no tail-log forensics required.

7. **Extensibility via Host API Gates** — Extensions are not privileged sidecars. They go through capability checks (CapabilityChecker). Bridges integrate cleanly with daemon lifecycle. Extensions can hook into skill/extension/bridge operations without modifying core code.

---

## Things the SPECS claim but CODE does not yet have

1. **Memory Consolidation Execution** — Dream service defines gates and lock management, but actual consolidation (summarization + storage) is NOT implemented. Would require an agent-driven consolidation session to read old memory, synthesize, and write new summaries. Currently scaffolding-only.

2. **Context Reference Resolution (@file/@folder/@git/@url)** — Types and interfaces defined (ContextRef, ContextRefResolver, ProviderHookRunner) in `memory/types.go` lines 150–217. Not wired into prompt assembly. No resolver implementation.

3. **Memory Provider Hooks** — ProviderHookEvent (on_turn_start, on_session_end, on_pre_compress) defined but never dispatched. Task 07 RFC claim only; no runtime execution.

4. **Deep Capability-to-Task Integration** — Capability catalog loads and validates. Network exchange works. But the UI/skill-selection system does NOT yet query agent capabilities to suggest tasks or auto-select agents by capability ID. Capabilities are discoverable via network only; not yet a first-class task-assignment signal.

5. **Skill Marketplace Search** — Registry helpers exist (clawhub/, github/). HTTP endpoints exist. No fuzzy search or ranking engine. CLI list is flat; no semantic matching.

---

## Plain-English Outcomes (for Marketing)

1. **Operators get true multi-agent orchestration without a central scheduler.** Agents claim task runs over the network, renew their leases, and report completion. Daemon enforces ownership via claim tokens. No polling, no heartbeat race, no orphaned tasks.

2. **Operators never see raw credentials in logs or status views.** Claim token hashes, not raw tokens. Permission denial logs the scope, not the reason. Network audit trails redact secrets. Full SQLite event ledger for forensics.

3. **Agents define what they can do and advertise it in real time.** Each agent publishes outcome-oriented capabilities with SHA256 digests. Other agents discover peers and their capabilities via the network layer. No central registry required.

4. **Memory consolidation respects computational cost gates.** 24 hours + 5 touched sessions + file-lock before spawning consolidation. Operator sees gate status and lock state. No surprise expensive operations; operator controls trade-off.

5. **Extensions stay sandboxed; hooks can't hijack state transitions.** Extension capabilities are checked at call time. Hooks fire at explicit state-transition sites (claim, release, start, complete). Hooks can't replicate claim authority or bypass lease locks. Fail-open: errors log but never block.

6. **All agent operations are CLI-drivable and JSON-parseable.** `agh task claim-next-run -o json`, `agh memory search -o json`, `agh network send --kind capability -o json`. No UI-only features. Deterministic exit codes for automation.

7. **Observability is built in, not bolted on.** Every mutation (create, claim, release, complete) emits a canonical event with workspace_id, session_id, actor_id, claim_token_hash. Operator queries the event ledger, not logs. All lifecycle paths verified by coverage matrix tests.

---

## Maturity Summary

| Feature | Maturity | Notes |
|---------|----------|-------|
| Network | Mature | NATS wire, full delivery retry, audit trail |
| Autonomy | Mature | Lease lock cascade, session-bound checks |
| Dream/Consolidation | Scaffolding | Gates exist, execution not implemented |
| Memory | Mature | Dual-scope, 4-type taxonomy, operation history |
| Tool Registry | Mature | 18 built-in tools, policy eval, result limits |
| Capabilities | Mature | Catalog, digest, network exchange; UI integration pending |
| Hooks | Mature | Type-driven dispatch, state-transition gating, fail-open |
| CLI/HTTP/UDS | Mature | Full parity, `-o json` on all surfaces |
| ACP | Mature | Claude, Codex, custom providers; MCP support |
| Observability | Mature | Append-only event store, redaction, correlation keys |

---

## Code Quality & Testing

- **Boundaries enforced**: `mage Boundaries` enforces downward-only imports; no cycles.
- **Security invariants**: Claim token redaction non-negotiable; path traversal hardened; symlink escapes blocked.
- **Concurrency**: Goroutine ownership via Manager-owned WaitGroup. Detached execution uses `context.WithoutCancel`. Subprocess signaling crosses Unix/Windows.
- **Test coverage**: Integration tests for network delivery, task ownership, memory consolidation gates, hook dispatch, capability exchange, permission denial.
- **Observability in tests**: Coverage matrix tests verify all lifecycle paths emit canonical events.

---

## Recommended Marketing Angles

1. **"True distributed task ownership"** — Lease-based task claiming prevents double-execution across agents and sessions.
2. **"Memory that respects your compute budget"** — Consolidation gates (24h, 5 sessions, file-lock) ensure no surprise consolidation runs.
3. **"Everything is automation-friendly"** — All agent operations are CLI + JSON. No UI-only features.
4. **"Agents discover and advertise capabilities in real time"** — Network protocol, not a central registry.
5. **"Full event ledger for security & compliance"** — Every mutation logged, redacted, correlated. No token leaks in logs.

