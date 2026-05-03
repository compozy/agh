---
provider: coderabbit
pr: "90"
round: 2
round_created_at: 2026-05-03T03:57:53.330715Z
status: resolved
file: extensions/bridges/teams/provider_test.go
line: 769
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KWjj,comment:PRRC_kwDOR5y4QM69Zj0K
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the blocked-redirect failure mode, not just “any error”.**

These subtests still pass if the request is rejected before the redirect path is exercised—for example, if loopback URL validation changes later. Assert a `*bridgesdk.HTTPError` with `StatusCode == http.StatusTemporaryRedirect` (or equivalent) so the test proves the client stopped on the 307 response.

<details>
<summary>Suggested assertion pattern</summary>

```diff
-		if _, err := fetchTeamsOpenIDMetadata(context.Background(), trusted.URL); err == nil {
-			t.Fatal("fetchTeamsOpenIDMetadata(redirect) error = nil, want non-nil")
-		}
+		_, err := fetchTeamsOpenIDMetadata(context.Background(), trusted.URL)
+		var httpErr *bridgesdk.HTTPError
+		if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusTemporaryRedirect {
+			t.Fatalf("fetchTeamsOpenIDMetadata(redirect) error = %v, want blocked redirect HTTP 307", err)
+		}
```
</details>

 
As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".


Also applies to: 799-801, 839-841

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@extensions/bridges/teams/provider_test.go` around lines 767 - 769, The test
currently only checks for any non-nil error from fetchTeamsOpenIDMetadata;
change the assertion to specifically assert the error is a *bridgesdk.HTTPError
(use errors.As or require.ErrorAs) and that its StatusCode equals
http.StatusTemporaryRedirect (307) so we prove the client stopped on the
redirect response; update the same pattern in the sibling assertions around the
other subtests (the blocks at the other reported locations) to use ErrorAs +
check err.StatusCode == http.StatusTemporaryRedirect instead of a generic
non-nil check.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: the redirect regression tests at the cited lines only assert `err != nil`, so they would still pass if the request failed for an unrelated reason before the redirect response was observed. That leaves the redirect-stop behavior under-specified.
- Fix approach: tighten the three redirect assertions to require `*bridgesdk.HTTPError` with `StatusCode == http.StatusTemporaryRedirect`, so the tests prove the client stops on the 307 response and never follows the redirect target.
- Resolution: the OpenID metadata, JWKS, and token redirect tests now assert the exact blocked-redirect `*bridgesdk.HTTPError` shape and `307` status code.
- Verification: `go test ./extensions/bridges/teams ./internal/network -count=1 -race`, `make verify`.
