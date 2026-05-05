---
status: resolved
file: internal/cli/agent_identity.go
line: 45
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsU,comment:PRRC_kwDOR5y4QM67YHCu
---

# Issue 023: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap the session lookup failure with session context.**

Returning `client.GetSession` errors verbatim loses which lookup failed and makes CLI failures harder to diagnose. Wrap this with the `sessionID` so the caller gets actionable context.


As per coding guidelines, `**/*.go`: "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)` in Go".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/agent_identity.go` around lines 43 - 45, The call to
client.GetSession returns errors without context; update the error return in the
function that calls client.GetSession (returning
agentidentity.SessionSnapshot{}) to wrap the original error with the sessionID
for actionable context (use fmt.Errorf with "%s: %w" or similar), e.g. wrap the
error from client.GetSession(ctx, sessionID) so the caller sees which session
lookup failed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `agentSessionLookup` returns `client.GetSession` errors without adding the session id being resolved. That loses actionable CLI context and violates the repository's wrapped-error convention.
- Fix: Wrap lookup failures with `fmt.Errorf("cli: lookup agent session %q: %w", sessionID, err)`.
