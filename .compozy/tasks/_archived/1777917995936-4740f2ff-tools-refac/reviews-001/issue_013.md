---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/udsapi/hosted_mcp.go
line: 156
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJw,comment:PRRC_kwDOR5y4QM680KIN
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Release is missing the same peer check as the other hosted-MCP endpoints.**

`bind`, `projection`, and `call` all derive peer info from the UDS connection before touching the hosted service, but `release` accepts an arbitrary `bind_id` and tears it down directly. Because `ReleaseBind` only receives the id, another local client that learns a bind id can revoke someone else’s bind.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/hosted_mcp.go` around lines 145 - 156, releaseHostedMCP
currently calls HostedMCP.ReleaseBind(req.BindID) without validating the
requesting peer, allowing one local client to revoke another's bind; replicate
the peer-check logic used by the bind/projection/call handlers: extract the peer
identity from the UDS connection/context (same method those handlers use),
verify that the peer owns or is authorized for req.BindID (either call an
existing method that accepts peer info or query HostedMCP for the bind owner)
and only then call ReleaseBind (or a ReleaseBindWithPeer equivalent) and
respond; update releaseHostedMCP to perform this ownership check before
teardown.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `releaseHostedMCP` decodes a bind id and calls `HostedMCP.ReleaseBind` without deriving the UDS peer or checking bind ownership. The other hosted-MCP endpoints validate peer identity before touching bind state. The production fix needs a peer-checked release path on the hosted service, so this remediation will minimally touch `internal/mcp/hosted.go` outside the original file list to add and use `ReleaseBindForPeer`.
