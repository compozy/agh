---
provider: coderabbit
pr: "105"
round: 6
round_created_at: 2026-05-06T03:03:04.040959Z
status: resolved
file: internal/daemon/network_e2e_assertions_test.go
line: 158
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_1nZK,comment:PRRC_kwDOR5y4QM6-TX8X
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Preserve the ability to assert empty routing fields.**

`optionalAuditFieldMatches()` currently makes `""` mean “don't care”. For these new routing fields, that means the helper cannot catch an audit row that wrongly carries `thread_id`, `direct_id`, or `work_id` when the test expects that field to be absent. Please switch this to a tri-state expectation (`*string`, or an explicit `check` flag) so tests can distinguish “ignore” from “must be empty”.
 

As per coding guidelines, "MUST test meaningful business logic, not trivial operations."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/network_e2e_assertions_test.go` around lines 152 - 158,
optionalAuditFieldMatches currently treats an empty want string as “don't care,”
which prevents tests from asserting that routing fields must be absent; change
the function signature (optionalAuditFieldMatches) to use a tri-state
expectation—e.g. accept a *string (nil = ignore, pointer to "" = must be empty,
pointer to "X" = must equal) or add an explicit check flag alongside the desired
string—then implement logic to: return true when want is nil (ignore), return
true only if strings.TrimSpace(got) == "" when want points to "", and otherwise
compare trimmed values; update all callers to pass nil/ptr values (or the flag)
so tests can distinguish ignore vs must-be-empty vs must-equal.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `optionalAuditFieldMatches()` currently uses an empty wanted string as "ignore this field", which means the helper cannot assert that `thread_id`, `direct_id`, `work_id`, or peer-routing fields must actually be absent. That weakens the audit assertions exactly where direct/thread routing regressions matter. I will switch the helper to a tri-state expectation so callers can distinguish ignore vs must-be-empty vs must-equal, then update the current assertions to pin the expected empty routing fields explicitly.
  Resolved by switching the audit helper to a tri-state pointer expectation and updating the direct-room assertions to require empty `thread_id` when appropriate. Fresh `make verify` passed afterward.
