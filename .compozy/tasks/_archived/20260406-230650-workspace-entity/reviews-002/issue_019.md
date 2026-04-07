---
status: resolved
file: internal/session/session.go
line: 248
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCc,comment:PRRC_kwDOR5y4QM61T6H3
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap the inactive-state branch with `ErrSessionNotActive`.**

`beginPromptSetup()` now returns a plain formatted error when the session is not active. Callers that rely on `errors.Is(err, ErrSessionNotActive)`—including the shared API status mapping added in this PR—won't match it, so prompt requests racing with a stop transition can be reported as generic internal failures instead of a normal session-state error.

<details>
<summary>🐛 Proposed fix</summary>

```diff
 	if s.State != StateActive {
-		return nil, fmt.Errorf("session: session %q is not active", s.ID)
+		return nil, fmt.Errorf("%w: %s", ErrSessionNotActive, s.ID)
 	}
```
</details>

As per coding guidelines, `**/*.go`: `Use explicit error returns with wrapped context: fmt.Errorf("context: %w", err)` and `Use errors.Is() and errors.As() for error matching — never compare error strings`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/session.go` around lines 227 - 248, The inactive-state
branch in beginPromptSetup currently returns a plain fmt.Errorf string which
prevents callers from matching ErrSessionNotActive; change the error return in
beginPromptSetup (the check comparing s.State to StateActive) to wrap
ErrSessionNotActive using the %w verb so callers can use errors.Is(err,
ErrSessionNotActive) — e.g. produce a formatted error that includes the session
ID and wraps ErrSessionNotActive; keep the rest of the function flow the same.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `beginPromptSetup` returns a plain formatted error when the session is not active.
  - That breaks `errors.Is(err, ErrSessionNotActive)` checks and can bypass the existing status mapping logic that relies on the sentinel.
  - I will wrap `ErrSessionNotActive` there and add a focused test that protects the error contract.
  - Test coverage for this requires touching `internal/session/session_test.go`, which is outside the listed batch files but is the minimal place to exercise `beginPromptSetup` directly.
