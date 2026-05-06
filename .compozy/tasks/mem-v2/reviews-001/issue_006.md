---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 3441
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Ism,comment:PRRC_kwDOR5y4QM6-UFVq
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Stop building JSON request bodies with string concatenation.**

`runtime.workspace` comes from `t.TempDir()`, so on Windows the backslashes make this invalid JSON. `message` has the same problem for quotes/newlines. Marshal a struct/map instead; this helper is reused widely enough that one bad path or prompt will break multiple integration tests.
 
<details>
<summary>💡 Suggested fix</summary>

```diff
-	resp := mustHTTPRequest(
-		t,
-		runtime.client,
-		http.MethodPost,
-		mustURL(runtime.host, runtime.port, "/api/sessions"),
-		[]byte(`{"agent_name":"coder","workspace_path":"`+runtime.workspace+`"}`),
-		nil,
-	)
+	body := mustIntegrationJSON(map[string]any{
+		"agent_name":     "coder",
+		"workspace_path": runtime.workspace,
+	})
+	resp := mustHTTPRequest(
+		t,
+		runtime.client,
+		http.MethodPost,
+		mustURL(runtime.host, runtime.port, "/api/sessions"),
+		body,
+		nil,
+	)
```

```diff
-	resp := mustHTTPRequest(
-		t,
-		runtime.client,
-		http.MethodPost,
-		mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/prompt"),
-		[]byte(`{"message":"`+message+`"}`),
-		nil,
-	)
+	body := mustIntegrationJSON(map[string]any{
+		"message": message,
+	})
+	resp := mustHTTPRequest(
+		t,
+		runtime.client,
+		http.MethodPost,
+		mustURL(runtime.host, runtime.port, "/api/sessions/"+sessionID+"/prompt"),
+		body,
+		nil,
+	)
```
</details>


Also applies to: 3555-3562

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/httpapi/httpapi_integration_test.go` around lines 3434 - 3441,
The test builds JSON request bodies via string concatenation (see the
mustHTTPRequest call that injects runtime.workspace and other variables), which
breaks on Windows (backslashes) and with quotes/newlines in message; instead,
create a Go value (struct or map) for the request body, marshal it with
json.Marshal, and pass the resulting bytes to mustHTTPRequest; update the two
occurrences referenced (around the mustHTTPRequest that posts to "/api/sessions"
and the other block at lines ~3555-3562) to use e.g. a map{"agent_name":
"coder", "workspace_path": runtime.workspace} and json.Marshal so inputs are
properly escaped.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The HTTP integration helpers still build JSON bodies with string concatenation in `createIntegrationSession` and `sendPrompt`.
  - Both helpers interpolate dynamic values (`runtime.workspace`, prompt text), so quotes, backslashes, or newlines can produce invalid JSON and make the helpers platform-sensitive.
  - Fix approach: marshal Go values through the existing JSON helper instead of concatenating literals, and keep the helper behavior unchanged otherwise.
