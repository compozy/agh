# Auto-Spawn, Delegation & Dynamic Agent Creation — Gap Analysis

> **Slice:** can a running agent decompose a sub-task, create a sub-agent
> with a custom role/prompt, coordinate with it on a channel, and reclaim
> resources when it finishes? **Today: no — there is zero agent-initiated
> spawn surface.**

---

## 1. TL;DR

- AGH does spawn ACP subprocesses, but **only the daemon does** — sessions
  are created from outside (CLI, HTTP, automation, dream consolidation,
  bridges, extensions). The session itself, and therefore the agent
  running inside it, has **no callable surface to spawn another session**
  (`internal/session/manager_lifecycle.go:20`,
  `internal/session/manager.go:39`).
- `session.CreateOpts` knows nothing about a parent: no `ParentSessionID`,
  no role override, no budget, no lifetime. The only session "type"
  classifier is `user | dream | system` (`internal/session/session.go:36`).
- Sub-agent role/prompt assembly is fully driven by `agent.Prompt` plus
  the workspace assembler at startup (`internal/session/manager_start.go:280`).
  There is no path for an agent to **inject an ad-hoc system prompt** for
  a freshly spawned child.
- Coordination already exists at the channel layer (`internal/network/peer.go`,
  `internal/session/network_peer.go`) and `AGH_SESSION_CHANNEL` /
  `AGH_PEER_ID` env vars are wired (`internal/session/manager_start.go:417-427`),
  but *agents cannot pick a channel for a spawned child* because they
  cannot spawn at all.
- Lifecycle, supervision, and reclaim are mature for daemon-initiated
  sessions (`finalizeStopped` at `internal/session/manager_lifecycle.go:145`),
  but there is **no parent→child link, no recursion guard, no per-tree
  budget, no reaper for orphaned children.** `internal/network/peer.go`
  treats every `LocalPeer` as flat siblings.
- The closest precedent in-tree is `automation.Dispatcher.dispatchAttempt`
  (`internal/automation/dispatch.go:367`) and `memory/consolidation`'s
  `spawnSession` (`internal/memory/consolidation/runtime.go:410`):
  trigger → `Sessions.Create` → `Sessions.Prompt` → `Sessions.Stop`.
  Both run **inside the daemon**, not from an agent's context.
- Bottom line: the substrate (manager, ACP driver, network peers, task
  parent/child linkage in `internal/task/manager.go:237`) is rich enough
  that an "agent-initiated spawn" tool is mostly a wiring + safety job,
  not a new-runtime job.

---

## 2. Current Spawn Model — Who Can Create a Session?

### 2.1 Single create entrypoint

All session creation funnels through one method:

- `(*Manager).Create(ctx, opts CreateOpts) (*Session, error)` at
  `internal/session/manager_lifecycle.go:20` →
  `prepareCreateStart` (`internal/session/manager_start.go:56`) →
  `startSession` (`internal/session/manager_start.go:129`) →
  `m.driver.Start(ctx, startOpts)` invoked at
  `internal/session/manager_start.go:221`.

`CreateOpts` is intentionally tiny and *parent-free*:

```go
// internal/session/manager.go:39
type CreateOpts struct {
    AgentName     string
    Provider      string
    Name          string
    Workspace     string
    WorkspacePath string
    Channel       string
    Type          Type   // user | dream | system
}
```

There is no `ParentSessionID`, no `SystemPromptOverlay`, no
`Budget`, no `TTL`, no `RoleOverride`.

### 2.2 Existing callers (all *daemon-side*, none *agent-side*)

| Caller | File / line | Trigger | Type |
|---|---|---|---|
| CLI `agh session new` | `internal/cli/session.go:67` | human user | user |
| HTTP `POST /api/sessions` | `internal/api/httpapi/routes.go:67` | human user / programmatic API client | user |
| Network channel bootstrap | `internal/api/core/network_details.go:584` | channel creation seeds N session peers | user |
| Automation dispatcher | `internal/automation/dispatch.go:399` | trigger / cron / extension event | system |
| Dream consolidation | `internal/memory/consolidation/runtime.go:426` | scheduled memory consolidation | dream |
| Extension host RPC `sessions.create` | `internal/extension/host_api.go:760` | extension subprocess (out-of-band) | system |
| Bridge runtime | `internal/extension/host_api_bridges.go:673` | inbound bridge message needs a session | user |

**Critical observation:** every call site is a daemon-internal Go
package (or an external IPC path mediated by the daemon). The session
process itself — i.e., the running ACP agent — has **no inbound RPC
channel back to `Manager.Create`**. The agent only speaks to the daemon
through `acp` (its own JSON-RPC stream over stdio, handled by
`internal/acp/handlers.go`) and through whatever tools / MCP servers
the daemon attaches at start. None of those tools today wraps
`Manager.Create`.

### 2.3 Lifecycle that *does* exist (use as the reclaim template)

`finalizeStopped` (`internal/session/manager_lifecycle.go:145`) already
handles the full reclaim arc:

1. `claimOrWaitFinalization` — single-writer barrier.
2. `beginStoppingSession` → `persistStopClassification` →
   `recordProcessExitEvent` → `recordSessionStoppedEvent`.
3. `dispatchAgentStopped` hook fan-out.
4. `finalizeEnvironment` — environment teardown
   (`sandbox.SyncReasonStop` / `SyncReasonCrash`).
5. `closeSessionRecorder` → `markSessionStopped` → `leaveSessionNetwork`.
6. `removeActive` and `notifier.OnSessionStopped`.

`watchProcess` at `internal/session/manager_lifecycle.go:104` already
catches PID exit and calls `handleProcessExit`. So *given a child session
exists*, every reclaim primitive is in place. The gap is upstream:
**nothing creates that child on the agent's behalf.**

### 2.4 Concurrency / quota

- `Manager.reserve` (`internal/session/manager.go:468`) enforces
  `effectiveMaxSessions` (`internal/session/manager_helpers.go:67`).
  Default ceiling: `LimitsConfig.MaxSessions = 10`
  (`internal/config/config.go:52`, default at line 397). This cap is
  *daemon-global*, not per-tree, not per-workspace, not per-parent.
- Automation has its own concurrency gate (`Dispatcher.tryAcquire` at
  `internal/automation/dispatch.go:947`), but no equivalent exists for
  agent-initiated spawn (because that path doesn't exist).

---

## 3. The Autonomy Gap

### 3.1 Can an agent create another session?

**No.** The agent process talks to the daemon via the ACP JSON-RPC
methods registered in `internal/acp/handlers.go` (filesystem, terminal,
permissions). There is **no `session.spawn` ACP method, no MCP tool, no
HTTP loopback exposed back into the agent's tool surface**. The host API
in `internal/extension/host_api.go:760` *is* a `sessions.create` RPC,
but it is gated to **extension subprocesses**, not to in-session ACP
agents — extensions are a separate trust principal with their own
handler (`HostAPIHandler`) attached to a different transport than the
agent's stdio channel.

### 3.2 Can an agent supply an ad-hoc role / system prompt?

**No.** Prompt assembly happens once during `prepareSessionStartRuntime`
(`internal/session/manager_start.go:270`):

1. `m.resolveWorkspaceAgent(spec.agentName, …)` — load definition by
   name (must already be a registered agent on disk).
2. `m.startupPrompt(...)` runs the configured
   `PromptAssembler` / `StartupPromptAssembler`.
3. Optional `m.startupOverlay.Apply(...)`.
4. The fully-baked string is shoved into
   `agentDef.Prompt` and passed via `acp.StartOpts.SystemPrompt`
   (`internal/session/manager_start.go:404`).

There is no path for a parent agent to say "spawn me a researcher with
*this* one-shot system prompt and *only* the read tools." The closest
proxy is to register a new agent definition on disk before calling
`Create`, which is workflow-grade plumbing, not an in-flight tool call.

### 3.3 Can a parent and child coordinate on a channel?

**Partially.** `CreateOpts.Channel` exists (`internal/session/manager.go:46`)
and joining the network registry is wired (`joinNetworkPeer` in
`manager_helpers.go`, env vars exported at
`internal/session/manager_start.go:417-427`). So *if* the spawn surface
existed, the parent could pass its current `AGH_SESSION_CHANNEL` and the
child would land on the same `LocalPeer` group
(`internal/network/peer.go:124`). But:

- `LocalPeer` has no `ParentSessionID` field
  (`internal/network/peer.go:14`); the topology is flat.
- `network.Manager.Send` (`internal/network/manager.go:560`) routes by
  `peer_id`, with no notion of "reply only to my parent" or
  "broadcast only to my children."
- `whois`/discovery (`PeerRegistry.MatchLocalPeers` at
  `internal/network/peer.go:276`) cannot filter by spawn lineage.

### 3.4 Budgets, lifetime, supervision, reclaim

- **Budget:** none. `aghconfig.LimitsConfig` is daemon-global. There is
  no per-spawn-tree token cap, iteration cap, wall-clock cap, or cost
  cap. Compare hermes `delegation.max_iterations`
  (`.resources/hermes/run_agent.py:218`) and
  `delegation.child_timeout_seconds`
  (`.resources/hermes/tools/delegate_tool.py:363`).
- **Lifetime:** sessions live until `Stop`/process exit. There is no
  TTL or auto-stop-on-parent-stop. If the parent stops, orphan children
  remain active.
- **Supervision:** `aghconfig.SessionSupervisionConfig`
  (`internal/config/config.go:67`) covers heartbeat / inactivity /
  cancel grace, but it is identical for every session — no
  parent-aware policy.
- **Reclaim:** the daemon-global reaper hook
  (`internal/daemon/orphan.go`) cleans crashed sessions on boot but has
  no notion of "this session was spawned by SESS-X; if SESS-X is gone,
  collect it."

### 3.5 Existing parent/child precedent — but in `task`, not `session`

`internal/task/manager.go:237` already implements a clean
`CreateChildTask` with parent linkage and audit events
(`taskEventChildCreated`). That's the right shape — but tasks are
durable work-items, not running agent processes. The autonomous-spawn
slice needs the same shape applied to **`Session` itself**.

---

## 4. Reference Comparisons

### 4.1 Claude Code — `Agent` / `Task` tool

`/.resources/claude-code/tools/AgentTool/AgentTool.tsx:81-138` defines
the spawn tool the LLM can call:

```ts
description, prompt, subagent_type?, model?, run_in_background?,
name?, team_name?, mode?, isolation?, cwd?
```

Key patterns we lack:
- `subagent_type` + a registered catalog of agent definitions resolved
  at call time.
- `isolation: "worktree"` for filesystem isolation of children — they
  run in a temporary git worktree
  (`/.resources/claude-code/tools/AgentTool/forkSubagent.ts`).
- `run_in_background: true` so the parent doesn't block; results arrive
  as `<task-notification>` XML in the next user turn (see
  `docs/ideas/from-claude-code/analysis_multi_agent.md:301`).
- Anti-recursion guard: detects `<fork-boilerplate>` in conversation
  history and rejects further forks at call time.
- Context inheritance for forks: parent's full message history reused
  for prompt-cache sharing.

### 4.2 Hermes — `delegate_task` Python tool

`/.resources/hermes/tools/delegate_tool.py:1-200` is the cleanest
reference for the *ground rules* a Go port should adopt:

- `DELEGATE_BLOCKED_TOOLS = frozenset(["delegate_task", "clarify",
  "memory", "send_message", "execute_code"])` — children **never** get
  the delegate tool itself. This breaks the recursion fork-bomb at the
  root.
- `MAX_DEPTH = 1` default, configurable via
  `delegation.max_spawn_depth` up to `_MAX_SPAWN_DEPTH_CAP = 3`.
- `_get_max_concurrent_children() → 3` default, capped per-call.
- `_get_child_timeout() → 600s` hard wall-clock cap per child.
- Active subagent registry (`_active_subagents` map at line 151) +
  `interrupt_subagent(subagent_id)` so the parent (or operator) can
  pause/kill mid-flight.
- Pause-spawn switch (`set_spawn_paused`) so an operator can quench all
  new spawns without killing in-flight ones.
- `_build_child_system_prompt` (line 534) renders a focused single-task
  prompt that *appends* an "Orchestrator role" block when the child is
  itself allowed to delegate (capped by depth).
- `_subagent_auto_deny` callback — children that hit a permission prompt
  in a worker thread auto-deny instead of deadlocking on stdin.

### 4.3 OpenClaw — `sessions_spawn` tool

`/.resources/openclaw/src/agents/subagent-system-prompt.ts:4-112` and
`/.resources/openclaw/src/config/agent-limits.ts:1-23` show the same
shape in TypeScript:

- `DEFAULT_SUBAGENT_MAX_CHILDREN_PER_AGENT = 5`,
  `DEFAULT_SUBAGENT_MAX_SPAWN_DEPTH = 1`,
  `DEFAULT_SUBAGENT_MAX_CONCURRENT = 8`.
- The spawned child gets a system prompt that explicitly enumerates:
  "You are a subagent", "Stay focused", "Be ephemeral", "Don't initiate".
- Push-based result delivery: completion events auto-announce to the
  parent's session (no polling).
- Two runtime modes: `runtime: "subagent"` (in-tree managed child) vs
  `runtime: "acp"` (a new ACP agent). Useful split for AGH because we
  already have the ACP path.

### 4.4 Multica

Multica's daemon model is much closer to ours (Go + per-session
process), but it does not implement agent-initiated spawning either —
spawning is admin/automation territory. Useful as a *negative* control:
confirms that this surface is genuinely missing in the broader
ecosystem, not just AGH.

### 4.5 sandbox-agent

The `sandbox-agent` repo has no spawn primitive but its sandboxing
research (`/.resources/sandbox-agent/research/detect-sandbox.md`,
`process-terminal-design.md`) is the right reference once we have a
spawn API and need to confine child filesystem/network reach. It pairs
well with the `isolation: "worktree"` Claude Code pattern.

---

## 5. Concrete Proposals

> Naming convention: **(EXISTS)** = already in tree, **(PROPOSED)** = new.

### 5.1 Extend `session.CreateOpts` with parent + role override

```go
// internal/session/manager.go — extension (PROPOSED)
type CreateOpts struct {
    // existing fields …
    ParentSessionID  string                 // PROPOSED — empty = root
    SpawnRole        *SpawnRole             // PROPOSED — ad-hoc role
    Budget           *SpawnBudget           // PROPOSED
    TTL              time.Duration          // PROPOSED — auto-stop after
    AutoStopOnParent bool                   // PROPOSED — reclaim with parent
}

type SpawnRole struct {
    BaseAgentName  string                  // resolves an existing agent def
    PromptOverlay  string                  // appended to baked prompt
    Permissions    aghconfig.PermissionMode // narrow only, never widen
    AllowedTools   []string                // optional MCP/tool allowlist
    AllowedSkills  []string                // narrow skill registry
}

type SpawnBudget struct {
    MaxTurns        int
    MaxWallClock    time.Duration
    MaxTokens       int64                  // honored by ACP driver if available
}
```

Add a new constant `SessionTypeSpawned Type = "spawned"`
(`internal/session/session.go:36`) so observability can distinguish
ad-hoc children from `user`/`system`/`dream` sessions and so dashboards
can filter the spawn tree.

### 5.2 New `session.Manager.Spawn` method

```go
// internal/session/manager_lifecycle.go (PROPOSED)
func (m *Manager) Spawn(
    ctx context.Context,
    parentID string,
    opts SpawnOpts,
) (*Session, error)
```

Internally:
1. Validate `parentID` exists, is `StateActive`, and is not itself a
   leaf-only child (depth check).
2. Compose a `CreateOpts` with `ParentSessionID = parentID`, propagate
   `Channel` from parent (or use override), inherit `Workspace`.
3. Apply `SpawnRole` by *narrowing* (never widening) `Permissions` and
   `AllowedTools`; reject any attempt to set `PermissionModeApproveAll`
   from a non-system parent.
4. Call existing `startSession` plumbing — almost zero new path.
5. Record `parent_id` in `meta.json` (extend
   `store.SessionMeta` and `store/schema.go` — currently no field).
6. Emit `EventTypeSessionSpawned` (PROPOSED) with `parent_id` and
   `spawn_depth` so the per-session event stream supports tree
   reconstruction (mirrors `EventTypeSessionStopped` shape in
   `internal/session/manager_lifecycle.go:301`).

### 5.3 ACP-side surface — make `session.spawn` an ACP method

The agent talks JSON-RPC via the handlers in
`internal/acp/handlers.go`. Add a new method:

```jsonrpc
session/spawn  {parent_session_id, role, budget?, channel?, prompt}
  -> {session_id, peer_id, channel}
session/send   {target_session_id, payload}
  -> {message_id}
session/wait   {session_id, timeout?}
  -> {final_event, transcript_excerpt}
session/stop   {session_id, reason?}
```

This keeps with the ACP model already used for filesystem / terminal
ops. The handler thin-wraps `Manager.Spawn`, then returns the IDs the
parent's prompt context needs to coordinate.

For LLM-tool ergonomics, surface this as **MCP tools** registered by
the daemon when the parent's role allows spawning:

- `mcp__agh__spawn_agent(role, prompt, channel?, budget?)`
- `mcp__agh__send_to_agent(session_id, message)` — already half-built
  via `network.Manager.Send` (`internal/network/manager.go:560`).
- `mcp__agh__wait_for_agent(session_id, timeout?)`
- `mcp__agh__list_my_agents()` / `mcp__agh__stop_agent(session_id)`

These tools can be filtered into the parent's MCP attach set the same
way `internal/session/manager_start.go:304` (`resolveStartMCPServers`)
already handles skill MCPs.

### 5.4 CLI parity — `agh session spawn`

A user-facing CLI is useful for ops parity (mirror what `agh session
new` does today, but with parent context):

```text
agh session spawn \
  --parent <id> \
  --role researcher \
  --prompt "Investigate failing test in foo_test.go" \
  --budget-turns 10 --budget-wall 5m \
  --channel inherit \
  --auto-stop-on-parent
```

Wire it as a new cobra command next to `newSessionCreateCommand` at
`internal/cli/session.go:35`. Same client / HTTP path: add
`POST /api/sessions/:id/spawn` route in
`internal/api/httpapi/routes.go:67`.

### 5.5 Channel coordination

- **Auto-channel:** when `opts.Channel == "inherit"` (PROPOSED sentinel)
  set the child's `AGH_SESSION_CHANNEL` to the parent's channel. If the
  parent has no channel, mint a private one named
  `spawn:<parent_session_id>` so parent + children form a closed group.
- Add `LocalPeer.ParentSessionID` (`internal/network/peer.go:14`) and
  index it in `PeerRegistry` so `MatchLocalPeers` can filter by
  lineage. Then `whois ?parent=<sess>` returns only the spawn-tree.
- The existing `network.Manager.Send`
  (`internal/network/manager.go:560`) already moves bytes; the only
  delta is a peer-card extension `agh.parent_session_id` so remote
  daemons can see the same lineage.

### 5.6 Lifetime + reclaim contract

- New `lifecycleManager` (PROPOSED) goroutine inside the `session`
  package, owned by `Manager`, that:
  - Walks `m.sessions` periodically (or on each parent state change),
  - For every child with `AutoStopOnParent`, when parent enters
    `StateStopping`/`StateStopped`, calls
    `StopWithCause(child, CauseParentStopped, parent.ID)`.
  - For every child past `TTL`, calls
    `StopWithCause(child, CauseTTLExpired, ...)`.
- Mirror `automation.Dispatcher.tryAcquire`/`release`
  (`internal/automation/dispatch.go:947`) but per-spawn-tree:
  `treeQuota[rootSessionID]` capped at e.g. 5 concurrent descendants.

### 5.7 Persisted tree shape

Add columns to global session table (`internal/store/schema.go`):
`parent_session_id TEXT`, `spawn_depth INTEGER NOT NULL DEFAULT 0`,
`spawn_role_overlay JSON`, `spawn_budget JSON`. This makes the agent
tree queryable via `globaldb` and lets the web UI render lineage.

---

## 6. Safety / Limits

| Risk | Mitigation (proposed) | Reference |
|---|---|---|
| Fork bomb (child spawns child spawns child …) | Hard cap `SpawnDepth ≤ N` (default 1, cap at 3 like hermes & openclaw); enforce in `Manager.Spawn` *and* by stripping the spawn MCP tool from the child's tool set when `depth+1 == max`. | hermes `MAX_DEPTH = 1` (`/.resources/hermes/tools/delegate_tool.py:129`); openclaw `DEFAULT_SUBAGENT_MAX_SPAWN_DEPTH = 1` (`/.resources/openclaw/src/config/agent-limits.ts:7`) |
| Concurrent-child explosion | Per-parent cap + per-tree cap; reject with typed `ErrSpawnQuotaExhausted`. Default 5 children/parent, 16 descendants/tree. | openclaw `DEFAULT_SUBAGENT_MAX_CHILDREN_PER_AGENT = 5` (line 5) |
| Runaway child draining tokens / wall clock | `SpawnBudget.MaxWallClock` enforced by lifecycleManager; `MaxTurns` enforced by ACP turn counter; `MaxTokens` advertised to driver where supported. Default wall clock 10 min. | hermes `delegation.child_timeout_seconds = 600` (`tools/delegate_tool.py:363`) |
| Privilege escalation via role overlay | `SpawnRole.Permissions` may only narrow parent's mode. Validation in `startPermissions` (`internal/session/manager_helpers.go:55`) — reject any overlay that maps to `PermissionModeApproveAll` unless parent already holds it (system/dream). | — |
| Tool / MCP escape | `SpawnRole.AllowedTools` is an allowlist intersected with parent's set; never a superset. Strip `mcp__agh__spawn_agent` from leaves. Mirror hermes' `DELEGATE_BLOCKED_TOOLS`. | `/.resources/hermes/tools/delegate_tool.py:41` |
| Orphaned children if parent crashes | `AutoStopOnParent = true` default; lifecycleManager reaps on parent state transition; orphan reaper (`internal/daemon/orphan.go`) extended to also stop sessions whose `parent_session_id` is missing on boot. | reuse `finalizeStopped` (`internal/session/manager_lifecycle.go:145`) |
| Shared-channel flood | Per-channel send-rate cap in `network.Manager.Send`; cite spawn-tree as a `peer_id` set so routers can throttle a noisy tree without affecting siblings. | network audit (`internal/network/audit.go`) |
| Filesystem stomping between parent and child | Optional `SpawnRole.Isolation = "worktree"` borrowing claude-code's pattern (`/.resources/claude-code/tools/AgentTool/forkSubagent.ts`). Phase 2: hook into `sandbox-agent` patterns. | — |
| Auto-approval deadlock when child hits permission prompt | Children inherit auto-deny by default for permissions the parent itself would have to approve (parent UI not reachable from child). Mirror hermes' `_subagent_auto_deny` callback. | `/.resources/hermes/tools/delegate_tool.py:69` |
| Pause/kill operator escape hatch | Daemon-level `spawn_paused` flag (CLI: `agh session spawn-pause [--all\|--tree=ID]`) that fails new `Manager.Spawn` calls fast without killing in-flight children. | hermes `set_spawn_paused` (line 154) |

---

## 7. Open Questions

1. **Billing / quota** — should token budgets be per-tree or per-leaf?
   Per-tree mirrors hermes (children share parent's iteration budget,
   `/.resources/hermes/run_agent.py:943`) and gives a single throttle
   point; per-leaf is easier to reason about. Recommend **per-tree
   wall-clock + per-leaf token + per-leaf turn caps** to bound both
   total cost and individual runaway loops.
2. **Observability of agent trees** — do we expose a first-class
   `/api/sessions/tree?root=<id>` endpoint and a sidebar visualization,
   or do we lean on the existing `parent_session_id` column and let
   clients aggregate? A tree view materially changes the web UI
   (`web/src/systems/sessions/`) — needs a UX call.
3. **Ephemeral vs durable spawns** — dream sessions today already use
   `SessionTypeDream` and a different permission posture. Are spawned
   sub-agents always ephemeral (deleted from disk on stop, like dream
   sessions effectively are after consolidation), or do they persist
   their `events.db` for replay? Recommend **ephemeral by default**
   (the parent's transcript captures the meaningful summary), with a
   `SpawnRole.Persist = true` opt-in for cases like long research
   sessions whose transcript is the deliverable.
4. **Cross-daemon delegation** — does `Manager.Spawn` ever route to a
   *remote* daemon (over the agora network)? If yes, the protocol
   needs a `kind: "spawn"` envelope with a peer-card capability check
   and a remote `peer_id` reservation. If no, document the boundary so
   recipes don't accidentally try to. Likely Phase 3.
5. **Recipe / skill activation in a spawned context** — when a parent
   spawns a child with `SpawnRole.AllowedSkills = [...]`, do we re-run
   the workspace skill registry against that filter, or apply the
   filter in-process? The latter is simpler but may break path-pattern
   conditional activation (`internal/skills/`); needs a probe.
6. **Coordinator-mode flag** — claude-code's coordinator mode strips
   the parent's tools entirely (`/.resources/claude-code/coordinator/`).
   Worth a kernel-level switch on the *parent* session, not the spawn
   call, to avoid every prompt re-asserting "be a coordinator".
7. **What does "stop" mean for a child mid-task?** If the parent calls
   `mcp__agh__stop_agent(child)` while the child is mid-tool-call, do
   we issue ACP cancel, SIGTERM, then SIGKILL? Existing
   `manager_stop_integration_test.go` patterns apply, but the
   parent-to-child stop is a new entitlement check (only the parent or
   admin can stop a child, not a sibling).

---

## File Inventory (cited)

- `internal/session/manager.go:39` — `CreateOpts` struct (no parent
  fields).
- `internal/session/manager.go:468` — `reserve` quota gate.
- `internal/session/manager_lifecycle.go:20` — `Create` entrypoint.
- `internal/session/manager_lifecycle.go:104` — `watchProcess` reclaim.
- `internal/session/manager_lifecycle.go:145` — `finalizeStopped` reclaim
  template.
- `internal/session/manager_start.go:56` — `prepareCreateStart`.
- `internal/session/manager_start.go:129` — `startSession`.
- `internal/session/manager_start.go:221` — `m.driver.Start`
  (subprocess spawn point).
- `internal/session/manager_start.go:270` — `prepareSessionStartRuntime`
  (prompt + MCP assembly).
- `internal/session/manager_start.go:389` — `sessionStartOpts` (ACP
  start options including `SystemPrompt`).
- `internal/session/manager_start.go:417-427` — channel/peer env
  injection.
- `internal/session/manager_helpers.go:55` — `startPermissions`
  (permission selection by session type).
- `internal/session/session.go:36` — `Type` constants
  (`user|dream|system`).
- `internal/network/peer.go:14` — `LocalPeer` (no parent linkage).
- `internal/network/peer.go:124` — `RegisterLocal`.
- `internal/network/peer.go:276` — `MatchLocalPeers`.
- `internal/network/manager.go:560` — `Send` (peer-id routing only).
- `internal/automation/dispatch.go:367` — `dispatchAttempt` (closest
  in-tree precedent for trigger-spawn).
- `internal/automation/dispatch.go:399` — `sessions.Create` call site.
- `internal/automation/dispatch.go:947` — `tryAcquire` concurrency gate.
- `internal/memory/consolidation/runtime.go:410` — `spawnSession` for
  dream consolidation (template for ephemeral one-shot child).
- `internal/extension/host_api.go:760` — extension `sessions.create`
  RPC (closest existing surface, but for extensions only).
- `internal/extension/host_api_bridges.go:673` — bridge-driven session
  creation.
- `internal/api/core/network_details.go:584` — channel bootstrap
  spawning N peer sessions.
- `internal/api/httpapi/routes.go:67` — `POST /api/sessions` HTTP
  surface.
- `internal/cli/session.go:35` — `agh session new` CLI.
- `internal/task/manager.go:237` — `CreateChildTask` (existing
  parent-link template to mirror for sessions).
- `internal/config/config.go:52` — `LimitsConfig.MaxSessions`
  (daemon-global cap).
- `internal/config/config.go:67` — `SessionSupervisionConfig`.
- `internal/daemon/orphan.go` — orphan reaper to extend for
  parent-aware reclaim.
- `internal/daemon/harness_detached_work.go:1-100` — existing template
  for parent-session metadata propagation through tasks.
- `docs/ideas/from-claude-code/analysis_multi_agent.md` — claude-code
  pattern study (coordinator mode, fork, task notifications).
- `docs/ideas/from-claude-code/our_system_kernel.md:101-117` — earlier
  workgroup hierarchy design (master/worker/advisor) — useful prior
  art for role taxonomy if we revive it.
- `docs/ideas/orchestration/multi-agent-patterns-analysis.md:36-65` —
  choreography vs orchestration analysis maps directly onto this slice.
- `.resources/hermes/tools/delegate_tool.py:1-200, 363, 534` — primary
  reference for delegation safety.
- `.resources/openclaw/src/agents/subagent-system-prompt.ts:4-112` —
  primary reference for spawned-agent system prompt shape.
- `.resources/openclaw/src/config/agent-limits.ts:1-23` — primary
  reference for default quotas.
- `.resources/claude-code/tools/AgentTool/AgentTool.tsx:81-138` —
  primary reference for the LLM-facing tool schema.
