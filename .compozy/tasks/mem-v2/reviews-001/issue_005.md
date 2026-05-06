---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1596
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isj,comment:PRRC_kwDOR5y4QM6-UFVm
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the 401 payload for invalid webhook signatures.**

Right now any `401` passes here, including an unrelated auth failure or the wrong error schema. Decode and check the error body so the signature-validation contract is actually covered. As per coding guidelines, "Assert both HTTP status code AND response body in tests; status-code-only assertions are insufficient".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/httpapi/httpapi_integration_test.go` around lines 1574 - 1596,
The test currently only checks for HTTP 401 but not the response payload; update
the invalid webhook assertion to read and decode the response body from
invalidResp (use io.ReadAll on invalidResp.Body and close it), unmarshal the
JSON into the project's error response shape (or a small local struct with
fields like Code and Message), and assert that the error payload indicates a
signature-validation failure (e.g., error code/message referencing invalid
webhook signature) for the request sent with core.WebhookSignatureHeader =
"sha256=deadbeef"; keep the status code assertion and add explicit checks on the
decoded fields to ensure the contract for signature validation is enforced.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The invalid webhook branch in `TestHTTPAutomationTriggersRoundTrip` still checks only `401` and closes the body without asserting the response payload.
  - The HTTP transport uses the shared `contract.ErrorPayload` shape, so the test should verify that the unauthorized response is specifically the invalid-signature path rather than any unrelated auth failure.
  - Fix approach: decode the error payload and assert that the message references the invalid webhook signature.
