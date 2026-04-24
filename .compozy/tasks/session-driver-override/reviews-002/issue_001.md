---
status: resolved
file: internal/api/httpapi/transport_parity_integration_test.go
line: 211
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RcPB,comment:PRRC_kwDOR5y4QM6628Dj
---

# Issue 001: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Handle response read/close errors instead of discarding them.**

At Line 207 and Line 211, read/close errors are ignored with `_`, which can mask the real failure cause in this integration path.  


<details>
<summary>🔧 Proposed fix</summary>

```diff
 		if stopResp.StatusCode != http.StatusNoContent {
-			body, _ := io.ReadAll(stopResp.Body)
-			_ = stopResp.Body.Close()
+			body, readErr := io.ReadAll(stopResp.Body)
+			closeErr := stopResp.Body.Close()
+			if readErr != nil {
+				t.Fatalf("read HTTP stop response body error = %v", readErr)
+			}
+			if closeErr != nil {
+				t.Fatalf("close HTTP stop response body error = %v", closeErr)
+			}
 			t.Fatalf("HTTP stop session status = %d, want %d; body=%s", stopResp.StatusCode, http.StatusNoContent, string(body))
 		}
-		_ = stopResp.Body.Close()
+		if err := stopResp.Body.Close(); err != nil {
+			t.Fatalf("close HTTP stop response body error = %v", err)
+		}
```
</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/transport_parity_integration_test.go` around lines 206 -
211, The test currently discards errors when reading and closing stopResp.Body;
update the stop response handling in the test (references: stopResp, io.ReadAll,
stopResp.Body.Close, t.Fatalf, http.StatusNoContent) to check and surface both
the io.ReadAll error and the Body.Close error instead of assigning them to `_`.
Specifically, read the body into a variable, check the read error and include it
in the t.Fatalf message if non-nil, ensure the body is closed (use defer or
explicit close), capture any close error and include that in the failure message
as well so the final t.Fatalf shows status code, body string, read error and/or
close error when status is not http.StatusNoContent.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: The failure branch still discards both `io.ReadAll` and `Body.Close` errors for `stopResp`, which can hide the actual transport failure cause in this integration test. I will read and close the body explicitly, fail on either error, and keep the HTTP status assertion tied to the real response payload.
