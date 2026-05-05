# ADR-008: Reviewer Routing Uses Channels Without Channel Authority

## Status

Accepted

## Date

2026-05-05

## Context

AGH network channels already participate in task coordination. The orchestration hardening spec explicitly says channels carry conversation, handoff, blocker, and result content, but they do not define task ownership or terminal state. The review-gate proposal naturally wants to use channels for reviewer discovery and coordination, especially for review, ideation, task splitting, and handoff workflows.

However, if a channel message can become the review verdict, channel state becomes workflow authority. That would violate the orchestration authority model and make replay depend on channel/thread history instead of task-owned state.

## Decision

Reviewer routing may use task execution profile reviewer fields, channels, peers, agents, providers, models, and capability filters, but review authority remains in `task.Service`.

Task execution profiles may be used to:

- select a reviewer agent, provider, or model;
- constrain allowed/preferred reviewer agents;
- select a review channel or peer;
- apply capability filters.

Channels may be used to:

- announce `review_request` coordination messages;
- identify eligible reviewers by channel membership;
- let reviewers ask clarifying questions;
- let coordinators hand off review context;
- carry non-authoritative review discussion.

Channels must not be used to:

- infer `approved` / `rejected` / `blocked` verdicts;
- store the only copy of review evidence;
- own retry or continuation policy;
- determine task/run ownership;
- determine terminal state;
- lower or reset review circuit state.

A verdict becomes authoritative only when a reviewer calls the native/API/UDS/CLI review submission path and `task.Service.RecordRunReview` persists the typed verdict.

`TaskExecutionProfile.Review` is selection input only. It does not approve work, reject work, open circuits, create continuation runs, or bypass reviewer authorization.

Reviewer sessions must be bound to a persisted review request before the native `submit_run_review` tool is visible. The binding is stored on task-owned review state and looked up by session id. Coordinator sessions may submit a verdict only when they were explicitly routed and bound as the reviewer for that review; ordinary coordinator routing authority is not enough.

When `allow_original_worker = false`, routing validates candidate reviewer session, agent, peer, and actor identity against named reviewed-run provenance columns: `session_id`, `claimed_by`, `claimed_agent_name`, `claimed_peer_id`, `terminalized_by_session_id`, `terminalized_by_agent_name`, `terminalized_by_peer_id`, `terminalized_by_actor_kind`, and `terminalized_by_actor_ref`. It fails closed when identity cannot be determined.

## Consequences

- The coordinator can use channels naturally while preserving task-owned review state.
- Review can route to a peer, channel member, or local reviewer session without introducing channel-owned workflow semantics.
- Task-specific reviewer agents/providers/models become explicit and queryable without becoming authority.
- Boundary tests must prove channel messages cannot record review verdicts.
- Boundary tests must prove an unbound reviewer/coordinator session cannot submit a verdict and that `allow_original_worker = false` blocks self-review.
- Bridge-delivered review workflows remain out of MVP as a primary gate; they can be explored later only if they still persist verdicts through `task.Service`.

## Rejected Alternatives

### Parse verdicts from channel messages

Rejected because it makes channel transcript content a source of runtime authority.

### Create a review channel bus in `internal/notifications`

Rejected because `internal/notifications` is a durable cursor primitive only. It does not own review workflow state.

### Require reviewers to be local only

Rejected because channel/peer/capability routing is useful and aligns with AGH's network model. The boundary issue is authority, not routing.

### Treat review profile fields as verdict authority

Rejected because profile fields only choose who reviews. The persisted typed verdict path remains the authority.

## References

- [`../_techspec.md`](../_techspec.md)
- [`../_techspec_orchestration.md`](../_techspec_orchestration.md)
- [`../_techspec_review_gate.md`](../_techspec_review_gate.md)
- [`../analysis/analysis_task-execution-profile.md`](../analysis/analysis_task-execution-profile.md)
- [`adr-001-orchestration-hardening-extends-existing-autonomy.md`](adr-001-orchestration-hardening-extends-existing-autonomy.md)
- [`adr-010-task-execution-profiles-are-typed-overlays.md`](adr-010-task-execution-profiles-are-typed-overlays.md)
- [`adr-003-shared-durable-notification-cursors.md`](adr-003-shared-durable-notification-cursors.md)
