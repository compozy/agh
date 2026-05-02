---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/api/httpapi/server_test.go
line: 750
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdT,comment:PRRC_kwDOR5y4QM69XbzS
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Assert the forbidden error payload, not only the status code.**

This new subtest can false-pass on any 403 path. Decode and assert the error body so it validates the loopback policy contract (same pattern used by the other forbidden-path tests in this file).

<details>
<summary>Suggested patch</summary>

```diff
 	if got, want := resp.StatusCode, http.StatusForbidden; got != want {
 		body, _ := io.ReadAll(resp.Body)
 		t.Fatalf("GET /api/vault/secrets status = %d, want %d; body=%s", got, want, string(body))
 	}
+	var payload contract.ErrorPayload
+	decodeServerJSON(t, resp, &payload)
+	if got, want := payload.Error, errLoopbackMutationRequired.Error(); got != want {
+		t.Fatalf("payload.Error = %q, want %q", got, want)
+	}
```
</details>

   
As per coding guidelines: "Testing: assertions MUST include both status-code and response body checks; status-code-only is insufficient."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	t.Run("Should block vault metadata reads on non-loopback HTTP", func(t *testing.T) {
		resp := doServerRequest(
			t,
			http.DefaultClient,
			http.MethodGet,
			mustURL("127.0.0.1", server.Port(), "/api/vault/secrets?namespace=sessions"),
			nil,
		)
		defer func() {
			_ = resp.Body.Close()
		}()
		if got, want := resp.StatusCode, http.StatusForbidden; got != want {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("GET /api/vault/secrets status = %d, want %d; body=%s", got, want, string(body))
		}
		var payload contract.ErrorPayload
		decodeServerJSON(t, resp, &payload)
		if got, want := payload.Error, errLoopbackMutationRequired.Error(); got != want {
			t.Fatalf("payload.Error = %q, want %q", got, want)
		}
	})
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/server_test.go` around lines 735 - 750, The subtest
"Should block vault metadata reads on non-loopback HTTP" only checks
StatusForbidden and can false-pass; update it to also read and decode the
response body and assert the JSON error payload matches the expected
forbidden-loopback policy error (same pattern used by other forbidden-path
tests). After calling doServerRequest/mustURL and obtaining resp, read
resp.Body, unmarshal the error JSON and assert the error fields/message
correspond to the loopback-violation response (in addition to asserting
resp.StatusCode == http.StatusForbidden) so the test validates both status and
body.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The non-loopback vault-read test currently asserts only `403`, which can false-pass on the wrong failure mode.
  - This violated the repo’s test contract that status-only assertions are insufficient; I updated `internal/api/httpapi/server_test.go` to decode `contract.ErrorPayload` and assert the expected loopback-policy error body.
  - Verification: `make verify` passed with the stronger forbidden-path assertion.
