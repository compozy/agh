---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/api/core/bridges.go
line: 403
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Lrj,comment:PRRC_kwDOR5y4QM6-UJdl
---

# Issue 001: _⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_ | _⚡ Quick win_

**Reject foreign subscription IDs and bridge targets on create.**

This path authorizes only the task, then accepts arbitrary `subscription_id`/scope fields and persists through an upsert keyed on `subscription_id`. A caller who knows another subscription ID can overwrite that record onto a different task, and there is also no check that the chosen bridge instance belongs to the same workspace/scope as the task.

Generate IDs server-side for create, reject existing IDs that are already bound to another task, and verify the bridge instance’s scope/workspace matches the task before calling the store.
 


Also applies to: 661-694

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/bridges.go` around lines 352 - 403, The
CreateTaskBridgeNotificationSubscription handler currently trusts
client-supplied subscription IDs and bridge instance selection, allowing ID
collision/overwrite and cross-workspace binding; fix it by (1) generating a new
subscription ID server-side instead of accepting req.SubscriptionID (update
taskBridgeNotificationSubscriptionFromRequest to allow nil/empty incoming ID and
return/accept a generated ID), (2) before PutBridgeTaskSubscription, check that
the existing subscription (if any) with that ID does not belong to a different
task and reject the request if it does, and (3) validate that the bridge
instance returned by bridges.GetInstance belongs to the same workspace/scope as
the authorized task (use the task/actor or manager scope from
authorizeTaskBridgeNotification) and reject mismatched scope with an error;
apply the same changes to the equivalent create path referenced in the second
location.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `CreateTaskBridgeNotificationSubscription` still accepts client-controlled `subscription_id` and persists the normalized subscription through an upsert keyed by that ID.
  - The handler authorizes only task existence, then checks bridge existence without verifying the bridge instance scope/workspace against the authorized task.
  - Root cause: create-time ownership and scope invariants are enforced too late or not at all; the API trusts transport fields that must remain daemon-owned.
  - Planned fix: make create path generate daemon-owned subscription IDs, reject/ignore client-owned IDs on create, and validate the selected bridge instance against the authorized task scope/workspace before persistence.
  - Resolved: create now generates server-owned subscription IDs, persists authoritative task scope/workspace, validates the bridge instance against the authorized task, and regression tests cover the create/list/get/delete flow.
