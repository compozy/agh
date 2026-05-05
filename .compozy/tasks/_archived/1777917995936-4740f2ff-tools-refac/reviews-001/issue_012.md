---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/udsapi/hosted_mcp.go
line: 82
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJu,comment:PRRC_kwDOR5y4QM680KIK
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Return HTTP errors before upgrading this endpoint to SSE.**

`PrepareSSE` sends the `200 text/event-stream` response before the first `Projection` call. That means missing/invalid `bind_id`, peer, nonce, or auth failures now show up as an `"error"` event on a successful stream instead of the mapped `400`/`403`/`503` response. Clients lose the ability to distinguish setup failures from a live stream that later errored.




Also applies to: 90-97

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/hosted_mcp.go` around lines 76 - 82, Validate request
parameters and perform all auth/peer/nonce checks and call core.RespondError (or
other HTTP error responses) before calling core.PrepareSSE so the HTTP status
code and body are sent on failure instead of being emitted as SSE events;
specifically, move or duplicate the validation logic that inspects bindID (from
c.Query("bind_id")), lastDigest, peer/nonce/auth checks and any calls that can
return 400/403/503 so they run and return on error prior to invoking
core.PrepareSSE (and do the same for the other block around the Projection call
at the later section), then only call core.PrepareSSE and proceed to
writer/Projection when all validations pass.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `streamHostedMCPProjection` calls `PrepareSSE` before the first hosted-MCP `Projection` check. Missing bind IDs, invalid peers, nonce/auth problems, or disabled service errors therefore become successful SSE streams with an error event instead of HTTP 400/403/503 setup failures. Validate and fetch the first projection before upgrading to SSE, then use SSE errors only for failures after the stream is established.
