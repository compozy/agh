---
status: resolved
file: internal/network/audit_test.go
line: 159
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562aok,comment:PRRC_kwDOR5y4QM63mgR5
---

# Issue 024: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap this new case in `t.Run("Should...")` (table-driven style default).**

The assertions are good, but this newly added test should follow the repository’s required subtest pattern for consistency with test review rules.  
As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (t.Run) as default in Go tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/audit_test.go` around lines 114 - 159, The new
TestAuditWriterRecordTaskIngress must be converted to use a subtest via t.Run to
follow the repository's required subtest pattern: wrap the existing test body
(setup of storeSink, NewAuditWriter, writer.now, the call to
writer.RecordTaskIngress, and all assertions on storeSink.entries[0] including
SessionID, Kind, Direction, Reason, and Size) inside a t.Run("Should record task
ingress", func(t *testing.T) { ... }) closure (and keep/relocate t.Parallel
inside the subtest if desired); ensure the same unique symbols
(TestAuditWriterRecordTaskIngress, writer.RecordTaskIngress, storeSink.entries)
are used unchanged, and keep all assertions intact inside the subtest.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  This test body was added as a single top-level case even though the repository standard for Go tests is subtests with `t.Run("Should ...")`.
  The behavior assertion itself is fine; the required change is structural. I will wrap the existing task-ingress assertions in a `Should ...` subtest so the file follows the enforced testing pattern consistently.
  Resolution: Wrapped the task-ingress assertions in `t.Run("Should record task ingress", ...)` without weakening the existing checks.
