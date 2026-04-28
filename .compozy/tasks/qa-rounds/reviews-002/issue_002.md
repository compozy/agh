---
status: resolved
file: internal/api/core/network_details.go
line: 217
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-JRP8,comment:PRRC_kwDOR5y4QM68BYWu
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Cursor pages can incorrectly 404 history-only channels.**

An empty `rawMessages` slice does not mean the channel is missing once `before`/`after` is in play. For history-only rooms with no live sessions/peers/metadata, paging past the last visible item will hit this branch and return 404 instead of `200` with an empty page.

<details>
<summary>Suggested fix</summary>

```diff
-	if len(rawMessages) == 0 && !networkChannelExists(sessions, peers, metadata, channel) {
+	if len(rawMessages) == 0 &&
+		strings.TrimSpace(query.BeforeMessageID) == "" &&
+		strings.TrimSpace(query.AfterMessageID) == "" &&
+		!networkChannelExists(sessions, peers, metadata, channel) {
 		notFoundErr := fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
 		h.respondError(c, http.StatusNotFound, notFoundErr)
 		return
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if len(rawMessages) == 0 &&
		strings.TrimSpace(query.BeforeMessageID) == "" &&
		strings.TrimSpace(query.AfterMessageID) == "" &&
		!networkChannelExists(sessions, peers, metadata, channel) {
		notFoundErr := fmt.Errorf("%w: %s", errNetworkChannelNotFound, channel)
		h.respondError(c, http.StatusNotFound, notFoundErr)
		return
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/network_details.go` around lines 213 - 217, The current
check treats an empty rawMessages slice as "not found" and calls h.respondError,
which incorrectly 404s history-only channels when paging (using before/after);
instead, first call networkChannelExists(sessions, peers, metadata, channel) and
only return the 404 via h.respondError if the channel truly does not exist; if
the channel exists but rawMessages is empty (especially when before/after is
set), return a 200 empty page response (use the same success response path that
would return an empty list) so paging past the end yields a 200 with no messages
rather than a 404.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `NetworkChannelMessages` currently treats `len(rawMessages) == 0` plus no live session/peer/metadata as a missing channel.
  - With cursor pagination, the store can legitimately return an empty slice for a history-only channel after the caller pages past the last visible item; in that case the handler should return `200` with an empty `messages` array, not `404`.
  - Fix approach: only run the missing-channel 404 check for non-cursor requests. Cursor requests with an empty page will continue through the normal success response path.
  - Additional regression coverage in `internal/api/core/network_test.go` is required to prove the behavior because the scoped production file has no local test cases.

## Resolution

- Updated `NetworkChannelMessages` so the missing-channel 404 branch applies only to non-cursor requests.
- Added regression coverage that proves an empty cursor page on a history-only channel returns `200` with no messages.
- Verified with targeted `go test -race ./internal/api/core -run 'TestListAgentsWorkspaceResolverUnavailable|TestBaseHandlersNetworkChannelMessagesPreserveRemoteAuthors' -count=1`.
- Verified the repository gate with `make verify` after code changes.
