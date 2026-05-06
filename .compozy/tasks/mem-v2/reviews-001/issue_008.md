---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 2948
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isp,comment:PRRC_kwDOR5y4QM6-UFVu
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Stop hand-building JSON in the shared request helpers.**

`createIntegrationSession` and `sendPrompt` splice dynamic strings directly into JSON literals, and `sendPrompt` also ignores the response read/close errors. A quoted prompt or path will produce invalid JSON, and a truncated response will be silently swallowed across every caller of these helpers.

 
<details>
<summary>Suggested fix</summary>

```diff
 func createIntegrationSession(t *testing.T, runtime integrationRuntime) string {
 	t.Helper()
+	body, err := json.Marshal(map[string]string{
+		"agent_name":     "coder",
+		"workspace_path": runtime.workspace,
+	})
+	if err != nil {
+		t.Fatalf("json.Marshal(create session body) error = %v", err)
+	}

 	resp := mustUnixRequest(
 		t,
 		runtime.client,
 		http.MethodPost,
 		"http://unix/api/sessions",
-		[]byte(`{"agent_name":"coder","workspace_path":"`+runtime.workspace+`"}`),
+		body,
 		nil,
 	)
@@
 func sendPrompt(t *testing.T, runtime integrationRuntime, sessionID string, message string) {
 	t.Helper()
+	body, err := json.Marshal(map[string]string{"message": message})
+	if err != nil {
+		t.Fatalf("json.Marshal(prompt body) error = %v", err)
+	}

 	resp := mustUnixRequest(
 		t,
 		runtime.client,
 		http.MethodPost,
 		"http://unix/api/sessions/"+sessionID+"/prompt",
-		[]byte(`{"message":"`+message+`"}`),
+		body,
 		nil,
 	)
 	if resp.StatusCode != http.StatusOK {
 		body, _ := io.ReadAll(resp.Body)
 		_ = resp.Body.Close()
 		t.Fatalf("prompt status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
 	}
-	_, _ = io.ReadAll(resp.Body)
-	_ = resp.Body.Close()
+	if _, err := io.ReadAll(resp.Body); err != nil {
+		t.Fatalf("io.ReadAll(prompt response) error = %v", err)
+	}
+	if err := resp.Body.Close(); err != nil {
+		t.Fatalf("prompt response body close error = %v", err)
+	}
 }
```
</details>
As per coding guidelines, "Never ignore errors with `_` in production code or in tests; every error must be handled or have a written justification".


Also applies to: 3056-3070

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/udsapi/udsapi_integration_test.go` around lines 2942 - 2948, The
helpers createIntegrationSession and sendPrompt currently construct request
bodies by concatenating strings (e.g., embedding runtime.workspace or the prompt
directly into JSON) and ignore response read/close errors; replace hand-built
JSON with proper encoding using encoding/json (marshal a struct or map) when
building the POST body in mustUnixRequest calls inside createIntegrationSession
and sendPrompt to ensure proper quoting/escaping, and ensure all response bodies
are closed and io.ReadAll errors are checked and returned/handled (do not use
blank identifier for errors) so callers receive and can assert on real errors.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The UDS integration helpers `createIntegrationSession` and `sendPrompt` still splice dynamic values into JSON literals exactly as described in the review.
  - `sendPrompt` also still discards the `io.ReadAll` and `Close` errors on the response body, which violates the repo rule against ignored errors even in tests.
  - Fix approach: marshal helper payloads with `encoding/json` and handle the response read/close errors explicitly.
