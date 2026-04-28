---
status: resolved
file: internal/api/core/network_details.go
line: 1242
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-IGMO,comment:PRRC_kwDOR5y4QM67_zc_
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**This turns cursor pagination into full-history reads.**

Lines 1238-1241 clear `BeforeMessageID`, `AfterMessageID`, and `Limit` before calling `ListNetworkMessages`, so every page request now loads the entire channel/peer history and trims it in memory. On active channels this becomes O(total_messages) per request and will regress latency and memory usage quickly.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 1233 - 1242, The helper
listTimelineRawMessages currently wipes out pagination by resetting
query.BeforeMessageID, query.AfterMessageID, and query.Limit before calling
networkStore.ListNetworkMessages, causing full-history reads; restore and pass
through the original cursor fields instead of clearing them (i.e., stop
modifying the incoming query in listTimelineRawMessages) or introduce an
explicit parameter or a new store method (e.g., ListNetworkMessagesFullHistory)
if a true full-scan is required; ensure calls that expect paginated results
continue to use the unmodified query and only perform full reads when explicitly
requested.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `listTimelineRawMessages` clears `BeforeMessageID`, `AfterMessageID`, and `Limit` before calling the store, so cursor requests scan the full raw channel or peer lane. The root cause is conflating raw fetch shape with handler-side visible timeline pagination. The fix should preserve cursor fields in the store query so cursor pages do not read the opposite side of history, while keeping `Limit` at the handler layer because public timelines filter directed traffic and coalesce presence before pagination. Existing assertions in `internal/api/core/network_test.go` explicitly expect cleared raw cursors; those tests need a minimal out-of-scope update to validate cursor pass-through while preserving handler-side limit semantics.

## Resolution

- Preserved raw `before`/`after` cursor fields when loading timeline messages while keeping handler-side `Limit` behavior for filtered/coalesced timelines.
- Updated network timeline tests to assert cursor pass-through for channel and peer pagination.
- Removed the now-dead pagination error return flagged by lint.
- Verified through targeted network tests and `make verify`.
