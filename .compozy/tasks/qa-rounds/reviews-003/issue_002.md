---
status: resolved
file: internal/api/core/network_details.go
line: 1235
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-KLdo,comment:PRRC_kwDOR5y4QM68CnLG
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "ListNetworkMessages declarations/implementations:"
rg -n -C4 --type=go '\bListNetworkMessages\s*\('

echo
echo "NetworkMessageQuery cursor handling:"
rg -n -C4 --type=go 'BeforeMessageID|AfterMessageID|Limit'
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
# First, examine the store implementation to see how it handles cursors with Limit == 0
sed -n '75,150p' internal/store/globaldb/global_db_network_messages.go

# Then check the callers of listTimelineRawMessages in network_details.go
rg -n 'listTimelineRawMessages' internal/api/core/network_details.go
```

Repository: compozy/agh

Length of output: 2121

---

🏁 Script executed:

```shell
# Get the full buildNetworkMessageListQuery implementation
sed -n '125,220p' internal/store/globaldb/global_db_network_messages.go

# Get the calling context and complete listTimelineRawMessages
sed -n '1200,1240p' internal/api/core/network_details.go
```

Repository: compozy/agh

Length of output: 4330

---

🏁 Script executed:

```shell
# Verify the store applies cursors regardless of Limit value
sed -n '145,165p' internal/store/globaldb/global_db_network_messages.go

# Check if there are any other places where cursors are cleared before raw fetches
rg -n 'BeforeMessageID\s*=|AfterMessageID\s*=' internal/api/core/network_details.go
```

Repository: compozy/agh

Length of output: 943

---

🏁 Script executed:

```shell
# Look at the full switch statement in buildNetworkMessageListQuery
sed -n '145,175p' internal/store/globaldb/global_db_network_messages.go

# Verify that the tests DO check for empty cursors in raw fetches
grep -A5 "want empty raw" internal/api/core/network_test.go | head -20
```

Repository: compozy/agh

Length of output: 1817

---



</details>

**Strip cursors in `listTimelineRawMessages` to prevent store-side filtering before visibility coalescing.**

The `listTimelineRawMessages` function sets `Limit = 0` but does not clear `BeforeMessageID` / `AfterMessageID`. The store's `buildNetworkMessageListQuery` applies cursor filters regardless of Limit value, which can split episodes or drop cursor anchors before the handler applies visibility filtering and coalescing. Add:

```
rawQuery.BeforeMessageID = ""
rawQuery.AfterMessageID = ""
```

This ensures raw fetches retrieve the complete unfiltered dataset for handler-side pagination after visibility operations.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 1228 - 1235, The function
listTimelineRawMessages sets rawQuery.Limit = 0 but leaves cursor fields set,
which lets the store apply BeforeMessageID/AfterMessageID filtering prematurely;
modify listTimelineRawMessages to also clear rawQuery.BeforeMessageID and
rawQuery.AfterMessageID (set them to empty strings) before calling
networkStore.ListNetworkMessages so the store returns the full unfiltered
dataset for handler-side visibility coalescing and pagination.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `listTimelineRawMessages` clears `Limit` but leaves `BeforeMessageID` and `AfterMessageID` intact.
  - The global store applies cursor filters before returning rows regardless of `Limit`, which can remove the cursor anchor or split raw presence episodes before handler-side visibility filtering and coalescing.
  - Fix: clear both cursor fields on the raw query before calling `ListNetworkMessages`, while keeping the original query for handler-side pagination.

## Resolution

- Updated `listTimelineRawMessages` to clear `BeforeMessageID` and `AfterMessageID` before calling `ListNetworkMessages`.
- Updated channel and peer timeline tests so raw store fetches must receive empty cursors while handler-side pagination still uses the original request query.
- Verified with `go test -race ./internal/api/core -count=1` and `make verify`.
