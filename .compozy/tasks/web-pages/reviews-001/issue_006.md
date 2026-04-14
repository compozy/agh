---
status: resolved
file: internal/api/core/network_details.go
line: 418
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg31,comment:PRRC_kwDOR5y4QM63ZMHh
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Persisted history is ignored by the detail not-found check.**

`networkChannelDetailPayload` returns 404 before it even looks at `NetworkStore`. After the last session or peer disconnects, a channel with stored messages becomes impossible to reopen from the UI, while the messages endpoint still treats it as existing.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 415 - 418,
networkChannelDetailPayload currently returns errNetworkChannelNotFound when
filteredSessions and peers are empty, ignoring persisted messages; update the
not-found check to also consult the persistent store (e.g., NetworkStore) for
existing history before returning: after computing filteredSessions :=
sessionsForChannel(sessions, channel) and checking peers, call the appropriate
store method (e.g., NetworkStore.HasChannel, NetworkStore.GetMessages, or
equivalent) to see if the channel has persisted messages and only return
fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel) when filteredSessions
== 0 && len(peers) == 0 && the store reports no history; keep the error type
errNetworkChannelNotFound and reuse existing symbols (filteredSessions,
sessionsForChannel, peers, NetworkStore, networkChannelDetailPayload) to locate
where to add the store lookup.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `networkChannelDetailPayload` returns not found before consulting persisted message history, so a channel with stored timeline rows but no live peers or sessions cannot be reopened.
- Fix approach: consult `NetworkStore` before returning `errNetworkChannelNotFound` and allow detail payloads backed by persisted history alone.
