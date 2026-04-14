---
status: resolved
file: internal/api/core/network_details.go
line: 93
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg3t,comment:PRRC_kwDOR5y4QM63ZMHX
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Read-back failure leaves a created channel behind.**

By this point all sessions may already exist. Returning 500 here without rollback means the client sees a failure, retries, and can create duplicate sessions in the same channel. Either roll back `createdIDs` on this path or return a minimal success payload from the data you already have.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 89 - 93, On error from
h.networkChannelDetailPayload (the detailErr path) ensure you don't leave
created sessions behind: either perform a rollback by deleting the newly created
session IDs (use the same service/repo method your create path uses — e.g., call
service.DeleteSessions(ctx, createdIDs) or repo.DeleteMany(createdIDs), logging
any deletion errors) before returning the error, or instead return a minimal
success payload built from the data you already have (channel, service and
createdIDs) rather than calling h.respondError; update the code at the
h.networkChannelDetailPayload error branch to implement one of these two
behaviors and keep any deletion/logging calls idempotent and non-blocking for
the client response.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `CreateNetworkChannel` rolls back partially created sessions when `Sessions.Create` fails, but not when the final read-back through `networkChannelDetailPayload` fails after all sessions were already created.
- Fix approach: roll back `createdIDs` on the detail read-back failure path before returning the error.
