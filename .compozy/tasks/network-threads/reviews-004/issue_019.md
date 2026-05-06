---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/hooks/dispatch_events.go
line: 136
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:221137dc1057
review_hash: 221137dc1057
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 019: Return zero correlation when the network actor id is unknown.
## Review Comment

If both `PeerFrom` and `SessionID` are blank, this branch still emits `ActorKind: "network_peer"` with an empty `ActorID`. That creates a partial canonical correlation record and differs from the fallback path below, which returns `DispatchCorrelation{}` when no identifier exists. Guard `actorID == ""` before building the struct.

As per coding guidelines, "Every domain operation must emit a canonical event with correlation keys (`workspace_id`, `session_id`, `parent_session_id`, `root_session_id`, `agent_name`, `task_id`, `run_id`, `claim_token_hash`, `lease_until`, `workflow_id`, `coordinator_session_id`, `scheduler_reason`, `hook_event`, `hook_name`, `spawn_depth`, `actor_kind`, `actor_id`, `release_reason`)."

## Triage

- Decision: `valid`
- Notes: `CorrelationFromPayload` always returns `DispatchCorrelation{ActorKind: "network_peer"}` for `NetworkPayload`, even when both `PeerFrom` and `SessionID` trim to empty strings. That emits a partial correlation record with an empty actor id. Guard the empty-id case and return the zero correlation payload instead.
