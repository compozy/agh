---
provider: coderabbit
pr: "108"
round: 2
round_created_at: 2026-05-06T04:43:32.489895Z
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 92
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2cOk,comment:PRRC_kwDOR5y4QM6-Uf9q
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the daemon status payload, not only the status code.**

This block closes the response immediately, so a handler returning the wrong JSON shape would still pass. Decode the body and assert at least one `daemon` field here.


As per coding guidelines "Always assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/httpapi/httpapi_integration_test.go` around lines 81 - 92, The
test currently only checks statusResp.StatusCode after calling mustHTTPRequest
for GET "/api/daemon/status" and immediately closes statusResp.Body, so it
doesn't validate the JSON payload; update the test in
httpapi_integration_test.go to read and decode statusResp.Body (from the
mustHTTPRequest call using runtime.client and runtime.host/runtime.port) into a
struct/map and assert that the response contains a "daemon" field (and any
expected subfields) before closing the body; use the existing statusResp
variable and fail the test with t.Fatalf or t.Errorf if decoding fails or the
"daemon" field is missing/incorrect.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the integration test checks only the HTTP status for `/api/daemon/status` and never validates the returned JSON contract.
- Evidence: the current test closes `statusResp.Body` immediately after asserting `200 OK`, so payload regressions would pass unnoticed.
- Fix plan: decode the daemon status payload and assert at least one expected daemon field before closing the response.
- Resolution: updated `internal/api/httpapi/httpapi_integration_test.go` to decode the response wrapper and assert populated `daemon` fields that the handler contract actually guarantees.
- Verification: targeted integration-tagged `go test` for `internal/api/httpapi` passed, and fresh `make verify` passed on 2026-05-06.
