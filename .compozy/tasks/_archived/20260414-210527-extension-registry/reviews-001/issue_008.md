---
status: resolved
file: internal/registry/github/client.go
line: 284
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WUr,comment:PRRC_kwDOR5y4QM63mad6
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Surface download HTTP failures before validating the archive body.**

For binary requests, `doRequest()` returns the final `*http.Response` even when GitHub answered `401`, `404`, or a retried `5xx`. `Download()` then turns that API failure into `"unexpected download content type"` or `"missing Content-Type"` instead of the real cause.


<details>
<summary>🛠️ Suggested fix</summary>

```diff
 		switch {
 		case selection.asset != nil:
 			response, err = c.doRequest(ctx, http.MethodGet, firstNonEmpty(selection.asset.URL, selection.asset.BrowserDownloadURL), acceptBinary, true)
 			if err != nil {
 				return nil, fmt.Errorf("github: download asset for %q: %w", repo.full, err)
@@
 		default:
 			return nil, fmt.Errorf("github: no download candidate resolved for %q", repo.full)
 		}
+
+		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
+			return nil, responseError(response, "download", repo.full)
+		}
 
 		contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
 		if err := validateDownloadContentType(contentType); err != nil {
 			_ = response.Body.Close()
 			return nil, err
```
</details>


Also applies to: 417-477

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/github/client.go` around lines 248 - 284, The Download flow
currently proceeds to validateContentType even when c.doRequest returned an HTTP
error response (e.g., 401/404/5xx), masking the real failure; update the logic
in Download (the switch handling selection.asset / selection.useTarball where
response is assigned from c.doRequest) to check response.StatusCode and treat
non-2xx codes as errors (closing response.Body) before calling
validateDownloadContentType, returning a descriptive error (including status
code and URL) so callers of Download get the actual HTTP failure instead of
"unexpected download content type"; apply the same change in the other mirrored
block (around lines noted 417-477) that also returns a response from
c.doRequest.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `Client.Download()` trusts the binary `doRequest()` response and validates the body content type before checking the final HTTP status. That masks 401/404/5xx download failures as content-type errors. I will reject non-2xx download responses before archive validation in `internal/registry/github/client.go` and add coverage in `internal/registry/github/client_test.go`, which is outside the listed scope but is the minimal targeted regression test location.
- Resolution: `internal/registry/github/client.go` now rejects non-2xx download responses before content-type validation, with HTTP-failure regression coverage in `internal/registry/github/client_test.go`.
- Verification: `go test ./internal/registry/...`; `make verify`
