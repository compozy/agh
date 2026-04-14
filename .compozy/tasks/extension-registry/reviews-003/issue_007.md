---
status: resolved
file: internal/registry/github/client.go
line: 274
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lNG,comment:PRRC_kwDOR5y4QM63oCt4
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Ignored error on `response.Body.Close()` violates guidelines.**

Line 272 discards the close error with `_`. Per coding guidelines, every error must be handled or have a written justification.


<details>
<summary>🛠️ Suggested fix</summary>

```diff
 	if err := validateDownloadContentType(contentType); err != nil {
-		_ = response.Body.Close()
-		return nil, err
+		closeErr := response.Body.Close()
+		return nil, joinErrors(err, closeErr)
 	}
```
</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/github/client.go` around lines 270 - 274, The code
currently discards the error from response.Body.Close() when
validateDownloadContentType fails; update the error handling around
response.Body.Close() (the block near validateDownloadContentType and the
response variable) to capture and handle the close error instead of using `_`,
e.g., call response.Body.Close(), check the returned error, and return or wrap
that error (or log it with context) along with the original validation error so
no Close() error is ignored.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the download content-type validation branch in `Client.Download()` closes the response body with `_ = response.Body.Close()`, so a close failure is discarded.
- Why this is a real defect: it violates the repo's no-ignored-errors rule and loses useful context on failed download cleanup.
- Intended fix: route the close through `closeResponseBody()`/`joinErrors()` and add a regression test with a close-failing response body.
- Resolution: `Client.Download()` now joins the content-type validation failure with `closeResponseBody(...)` instead of discarding close errors.
- Verification: Added `TestClientDownloadJoinsCloseErrorOnContentTypeValidationFailure` in `internal/registry/github/client_test.go`, ran `go test ./internal/registry/github`, and then `make verify`.
