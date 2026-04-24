---
status: resolved
file: internal/store/globaldb/global_db_network_channels_test.go
line: 94
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeK,comment:PRRC_kwDOR5y4QM66CAkt
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't assert a stable order until the store query orders rows.**

`internal/store/globaldb/global_db_network_channels.go:88-137` currently builds `SELECT ... FROM network_channels` without `ORDER BY` or `LIMIT`, so `entries[0].Channel == "ops.alerts"` depends on SQLite's incidental row order and `Limit: 10` is not actually being exercised. Either sort/limit in the store query and assert that here, or compare the returned channels as a set.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_network_channels_test.go` around lines 81 -
94, The test assumes a stable row order and a working LIMIT, but the store query
in global_db_network_channels.go (the ListNetworkChannels implementation)
currently omits ORDER BY and LIMIT; update the store code to apply a
deterministic ORDER BY (e.g., by Channel or created_at) and honor Limit from
NetworkChannelQuery inside ListNetworkChannels, or alternatively change the test
in global_db_network_channels_test.go to assert the returned channels as an
unordered set (compare entries' Channel values regardless of order) and verify
the length respects Limit; reference ListNetworkChannels, NetworkChannelQuery,
entries, and global_db_network_channels.go when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: `ListNetworkChannels()` already applies `ORDER BY updated_at DESC, channel ASC` and routes `query.Limit` through `store.AppendLimit()`. The test is asserting deterministic behavior against the current implementation, so the specific review concern is stale relative to the code under review.
- Resolution: no code change. The current implementation already applies the requested ordering and limit handling.
- Verification: `go test ./internal/store/globaldb` and `make verify`
