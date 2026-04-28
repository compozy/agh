---
status: resolved
file: internal/api/core/network_details.go
line: 1445
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-KLds,comment:PRRC_kwDOR5y4QM68CnLM
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Reject cursors that are not present in the visible timeline.**

When a `before`/`after` cursor points to a row that was filtered out or coalesced away, `indexNetworkTimelineViewByMessageID` returns `-1` and pagination silently falls back to the first/last page. That returns duplicated/incorrect pages instead of the existing `"message cursor not found"` validation error.


<details>
<summary>Suggested fix</summary>

```diff
-func paginateNetworkTimelineViews(
+func paginateNetworkTimelineViews(
 	views []networkTimelineMessageView,
 	query store.NetworkMessageQuery,
-) []networkTimelineMessageView {
+) ([]networkTimelineMessageView, error) {
 	paginated := views
 	if before := strings.TrimSpace(query.BeforeMessageID); before != "" {
 		index := indexNetworkTimelineViewByMessageID(paginated, before)
-		if index >= 0 {
-			paginated = paginated[:index]
+		if index < 0 {
+			return nil, sql.ErrNoRows
 		}
+		paginated = paginated[:index]
 	}
 	if after := strings.TrimSpace(query.AfterMessageID); after != "" {
 		index := indexNetworkTimelineViewByMessageID(paginated, after)
-		if index >= 0 {
-			paginated = paginated[index+1:]
+		if index < 0 {
+			return nil, sql.ErrNoRows
 		}
+		paginated = paginated[index+1:]
 	}
 	if query.Limit <= 0 || len(paginated) <= query.Limit {
-		return paginated
+		return paginated, nil
 	}
 	if strings.TrimSpace(query.BeforeMessageID) != "" {
-		return paginated[len(paginated)-query.Limit:]
+		return paginated[len(paginated)-query.Limit:], nil
 	}
-	return paginated[:query.Limit]
+	return paginated[:query.Limit], nil
 }
```

`networkTimelinePayloads` should then return an error and let `respondNetworkMessageError(...)` keep the existing 400 mapping.
</details>


Also applies to: 1456-1464

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 1435 - 1445, The
pagination code currently silences missing cursors by treating
indexNetworkTimelineViewByMessageID(...)= -1 as "not found" and continues;
instead, when indexNetworkTimelineViewByMessageID returns -1 for either
query.BeforeMessageID or query.AfterMessageID you should return a validation
error from networkTimelinePayloads and propagate it so
respondNetworkMessageError can map it to a 400. Concretely, in the blocks that
handle BeforeMessageID and AfterMessageID, check the index result and if index
== -1 return a descriptive error (e.g., "message cursor not found") from
networkTimelinePayloads rather than trimming/ slicing paginated; ensure the
caller uses respondNetworkMessageError to convert that error to the existing 400
response.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `paginateNetworkTimelineViews` treats a missing cursor index as a no-op and returns the first or last page instead of rejecting the request.
  - After public/direct visibility filtering and presence coalescing, a raw cursor can point at a row that is not present in the visible timeline; silently ignoring that cursor duplicates pages and masks invalid client state.
  - Fix: make timeline pagination return an error when `before` or `after` is absent from the visible/coalesced views, propagate it through the channel and peer handlers, and rely on `respondNetworkMessageError` for the existing 400 "message cursor not found" response.

## Resolution

- Changed visible timeline pagination to return a cursor-not-found error when `before` or `after` is absent from the visible/coalesced views.
- Propagated that error through both channel and peer message handlers so `respondNetworkMessageError` preserves the existing 400 response.
- Added regression coverage for hidden directed-message cursors and coalesced-away presence cursors returning `message cursor not found`.
- Verified with `go test -race ./internal/api/core -count=1` and `make verify`.
