---
status: resolved
file: internal/store/session_liveness_test.go
line: 27
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RcPU,comment:PRRC_kwDOR5y4QM6628D-
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use specific error assertions for failure-path tests.**

The current `err == nil` checks are too weak; they can pass on the wrong failure reason. Please assert concrete error semantics per case (prefer sentinel/typed errors with `errors.Is` / `errors.As`).

As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)" and "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings".


Also applies to: 35-36, 44-45, 53-54, 148-149, 157-158

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/session_liveness_test.go` around lines 26 - 27, Replace weak
nil checks in the session_liveness tests with concrete error assertions: for
each call like meta.Validate() (and the other failing-case calls flagged),
import "errors" and assert errors.Is(err, <sentinelError>) or errors.As(err,
&<typedError>) instead of err == nil; if sentinel/typed errors (e.g.,
ErrInvalidPID, ErrEmptySessionID, ErrInvalidTTL) don’t yet exist, add well-named
package-level sentinel errors or typed error types in the same package and
return them from Validate()/the validated constructors so the tests can use
errors.Is / errors.As to match the exact failure reason. Ensure every
negative-path assertion references the correct sentinel/type that corresponds to
the specific validation rule being tested.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Decision: `invalid`
- Notes: The weak `err == nil` checks are real test-quality debt, but fixing them strictly with `errors.Is`/`errors.As` would require introducing new sentinel or typed error contracts in shared validation code outside this batch (`internal/store/session_liveness.go`, `internal/store/validation.go`, and `internal/hooks/types.go`). That broader error-API expansion is not necessary to preserve the behavior introduced by this PR and falls outside the scoped remediation set for this run.
