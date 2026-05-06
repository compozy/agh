---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/core/memory_services_test.go
line: 79
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isf,comment:PRRC_kwDOR5y4QM6-UFVe
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the HTTP status for every handler call.**

Only `statusResp` checks `resp.Code`; the remaining requests decode bodies immediately. A 4xx/5xx JSON error body can still decode into these structs and turn the failure into a misleading zero-value assertion. Add explicit `http.StatusOK` checks before decoding each response.

 

<details>
<summary>Suggested pattern</summary>

```diff
 failuresResp := performRequest(t, engine, http.MethodGet, "/memory/extractor/failures", nil)
+if failuresResp.Code != http.StatusOK {
+	t.Fatalf("failures status code = %d, want %d", failuresResp.Code, http.StatusOK)
+}
 var failuresPayload contract.MemoryExtractorFailuresResponse
 decodeJSON(t, failuresResp.Body.Bytes(), &failuresPayload)
```
</details>

As per coding guidelines "Always assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient".


Also applies to: 95-121, 152-170

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/memory_services_test.go` around lines 54 - 79, The tests
call performRequest and immediately decode bodies (variables failuresResp,
retryResp, drainResp) which can mask HTTP errors; before decoding each response
for the memory extractor endpoints ("/memory/extractor/failures",
"/memory/extractor/retry", "/memory/extractor/drain") assert resp.Code ==
http.StatusOK (use the response objects failuresResp, retryResp, drainResp) and
fail the test if not OK, then proceed to decode into
contract.MemoryExtractorFailuresResponse, contract.MemoryExtractorRetryResponse
and contract.MemoryExtractorDrainResponse; apply the same pattern to the other
test blocks referenced (lines ~95-121 and ~152-170) so every handler call checks
status code before decoding.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/core/memory_services_test.go` still decodes multiple responses immediately after `performRequest` without asserting `resp.Code` first.
  - That can hide handler regressions because the shared error payload may still deserialize into zero-value structs and keep the test misleadingly green.
  - Fix approach: add explicit `http.StatusOK` assertions before every decode in the extractor/provider/session-ledger handler tests that currently skip them.
