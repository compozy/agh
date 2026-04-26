# Autonomous AGH — Synthesis & Roadmap

> Ten parallel deep-research slices into "what does AGH need to become a truly autonomous agent OS where agents discover channels, claim tasks, coordinate with peers, spawn sub-agents, and self-correct without a human in the loop?"
>
> Source slices:
> - [analysis_agent_identity_prompts.md](analysis_agent_identity_prompts.md)
> - [analysis_agent_cli_surface.md](analysis_agent_cli_surface.md)
> - [analysis_network_channels_discovery.md](analysis_network_channels_discovery.md)
> - [analysis_task_discovery_claim.md](analysis_task_discovery_claim.md)
> - [analysis_inter_agent_comm_patterns.md](analysis_inter_agent_comm_patterns.md)
> - [analysis_memory_knowledge_sharing.md](analysis_memory_knowledge_sharing.md)
> - [analysis_auto_spawn_delegation.md](analysis_auto_spawn_delegation.md)
> - [analysis_skills_tools_registry.md](analysis_skills_tools_registry.md)
> - [analysis_orchestration_control_loop.md](analysis_orchestration_control_loop.md)
> - [analysis_observability_self_correction.md](analysis_observability_self_correction.md)

---

## 1. TL;DR

AGH already has roughly **80% of the substrate** for autonomy: a session manager with rich lifecycle hooks, a network protocol with capability briefs, an append-only event log, an FTS5 memory store with dream consolidation, a typed task state machine, a skills registry with workspace overlay, and four independent dispatchers (`automation`, `task`, `network`, `session`). What is missing is structural — not algorithmic, not even mostly new code. **Across all ten slices the same shape repeats:**

1. The daemon owns rich state. Agents inside ACP sessions cannot reach it because the **agent-facing surface is operator-shaped** (every CLI command demands explicit `--session/--channel/--workspace`, no implicit identity, no `recv --wait`, no `task next`, no `spawn`, no `me`).
2. **Capability data is published everywhere and consumed nowhere.** `CapabilityCatalog` ships in `peer cards`, `whois` answers, and prompt context — but the task scheduler doesn't see it, the channel discovery doesn't index it, the memory recall doesn't filter by it, and the agent doesn't see *its own* advertised capabilities.
3. **Sessions are mono-channel, mono-task, caller-spawned.** Multi-home channels, agent-initiated spawn, multi-run tasks, parent/child agent trees — none exist, even though every dependency for them is in place.
4. **There is no control loop.** Four dispatchers all run; nothing asks "I have N idle agents and M ready tasks — who runs what?". `automation.Dispatcher.gate` is the only global concurrency primitive and it caps automation only.
5. **Lifecycle hooks exist but the autonomy slots on them are empty.** `OnTurnEnd → memory extractor`, `OnSessionStop → workflow correlator`, `OnEchoReceived → peer-fact write`, `OnRunQueued → scheduler tick`, `OnLoopDetected → recovery prompt` — all are well-shaped slots with nothing wired in.
6. **`AGH` is greenfield-alpha** (`CLAUDE.md`), so we can renumber, restructure, and break wire formats freely. The cost of a clean break is zero.

The biggest single insight is that **the daemon should grow a thin "autonomy kernel"** that wires capability matching, queue claim, agent-callable telemetry, and inter-agent envelopes into a single coherent loop — without an event bus, without reflection, without changing the package layout philosophy. The biggest single fix is the **Situation Surface**: a runtime block injected at session start (and refreshed per turn) that tells the agent who it is, what channels exist, what tasks are queued, who its peers are, and what its own capabilities are. Everything else is leverage on top of that one block.

---

## 2. The Unifying Architectural Observation

| Layer in autonomy stack | What AGH has | What's missing | Slice that surfaces it |
|---|---|---|---|
| **A. Identity / situation** | Env vars (`AGH_SESSION_ID`, `AGH_PEER_ID`), composed prompt assembler, capability catalog | Live peer roster, self-capability mirror, task envelope, per-turn `<situation-update>` reminder | identity, memory |
| **B. Agent-callable surface** | `agh network/task/memory/skill` operator CLI; `delivery.go` even hand-builds shell snippets the agent must paste | Implicit identity, `agh me / ch recv --wait / task next / spawn`, JSONL streaming, exit-code taxonomy, idempotency keys | CLI, skills |
| **C. Discovery & matching** | `CapabilityCatalog`, `PeerCard.Capabilities`, channel registry with `purpose` text | Channel manifest, multi-home sessions, agent-declared interests, capability-indexed channel/task/skill/peer matching | channels, tasks, skills |
| **D. Coordination protocol** | 7 envelope kinds, `interaction_id` lifecycle, `whois` + capability catalog | Hand-off, multi-target, mentions, status broadcasts, contract-net (`propose/bid/award`), vote/escalate | comm patterns, orchestration |
| **E. Control loop / scheduling** | Four independent dispatchers, `Dispatcher.gate`, boot-time `RecoverTaskRunsOnBoot` | Idle-agent registry, atomic `ClaimNextRun(criteria)`, priority queue, leases + heartbeats, work-stealing, circuit breaker | orchestration, tasks |
| **F. Spawn & lineage** | `Manager.Create` from CLI/HTTP/automation/dream/extension/bridge | Agent-initiated `Manager.Spawn`, `ParentSessionID`, role overlay, budgets/TTL, parent-aware reaper, depth caps | spawn |
| **G. Knowledge accumulation** | Two memory scopes (global, workspace), FTS5, dream consolidation | Peer/channel/session scopes, `agent_name` provenance (column exists, never written), per-turn extractor, episodic summaries, peer-fact ledger fed by `echo`/`receipt` | memory |
| **H. Self-correction & telemetry** | `prompt_activity.go` watchdog, append-only events, hook telemetry, `Health.Activities` | Max-iterations + budget circuit breaker (`IterationCurrent` is dead column), repetition detector, recovery prompt injection, agent-callable telemetry tools, eval harness, workflow correlation | observability |

The pattern: **for each layer, the data structure exists; the consumers don't.** This is exceptionally good news — the work ahead is integration and ergonomics, not greenfield design.

---

## 3. Cross-Cutting Themes

### 3.1 "Capability catalog is published but never matched"

The same six lines of code (`internal/network/capability_brief.go:18-51`, `internal/network/capability_catalog.go:14-109`, `internal/config/capabilities.go:43`, `internal/session/interfaces.go:35-49`) are referenced by **eight of ten slices** as the "right shape" — and every slice independently observes it is **never consumed by a routing/matching/recall decision**. The capability brief is a peer-card decoration, not a runtime index.

Concrete blast radius:
- Task `ClaimRun` doesn't read it (orchestration, tasks).
- Channel discovery doesn't index by it (channels).
- Memory recall doesn't filter by it (memory).
- The agent's own prompt doesn't see its own advertisement (identity).
- Skills-on-the-wire doesn't project the loaded skill set (skills).
- Watchdog doesn't see "this agent has no capability for the work it's stuck on" (observability).

**One missing primitive (capability-index across stores) unblocks six gaps.**

### 3.2 "Session boot is a frozen one-shot; autonomy needs a refresh loop"

`startSession` (`internal/session/manager_start.go:129`) builds the system prompt, resolves MCP servers, joins the network channel, and never revisits any of it. The slices propose six refreshable surfaces for the same architectural reason:

| Surface | Today | Needed |
|---|---|---|
| System prompt (identity slice) | Built once at start | `[]string` with dynamic boundary; per-turn `<situation-update>` reminder |
| Skill catalog (skills slice) | `skillRegistry.ForWorkspace` once at start | Subscribe to `Registry.GlobalVersion`, push deltas as synthetic system message |
| Memory recall (memory slice) | Single-shot at start (`Assembler.PromptSection`) | Augment per-turn with channel/peer/session-scope filters |
| Peer roster (channels slice) | One channel, joined after process start | Multi-home sessions, `KindChannelAnnounce` SSE, auto-rejoin on manifest change |
| Task assignment (tasks slice) | Set at automation `Dispatch` time | `task.next` poll, lease renewal, heartbeat, queue notifier |
| Self-status (observability slice) | Daemon side only (`prompt_activity.go`) | Agent-callable `agh.session.stats`, watchdog → recovery prompt |

The single biggest unifying change is to **treat the system prompt as `[]string` with a `__DYNAMIC_BOUNDARY__` marker** (claude-code pattern), make the after-boundary section refreshable, and wire it to events from all six surfaces above.

### 3.3 "Hooks are ready, listeners are missing"

`internal/hooks/events.go`, `internal/session/manager_hooks.go`, and `internal/observe/observer.go` already define rich lifecycle slots. Slices identify these unfilled wiring points:

| Hook | Owner today | Proposed listener | Source slice |
|---|---|---|---|
| `OnTurnEnd` | session.Manager | Memory extractor → `ScopeSession/scratch.md` | memory |
| `OnPreCompact` | session lifecycle | Memory flush before context discard | memory |
| `OnSessionEnd` | session lifecycle | Episodic summary writer + workflow correlator | memory, observability |
| `OnEchoReceived` (network) | not wired | Peer reputation ledger writer | memory, comm |
| `OnReceiptReceived` (network) | not wired | Peer outcome ledger + capability success rate | memory, skills |
| `OnRunQueued` (task) | recordTaskEvent fires; no notifier | Scheduler tick (idle-agent × ready-run match) | orchestration, tasks |
| `OnLoopDetected` (proposed) | not wired | Recovery prompt injection; circuit breaker | observability |
| `OnBudgetExceeded` (proposed) | not wired | CancelPrompt with `CauseBudgetExceeded` | observability |
| `OnPeerJoined/Left` | network heartbeat fires; no synthetic re-prompt | Synthetic `<situation-update>` to active sessions | identity, channels |
| `OnSkillRegistryDelta` | `Registry.GlobalVersion` exists; no consumer | Push delta to ACP driver as synthetic system message | skills |

These are zero-architecture changes — they are data-flow wiring across packages that already interface through Go interfaces (no event bus needed).

### 3.4 "Operator surface vs agent surface are conflated"

Smoking-gun evidence: `internal/network/delivery.go:820-988` literally hand-builds multi-line `agh network send --session "$AGH_SESSION_ID" --channel "..." --to "..." --reply-to "..." --causation-id "..."` shell snippets and injects them into delivered envelopes as guidance text. That entire string-construction path exists **only because the CLI doesn't infer caller identity**. It is the single clearest sign the current CLI was built for humans, not agents.

The proposed split (CLI slice §4):
- `agh network/task/memory/skill/session` — operator surface, explicit flags, stable contract.
- `agh me / agh ch / agh task next/done/fail / agh spawn` — agent surface, identity from env, JSONL output, exit-code taxonomy, idempotency keys.
- `delivery.go` reply guidance becomes one verb: `agh ch reply --to-message <id>`.

### 3.5 "Greenfield-alpha" is an enabling constraint, not a complication

Per `CLAUDE.md`, AGH has zero tolerance for backward-compat. **Five of ten slices** propose changes that would be expensive in any other repo (renaming `localsByID: map[sessionID]LocalPeer` to `[]LocalPeer`; bumping protocol to `agh-network/v1`; replacing `session.Channel` with `Channels []string`; renumbering `Type` constants to add `SessionTypeSpawned`; rewriting `Backend.Write` signature to carry agent provenance) and every one of them is, per the project rule, a clean break with no migration code.

This means the autonomy kernel can land as **one coordinated wire-format + schema bump** rather than as a year of additive carry-forward shims.

---

## 4. Per-Slice Gap Summary

Compact view; each line links back to the slice that argues it.

### Identity & system prompts
- No "Situation" / runtime-facts block (peer roster, self-capability mirror, task envelope) injected at session start.
- No per-turn `<situation-update>` reminder channel.
- `joinNetworkPeer` runs *after* the prompt is frozen — even if data existed, the ordering blocks injection.
- No tool-aware section gating; no `[]string` prompt with dynamic boundary; capabilities never reach `acp.StartOpts`.
- Frontmatter has no `role`/`mission`/`peer_alias`/`claims_capabilities` — same `AGENT.md` cannot serve two roles.

### Agent CLI surface
- Identity is operator-explicit; `delivery.go` hand-builds shell snippets to compensate.
- Missing: `agh me`, `agh me context`, `agh ch recv --wait`, `agh ch join/leave/whois/discover`, `agh ch reply`, `agh task next/pass/lease-extend/done/fail/subscribe/comment`, `agh spawn`.
- No exit-code taxonomy, no JSONL streaming, no idempotency keys on `network send`.

### Network channels discovery
- Channel is an opaque string, not a typed `ChannelManifest` (purpose, topics, roles needed, owner, parent, expiry).
- Sessions are mono-channel by registry design (`localsByID: sessionID → LocalPeer`).
- No `KindChannelAnnounce`, no channel `whois`, no `GET /api/network/channels/stream` SSE.
- `bundle.DeclaredChannels` are dead inventory.
- No agent-callable `JoinAdditionalChannel`/`CreateChannel`/`MatchChannels(query)`.

### Task discovery & claim
- No lease columns (`claim_token`/`lease_until`/`heartbeat_at` missing in schema).
- `ClaimRun` is racy (no `WHERE status='queued'` conditional, no row lock, no `RETURNING`).
- No `ClaimNextRun(criteria)` atomic pop, no capability-aware claim, no priority-aware ordering.
- Single-open-run-per-task gate blocks pool-style parallelism *and* takeover after a stall.
- Agents have no in-process tool family (`task.create_child`, `task.claim_next`, `task.heartbeat`, `task.complete`).
- No queue notifier for "ready work appeared" — only per-task SSE that requires knowing the taskID.

### Inter-agent communication patterns
- 7 verbs cover broadcast / 1:1 RPC / presence / identity-probe / capability-publish, but lack hand-off, multi-target, mentions, public status, vote, offer/accept/decline, escalate, react, cancel.
- `interaction_id` is two-party-locked; ownership cannot transfer mid-flight.
- `intent` is unregistered free-form string with no router-side meaning.
- No mention parser inside body text.
- `expires_at` is replay-window upper bound only; no automatic timeout `trace.failed`.

### Memory & knowledge sharing
- Two scopes only: global + workspace. No peer / channel / session scope.
- `agent_name` is plumbed end-to-end but never written (always `"daemon"`) and never queried.
- No automatic write path other than 24h dream — no per-turn extractor, no pre-compaction flush, no on-session-end episodic summary.
- Network primitives `echo`/`trace` are stored in `network_timeline_log` and never promoted to durable knowledge.
- Recall is scope-blind to channel/peer/identity; recall block has no actionable provenance.

### Auto-spawn & delegation
- `Manager.Create` is daemon-internal only — agents have no callable surface back to it.
- `CreateOpts` has no `ParentSessionID`, no `SpawnRole`, no `Budget`, no `TTL`.
- `LocalPeer` is flat — no lineage, no parent/child filter for routing.
- `LimitsConfig.MaxSessions` is daemon-global; no per-tree quota, no depth cap, no orphan reaper aware of lineage.
- The right-shape precedent (`task.CreateChildTask`) exists but only on tasks.

### Skills & tools registry
- `tools.Tool` is descriptive only — no `Call`, no `IsAvailable`, no permissions hook.
- Skill catalog is frozen at session start — no live delta channel (cf. claude-code `deferred_tools_delta`).
- No on-demand tool/skill body load mid-turn (HTTP `GET /api/skills/:name/content` exists but agents have no tool that calls it).
- No per-role / per-session tool scoping (`AgentDef.Tools` is flat `[]string`).
- Skills are absent from peer cards (`agh.skill_brief` doesn't exist).
- No usage telemetry, no install-from-session API, no availability gating, no version pinning, no signed bundles.

### Orchestration control loop
- Four independent dispatchers; zero global control loop.
- No idle-agent registry, no `ClaimNextRun(criteria)`, no priority queue, no backpressure, no leader, no contract-net, no work-stealing, no circuit breaker, no workflow correlation.
- Boot reconciliation is the only centralized scheduling decision — runs once.
- The cleanest path is `internal/scheduler/` as a daemon goroutine + `task.Service.ClaimNextRun` + 3 new envelope kinds (`propose`/`bid`/`award`), separating the deterministic scheduler from the LLM-driven coordinator-agent.

### Observability & self-correction
- Append-only event log, hook telemetry, network audit, health snapshot all in place.
- `IterationCurrent`/`IterationMax` exist in schema, **never incremented**.
- No max-iterations / cost-budget circuit breaker, no repetition detector, no recovery-prompt injection (only cancel-cold).
- No agent-callable telemetry (`agh.session.stats`, `agh.peer.status`, `agh.interaction.status`).
- No eval/replay harness, no workflow correlation, no `/metrics` endpoint, no per-tool latency histogram.

---

## 5. Proposed Roadmap — Four Pillars

The slices independently converge on a four-pillar structure, sequenced so each unblocks the next. Names are deliberately chosen so a single PR / feature can carry the work.

### Pillar 1 — **Situation Surface** (P0; unblocks every other pillar)

> *The agent always knows: who it is, where it is, who's around, what's queued, what it can do.*

Single load-bearing change: turn the system prompt into a refreshable two-part document with a dynamic boundary, and wire a `<situation>` block into both the static and dynamic halves.

Concrete additions:
1. `internal/daemon/situation_section.go` — new prompt provider rendering `<situation>` (self peer-id/session/workspace/model/channel/started-at + peer roster + queued task envelope).
2. `internal/daemon/self_capabilities_section.go` — render `AgentDef.Capabilities` so the agent can answer "what can I do?".
3. Reorder `joinNetworkPeer` to run **before** prompt freeze (or accept a synthetic-self snapshot path).
4. Switch `composed_assembler.go` to `[]string` with `__AGH_DYNAMIC_BOUNDARY__` sentinel.
5. Add `daemon/situation_reminder_augmenter.go` — per-turn `<situation-update changed="...">` on user-message side.
6. Promote `AgentDef` frontmatter with `role`, `mission`, `peer_alias`, `claims_capabilities`.
7. Add `TaskContextProvider` for task/automation-launched sessions; carry `TaskID/RunID` through `StartupPromptContext`.
8. Inject `PeerID`, `Channel`, `Capabilities`, `TaskID` into `acp.StartOpts`.

Why first: every other pillar amplifies its value, but the system prompt is also the single thing every agent reads first. Without it, the rest is unreachable knowledge.

### Pillar 2 — **Agent Kernel CLI** (P0; the other half of agent autonomy)

> *Every read/write the daemon supports is reachable from inside a session as a verb-style command with implicit identity.*

Concrete additions (CLI + matching UDS endpoints + contract types):
1. **Identity-implicit CLI**: every command resolves `AGH_SESSION_ID` / `AGH_AGENT` automatically; `clientFromDeps` adds a middleware that tags requests with the caller session for audit.
2. **`agh me` namespace**: `agh me`, `agh me context [--include task,inbox,memory,peers]`, `agh me capabilities set`, `agh me status --working/--idle`, `agh me logout`.
3. **`agh ch` namespace** (alias of `network`): `agh ch list/peers/send/inbox` (existing) + **new** `agh ch recv --wait`, `agh ch join/leave/whois/discover --capability`, `agh ch reply --to-message <id>` (kills the `delivery.go:820-988` snippet hack).
4. **`agh task` agent verbs**: `agh task next [--capability ... --wait]`, `agh task pass`, `agh task lease-extend`, `agh task done` (alias), `agh task fail` (alias), `agh task subscribe`, `agh task comment`.
5. **`agh spawn`**: `agh spawn --agent <name> [--prompt "..." --bind-task --bind-channel inherit --auto-stop-on-parent --budget-turns N --budget-wall 5m]`.
6. **Output contracts**: `-o json` is stable; `-o jsonl` for streaming; exit-code taxonomy `0/1/2/3/4/5/6/7`; idempotency-key header pass-through.
7. **Built-in `agh.*` MCP tools** mirroring the CLI for ACP runtimes that prefer typed tools (claude-code style).

Why second: gives Pillar 1's prompted agent something to **call back with**. Without this, the system prompt block is descriptive text the agent can't act on.

### Pillar 3 — **Autonomy Kernel** (P1; closes the loop)

> *A daemon-side scheduler matches idle agents to ready work, with leases, capability-aware claim, contract-net for cross-daemon, and parent/child agent trees.*

New package `internal/scheduler/`:
- `agents.go` — idle-agent registry indexed by capability, populated via `session.Manager` lifecycle hooks.
- `claim.go` — `task.Service.ClaimNextRun(ClaimCriteria)` thin-wrap with `BEGIN IMMEDIATE` SQLite (multica `FOR UPDATE SKIP LOCKED` analog).
- `lease.go` — schema columns `claim_token`, `claimed_session_id`, `lease_until`, `heartbeat_at`; sweep loop reusing `RecoverRunOnBoot`.
- `loop.go` — `Loop.Tick()` driven by `task.run_enqueued` + `session.turn_end` notifiers (no polling).
- `policy.go` — `ClaimPolicy` interface (priority/age/round-robin/capability-weighted).

Network protocol additions (one wire bump to `agh-network/v1`):
- 9 new envelope kinds: `KindStatus`, `KindOffer`, `KindAccept`, `KindDecline`, `KindHandoff`, `KindCancel`, `KindVote`, `KindReact`, `KindEscalate`.
- 5 new envelope fields: `ThreadID`, `Audience`, `Mentions`, `Priority`, `DeadlineAt`.
- Registered `intent` taxonomy on `say`/`direct`.
- In-text mention parser (`@peer`, `@role:reviewer`, `@all`).
- Hand-off lifecycle: `KindHandoff` mutates `Interaction.Target`.
- Channel manifest (`KindChannelAnnounce`, `KindChannelWhois`); `localsByID: []LocalPeer` for multi-home; `bundle.DeclaredChannels` becomes live.

Spawn/delegation:
- `CreateOpts` extension (`ParentSessionID`, `SpawnRole {BaseAgentName, PromptOverlay, Permissions, AllowedTools, AllowedSkills}`, `SpawnBudget`, `TTL`, `AutoStopOnParent`); new `SessionTypeSpawned`.
- `Manager.Spawn(parentID, opts)` thin-wrapping `startSession`.
- `LocalPeer.ParentSessionID` + peer-card `agh.parent_session_id`.
- Hard caps: depth=1 (cap=3), 5 children/parent, 16 descendants/tree, wall=10m.
- Reaper extension: auto-stop children on parent stop / TTL expiry / orphan-on-boot.

Coordinator-agent (separate from scheduler):
- Bundled agent definition with stripped tool surface (only `task.create`, `task.claim`, `agent.list`, `agent.send`).
- Daemon auto-spawns one per workspace on first work appearance.

Why third: this is the actual autonomy. Pillars 1+2 give the agent eyes and hands; Pillar 3 gives the world a metabolism.

### Pillar 4 — **Memory + Self-Correction Loop** (P1, ships alongside P3)

> *Agents accumulate knowledge from coordination signals, peers, and outcomes; the daemon detects loops and budget overruns and either nudges or stops.*

Memory:
1. New scopes: `ScopePeer`, `ScopeChannel`, `ScopeSession` (filesystem dirs + catalog columns).
2. Plumb `agent_name` through `Backend.Write(WriteOpts{...})` and `OperationHistoryQuery` / `SearchOptions`.
3. Wire memory writes into the empty hook slots: `OnTurnEnd`, `OnPreCompact`, `OnSessionEnd`, `OnEchoReceived`, `OnReceiptReceived`, `OnPeerJoined`.
4. Skill outcome ledger (success rate, last failures) → optional auto-disable / circuit-break.
5. `ScopeSkill` virtual scope for in-prompt success-rate hints when an agent considers a skill.
6. Recall provenance: render `<memory-context>` fenced block with file pointer + agent + scope + score + freshness.
7. `workflow_id` cross-session correlation field on `SessionEvent` content JSON.
8. Append-only logs for `ScopePeer`/`ScopeChannel` to avoid concurrent-write conflicts.

Observability:
1. Wire `IterationCurrent`/`IterationMax` (the dead column gets life). New `CauseMaxIterations`.
2. Repetition detector: bounded LRU of `(tool_name, sha256(input))`; flag `Liveness.StallState="loop_detected"`.
3. Recovery prompt injection between warning and timeout (synthetic prompt path already exists in `synthetic_prompt.go`).
4. Cost circuit breaker: `MaxBudgetUSD`/`MaxTokens` checked inside `OnAgentEventForSession` after `UpdateTokenStats`.
5. Agent-callable telemetry tools: `agh.session.recent_events`, `agh.session.stats`, `agh.peer.status`, `agh.channel.recent`, `agh.interaction.status`, `agh.failures.recent`, `agh.hook.runs`.
6. New SSE: `GET /api/sessions/:id/activity/stream`, `GET /api/observe/alerts/stream`.
7. New hook events: `runtime.progress`, `runtime.warning`, `runtime.timeout`, `runtime.loop_detected`, `session.budget_exceeded`.
8. `AlertNotifier` interface (log → webhook → ring) for operator push.
9. Eval harness skeleton under `internal/eval/`: YAML cases (paperclip-style), trajectory dump (hermes-style), VCR fixtures (claude-code-style).
10. Optional `/metrics` endpoint (Prom format).

Why fourth (but in parallel with P3): memory closes the loop on coordination outcomes; observability closes the loop on autonomy safety. Both depend on the same hook slots P3 is wiring; doing them in the same release minimizes churn.

---

## 6. Top 10 Specific Changes to Land First

If only ten PRs ship, these maximise blast radius. Each is small, additive, and unblocks 2-3 other gaps.

| # | Change | Touches | Unblocks |
|---|---|---|---|
| 1 | `daemon/situation_section.go` rendering `<situation>` (peer roster + self peer-id) at session start, with `joinNetworkPeer` reordered before prompt freeze | `internal/daemon`, `internal/session/manager_helpers.go`, `internal/session/manager_start.go` | identity, channels, comm |
| 2 | `daemon/self_capabilities_section.go` rendering own `CapabilityCatalog` into the prompt | `internal/daemon`, `internal/config/capabilities.go` | identity, skills, comm |
| 3 | Implicit-identity middleware on UDS + per-command env resolution; replace `delivery.go:820-988` reply guidance with one `agh ch reply --to-message <id>` verb | `internal/cli`, `internal/api/udsapi`, `internal/network/delivery.go` | CLI, comm |
| 4 | `agh ch recv --wait` (SSE long-poll) and `agh task next --capability X --wait` atomic pop; add `agh me context` heartbeat read | `internal/cli`, `internal/api/udsapi`, `internal/task/manager.go` | CLI, tasks, orchestration |
| 5 | Schema additions: `task_runs.{claim_token, claimed_session_id, lease_until, heartbeat_at}` + `tasks.required_capabilities_json`; lease-sweep loop reusing `RecoverRunOnBoot` | `internal/store/globaldb`, `internal/task/manager.go`, `internal/daemon/task_runtime.go` | tasks, orchestration |
| 6 | `internal/scheduler/` package + `task.Service.ClaimNextRun(ClaimCriteria)` + `CapabilityProvider` interface implemented by `session.Manager` | new `internal/scheduler`, `internal/task`, `internal/session` | orchestration, tasks |
| 7 | Multi-home sessions (`localsByID: []LocalPeer`) + agent-declared interests in `AgentDef` + `KindChannelAnnounce` + `JoinAdditionalChannel` | `internal/network/peer.go`, `internal/network/envelope.go`, `internal/session/interfaces.go`, `internal/config/agent.go` | channels, comm, identity |
| 8 | Memory hook wiring (`OnTurnEnd`/`OnSessionEnd`/`OnReceiptReceived`/`OnEchoReceived`) + `ScopePeer`/`ScopeChannel`/`ScopeSession` + real `agent_name` provenance | `internal/memory/*`, `internal/session/manager_hooks.go`, `internal/network/audit.go` | memory, observability |
| 9 | Watchdog hardening: increment `IterationCurrent`, repetition detector, `MaxBudgetUSD` circuit breaker, recovery-prompt injection between warning and timeout | `internal/session/prompt_activity.go`, `internal/session/synthetic_prompt.go`, `internal/observe/observer.go` | observability, orchestration |
| 10 | `Manager.Spawn(parentID, SpawnOpts)` + `agh spawn` CLI + `LocalPeer.ParentSessionID` + parent-aware reaper, with safety caps (depth=1, 5 children) | `internal/session/manager_lifecycle.go`, `internal/cli`, `internal/network/peer.go`, `internal/daemon/orphan.go` | spawn, orchestration |

These ten shipped together are the **autonomy MVP**. They wire identity → CLI → claim → schedule → spawn → memory → self-correct in a single coherent loop.

---

## 7. Cross-Slice Open Questions

Questions that span multiple slices and need decisions before implementation.

1. **Wire-format bump now or never** — `agh-network/v1` is greenfield-clean; bumping protocol while we add 9 envelope kinds + 5 fields + channel manifest is cheaper than dribbling them out. Decision: **bump v1 once**, plan documented in a single network RFC.
2. **Coordinator-agent vs scheduler split** — orchestration slice argues for two separate concerns (deterministic scheduler in daemon, LLM-driven coordinator as a bundled agent). Identity/spawn slices implicitly assume the same split. Confirm: do not conflate.
3. **Single-leader vs decentralized** — start single-leader-per-daemon, defer election. Cross-daemon contract-net (`propose/bid/award`) is the upgrade path.
4. **Memory privacy across workspaces** — `ScopePeer/peers/<fp>` is global; `ScopeChannel` is workspace-scoped unless the channel is global. Document explicitly.
5. **Spawn permission inheritance** — children narrow, never widen. `mcp__agh__spawn_agent` is stripped from leaves at depth=cap.
6. **Capability matcher exactness** — start exact-match `(id, version)`; defer scored matching until data shows it's needed.
7. **Output contract surface** — `-o json` is stable; `-o jsonl` for streaming; everything else (`human`, `toon`) is unstable display only. Promote JSON Schemas for every CLI response from `internal/api/contract` to first-class generated artifacts.
8. **Scheduler home** — daemon goroutine, **not** an agent. Avoids dependency cycle (the placer of agents is itself an agent).
9. **Halting policy when `MaxIterations`/`MaxBudgetUSD` trip** — recommend recovery-prompt-then-cancel (claude-code's pattern for max-output, AGH's pattern for max-turns combined).
10. **Eval determinism strategy** — VCR (recorded ACP responses) for fast/cheap CI, rubric/promptfoo for nightly broader runs. Both, not one.

---

## 8. What This Roadmap Does NOT Try to Solve

Honest scope statement, so we don't over-promise.

- **Cross-daemon swarm at scale** — contract-net verbs land, but production-grade leader election, gossip protocols, and Byzantine fault tolerance are out of scope for the autonomy MVP. Single-daemon autonomy is the goal.
- **Embedding-backed memory** — FTS5 + structured facts is the immediate target; vector backends are a separate Phase-2 plug-in (`memory-gaps/README.md` flagged this).
- **Signed skill bundles / supply-chain hardening** — proposals exist (sigstore, transparency log) but ship after the autonomy kernel. Marketplace allowlist remains the gate in the meantime.
- **Cross-organization trust** — `network/auth` is a separate work-stream. Per-session HMAC tokens are flagged in the CLI slice as the right shape but explicitly defer to Phase 3.
- **Replacing the four dispatchers with one** — orchestration slice argues for *one new component* (`internal/scheduler/`) layered on top, not for collapsing automation/task/network/session into a monolith. Keep them; coordinate them.

---

## 9. Index of Slice Documents

All slices live in `.compozy/tasks/autonomous/analysis/`.

| File | Slice | Key proposal count |
|---|---|---|
| `analysis_agent_identity_prompts.md` | Identity & system prompts | 10 ranked changes (P1: SituationProvider, P2: self-capabilities, P3: TaskContextProvider, P4: per-turn `<situation-update>`, …) |
| `analysis_agent_cli_surface.md` | Agent CLI surface | 5 namespaces (`me`, `ch`, `task`, `spawn`, `memory`); ~30 verbs; exit-code + JSON schema policy |
| `analysis_network_channels_discovery.md` | Network channels & discovery | 9 dependency-ordered proposals (channel manifest, multi-home, `KindChannelAnnounce`, agent-declared interests, …) |
| `analysis_task_discovery_claim.md` | Task discovery & claim | Schema additions, `ClaimNextRun`, `CapabilityProvider`, `QueueWatcher`, agent tool family, lease sweep |
| `analysis_inter_agent_comm_patterns.md` | Inter-agent communication patterns | 9 new kinds, 5 new envelope fields, registered `intent` taxonomy, mention parser, hand-off lifecycle, helper API |
| `analysis_memory_knowledge_sharing.md` | Memory & knowledge sharing | 10 proposals A–J: 3 new scopes, agent provenance, lifecycle hooks, skill ledger, fact API, recall provenance, workflow_id |
| `analysis_auto_spawn_delegation.md` | Auto-spawn & delegation | `Manager.Spawn`, `CreateOpts` extension, peer lineage, safety cap matrix, MCP tool surface |
| `analysis_skills_tools_registry.md` | Skills & tools registry | 9 proposals P1–P9: runtime tool, `internal/catalog/`, `agh.*` tools, `ToolPolicy`, live deltas, network skill brief, install-from-session, telemetry, availability |
| `analysis_orchestration_control_loop.md` | Orchestration control loop | `internal/scheduler/`, contract-net (`propose/bid/award`), coordinator-agent split, multica SQL pattern |
| `analysis_observability_self_correction.md` | Observability & self-correction | Watchdog hardening (4), agent-callable telemetry (7), workflow-id, alert notifier, hook events, replay/eval harness |

---

## 10. Reference File Index — Load-Bearing Paths from Analyzed Projects

Consolidated index of the most-cited files across all ten slices, so future work can jump directly to the precedents instead of re-discovering them. Paths are absolute under `/Users/pedronauck/Dev/compozy/agh/.resources/`.

### 10.1 Hermes — single-agent reference (memory, delegation, prompt, tools)

| Topic | Path | What to read |
|---|---|---|
| Identity / system prompt | `hermes/run_agent.py:4361-4502` (`_build_system_prompt`) | Layered identity assembly: SOUL → tool-aware guidance → model preamble → memory → skills → context files → live timestamp + session id + env hints + platform hints |
| Identity bootstrap (CLI) | `hermes/hermes_cli/main.py:35` | `honcho identity <file>` — peer identity seeding |
| Identity tips | `hermes/hermes_cli/tips.py:216` | SOUL.md replacing `DEFAULT_AGENT_IDENTITY` |
| Memory provider interface | `hermes/agent/memory_provider.py:42-86, 60-81, 92-119, 163-186` | Pluggable provider, lifecycle hooks (`initialize`, `prefetch`, `sync_turn`, `on_pre_compress`, `on_delegation`) |
| Memory manager (context fencing) | `hermes/agent/memory_manager.py:46-82, 84-142` | `<memory-context>` fenced injection + builtin/external orchestration |
| Trajectory dump | `hermes/agent/trajectory.py` | Success/failure trajectory JSONL — eval harness precedent |
| Insights | `hermes/agent/insights.py` | Post-run analytics from trajectories |
| Delegation (subagent spawn) | `hermes/tools/delegate_tool.py:1-200, 308-323, 363, 497-515, 534, 1792, 1965, 1986-2047` | `MAX_DEPTH`, `DELEGATE_BLOCKED_TOOLS`, `_get_max_concurrent_children`, `interrupt_subagent`, `set_spawn_paused`, `_subagent_auto_deny`, `DelegateEvent` enum |
| Tool registry | `hermes/tools/registry.py:80-97, 100-227, 194-203, 258-286` | Singleton with `register()`, `check_fn` availability gating, MCP collision rules |
| Toolsets | `hermes/toolsets.py:31-454, 504-554` | Recursive toolset composition + `all`/`*` aliases |
| Skill manager (agent-callable CRUD) | `hermes/tools/skill_manager_tool.py:32-200, 72-96, 118-200` | Agent-callable skill CRUD + security scan + frontmatter validation |

### 10.2 Multica — multi-agent dispatch + chat protocol reference

| Topic | Path | What to read |
|---|---|---|
| CLI / daemon model | `multica/CLI_AND_DAEMON.md` | Polling + claim + heartbeat model; agent-callable verbs |
| Task claim (atomic SQL) | `multica/server/pkg/db/queries/agent.sql:116-140` | `UPDATE … WHERE id = (SELECT … ORDER BY priority DESC, created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED)` — the entire scheduler in one query |
| Task service (claim + complete) | `multica/server/internal/service/task.go:38-77, 82-111, 176-237, 241-304, 348-463` | `EnqueueTaskForIssue`, `EnqueueTaskForMention`, `ClaimTask`, `ClaimTaskForRuntime`, idempotent `CompleteTask` |
| Chat / message envelope | `multica/server/pkg/protocol/messages.go:6-9` | Radically simple `{Type, Payload}` with one verb per intent |
| Mention parser | `multica/server/internal/util/mention.go:7-44` | `[@Label](mention://type/id)` markdown link parser, `IsMentionAll()` short-circuit |
| Chat store | `multica/packages/core/chat/index.ts:1` | TanStack Query store conventions |
| Autopilot queries | `multica/packages/core/autopilots/queries.ts:11-34` | React-Query key conventions |

### 10.3 OpenClaw — agent harness, sub-agents, skills, capability discovery

| Topic | Path | What to read |
|---|---|---|
| Agent system prompt builder | `openclaw/test/helpers/agents/prompt-composition-scenarios.ts:84` | `buildAgentSystemPrompt(runtimeInfo, userTimezone, toolNames, acpEnabled, skillsPrompt, …)` |
| Subagent system prompt template | `openclaw/src/agents/subagent-system-prompt.ts:4-112` | "You are a subagent / Stay focused / Be ephemeral / Don't initiate" |
| Subagent limits | `openclaw/src/config/agent-limits.ts:1-23` | `DEFAULT_SUBAGENT_MAX_CHILDREN_PER_AGENT=5`, `MAX_SPAWN_DEPTH=1`, `MAX_CONCURRENT=8` |
| Multi-agent concept | `openclaw/docs/concepts/multi-agent.md:104, 120, 140` | Channels = chat platform bindings; runtime configs |
| Sub-agent thread bindings | `openclaw/docs/tools/subagents.md:142, 148` | Idle TTL ad-hoc channel for sub-agents |
| Outbound message | `openclaw/src/infra/outbound/message.ts:73, 96, 239` | First-class `replyToId`, `MessagePollParams`, `deliveryMode: direct|gateway` |
| CLI surface | `openclaw/src/cli/` | Capability-cli, pairing-cli — first-class capability discovery as CLI verbs |
| Memory short-term promotion | `openclaw/extensions/memory-core/src/short-term-promotion.ts` | Explicit promotion path from short-term observations to durable memory |
| Memory dream phases | `openclaw/extensions/memory-core/src/dreaming-{phases,narrative,repair,markdown,shared}.ts` | Per-phase decomposition (vs AGH's monolithic `internal/memory/prompt.go`) |
| Memory embedding plugin | `openclaw/extensions/memory-lancedb/` | Embedding-backed memory plugin reference |
| Active memory | `openclaw/extensions/active-memory/` | Currently-active context vs durable corpus split |
| Session search visibility | `openclaw/extensions/memory-core/src/session-search-visibility.ts` | Restrict search by session/peer scope |

### 10.4 Paperclip — MCP-shaped agent tool surface, sandbox lease, eval harness

| Topic | Path | What to read |
|---|---|---|
| MCP tool surface (agent verbs) | `paperclip/packages/mcp-server/README.md` | `paperclipCheckoutIssue`, `paperclipReleaseIssue`, `paperclipAddComment`, `paperclipAskUserQuestions`, `paperclipApprovalDecision` |
| Heartbeat context (single read) | `paperclip/packages/mcp-server/README.md` (`paperclipGetHeartbeatContext`) | Single `agh me context` analog returning everything an agent needs to decide next |
| API escape hatch | `paperclip/packages/mcp-server/README.md` (`paperclipApiRequest`) | Generic REST escape when typed tools insufficient |
| Sandbox lease pattern | `paperclip/packages/plugins/sandbox-providers/e2b/plugin.test.ts:93-352` | `acquire/extend/release` with `providerLeaseId`, `reuseLease`, pause-vs-kill |
| Plugin stream bus | `paperclip/server/src/routes/plugins.ts:346` + `plugin-stream-bus.ts:5` | `(pluginId, channel, companyId)` triple as multi-tenant scoping key |
| Eval harness (promptfoo) | `paperclip/evals/README.md` + `paperclip/evals/promptfoo/cases/` | Phased eval rollout; flat YAML-per-case; categories (`core`, `governance`); deterministic asserts |
| Plugin worker manager | `paperclip/server/src/plugin-worker-manager.ts:393, 576, 695` | Worker holds N stream channels; synthetic open/close events |

### 10.5 Claude Code — the agent harness reference (prompts, tools, sub-agents, memory)

| Topic | Path | What to read |
|---|---|---|
| Agent tool (LLM-facing schema) | `claude-code/tools/AgentTool/AgentTool.tsx:81-138` | `description, prompt, subagent_type, model, run_in_background, name, team_name, mode, isolation, cwd` |
| Sub-agent fork (worktree isolation) | `claude-code/tools/AgentTool/forkSubagent.ts` | `isolation: "worktree"` git worktree pattern |
| Stop hooks | `claude-code/query/stopHooks.ts:23-30, 65-100` | `Stop`, `TeammateIdle`, `TaskCompleted` families that can `preventContinuation`/`blockingErrors` |
| Query loop / max-turns / withhold-recover | `claude-code/query.ts:1031, 1107, 1267-1302, 1705-1711` | `state.turnCount > maxTurns`, recoverable error withhold-then-continue, `error_max_budget_usd` |
| Cost tracker (in-loop budget) | `claude-code/cost-tracker.ts` | `getTotalCost() >= maxBudgetUsd → return error_max_budget_usd` |
| Find relevant memories | `claude-code/memdir/findRelevantMemories.ts` | LLM side-query selecting top-5 relevant memories per query |
| Team memory paths | `claude-code/memdir/teamMemPaths.ts` | Team memory subdirectory + per-type scoping rules |
| Team memory sync | `claude-code/services/teamMemorySync/` | Team memory sync with secret guard |
| Session memory | `claude-code/services/SessionMemory/sessionMemory.ts` | Running session memory pattern |
| Skills loader | `claude-code/skills/loadSkillsDir.ts:67-94` | 5 settings sources (managed, user, project, plugin, mcp) |
| Plugins | `claude-code/services/plugins/` | Bundle of bundles approach |
| Memory utils | `claude-code/utils/memory/types.ts` | Types for the memdir model |

### 10.6 Sandbox-agent — sandboxing research (relevant after spawn API exists)

| Topic | Path | What to read |
|---|---|---|
| Sandbox detection | `sandbox-agent/research/detect-sandbox.md` | How to detect if running inside a sandbox |
| Process / terminal design | `sandbox-agent/research/process-terminal-design.md` | Sandboxed process design — pairs with `isolation: "worktree"` |

### 10.7 Collaborator-AI — Electron client (limited applicability)

| Topic | Path | What to read |
|---|---|---|
| ACP agent IPC | `collaborator-ai/collab-electron/src/main/acp-agent.ts` | Per-pane ACP agent management |
| tmux glue | `collaborator-ai/collab-electron/src/main/tmux.ts` | Pane-as-process pattern |

---

## 11. Internal AGH — Load-Bearing Paths Cited Across Slices

For navigation: the files most frequently cited as "the place to change" or "the place to read" across the ten slices.

### 11.1 Session lifecycle
- `internal/session/manager.go:39, 79-148, 302-593` — `CreateOpts`, hook configuration, lifecycle methods
- `internal/session/manager_lifecycle.go:20, 104, 145, 184-206` — `Create`, `watchProcess`, `finalizeStopped`, MCP resolution
- `internal/session/manager_start.go:56, 129, 187-197, 221, 270-280, 304, 389-404, 417-427` — `prepareCreateStart`, `startSession`, `joinNetworkPeer` ordering, `sessionStartOpts`, env injection
- `internal/session/manager_helpers.go:55, 67, 92, 130, 148` — permission selection, capability resolution, network join
- `internal/session/manager_hooks.go:82-475, 163-188, 301-323` — full hook event dispatch surface
- `internal/session/interfaces.go:35-49, 53-58, 71, 225-229` — `NetworkPeerCapability`, `NetworkPeerJoin`, `PromptInputAugmenter`, `ApproveRequest`
- `internal/session/session.go:36, 54, 78, 382-454` — `Type` constants, `Channel`, `markRuntimeStalled`, `observeRuntimeActivity`
- `internal/session/prompt_activity.go:84-122, 124-133, 170-277, 411-413` — watchdog supervisor, activity events, timeout/cancel/stop
- `internal/session/network_peer.go:9-31` — `networkPeerCapabilities` projection
- `internal/session/synthetic_prompt.go` — synthetic re-prompt path (already exists, unused for autonomy)
- `internal/session/prompt_overlay.go:12, 18, 36` — `StartupPromptContext`, `StartupPromptOverlay`
- `internal/session/crash_bundle.go` — crash bundle persistence
- `internal/session/liveness.go`, `notifier.go` — liveness/notifier surface

### 11.2 Network protocol & runtime
- `internal/network/envelope.go:14-34, 16-34, 168-185, 217-225, 218, 227-235, 238, 248, 251, 258, 260, 268-288, 291, 302-307` — Kind enum, Envelope struct, all Body types
- `internal/network/router.go:111-163, 174, 200-248, 251, 293, 412-414, 600-697, 642-697, 764-778, 867-878, 914-917, 929` — send/receive, dedup, lifecycle dispatch, whois, replay window
- `internal/network/peer.go:14, 24, 36, 55, 60, 124, 136, 179-184, 276, 351, 452, 660-668` — `LocalPeer`, `RemotePeerEntry`, `PeerInfo`, registry maps, `RegisterLocal`, `MatchLocalPeers`
- `internal/network/manager.go:65-71, 335, 560, 576-620, 635, 650, 879, 914, 979` — `managedSession`, `JoinChannel`, `Send`, `ListPeers`, `ListChannels`, `Heartbeat`
- `internal/network/lifecycle.go:23-32, 107-138, 142-158, 217-235, 261-289, 307-311` — `Interaction` state machine, `OpenInteraction`, `validateInteractionDirection`, lifecycle effects per kind
- `internal/network/delivery.go:46, 63-82, 88-92, 742-989, 1018-1049` — `deliveryCoordinator`, drop reasons, `formatNetworkMessage`, `replyGuidanceContext`, `previewForBody`
- `internal/network/capability_brief.go:11-51` — peer-card capability projection
- `internal/network/capability_catalog.go:14, 24-40, 42-109, 299-313` — rich capability catalog over whois
- `internal/network/audit.go:18-26, 107-213, 131-169, 236, 256-308` — audit directions, file writer, normalized rows
- `internal/network/stats.go:74-105, 74-124` — runtime stats, network status
- `internal/network/tasks.go:42-58, 105, 120, 222-258, 255, 366-416` — peer task surface, channel binding validation
- `internal/network/transport.go:355-374` — NATS subject conventions
- `internal/network/rules/channel.go:5` — channel name regex grammar
- `internal/network/validate.go:91` — `ValidateChannel`

### 11.3 Task subsystem
- `internal/task/manager.go:18-39, 57-69, 174-233, 237-264, 474-485, 518, 837-913, 1171-1176, 1282, 1330-1378, 1392-1441, 1428, 1444-1505, 1589-1656, 1685-1745, 2775-2814, 2811` — task service, `CreateTask`, `CreateChildTask`, lifecycle, `EnqueueRun`, `ClaimRun`, `StartRun`, `CompleteRun`/`FailRun`, `RecoverRunOnBoot`, event recording
- `internal/task/types.go:21-100, 108, 228-298, 233, 243, 253-258, 387-453, 406-416, 414, 481-487` — `Status`/`RunStatus` enums, `Task`/`Run`/`Event`/`RunIdempotency`/`Patch`/`Dependency`, `RunQuery`
- `internal/task/interfaces.go:9-39, 81-98, 86, 135-141` — `Manager`, `RunStore`, `SessionExecutor`
- `internal/task/actors.go:7, 8-15, 19-83, 34-47, 42` — `FullAccessAuthority`, agent-session actor minting
- `internal/task/limits.go:14-15` — `MaxHierarchyDepth = 8`
- `internal/task/live.go:95-127, 649-681` — per-task SSE `Stream`, fan-out
- `internal/task/live_types.go:104-115` — `RunOperationalSummary`

### 11.4 Storage
- `internal/store/globaldb/global_db.go:267-326, 335-371, 338-342, 344-349` — `tasks` and `task_runs` DDL (no lease columns)
- `internal/store/globaldb/global_db_task_aux.go:611-655, 702-742, 733, 1059-1095, 1334-1356` — `ReserveQueuedRun`, single-open-run gate, `withTaskImmediateTransaction`
- `internal/store/globaldb/global_db_network_channels.go:30-47, 141-154` — channel metadata table
- `internal/store/globaldb/global_db_network_messages.go` — `network_timeline_log` (echo, trace, etc. — never observed by memory)
- `internal/store/sessiondb/session_db.go:23-67, 324-376, 379-401, 464-476, 478-495, 514-542` — append-only events, query, history, writer loop, `writeEvent`

### 11.5 Memory
- `internal/memory/types.go:25-33, 39-47, 50-58, 57, 61-65, 95-101, 124-134, 236-248` — `Scope`, `Header`, `WriteOpts`, `OperationHistoryQuery`, `SearchOptions`, `Backend` interface, `DefaultScopeForType`
- `internal/memory/store.go:24-25, 36-46, 149-172, 327-374, 506-545, 546-600, 561-579, 1000-1007, 1105-1115` — write/read/scan/index, atomic write, `pathFor`
- `internal/memory/catalog.go:31, 34-87, 78-86, 96-101, 458-521, 545-548, 552, 568-637` — FTS5 catalog, operation log (always `"daemon"`), `replaceScope`, queries
- `internal/memory/recall.go:14, 22-59, 53-58, 61-101, 72-76` — per-turn recall augmenter, render
- `internal/memory/assembler.go:50-95` — prompt-section injector
- `internal/memory/dream.go:171-213, 314-361` — consolidation gates + spawn
- `internal/memory/consolidation/runtime.go:259-260, 410-464, 426` — dream session lifecycle
- `internal/memory/prompt.go:5-51` — 4-phase consolidation prompt
- `internal/memory/staleness.go:28-40` — freshness calculation
- `internal/memory/lock.go:21-145` — dream lock

### 11.6 Skills, tools, capabilities
- `internal/skills/types.go:21, 33-47, 58-64` — `Skill`, source precedence, `Provenance`
- `internal/skills/registry.go:29-88, 40, 127-144, 147-200, 504, 705-715` — `Registry`, `LoadContent`, `ForWorkspace`, `GlobalVersion`
- `internal/skills/loader.go:138-200` — FS scan
- `internal/skills/bundled/embed.go:1-17` + `bundled/skills/` — bundled SKILL.md
- `internal/skills/resource.go:13-87` — `SkillResourceSpec`
- `internal/skills/watcher.go:46-118` — 3s poll watcher
- `internal/skills/mcp_sidecar.go:15-89` — `mcp.json` merge
- `internal/skills/mcp.go:36-100, 131-150` — MCP resolver
- `internal/skills/catalog.go:36-110, 47, 65, 96-110` — `<available-skills>` block
- `internal/registry/installer.go:21-86, 42-100` — marketplace installer + verification
- `internal/registry/installer_checksum.go` — checksum verification
- `internal/tools/tool.go:14-25, 91-97, 91-136, 133-136` — `Tool` record, `ToolProvider` interface (no `Call`)
- `internal/tools/resource.go:12-61` — `ToolResourceKind = "tool"`
- `internal/toolruntime/registry.go:128-565` — process registry (interrupt scopes)
- `internal/config/capabilities.go:28-44, 43, 593` — `CapabilityDef`, `CapabilityCatalog`, `computeCapabilityDigest`
- `internal/config/agent.go:17, 17-28, 22, 26` — `AgentDef`, `Capabilities`, `Tools`

### 11.7 Daemon, observability, hooks
- `internal/daemon/composed_assembler.go:126, 154` — `AssembleStartup`
- `internal/daemon/prompt_sections.go:53, 63, 99, 148` — `PromptSectionDescriptor`, default chain, bundled provider
- `internal/daemon/section_selector.go:33` — `Select`
- `internal/daemon/harness_context.go:201, 477, 485` — `ResolvePrompt`, `resolveSections`, `ChannelBound`
- `internal/daemon/boot.go:135-194, 1419-1444, 796-834, 982-1045` — boot sequence
- `internal/daemon/task_runtime.go:227-305, 240-305, 261-267, 264, 271-281, 289, 327-350` — task runtime wiring + boot recovery
- `internal/daemon/orphan.go` — orphan reaper
- `internal/daemon/harness_observability.go:296` — per-section observability digests
- `internal/daemon/harness_detached_work.go:1-100` — detached-work template
- `internal/daemon/tool_mcp_resources.go:257-353` — `toolMCPSourceSyncer`
- `internal/observe/observer.go:109-136, 421-461, 464-496, 505-529, 616-630, 616-698, 632-659, 661-698` — Observer, session lifecycle, `event_summaries`, `token_stats`, `permission_log`
- `internal/observe/health.go:78-96, 99-160, 187-199, 228-229, 295-347` — `Health`, `Activities`, agent probes
- `internal/observe/query.go:52-87` — `QueryHookRuns`
- `internal/observe/retention.go:180-216, 198-216` — sweep loop
- `internal/observe/bridges.go:91-156, 159-226` — bridge health
- `internal/observe/tasks.go:19-27` — task health view
- `internal/hooks/events.go:46-97, 80-97` — hook event registry
- `internal/hooks/telemetry.go:31-46, 31-110, 120-130, 152-159, 301-313` — counters, latency, drop, depth violation
- `internal/hooks/types.go:259-272` — `HookRunRecord`
- `internal/diagnostics/redact.go:22-45` — token/secret redaction
- `internal/logger/logger.go:44-89` — slog logger

### 11.8 API, CLI, transport
- `internal/api/contract/contract.go:38, 120-129, 294-300, 467-475` — session info, `AgentPayload`, `PeerCard`
- `internal/api/contract/bundles.go:83, 94, 98` — `DeclaredNetworkChannelPayload`, `BundleNetworkSettingsPayload`
- `internal/api/core/network_details.go:44, 454, 471, 584, 611` — channel CRUD
- `internal/api/core/handlers.go:389-427` — handlers
- `internal/api/core/session_stream.go` — session SSE
- `internal/api/core/skills.go:15-180, 109-179` — skills HTTP
- `internal/api/core/bundles.go:123, 138` — bundle activation
- `internal/api/httpapi/routes.go:41, 67, 78, 91, 96-97, 162, 182-188, 263` — HTTP routes
- `internal/api/httpapi/handlers.go:124` — `privilegedMutationGuard`
- `internal/api/httpapi/server.go:24` — SSE poll cadence (100ms)
- `internal/api/udsapi/routes.go:228` — UDS routes
- `internal/cli/root.go` — root cobra config
- `internal/cli/client.go:443` — `clientFromDeps`
- `internal/cli/network.go` — network CLI
- `internal/cli/task.go:38-55, 481-761, 535-567, 569-593` — task CLI
- `internal/cli/session.go:35, 67` — session CLI
- `internal/cli/skill_commands.go:15-318, 253` — skill CLI
- `internal/cli/whoami.go:1-44` — env echo (no daemon round-trip)
- `internal/cli/format.go` — output formats
- `internal/extension/host_api.go:760` — extension `sessions.create` RPC (closest existing precedent for spawn surface)
- `internal/extension/host_api_bridges.go:673` — bridge-driven session creation

### 11.9 Automation
- `internal/automation/dispatch.go:21-22, 200-260, 215, 321-365, 367-498, 399-441, 443-498, 556-657, 588, 596-606, 608-614, 929-938, 947-961, 963-968, 1012-1030` — Dispatcher, gate, dispatch flow, fire-limit, task-backed delegation
- `internal/automation/schedule.go:63-128, 188-194` — Scheduler with per-job goroutines
- `internal/automation/trigger.go:163-200, 172-177` — TriggerEngine with index-driven matching
- `internal/automation/manager.go:1180, 1200` — actor recording
- `internal/automation/model/template.go` — prompt template interpolation

### 11.10 Internal docs / prior analysis (cross-references)
- `docs/ideas/orchestration/multi-agent-patterns-analysis.md` — choreography vs orchestration; §3.1, §3.4, §3.6, §6
- `docs/ideas/from-claude-code/_meta.md` — index of all claude-code analyses
- `docs/ideas/from-claude-code/analysis_prompt_architecture.md` — prompt architecture deep dive
- `docs/ideas/from-claude-code/analysis_multi_agent.md` — coordinator mode, sub-agents, fork, task notifications
- `docs/ideas/from-claude-code/analysis_memory_autonomous.md` — memdir + extractMemories + autoDream
- `docs/ideas/from-claude-code/analysis_tool_system.md` — `buildTool`, `ToolSearch`, deferred loading
- `docs/ideas/from-claude-code/analysis_query_engine.md` — query loop, transitions, cost tracker, recovery
- `docs/ideas/from-claude-code/analysis_services_infra.md` — VCR fixture, services
- `docs/ideas/from-claude-code/our_system_kernel.md:101-117, 336-342` — earlier workgroup hierarchy + deprecated `agentTypeTools`
- `docs/ideas/from-claude-code/our_system_cli.md` — CLI surface analysis
- `docs/ideas/network/agora-spec-v0.1.md` and `agora-spec-v0.2.md:51, 213, 286-295, 329-549, 339-349, 387, 469-494, 516-549, 553-732, 652-667, 738-754` — Agora protocol drafts (echo, tribute, thread, recipe)
- `docs/ideas/network/agora-recipe-design.md` — recipe design
- `docs/ideas/network/draft_3.md:121-156` — Kiosko seven-acts vocabulary
- `docs/ideas/network/draft_5.md:140-156, 154` — ANP NACK reasons + 3-verb model
- `docs/ideas/network/agora-council_round1.md:40-44` and `round2.md` — council debates
- `docs/ideas/memory-gaps/README.md:74-98, 120-135, 179-188` — prior memory gap analysis (PT-BR)
- `docs/ideas/extensability/analysis.md:282-294, 309-324` — extensibility patterns + supply-chain
- `docs/ideas/qa-e2e/README.md:144-156, 860-873` — QA / E2E backlog

---

---

## 12. Closing Observation

Across ten independent deep-research passes, no slice concluded "we need to start over." Every slice concluded the same thing: **the substrate is mature; the autonomy layer was simply never wired in.** The pattern is so consistent it becomes the strongest possible argument that AGH's architecture is sound.

What's left is **integration work** — making the data structures and lifecycle hooks that already exist talk to each other through one coherent autonomy kernel. The four-pillar roadmap above sequences that work so that each pillar amplifies the next:

1. **Situation Surface** gives the agent eyes.
2. **Agent Kernel CLI** gives it hands.
3. **Autonomy Kernel (scheduler + spawn + comm verbs)** gives the world a metabolism.
4. **Memory + Self-Correction Loop** gives the system a nervous system.

The greenfield-alpha rule lets us ship Pillar 3's wire-format bump (`agh-network/v1`) and schema additions (lease columns, scopes, parent IDs) cleanly, without legacy carry-forward. The ten changes in §6 are the autonomy MVP — they're each small, each touches a single package, and each is shipped with the existing test conventions (`make verify` blocking gate, `-race`, table-driven, ≥80% coverage).

The honest assessment is that AGH is **two coordinated releases away from being a real autonomous agent OS** — not because the work is small, but because the substrate already paid for the hard parts. Now we wire it up.
