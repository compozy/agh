---
status: resolved
file: internal/api/core/network_details.go
line: 216
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-KLdi,comment:PRRC_kwDOR5y4QM68CnK_
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Paginated empty pages now mask unknown channels.**

With any `before`/`after` cursor, this branch skips the 404 even when there is no metadata, no live peer/session, and no stored history. That makes `/network/channels/<unknown>/messages?after=...` indistinguishable from “past end of history”.


<details>
<summary>Suggested direction</summary>

```diff
-	if len(rawMessages) == 0 &&
-		strings.TrimSpace(query.BeforeMessageID) == "" &&
-		strings.TrimSpace(query.AfterMessageID) == "" &&
-		!networkChannelExists(sessions, peers, metadata, channel) {
-		notFoundErr := fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
-		h.respondError(c, http.StatusNotFound, notFoundErr)
-		return
-	}
+	if len(rawMessages) == 0 && !networkChannelExists(sessions, peers, metadata, channel) {
+		history, err := networkStore.ListNetworkMessages(c.Request.Context(), store.NetworkMessageQuery{
+			Channel: channel,
+			Limit:   1,
+		})
+		if err != nil {
+			h.respondNetworkMessageError(c, err)
+			return
+		}
+		if len(history) == 0 {
+			notFoundErr := fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
+			h.respondError(c, http.StatusNotFound, notFoundErr)
+			return
+		}
+	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 213 - 216, The current
conditional lets requests with before/after cursors fall through and mask
unknown channels; ensure unknown channels yield a 404 regardless of pagination
cursors by checking channel existence first: call networkChannelExists(sessions,
peers, metadata, channel) and if it returns false and len(rawMessages) == 0 then
return the 404 error immediately (i.e., move or add the existence check out of
the combined if that includes query.BeforeMessageID/query.AfterMessageID so that
channel non-existence is handled separately from “past end of history”
pagination).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `NetworkChannelMessages` currently suppresses the unknown-channel 404 whenever `before` or `after` is present.
  - Once raw timeline loading is corrected to fetch the complete unpaginated channel history, the handler can distinguish an unknown channel from an empty page by checking `len(rawMessages) == 0 && !networkChannelExists(...)` independently of cursor inputs.
  - Fix: remove the cursor guards from the not-found branch and add regression coverage for a cursor request against a channel with no metadata, peers, sessions, or stored history.

## Resolution

- Removed the cursor guards from the channel not-found branch in `NetworkChannelMessages`.
- Added regression coverage for a paginated request against a channel with no metadata, peers, sessions, or stored history returning 404 instead of an empty 200 page.
- Verified with `go test -race ./internal/api/core -count=1` and `make verify`.
