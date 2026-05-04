---
status: resolved
file: internal/api/testutil/apitest.go
line: 1032
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsS,comment:PRRC_kwDOR5y4QM67YHCs
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`WaitInbox` fallback no longer exercises the behavior this API is meant to test.**

Line 1031 delegates straight to `Inbox`, so the stub ignores `channel` and returns immediately instead of modeling the wait path. That can let tests pass even when production code is polling the wrong channel or mishandling timeout/cancellation behavior.

<details>
<summary>Safer fallback</summary>

```diff
 func (s StubNetworkService) WaitInbox(
 	ctx context.Context,
 	sessionID string,
 	channel string,
 ) ([]network.Envelope, error) {
 	if s.WaitInboxFn != nil {
 		return s.WaitInboxFn(ctx, sessionID, channel)
 	}
-	return s.Inbox(ctx, sessionID)
+	return nil, errors.New("stub network service WaitInbox not implemented")
 }
```
</details>


As per coding guidelines, "Check dependent package APIs before writing integration code or tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/testutil/apitest.go` around lines 1023 - 1032, The
StubNetworkService.WaitInbox fallback currently ignores the channel and returns
immediately by delegating to Inbox; change it so when WaitInboxFn is nil the
stub simulates the real "wait" behavior: repeatedly call s.Inbox(ctx, sessionID)
(or otherwise check the underlying inbox) filtering envelopes by the requested
channel, loop with a short sleep or use context-aware waiting until matching
envelopes are found or ctx is done, and return ctx.Err() on
cancellation/timeout; update the StubNetworkService.WaitInbox implementation
(referencing WaitInbox, WaitInboxFn, Inbox, and StubNetworkService) to honor the
channel parameter and context cancellation rather than returning immediately.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `StubNetworkService.WaitInbox` falls back to `Inbox`, which ignores the requested channel and returns immediately. That hides missing `WaitInboxFn` setup in tests and does not model the production wait contract. Fix by making the default fallback fail explicitly when `WaitInboxFn` is not configured; tests that exercise waiting must provide a wait implementation.
