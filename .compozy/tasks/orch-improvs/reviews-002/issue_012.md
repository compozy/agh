---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/situation/service_test.go
line: 379
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-a,comment:PRRC_kwDOR5y4QM6-VcC4
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the redacted reviewer reason positively.**

Line 377 only proves the value is not `"review_rejected"`. An empty or unrelated reason still passes, so this case will miss regressions where the reviewer rationale stops being propagated or redacted correctly. Please assert the expected redacted value, or at least require `bundle.ReviewContinuation.Reason` to contain the redaction marker and exclude the raw token.

 
As per coding guidelines, "Verify tests can fail when business logic changes."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/situation/service_test.go` around lines 377 - 379, The current test
only checks that bundle.ReviewContinuation.Reason is not "review_rejected";
update the assertion to verify the redacted reviewer reason positively by
asserting the exact expected redacted string or at minimum assert that
bundle.ReviewContinuation.Reason contains the redaction marker (e.g.,
"[REDACTED]") and does not contain the raw reviewer token; locate the assertion
around the ReviewContinuation.Reason check in service_test.go and replace the
negative equality check with a positive containment/exclusion check against the
redaction marker and the raw token.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the test only asserts that the continuation reason is not the sentinel value, which does not prove reviewer rationale redaction still works.
- Fix approach: Assert the redaction marker positively and ensure the raw secret never appears in the reason string.
