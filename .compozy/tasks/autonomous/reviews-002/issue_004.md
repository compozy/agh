---
status: resolved
file: internal/api/core/agent_channels.go
line: 103
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59q6tb,comment:PRRC_kwDOR5y4QM67Yhp_
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject malformed `wait`/`limit` values instead of defaulting them.**

`parseBoolQuery` and `parsePositiveIntQuery` silently turn bad input into `false` and `0`. For `/api/agent/channels/{channel}/recv`, that means `?wait=maybe` disables long-polling and `?limit=abc` removes the cap instead of returning the documented invalid-request response.



Also applies to: 796-816

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/agent_channels.go` around lines 91 - 103, The handler is
currently calling parseBoolQuery and parsePositiveIntQuery which silently coerce
invalid inputs to false/0; update the recv handler that calls agentChannelInbox
(and similarly the other handler range referenced) to validate query parsing and
reject malformed values: call parseBoolQuery and parsePositiveIntQuery (or their
underlying parsing logic) in a way that returns an error on invalid input, and
if parsing fails respond with h.respondError(c, http.StatusBadRequest,
ErrInvalidRequest) (or the project’s documented invalid-request response)
instead of proceeding to agentChannelInbox; reference the handler that invokes
agentChannelInbox and the functions parseBoolQuery/parsePositiveIntQuery to
locate and fix the logic.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `parseBoolQuery` and `parsePositiveIntQuery` silently coerce malformed values to default values. `AgentChannelRecv` therefore treats `?wait=maybe` as `wait=false` and `?limit=abc` as unlimited instead of rejecting the request. The fix is to make query parsing return validation errors and have the receive handler respond with the documented bad-request path.
- Resolution: Changed query parsing to return errors for malformed `wait`/`limit` values and respond with bad-request validation errors before service access; verified by receive validation tests and full `make verify`.
