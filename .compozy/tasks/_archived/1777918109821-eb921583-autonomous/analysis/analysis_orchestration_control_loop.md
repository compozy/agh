# Orchestration Control Loop — Autonomy Gap Analysis

> **Slice owner:** the orchestration control loop — coordinator/dispatcher patterns, swarm vs hierarchy, scheduling, leader election, backpressure.
> **Date:** 2026-04-25.
> **Status:** read-only audit; no code modified.

---

## 1. TL;DR

AGH today has **four independent caller-driven dispatchers** (`automation.Dispatcher`, `task.Service`, `network.Router` + `deliveryCoordinator`, `session.Manager`) and **zero global control loop**. Nothing in the daemon ever asks "I have N idle agents and M ready tasks — who should run what?". Every `Dispatch`/`EnqueueRun`/`Send` call is initiated by a caller who has *already* decided which agent runs the work; the runtime only enforces concurrency caps, fire-limits, idempotency, and queue ordering at the per-record level. There is no priority queue across tasks, no backpressure that blocks producers when the system is hot, no leader-elected coordinator, no work-stealing, no auction or contract-net negotiation, no idle-agent registry, no agent-side `pop next` loop. Boot reconciliation (`recoverTaskRunsOnBoot` at `internal/daemon/task_runtime.go:289`) is the **only** centralized scheduling decision the daemon ever makes — and it runs exactly once.

For autonomy we need to add **one** scheduler component that owns the loop "what work is ready × what agents are idle × who decides who runs what". The cleanest path is a **coordinator-agent** sitting on top of the existing `task.Service` queue, plus a **contract-net protocol** layered on `internal/network/router.go` so swarms outside one daemon can negotiate too. Everything else (priority queues, backpressure, capability matching, leader election) is a thin layer over building blocks that already exist as separate primitives but were never wired into a loop.

---

## 2. Current orchestration components — what each really does today

There is no package called `orchestrator` and no file named `dispatcher.go`. Instead, four separate runtimes each own a slice of dispatch and **never talk to each other** about scheduling:

### 2.1 `automation.Dispatcher` — schedule/trigger → session, one shot

`internal/automation/dispatch.go:200-260` defines `Dispatcher`. It exposes one method that matters for orchestration:

- `Dispatch(ctx, DispatchRequest) (*Run, error)` — `dispatch.go:321`.

What it does:

1. Acquires one slot from a buffered channel `gate chan struct{}` of size `maxConcurrent` (`dispatch.go:215`, `:947-961`). This is the **only** global concurrency primitive in AGH.
2. Reserves a `Run` row, optionally honoring a per-job/trigger fire-limit window (`dispatch.go:443-498`). Fire-limit is **per-definition**, not global.
3. Either creates a fresh `session.Session` and prompts it (`dispatch.go:398-441`), or — if `Job.Task != nil` — creates a task + enqueues a run + marks the automation run `RunDelegated` (`dispatch.go:556-614`). The two modes are mutually exclusive.
4. Retries on transient errors with backoff (`dispatch.go:329-365`).

What it does **not** do:

- Pick an agent. The agent name is hard-coded in `Job.AgentName` / `Trigger.AgentName` (`dispatch.go:963-968`).
- Decide priority. There is no priority field on `DispatchRequest`; `gate` is FIFO via Go channel semantics, not a priority queue.
- Backpressure beyond reject. If `gate` is full it returns `ErrConcurrencyLimitReached` (`dispatch.go:21-22`, `:368-375`). It does not block, queue, or shed.
- Watch agent health. Once it calls `sessions.Create` + `sessions.Prompt` it just collects events; it has no notion of "this agent is overloaded, send to a different one".

### 2.2 `automation.Scheduler` and `automation.TriggerEngine` — clock + event → dispatch

`internal/automation/schedule.go:63-128` — `Scheduler` owns durable cursor-driven scheduled-job dispatch. One **goroutine per job** runs the per-registration loop (`schedule.go:188-194`). It calls `dispatcher.Dispatch` when the cron tick fires.

`internal/automation/trigger.go:163-200` — `TriggerEngine` matches normalized activations (webhooks, hook completions, memory events, session lifecycle) against registered triggers and forwards to `dispatcher.Dispatch`. The matching is index-driven (`trigger.go:172-177`: `webhookIndex`, `deliveries`), not priority-driven.

Both feed the same shared `Dispatcher.gate` for global concurrency.

### 2.3 `task.Service` — typed work record + state machine, no loop

`internal/task/manager.go:57-69` defines `Service`. It implements `Manager` (`internal/task/interfaces.go:9-39`). The lifecycle methods that matter:

- `EnqueueRun(ctx, EnqueueRun, ActorContext) (*Run, error)` — `manager.go:1330`. Reserves one `queued` run via `store.ReserveQueuedRun` (`internal/store/globaldb/global_db_task_aux.go:611-655`).
- `ClaimRun(runID, ClaimRun, actor)` — `manager.go:1392`. Caller-pushed transition `queued → claimed`.
- `StartRun(runID, StartRun, actor)` — `manager.go:1444`. Caller-pushed transition `claimed → starting → running`. Calls the injected `SessionExecutor` to spin up an ACP session and bind it.
- `CompleteRun` / `FailRun` / `CancelRun` — caller-pushed terminal transitions.

Status enum in `internal/task/types.go:82-100`: `queued → claimed → starting → running → {completed | failed | canceled}`.

What is **not** here:

- **No `ClaimNextRun(criteria)`**. There is no atomic "give me any queued run that matches my capabilities" RPC. `RunQuery` (`types.go:481-487`) only filters reads.
- **No agent registry inside `task.Service`**. The manager does not know which agents are alive, what capabilities they advertise, or how many runs each is currently executing.
- **Priority lives on `Task`, not `Run`** (`types.go:40-54`, field `Task.Priority`); the queue is **not** sorted by priority anywhere — `MaxAttempts` is the only run-level quota and there is no priority-aware claim path. Compare with multica (`/Users/pedronauck/Dev/compozy/agh/.resources/multica/server/pkg/db/queries/agent.sql:122-140`) which orders queued tasks `ORDER BY atq.priority DESC, atq.created_at ASC` inside a `FOR UPDATE SKIP LOCKED` claim — AGH has neither the ordering nor the skip-locked claim.
- **No leases, no heartbeats**. See sibling analysis `analysis_task_discovery_claim.md` §3.2 — the schema is missing `claim_token`, `lease_until`, `heartbeat_at`. So even if a coordinator existed it has no fencing token to prevent two claims racing.
- **No backpressure**. `EnqueueRun` always succeeds if the task is in a queue-able state; the store check refuses only when a non-terminal run already exists for that exact task. There is no "queue depth too high, slow down" feedback to producers.

### 2.4 `network.Router` + `deliveryCoordinator` — per-session inbox, no swarm scheduling

`internal/network/router.go:111-163` — `Router` does **wire-level** routing: subject selection, presence preflight, replay-window dedup, lifecycle transitions. It is not a work scheduler. `Router.Send` (`router.go:251`) and `Router.Receive` (`router.go:293`) both operate on **one envelope at a time** with no awareness of "who in the channel is least busy".

`internal/network/lifecycle.go:23-32` — `Interaction` tracks one initiator/target pair through `submitted → working → needs_input → {completed | failed | canceled}`. **Both endpoints are fixed at open time** (`lifecycle.go:107-138` `OpenInteraction`); the protocol does not support "open an interaction with whichever peer in `channel:reviewers` accepts first".

`internal/network/delivery.go:63-82` — `deliveryCoordinator` owns **per-session inbound queues** with `maxQueueDepth` cap (`delivery.go:88-92` `inboundQueue`). On overflow it drops the oldest envelope and emits `deliveryDropReasonQueueFull` (`delivery.go:46`). This is **per-session backpressure** (drop policy), not cross-session coordination. The coordinator runs inline goroutines per delivery — there are no shared workers, no global rate limiter, no priority among sessions.

`internal/network/manager.go:65-71` `managedSession` — every session that joins a channel gets one entry; presence is published via periodic `greet` heartbeats (`router.go:200-248` `StartHeartbeat`). This is **service discovery + liveness**, not load-aware routing.

`internal/network/capability_catalog.go:62-109` — peers can answer `whois` queries that include a `capability_catalog`, returning a typed `whoisCapabilityCatalogPayload`. **Capability catalog is publishable but no router code consults it when picking a delivery target** — it is purely advertised metadata for human-facing UIs and prompt context.

### 2.5 `session.Manager` — per-session state machine, no peer-aware decisions

`internal/session/manager.go:302-593` — `Manager` is a **per-session** orchestrator: `Create`, `Prompt`, `StopWithCause`, `applyRuntimeDefaults`, `claimFinalization`. It has lifecycle hooks (`internal/session/manager_hooks.go:82-475`) for pre-create, post-create, pre-stop, turn-start, turn-end, message-start, message-delta, etc. — but those hooks fire **per-session events**, not orchestration decisions across sessions.

`Manager.SetTurnEndNotifier` (`manager.go:388`) lets the network manager learn when a session finishes a turn (used by `network.Manager.OnTurnEnd` to advance interactions). There is no "session went idle, register me with the scheduler" callback.

### 2.6 `internal/daemon/boot.go` — composition only, no orchestration

`boot.go:135-194` `Daemon.boot` wires the runtimes in fixed order: config → prompt providers → runtime services → tasks → network → hooks → automation → bundles → resource reconcile → extensions → settings → servers → finalize. After `bootFinalize` (`boot.go:1419-1444`) the daemon hands control to the HTTP/UDS servers and **the only background goroutines that survive are**: `Scheduler` job loops, network heartbeats, hook subprocess executors, retention sweepers, and `consolidation.Runtime` (dream). **None of those is a "find work + assign agents" scheduler.**

---

## 3. What is NOT a control loop today

This section is the honest gap list.

| # | Missing piece | Where it should live | Today's substitute |
|---|---|---|---|
| **G1** | **Idle-agent registry**. Nothing tracks "session S in workspace W is alive, has skills X/Y/Z, currently has 0 prompts in flight". | A new `internal/scheduler/agents.go` (or a method on `session.Manager` + capability projection). | Capability is published to network peer cards (`internal/network/capability_brief.go`) but never read by a scheduler. `session.Manager.IsPrompting(id)` (`manager.go:400`) exists but is not aggregated. |
| **G2** | **`ClaimNextRun(criteria)` atomic pop**. No "give me the next queued task matching capability `code-review` in workspace `acme`". | `task.Service` (sibling analysis spells this out — see `analysis_task_discovery_claim.md` §3.2). | `RunStore.ListTaskRunsByStatus` exists at `internal/task/interfaces.go:86` but is read-only; no atomic claim. Multica solves this with `FOR UPDATE SKIP LOCKED + ORDER BY priority DESC` — AGH has neither. |
| **G3** | **Cross-task priority queue**. Tasks have a `Priority` enum (`internal/task/types.go:40-54`) but no code orders the queue by it. | `RunStore.ReserveQueuedRun` ordering + `ClaimRun` selection. | FIFO by `queued_at` only (implicit via creation order). |
| **G4** | **Backpressure across channels**. Producers (CLI `agh task create`, network peer `EnqueueRunFromPeer`, automation, extensions) all enqueue freely; nothing throttles when 200 runs are queued and zero agents are idle. | A bounded admission controller in `task.Service.EnqueueRun` + a `429`-style ingress reject. | Only `automation.Dispatcher.gate` (`dispatch.go:215`) caps **automation-launched sessions** at `maxConcurrent`. Network deliveries cap **per-session inbox** (`network/delivery.go:88-92`). Neither protects the task queue. |
| **G5** | **Leader / coordinator agent**. There is no agent that owns "look at the system, decide what should run next". | A first-class agent role + bootstrap policy in `daemon`. | The closest existing thing is `automation.Dispatcher` deciding to spawn a fixed agent for a fixed schedule — that is a **schedule executor**, not a coordinator. |
| **G6** | **Leader election**. Only one daemon owns the SQLite home (`internal/daemon/lock.go`); within that daemon, all goroutines are equal — there is no "which goroutine is the orchestrator right now" semantic. Across daemons, AGORA peers are **all equal**: there is no protocol primitive that says "this peer is the coordinator for channel X". | New protocol verb (e.g. `lead`) in `internal/network/envelope.go` Kind set, plus a `network/leader.go` runtime. | None. The closest analog is the **interaction lifecycle owner** which is implicit (the `Initiator` in `internal/network/lifecycle.go:23-32`) but that is per-interaction, not per-channel. |
| **G7** | **Auction / contract-net negotiation**. No `call-for-proposals` → `bid` → `award` flow exists. | New `Kind` in `internal/network/envelope.go` (e.g. `propose`, `bid`, `award`) + state machine in `lifecycle.go`. | `direct` + `interaction_id` (`router.go:32-44`) gives request/response between **two pre-known peers**, but there is no broadcast-RFP primitive. |
| **G8** | **Work-stealing / rebalancing**. If agent A is overloaded and agent B is idle, nothing moves the work. | Scheduler tick + lease takeover. | The boot recovery path (`internal/task/manager.go` `RecoverTaskRunsOnBoot`, called from `internal/daemon/task_runtime.go:289`) is the **only** rebalance — and it runs once at boot. |
| **G9** | **Failure-aware routing / circuit breaker**. The orchestration analysis in `docs/ideas/orchestration/multi-agent-patterns-analysis.md` §3.4 already calls this out: AGH has no circuit breaker pattern. If agent A keeps failing, automation keeps sending it work. | Wrap `automation.SessionCreator` and `task.SessionExecutor` with a circuit-breaker decorator. | None. `internal/automation/dispatch.go:329-365` has retry-with-backoff but no per-agent open/closed/half-open state. |
| **G10** | **Workflow correlation across sessions**. Same gap from `multi-agent-patterns-analysis.md` §3.6: per-session observability exists, per-workflow does not. The orchestrator needs to know "these 5 sessions were spawned for one user goal". | `workflow_id` field on `Run` + propagate through `Session.Metadata` + observer query. | `internal/observe/observer.go:109-136` tracks per-session, no group key. |

---

## 4. Pattern catalog for autonomous orchestration

Each row: **what it is → where AGH would support it today → what's missing**.

### 4.1 Coordinator-agent (centralized orchestrator)

> *One designated agent owns workflow decomposition and worker delegation. Workers never talk to each other.*

- **AGH today:** `automation.Dispatcher` is the only thing that even resembles a coordinator — it reads a job spec, picks the agent, spawns a session, collects output. But it is **schema-driven, not LLM-driven**: it cannot synthesize, it cannot decide to fan out, it has no tools beyond "create one session for the agent named in the job". The Claude Code coordinator-mode pattern from `docs/ideas/from-claude-code/analysis_multi_agent.md:5-26` (loses tools, gets `Agent`/`SendMessage`/`TaskStop`, synthesizes findings) does not exist anywhere.
- **What's needed:**
  1. A **coordinator role** in agent definitions (`internal/config` agent def loader) that: strips normal tools, injects a coordinator system prompt, exposes `task.create` + `task.claim` + `task.complete` + `agent.list` + `agent.send` as in-process tools.
  2. The `task.Service` becomes the coordinator's blackboard — it spawns child tasks, watches their `Run` events via `task.LiveService`, synthesizes results.
  3. Bootstrap policy in `daemon` so the coordinator is always running (or spawned on demand by `automation.Trigger` when a workspace gets its first work).

### 4.2 Hierarchical (manager / worker; supervisor tree)

> *Tree of agents — manager at top, mid-level managers per domain, workers at leaves. Each level only talks up + down.*

- **AGH today:** `Task.ParentTaskID` (`internal/task/types.go:233`) and `MaxHierarchyDepth = 8` (`internal/task/limits.go:14-15`) already give us a **task hierarchy** — but no **agent hierarchy** that maps onto it. `task.Service.CreateChildTask` exists (`interfaces.go:12`) but the parent/child relationship is data-structural, not control-structural; nothing forces "the agent that owns parent task P also owns child claim arbitration".
- **What's needed:**
  1. An **agent-tree** abstraction parallel to the task tree: `manager_agent_id` on `Task`, with a constraint that the manager is the only ActorIdentity allowed to `EnqueueRun` for child tasks under it.
  2. A supervisor tick (analog of Erlang OTP supervisors) in `task.Service.LiveService` that watches children and re-enqueues failures up to `MaxAttempts`.
  3. Reuse `internal/network/lifecycle.go`'s `Initiator/Target` model to express the manager-worker pairing on the network too.

### 4.3 Market / auction (bidding for tasks)

> *Tasks are posted; agents bid (with cost/time/confidence); a market clearing rule picks the winner.*

- **AGH today:** **none of the building blocks exist**. There is no `bid` envelope kind, no cost metadata on tasks, no confidence projection from agents.
- **What's needed:**
  1. Two new `Kind` values in `internal/network/envelope.go` (e.g. `propose` for the call, `bid` for responses; reuse `award` semantics on top of `direct`).
  2. A new `Task` field `Auction *AuctionPolicy` describing pricing rule (lowest cost / highest confidence / first reasonable bid).
  3. State machine extension in `internal/network/lifecycle.go` for `auction_open → bidding → awarded → working → …`.
  4. This is heavyweight; **only worth doing if cross-org coordination is a goal**. Inside one daemon a coordinator-agent (§4.1) is simpler.

### 4.4 Blackboard / swarm (shared state, agents pull autonomously)

> *No coordinator. Agents read a shared workspace, decide independently to act, write back results. Conflict resolution via locks/leases.*

- **AGH today:** `task.Service` is **almost** a blackboard — it is a typed store with events (`internal/task/manager.go:18-39`), it has dependency edges (`Dependency`, `internal/task/types.go:253-258`), and it broadcasts changes via `LiveService.Stream`. But there is no `ClaimNextRun(criteria)` so agents cannot autonomously pull (§G2). The closest swarm primitive is the network `say` broadcast (`internal/network/router.go:412-414`) which fans out to all peers in a channel — but `say` carries free-text, not work.
- **What's needed:**
  1. `task.Service.ClaimNextRun(ctx, ClaimCriteria)` with `FOR UPDATE SKIP LOCKED` semantics (mirror the multica query referenced above).
  2. A `task.events.queued` notifier (or `network.say` envelope variant) that wakes idle agents instead of forcing them to poll.
  3. Lease columns + heartbeat (already covered as G2 in this doc and §3 of `analysis_task_discovery_claim.md`).
  4. Capability index so agents only see runs they can do (today they would see all queued runs in a workspace).

### 4.5 Contract-net protocol (FIPA-style)

> *Hybrid: manager broadcasts a call-for-proposals, capable agents bid, manager awards one, awardee reports progress, manager confirms.*

- **AGH today:** the network protocol (`internal/network/router.go`, `lifecycle.go`) has all the **transport** but none of the **roles**. `direct` + `interaction_id` is a fixed two-party pair; there is no broadcast call with collected typed responses.
- **What's needed:**
  1. Three new envelope kinds: `propose` (broadcast call, includes capability spec + deadline), `bid` (response carrying agent identity + cost), `award` (acceptance to one bidder, becomes a directed `direct` interaction).
  2. Lifecycle extension: `proposed → bidding (timer) → awarded → working → {completed | failed}`.
  3. Reuse the existing `Interaction` table (`internal/network/lifecycle.go:23-32`) keyed by the awarded `interaction_id`.
  4. **This is the most flexible primitive** — coordinator-agent, market, and blackboard can all be built on top of it.

---

## 5. Reference comparisons

### 5.1 multica — pull-based per-runtime claim with priority

`/Users/pedronauck/Dev/compozy/agh/.resources/multica/server/internal/service/task.go:176-237` `ClaimTask` and `:241-304` `ClaimTaskForRuntime` are the canonical pattern AGH is missing. Key moves:

- `EnqueueTaskForIssue` (`task.go:38-77`) and `EnqueueTaskForMention` (`:82-111`) write to `agent_task_queue` with explicit `Priority` (`task.go:67`).
- `ClaimTask(ctx, agentID)` checks `MaxConcurrentTasks` (`:201`), then runs the SQL claim query at `pkg/db/queries/agent.sql:116-140` which is `UPDATE … WHERE id = (SELECT … ORDER BY priority DESC, created_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED)`. **This single query is the entire scheduler**: priority + age + max-concurrency + atomic claim.
- `ClaimTaskForRuntime` (`:241-304`) walks all pending tasks bound to a runtime, deduplicates by agent, and tries to claim — this is a primitive **work-stealing** pattern at the runtime granularity.
- Idempotent `CompleteTask` handling (`:348-463`) handles parallel races where multiple workers claim the same logical work — uses `pgx.ErrNoRows` as "already finalized" signal.

**AGH directly applicable:** port the SQL pattern to SQLite (the SQLite analog is `RETURNING` + a sub-select; SQLite does not support `FOR UPDATE SKIP LOCKED`, so the AGH equivalent must use either `BEGIN IMMEDIATE` transactions or an in-memory lock around the reservation step). The conceptual move (priority + age ordering, atomic claim, max-concurrency check) is exactly what §G2/G3/G8 ask for.

### 5.2 hermes — single-agent, no orchestration loop

`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/agent/` is a **single-agent** runtime — adapters for Anthropic, Bedrock, Codex, Gemini, plus context engine and memory. Grep for `coordinator|orchestrat|scheduler|dispatcher` returns only credential-source and memory-manager files (string matches). Hermes is not a multi-agent orchestrator and offers no coordinator pattern to copy. The relevant lesson: **the orchestrator does not have to live in the agent**; AGH's daemon is the right place.

### 5.3 collaborator-ai — Electron front-end, no orchestrator

`/Users/pedronauck/Dev/compozy/agh/.resources/collaborator-ai/collab-electron/src/main/` is an Electron main process plus ACP agent IPC (`acp-agent.ts`) and tmux glue (`tmux.ts`). It does **session-level orchestration of one agent per pane**, not multi-agent coordination. The closest pattern is the tmux-pane-as-process model — already covered by AGH's session model. No applicable scheduler pattern.

### 5.4 Claude Code (from `docs/ideas/from-claude-code/analysis_multi_agent.md`)

This is the highest-value reference because the patterns are AGH-shaped:

- **Coordinator mode** (`coordinator/coordinatorMode.ts` per the analysis at §1) — the coordinator loses normal tools and gets `Agent`/`SendMessage`/`TaskStop`. It explicitly synthesizes findings before delegating again. This maps onto AGH as a coordinator agent definition + a scoped tool surface.
- **Background agents = LocalAgentTask**, results returned as `<task-notification>` XML in user-role messages (`§2.C`). AGH today does not have this notification mechanism — it has the `Run` lifecycle events but they don't auto-inject into the parent session prompt.
- **Fork-as-primitive** (`§5`) — autonomous parallel decomposition with cache-optimized message construction. AGH has no analog; the only fan-out today is `automation.Dispatcher` calling `Dispatch` once per scheduled job.
- **Team config** at `~/.claude/teams/{team-name}/config.json` (`§6`) is a lightweight service registry. AGH's analog would be a `team.toml` resource type wired through the existing resource kernel (`internal/resources/`) — currently does not exist as a kind.
- **Task list as coordination primitive** (`§6 + §7`) — exactly the AGH `task.Service`. The Claude Code claim discipline (workers prefer lowest ID) is a soft policy; AGH should make it a hard policy in the SQL claim query.

---

## 6. Concrete proposal — coordinator-agent + task-blackboard, with contract-net stubs for cross-daemon

### 6.1 Recommended path

Pick **coordinator-agent + task-as-blackboard inside one daemon**, with a small **contract-net protocol** veneer on the network for swarm scenarios that span daemons. Reasons:

1. The biggest gap (§G1, G2, G3, G4, G8) is **inside one daemon**: there is no scheduler. Fixing it where the data already lives (`task.Service`) is cheaper than designing a new component.
2. Coordinator-agent matches the proven Claude Code pattern, matches the talk's recommendation in `docs/ideas/orchestration/multi-agent-patterns-analysis.md` §3.1 (Layer A = orchestration), and reuses the existing session manager unchanged.
3. Contract-net on the network lets us extend later without a rewrite — the network protocol stays loosely coupled to scheduling.

### 6.2 Concrete additions per package

**`internal/scheduler/` (NEW)** — the missing component.

```
internal/scheduler/
  agents.go           # idle-agent registry indexed by capability
  claim.go            # ClaimNextRun(criteria) atomic pop, priority-aware
  lease.go            # lease_until + heartbeat tracking (consumes G2)
  loop.go             # Loop.Tick() — match idle agents × ready runs
  policy.go           # ClaimPolicy interface (FIFO / priority / round-robin / capability-weighted)
  notifier.go         # task.queued + agent.idle event wiring
  doc.go
```

- `Loop.Tick()` runs on a `notifier`-driven channel (no polling): subscribes to `task.run_enqueued` (already emitted at `internal/task/manager.go:1367`) and `session.post_create` / `session.turn_end` (already in `internal/session/manager_hooks.go:163-188`, `:301-323`), then attempts `ClaimNextRun` for each idle agent that matches the new run's capability.
- Owns one `sync.RWMutex`-protected map of idle agents; `agents.go` adds/removes entries on session lifecycle hooks.
- Calls into a new `task.Service.ClaimNextRun(ctx, ClaimCriteria, actor)` that wraps the multica-style atomic SQL claim in a `BEGIN IMMEDIATE` SQLite transaction.

**`internal/task/` (EXTEND)**

- Add `ClaimNextRun(ctx, ClaimCriteria, actor)` to the `Manager` interface (`internal/task/interfaces.go:9-39`) and to `Service`.
- Extend the `RunStore` (`interfaces.go:81-98`) with `ReserveNextQueuedRun(ctx, ClaimCriteria) (Task, Run, bool, error)` mirroring the existing `ReserveQueuedRun` shape.
- Add `RequiredCapability *CapabilitySpec` to `Task` and `CreateTask`. The spec mirrors `session.NetworkPeerCapability` so it interoperates with the network capability catalog (`internal/network/capability_catalog.go:24-40`).
- Add `Lease`/`Heartbeat` columns per sibling analysis §G2 (cross-reference `analysis_task_discovery_claim.md`).
- Add `WorkflowID` field for cross-session correlation (G10).

**`internal/automation/` (REWIRE, not extend)**

- Stop hard-coding `Job.AgentName` on every dispatch path. Instead: when `Job.RouteByCapability == true`, the dispatcher creates a task + enqueues a run with `RequiredCapability` set, and lets the new scheduler pick the agent.
- Keep the explicit-agent path for backward compatibility but mark it as the **fast path for direct schedule execution**, not the default.
- The `Dispatcher.gate` (`internal/automation/dispatch.go:215`) becomes a fallback throttle; the real cap moves to `scheduler.policy` so all producers (automation, network, CLI, agents) are rate-limited together.

**`internal/session/` (EXTEND)**

- Add a `CapabilityProvider` interface that publishes one session's capability set to the scheduler. Reuse `NetworkPeerCapability` (`internal/session/interfaces.go:35-49`) so a session that joins a network channel and a session that just registers locally use the same shape.
- Hook `manager_hooks.go` post-create / post-stop to call `scheduler.RegisterAgent` / `UnregisterAgent`.
- Hook turn-end (already exists at `manager_hooks.go:301-323`) to mark the session "idle" so the scheduler's tick can target it.

**`internal/network/` (EXTEND for cross-daemon)**

- Add three envelope `Kind` values in `envelope.go`: `propose` (broadcast RFP), `bid` (typed response), `award` (directed accept that opens a `direct` interaction).
- Extend `lifecycle.go` with an `auction` state set: `proposed → bidding → awarded → working → {completed | failed | canceled}`. Keep it strictly on top of the existing state machine — no breaking change.
- Add a new `network.AuctionRouter` (sibling to `network.Router`) that owns deadline timers, bid collection, and the clearing rule. Default rule: first-bid-from-capable-peer wins; pluggable for cost/confidence later.

**`internal/daemon/` (WIRE)**

- New `bootScheduler(ctx, state, cleanup)` in `boot.go` (`internal/daemon/boot.go:135-194`) sequenced **after** `bootTasks`, **before** `bootNetwork`. The scheduler depends on `task.Service` and `session.Manager`.
- Cleanup hook on shutdown to drain in-flight claims before stopping sessions.

### 6.3 Why a single coordinator inside the daemon, not in an agent

The orchestration literature (and the multi-agent analysis at `docs/ideas/orchestration/multi-agent-patterns-analysis.md` §6) is consistent: **the daemon is an OS, not an AI framework**. Sessions are processes. The OS kernel (Linux scheduler) does not run in user-space — and AGH's scheduler should not run in an LLM. The scheduler picks agents; the **coordinator-agent** then composes work for them. They are different concerns:

- `internal/scheduler/` = mechanical match of capability × queued work × idle agent. Deterministic, fast, unit-testable.
- coordinator-agent (a bundled agent definition) = LLM-driven decomposition: "this user goal becomes these 5 child tasks". Non-deterministic, slow, prompt-tuned.

Both must exist; conflating them is the mistake the talk warns against.

---

## 7. Open questions

1. **Single leader vs decentralized.** Inside one daemon a single in-process scheduler is obviously right (one process = one scheduler). Across daemons — should the coordinator be **elected** for a workspace/channel, or should every daemon run a peer-equal scheduler that auctions for cross-daemon work? Recommendation: start single-leader-per-daemon, defer election; revisit only when cross-daemon swarms become real workloads.
2. **Where the loop runs.** Two viable homes for the in-daemon scheduler:
   - **Daemon goroutine** (recommended) — boots with the daemon, wired through `internal/daemon/boot.go`, uses the existing notifier/hook plumbing. Survives restart. No LLM cost. Matches the OS-kernel framing.
   - **Bundled agent** — runs as a long-lived ACP session that uses scheduler tools. More flexible (LLM can override the policy) but adds latency, cost, and a dependency cycle (the scheduler that places agents is itself an agent the daemon must place first). Reject for now.
3. **Failure recovery.** Two failure modes need explicit answers before shipping:
   - **Scheduler crash / daemon restart.** Today `recoverTaskRunsOnBoot` (`internal/daemon/task_runtime.go:289`) is the only recovery. Once leases exist, the scheduler must on boot also reclaim expired leases and re-add the freed runs to the queue — extend `RecoverTaskRunsOnBoot` to include lease expiry.
   - **Agent crash mid-run.** Today the session crash-bundle path (`internal/session/crash_bundle.go`) exists but no scheduler observes it. The scheduler should subscribe to `session.post_stop` with `cause=crash` and immediately re-queue the bound run (subject to `MaxAttempts`).
4. **Backpressure semantics.** When `EnqueueRun` is called and the queue depth is over budget, do we (a) reject with `429`, (b) block the producer, (c) queue but emit a `system_health.queue_pressure` event so producers can back off voluntarily? The contract-net pattern naturally supports (c) via `receipt(rejected, reason_code: "busy")` — that primitive already exists in `internal/network/router.go`.
5. **Priority semantics.** `Task.Priority` is an enum (`low/medium/high/urgent`, `internal/task/types.go:40-54`). The scheduler needs a stable ordering — strict priority can starve low-priority work. Recommend **weighted round-robin within priority bands** as the default policy in `scheduler/policy.go`, with strict-priority as opt-in.
6. **Capability matching.** Should the matcher be exact (`capability.id == required.id`) or scored (semantic similarity over `summary`/`outcome`)? Start exact-match with explicit `version` constraints; defer scored matching until we have data showing the exact matcher is too restrictive.
7. **Coordinator-agent bootstrap.** Does the daemon auto-spawn the coordinator at boot for every workspace, or only when work appears? Auto-spawn-on-first-task seems right (avoids paying for an LLM session for empty workspaces) but introduces a cold-start latency on the first task — measure before deciding.

---

## 8. References to existing files cited in this analysis

- `internal/automation/dispatch.go:200-260, :321, :443-498, :556-614, :329-365, :947-961, :215, :963-968`
- `internal/automation/schedule.go:63-128, :188-194`
- `internal/automation/trigger.go:163-200, :172-177`
- `internal/task/manager.go:57-69, :1330, :1392, :1444, :1685-1745, :1367, :18-39`
- `internal/task/interfaces.go:9-39, :81-98, :86`
- `internal/task/types.go:21-38, :40-54, :82-100, :233, :253-258, :481-487`
- `internal/task/limits.go:14-15`
- `internal/task/actors.go:34, :42`
- `internal/network/router.go:111-163, :251, :293, :200-248, :32-44, :412-414`
- `internal/network/lifecycle.go:23-32, :107-138`
- `internal/network/delivery.go:63-82, :88-92, :46`
- `internal/network/manager.go:65-71`
- `internal/network/capability_catalog.go:24-40, :62-109`
- `internal/network/capability_brief.go:18-51`
- `internal/network/tasks.go:42-58, :222-258`
- `internal/session/manager.go:302-593, :388, :400`
- `internal/session/manager_hooks.go:82-475, :163-188, :301-323`
- `internal/session/interfaces.go:35-49`
- `internal/daemon/boot.go:135-194, :1419-1444, :796-834, :982-1045`
- `internal/daemon/task_runtime.go:227-305, :289`
- `internal/observe/observer.go:109-136`
- `docs/ideas/orchestration/multi-agent-patterns-analysis.md` (full doc, esp. §3.1, §3.4, §3.6, §4.2)
- `docs/ideas/from-claude-code/analysis_multi_agent.md` (esp. §1, §2.C, §5, §6, §7)
- Sibling: `.compozy/tasks/autonomous/analysis/analysis_task_discovery_claim.md` (esp. §3.2 — gap list aligned with G2 here)
- Reference: `/Users/pedronauck/Dev/compozy/agh/.resources/multica/server/pkg/db/queries/agent.sql:116-140` (`ClaimAgentTask`)
- Reference: `/Users/pedronauck/Dev/compozy/agh/.resources/multica/server/internal/service/task.go:176-304` (`ClaimTask` / `ClaimTaskForRuntime`)
