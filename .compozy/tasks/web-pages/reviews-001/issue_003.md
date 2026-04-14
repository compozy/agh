---
status: resolved
file: internal/api/core/network.go
line: 60
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg33,comment:PRRC_kwDOR5y4QM63ZMHj
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid a full session scan in `NetworkPeers`.**

This now does `h.Sessions.ListAll(...)` on every peers request, even if only a handful of peers were returned. That makes the endpoint O(all sessions), and it also turns a session-store blip into a 500 even though `service.ListPeers(...)` already succeeded. Prefer looking up only the returned `peer.SessionID` values and falling back to the plain peer payload if enrichment fails.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network.go` around lines 50 - 60, Replace the full scan
h.Sessions.ListAll(...) with a targeted lookup for only the session IDs present
in peers: collect unique peer.SessionID values from peers, call your
session-store lookup for those IDs (e.g., h.Sessions.Get(ctx, id) per id or a
batch method like h.Sessions.ListByIDs(ctx, ids) if available), build
sessionByID from that result, and then call
networkPeerPayloadFromInfoWithSessions(peer, sessionByID) for each peer; do not
return a 500 on session-store errors—log or ignore individual lookup failures
and fall back to using the plain peer payload when enrichment is missing.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `NetworkPeers` currently loads every session via `Sessions.ListAll` just to enrich the returned peer slice, which makes the endpoint O(all sessions) and turns a non-critical enrichment failure into a full `500`.
- Fix approach: look up only the returned peer `session_id` values via targeted session status calls, log lookup failures, and fall back to the plain peer payload when enrichment is unavailable.
