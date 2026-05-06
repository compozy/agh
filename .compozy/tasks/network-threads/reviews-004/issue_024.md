---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/hooks/payloads.go
line: 587
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0DsN,comment:PRRC_kwDOR5y4QM6-RRZT
---

# Issue 024: _⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _🏗️ Heavy lift_

**Add the canonical correlation fields to `NetworkPayload` before this hook surface settles.**

These new network hook payloads only expose `session_id` plus envelope metadata, so downstream consumers lose the workspace/agent/workflow/task correlation keys that the rest of the hook surface carries. Embedding `SessionContext` or a dedicated canonical correlation block here would keep the new network events joinable with the rest of the lifecycle stream.

 

As per coding guidelines, `internal/**/*.go`: "Every domain operation must emit a canonical event with correlation keys (`workspace_id`, `session_id`, `parent_session_id`, `root_session_id`, `agent_name`, `task_id`, `run_id`, `claim_token_hash`, `lease_until`, `workflow_id`, `coordinator_session_id`, `scheduler_reason`, `hook_event`, `hook_name`, `spawn_depth`, `actor_kind`, `actor_id`, `release_reason`)."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/hooks/payloads.go` around lines 547 - 587, NetworkPayload currently
only includes session/envelope metadata, so add the canonical correlation fields
required by our domain events to make network hooks joinable: extend the
NetworkPayload struct to include the correlation keys (workspace_id,
parent_session_id, root_session_id, agent_name, task_id, run_id,
claim_token_hash, lease_until, workflow_id, coordinator_session_id,
scheduler_reason, hook_event, hook_name, spawn_depth, actor_kind, actor_id,
release_reason) and any missing SessionContext fields (e.g., session_id already
present) so downstream consumers can correlate events; update the NetworkPayload
type definition to include these new JSON-tagged string/nullable fields (and
adjust NetworkObservationPatch if needed) ensuring field names match the
canonical names in our guideline.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Reasoning: `NetworkPayload` is intentionally the compact observation payload for committed conversation writes, not a full session-lifecycle context. Current tests explicitly verify that `SessionContextFromPayload(NetworkPayload)` is unsupported and that the payload excludes richer/raw message material. The conversation-store write path also does not carry the proposed canonical correlation block, so adding it here would be a broader API/contract redesign across network persistence, dispatch helpers, and consumers rather than a scoped regression fix in this batch.
- Resolution: leave this batch focused on correctness issues in the current payload contract. If richer correlation is desired, it should be tracked as a dedicated follow-up that redesigns the network hook payload surface end-to-end.
- Verification: analysis completed and full `make verify` stayed green without additional payload-contract changes.
