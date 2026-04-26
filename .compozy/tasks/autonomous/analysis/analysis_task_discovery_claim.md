# Task Discovery, Claim & Lease — Autonomy Gap Analysis

> **Slice owner:** task lifecycle from an agent's perspective — discovery, claim, lease, status update, completion, re-queue.
> **Date:** 2026-04-25.
> **Status:** read-only audit; no code modified.

---

## 1. TL;DR

Today an AGH task is a **typed audit record with a manager-driven state machine**, not a discoverable work queue. The `task.Service` (`internal/task/manager.go:57`) exposes `EnqueueRun → ClaimRun → StartRun → CompleteRun/FailRun`, but every transition is **caller-pushed**, every `claim_token`/`lease_until`/`heartbeat_at` column is **missing from the schema** (`internal/store/globaldb/global_db.go:335-371`), and the database reservation gate **rejects any second open run per task** (`internal/store/globaldb/global_db_task_aux.go:733`, `:1334`). Idle agents have no `pop next` API, no capability matching, no lease renewal, no escalation on stuck claims — so an autonomous agent literally cannot wake up, look at the queue, find work it can do, claim it safely, and report back without a human or pre-configured automation pinning the run to one specific session up-front.

For full autonomy we need: (a) a `ClaimNextRun(ctx, criteria)` that atomically pops a queued run that matches an agent's capabilities, (b) lease columns + heartbeat / takeover semantics, (c) a `task.run.queued` notifier so any subscribed agent can react to new work without polling, (d) an in-process `task.*` tool family on the agent side so a session can create child tasks, claim them, post progress, and complete them through the same trusted actor context the daemon already mints (`internal/task/actors.go:34`, `:42`).

---

## 2. Current task lifecycle (state machine + file:line refs)

### 2.1 Task statuses

`internal/task/types.go:21-38` defines `Status`: `draft → pending → blocked → ready → in_progress → completed | failed | canceled`.

The reconciler (`m.canonicalTaskStatus`, called from `reconcileTaskCascade`, e.g. `internal/task/manager.go:518`, `:1282`, `:1428`) is the **only** code that decides which of these a task may sit in — agents cannot mutate `Status` directly, only via `Patch` (`internal/task/types.go:406-416`) which doesn't even include a `Status` field, and via run-lifecycle calls that ripple back through `reconcileTaskCascade`.

### 2.2 Run statuses

`internal/task/types.go:82-100` defines `RunStatus`: `queued → claimed → starting → running → completed | failed | canceled`.

The schema enforces these via a CHECK constraint (`internal/store/globaldb/global_db.go:338-342`).

### 2.3 The end-to-end happy path today

| # | Event | Code | Who triggers it |
|---|---|---|---|
| 1 | `CreateTask` writes one row, emits `task.created` event | `internal/task/manager.go:174-233` | **Human (CLI/Web/UDS/HTTP)**, **automation dispatch** (`internal/automation/dispatch.go:588`), or **network peer** (`internal/network/tasks.go:120`). All three paths derive an `ActorContext` via the helpers in `internal/task/actors.go:19-83`. **Agents inside a session cannot call `CreateTask`** — there is no in-process tool exposed to them; the only agent-as-creator path is the `DeriveAutomationLinkedAgentSessionActorContext` (`internal/task/actors.go:42`) which is recorded but never wired to a session-callable surface. |
| 2 | Status reconciles to `ready` (or `blocked` if dependencies open, or `pending` for `draft → publish`) | `internal/task/manager.go:474-485`, `:518` | Manager-internal. |
| 3 | `EnqueueRun` reserves one queued run | `internal/task/manager.go:1330-1378` calling `store.ReserveQueuedRun` (`internal/store/globaldb/global_db_task_aux.go:611-655`) | **Human via CLI** (`internal/cli/task.go:535-567`), **automation** (`internal/automation/dispatch.go:596-606`), or **network peer** (`internal/network/tasks.go:255`). The store **refuses** to enqueue if any non-terminal run exists for the task (`internal/store/globaldb/global_db_task_aux.go:733`, `:1334`). |
| 4 | `task.run_enqueued` event recorded → fan-out via `EventObserver.OnTaskEvent` and SSE `Stream` subscribers | `internal/task/manager.go:1367`, `:2775-2814`; live fan-out at `internal/task/live.go:649-681` | Manager-internal, best-effort. **No notifier kind is dedicated to "queue has new work" — it is one event among many in a per-task stream**, only useful to clients already subscribed to that specific `taskID`. |
| 5 | `ClaimRun` flips `queued → claimed`, records `claimed_by` actor + `claimed_at` | `internal/task/manager.go:1392-1441` | **Human via CLI `agh task run claim <run-id>` (`internal/cli/task.go:569-593`)**, **automation** (the dispatcher does not actually call `ClaimRun` — it calls `EnqueueRun` then immediately delegates to a session it created itself, marking the parent automation run `RunDelegated`, see `internal/automation/dispatch.go:556-614`), or **network peer (only `EnqueueRun` is exposed; there is no `ClaimRunFromPeer`** in `internal/network/tasks.go:42-58`). The claim takes the entire **caller's** actor context — there is no `claim_next_for_capability` method, no atomic compare-and-set against a token, and the only mutual exclusion is the existence check inside `requireRunTransition` (`internal/task/manager.go:1417`). |
| 6 | `StartRun` flips `claimed → starting → running`, asks the injected `SessionExecutor.StartTaskSession` to spin up an ACP session and binds it via `AttachRunSession` | `internal/task/manager.go:1444-1505`, `:837-913` | **The same actor that claimed must call StartRun**, but nothing in the manager enforces "claimer == starter" — the only check is `actor.Authority.Write` (`internal/task/manager.go:1445`) and a status transition guard. |
| 7 | Session runs → user/agent prompts/tool calls/results stream into SessionDB. The task domain learns nothing about progress until `CompleteRun` / `FailRun` / `CancelRun` is called. | n/a | **There is no per-run heartbeat or progress hook from the session back into the task domain.** `RunOperationalSummary` in `internal/task/live_types.go:104-115` is computed *on demand* by joining session events when the UI asks — it is not a push channel. |
| 8 | `CompleteRun`/`FailRun` flips run terminal, reconciles task status. | `internal/task/manager.go:1589-1633`, `:1636-1656` | **Caller-pushed**. The runtime bridge (`SessionExecutor`) does call back into the manager when a session finishes — see the boot recovery path (`internal/task/manager.go:1685-1745`) — but the *normal* completion handshake is initiated externally (CLI `agh task run complete`, automation post-fire, or HTTP). |
| 9 | Boot recovery: orphaned `claimed/starting/running` runs are requeued, marked-running, or failed | `internal/task/manager.go:1685-1745`; daemon driver `internal/daemon/task_runtime.go:327-350` | **Daemon at startup only.** There is no live "this run hasn't pinged in N seconds, take it back" loop — only a one-shot reconciliation at boot. |

> **`HUMAN-DRIVEN STEP HERE`** — the entire chain from step 5 onward assumes either (a) a person at a CLI typing `agh task run claim`, or (b) a pre-configured automation that *creates and immediately delegates* the task to a specific agent it spawns itself. **There is no path where Agent X, idle in workspace W, asks "is there work for me?" and the daemon hands it the next matching queued run.**

### 2.4 Observability + audit

Every transition writes a typed event row (`internal/task/manager.go:18-39`) and fans out via two channels:

1. `EventObserver.OnTaskEvent` — daemon-injected reentry bridge for harness reuse (`internal/daemon/task_runtime.go:264`, `:271-281`).
2. `Stream(taskID)` — per-task SSE subscriber (`internal/task/live.go:95-127`); **scoped to one task root**, so subscribing to "all queued runs across all workspaces" requires N separate streams.

This is rich for debugging but **does not constitute a discovery channel** — there is no `tasks.firehose` or `tasks.queue.<workspace>` topic an idle agent can subscribe to.

---

## 3. The autonomy gap

### 3.1 Who picks the agent today vs in an autonomous future

| Concern | Today | Autonomous target |
|---|---|---|
| Who decides which agent runs a given task? | The **caller** of `StartRun` decides (because `StartTaskSession` is given the run + actor context, and the executor spawns whatever session it likes with the workspace bound to the task — see `internal/daemon/task_runtime.go:261-267` for the wiring; the bridge is `*sessionBridge` from `internal/daemon/task_bridge.go`, not shown here). For automation jobs, the dispatcher does it inline and immediately calls `delegate` on the run (`internal/automation/dispatch.go:608-614`). | A **broker** matches a queued run to an idle, capability-matched agent. The agent claims via a typed RPC, not a CLI command. |
| What capability metadata exists on a task? | None. `Task` has `Owner` (`OwnerKind/OwnerRef` — used for inbox triage, not for routing), `NetworkChannel` (channel binding only — see grammar in `internal/network/`), `Priority`, `MaxAttempts`. **Required-capability is not a first-class field.** | Tasks need a `RequiredCapability` (or `CapabilitySpec` mirroring `session.NetworkPeerCapability` from `internal/session/interfaces.go:35-49`) so claim can be matched against agent skills. |
| What capability metadata exists on an agent session? | `session.NetworkPeerCapability` (`internal/session/interfaces.go:35-49`) — already structured (ID, summary, outcome, version, requirements, etc.) and projected to the network peer card via `internal/network/capability_brief.go:18-51`. **But the task package has zero awareness of this type.** | Reuse `NetworkPeerCapability` as the matching key on the task side; expose it via a narrow `CapabilityProvider` interface so `task.Service` can ask "which sessions advertise X?" without importing `session/`. |

### 3.2 How is "ready to claim" signaled today?

- **Push, scoped to one taskID**: `task.Service.Stream(ctx, taskID, …)` (`internal/task/live.go:95`) emits `task.run_enqueued` (and every other event) over SSE, but the consumer must already know the `taskID`.
- **Pull, untyped**: `ListTasks(query)` (`internal/task/manager.go:1171-1176`) supports filtering by `Status` and `OwnerKind/OwnerRef`. There is **no `RunQuery` filter for `status=queued AND no_session_bound`** at the manager layer — only the store-level `ListTaskRunsByStatus` (`internal/task/interfaces.go:86`) used at boot recovery.
- **No firehose**: there is no manager-level "subscribe to all queue events for workspace W" or "for capability C" primitive.

**Gap**: an idle agent has no way to discover queued work without walking every task in the workspace and re-fetching its runs.

### 3.3 Can agents discover tasks independently or are they pushed?

Today: **pushed via the prompt only**. The session prompt either contains the assignment (automation-rendered template, `internal/automation/dispatch.go:1012-1030`) or the user typed it. The session has no `task.*` tool family.

Evidence of the absence:
- `grep -rn "Capabilit\|capability" /Users/pedronauck/Dev/compozy/agh/internal/task/` returns one match — a comment in `internal/task/actors.go:7` that says "after ingress-level authentication and **capability checks**" — but those checks are about transport authority, not task-content matching.
- The CLI exposes `task list / get / create / update / cancel / child / dependency / run …` (`internal/cli/task.go:38-55`) but these are operator commands, not in-process tools registered against an ACP session.
- `internal/skills/bundled/` contains no task tool. The session has no equivalent of Claude Code's `TaskCreate / TaskUpdate / TaskList / TaskClaim` tools described in `docs/ideas/from-claude-code/analysis_multi_agent.md:225-278`.

### 3.4 Lease / heartbeat semantics

**There are none.** The schema (`internal/store/globaldb/global_db.go:335-371`) has these run columns:

```
id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id,
origin_kind, origin_ref, idempotency_key, network_channel,
queued_at, claimed_at, started_at, ended_at, error, metadata_json, result_json
```

Notably absent: `lease_until`, `heartbeat_at`, `claim_token`, `claim_expires_at`, `claimed_by_session_id`. The `claimed_by` columns (`claimed_by_kind`, `claimed_by_ref`) record the **actor identity** that claimed (`internal/store/globaldb/global_db.go:344-349`), not a *session* lease.

Consequences:
1. A dead claimer (crashed session, orphaned process) holds the run **indefinitely** until daemon restart triggers `RecoverRunOnBoot` (`internal/task/manager.go:1685`).
2. There is no "claim renewal" RPC — once claimed, the run is yours forever or until you explicitly complete/fail/cancel.
3. There is no "the broker took it back from you because you went silent" path. Two simultaneous agents can never *legitimately* race for the same run because `validateNoOpenRunForQueuedRunReservation` (`internal/store/globaldb/global_db_task_aux.go:1334`) prevents enqueueing a second run while one is open — but the same gate also prevents a takeover after a stall.

### 3.5 Sub-task creation by agents

`CreateChildTask(ctx, parentTaskID, spec, actor)` exists (`internal/task/manager.go:237-264`) and the `ActorKindAgentSession` actor kind is defined (`internal/task/types.go:108`). The wiring is half-built:

- `DeriveAgentSessionActorContext(sessionRef)` mints the actor (`internal/task/actors.go:34`).
- `DeriveAutomationLinkedAgentSessionActorContext` does it for automation-launched sessions (`internal/task/actors.go:42`); the dispatcher *records* this provenance via `RecordAutomationSessionTaskActor` (`internal/automation/dispatch.go:929-938`).

**But**: nothing exposes that actor context to the running ACP agent as a callable tool. There is no `task.create_child` tool, no `task.update_status`, no `task.add_dependency`. So even though the manager has every API needed, the agent has no in-process way to invoke them. The agent's only "create work for someone else" path is to write a markdown TODO into a shared file or, if automation is involved, return text that an outer policy interprets.

### 3.6 Race conditions in the current claim flow

`ClaimRun` locks via:
1. `loadRunWithTask` reads (`internal/task/manager.go:1410`).
2. `requireRunTransition(run, TaskRunStatusClaimed)` — purely state-comparison, no DB-level row lock (`internal/task/manager.go:1417`).
3. `store.UpdateTaskRun(ctx, run)` — last-write-wins UPDATE.

If two callers race the same `runID`, both can pass the in-memory transition check and the second one's UPDATE simply overwrites the claim metadata silently. SQLite `BEGIN IMMEDIATE` is used in `withTaskImmediateTransaction` (`internal/store/globaldb/global_db_task_aux.go:1059-1095`) but `UpdateTaskRun` itself (not shown) and the `ClaimRun` path do **not** open a transaction or perform a `WHERE status = 'queued'` conditional update. **This is exploitable in autonomous mode** where many idle agents would race the same queued run.

---

## 4. Reference comparisons

### 4.1 Hermes (`/Users/pedronauck/Dev/compozy/agh/.resources/hermes/tools/delegate_tool.py`)

- **In-process delegation**, not a queue: `delegate_task(goal | tasks=[…])` (line `1792`) builds N child `AIAgent` instances on the parent's main thread and runs them in a `ThreadPoolExecutor(max_workers=max_concurrent_children)` (line `1965`).
- **Capability matching = toolset filter**: `toolsets` arg per task picks which toolsets the child sees; blocked tools enumerated as a frozenset (`DELEGATE_BLOCKED_TOOLS`, line `38-46`). No queue, no claim — the parent fully owns the child's lifecycle and blocks until results return.
- **No lease**: parent waits via `as_completed` with interrupt polling (line `1986-2047`); a stuck child is killed when the parent is interrupted, not via a heartbeat.
- **Not a model for AGH's queue**: hermes is an orchestration-mode-only system. It has no broker pattern. **Useful only for the "run children synchronously" mode** that AGH's automation dispatcher already implements via `delegateRun`.

### 4.2 Multica (`/Users/pedronauck/Dev/compozy/agh/.resources/multica/packages/core/`)

- **TS/React state-management front-end** wrapping a server. The relevant package is `autopilots/` — exposes `listAutopilots`, `getAutopilot`, `listAutopilotRuns` (queries.ts:11-34). Mutations live in `autopilots/mutations.ts` (not read in detail). **No claim/lease primitive** — this is a UI for human-driven runs against a remote control plane.
- **Inbox** (`packages/core/inbox/`) is a notifications surface, not a work queue. Greps for `claim|lease|queue|heartbeat` returned zero hits inside the package.
- **Useful pattern**: the React-Query key conventions (`autopilotKeys.runs(wsId, id)`) map cleanly to AGH's TanStack Query setup if/when a "queued runs across workspace" view ships.

### 4.3 OpenClaw (`/Users/pedronauck/Dev/compozy/agh/.resources/openclaw/packages/`)

- Greps for `claim|lease|capability|queue|dispatch` only matched HTTP-fetch SSRF guard `release()` and `queueMicrotask` calls — **no agent-task-queue primitive at all**. OpenClaw is a chat product, not a task-broker reference.

### 4.4 Paperclip (`/Users/pedronauck/Dev/compozy/agh/.resources/paperclip/packages/plugins/sandbox-providers/e2b/`)

- **The most directly relevant lease pattern**: `onEnvironmentAcquireLease`, `onEnvironmentResumeLease`, `onEnvironmentReleaseLease` (plugin.test.ts:93-352) for sandbox VMs, with explicit `reuseLease: bool` on acquire and a `providerLeaseId` token returned to the caller.
- Acquire returns `{ providerLeaseId, metadata }`. Release decides between *pause* (reusable) and *kill* (ephemeral). On a failed pause, falls back to kill (line 336-352).
- **Mappable to task runs**: the same shape — `acquire(criteria) → { runId, claimToken, leaseUntil }`, `release(runId, outcome)`, `extend(runId, claimToken)` — would give AGH the ownership semantics it lacks. Paperclip's split between *reusable* and *ephemeral* leases is also worth borrowing for "interactive session vs one-shot worker".

### 4.5 Synthesis from internal docs

- `docs/ideas/from-claude-code/analysis_multi_agent.md:225-278` — Claude Code's task system (`TaskCreate, TaskUpdate, TaskList, TaskGet, TaskStop, TaskOutput`) with `owner`, `blocks`, `blockedBy` and the rule **"teammates claim tasks by setting `owner` via TaskUpdate"** (line 275). This is the **simplest possible claim primitive** — cooperative, advisory ownership via patch — and AGH already has 90% of it: `Task.Owner *Ownership` + `Patch.Owner` (`internal/task/types.go:243`, `:414`).
- `docs/ideas/network/agora-spec-v0.2.md:652-667` — `call` step kind, `role: "<capability>"` matched via `greet.skills`. Not a queue, but the key shape for capability-routed direct calls. The protocol assumes 1:1 dispatch; queue semantics live at the daemon/recipe layer, not on the wire.
- `docs/ideas/orchestration/multi-agent-patterns-analysis.md:368-376` (P0–P3 table) puts coordinator mode at P0 and explicitly excludes "claim queue with capability matching" — the analysis still treats agent-discovery as a Phase-3 network concern. **The current slice argues that this is a Phase-2 daemon concern that must land before the network protocol matters**.

---

## 5. Concrete proposals

### 5.1 Schema additions (`internal/store/globaldb/global_db.go:335-371`)

Add four columns to `task_runs` and a new table:

```sql
ALTER TABLE task_runs ADD COLUMN claim_token        TEXT;       -- opaque, server-issued
ALTER TABLE task_runs ADD COLUMN claimed_session_id TEXT REFERENCES sessions(id);
ALTER TABLE task_runs ADD COLUMN lease_until        TEXT;       -- RFC3339 UTC
ALTER TABLE task_runs ADD COLUMN heartbeat_at       TEXT;
CREATE INDEX idx_task_runs_lease ON task_runs(status, lease_until)
  WHERE status IN ('claimed','starting','running');

-- Required-capability metadata on the task row itself:
ALTER TABLE tasks ADD COLUMN required_capabilities_json TEXT;
-- JSON array of { id, version_min, requirements:[…] }
```

> Note: per `CLAUDE.md` "no migration code, delete the old thing" — these go straight into the boot-time DDL block (`internal/store/globaldb/global_db.go:335`) and existing dev DBs get rebuilt.

### 5.2 New manager APIs (`internal/task/interfaces.go`, `internal/task/manager.go`)

```go
// In package task — extend Manager:
type Manager interface {
    // … existing methods …

    // ClaimNextRun atomically pops one queued run that matches criteria.
    // Returns (nil, nil) if no work is available (not an error).
    ClaimNextRun(ctx context.Context, criteria ClaimCriteria, actor ActorContext) (*Run, error)

    // ExtendRunLease renews a held lease. Caller must present its claim token.
    ExtendRunLease(ctx context.Context, runID string, token string, actor ActorContext) (*Run, error)

    // HeartbeatRun records liveness without changing lease deadline.
    HeartbeatRun(ctx context.Context, runID string, token string, progress *RunProgress, actor ActorContext) error
}

type ClaimCriteria struct {
    WorkspaceID    string
    Capabilities   []string  // claimer's advertised capability IDs
    PriorityFloor  Priority  // skip lower-priority work
    NetworkChannel string    // optional channel binding
    LeaseDuration  time.Duration // requested; manager clamps
}

type RunProgress struct {
    Note     string `json:"note,omitempty"`
    PercentDone *float64 `json:"percent_done,omitempty"`
}
```

Storage surface (`internal/task/interfaces.go:81-98`) gains:

```go
type RunStore interface {
    // … existing …

    // ClaimNextRun is one atomic SQL operation (UPDATE … WHERE status='queued' …
    // RETURNING …) that picks the highest-priority unclaimed run matching
    // workspace + required-capabilities subset.
    ClaimNextRun(ctx context.Context, criteria ClaimCriteria, claimToken string, leaseUntil time.Time, actor ActorIdentity) (Run, Task, bool, error)

    ExtendRunLease(ctx context.Context, runID, token string, leaseUntil time.Time) (Run, error)
    HeartbeatRun(ctx context.Context, runID, token string, heartbeatAt time.Time) error
    ExpireLeasedRuns(ctx context.Context, now time.Time) ([]Run, error)
}
```

### 5.3 Capability advertising

Wire `session.NetworkPeerCapability` into the task package via a narrow consumer-side interface (Go-style), defined in `internal/task/interfaces.go`:

```go
type CapabilityProvider interface {
    // SessionsAdvertising returns session IDs that advertise *all* capabilityIDs.
    SessionsAdvertising(ctx context.Context, workspaceID string, capabilityIDs []string) ([]string, error)
}
```

`session.Manager` already owns the catalog (`internal/session/manager_helpers.go:92`) and implements the brief projection used by the network (`internal/network/capability_brief.go:18`). Implement the new interface there and inject it into `task.NewManager` via a new `WithCapabilityProvider(...)` option. The **task package never imports session** — it only sees the interface.

### 5.4 Discovery notifier (no event bus required, per CLAUDE.md)

Today `EventObserver.OnTaskEvent` already fires for every task event (`internal/task/manager.go:2811`). Add a typed sibling:

```go
// In internal/task/live_types.go:
type QueueWatcher interface {
    OnRunQueued(ctx context.Context, summary RunQueuedNotice)
}

type RunQueuedNotice struct {
    RunID                 string
    TaskID                string
    WorkspaceID           string
    Priority              Priority
    RequiredCapabilities  []string
    NetworkChannel        string
    QueuedAt              time.Time
}
```

Inject via `WithQueueWatcher(QueueWatcher)`. The daemon hooks an implementation that fans out to (a) UDS-connected idle agents, (b) the SSE multiplexer, (c) the network module so peers can see remote work. **One typed interface, no reflection, no NATS-style event bus** — matches the CLAUDE.md "Notifier pattern for fan-out" rule.

### 5.5 Agent-side tools (`internal/skills/bundled/` or a new `internal/agenttools/`)

Add an in-process tool family registered against ACP sessions whose `actor.Kind == ActorKindAgentSession`:

| Tool | Calls | Notes |
|---|---|---|
| `task.create_child` | `Manager.CreateChildTask` with `DeriveAgentSessionActorContext(session.ID)` | Uses the actor context already minted by `internal/task/actors.go:34` |
| `task.list_queue` | `Manager.ListQueueOffers(criteria)` (new thin wrapper around the new `ClaimNextRun` read view) | Read-only; returns top-N matching offers |
| `task.claim_next` | `Manager.ClaimNextRun` | Returns `{run_id, claim_token, lease_until}` |
| `task.heartbeat` | `Manager.HeartbeatRun` | Agent calls every N seconds during long work |
| `task.complete` | `Manager.CompleteRun` | With `claim_token` validation |
| `task.fail` | `Manager.FailRun` | Same |
| `task.update_progress` | `Manager.HeartbeatRun` with `RunProgress.Note` | Lets the task timeline carry agent-authored progress notes (not session events) |

**These are not skills (which are prompt-time content); they are first-class ACP tools the daemon registers per session, enforcing the actor context server-side so the agent cannot impersonate another session.**

### 5.6 Lease enforcement loop

Add a daemon background loop (`internal/daemon/task_runtime.go` next to `recoverTaskRunsOnBoot`) that runs every `defaultTaskLeaseSweep` (e.g. 10s):

```go
expired, _ := store.ExpireLeasedRuns(ctx, now)
for _, run := range expired {
    manager.RecoverRunOnBoot(ctx, run.ID, RunBootRecovery{
        Action: RunBootRecoveryRequeue,
        Reason: "lease_expired",
        Detail: fmt.Sprintf("no heartbeat since %s", run.HeartbeatAt),
    }, daemonActor)
}
```

This reuses the existing `RunBootRecovery` machinery (`internal/task/manager.go:1685-1745`). The recovered run goes back to `queued` and the queue notifier fires again — another idle agent can pick it up.

### 5.7 Example flow — fully autonomous

```
[Agent A, session sess-A, capability "code.review"]
└── creates task via tool task.create_child(parent=null, scope=workspace,
        title="Review PR #123", required_capabilities=["code.review"])
    └── manager.CreateTask(actor=DeriveAgentSessionActorContext("sess-A"))
        ├── reconcile → status=ready
        └── recordTaskEvent("task.created")

[Agent A] enqueues a run for the new task:
└── tool task.run_enqueue(task_id=t1)
    └── manager.EnqueueRun → store.ReserveQueuedRun → run r1 in 'queued'
        └── recordTaskEvent("task.run_enqueued")
            └── QueueWatcher.OnRunQueued({run:r1, task:t1, capabilities:["code.review"]})

[QueueWatcher fan-out]
├── push to all UDS-attached agent sessions in workspace W
└── push to network peers in any channel bound to t1

[Agent B, session sess-B, capability "code.review", idle]
└── receives RunQueuedNotice
└── calls tool task.claim_next(criteria={workspace:W, capabilities:["code.review"]})
    └── manager.ClaimNextRun
        └── store.ClaimNextRun (atomic SQL):
              UPDATE task_runs SET status='claimed', claimed_by_kind='agent_session',
                                   claimed_by_ref='sess-B',
                                   claimed_session_id='sess-B',
                                   claim_token=:tok, lease_until=:now+90s
              WHERE id IN (SELECT id FROM task_runs
                           WHERE status='queued' AND task_id IN (
                             SELECT id FROM tasks
                             WHERE workspace_id=:ws AND
                                   required_capabilities ⊆ :agent_caps
                           )
                           ORDER BY priority_rank, queued_at LIMIT 1)
              RETURNING *
        └── recordTaskEvent("task.run_claimed", {claimed_by_session:"sess-B"})

[Agent B] starts working in its own session:
├── periodic tool task.heartbeat(run=r1, token=tok, progress={note:"reading diff"})
│   └── manager.HeartbeatRun → store.HeartbeatRun → lease_until extended
└── tool task.complete(run=r1, token=tok, result={"approved":true,"comments":[…]})
    └── manager.CompleteRun → reconcile → task t1 status=completed
        └── recordTaskEvent("task.run_completed")
            └── QueueWatcher (or per-task SSE) notifies Agent A
```

If Agent B crashes mid-work:
```
[lease sweep loop, t+90s]
└── store.ExpireLeasedRuns → r1 (lease_until past)
└── manager.RecoverRunOnBoot(r1, action=Requeue, reason="lease_expired")
    └── status=queued, claim_token=null, claimed_*=null
    └── QueueWatcher.OnRunQueued fires again
[Agent C picks up r1]
```

---

## 6. Open questions

1. **Atomic claim semantics on SQLite.** `ClaimNextRun` needs `UPDATE … WHERE status='queued' AND id=(subquery ORDER BY … LIMIT 1) RETURNING *`. SQLite supports `RETURNING` since 3.35 (2021) — confirm the bundled driver version. Fallback: wrap in `BEGIN IMMEDIATE` (already used by `withTaskImmediateTransaction`, `internal/store/globaldb/global_db_task_aux.go:1059`).
2. **Capability matching expressivity.** Subset matching (`required ⊆ advertised`) is fine for a first cut. Do we need version constraints (`code.review >= v2`)? Outcome guarantees? See `session.NetworkPeerCapability` (`internal/session/interfaces.go:35-49`) — it already has `Version`, `Outcome`, `Requirements` — but exposing these to the SQL claim query requires either JSON1 expressions or denormalising them.
3. **Retry policy interaction with `MaxAttempts`.** `Task.MaxAttempts` defaults to 3 (`internal/store/globaldb/global_db.go:279`). When a leased run is reaped and re-queued, does `attempt` increment? Today `nextTaskRunAttemptWithExecutor` (referenced from `createQueuedRunWithExecutor`) increments per *new run*, not per re-claim. Decision needed: re-claim of the same run row should *not* increment attempt; only a brand-new `EnqueueRun` after `FailRun` should.
4. **Single-open-run-per-task constraint.** `validateNoOpenRunForQueuedRunReservation` (`internal/store/globaldb/global_db_task_aux.go:1334`) prevents parallel runs of one task. For autonomous worker pools (e.g. "code review" tasks farmed out to a pool), do we want N parallel runs per task? Current rule is *probably right* (one run per task = one attempt) but the broker may need a separate "task pool" abstraction (one task fans out to N child tasks each with one run).
5. **Escalation on persistently failing tasks.** Today `MaxAttempts` exists on the task but is not enforced anywhere in the run lifecycle (`grep MaxAttempts` shows it's only echoed back in summaries). Need an enforcement point: when a `FailRun` would push `attempt+1 > MaxAttempts`, transition the task to `failed` and emit an escalation event so a higher-tier agent (or human) can intervene.
6. **Network surface for claim.** `internal/network/tasks.go:42-58` only exposes `Get/Create/Update/Cancel/EnqueueRun` from peer. `ClaimNextRun` from a network peer raises trust questions — a remote daemon claiming local work means binding a remote session's capabilities to a local run. Defer to Phase 3 with an explicit `network_claim_authority` capability gate.
7. **Lease duration vs heartbeat interval.** Defaults: `LeaseDuration=90s`, `HeartbeatInterval=30s`, sweep loop `10s`. These need real numbers driven by ACP turn latency. Likely a config knob in `internal/config/`.
8. **Claim-token vs session-id binding.** Why a token *and* a session ID? Because a session may legitimately reconnect (e.g. after daemon restart) — the token is the proof-of-claim that survives session renumbering. Alternative: tie the lease to `claimed_session_id` only, but then session ID rotation breaks the lease. Token is safer.
9. **Race: enqueue + claim simultaneity.** If a publisher enqueues r1 and a claimer races with `ClaimNextRun` *before* the queue notifier fires, both are correct (the SQL UPDATE either finds a queued row or doesn't). The notifier is therefore a hint, not a contract — which is the right shape for a notifier.
10. **Is the existing `Owner` field redundant?** `Patch.Owner` (`internal/task/types.go:414`) lets the operator set `OwnerKind=agent_session, OwnerRef=sess-B` today. This is the Claude Code "set owner via TaskUpdate" claim primitive. The proposed `claim_token + lease_until` adds *enforcement* on top; `Owner` becomes the **assignment hint** (operator says "I want sess-B to handle this") and `claim_token` becomes the **operational lock** (sess-B actually has it, with a deadline). They are distinct and both useful.

---

## Appendix A — File:line index

- `internal/task/types.go:21-100` — Status + RunStatus enums.
- `internal/task/types.go:228-298` — `Task`, `Run`, `Event`, `RunIdempotency` structs.
- `internal/task/types.go:387-453` — Mutation request types (`CreateTask`, `Patch`, `EnqueueRun`, `ClaimRun`, `StartRun`, `CancelRun`).
- `internal/task/interfaces.go:10-39` — `Manager` interface.
- `internal/task/interfaces.go:81-98` — `RunStore` (note: `ReserveQueuedRun`, no `ClaimNext`).
- `internal/task/interfaces.go:135-141` — `SessionExecutor` (start/attach/stop only — no claim hand-off).
- `internal/task/manager.go:174-233` — `CreateTask` happy path.
- `internal/task/manager.go:237-264` — `CreateChildTask`.
- `internal/task/manager.go:1330-1378` — `EnqueueRun`.
- `internal/task/manager.go:1392-1441` — `ClaimRun` (one specific runID, no race protection).
- `internal/task/manager.go:1444-1505` — `StartRun`.
- `internal/task/manager.go:1685-1745` — `RecoverRunOnBoot` (the only existing "take it back" path).
- `internal/task/manager.go:2775-2814` — `recordTaskEvent` + observer fan-out.
- `internal/task/live.go:95-127` — Per-task SSE `Stream`.
- `internal/task/live.go:649-681` — `emitTaskLiveRecordBestEffort` fan-out.
- `internal/task/actors.go:8-15` — `FullAccessAuthority` (every authenticated principal gets full task auth).
- `internal/task/actors.go:34-47` — Agent-session actor minting.
- `internal/store/globaldb/global_db.go:267-326` — `tasks` table DDL.
- `internal/store/globaldb/global_db.go:335-376` — `task_runs` table DDL (no lease columns).
- `internal/store/globaldb/global_db_task_aux.go:611-655` — `ReserveQueuedRun` entry.
- `internal/store/globaldb/global_db_task_aux.go:702-742` — Reservation transaction (calls `validateNoOpenRunForQueuedRunReservation`).
- `internal/store/globaldb/global_db_task_aux.go:1334-1356` — Single-open-run gate.
- `internal/network/tasks.go:42-58` — Network-peer task surface (notice: no `ClaimRunFromPeer`).
- `internal/network/capability_brief.go:13-51` — Capability projection for peer cards.
- `internal/automation/dispatch.go:556-657` — Task-backed automation dispatch (the only existing "automated end-to-end" path; immediately delegates).
- `internal/cli/task.go:481-761` — All CLI subcommands for run lifecycle (`enqueue`, `claim`, `start`, `attach-session`, `complete`, `fail`, `cancel`).
- `internal/daemon/task_runtime.go:240-305` — Manager wiring.
- `internal/daemon/task_runtime.go:327-350` — Boot recovery.
- `internal/session/interfaces.go:35-49` — `NetworkPeerCapability` (the type to reuse for capability matching).
