# Senior Architecture Review — Autonomous AGH TechSpec

Reviewer: Opus (architect-advisor lens)
Date: 2026-04-25
Inputs: `_techspec.md`, `adrs/adr-001..009.md`, `analysis/analysis.md`, plus targeted reads of `internal/hooks/events.go`, `internal/hooks/payloads.go`, `internal/session/{manager.go,interfaces.go}`, `internal/store/globaldb/global_db_task_aux.go`, `internal/task/manager.go`, `internal/daemon/{hooks_bridge.go,prompt_sections.go}`.

---

## 1. Executive Verdict

**Approve with changes.**

The substrate analysis is correct, the four-layer split (Situation / Agent CLI / Autonomy Kernel / Memory) is the right shape, and ADR-003/004/006/009 each lock the only sane choice. The phased build order is genuinely incremental and respects AGH's existing extension model. There is no need for a redesign.

What needs to land before `cy-create-tasks` runs:

1. A clear **scheduler vs agent-pull boundary** (currently both `agh task next --wait` and `internal/scheduler` claim runs — the spec must say which is authoritative and which is an optimization).
2. An explicit **coordinator-spawn trigger** (the spec says "spawn on first work that requires semantic orchestration" but never defines the signal).
3. An explicit **lease/sweep/heartbeat invariant set** (race contract for stale tokens, recovered runs, and concurrent complete-after-recovery).
4. A **permission-narrowing model** for spawned sessions (ADR-006 mandates narrowing; the spec never describes the comparator).
5. A **TTL × active-lease interaction policy** (reaper vs. lease ownership).
6. **Naming alignment** of new hook families with the existing taxonomy (`autonomy.*` is novel; the existing axis is `task.*` / `agent.*` / `session.*` / `automation.*`).
7. A **hard MVP cut line** between steps 1–10 (kernel, ship-blocking) and steps 11–15 (extensions, post-MVP).

Each of those is a 1–2 paragraph addition or one ADR refinement, not a redesign.

---

## 2. Critical Issues (block decomposition until resolved)

### 2.1 Scheduler vs. agent-pull authority is undefined

The spec ships **two** independent claim paths:

- `POST /agent/tasks/claim-next` (`_techspec.md:204`, `agh task next --wait` in CLI), called by agents directly.
- `internal/scheduler` (`_techspec.md:298`), which "owns deterministic work placement: idle-session indexing, capability matching, task-run claim, lease renewal checks, backpressure, and recovery."

Both end up calling `ClaimNextRun(criteria)`. So who actually claims?

- If the scheduler is authoritative, then `agh task next --wait` is redundant — agents only react to push (a synthetic prompt + envelope from the scheduler).
- If agents pull, then the scheduler's "matching" and "placement" are advisory at best — its only deterministic responsibilities are **lease sweep, idle index, and notification**.

These are very different designs. The current text implies the second (because the agent CLI verb exists and the data flow at `_techspec.md:53–60` describes the scheduler "calls `ClaimNextRun`"), but then the scheduler also does matching, which is meaningless if agents independently pick what to claim.

**Required:** declare the chosen model. The cleanest version for an MVP is:

> Scheduler is **not** an authoritative placer. It is a sweep/notify daemon goroutine that (a) maintains the idle-agent index, (b) sweeps expired leases on a tick, (c) wakes idle agents (via existing notification surface) when matching ready work appears. **All claims happen via `ClaimNextRun(criteria)` initiated by the agent's pull verb.** The scheduler never calls `ClaimNextRun` itself.

That collapses the apparent "scheduler vs coordinator" tension and removes the duplicate-decision risk noted in ADR-004's risks section.

If the chosen model is the opposite (scheduler claims and pushes), then `agh task next --wait` is removed from MVP and replaced with a "wait-for-assignment" stream verb.

Either is fine; both is not.

### 2.2 Coordinator-spawn trigger is unspecified

ADR-005 + `_techspec.md:33` say the daemon spawns one coordinator per workspace "on first work that requires semantic orchestration." But nothing in the spec defines that signal:

- Is it a task field (`requires_orchestration: true`)?
- Is it inferred from a task type / role?
- Is it a coordinator spawning itself in response to a network message?
- Is it triggered by `cy-` style top-level prompts only?

Without an explicit trigger, the scheduler can't decide when to wake a coordinator, and the coordinator's very existence becomes implicit. This is the highest-leverage missing definition in the entire spec because it determines:

- which task DDL field to add (`required_role`, `orchestration_required`, etc.);
- which hook payload carries the trigger (`autonomy.coordinator.pre_spawn` needs to know *why* it's being asked to spawn);
- whether the coordinator can spawn another coordinator (recursion gate).

**Required:** add a "Coordinator Trigger" subsection to the System Architecture covering signal source, idempotency (one coordinator per workspace at a time), and re-entry rules.

### 2.3 Lease semantics are missing the race contract

`_techspec.md:316–323` lists the right unit-test surface but the spec never names the invariants the tests should defend. Concretely missing:

- **Stale-heartbeat rule**: agent A's lease expires, sweep recovers it, agent B claims it; A then sends `heartbeat(token=A)`. Does A get `ErrLeaseRecovered`? Or does it succeed because A doesn't know yet? AGH defaults must be: explicit error, no silent extension.
- **Late-complete rule**: same scenario, A sends `complete(token=A, result=...)`. The completion must fail; A's result must not overwrite B's progress. The spec mentions claim-token checks but does not specify what happens when the token *is* valid but the claimed_session_id has been reassigned.
- **Sweep concurrency**: if the sweep sweeps a run at the same time the holder heartbeats, who wins? `BEGIN IMMEDIATE` makes this serialized at the SQL layer; spec should state that the sweep uses a CAS predicate (`WHERE claim_token = ? AND lease_until = ?`) and fails closed.
- **Heartbeat budget**: max lease extension before forced re-claim? Otherwise a wedged agent can heartbeat forever.
- **Boot recovery**: `_techspec.md:330` mentions "lease expiry on daemon restart -> recovery makes the run claimable without duplicate completion." That requires a deterministic sweep at boot that runs **before** the scheduler accepts new claims. Spec should state ordering.

These are race-class invariants and need to be in the design before any task file is written, because the schema and API design depend on them.

### 2.4 Permission narrowing has no comparator

ADR-006 + `_techspec.md:166` say:

- `permission_policy_json TEXT`
- "Children can never widen parent permissions, tools, or skills."

But the comparison model is not specified anywhere:

- Is the permission set a flat list of strings? A typed enum? A set of fs paths? A YAML doc?
- Is the comparator subset over a known atom space, or a structural diff?
- What about *new* permission categories the parent doesn't know about (additive forward-compat)?
- Where does the comparator live — `internal/session`? new `internal/permissions`?
- How does it interact with existing `permission.request` hooks?

Without a comparator, "narrowing" is unverifiable and ADR-006's core safety property is asserted but not enforced. This is a critical issue because the MVP coordinator is *expected* to spawn workers with narrowed tool/skill sets — that's the whole point of `ToolAllowlist`.

**Required:** name the comparator (a function signature is enough), state the atom space (probably "tools, skills, workspace-paths, network-channels"), and lock the failure mode (reject spawn vs. silently narrow vs. log-and-allow).

### 2.5 TTL × active-lease interaction is undefined

ADR-006 and `_techspec.md:478` mention TTL/parent-stop reapers, but the interaction with task leases held by spawned sessions is left as "according to policy" without naming the policy.

The two clean choices:

- **Lease wins**: TTL/parent-stop is delayed until active leases are released or expired. Risk: parent stop hangs.
- **Reaper wins**: reaper calls `ReleaseRun` for every claimed run on the dying session before terminating it; the run is requeued. Risk: silent loss of in-flight progress.

I recommend **reaper wins, but with structured `ReleaseReason` ("parent_stopped" / "ttl_expired") on every released run**, so the operator and the scheduler can decide whether to requeue or fail. State this explicitly so task decomposition can implement the right hook payload.

---

## 3. Major Issues (important design gaps)

### 3.1 Hook taxonomy diverges from existing pattern

`internal/hooks/events.go:8–21` defines families: `session`, `environment`, `input`, `prompt`, `event`, `automation`, `agent`, `turn`, `message`, `tool`, `permission`, `context`. New families in the spec:

- `autonomy.coordinator.*` — should be `coordinator.*` (extend taxonomy with one new family).
- `autonomy.scheduler.*` — should be `scheduler.*` or, better, **not added**: replace with a `task.run.*` extension (since these events are about run claim/lease, which are already a task-domain concern).
- `task.run.*` — good, this is the right shape (extends existing `task.*` semantics in event names like `task.created`/`task.run_enqueued` already used in `internal/task/manager.go:28–37`).
- `workflow.*` — premature; workflows aren't a first-class concept in the codebase yet (no `internal/workflow`). Don't add a hook family for an entity that doesn't exist; carry workflow correlation as a metadata field on existing payloads.

Recommendation: add families `coordinator`, `scheduler`, `spawn` (NOT `autonomy.*`); fold every "scheduler.*" event under either `task.run.*` or `scheduler.*` (decision falls out of issue 2.1); drop `workflow.*` from MVP.

### 3.2 Spawn lifecycle hooks should be explicit, not "extending session hook payloads"

`_techspec.md:260–262` says:

> Prefer extending existing session/agent hook payloads with lineage fields. Add spawn-specific events only if existing session hooks cannot express parent/child semantics cleanly.

They cannot. A spawn is semantically distinct from a session-create because:

- It carries `ParentSessionID`, role overlay, narrowed permissions, TTL, budget — none in `SessionContext` (`internal/hooks/payloads.go:21–32`).
- Operators want to deny spawns specifically (e.g., "no nested spawn from coordinator workers"), which `session.pre_create` can't express without contorting the payload.
- Auditing spawns is a separate operational concern than auditing session creation (different access paths, different expected volumes).

Add `spawn.pre_create`, `spawn.created`, `spawn.parent_stopped`, `spawn.ttl_expired`, `spawn.reaped` as a new family. Existing `session.*` hooks fire as well, but `spawn.*` carries lineage payloads natively.

### 3.3 Coordinator's CLI surface is incomplete

The agent CLI surface (`_techspec.md:213–226`) lists `me`, `ch`, `task next/heartbeat/done/fail/release`, `spawn`. The coordinator needs **task creation** verbs (`agh task create`, `agh task add-dep`, `agh task assign-criteria`) that the spec never enumerates as agent-callable. Without those, the coordinator's "decomposition" capability is a no-op — it can't materialize child tasks for workers to claim.

Either:

- Add `agh task create` (and dependency verbs) to the agent surface, with appropriate identity/permission gating, OR
- State that the coordinator uses operator CLI / direct task service for creation (and accept that this widens its tool surface beyond the "restricted" claim in ADR-005).

### 3.4 Capability matching has no index strategy

`_techspec.md:150` says capabilities live as `required_capabilities_json TEXT` on tasks/runs and that exact-match suffices. But:

- The scheduler / `claim-next` query needs to filter `WHERE status='queued' AND <capability subset>` efficiently.
- SQLite JSON1 functions support this but require either generated columns or careful query shape; full-table-scan on every claim attempt is not acceptable past O(100) ready runs.
- If the design is "one capability per run for MVP," say so. If it's a set, name the index strategy (probably a side-table `task_run_capabilities(run_id, capability)` with `(capability, status)` index, populated transactionally with the run).

This isn't a critical issue because exact-match works; it is a major issue because the wrong choice here will get baked into a schema that's painful to change once data exists, even in alpha.

### 3.5 Idle-agent registry boundary is fuzzy

`ClaimPolicy.Next(ctx, idle []IdleAgent, queued []QueuedRun)` (spec interface) implies the scheduler holds an in-memory `idle []IdleAgent` view. But:

- Where is this populated? `session.Manager` lifecycle hooks? `OnTurnEnd`? `OnAgentSpawned`?
- Is it eventually consistent or transactional?
- What's the recovery path if the daemon crashes between an agent going idle and the registry being updated?

Pick one model. The clean version is: registry is in-memory only, populated by session lifecycle observation (using the existing `Notifier`/hook bridge), and is **always rebuildable from `sessions` + `task_runs` state at boot**. The spec says "rebuildable" once at `_techspec.md:67` but does not enumerate the rebuild source.

### 3.6 `WorkflowLifecyclePayload` is undefined

The hook taxonomy lists workflow events, the metrics list `workflow.completed/failed`, the structured log fields list `workflow_id` — but the workflow concept is not introduced anywhere in the architecture section. Where is a workflow created? Where is its state stored? What is its relationship to a `task` (a task is part of a workflow? a workflow is a graph of tasks?)?

This needs either a one-paragraph definition or removal from MVP. ADR-004 implies workflows are emergent from coordinator behavior (a coordinator orchestrates a "workflow" by creating child tasks). If that's the case, `workflow_id` is a metadata field (a coordinator-issued correlation ID stamped onto child tasks), not a first-class entity, and there should be no workflow hook family.

### 3.7 Build order step 8 vs step 7 ordering

Step 7 ("agent task verbs") depends on step 6 (claim/lease store API). Step 8 ("mechanical scheduler") depends on 6 and 7. But once steps 6+7 ship, agents can already self-claim work — autonomy's pull half is functional. The scheduler is then optional for the MVP.

Recommendation: explicitly call this out as a **demo milestone** ("after steps 1–7, agents can self-claim ready tasks; this is the first end-to-end autonomy validation point") and treat step 8 as adding the *push/notify* half. This re-frames what "MVP" means and reduces decomposition risk.

### 3.8 Identity inference rules for agent CLI

The spec relies on `AGH_SESSION_ID` and `AGH_AGENT` env vars (`_techspec.md:21`) but never specifies:

- What if an agent restarts (resume) and the env vars carry the old session ID?
- What if an agent's tool subprocess spawns its own subprocess that inherits env? Does the grandchild claim work as the original agent?
- What if env is missing? Hard-fail with exit code? Fall back to operator behavior?

This matters because every agent verb depends on identity. State the invariants in `internal/cli` design notes (probably: env is authoritative, missing env means non-agent caller, resume rotates `AGH_SESSION_ID` so stale env returns explicit error).

### 3.9 `/agent/telemetry/session` is over-broad

The endpoint returns "recent activity, loop state, budget state, and hook failures" (`_techspec.md:210`) — four concerns in one read. This will grow into a dumping ground. Split into:

- `/agent/telemetry/activity`
- `/agent/telemetry/loop`
- `/agent/telemetry/budget`
- `/agent/telemetry/hooks`

Or, per the analysis's `agh.session.stats` precedent, define a stable JSON envelope where each subsection is independently versioned.

### 3.10 No backpressure model

"Backpressure" appears in `_techspec.md:299` but is never defined. What's the unit of pressure?

- Per-session in-flight prompt count?
- Per-workspace concurrent claimed runs?
- Per-coordinator child count?
- Daemon-global pending claim queue depth?

Pick at least one for MVP and state the threshold source (config, hardcoded, both). The natural primitive is **per-session lease cap** ("a session can hold at most N active leases"), which falls out of `claimed_session_id` indexing; the cap defaults to 1 for the MVP.

---

## 4. Minor Issues

### 4.1 `claim_attempts` / `last_claim_error` lifecycle

These columns appear in the schema (`_techspec.md:139–140`) but no API resets or reads them. State whether they're advisory metrics or used by recovery (e.g., max-attempts-before-fail).

### 4.2 `AutoStopOnParent bool` vs ADR-006 wording

`SpawnOpts.AutoStopOnParent` is a single bool; ADR-006 says "unless explicitly configured otherwise within hard caps" without enumerating the caps. Either remove the override (children always auto-stop) or add the cap fields (`MaxOrphanGracePeriod`, etc.).

### 4.3 "Scheduler policy" interface for one policy

`ClaimPolicy` interface (`_techspec.md:94–96`) is added before there are multiple policies. The repo has the explicit rule: "Maps for <10 items — no registry interfaces for small collections." Until there's a second policy, hardcode the priority+age rule and drop the interface.

### 4.4 `agh me logout` is in the analysis but not the spec

Verify intentional. If yes, fine; if dropped, document why (probably correct: agents don't log out of themselves).

### 4.5 Resource kinds list

ADR-009 mentions "candidate resource kinds: coordinator policy/config and workflow/eval definitions." Spec should explicitly state **zero new resource kinds in MVP** (since coordinator config goes through `[autonomy.coordinator]` and workflows aren't a thing yet). This prevents accidental scope expansion during decomposition.

### 4.6 "Greenfield alpha" vs "no migration code" — schema bumps

The spec is right to add columns rather than rewrite tables (fits SQLite well), but state explicitly: **schema changes ship as one DDL transaction at boot; failure aborts daemon start.** No half-migrated state.

### 4.7 Naming: `ReleaseRun` vs `release` vs `pass`

Analysis used `agh task pass`. Spec uses `agh task release`. ADR doesn't mention either. Pick one (release is clearer; pass implies "to another agent") and use it consistently.

### 4.8 `agh me context` payload schema isn't defined

The CLI verb is in `_techspec.md:215` and the endpoint at `:198`, but the JSON shape (which sections, in what order, with what truncation rules) isn't specified anywhere. Reference paperclip's `paperclipGetHeartbeatContext` from the analysis is a good precedent — borrow the field set explicitly.

### 4.9 `interactionCurrent`/`interactionMax` (analysis section) and budget primitives

The analysis identifies `IterationCurrent`/`IterationMax` as a dead column. Spec mentions iteration loops (step 13) but doesn't claim ownership of this column. State: MVP wires `IterationCurrent` increment in the existing turn lifecycle hook; budget circuit breaker reads it.

### 4.10 Eval/replay harness step 14 vs cross-slice question

Cross-slice question 10 in the analysis (`analysis.md:326`) said "VCR for CI, rubric/promptfoo for nightly — both, not one." Spec step 14 is silent on which. State the choice or defer the decision to a separate ADR-010.

---

## 5. Over-Engineering Candidates (defer or narrow for MVP)

| Item | Source | Why defer |
|---|---|---|
| `ClaimPolicy` interface | `_techspec.md:94` | One policy in MVP; ship a hardcoded function. |
| Workspace-level coordinator overrides | ADR-005 | Global config is enough for first release. |
| `WorkflowLifecyclePayload` and `workflow.*` hooks | `_techspec.md:188, 256` | Workflow is not yet an entity. |
| Eval/replay harness (step 14) | `_techspec.md:358` | Defer — needs its own ADR; not on the autonomy critical path. |
| Web visibility (step 15) | `_techspec.md:359` | Backend contracts must stabilize first; explicitly post-MVP. |
| Network protocol changes (step 11) | `_techspec.md:355` | "Minimal" is still a wire bump. Land MVP local autonomy without it; do the network bump as a separate TechSpec once usage shows what's actually needed. |
| Memory provenance + summaries (step 12) | `_techspec.md:356` | Provenance-only (write `agent_name` correctly) is MVP-grade. Session-end summaries are a separate task that can ship after kernel. |
| Self-correction loop detection (step 13) | `_techspec.md:357` | Iteration counter + budget ceiling is MVP. Repetition detector + recovery prompt injection is post-MVP. |

The cleanest MVP cut is **steps 1–10**, with step 10 (coordinator bootstrap) as the autonomy demo milestone. Steps 11–15 are a follow-on TechSpec.

---

## 6. Missing Extensibility Hooks/Resources/Providers

The spec is generally faithful to ADR-009, but several extension points referenced in the analysis are missing from the techspec proper:

### 6.1 `CapabilityProvider` interface

Mentioned in `analysis.md:303` as the bridge between `session.Manager` and the scheduler's idle/match logic. Not in the techspec's "Core Interfaces" section. Add it: scheduler asks "what capabilities does session X advertise?" through this interface; default impl reads from `AgentDef.Capabilities`.

### 6.2 `IdleAgentRegistry` provider

`ClaimPolicy.Next` consumes `[]IdleAgent` but the source isn't specified. This should be an interface owned by the scheduler and implemented in `internal/session` (so session lifecycle stays the source of truth).

### 6.3 Spawn-decision hook

The spec has `autonomy.coordinator.pre_spawn` (about coordinator startup) but no `spawn.pre_create` for arbitrary agent-initiated spawn. ADR-006's safety invariants are operator concerns; without a hook, operators can't observe or veto a spawn before it happens. This is the same shape as `permission.request` for tool calls.

### 6.4 Coordinator "decision" hook

When the coordinator-agent decomposes a goal into child tasks, that's a semantic decision that operators want to audit. A `coordinator.decision` hook firing on every coordinator-issued task creation would close that gap. Right now the only signal is the `task.created` hook (existing), which doesn't carry the coordinator's reasoning context.

### 6.5 Lease-recovery hook

`task.run.lease_expired` is in the spec. Add `task.run.lease_recovered` (post-sweep) as a distinct event so observers can correlate expiry → recovery → reclaim cleanly. Without this, reconstructing the recovery story from logs requires JOINs across three event types.

### 6.6 Configuration provider for `[autonomy.coordinator]`

The config block is mentioned but no provider/resolver pattern is specified. The existing `aghconfig` model is presumably extended; state which file owns the parsing and add a `CoordinatorConfigResolver` interface so workspace overrides (when added) can plug in cleanly.

---

## 7. Task-Decomposition Guidance for `cy-create-tasks`

The phased build order at `_techspec.md:343–360` is the right granularity for **phases**, but each phase is too big for a single task file. Recommended decomposition (one task per bullet):

### Phase A — Foundations (steps 1–2)

1. **A1 — Autonomy config struct + DTOs**: `CoordinatorConfig`, `SpawnOpts`, `ClaimCriteria`, `ClaimedRun` types in `internal/api/contract` and `internal/config`. Pure types, no behavior. Includes config-resolution unit tests (workspace > global > bundled).
2. **A2 — Hook taxonomy additions**: new families (`coordinator`, `scheduler`, `spawn`, `task.run`), payload structs, dispatch methods, introspection descriptors. Pure additions to `internal/hooks/*`. Tests: family validation, descriptor completeness.

### Phase B — Situation Surface (step 3)

3. **B1 — `SelfContextProvider`**: render `<situation>` block (peer-id, session, workspace, agent, channel, model). Wired through `session.PromptProvider`.
4. **B2 — `PeerRosterProvider`**: render current peer list. Reorders `joinNetworkPeer` to run before prompt freeze (or accepts a synthetic-self snapshot — pick one).
5. **B3 — `TaskEnvelopeProvider`**: render task/run context for task-launched sessions. Threads `TaskID`/`RunID` through `StartupPromptContext`.
6. **B4 — `SituationReminderAugmenter`**: per-turn delta updates via `PromptInputAugmenter`. Bounded; renders only what changed.

### Phase C — Agent CLI identity (steps 4–5)

7. **C1 — UDS identity middleware + env resolution**: every command tags requests with caller session; missing env returns deterministic exit code.
8. **C2 — `agh me` namespace**: `me`, `me context`, with stable JSON schema.
9. **C3 — `agh ch` namespace**: `list`, `recv --wait`, `reply --to-message`, `join`, `whois`. Replaces `delivery.go:820–988` reply guidance.

### Phase D — Task claim + lease (steps 6–7)

10. **D1 — Claim/lease schema**: DDL additions to `task_runs`, indexes, boot-time DDL transaction. Includes recovery sweep on boot, ordering guarantees stated explicitly.
11. **D2 — `ClaimNextRun` + lease API**: transactional implementation in `internal/store/globaldb` and `internal/task`. **High stakes — race-test heavy.** Token validation on heartbeat/complete/fail/release with explicit invariant doc.
12. **D3 — `agh task` agent verbs**: `next --wait`, `heartbeat`, `done`, `fail`, `release`. End-to-end claim → complete integration test.

### Phase E — Scheduler (step 8)

13. **E1 — Idle agent registry**: in-memory, populated from session lifecycle hooks, rebuildable from store at boot.
14. **E2 — Lease sweep loop**: ticked goroutine with explicit ownership/shutdown via `sync.WaitGroup` + `ctx.Done()`. Emits `task.run.lease_expired` and `task.run.lease_recovered` hooks.
15. **E3 — Idle notifier**: when ready work appears matching an idle agent's capabilities, emits a synthetic prompt or wakes its `agh task next --wait` poll.

### Phase F — Safe spawn (step 9)

16. **F1 — Lineage fields on session metadata**: `parent_session_id`, `root_session_id`, `spawn_depth`, etc.
17. **F2 — Permission narrowing comparator**: explicit atom space, comparison fn, validation in `SpawnOpts`. Tests: widening rejected, equal allowed, narrower allowed.
18. **F3 — Reaper integration**: parent-stop and TTL paths release held leases with structured `ReleaseReason`.
19. **F4 — `agh spawn` CLI + UDS endpoint**: end-to-end spawn → child runs → parent stops → child auto-stops.

### Phase G — Coordinator (step 10)

20. **G1 — Coordinator agent definition (bundled)**: restricted tool surface, prompt overlay, marked session type/metadata.
21. **G2 — Coordinator-spawn trigger**: deterministic signal (per critical issue 2.2). `autonomy.coordinator.pre_spawn` fires with the trigger payload.
22. **G3 — Coordinator end-to-end demo**: queued task → no idle agent → coordinator spawn → coordinator decomposes → workers claim → completion. This is the autonomy MVP demo.

### Out of MVP (defer to separate TechSpec)

Steps 11–15 (network evolution, memory expansion, self-correction, eval, web) should be split into their own TechSpecs after Phase G ships. They're each large enough to deserve dedicated ADRs and decomposition.

### General decomposition rules

- Each task gets its own integration test if it touches the daemon composition root or crosses package boundaries.
- D2 (claim/lease) and F2 (permission narrowing) deserve **dedicated review rounds** with `cy-review-round` because they are the safety primitives the rest of the system relies on.
- A2 (hook taxonomy) must land before any behavior task; a missing event family creates rework downstream.
- B-phase tasks can run in parallel (independent providers), but B4 depends on B1–B3 being stable.

---

## 8. Final Recommendation

**Approve with the seven changes in §1.** This is a sound, incremental design that respects AGH's existing architecture and avoids the temptation to introduce a separate plugin system or event bus. ADR-003 (extend `task_runs`), ADR-004 (split scheduler from coordinator), ADR-006 (spawn safety), and ADR-009 (typed hooks) are each the right call.

The biggest risk is **building two claim paths** (scheduler + agent pull) without picking which one is authoritative — issue 2.1. Resolving that single ambiguity makes most of the scheduler design fall into place. The next biggest is **the coordinator-spawn trigger** (2.2) — without it, the autonomy demo has no defined entry point.

Concrete next steps before `cy-create-tasks`:

1. Add a one-page "Scheduler vs Pull" decision section to `_techspec.md` resolving issue 2.1.
2. Add a "Coordinator Trigger" subsection resolving issue 2.2.
3. Add an "Invariants" subsection enumerating the lease/heartbeat/sweep race contract from issue 2.3.
4. Add a "Permission Narrowing" subsection naming the comparator from issue 2.4.
5. Add a "TTL × Lease" subsection naming the policy from issue 2.5.
6. Rename hook families per issue 3.1 and add `spawn.*` per issue 3.2.
7. Mark steps 1–10 as MVP and 11–15 as post-MVP follow-on at the top of "Development Sequencing."

These are additive and do not change the fundamental shape. Once they land, decomposition into the 22-task plan in §7 should be mechanical.
