---
status: resolved
file: internal/udsapi/helpers_test.go
line: 194
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoC1,comment:PRRC_kwDOR5y4QM61T6IW
---

# Issue 028: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Consider extracting shared test stubs.**

The `stubWorkspaceService` implementation is duplicated between `httpapi/helpers_test.go` and `udsapi/helpers_test.go`. Consider extracting to an `internal/testutil` package or using a shared test helper file to reduce duplication and ensure consistent behavior.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/udsapi/helpers_test.go` around lines 137 - 194, The
stubWorkspaceService test double is duplicated across httpapi/helpers_test.go
and udsapi/helpers_test.go; extract it into a shared test helper (e.g.,
internal/testutil) and update callers to import and use the shared stub. Move
the type stubWorkspaceService and its methods (Register, Unregister, Update,
List, Get, Resolve, ResolveOrRegister) into the new package, keep the same
exported or package-visible names, and update both test files to reference the
centralized stub to remove duplication and keep behavior consistent.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  This is a cross-package cleanup suggestion, not a correctness defect in the
  scoped udsapi files. Fixing it would require unrelated changes in
  `internal/httpapi/helpers_test.go`, which is outside this batch, and would add
  refactor churn without affecting runtime behavior or verification. No change.
