---
status: resolved
file: internal/session/manager.go
line: 341
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1o,comment:PRRC_kwDOR5y4QM67HMW2
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap supervision validation failures with constructor context.**

`applyRuntimeDefaults()` returns `m.supervision.Validate()` verbatim, so boot failures lose the `session:` context that the rest of `NewManager()` adds. Wrap that error before returning it.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/manager.go` around lines 338 - 341, applyRuntimeDefaults()
currently returns m.supervision.Validate() directly, which loses the "session:"
context from NewManager on failures; change applyRuntimeDefaults so after
setting defaults it captures the error from m.supervision.Validate() and returns
a wrapped error (e.g. fmt.Errorf("session: %w", err)) so validation failures
include the session constructor context; reference m.supervision,
aghconfig.SessionSupervisionConfig{},
aghconfig.DefaultSessionSupervisionConfig(), applyRuntimeDefaults and Validate()
when making the change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `applyRuntimeDefaults` returns `m.supervision.Validate()` directly, so invalid supervision configuration lacks the `session:` constructor context.
  - The fix is to wrap validation errors with `fmt.Errorf("session: %w", err)` and add constructor error coverage.
