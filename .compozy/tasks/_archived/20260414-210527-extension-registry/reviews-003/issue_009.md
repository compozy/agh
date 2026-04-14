---
status: resolved
file: internal/registry/github/client.go
line: 508
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lNU,comment:PRRC_kwDOR5y4QM63oCuH
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Ignored error on rate limit body close and silent parse failure.**

Line 501 discards the close error with `_`. Additionally, line 497-498 silently ignores the parse error for `X-RateLimit-Remaining` header. Consider logging these for observability.


<details>
<summary>🛠️ Suggested fix</summary>

```diff
 	remaining, err := strconv.Atoi(remainingValue)
 	if err != nil {
+		c.logger.Debug("github: invalid X-RateLimit-Remaining header", "value", remainingValue, "error", err)
 		return nil
 	}
 	if remaining == 0 {
-		_ = response.Body.Close()
+		if closeErr := response.Body.Close(); closeErr != nil {
+			c.logger.Debug("github: close response body on rate limit", "error", closeErr)
+		}
 		return errors.New("github: rate limit exceeded; set GITHUB_TOKEN for higher limits")
 	}
```
</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/github/client.go` around lines 498 - 508, In the rate-limit
handling block, stop ignoring errors: when parsing X-RateLimit-Remaining
(variable remaining) handle the strconv.Atoi error instead of discarding it — if
err != nil and c.logger != nil call c.logger.Warn (include the header value and
err) and decide to treat remaining as 0 or skip rate checks; likewise replace `_
= response.Body.Close()` with proper error handling (if err :=
response.Body.Close(); err != nil and c.logger != nil call c.logger.Warn or
c.logger.Error with the error and context including
response.Request.URL.String()); reference symbols: response, remaining,
rateLimitWarnThreshold, c.logger. Ensure logs include the error variable so
parsing and close failures are observable while keeping the prior control flow.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `checkRateLimit()` silently returns on `strconv.Atoi` failure for `X-RateLimit-Remaining`, and the `remaining == 0` branch discards `response.Body.Close()` errors.
- Why this is a real defect: malformed rate-limit headers become invisible to observability, and body-close failures on a hard rate-limit response are currently dropped.
- Intended fix: log invalid header parsing via the existing logger, return a joined error when closing the body fails on a zero-remaining response, and add regression coverage for both paths.
- Resolution: `checkRateLimit()` now logs invalid `X-RateLimit-Remaining` values and joins zero-remaining failures with `closeResponseBody(...)`.
- Verification: Added `TestCheckRateLimitLogsInvalidRemainingHeader` and `TestCheckRateLimitJoinsCloseErrorWhenRemainingZero` in `internal/registry/github/client_test.go`, ran `go test ./internal/registry/github`, and then `make verify`.
