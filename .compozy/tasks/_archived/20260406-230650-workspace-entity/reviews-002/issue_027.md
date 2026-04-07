---
status: resolved
file: internal/udsapi/helpers_test.go
line: 152
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoC2,comment:PRRC_kwDOR5y4QM61T6IZ
---

# Issue 027: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Same semantic inconsistency as httpapi helpers.**

`Register` returning `ErrWorkspaceNotFound` as default is semantically inconsistent (see comment on `internal/httpapi/helpers_test.go`).

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/udsapi/helpers_test.go` around lines 147 - 152, The stub method
Register currently returns workspacepkg.ErrWorkspaceNotFound by default which is
semantically wrong for an unimplemented stub; change the default return in
stubWorkspaceService.Register to return a clear "not implemented" error (e.g.
workspacepkg.ErrNotImplemented) instead of ErrWorkspaceNotFound, and if that
error constant/type doesn't exist add it to the workspacepkg package; keep the
existing behavior of calling s.registerFn when non-nil.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  An unconfigured `stubWorkspaceService.Register` currently returns the domain
  error `ErrWorkspaceNotFound`, which misreports a missing test double as a real
  workspace lookup failure. A loud local "not implemented" stub error is more
  accurate and keeps unexpected calls from masquerading as domain behavior.
  Plan: introduce a test-only sentinel error and use it for unconfigured stub
  methods.
