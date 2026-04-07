---
status: resolved
file: internal/udsapi/helpers_test.go
line: 194
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoC4,comment:PRRC_kwDOR5y4QM61T6Ia
---

# Issue 029: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Same consideration for `ResolveOrRegister` default.**

Same issue as httpapi helpers — `ResolveOrRegister` creating workspaces shouldn't default to "not found".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/udsapi/helpers_test.go` around lines 189 - 194, The
stubWorkspaceService.ResolveOrRegister default currently returns
workspacepkg.ErrWorkspaceNotFound which makes tests unable to simulate
successful registration; modify the default branch in
stubWorkspaceService.ResolveOrRegister (the path where resolveOrRegisterFn ==
nil) to return a constructed workspacepkg.ResolvedWorkspace representing a
newly-registered workspace (e.g., set Path to the incoming path and any minimal
required fields such as ID) and a nil error instead of the ErrWorkspaceNotFound;
keep the resolveOrRegisterFn override behavior unchanged.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  `ResolveOrRegister` has the same unconfigured-stub problem as `Register`:
  returning `ErrWorkspaceNotFound` hides missing test wiring. The suggested
  synthetic success would be worse because it would mask unexpected calls.
  Plan: make the default path fail loudly with the same local test-only
  "not implemented" sentinel instead.
