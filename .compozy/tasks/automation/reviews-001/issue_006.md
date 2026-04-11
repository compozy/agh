---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 260
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TB0K,comment:PRRC_kwDOR5y4QM623e7V
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wait for trigger execution before asserting run history.**

Lines 249-259 fetch run history immediately after `stopIntegrationSession(...)`. That path goes through the session observer and automation pipeline, so the run may not be persisted yet and this test can flake intermittently. Poll until the run appears or the deadline expires.



<details>
<summary>Proposed stabilization</summary>

```diff
-	runsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/triggers/"+created.Trigger.ID+"/runs", nil, nil)
-	if runsResp.StatusCode != http.StatusOK {
-		body, _ := io.ReadAll(runsResp.Body)
-		_ = runsResp.Body.Close()
-		t.Fatalf("trigger runs status = %d, want %d; body=%s", runsResp.StatusCode, http.StatusOK, string(body))
-	}
-	var runs contract.RunsResponse
-	decodeHTTPJSON(t, runsResp, &runs)
-	if len(runs.Runs) == 0 {
-		t.Fatalf("expected trigger run history, got %#v", runs.Runs)
-	}
+	var runs contract.RunsResponse
+	deadline := time.After(2 * time.Second)
+	ticker := time.NewTicker(25 * time.Millisecond)
+	defer ticker.Stop()
+	for {
+		runsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/triggers/"+created.Trigger.ID+"/runs", nil, nil)
+		if runsResp.StatusCode != http.StatusOK {
+			body, _ := io.ReadAll(runsResp.Body)
+			_ = runsResp.Body.Close()
+			t.Fatalf("trigger runs status = %d, want %d; body=%s", runsResp.StatusCode, http.StatusOK, string(body))
+		}
+
+		runs = contract.RunsResponse{}
+		decodeHTTPJSON(t, runsResp, &runs)
+		if len(runs.Runs) > 0 {
+			break
+		}
+
+		select {
+		case <-deadline:
+			t.Fatalf("expected trigger run history, got %#v", runs.Runs)
+		case <-ticker.C:
+		}
+	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	sessionID := createIntegrationSession(t, runtime)
	stopIntegrationSession(t, runtime, sessionID)

	var runs contract.RunsResponse
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		runsResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/automation/triggers/"+created.Trigger.ID+"/runs", nil, nil)
		if runsResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(runsResp.Body)
			_ = runsResp.Body.Close()
			t.Fatalf("trigger runs status = %d, want %d; body=%s", runsResp.StatusCode, http.StatusOK, string(body))
		}

		runs = contract.RunsResponse{}
		decodeHTTPJSON(t, runsResp, &runs)
		if len(runs.Runs) > 0 {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("expected trigger run history, got %#v", runs.Runs)
		case <-ticker.C:
		}
	}
	runID := runs.Runs[0].ID
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/udsapi_integration_test.go` around lines 246 - 260, The
test currently reads run history immediately after stopIntegrationSession and
can flake; modify the section after stopIntegrationSession(t, runtime,
sessionID) to poll the runs endpoint (use mustUnixRequest with http.MethodGet to
"http://unix/api/automation/triggers/"+created.Trigger.ID+"/runs") until
contract.RunsResponse.Runs is non-empty or a deadline expires (e.g., now+several
seconds), sleeping briefly between attempts (e.g., 100–200ms); on each attempt
decodeHTTPJSON into contract.RunsResponse, ensure response bodies are closed,
and if the deadline is reached call t.Fatalf reporting no run found. Use the
existing identifiers createIntegrationSession, stopIntegrationSession,
mustUnixRequest, decodeHTTPJSON, and contract.RunsResponse to locate and update
the code.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: This integration test queries trigger run history immediately after stopping the session, but the automation pipeline persists the run asynchronously through the observer path. I will make the assertion poll until the run appears or a deadline expires so the test reflects eventual persistence rather than racing it.
