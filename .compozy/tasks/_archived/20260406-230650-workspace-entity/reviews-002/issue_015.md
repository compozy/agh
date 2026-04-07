---
status: resolved
file: internal/httpapi/helpers_test.go
line: 203
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCR,comment:PRRC_kwDOR5y4QM61T6Hr
---

# Issue 015: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Same consideration for `ResolveOrRegister` default.**

`ResolveOrRegister` is designed to create a workspace if it doesn't exist, so defaulting to `ErrWorkspaceNotFound` doesn't align with its intended behavior.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/httpapi/helpers_test.go` around lines 198 - 203, The stub's
ResolveOrRegister currently returns workspacepkg.ErrWorkspaceNotFound by default
which contradicts its purpose to create/register a workspace; change the default
branch in stubWorkspaceService.ResolveOrRegister to return a newly created
workspacepkg.ResolvedWorkspace for the supplied path (i.e., a ResolvedWorkspace
instance representing the created workspace tied to the path) and nil error
instead of ErrWorkspaceNotFound so tests exercising implicit registration behave
correctly.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - For the same reason as issue 014, the fallback should fail loudly when a test does not wire `resolveOrRegisterFn`.
  - Returning a synthesized success value here would hide unexpected resolver calls and reduce the usefulness of the stub in negative-path tests.
  - Keeping `ErrWorkspaceNotFound` is the safer default for this test double.
