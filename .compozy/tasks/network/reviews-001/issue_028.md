---
status: resolved
file: internal/session/manager_lifecycle.go
line: 234
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZp,comment:PRRC_kwDOR5y4QM623eZ7
---

# Issue 028: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid surfacing `Stop` as failed when only network leave cleanup is canceled.**

At Line 232, if `ctx` is already canceled/deadline-exceeded, `leaveNetworkPeer` can fail after the session is fully stopped and persisted, making `Stop` return an error for a successful stop operation. Treat cancellation as best-effort cleanup instead of a hard stop failure.  


<details>
<summary>Suggested adjustment</summary>

```diff
-	if err := m.leaveNetworkPeer(ctx, session); err != nil {
-		errs = append(errs, fmt.Errorf("session: leave network space for %q: %w", session.ID, err))
-	}
+	if err := m.leaveNetworkPeer(ctx, session); err != nil {
+		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
+			m.sessionLogger(session).Warn("session: leave network space canceled", "session_id", session.ID, "error", err)
+		} else {
+			errs = append(errs, fmt.Errorf("session: leave network space for %q: %w", session.ID, err))
+		}
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if err := m.leaveNetworkPeer(ctx, session); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			m.sessionLogger(session).Warn("session: leave network space canceled", "session_id", session.ID, "error", err)
		} else {
			errs = append(errs, fmt.Errorf("session: leave network space for %q: %w", session.ID, err))
		}
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager_lifecycle.go` around lines 232 - 234, The call to
m.leaveNetworkPeer in Stop/manager lifecycle should not turn a
canceled/deadline-exceeded context into a hard Stop failure; update the error
handling around m.leaveNetworkPeer(ctx, session) so that if it returns
context.Canceled or context.DeadlineExceeded (use errors.Is(err,
context.Canceled) / errors.Is(err, context.DeadlineExceeded)) you treat it as
best-effort and do not append it to errs, otherwise keep wrapping and appending
the error as before; locate the call to m.leaveNetworkPeer and change the
conditional that appends to errs to ignore those context cancellation errors.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `finalizeStopped` persists the stopped state before attempting `leaveNetworkPeer`, so a later `context.Canceled` or `context.DeadlineExceeded` from that best-effort cleanup should not turn a successful stop into an error return. The fix is to suppress cancellation/deadline leave failures, log them as warnings, and keep returning hard errors only for real leave failures.
  Resolved by downgrading canceled/deadline leave failures to warnings in `internal/session/manager_lifecycle.go` and by adding stop-path coverage for both cases in `internal/session/manager_hooks_test.go`. Verified with package tests and a clean `make verify`.
