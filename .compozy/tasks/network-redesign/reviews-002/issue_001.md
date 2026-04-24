---
status: resolved
file: internal/acp/types_test.go
line: 206
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167301360,nitpick_hash:ae87e4cae610
review_hash: ae87e4cae610
source_review_id: "4167301360"
source_review_submitted_at: "2026-04-24T01:39:58Z"
---

# Issue 001: Assert the flushed ToolCallID, not just the event type.
## Review Comment

This case still passes if the buffer releases the wrong deferred result as long as it is *some* `EventTypeToolResult`. Checking the emitted `ToolCallID` makes the cap test prove the newest retained entry actually wins.

As per coding guidelines, `Ensure tests verify behavior outcomes, not just function calls`.

## Triage

- Decision: `valid`
- Root cause: The bounded-buffer branch in `TestEmitPromptEventDeferredToolResultsStayBounded` proves that the oldest deferred result is evicted, but the final assertion only checks that some `tool_result` event flushes after the newest tool call. It does not assert that the flushed deferred result is specifically bound to `tool-128`.
- Fix plan: Extend the final assertion in `internal/acp/types_test.go` to verify the emitted deferred result keeps `ToolCallID == "tool-128"`, which makes the cap test prove the newest retained result wins.
- Outcome: Updated the ACP cap test to assert the flushed deferred result carries `ToolCallID == "tool-128"`. Verified with `go test ./internal/acp -count=1` and the full `make verify` gate.
