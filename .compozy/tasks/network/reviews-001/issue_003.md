---
status: resolved
file: internal/api/udsapi/extensions_additional_test.go
line: 81
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBZc,comment:PRRC_kwDOR5y4QM623eZt
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Strengthen negative-path checks with specific error assertions.**

These cases currently assert only status code. Add targeted assertions on error payload/message (e.g., contains `path`, `checksum`, `name`, `not implemented`) to avoid false positives where status is correct but error semantics regress.

<details>
<summary>Suggested diff</summary>

```diff
 import (
 	"context"
 	"errors"
 	"net/http"
 	"os"
+	"strings"
 	"testing"
@@
 	missingPath := performRequest(t, engine, http.MethodPost, "/api/extensions", []byte(`{"checksum":"sha256:abc"}`))
 	if missingPath.Code != http.StatusBadRequest {
 		t.Fatalf("missing path status = %d, want %d; body=%s", missingPath.Code, http.StatusBadRequest, missingPath.Body.String())
 	}
+	if !strings.Contains(strings.ToLower(missingPath.Body.String()), "path") {
+		t.Fatalf("missing path body should mention path; body=%s", missingPath.Body.String())
+	}
@@
 	missingChecksum := performRequest(t, engine, http.MethodPost, "/api/extensions", []byte(`{"path":"/tmp/ext-a"}`))
 	if missingChecksum.Code != http.StatusBadRequest {
 		t.Fatalf("missing checksum status = %d, want %d; body=%s", missingChecksum.Code, http.StatusBadRequest, missingChecksum.Body.String())
 	}
+	if !strings.Contains(strings.ToLower(missingChecksum.Body.String()), "checksum") {
+		t.Fatalf("missing checksum body should mention checksum; body=%s", missingChecksum.Body.String())
+	}
@@
 	blankName := performRequest(t, engine, http.MethodPost, "/api/extensions/%20%20/enable", nil)
 	if blankName.Code != http.StatusBadRequest {
 		t.Fatalf("blank name status = %d, want %d; body=%s", blankName.Code, http.StatusBadRequest, blankName.Body.String())
 	}
+	if !strings.Contains(strings.ToLower(blankName.Body.String()), "name") {
+		t.Fatalf("blank name body should mention name; body=%s", blankName.Body.String())
+	}
@@
 	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", nil)
 	if recorder.Code != http.StatusNotImplemented {
 		t.Fatalf("approve status = %d, want %d; body=%s", recorder.Code, http.StatusNotImplemented, recorder.Body.String())
 	}
+	if !strings.Contains(strings.ToLower(recorder.Body.String()), "not implemented") {
+		t.Fatalf("approve body should mention not implemented; body=%s", recorder.Body.String())
+	}
```
</details>

As per coding guidelines, `**/*_test.go`: "MUST have specific error assertions (ErrorContains, ErrorAs)".



Also applies to: 113-116, 149-152

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/extensions_additional_test.go` around lines 73 - 81, The
tests currently only assert HTTP 400 for invalid POST /api/extensions but not
the error semantics; update the negative-path checks that call performRequest
(variables missingPath and missingChecksum) to also assert the response
body/error payload contains the expected field names (e.g., "path" for
missingPath and "checksum" for missingChecksum) using the project's test helpers
(ErrorContains or equivalent string containment assertion) so the failure is
specific; apply the same pattern to the other two failing checks referenced (the
blocks at the other occurrences around the checks at lines noted: 113-116 and
149-152) to ensure each bad-request test verifies the concrete error message or
JSON key indicating the exact missing/unsupported field.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the invalid-request checks only assert HTTP status, so a handler can regress to the wrong validation message while still passing the tests.
- Fix approach: extend the negative-path assertions to verify the response body mentions the expected field or error semantics in addition to the status code.
