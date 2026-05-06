---
provider: coderabbit
pr: "106"
round: 2
round_created_at: 2026-05-06T05:52:55.253953Z
status: resolved
file: internal/session/query_test.go
line: 557
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3H-Y,comment:PRRC_kwDOR5y4QM6-VcC0
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Rename this subtest to the required `Should ...` form.**

Please align the new case with the repository test naming convention, e.g. `t.Run("Should wait for finalization before reopening stored events", ...)`.
 
As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/session/query_test.go` at line 557, Rename the t.Run subtest whose
current title starts with "finalizing active session waits before reading a
closed recorder handle" to follow the repository convention using the "Should
..." pattern (for example: t.Run("Should wait for finalization before reopening
stored events", func(t *testing.T) { ... })), updating the test title string in
the t.Run call in internal/session/query_test.go so it matches other subtests
and the required naming convention.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the new subtest title in `internal/session/query_test.go` does not follow the repository-mandated `Should ...` naming convention.
- Fix approach: Rename the subtest to the required form while updating the surrounding assertions for the blocking proof.
