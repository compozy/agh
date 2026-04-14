---
status: resolved
file: internal/registry/github/client.go
line: 507
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564r_W,comment:PRRC_kwDOR5y4QM63phd5
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`According to GitHub REST API official documentation, can a successful response still include `X-RateLimit-Remaining: 0`, meaning the current request consumed the last remaining request? If yes, what is the recommended way to distinguish a real rate-limit error from a successful final in-quota response?`

💡 Result:

Yes, according to GitHub REST API official documentation, a successful response (e.g., 200 OK) can include the X-RateLimit-Remaining: 0 header. This occurs when the current request consumes the last remaining request in the rate limit window, leaving 0 requests remaining afterward. The recommended way to distinguish a real rate-limit error from a successful final in-quota response is by the HTTP status code. Rate-limit errors return 403 Forbidden or 429 Too Many Requests with X-RateLimit-Remaining: 0 and an error message in the response body. Successful responses have 2xx status codes (like 200 OK), even if X-RateLimit-Remaining: 0. Clients should always check the status code first: treat 2xx as success (process the response body), and only on 403/429 with rate limit headers treat as error and wait until X-RateLimit-Reset or Retry-After.

Citations:

- 1: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
- 2: https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2026-03-10
- 3: https://docs.github.com/en/rest/overview/troubleshooting
- 4: https://docs.github.com/en/rest/using-the-rest-api/troubleshooting-the-rest-api

---



</details>

**Check HTTP status code before treating `remaining == 0` as a rate-limit error.**

GitHub's REST API returns `X-RateLimit-Remaining: 0` on successful 2xx responses when the request consumes the final remaining quota. The current `checkRateLimit()` incorrectly treats this as a hard error regardless of status code, causing valid successful responses to fail with dropped bodies. Rate-limit errors are distinguished by HTTP status (403 Forbidden or 429 Too Many Requests), not the header value alone. Gate the error return on status code `403` or `429`; otherwise, log as a warning for monitoring.

<details>
<summary>Suggested fix</summary>

```diff
 	if remaining == 0 {
-		rateLimitErr := errors.New("github: rate limit exceeded; set GITHUB_TOKEN for higher limits")
-		return joinErrors(rateLimitErr, closeResponseBody(response.Body, fmt.Sprintf("rate limit response for %s", requestURLString(response))))
+		if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
+			rateLimitErr := errors.New("github: rate limit exceeded; set GITHUB_TOKEN for higher limits")
+			return joinErrors(rateLimitErr, closeResponseBody(response.Body, fmt.Sprintf("rate limit response for %s", requestURLString(response))))
+		}
+		if c.logger != nil {
+			c.logger.Warn("github: rate limit exhausted after successful response", "url", requestURLString(response))
+		}
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/github/client.go` around lines 489 - 507, In
checkRateLimit, don't treat remaining == 0 as a hard error unconditionally;
first inspect response.StatusCode and only construct/return the rate-limit error
(using rateLimitErr, joinErrors and closeResponseBody) when the status code is
403 or 429; for other status codes with X-RateLimit-Remaining == "0" simply log
a warning (via c.logger.Debug/Warning) including the status and request URL and
return nil so successful 2xx responses aren't treated as failures. Ensure you
still handle parsing errors and nil response as before and refer to the existing
symbols checkRateLimit, remaining, requestURLString, closeResponseBody and
joinErrors when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `checkRateLimit` treats `X-RateLimit-Remaining: 0` as a hard error even for successful responses that legitimately consumed the last available request.
- Evidence: [`internal/registry/github/client.go`](internal/registry/github/client.go) lines 504-506 ignore the HTTP status code and always close/abort when `remaining == 0`.
- Fix plan: gate the hard failure on `403`/`429`, keep logging for successful exhaustion, and update tests for both the error and non-error paths.
- Resolution: Rate-limit exhaustion now fails only on `403/429`, while successful final-quota responses log a warning and continue. Verified with package tests and `make verify`.
