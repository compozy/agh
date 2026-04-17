---
status: resolved
file: internal/api/udsapi/transport_parity_integration_test.go
line: 84
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM570BHn,comment:PRRC_kwDOR5y4QM646Fv-
---

# Issue 001: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Handle `resp.Body.Close()` errors instead of discarding them.**

Line 83 ignores a returned error (`_ = resp.Body.Close()`), which can hide transport/IO failures in this path.



<details>
<summary>Proposed fix</summary>

```diff
 			body, readErr := io.ReadAll(resp.Body)
-			_ = resp.Body.Close()
+			closeErr := resp.Body.Close()
 			if readErr != nil {
 				return fmt.Errorf("read UDS approval response: %w", readErr)
 			}
+			if closeErr != nil {
+				return fmt.Errorf("close UDS approval response body: %w", closeErr)
+			}
 			if err := e2etest.ValidateUDSApprovalNotImplemented(resp.StatusCode, body); err != nil {
 				return err
 			}
```
</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
			body, readErr := io.ReadAll(resp.Body)
			closeErr := resp.Body.Close()
			if readErr != nil {
				return fmt.Errorf("read UDS approval response: %w", readErr)
			}
			if closeErr != nil {
				return fmt.Errorf("close UDS approval response body: %w", closeErr)
			}
			if err := e2etest.ValidateUDSApprovalNotImplemented(resp.StatusCode, body); err != nil {
				return err
			}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/udsapi/transport_parity_integration_test.go` around lines 82 -
84, The test currently discards the error returned by resp.Body.Close(); change
the code that calls resp.Body.Close() (near the variables body, readErr and
resp) to capture and handle the error instead of using `_ = resp.Body.Close()`:
call err := resp.Body.Close() and if err != nil report the failure (e.g.,
t.Fatalf or t.Errorf with a clear message including the error) so any
transport/IO close failures are surfaced in the test.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding matches the current test code: `resp.Body.Close()` is explicitly discarded after reading the approval response body.
  - In this path the callback already returns structured errors, so dropping the close error can hide a transport or response-body failure during the UDS approval-gap assertion.
  - Fix implemented: capture the close error, join it with the read error when both occur, and keep the existing not-implemented validation unchanged.
  - Verification: `go test -tags integration ./internal/api/udsapi -run TestUDSTransportApprovalRouteDocumentsNotImplementedGap -count=1` and full `make verify` both passed after the change.
