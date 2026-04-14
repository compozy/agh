---
status: pending
file: internal/registry/github/client.go
line: 471
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lNM,comment:PRRC_kwDOR5y4QM63oCt_
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Ignored error on retry body close violates guidelines.**

Line 465 discards the close error with `_`. While this is in a retry path, the error should still be handled or logged.


<details>
<summary>🛠️ Suggested fix</summary>

```diff
 		if retryable && attempt < c.maxRetries {
-			_ = response.Body.Close()
+			if closeErr := response.Body.Close(); closeErr != nil {
+				c.logger.Debug("github: close response body before retry", "error", closeErr)
+			}
 			if err := c.sleep(ctx, backoff); err != nil {
```
</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError
		if retryable && attempt < c.maxRetries {
			if closeErr := response.Body.Close(); closeErr != nil {
				c.logger.Debug("github: close response body before retry", "error", closeErr)
			}
			if err := c.sleep(ctx, backoff); err != nil {
				return nil, fmt.Errorf("github: retry wait aborted: %w", err)
			}
			backoff = nextBackoff(backoff, c.maxBackoff)
			continue
		}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/github/client.go` around lines 463 - 471, The retry path
currently discards the error from response.Body.Close() (the `_ =
response.Body.Close()` line); change this to capture the error and handle it
instead of ignoring it: call err := response.Body.Close() and if err != nil then
log or surface it (e.g., use the client logger on c if available to Warnf/Debugf
with context "github: failed to close response body before retry" and include
err), otherwise decide to wrap/return the error; keep the existing retry logic
around attempt, c.maxRetries, c.sleep, backoff and nextBackoff unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
