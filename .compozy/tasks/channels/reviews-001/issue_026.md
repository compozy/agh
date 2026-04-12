---
status: resolved
file: internal/extension/channel_delivery_notifier.go
line: 45
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLu,comment:PRRC_kwDOR5y4QM623eJH
---

# Issue 026: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Error ignored without justification.**

`FailSession` error is discarded. Per coding guidelines, every error must be handled or have a written justification. At minimum, add a comment explaining why this is safe to ignore, or log the error.


<details>
<summary>🔧 Suggested fix (add logging)</summary>

```diff
 	if n.broker != nil && sess != nil {
-		_ = n.broker.FailSession(ctx, sess.ID, "")
+		if err := n.broker.FailSession(ctx, sess.ID, ""); err != nil {
+			// Best-effort cleanup; session is already stopping so failure is non-critical
+			_ = err
+		}
 	}
```

Or if truly safe to ignore, add a comment:
```diff
 	if n.broker != nil && sess != nil {
-		_ = n.broker.FailSession(ctx, sess.ID, "")
+		// Best-effort: failing already-stopped deliveries is non-critical
+		_ = n.broker.FailSession(ctx, sess.ID, "")
 	}
```
</details>

As per coding guidelines: "Never ignore errors with `_` — every error must be handled or have a written justification."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if n.broker != nil && sess != nil {
		if err := n.broker.FailSession(ctx, sess.ID, ""); err != nil {
			// Best-effort cleanup; session is already stopping so failure is non-critical
			_ = err
		}
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/channel_delivery_notifier.go` around lines 43 - 45, The
call to FailSession on n.broker currently discards its error ( _ =
n.broker.FailSession(ctx, sess.ID, "") ); update this to handle the error:
capture the returned error and either log it via the component's logger (e.g.,
n.logger.Errorf or n.log.Warnf) with context (session ID and action), or if you
intentionally want to ignore it, replace the discard with a short comment
explaining why ignoring is safe and referencing any invariants; ensure you
modify the call site (the block that checks n.broker != nil && sess != nil) so
the error is not silently dropped.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: `OnSessionStopped` currently discards `FailSession` errors with `_ =`, which violates the repo rule against silently ignoring errors. This notifier cannot return an error, so the best available fix is to make the best-effort behavior explicit instead of silently swallowing it.
- Root cause: Cleanup failure from broker projection is intentionally non-fatal, but the implementation does not document that invariant and hides the decision behind `_ =`.
- Fix plan: Replace the silent discard with an explicit error branch and justification comment while preserving downstream notification behavior.
- Resolution: The notifier now logs broker cleanup failures via `slog` instead of silently discarding them, and the change passed `make verify`.
