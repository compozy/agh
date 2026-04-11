---
status: resolved
file: internal/api/core/automation.go
line: 621
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0H,comment:PRRC_kwDOR5y4QM623e7S
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Unbounded request body read is a potential DoS vector.**

`io.ReadAll(c.Request.Body)` reads the entire body without a size limit. A malicious client could send an arbitrarily large payload to exhaust server memory.


<details>
<summary>🛡️ Proposed fix: Limit request body size</summary>

```diff
-	payload, err := io.ReadAll(c.Request.Body)
+	const maxWebhookPayloadSize = 1 << 20 // 1 MB
+	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookPayloadSize))
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/automation.go` around lines 618 - 621, The code uses
io.ReadAll(c.Request.Body) which is unbounded; replace it with a size-limited
read by wrapping c.Request.Body with a MaxBytesReader or LimitReader before
reading: define a constant (e.g. maxWebhookBodySize) and do c.Request.Body =
http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodySize) (or use
io.LimitReader) and then call io.ReadAll on that wrapped reader; update the
error handling around the existing payload, err block to return a clear error
when the body exceeds the limit.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `webhookRequestFromHTTP` currently performs an unbounded `io.ReadAll` on the request body, so a large payload can force unnecessary memory growth before validation runs. I will cap the body size at the HTTP boundary, return a clear validation/transport error when the limit is exceeded, and add coverage for the bounded-read path.
