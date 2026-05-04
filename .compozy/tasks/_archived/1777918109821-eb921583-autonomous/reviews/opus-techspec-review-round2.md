# Opus Review Round 2: Autonomous AGH TechSpec

Reviewer: Opus (architect-advisor lens)
Date: 2026-04-25
Inputs: `_techspec.md`, `adrs/adr-001..010.md`, `analysis/analysis.md`, `reviews/opus-techspec-review.md`, `reviews/gpt54mini-{multica,paperclip,agh-code}-analysis.md`. Code spot-checks: `internal/task/manager.go`, `internal/task/types.go`, `internal/hooks/events.go`, `internal/store/globaldb/global_db.go`, `internal/session/manager.go`, `internal/cli/task.go`, `internal/api/contract/contract.go`, `internal/config/`.

## Verdict

**Approve with changes.**

The substrate analysis is sound, the four-layer model is correct, and the round-1 critical issues (scheduler vs pull authority, coordinator trigger, lease invariants, permission narrowing, TTL × lease, hook taxonomy, MVP cut line) are all answered in the current TechSpec or the new ADR-010. The Multica and Paperclip cross-checks confirm the model matches industry-precedent shapes.

What still needs to be tightened before `cy-create-tasks` runs is mostly **reconciliation with the existing AGH code**. Three issues are blocking because they would force `cy-create-tasks` to either invent a bridge, choose a schema strategy under-specified by the TechSpec, or generate tasks that conflict with already-shipped types. Everything else is non-blocking polish.

## Blocking Issues

### B1. Task-domain audit events vs autonomy hooks bridge is unspecified

- **Severity**: high
- **Files/sections**: `_techspec.md:57, 232–239, 304–339`, `adrs/adr-009.md:21–28, 86–92`
- **Why it matters**: The TechSpec states "Run enqueue emits the existing `task.run_enqueued` domain event, dispatches configured hooks". In the current code, `task.run_enqueued` is an audit-log row written by `Service.recordTaskEvent` (`internal/task/manager.go:2775`) into the task event store, not a hook dispatched through `internal/hooks`. Hook families in `internal/hooks/events.go:8–21` are: `session, environment, input, prompt, event, automation, agent, turn, message, tool, permission, context`. There is no `task.*` hook family and no bridge between task audit events and the hook runtime. The TechSpec's autonomy plan adds `task.run.pre_claim`, `task.run.post_claim`, `task.run.lease_*`, `coordinator.*`, `scheduler.*`, `spawn.*` as hooks but never says how a daemon component receives the `task.run_enqueued` audit event and turns it into a hook dispatch (or whether the task service itself calls into hooks at the same call sites). Without that resolution, `cy-create-tasks` cannot decompose step 2 (hook taxonomy) or step 6 (claim/lease) coherently — both depend on which side owns dispatch. It also affects the coordinator trigger (B2) because the trigger flows from the same event.
- **Concrete fix**: Add a "Domain Events vs Hooks Bridge" subsection to the TechSpec naming the chosen mechanic. Two clean options:
  1. Have the task service dispatch typed hook events at the same call sites where it calls `recordTaskEvent` (e.g., `task.run.pre_claim` is a true `internal/hooks` hook dispatched inside `ClaimNextRun` before commit; `task.run_enqueued` audit row continues to be written by `recordTaskEvent` post-commit). Hook surface is owned by `internal/task` calling into `internal/hooks` through a thin `HookDispatcher` interface (defined in `internal/task`, implemented in daemon).
  2. Add a new `internal/daemon/task_hook_bridge.go` that subscribes to the existing task event notifier (`m.notifyTaskObserverBestEffort` at `internal/task/manager.go:2811`) and re-emits filtered task events as hooks. Pre-* hooks are not possible on this path; the bridge is observation-only.
  Pick option 1 for MVP because the spec already calls for `pre_claim`/`pre_create` mutability that can deny a request before commit. Option 2 cannot deny pre-commit. Add a note: existing audit events stay where they are; hooks are an additional surface co-emitted by the task service.

### B2. Capability-match index strategy must be decided, not deferred

- **Severity**: high
- **Files/sections**: `_techspec.md:189–206`, `adrs/adr-003.md:35–37`
- **Why it matters**: The TechSpec literally says "task decomposition must choose an index strategy before implementation." That punt means `cy-create-tasks` will receive an instruction to decide between two materially different designs (JSON column scan vs side-table) with no guidance, and the round-1 reviewer flagged this same issue (3.4). Worse, the schema for `task_runs` is changing in step 6 and capability fields land alongside claim fields, so the side-table-or-not decision has to be made before the migration DDL is written. Without a decision, two of the highest-risk MVP tasks (D1 schema and D2 ClaimNextRun) cannot be specified atomically.
- **Concrete fix**: Pick exact-match-on-bounded-set for MVP and ship as a side table:
  - `task_run_required_capabilities(run_id TEXT, capability_id TEXT, PRIMARY KEY(run_id, capability_id))` with index `(capability_id, run_id)`.
  - `task_run_preferred_capabilities(run_id TEXT, capability_id TEXT, ...)` analogous (or a tagged column on the same table).
  - `ClaimNextRun(criteria)` joins on the side table for capability filtering.
  - The original `required_capabilities_json` column can stay as a serialized denormalized projection for prompt rendering or be omitted entirely.
  The TechSpec should state this and remove the "choose later" line. Update step 6 to enumerate the side tables in the DDL plan.

### B3. Schema duplication between proposed columns and existing run model

- **Severity**: high
- **Files/sections**: `_techspec.md:170–204`, `internal/store/globaldb/global_db.go` (run schema), `internal/task/types.go:260–278` (Run struct)
- **Why it matters**: The TechSpec proposes adding `claimed_session_id`, `created_by_actor_kind`, `created_by_actor_id`, `started_by_actor_kind`, `started_by_actor_id`, and `execution_requested_at` to `task_runs`. The current `Run` struct already carries `ClaimedBy *ActorIdentity` (kind+ref tuple) and `Origin Origin` (kind+ref tuple), and `Task` already carries `CreatedBy ActorIdentity`. `task_runs` already has `claimed_by_kind`, `claimed_by_ref`, `session_id`, `origin_kind`, `origin_ref`, `queued_at`. Adding parallel fields creates two sources of truth for the same actor data and risks cy-create-tasks emitting tasks that simultaneously add and use both. `claimed_session_id` is also redundant with `session_id` and with the `(claimed_by_kind='agent_session', claimed_by_ref=<sessionID>)` tuple already in place.
- **Concrete fix**: Reconcile in the TechSpec data-model section:
  - Drop `claimed_session_id` from the proposed schema. Use the existing `session_id` column (or `claimed_by_ref` when `claimed_by_kind='agent_session'`). State explicitly which of the two is the canonical "the agent session that owns this run" identifier post-MVP.
  - Drop `created_by_actor_kind/id` and `started_by_actor_kind/id` from `task_runs`. Task creation already records actor on the task; per-run actors already record through `claimed_by_*` and `Origin`. If the spec needs a "who triggered this run enqueue" fact, add a single pair (e.g., `enqueued_by_kind`/`enqueued_by_ref`) — but only if existing actor plumbing on the enqueue path is insufficient; verify against `internal/task/manager.go:1330–1378`.
  - Drop `execution_requested_at`. `queued_at` already records the run-enqueue moment.
  - Keep: `claim_token`, `lease_until`, `heartbeat_at`, `claim_attempts`, `last_claim_error`. These are genuinely new.
  - Keep `execution_mode` if and only if the TechSpec needs to distinguish "coordinated" vs "manual" runs at the storage layer; otherwise treat it as a `metadata_json` field. Note the TechSpec itself flags this ambiguity ("on task runs or equivalent start request metadata").
  This reconciliation should be explicit because the existing fields are easy to overlook from outside the package and `cy-create-tasks` will most likely read the spec literally and create migration tasks that double-up the schema.

## Non-Blocking Issues

### N1. Publish/start/approve trigger maps onto three different existing operations

- **Severity**: medium
- **Files/sections**: `_techspec.md:11, 56, 70, 86, 304`, `adrs/adr-005.md:21–27`, `adrs/adr-010.md:13, 28`
- **Why it matters**: The TechSpec uses "publish, start, or approve execution" interchangeably as the coordinator trigger. The current task code distinguishes:
  - `task.published` event (`internal/task/manager.go:21, 522`) — draft → published state move
  - `task.approved` event (`internal/task/manager.go:22, 588`) — approval-policy gate
  - `task.run_enqueued` event (`internal/task/manager.go:28, 1367`) — actual run creation
  These are distinct lifecycle moments. The coordinator trigger is the run-enqueue boundary (and ADR-005 says so), but mixing the three terms in narrative prose risks task decomposition treating publish-as-trigger or approve-as-trigger.
- **Concrete fix**: In every prose passage, explicitly say "the run-enqueue boundary (`task.run_enqueued`)" rather than "publish/start/approve". Keep "publish/start/approve" only where the spec describes user-facing actions that result in a run enqueue. Add one line that spells out the mapping: publish-without-approval-policy auto-enqueues a run; publish-with-manual-approval enqueues on approval; explicit `task start` (when added) enqueues directly. This preserves the user-action surface while collapsing the trigger to one event.

### N2. Workspace scope vs global scope is not addressed

- **Severity**: medium
- **Files/sections**: `_techspec.md:84–91`, `adrs/adr-005.md:19, 39`
- **Why it matters**: ADR-005 says "one coordinator per workspace." The current task model has `Scope` = `global | workspace` (`internal/task/types.go:11`). What happens for `ScopeGlobal` task runs? Is there a dedicated daemon-global coordinator? Does global work pin to a default workspace's coordinator? Is global execution disallowed in MVP? Without an answer, `cy-create-tasks` cannot specify the coordinator-bootstrap test matrix (Phase G).
- **Concrete fix**: Add one sentence to the Coordinator Trigger subsection: "Global-scope task runs trigger a daemon-global coordinator (one per daemon, distinct from any workspace coordinator)." Or, simpler for MVP: "Global-scope tasks do not trigger coordinator auto-spawn in MVP; they require explicit operator assignment." Pick one and move on.

### N3. ClaimCriteria.SessionID semantics are ambiguous

- **Severity**: low
- **Files/sections**: `_techspec.md:108–115`
- **Why it matters**: `ClaimCriteria` includes both `WorkspaceID` and `SessionID`. The intent is presumably "the calling session ID for audit and ownership recording", not "filter to runs already pinned to this session". A reader implementing the SQL filter from the criteria struct alone could go either way.
- **Concrete fix**: Rename `SessionID` to `ClaimerSessionID` (or `ActingSessionID`). Add a one-line doc comment in the spec saying it identifies the would-be claimant for ownership recording, never as a filter.

### N4. Hook patch surface for `task.run.pre_claim` could enable starvation

- **Severity**: low
- **Files/sections**: `_techspec.md:233–242, 343–347`
- **Why it matters**: Pre-claim payloads listed as having patches mean a hook can mutate the criteria before the daemon tries to claim. A misbehaving or malicious hook could rewrite criteria to never match, starving an otherwise-eligible session. The TechSpec's "hooks may deny or narrow where explicitly safe" is the right posture but doesn't enumerate which fields are mutable.
- **Concrete fix**: In the hook payloads section, mark `TaskRunPreClaimPayload` as mutation-allowed only on `RequiredCapabilities` and `PriorityMin` (or denote "observation-only" for MVP). Pre-claim hook denial is fine; pre-claim hook mutation is risky and not needed for MVP.

### N5. Scheduler hook events overlap with task-domain run events

- **Severity**: low
- **Files/sections**: `_techspec.md:316–321`
- **Why it matters**: `scheduler.wake`, `scheduler.no_match`, `scheduler.recovered` are listed as hook events. But `scheduler.recovered` is a recovery action over a run — equivalent to `task.run.lease_recovered` from a different angle. Two events for the same fact creates double-count risk in observability.
- **Concrete fix**: Keep `scheduler.wake`, `scheduler.no_match` (which describe scheduler-level behavior). Remove `scheduler.recovered` and rely on `task.run.lease_recovered` to describe per-run recovery. Or vice versa, but not both.

### N6. `agh ch join` / `JoinAdditionalChannel` conflicts with single-channel sessions

- **Severity**: low
- **Files/sections**: `_techspec.md:285, 270`, `adrs/adr-007.md:18, 24`
- **Why it matters**: The CLI lists `agh ch join` and the API lists `POST /agent/channels/{channel}/join`, but ADR-007 explicitly keeps single-channel sessions for MVP. If sessions can only have one channel, "join" either replaces the current channel or is a no-op.
- **Concrete fix**: Either:
  - Drop `agh ch join` and the join endpoint from MVP and call out that channel switching is a separate operator action; OR
  - Document `join` as switch-channel semantics for MVP, with a forward path to multi-home in the post-MVP network evolution.
  The first is cleaner.

### N7. `agh task next` and `agh task done` aliasing

- **Severity**: low
- **Files/sections**: `_techspec.md:285–298`
- **Why it matters**: Existing CLI has `task complete` and the spec adds `task done` as an alias plus introduces `task next` and `task release`. Round-1 review noted the analysis used `pass` while the spec says `release`. Adding an alias requires test coverage for both names and increases CLI surface for marginal gain.
- **Concrete fix**: Pick canonical names: `next`, `heartbeat`, `release`, `complete`, `fail`. Drop the `done` alias for `complete`. Drop the `pass` alias entirely. Operator commands continue to use `complete`; agent commands use the same canonical names with implicit identity. One name per concept, no aliases.

### N8. `agh task create` permissioning for agents is undefined

- **Severity**: medium
- **Files/sections**: `_techspec.md:298`, `adrs/adr-002.md:20`
- **Why it matters**: The CLI lists `agh task create` "for coordinator and permitted agent-side decomposition" without saying how the daemon decides which agents are "permitted." Without a permission predicate, this becomes either "all agents can create tasks" (security smell) or "only coordinator can" (scope limit; round-1 issue 3.3 noted the coordinator needs this verb).
- **Concrete fix**: Add a one-paragraph permission rule: "Agent-initiated task creation is gated by a session-level capability atom (e.g., `task.create`). Coordinator sessions receive it by default; spawned worker sessions do not unless the spawn explicitly grants it (cannot widen). Hook `task.created` fires with the creator session and any hook can deny." Then `cy-create-tasks` can decompose this as one safety task.

### N9. Coordinator-config workspace-override resolution is "may ship later"

- **Severity**: low
- **Files/sections**: `_techspec.md:222–229`, `adrs/adr-005.md:31–37`
- **Why it matters**: TechSpec says workspace overrides may ship after global config. That's fine, but `CoordinatorConfig` resolver call sites need a stable signature so adding workspace lookup later is not breaking.
- **Concrete fix**: In the implementation-design section, lock the resolver signature: `ResolveCoordinatorConfig(ctx, workspaceID string) (CoordinatorConfig, error)` from day one. Day-1 implementation may ignore `workspaceID` and return global config; day-N implementation reads the workspace override. This preserves call-site stability without forcing workspace plumbing into MVP.

### N10. `claim_attempts` and `last_claim_error` columns are added but not written

- **Severity**: low
- **Files/sections**: `_techspec.md:179–180`
- **Why it matters**: Round-1 reviewer flagged this (4.1). Schema columns added without a writer become dead state. If they exist for future capacity, say so; if they're for a specific recovery rule (e.g., max-attempts-before-fail), spec the rule.
- **Concrete fix**: Either remove the two columns from MVP and re-introduce them when their writer ships, or document a single rule (e.g., "increment `claim_attempts` on each `ClaimNextRun` success; do not enforce a cap in MVP — observability only").

### N11. MVP memory work scope is fuzzy

- **Severity**: low
- **Files/sections**: `_techspec.md:46–47, 376, 380–387`, `adrs/adr-008.md`
- **Why it matters**: The TechSpec's MVP cut line says steps 11–15 are post-MVP; memory work is in step 12. But `agent_name` provenance is referenced as ground-state assumed-correct in autonomy hook payloads, log fields, and the coordinator-decision hook. Either provenance plumbing is in MVP (because hooks need it) or post-MVP (because step 12 is post-MVP).
- **Concrete fix**: State explicitly: "`agent_name` and `session_id` provenance plumbing through hook payloads and structured logging is part of MVP. Broader memory scope work (peer/channel scopes, automatic per-turn extraction, session-end summaries) is post-MVP." Then memory-package writes that need agent_name correctly written are scoped into MVP D-phase tasks where they're consumed.

### N12. SchedulerWakePayload is unnecessary as a hook

- **Severity**: low
- **Files/sections**: `_techspec.md:240, 318–320`, `adrs/adr-009.md:24`
- **Why it matters**: Scheduler wakeups are pure internal optimization; nothing external should react to "scheduler ticked because new run appeared." Promoting it to a hook adds payload/patch/introspection work for no operator value (the operator-meaningful fact is "run was enqueued", which is already covered by `task.run.pre_claim` / `task.run.post_claim`).
- **Concrete fix**: Demote `scheduler.wake` and `scheduler.no_match` to internal observability events (metrics + logs only). Remove from hook taxonomy. Keeps ADR-009's "internal scheduler bookkeeping private" wording honest.

### N13. `IterationCurrent` increment work isn't claimed by any MVP step

- **Severity**: low
- **Files/sections**: `_techspec.md:438–448`
- **Why it matters**: Analysis flagged `IterationCurrent` as a dead column. Step 13 (post-MVP) is "self-correction and telemetry beyond minimal counters." If iteration counting is not claimed by an MVP step, the dead column stays dead through the autonomy MVP — which is fine, but the spec should not imply otherwise.
- **Concrete fix**: One sentence: "MVP does not increment `IterationCurrent`; the column remains as-is until the post-MVP self-correction step." Or pull the increment work into MVP if any decision actually depends on it (none do per the current spec).

### N14. `agh me context` payload is not specified

- **Severity**: low
- **Files/sections**: `_techspec.md:268`
- **Why it matters**: Round-1 4.8 flagged this; still no field set in the spec. `cy-create-tasks` will spec this freely if not pinned, leading to drift between the situation surface (Phase B) and the agent CLI (Phase C).
- **Concrete fix**: Add a "MeContext payload" mini-section listing the section names (e.g., `self`, `workspace`, `task`, `inbox_summary`, `peer_roster`, `capabilities`, `limits`) and stable order. Truncation rules can be looser ("each section caps at N entries; full data via dedicated endpoints").

### N15. Missing operator-facing publish/start CLI is implied but not listed

- **Severity**: low
- **Files/sections**: `_techspec.md:301–302`
- **Why it matters**: The spec says "Existing or future operator commands remain explicit, for example `agh task create --workspace ...`, `agh task start --workspace ...`". `agh task start` is not currently a verb (`internal/cli/task.go` has `enqueue`, `claim`, `start` for runs but no top-level `task start`). Manual-control flows depend on this verb existing or on `task publish` + auto-enqueue.
- **Concrete fix**: Either commit to adding `agh task publish` and `agh task start` as MVP operator verbs (and list them in the CLI section) or commit to "publishing a draft auto-enqueues a run when no approval is required" and remove the `task start` reference. Pick one.

### N16. ADR dates are all 2026-04-25 but ADR-010 is 2026-04-26

- **Severity**: trivial
- **Files/sections**: `adrs/adr-010.md:9`
- **Why it matters**: Cosmetic, but the ADR set should read as one coherent design pass. Mixed dates suggest unfinished review.
- **Concrete fix**: Align dates if they were authored in the same pass; otherwise leave as-is.

## Consistency Checks

### TechSpec vs ADRs

- **Aligned**: ADR-001 phased scope, ADR-002 CLI-before-MCP, ADR-003 lease-on-task_runs, ADR-004 split scheduler/coordinator, ADR-005 spawn-on-run-enqueue, ADR-006 safe spawn (lineage/TTL/narrowing), ADR-007 minimal network, ADR-008 memory provenance first, ADR-009 typed hooks (no event bus), ADR-010 manual-first.
- **Tightening needed**: ADR-009 references `internal/daemon/hooks_bridge.go` as the extension point but the TechSpec uses `internal/hooks` directly (B1). Reconcile: either ADR-009 specifies a bridge or the bridge is dropped.
- **Tightening needed**: ADR-005 says one coordinator "per workspace"; TechSpec also says global-scope tasks exist (`Scope` enum). Behavior for global scope is unaddressed (N2).

### TechSpec/ADRs vs current AGH code

- **Confirmed correct**:
  - `task_runs` lacks `claim_token`, `lease_until`, `heartbeat_at` columns. Net-new (B3 still applies for the additional duplicative columns).
  - `ClaimNextRun(criteria)` does not exist. Current `ClaimRun(runID, ...)` is per-run. Net-new.
  - `RecoverRunOnBoot` exists at `internal/task/manager.go:1685` (boot recovery is real).
  - Hook taxonomy in `internal/hooks/events.go` does not include `task.*`, `coordinator.*`, `scheduler.*`, `spawn.*`. All net-new.
  - `CreateOpts` has no `ParentSessionID`, no spawn semantics. Net-new.
  - `internal/cli/task.go` has `enqueue`, `claim`, `start`, `attach-session`, `complete`, `fail`, `cancel`. Missing `next`, `heartbeat`, `release`. Net-new.
  - `internal/api/contract/contract.go` has no `ClaimCriteria`, `ClaimedRun`, `SpawnOpts`, `CoordinatorConfig`. Net-new.
  - No `[autonomy.coordinator]` config block. Net-new.
- **Mismatches**:
  - Task domain emits `task.run_enqueued` as an audit event via `recordTaskEvent` at `internal/task/manager.go:1367`, not as an `internal/hooks` dispatch (see B1).
  - Existing `Run` actor fields (`ClaimedBy`, `Origin`, plus task-level `CreatedBy`) overlap with proposed schema columns (see B3).
  - Existing task statuses include `pending`, `blocked`, `in_progress`, etc. — not just draft/ready/queued. The TechSpec's three-state mental model ("draft / blocked / ready") is incomplete; the actual machine has more states. This is fine for prose but `cy-create-tasks` should reference the canonical enum at `internal/task/types.go:21–38`.

### Round 1 findings

- **Resolved**: Scheduler vs pull authority (round-1 §2.1) — TechSpec §3.4 ("Scheduler and Claim Authority") explicitly makes ClaimNextRun authoritative; scheduler is sweep/notify only. Confirmed.
- **Resolved**: Coordinator trigger (round-1 §2.2) — TechSpec §3.5 ("Coordinator Trigger") and ADR-005 fully specify it.
- **Resolved**: Lease invariants (round-1 §2.3) — TechSpec "Lease invariants" subsection enumerates stale-heartbeat, late-complete, sweep concurrency, boot recovery ordering, lease cap.
- **Resolved**: Permission narrowing comparator (round-1 §2.4) — "Permission narrowing" subsection names atom space (tools, skills, MCP server IDs, workspace path grants, network channels, env profile grants) and reject-on-unknown rule.
- **Resolved**: TTL × lease (round-1 §2.5) — "TTL and active leases" subsection states reaper-wins with structured release reasons.
- **Resolved**: Hook family naming (round-1 §3.1) — TechSpec uses `coordinator.*`, `scheduler.*`, `spawn.*`, `task.run.*`, no `autonomy.*` prefix, no `workflow.*` family.
- **Resolved**: MVP cut line (round-1 §1.7) — Build Order explicitly marks 1–10 MVP and 11–15 post-MVP.
- **Still open**: Round-1 §3.4 (capability-match index strategy) → escalated to B2 above. The TechSpec acknowledged it as a deferred decision, but cy-create-tasks needs it locked.
- **Still open**: Round-1 §3.6 (workflow concept) — TechSpec correctly drops `workflow.*` hook family but `workflow_id` still appears as correlation metadata in payloads and structured logs. Acceptable; matches ADR-009 wording. No further action.
- **Still open**: Round-1 §4.4 (`agh me logout`) — TechSpec dropped this verb; correct.
- **Still open**: Round-1 §4.10 (eval determinism) — Step 14 is post-MVP; defer is fine.

### GPT-5.4 Mini findings

- **Multica analysis**: confirms the corrected model. One wording-clarity ask (creation vs run-enqueue distinction) → already covered by N1.
- **Paperclip analysis**: confirms the model. Two clarifications: "claim_token/lease distinct from sandbox/environment lease" — TechSpec already says this at line 27 ("`lease` means task-run ownership lease; intentionally separate from future sandbox, workspace runtime, or environment leases"). And "claim vs wakeup boundary explicit" — TechSpec §3.4 ("Scheduler and Claim Authority") covers it. No further action.
- **AGH-code analysis**: notes `claim_token`/`lease_until`/`heartbeat_at` are net-new and the approval path does not currently trigger coordinator behavior. Both correct; see B1, B3, N1.
- **Convergence**: All three GPT-5.4 reviews and round-1 Opus review agree on the same shape. Multica/Paperclip both warn against a second durable scheduler queue and against an `orchestration_required` task-creation flag — TechSpec respects both.

### Manual control model

- TechSpec explicitly supports user-created tasks, user-started sessions, direct prompting, and counter-check sessions on the same task/session contracts (§"Manual Control Contract", ADR-010). The integration test list (`_techspec.md:411–423`) covers the manual-create → manual-claim flow and the manual-counter-check session flow. ADR-010 mandates the integration tests.
- Risk: `agh task next --wait` from a user-started session pulls work that the coordinator might also be planning to push. The TechSpec's collapse to one claim path (ClaimNextRun) avoids the duplicate-claim risk; per-session lease cap (N17 below) bounds how much one session can hold.

### Coordinator trigger model

- Trigger = run-enqueue boundary. Idempotent per workspace. Coordinator cannot spawn a coordinator. Manual sessions don't trigger startup. Agent-created tasks inherit coordinator workflow. All consistent across ADR-005, ADR-010, and TechSpec §3.5.
- Gap: global-scope tasks (N2). Otherwise sound.

### Claim/lease model

- Single-source-of-truth: `task_runs`. Confirmed correct.
- One authoritative primitive: `ClaimNextRun`. Confirmed.
- Sweep is CAS, boot recovery before claim traffic, lease extension bounded. Confirmed.
- Per-session lease cap not stated. Round-1 §3.10 recommended adding "default 1 active lease per session" — TechSpec has not adopted. Recommend adding (see N17).

### Hooks/resources extensibility model

- Typed hooks, no event bus, no separate plugin system. Confirmed.
- New hook families (`coordinator`, `scheduler`, `spawn`, `task.run`) instead of an `autonomy.*` umbrella. Confirmed.
- Bridge mechanic from task domain events to hook dispatches is the missing piece (B1).
- Zero new resource kinds for MVP. Confirmed; aligns with ADR-009.

### MVP/post-MVP boundary

- Steps 1–10 are MVP, 11–15 post-MVP. Coordinator bootstrap at step 10 is the demo milestone.
- Demo milestone after step 7 (round-1 §3.7 recommendation) is implicit but not labeled. Recommend explicit labeling (see N18 below).

## Recommended Edits

### `_techspec.md`

- **§"Data Flow" (lines 53–62)**: rewrite step 1–3 to explicitly distinguish "task creation" (no claimable work, no coordinator), "approval/publish gates" (state moves), and "run enqueue" (claimable, coordinator trigger). Use canonical event name `task.run_enqueued` everywhere instead of mixing it with "publish/start/approve". (N1)
- **§"Implementation Design / Data Models / Task run claim fields" (lines 174–180)**: drop `claimed_session_id`, `last_claim_error`, optionally `claim_attempts`. State that `session_id` (existing) is the canonical owner reference. (B3, N10)
- **§"Implementation Design / Data Models / Task capability fields" (lines 189–206)**: replace the JSON-with-deferred-index-decision with a side-table design (B2). Drop `created_by_actor_kind/id`, `started_by_actor_kind/id`, `execution_requested_at`. Keep `execution_mode` only if needed at the storage layer; otherwise move to `metadata_json`. (B3)
- **§"Implementation Design" (anywhere appropriate)**: add a "Domain Events vs Hooks Bridge" subsection explaining that the task service co-emits typed `task.run.*` hooks at the same call sites as `recordTaskEvent`, with pre-* hooks dispatched before commit and post-* hooks dispatched after. (B1)
- **§"Hook Payloads" (lines 232–242)**: mark `TaskRunPreClaimPayload` mutation surface as observation-only OR limit to `RequiredCapabilities`/`PriorityMin`. Demote `scheduler.wake` and `scheduler.no_match` to internal observability events; remove from hook taxonomy. Drop `scheduler.recovered` (collapse with `task.run.lease_recovered`). (N4, N5, N12)
- **§"API Endpoints" (lines 265–280)**: drop `POST /agent/channels/{channel}/join` from MVP if ADR-007 stays single-channel. (N6)
- **§"CLI Commands" (lines 285–302)**: drop `agh task done` alias. Drop `agh task pass` (already not in spec; confirmed). Add a permission-rule sentence for `agh task create` saying agent identity must hold a `task.create` capability atom. Confirm operator `agh task publish` / `agh task start` verbs are MVP or document the auto-enqueue-on-publish path. (N7, N8, N15)
- **§"Core Interfaces" (lines 108–115)**: rename `ClaimCriteria.SessionID` to `ClaimerSessionID` with a doc comment. Lock `ResolveCoordinatorConfig(ctx, workspaceID)` resolver signature. (N3, N9)
- **§"Coordinator Trigger" (lines 84–91)**: add one sentence on `ScopeGlobal` task-run behavior. (N2)
- **§"Lease invariants" (lines 244–251)**: add per-session lease cap default ("a session may hold at most 1 active lease in MVP; cap is configurable"). (N17 below)
- **§"Build Order" (lines 432–448)**: label "after step 7, agents can self-claim ready tasks; this is the first end-to-end autonomy demo milestone". (N18 below)
- **§"MeContext payload" (new)**: enumerate sections in `agh me context`. (N14)
- **§"Memory provenance" (anywhere)**: state that `agent_name`/`session_id` provenance plumbing is MVP and broader memory scope work is post-MVP. (N11)
- **§"IterationCurrent"**: one sentence saying MVP does not increment this column. (N13)

### `adrs/adr-009.md`

- Reconcile the "Implementation Notes" section with the chosen bridge mechanic (B1). If the task service calls into hooks directly, the bridge mention should reflect that. If a separate bridge exists, name it.
- Add: hooks for `scheduler.*` are limited to operator-meaningful events (none in MVP unless promoted). (N12)

### `adrs/adr-005.md`

- Add: "Global-scope task runs trigger a daemon-global coordinator" or "are not coordinator-bootstrapped in MVP." (N2)

### `adrs/adr-010.md`

- (Optional) sync date to 2026-04-25 for consistency. (N16)

### Additional non-blocking issue worth recording

#### N17. Per-session lease cap default

- **Severity**: medium
- **Files/sections**: `_techspec.md:244–251`
- **Why it matters**: Round-1 §3.10 recommended a per-session lease cap as the natural backpressure unit. Not adopted yet. Without a cap, a single misbehaving session can claim every queued run.
- **Concrete fix**: Add to lease invariants: "A session may hold at most `N` active leases at a time. MVP default `N=1`. Cap is configurable per workspace; checked at claim time inside the same SQLite transaction."

#### N18. Mid-build demo milestone label

- **Severity**: low
- **Files/sections**: `_techspec.md:432–443`
- **Why it matters**: Round-1 §3.7 noted that after steps 1–7, agents can self-claim work end-to-end. Labeling this as a demo milestone helps MVP risk management — if step 8 (scheduler) hits trouble, steps 1–7 already deliver autonomy.
- **Concrete fix**: After step 7's bullet, add: "Demo milestone — at this point a user-started agent can self-claim and complete a queued task end-to-end. This is the first integrated autonomy validation point."

## Things To Keep As-Is

- **ADR-001 phased scope**: correct cut, well-justified, the right amount of pre-decided structure.
- **ADR-002 CLI before MCP**: stabilizing one contract first is the right call.
- **ADR-003 extend `task_runs`, no parallel queue**: single source of truth is the right invariant; matches Multica and Paperclip.
- **ADR-004 split scheduler from coordinator**: keeps mechanical safety deterministic and semantic decisions in an LLM session, with `ClaimNextRun` as the only claim path.
- **ADR-005 spawn-on-run-enqueue with config precedence**: avoids idle coordinator sessions, lets users draft tasks freely. Round-1 reviewer's concern about implicit trigger is addressed by the explicit run-enqueue boundary.
- **ADR-006 safe spawn (lineage, TTL, permission narrowing, reject unknown atoms, reaper-wins)**: every safety primitive is named, defaults are conservative, and the reaper-vs-lease policy is explicit.
- **ADR-007 minimal network evolution**: defers cross-daemon swarm correctly. No ADR-007 changes needed.
- **ADR-008 memory provenance before scopes**: provenance-first is the right ordering; broader scope work belongs in a later TechSpec.
- **ADR-009 first-class hooks, no event bus, no second plugin system, zero new resource kinds for MVP**: the right answer for AGH's existing extensibility model. The bridge mechanic (B1) is the only piece to clarify; the architectural posture is correct.
- **ADR-010 manual-first**: critical to preserve. The integration tests it requires (manual create → run enqueue → coordinator → worker; manual session direct prompt without coordinator) are the right bookends.
- **Lease invariants subsection**: complete and correct race contract. Don't reopen.
- **Permission narrowing subsection**: atom space and reject-on-unknown are the right design.
- **TTL × lease (reaper wins with structured release reasons)**: don't reopen; this is the right pick.
- **Build order steps 1–10 as MVP, 11–15 as post-MVP**: structurally sound. Just needs the in-progress demo milestone label (N18).
- **No `workflow.*` hook family**: correct; `workflow_id` as correlation metadata is the right shape until a workflow package exists.
- **No new resource kinds for MVP**: correct; coordinator config goes through global config + workspace override.
- **`ClaimNextRun` as canonical primitive**: correct; agent CLI verb wraps the same call.
- **CLI namespaces (`me`, `ch`, `task`, `spawn`)**: correct split between operator and agent surfaces.

---

**Summary**: this is an approve-with-changes. Three blocking issues (B1 hook bridge, B2 capability index, B3 schema duplication) need explicit resolution in the TechSpec before `cy-create-tasks` runs; resolving them is each a paragraph or table edit, not a redesign. Eighteen non-blocking issues are polish or clarifications that can land before or alongside task decomposition. The fundamental architecture — hooks not bus, ClaimNextRun authoritative, scheduler advisory, run-enqueue trigger, manual-first peers — is sound and confirmed by every cross-check.
