---
status: resolved
file: internal/network/audit_test.go
line: 385
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQj,comment:PRRC_kwDOR5y4QM67VX7B
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap these new test cases in `t.Run("Should...")` subtests.**

These additions bypass the required subtest pattern used elsewhere in the suite, so they drift from the repo’s enforced test structure. As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.



Also applies to: 387-414

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/audit_test.go` around lines 334 - 385, The test functions
(e.g., TestAuditWriterCoalescesRepeatedGreetHeartbeatsInTimeline and the other
test at 387-414) are not using the required t.Run("Should...") subtest pattern;
wrap each test body in a t.Run call with a descriptive "Should..." name, move
t.Parallel() inside the subtest (call t.Parallel() at the start of the t.Run
closure), and keep the existing assertions and helper calls (e.g.,
NewAuditWriter, writer.now override, RecordSent, and checks against storeSink)
unchanged inside the subtest closure so the tests conform to the suite pattern.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The greet coalescing regressions in `internal/network/audit_test.go` were added as top-level tests instead of named subtests.
  - Root cause: the new scenarios bypass the file's subtest convention even though they are discrete behavior cases.
  - Fix plan: wrap the affected test bodies in `t.Run("Should...")` blocks and keep their existing helper setup/assertions intact.
