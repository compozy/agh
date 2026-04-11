---
status: resolved
file: internal/automation/manager_test.go
line: 1128
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093889370,nitpick_hash:6bba2c226403
review_hash: 6bba2c226403
source_review_id: "4093889370"
source_review_submitted_at: "2026-04-11T14:58:56Z"
---

# Issue 008: Error comparison uses string matching instead of errors.Is().
## Review Comment

The `assertContextError` helper compares error messages using `err.Error() != want`, which is fragile and violates the coding guidelines. Consider using sentinel errors and `errors.Is()` for type-safe error matching, or at minimum use `strings.Contains()` if the exact message format might change.

As per coding guidelines: "Use errors.Is() and errors.As() for error matching — never compare error strings".

---

## Triage

- Decision: `invalid`
- Notes:
- The nil-context errors under test are created inline with fresh `errors.New(...)` calls in `internal/automation/manager.go`; they are not shared sentinel values.
- Converting this helper to `errors.Is()` would require a production API change outside this batch's scoped files. Replacing the exact comparison with substring matching would weaken the test and would be a workaround rather than a root-cause fix.
- Because the required production refactor is out of scope for this batch, the current exact contract assertions remain the correct choice here.
