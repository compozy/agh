# Child TechSpec: Task Run Review Gate

## Executive Summary

AGH will add a native task-run review gate inspired by the Codex Loop `goal` mechanism: after a worker terminalizes a run, AGH can request an independent reviewer selected through task-owned review profile policy, persist a typed verdict, and either accept the terminal result or enqueue a continuation run with bounded `missing_work` and `next_round_guidance`.

This child TechSpec is part of the aggregate program in [`_techspec.md`](_techspec.md). It is authoritative for review-gate policy, reviewer routing, persisted verdicts, continuation semantics, reviewer skill guidance, and review-gate API/UDS/CLI/tool/hooks.

The core design choice is conservative: review gate v1 is post-terminal. It does not add `pending_review` to the run lifecycle and does not block the execution terminal transition. A run can be historically `completed`, `failed`, or `canceled` while its review state later decides whether that terminal outcome is accepted or whether a follow-up run is queued. This preserves existing task-run transition semantics and avoids making review channels, reviewer transcripts, or prompt state part of execution authority.

## Research Basis

The design adapts useful pieces from local Codex research while rejecting Codex-specific hook mechanics:

- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/goal_confirm.go` defines a structured goal verdict with `completed`, `confidence`, `reason`, `missing_work`, and `next_round_guidance`, plus outcomes for completed/incomplete/error/timeout/invalid output.
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/hooks.go` runs a goal check on each Stop, continues unless the review says the goal is complete, and applies rapid-stop guardrails.
- `/Users/pedronauck/dev/ai/codex-loop-plugin/internal/loop/store.go` persists compact loop/review state across iterations.
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ThreadGoal.ts` models goals as thread-scoped objective/status/budget state.
- `.resources/codex/codex-rs/app-server-protocol/schema/typescript/v2/ReviewStartParams.ts` and related `ReviewTarget`, `ReviewDelivery`, and `ThreadSourceKind` types show review as a distinct thread/turn source, including `subAgentReview`.
- `internal/task/lease.go` already includes `review_request` in the default coordination message kinds, which means channels can already carry review coordination without becoming review authority.

AGH should copy the durable continuation pattern, typed verdict shape, missing-work guidance, and guardrails. AGH should not copy Codex Loop's shell command execution, local JSON loop store, prompt header activation, or hook-specific Stop semantics as runtime authority.

## Goals

- Add review-on-terminal-transition behavior for selected task runs.
- Persist review requests, attempts, verdicts, reviewer identity, evidence, and continuation guidance as task-owned state.
- Let the coordinator route reviewers through explicit peers, channel membership, and capability filters.
- Let task execution profiles select reviewer agents, providers, models, peers, channels, and capabilities without making those selectors verdict authority.
- Add a bundled `agh-task-reviewer` skill for one-shot reviewer sessions.
- Carry rejected-review guidance into the next worker through `TaskContextBundle`.
- Preserve channel/thread state as coordination transport only.
- Preserve `task_runs` and `task.Service` as the execution and review authorities.
- Make review state agent-operable through native tools, HTTP, UDS, CLI, hooks, observe/SSE, and generated contracts.

## Non-Goals

- Do not add `pending_review` to `task_runs.status` in MVP.
- Do not block `CompleteRunLease`, `FailRunLease`, or `CancelRun` while waiting for review.
- Do not infer a verdict from a channel message, thread transcript, bridge delivery, prompt text, or session summary.
- Do not store review verdicts only in channel/thread/session transcripts.
- Do not turn `internal/notifications` into a review workflow bus.
- Do not use bridge-delivered review workflows as the primary MVP gate.
- Do not implement multi-reviewer consensus, voting, quorum, or reviewer panels.
- Do not implement a general human approval matrix for every task class.
- Do not review every intermediate worker Stop if doing so requires a new execution state machine.
- Do not allow infinite retry or continuation loops. Every policy must have a bounded `max_rounds`.
- Do not expose raw claim tokens to reviewers.
- Do not use shell-configured reviewer commands as a core runtime primitive.
- Do not let `TaskExecutionProfile.Review`, reviewer profile fields, channel membership, or peer membership become review verdict authority.
- Do not make review approval equivalent to task approval policy. Existing manual `ApprovalPolicy` remains the pre-execution approval gate; review gate is a post-terminal quality/goal gate.

## MVP Boundary

In scope:

- Review policy defaults under `[task.orchestration.review]`.
- Task-level review policy overrides through task create/update surfaces.
- `TaskExecutionProfile.Review` integration for reviewer agent/provider/model hints plus peer/channel/capability selectors.
- A task-owned review table for requests/attempts/verdicts.
- Post-terminal review request creation when policy matches the terminal run.
- Reviewer routing through explicit peer, coordination channel, and capability selectors.
- Deterministic reviewer session bootstrap with bundled `agh-task-reviewer`.
- Native `submit_run_review` tool for reviewer sessions.
- Typed persisted verdicts with bounded evidence and continuation guidance.
- Continuation run enqueue on rejected verdicts while round limits allow.
- Circuit-open behavior for max rounds, reviewer failure exhaustion, rapid shallow terminalization, and invalid reviewer output.
- Context bundle projection of the latest rejected review for the next worker.
- HTTP, UDS, CLI, OpenAPI, generated TypeScript, hooks, observe/SSE, and docs.

Out of MVP:

- `pending_review` execution state.
- Pre-terminal or per-intermediate-turn review.
- External bridge-only reviewers that can approve by message alone.
- Reviewer marketplace/ranking.
- Multi-reviewer quorum.
- Review policies that mutate provider permissions or tool allowlists dynamically.
- Generic notification fan-out for review workflows.

## Authority Model

1. `task.Service` is the only authority for creating review requests, recording verdicts, opening review circuits, and enqueueing continuation runs.
2. `task_runs.status` remains execution history. Review does not rewrite a terminal run.
3. Review state lives in task-owned tables and task events, not in channels or notification cursors.
4. Reviewer sessions may read task context and write only through explicit review tools/API paths.
5. A reviewer may discuss evidence in a channel, but the verdict becomes authoritative only after `RecordRunReview` persists it.
6. The coordinator may route a reviewer but cannot mark a review approved/rejected unless it is also acting through the reviewer submission contract with a valid reviewer identity.
7. Channels are eligible reviewer discovery and coordination surfaces only.
8. `internal/notifications` may notify about review events in future work, but review replay and review authority use task review tables and `task_events`.
9. `TaskExecutionProfile.Review` is reviewer-selection input only. It does not approve, reject, block, retry, or terminalize work.
10. Reviewer agent/provider/model hints select reviewer execution shape; they do not bypass reviewer authorization, channel membership checks, tool policy, or persisted review binding.

## Architectural Boundaries

Review-gate implementation must preserve the aggregate package boundaries:

1. `internal/task` owns review request creation, reviewer-session binding, routing state persistence, verdict persistence, circuit state, and continuation run creation.
2. `internal/store/globaldb` stores review policy fields, continuation-run columns on `task_runs`, `task_run_reviews`, and review indexes through numbered migrations only.
3. `internal/coordinator` may resolve and route reviewers, but it must call task-service review APIs and cannot write verdicts directly.
4. `internal/daemon` wires reviewer sessions, native tools, coordinator runtime, and transport handlers as the composition root.
5. `internal/skills/bundled` stores `agh-task-reviewer` guidance only; it does not define review authority.
6. `internal/notifications` may notify future consumers about review events, but it must not become a review workflow bus.
7. `internal/network` and bridge channels carry review coordination only; they do not own verdicts.
8. HTTP and UDS transports mount shared `internal/api/core` review handlers and DTOs.
9. `internal/api/contract` owns generated review/profile DTOs and OpenAPI shapes; review HTTP/UDS/CLI implementations must not invent transport-only fields.

## Safety Invariants

1. A review verdict is authoritative only after `task.Service.RecordRunReview` persists it.
2. Channel messages, bridge messages, transcripts, and prompt summaries can never become verdicts.
3. A rejected review creates continuation work only through task-service-owned enqueue logic.
4. Review gate v1 never adds `pending_review` to `task_runs.status`.
5. Review does not rewrite a terminal run's historical execution status.
6. `TaskExecutionProfile.Review` selects reviewer execution shape only; it does not approve, reject, or block work.
7. Reviewer sessions must be bound to a persisted review request before `submit_run_review` is available.
8. Review packets and review events must never expose raw claim tokens.

## Lifecycle

### Terminal Run With Review Policy

1. A worker calls `CompleteRunLease`, `FailRunLease`, or the task service otherwise writes a terminal run.
2. `task.Service` commits the terminal run state, task status projection, and terminal `task_events` row.
3. After the terminal transaction commits, `task.Service` runs a follow-up review-request transaction through `RequestRunReviewForTerminalRun`.
4. If the policy does not match, no review request is created.
5. If the policy matches, `task.Service` creates a `task_run_reviews` row with `status = "requested"` and emits `task.run_review_requested`.
6. In the same follow-up transaction, `task.Service` writes the new `review_id` to `task_runs.review_request_id` and clears `task_runs.review_required = 0`.
7. Daemon composition root receives the typed `ReviewRouter.OnRunReviewRequested` callback emitted at the review-request call site and wakes coordinator routing. It must not tail `task_events` or `task_run_reviews` to discover new review work.
8. Coordinator runtime selects a reviewer by explicit reviewer agent/provider/model, peer, channel membership, and capability filters.
9. The reviewer session is bound to the persisted review request through `task.Service.BindRunReviewSession` before `submit_run_review` is exposed. The session loads `agh-task-reviewer`, receives a bounded review packet, and may coordinate through the configured review channel.
10. The reviewer calls `submit_run_review` with a typed verdict.
11. `task.Service.RecordRunReview` validates idempotency, actor eligibility, state transition, bounds, and policy limits.
12. `approved` accepts the run's terminal outcome and emits `task.run_review_approved`.
13. `rejected` records missing work. If `max_rounds` remains, `task.Service` enqueues a continuation run and emits `task.run_review_retry_enqueued`. If exhausted, it opens the review circuit and emits `task.run_review_circuit_opened`.
14. `blocked` opens a task review blocker and leaves the task blocked until an operator/agent resolves the blocker through existing task surfaces.
15. `error`, `timeout`, and `invalid_output` retry reviewer attempts until `max_review_attempts` is exhausted, then open the review circuit according to `failure_policy`.

Review-request transaction boundary:

- The terminal run transaction never waits for reviewer routing or reviewer session start.
- The terminal run transaction stores `task_runs.review_required`, `review_request_round`, and `review_policy_snapshot` with the terminal row.
- Review-request creation is a follow-up task-service transaction triggered at the terminal-write call site after the terminal transition is durable.
- The follow-up transaction is idempotent on `(run_id, review_round, attempt = 1)`. A duplicate trigger returns the existing attempt-1 review row, ensures `task_runs.review_request_id` points to it, clears `task_runs.review_required`, and does not emit duplicate review events.
- If AGH crashes after terminal commit but before review-request creation, daemon startup recovery asks `task.Service` for terminal runs with `review_required = 1` and empty `review_request_id`, then re-runs the same idempotent request path.
- Policy updates after terminalization do not affect the already-written `review_policy_snapshot` for that terminal run.

### Continuation Run

A continuation run is a normal `task_runs` row. It is not a hidden retry lane.

Review trigger and continuation metadata live on typed `task_runs` columns, not in `metadata_json`.

Terminal review trigger fields:

- `review_required`: true when the terminal run must create a review request.
- `review_request_round`: review round that should be requested for this terminal run.
- `review_policy_snapshot`: bounded policy enum captured when the run terminalized.
- `review_request_id`: review row created by the follow-up request transaction. It is null until the row is durable, then set while `review_required` is cleared.

Continuation fields:

- `parent_run_id`: the reviewed terminal run.
- `review_id`: the rejected review row that created the continuation.
- `review_round`: next round number.
- `continuation_reason = "review_rejected"`.
- `missing_work_json`: bounded canonical JSON array copied from reviewer missing-work items. It is stored as JSON only because missing-work items are an ordered guidance list, not a SQL matching dimension.
- `next_round_guidance`: bounded reviewer guidance.

The next worker receives this information through `TaskContextBundle.ReviewContinuation`. The worker must still claim the run through normal `ClaimNextRun` and mutate it through session-bound lease APIs/tools.

Continuation-run profile precedence:

1. The task's current `TaskExecutionProfile` at continuation enqueue time controls worker, reviewer, participant, coordinator-guidance, and sandbox selection.
2. Continuation columns on `task_runs` provide review context only; they do not override task profile selection or grant permissions.
3. The reviewed run's `coordination_channel_id`, `required_capabilities`, and `preferred_capabilities` are copied to the continuation only when the effective task profile does not define the corresponding worker/participant selectors.
4. Workspace defaults apply only after task profile and copied run-native fields resolve to inherit or empty.

### Approved Review

Approval means the reviewer accepts the terminal outcome for the configured review objective. It does not mutate the run from failed to completed or completed to failed. If a failed run is reviewed under an `always` policy and approved, the failure remains the accepted terminal result.

### Rejected Review

Rejection means the terminal run did not satisfy the review objective. It does not rewrite the previous run. It creates a new run only through `task.Service.EnqueueRun` / task-service-owned continuation logic, within `max_rounds`.

## Review Policy

Review policy is default-off. It can be enabled globally through config defaults and overridden per task.

```go
type TaskReviewPolicy string

const (
	TaskReviewPolicyNone      TaskReviewPolicy = "none"
	TaskReviewPolicyOnSuccess TaskReviewPolicy = "on_success"
	TaskReviewPolicyOnFailure TaskReviewPolicy = "on_failure"
	TaskReviewPolicyAlways    TaskReviewPolicy = "always"
)

type ReviewFailurePolicy string

const (
	ReviewFailurePolicyBlockTask ReviewFailurePolicy = "block_task"
	ReviewFailurePolicyFailTask  ReviewFailurePolicy = "fail_task"
)

type ReviewPolicy struct {
	Policy                   TaskReviewPolicy   `json:"policy"`
	MaxRounds                int                `json:"max_rounds"`
	MaxReviewAttempts        int                `json:"max_review_attempts"`
	Timeout                  time.Duration      `json:"timeout"`
	RapidTerminalWindow      time.Duration      `json:"rapid_terminal_window"`
	RapidTerminalLimit       int                `json:"rapid_terminal_limit"`
	FailurePolicy            ReviewFailurePolicy `json:"failure_policy"`
	ReviewerSelector         ReviewerSelector   `json:"reviewer_selector"`
	MissingWorkMaxItems      int                `json:"missing_work_max_items"`
	MissingWorkItemMaxBytes  int                `json:"missing_work_item_max_bytes"`
	ReasonMaxBytes          int                `json:"reason_max_bytes"`
	ReviewTextMaxBytes      int                `json:"review_text_max_bytes"`
	NextGuidanceMaxBytes    int                `json:"next_round_guidance_max_bytes"`
}
```

Task execution profiles feed the effective review selector:

```go
type ReviewProfile struct {
	AgentName           string           `json:"agent_name,omitempty"`
	Provider            string           `json:"provider,omitempty"`
	Model               string           `json:"model,omitempty"`
	AllowedAgentNames   []string         `json:"allowed_agent_names,omitempty"`
	PreferredAgentNames []string         `json:"preferred_agent_names,omitempty"`
	Selector            ReviewerSelector `json:"selector"`
}
```

`ReviewProfile` is persisted with the task execution profile tables defined in [`_techspec_orchestration.md`](_techspec_orchestration.md). The review-gate service reads the effective profile when creating or routing a review; it does not store profile policy only in review request metadata.

Effective reviewer selector precedence:

1. Start with config defaults and the task's `ReviewPolicy.ReviewerSelector`.
2. Overlay `TaskExecutionProfile.Review.Selector`.
3. Overlay top-level `ReviewProfile.AgentName`, `Provider`, `Model`, `AllowedAgentNames`, and `PreferredAgentNames` onto the same-named selector fields. These top-level fields are convenience/profile fields and win over the nested selector when both are set.
4. Apply `RouteRunReviewRequest.Selector` only as a narrowing/concrete routing request. It may choose a peer/channel/agent inside the effective selector, but it cannot widen allowed agents, capabilities, peers, channels, provider/model gates, or `ParticipantPolicy`.
5. Any conflict between layers fails closed with a deterministic selector validation error; the runtime must not silently fall back to a broader reviewer.

Policy matching:

- `none`: never create review requests.
- `on_success`: review completed runs.
- `on_failure`: review failed/canceled runs.
- `always`: review completed, failed, and canceled runs.

Default config:

```toml
[task.orchestration.review]
default_policy = "none"
max_rounds = 3
max_review_attempts = 2
timeout = "20m"
rapid_terminal_window = "2m"
rapid_terminal_limit = 3
missing_work_max_items = 20
missing_work_item_max_bytes = 512
reason_max_bytes = 2048
review_text_max_bytes = 12000
next_round_guidance_max_bytes = 4096
failure_policy = "block_task"
```

Task create/update APIs may override policy within these validated bounds. Web and docs must show that review is opt-in unless config changes the default.

## Reviewer Routing

Reviewer routing chooses who will perform the review. It does not decide the verdict.

```go
type ReviewerSelector struct {
	AgentName           string   `json:"agent_name,omitempty"`
	Provider            string   `json:"provider,omitempty"`
	Model               string   `json:"model,omitempty"`
	PeerID              string   `json:"peer_id,omitempty"`
	ChannelID           string   `json:"channel_id,omitempty"`
	CapabilityIDs       []string `json:"capability_ids,omitempty"`
	RequiredSkill       string   `json:"required_skill,omitempty"`
	AllowedAgentNames   []string `json:"allowed_agent_names,omitempty"`
	PreferredAgentNames []string `json:"preferred_agent_names,omitempty"`
	AllowCoordinator    bool     `json:"allow_coordinator,omitempty"`
	AllowOriginalWorker bool     `json:"allow_original_worker,omitempty"`
}
```

Selection order:

1. If `agent_name` is set, choose that local reviewer agent with the configured provider/model if it is available and policy-eligible.
2. Else, if `peer_id` is set, choose that peer/session if it is available and policy-eligible.
3. Else, if `channel_id` is set, choose an eligible member of the channel matching allowed/preferred agents and capability filters.
4. Else, choose an eligible local reviewer provider/session using allowed/preferred agents, capability filters, and default runtime policy.
5. If no reviewer is available, record `error` review attempt and apply retry/circuit policy.

Eligibility rules:

- Default is `allow_original_worker = false`; the same session that terminalized the run should not review itself unless explicitly allowed.
- Original-worker exclusion compares the candidate reviewer session id, agent name, peer id, and actor identity against named source columns on the reviewed run: `session_id`, `claimed_by`, `claimed_agent_name`, `claimed_peer_id`, `terminalized_by_session_id`, `terminalized_by_agent_name`, `terminalized_by_peer_id`, `terminalized_by_actor_kind`, and `terminalized_by_actor_ref`. If `allow_original_worker = false` and the original worker identity cannot be determined, routing fails closed with a deterministic eligibility error.
- Default is `allow_coordinator = true`, but coordinator review must still call the same `submit_run_review` tool/API path and use a reviewer identity.
- Agent/provider/model hints control reviewer session shape only. They do not bypass reviewer binding, channel membership, tool policy, or authorization.
- Capability filters are advisory eligibility filters, not authority. Actual verdict authority is still `RecordRunReview`.
- Review channel messages may include `review_request`, `blocker`, `handoff`, and `result` kinds. They must not be parsed as the verdict.

### Reviewer Session Binding

Reviewer session bootstrap has an explicit task-service binding step:

1. Coordinator runtime calls `RouteRunReview` to select a reviewer and persist route metadata.
2. Daemon composition root starts or reuses the reviewer session according to the routed agent/provider/model/peer/channel selection.
3. Before the reviewer receives `agh-task-reviewer` or `submit_run_review`, daemon calls `BindRunReviewSession(review_id, session_id)`.
4. `BindRunReviewSession` runs in a task-service transaction, verifies the review is still `routed` or `requested`, sets `reviewer_session_id`, `started_at`, `deadline_at`, and `status = "in_review"`, and rejects any second active session binding.
5. Native tool registration and invocation call `LookupReviewForSession(session_id)` and require a matching active review binding.
6. Session end, timeout, or reviewer crash records an `error` or `timeout` attempt through task-service review paths; it does not delete the binding silently.

The binding is authorization state, not prompt context. A session that only knows a `review_id` cannot submit a verdict unless the task-service lookup binds that session to the review.

## Bundled Skill: `agh-task-reviewer`

Create a new bundled skill under `internal/skills/bundled`:

```yaml
---
name: agh-task-reviewer
description: Review an AGH task run and submit a typed persisted verdict through the native review tool.
metadata:
  agh:
    version: 1
    kind: orchestration
    requires_active_task_claim: false
    requires_review_request: true
    authority: instructional_only
---
```

The skill must instruct reviewers to:

- Read the task objective, latest run summary, terminal result/error, relevant task events, and prior review history.
- Use coordination channels for questions or clarifications when useful.
- Never treat channel messages as persisted review verdicts.
- Decide whether the terminal run satisfies the objective and accepted constraints.
- Submit exactly one typed verdict through `submit_run_review`.
- Include bounded `missing_work` and `next_round_guidance` on rejected verdicts.
- Use `blocked`, `error`, `timeout`, or `invalid_output` honestly instead of approving uncertain work.
- Avoid leaking raw claim tokens or sensitive session internals in review text.

`agh-orchestrator` should also gain guidance for when to request/route reviews, but the reviewer's operational guidance belongs in `agh-task-reviewer`.

## Data Model

Schema changes must ship as numbered migrations in `internal/store/globaldb`; boot-time schema reconciliation is forbidden.

Side-table-vs-JSON decision: review requests, reviewer routing, verdict outcomes, review-request recovery, retry attempts, continuation lineage, and circuit state use typed columns or side tables because they are queryable runtime state. `task_run_reviews.missing_work_json` and `task_runs.missing_work_json` are the only review JSON fields, and only because missing-work items are bounded ordered guidance rather than a SQL matching dimension.

### Task Policy Columns

Add task-level policy and rollup fields to `tasks`:

| Column | Type | Purpose |
| --- | --- | --- |
| `review_policy` | TEXT NOT NULL DEFAULT `'none'` | `none`, `on_success`, `on_failure`, `always`. |
| `review_max_rounds` | INTEGER NOT NULL DEFAULT `3` | Per-task continuation cap. |
| `review_round` | INTEGER NOT NULL DEFAULT `0` | Number of rejected review continuations already created. |
| `last_review_id` | TEXT NULL | Latest review row for list/detail projections. |
| `last_review_outcome` | TEXT NULL | Latest verdict outcome. |
| `review_circuit_opened_at` | TIMESTAMP NULL | Circuit-open timestamp. |
| `review_circuit_reason` | TEXT NULL | Bounded circuit reason. |

Rationale: policy and current review rollup are queryable task state. Long evidence and missing-work items remain in review tables.

Review agent/provider/model selectors live in `task_execution_profiles` and profile selector side tables owned by the orchestration child spec. Review rows copy the selected reviewer route for auditability after routing, but the task profile remains the source of default reviewer-selection policy.

### `task_run_reviews`

Create a task-owned review table:

| Column | Type | Purpose |
| --- | --- | --- |
| `review_id` | TEXT PRIMARY KEY | Stable review id. |
| `task_id` | TEXT NOT NULL | Reviewed task. |
| `run_id` | TEXT NOT NULL | Reviewed terminal run. |
| `parent_review_id` | TEXT NULL | Prior rejected review if this is a continuation chain. |
| `policy` | TEXT NOT NULL | Effective policy captured from the terminal run's review snapshot. |
| `review_round` | INTEGER NOT NULL | 1-based review round for the task. |
| `attempt` | INTEGER NOT NULL | Reviewer attempt number for this run/round. |
| `status` | TEXT NOT NULL | `requested`, `routed`, `in_review`, `recorded`, `circuit_opened`, `canceled`. |
| `outcome` | TEXT NULL | `approved`, `rejected`, `blocked`, `error`, `timeout`, `invalid_output`. |
| `confidence` | REAL NULL | 0..1, only for recorded verdicts. |
| `reason` | TEXT NULL | Bounded reason. |
| `delivery_id` | TEXT NULL | Required idempotency key for submitted verdicts. |
| `missing_work_json` | TEXT NOT NULL DEFAULT `'[]'` | Bounded opaque array; not used for SQL matching. |
| `next_round_guidance` | TEXT NULL | Bounded guidance for continuation. |
| `review_text` | TEXT NULL | Bounded reviewer evidence/prose. |
| `reviewer_session_id` | TEXT NULL | Reviewer session when local. |
| `reviewer_agent_name` | TEXT NULL | Reviewer agent selected from the effective review profile. |
| `reviewer_provider` | TEXT NULL | Reviewer provider selected from the effective review profile. |
| `reviewer_model` | TEXT NULL | Reviewer model selected from the effective review profile. |
| `reviewer_peer_id` | TEXT NULL | Reviewer peer when routed through network/channel. |
| `reviewer_channel_id` | TEXT NULL | Coordination channel used for routing. |
| `reviewed_by_kind` | TEXT NULL | `session`, `peer`, `operator`, `system`. |
| `reviewed_by_ref` | TEXT NULL | Actor reference. |
| `requested_by` | TEXT NOT NULL | Actor that requested review. |
| `requested_at` | TIMESTAMP NOT NULL | Request time. |
| `routed_at` | TIMESTAMP NULL | Reviewer selected time. |
| `started_at` | TIMESTAMP NULL | Reviewer session started time. |
| `reviewed_at` | TIMESTAMP NULL | Verdict persistence time. |
| `deadline_at` | TIMESTAMP NULL | Review timeout deadline. |
| `created_at` | TIMESTAMP NOT NULL | Row creation time. |
| `updated_at` | TIMESTAMP NOT NULL | Row update time. |

Indexes:

- `(task_id, review_round, attempt)`
- unique `(run_id, review_round, attempt)`
- `(run_id, status)`
- `(status, deadline_at)`
- `(reviewer_session_id, status)`
- `(reviewer_agent_name, status)`
- `(reviewer_peer_id, status)`
- `(reviewer_channel_id, status)`
- unique `(reviewer_session_id)` where `reviewer_session_id IS NOT NULL AND status IN ('routed', 'in_review')`
- unique `(review_id, delivery_id)` where `delivery_id IS NOT NULL`

### `task_runs` Review Trigger and Continuation Columns

The review-gate migration owns all review trigger and continuation-source fields on `task_runs`. It must create `task_run_reviews` and these `task_runs` columns/FKs in the same numbered migration; the orchestration child must not add the `task_run_reviews` FKs first.

Add review trigger and continuation-source fields to `task_runs`:

| Column | Type | Purpose |
| --- | --- | --- |
| `review_required` | INTEGER NOT NULL DEFAULT `0` | Terminal run requires a follow-up review request. |
| `review_request_round` | INTEGER NOT NULL DEFAULT `0` | Review round to request after terminalization. |
| `review_policy_snapshot` | TEXT NOT NULL DEFAULT `''` | Effective policy captured when the run terminalized. |
| `review_request_id` | TEXT NULL REFERENCES `task_run_reviews(review_id)` | Review request row created for this terminal run. Written and `review_required` cleared in the same follow-up transaction. |
| `parent_run_id` | TEXT NULL REFERENCES `task_runs(id)` | Reviewed terminal run that caused this continuation. Null for normal runs. |
| `review_id` | TEXT NULL REFERENCES `task_run_reviews(review_id)` | Rejected review that created this continuation. Null for normal runs. |
| `review_round` | INTEGER NOT NULL DEFAULT `0` | Review round represented by this run. Zero means not a review continuation. |
| `continuation_reason` | TEXT NOT NULL DEFAULT `''` | Empty for normal runs; `review_rejected` for review-driven continuations. |
| `missing_work_json` | TEXT NOT NULL DEFAULT `'[]'` | Bounded canonical JSON array copied from the rejected verdict for context-bundle replay. |
| `next_round_guidance` | TEXT NOT NULL DEFAULT `''` | Bounded guidance copied from the rejected verdict. |

Indexes:

- `(parent_run_id)`
- `(review_request_id)` where `review_request_id IS NOT NULL`
- `(review_id)` unique where `review_id IS NOT NULL`
- `(task_id, review_round)` where `review_round > 0`
- `(review_required, review_request_round, task_id)` where `review_required = 1`

Rationale: continuation runs must be queryable without parsing `metadata_json`, and idempotent rejected-verdict replay must be able to find the continuation by `review_id`. The review row remains the verdict authority; the run columns are the durable execution-context snapshot for the next worker.

Original-worker exclusion reads source columns from the orchestration child run provenance fields: `session_id`, `claimed_by`, `claimed_agent_name`, `claimed_peer_id`, `terminalized_by_session_id`, `terminalized_by_agent_name`, `terminalized_by_peer_id`, `terminalized_by_actor_kind`, and `terminalized_by_actor_ref`. The review-gate implementation must not infer original-worker identity from channel transcripts.

### Review Events

Task events must include review lifecycle rows with durable `event_seq`:

- `task.run_review_requested`
- `task.run_review_routed`
- `task.run_review_recorded`
- `task.run_review_approved`
- `task.run_review_rejected`
- `task.run_review_blocked`
- `task.run_review_retry_enqueued`
- `task.run_review_circuit_opened`
- `task.run_review_canceled`

Event payloads must be bounded and must not include raw claim tokens or full transcripts.

## Core Interfaces

These are target service contracts. Implementations may embed them into existing `task.Manager` / `task.Service`; do not introduce a second review service that owns task state outside `internal/task`.

```go
type ReviewGateManager interface {
	RequestRunReview(ctx context.Context, req RunReviewRequest, actor ActorContext) (*RunReview, error)
	RouteRunReview(ctx context.Context, req RouteRunReviewRequest, actor ActorContext) (*RunReview, error)
	BindRunReviewSession(ctx context.Context, req BindRunReviewSessionRequest, actor ActorContext) (*RunReviewBinding, error)
	LookupReviewForSession(ctx context.Context, sessionID string) (*RunReviewBinding, error)
	RecordRunReview(ctx context.Context, req RecordRunReviewRequest, actor ActorContext) (*RunReviewResult, error)
	CancelRunReview(ctx context.Context, req CancelRunReviewRequest, actor ActorContext) (*RunReview, error)
	ListRunReviews(ctx context.Context, query RunReviewQuery) ([]RunReview, error)
	GetReviewContinuation(ctx context.Context, taskID string) (*ReviewContinuation, error)
}

type RunReviewOutcome string

const (
	RunReviewOutcomeApproved      RunReviewOutcome = "approved"
	RunReviewOutcomeRejected      RunReviewOutcome = "rejected"
	RunReviewOutcomeBlocked       RunReviewOutcome = "blocked"
	RunReviewOutcomeError         RunReviewOutcome = "error"
	RunReviewOutcomeTimeout       RunReviewOutcome = "timeout"
	RunReviewOutcomeInvalidOutput RunReviewOutcome = "invalid_output"
)

type RunReviewVerdict struct {
	Outcome           RunReviewOutcome `json:"outcome"`
	Confidence        float64          `json:"confidence"`
	Reason            string           `json:"reason"`
	MissingWork       []string         `json:"missing_work"`
	NextRoundGuidance string           `json:"next_round_guidance"`
	ReviewText        string           `json:"review_text,omitempty"`
	ReviewedBy        ActorIdentity    `json:"reviewed_by"`
	ReviewedAt        time.Time        `json:"reviewed_at"`
	DeliveryID        string           `json:"delivery_id"`
}

type RunReviewRequest struct {
	TaskID      string    `json:"task_id"`
	RunID       string    `json:"run_id"`
	ReviewRound int       `json:"review_round"`
	Policy      string    `json:"policy"`
	Reason      string    `json:"reason"`
	Now         time.Time `json:"now"`
}

type RouteRunReviewRequest struct {
	ReviewID string           `json:"review_id"`
	Selector ReviewerSelector `json:"selector"`
	Now      time.Time        `json:"now"`
}

type BindRunReviewSessionRequest struct {
	ReviewID  string    `json:"review_id"`
	SessionID string    `json:"session_id"`
	Now       time.Time `json:"now"`
}

type RunReviewBinding struct {
	ReviewID  string    `json:"review_id"`
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	SessionID string    `json:"session_id"`
	Deadline  time.Time `json:"deadline_at"`
}

type RecordRunReviewRequest struct {
	ReviewID string           `json:"review_id"`
	RunID    string           `json:"run_id"`
	Verdict  RunReviewVerdict `json:"verdict"`
	Now      time.Time        `json:"now"`
}

type RunReviewResult struct {
	Review          RunReview `json:"review"`
	ContinuationRun *Run      `json:"continuation_run,omitempty"`
	CircuitOpened   bool      `json:"circuit_opened,omitempty"`
}

type ReviewContinuation struct {
	ReviewID          string   `json:"review_id"`
	ReviewedRunID     string   `json:"reviewed_run_id"`
	ReviewRound       int      `json:"review_round"`
	Outcome           string   `json:"outcome"`
	Reason            string   `json:"reason"`
	MissingWork       []string `json:"missing_work"`
	NextRoundGuidance string   `json:"next_round_guidance"`
}
```

Idempotency:

- `delivery_id` is required for every `RecordRunReview` request and every `submit_run_review` tool call.
- `RecordRunReview` is idempotent only when `review_id`, `run_id`, actor identity, outcome, and `delivery_id` match the already persisted verdict.
- Conflicting replay returns a typed conflict error.
- Review-request creation is idempotent through unique `(run_id, review_round, attempt)` with `attempt = 1` for the initial request. Duplicate terminal/recovery triggers return the existing attempt-1 row, set `task_runs.review_request_id` if needed, clear `review_required`, and do not create another review row.
- Missing/empty `reason` is invalid for every outcome.
- `confidence` must be in `[0, 1]`.
- `approved` should normally have empty `missing_work`; non-empty missing work with `approved` is invalid.
- `rejected` must include at least one `missing_work` item or non-empty `next_round_guidance`.

Atomicity:

- `RecordRunReview` runs inside one `BEGIN IMMEDIATE` transaction.
- The transaction validates the review binding, deadline, actor identity, delivery id, current review status, round limits, field bounds, and idempotency.
- For `approved`, `blocked`, `error`, `timeout`, and `invalid_output`, the transaction writes the verdict, task review rollup fields, and task events together.
- For `rejected` while rounds remain, the transaction writes the verdict, increments `tasks.review_round`, updates task review rollup fields, creates exactly one continuation `task_runs` row with the typed continuation columns, and writes `task.run_review_recorded`, `task.run_review_rejected`, and `task.run_review_retry_enqueued` events together.
- If the same rejected verdict is replayed with the same `delivery_id`, `RecordRunReview` returns the existing continuation by querying `task_runs.review_id = review_id`; it does not enqueue a second run.
- If any write in the verdict-plus-continuation transaction fails, no verdict, rollup, continuation run, or review event is persisted.

Reviewer attempt retry model:

- Each reviewer retry inserts a new `task_run_reviews` row with the same `task_id`, `run_id`, `review_round`, and `parent_review_id`, and with `attempt = previous_attempt + 1`.
- The previous attempt must be terminal before the next attempt row is inserted. `error`, `timeout`, and `invalid_output` attempts persist as `status = "recorded"` with the corresponding `outcome`.
- Only one attempt for a `(run_id, review_round)` may be in `requested`, `routed`, or `in_review` at a time.
- Unique `(run_id, review_round, attempt)` makes retry replay idempotent without preventing bounded retries.

## Native Tool Contract

Add a reviewer-scoped native tool:

```json
{
  "name": "submit_run_review",
  "description": "Submit the typed persisted verdict for an AGH task-run review request.",
  "input_schema": {
    "type": "object",
    "required": ["review_id", "run_id", "outcome", "confidence", "reason", "missing_work", "next_round_guidance", "delivery_id"],
    "additionalProperties": false,
    "properties": {
      "review_id": { "type": "string" },
      "run_id": { "type": "string" },
      "outcome": { "type": "string", "enum": ["approved", "rejected", "blocked", "error", "timeout", "invalid_output"] },
      "confidence": { "type": "number", "minimum": 0, "maximum": 1 },
      "reason": { "type": "string" },
      "missing_work": { "type": "array", "items": { "type": "string" } },
      "next_round_guidance": { "type": "string" },
      "review_text": { "type": "string" },
      "delivery_id": { "type": "string" }
    }
  }
}
```

Tool authorization:

- Available only to reviewer sessions with an active binding returned by `LookupReviewForSession`. Coordinator sessions may use the tool only when they were explicitly routed and bound as the reviewer for that review.
- Does not require an active task claim.
- Does not expose claim token or worker lease state.
- Calls `task.Service.RecordRunReview`.
- The native tool registration must check `metadata.agh.requires_review_request = true` from bundled skill/frontmatter metadata and hide the tool from sessions without an active review binding.
- Operator/API/UDS/CLI verdict submission uses the explicit review verdict endpoint/command with server-derived operator actor authorization; it must not expose the native tool to unbound sessions. Debug-only native-tool bypass is out of MVP.

## Task Context Bundle Contract

Extend the orchestration child spec's `TaskContextBundle` with optional review continuation state:

```go
type TaskContextBundle struct {
	// existing fields from _techspec_orchestration.md
	ReviewContinuation *ReviewContinuation `json:"review_continuation,omitempty"`
	ReviewHistory      []RunReviewSummary  `json:"review_history,omitempty"`
}

type RunReviewSummary struct {
	ReviewID      string `json:"review_id"`
	RunID         string `json:"run_id"`
	ReviewRound   int    `json:"review_round"`
	Attempt       int    `json:"attempt"`
	Status        string `json:"status"`
	Outcome       string `json:"outcome,omitempty"`
	Reason        string `json:"reason,omitempty"`
	ReviewedAt    string `json:"reviewed_at,omitempty"`
	ReviewerLabel string `json:"reviewer_label,omitempty"`
}
```

Projection rules:

- The latest rejected review that produced the current continuation run appears as `ReviewContinuation`.
- For continuation runs, `ReviewContinuation` is read from the `task_runs.review_id` link and the typed continuation columns first, then cross-checked against `task_run_reviews` for audit fields.
- `ReviewHistory` is bounded by config and redacts long `review_text`.
- Context bundle never includes raw claim tokens, full channel transcripts, or full reviewer session transcript.
- Workers must treat review guidance as task context, not as permission to bypass claim/lease rules.

## API, UDS, and CLI

HTTP endpoints:

- `POST /api/task-runs/:run_id/reviews`
- `GET /api/task-runs/:run_id/reviews`
- `GET /api/tasks/:task_id/reviews`
- `GET /api/task-reviews/:review_id`
- `POST /api/task-reviews/:review_id/route`
- `POST /api/task-reviews/:review_id/verdict`
- `POST /api/task-reviews/:review_id/cancel`
- `GET /api/tasks/:task_id/review-policy`
- `PATCH /api/tasks/:task_id/review-policy`
- `GET /api/tasks/:task_id/execution-profile`
- `PATCH /api/tasks/:task_id/execution-profile`

`execution-profile` endpoints are defined by the orchestration child spec. Review-gate handlers use them for `ReviewProfile` read/update rather than creating a second review-profile endpoint. UDS must mount the same core handlers and DTOs. No transport-specific review semantics.

CLI commands:

```bash
agh task review request <run-id> --reason <text> --json
agh task review list --task <task-id> --json
agh task review list --run <run-id> --json
agh task review show <review-id> --json
agh task review submit <review-id> --run <run-id> --outcome rejected --missing-work <item> --next-round-guidance <text> --json
agh task review cancel <review-id> --reason <text> --json
agh task review policy get <task-id> --json
agh task review policy set <task-id> --policy on_success --max-rounds 3 --json
agh task profile get <task-id> --json
agh task profile set <task-id> --review-agent <agent> --review-channel <channel-id> --json
```

All surfaces must be deterministic, JSON-capable, and redacted.

## Hooks and Observe Events

Add hook events:

| Hook | Emitted By | Payload Notes |
| --- | --- | --- |
| `task.run_review_requested` | `task.Service` | `task_id`, `run_id`, `review_id`, `policy`, `round`, bounded reason |
| `task.run_review_routed` | coordinator/task service | reviewer identifiers, channel id, no transcript |
| `task.run_review_recorded` | `task.Service` | verdict summary |
| `task.run_review_approved` | `task.Service` | accepted terminal outcome |
| `task.run_review_rejected` | `task.Service` | missing-work count and bounded reason |
| `task.run_review_blocked` | `task.Service` | blocker reason |
| `task.run_review_retry_enqueued` | `task.Service` | continuation run id and parent run id |
| `task.run_review_circuit_opened` | `task.Service` | circuit reason and policy limit |
| `task.run_review_canceled` | `task.Service` | cancel reason |

Observe/SSE must expose these as task events after persistence. Hook dispatch must treat review events like other task hooks: event delivery is observable, but hook success/failure is not review authority.

Coordinator wake contract:

```go
type ReviewRouter interface {
	OnRunReviewRequested(ctx context.Context, event RunReviewRequestedEvent) error
}

type RunReviewRequestedEvent struct {
	TaskID   string `json:"task_id"`
	RunID    string `json:"run_id"`
	ReviewID string `json:"review_id"`
	Round    int    `json:"round"`
}
```

`internal/daemon` wires this callback at composition root. `task.Service` invokes it from the review-request call site after the review row and task event are durable. The callback is a wake-up signal only; the coordinator must read the persisted review row through task-service APIs before routing. Implementations must not tail `task_events`, poll `task_run_reviews`, or parse channel messages as the primary wake mechanism. Daemon startup recovery may ask `task.Service` for still-requested reviews through a bounded recovery query, but that is reconciliation of durable task-owned state, not an event-tail loop.

## Failure Policy

Review failures must be conservative. A reviewer failure cannot approve work.

Rules:

- Timeout creates a `timeout` outcome attempt.
- Reviewer session crash creates an `error` outcome attempt.
- Invalid tool payload creates an `invalid_output` outcome attempt.
- Attempts retry until `max_review_attempts`.
- After exhausted attempts, apply `failure_policy`.
- Default `failure_policy = "block_task"` opens a review circuit and blocks the task with a bounded reason.
- `failure_policy = "fail_task"` may fail the task only through task-service-owned terminal mutation and only if no active run exists.
- Rapid terminalization after repeated rejected continuations opens the review circuit when `rapid_terminal_limit` is exceeded within `rapid_terminal_window`.
- Circuit reset is explicit through API/UDS/CLI/task-service path and emits a task event.

## Security and Privacy

- Reviewer packets must not include raw claim tokens.
- Reviewer packets must not include full secret-bearing logs or unredacted provider credentials.
- Reviewer identity must be persisted as structured actor metadata.
- Review APIs must enforce operator/session/peer authorization.
- Reviewer native-tool access must require `LookupReviewForSession(session_id)`; review id alone is never sufficient. Operator submission, when allowed, goes through the HTTP/UDS/CLI verdict surface with server-derived operator identity, never through an unbound native-tool context.
- Channel-routed reviewers must be verified against channel membership and capability filters before session start.
- Profile-routed reviewer agent/provider/model selections must pass the same task profile config gates, provider authorization, and reviewer binding checks as default reviewers.
- Review text and guidance are bounded and sanitized for logs, SSE, hooks, web, and docs examples.
- A malicious reviewer can submit a bad verdict only if authorized as reviewer; audit history must make reviewer identity and route visible.

## Web and Docs Impact

Web impact:

- Generated TypeScript contract changes for review policy, review rows, verdicts, and review events.
- Task detail surfaces should show latest review state, current circuit status, and continuation guidance when present.
- Task profile surfaces should show review selector fields only when backend contracts expose them; web must not invent unsupported reviewer controls.
- Task stream handling must accept review events through generated types.
- Web must not infer review verdicts from channel messages or transcript content.

Docs impact:

- `packages/site` task/orchestration docs must explain review gate policy, post-terminal semantics, bounded continuation rounds, and channel routing.
- Native tool docs must document `submit_run_review`.
- CLI reference must include `agh task review ...` commands after cobra JSON regeneration.
- Configuration docs must document `[task.orchestration.review]`.
- Task execution profile docs must document review agent/provider/model selectors and the channel-authority boundary.
- Skill docs must document `agh-task-reviewer` as instructional only.

## Implementation Steps

1. Add review config defaults, validation, docs examples, and redacted config inspection.
2. Add numbered migrations for task review policy fields, `task_run_reviews`, and all `task_runs` review trigger/continuation columns in one migration so FKs are ordered.
3. Extend task store interfaces with review CRUD, indexes, and fresh/migrated DB tests.
4. Extend `task.Service` with `RequestRunReview`, `RouteRunReview`, `RecordRunReview`, circuit reset/open, and continuation-run creation.
5. Emit review task events and hooks from task-service-owned transitions.
6. Extend `TaskContextBundle` with `ReviewContinuation` and bounded review history.
7. Apply `TaskExecutionProfile.Review` to reviewer routing, including agent/provider/model hints and selector side tables.
8. Add reviewer routing in coordinator runtime using agent/peer/channel/capability selectors.
9. Add bundled `agh-task-reviewer`, update `agh-orchestrator` guidance, and implement bundled-skill loader support for `metadata.agh.requires_review_request`.
10. Add native `submit_run_review` tool, reviewer-session binding, `LookupReviewForSession`, and reviewer-session authorization.
11. Add HTTP/UDS/CLI surfaces and generated contract changes.
12. Add web generated type consumption and minimal review read-model rendering.
13. Add docs for policy, lifecycle, channels boundary, review profiles, skill behavior, and CLI/tool/API usage.
14. Run focused review-gate tests, aggregate scenario QA, and `make verify`.

## Test Strategy

Unit tests:

- Policy matching for `none`, `on_success`, `on_failure`, and `always`.
- Verdict validation and bounded fields.
- Idempotent `RecordRunReview` replay and conflicting replay rejection.
- Reviewer selector eligibility and original-worker exclusion.
- Review profile agent/provider/model selection and provider override config gates.
- Rapid terminalization circuit logic.

Store/migration tests:

- Fresh DB contains review fields/tables/indexes, including unique `(run_id, review_round, attempt)`.
- Migrated DB preserves existing task/run behavior.
- Review query indexes support task, run, reviewer, status, and deadline lookups.
- Continuation-run columns exist on `task_runs`, are indexed by `review_id`, and enforce one continuation run per rejected review.
- Review-request columns exist on `task_runs`, are indexed by `review_request_id`, and clear `review_required` when the attempt-1 review row is durable.
- Review profile selector side tables support reviewer agent/channel/capability lookup.
- JSON `missing_work_json` is bounded and canonicalized.

Task-service integration tests:

- Completed run with `on_success` creates review request.
- Failed run with `on_success` does not create review request.
- Failed run with `on_failure` creates review request.
- Approved review accepts terminal outcome without rewriting run status.
- Rejected review enqueues continuation run with parent/review metadata.
- Rejected review stores continuation metadata on `task_runs` and injects the same guidance into `TaskContextBundle.ReviewContinuation`.
- Replayed rejected verdict with the same `delivery_id` returns the existing continuation run instead of enqueueing a duplicate.
- Multi-round review chains preserve `parent_review_id` lineage and increase `tasks.review_round` monotonically.
- Max rounds opens circuit and does not enqueue another run.
- Timeout/error/invalid output retry and then apply `failure_policy`.
- Blocked verdict blocks task through task-service-owned state.
- Follow-up review-request creation after terminal run is idempotent on `(run_id, review_round)` and recovery-safe after a crash between terminal commit and review-request creation.
- Reviewer retry failures create new attempt rows with monotonically increasing `attempt` after the prior attempt is terminal.

Boundary tests:

- Channel message cannot record verdict.
- Notification cursor cannot advance or mutate review state.
- Scheduler cannot call review verdict persistence directly.
- Coordinator routing cannot approve/reject without `submit_run_review`.
- Reviewer session without a persisted `LookupReviewForSession` binding cannot see or call `submit_run_review`.
- Operator/debug contexts cannot call the native `submit_run_review` tool without a reviewer-session binding; operator verdict submission, if used, must go through the explicit API/UDS/CLI path.
- `allow_original_worker = false` rejects candidate reviewer sessions, agents, or peers that match the reviewed run's original worker identity.
- Web/client DTOs cannot provide raw claim token.

E2E/scenario QA:

- Coordinator spawns worker.
- Worker completes with insufficient work.
- Review request routes to reviewer through a coordination channel.
- Review request routes to the configured reviewer agent/provider/model when a task review profile sets one.
- Reviewer rejects with missing work and guidance.
- Continuation worker receives `ReviewContinuation` in context bundle.
- Worker fixes issue and completes.
- Reviewer approves.
- Bridge terminal notifier delivers final notification after durable task event replay.

## Risks

- Review gate could be mistaken for pre-execution manual approval. Mitigation: separate names, state, docs, and APIs from existing `ApprovalPolicy`.
- Reviewer routing through channels could drift into channel authority. Mitigation: native tool/API persistence is the only verdict path; boundary tests enforce this.
- Review profile routing could be mistaken for review authority. Mitigation: `TaskExecutionProfile.Review` only selects reviewer execution shape; `RecordRunReview` remains the verdict path.
- Rejected reviews could create runaway loops. Mitigation: `max_rounds`, rapid-terminal circuit, reviewer attempt limits, and explicit circuit reset.
- Review tables could become transcript stores. Mitigation: bounded evidence only; full transcripts remain session/channel history and are not authority.
- Adding review state before orchestration hardening could create duplicate context contracts. Mitigation: implement orchestration child first, then review-gate context fields.

## Architecture Decision Records

- [ADR-007: Review Gate Is a Post-Terminal Continuation Loop](adrs/adr-007-review-gate-post-terminal-continuation-loop.md)
- [ADR-008: Reviewer Routing Uses Channels Without Channel Authority](adrs/adr-008-review-routing-uses-channels-without-channel-authority.md)
- [ADR-009: Review Verdicts and Continuation Guidance Are Typed Task State](adrs/adr-009-review-verdicts-and-continuation-guidance-are-typed-task-state.md)
- [ADR-010: Task Execution Profiles Are Typed Task-Owned Overlays](adrs/adr-010-task-execution-profiles-are-typed-overlays.md)
