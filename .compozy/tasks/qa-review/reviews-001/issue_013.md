---
status: resolved
file: internal/store/sessiondb/session_db_test.go
line: 93
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQv,comment:PRRC_kwDOR5y4QM67VX7O
---

# Issue 013: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Move this new regression case under a `t.Run("Should...")` subtest.**

This scenario was added as a standalone test, but the repository test rules require the `t.Run("Should...")` pattern for test cases. As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/sessiondb/session_db_test.go` around lines 92 - 93, The test
function TestOpenSessionDBStripsCanonicalRawPayloadsAndVacuumsOldRows must be
converted into a subtest using t.Run("Should ...") to comply with test
conventions: wrap the existing body of
TestOpenSessionDBStripsCanonicalRawPayloadsAndVacuumsOldRows inside t.Run with a
descriptive "Should ..." name and move t.Parallel() into the subtest (or keep it
at the top-level if desired) so the test logic and assertions remain identical
but execute as a properly named subtest; update any references to local
variables accordingly within the t.Run closure.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TestOpenSessionDBStripsCanonicalRawPayloadsAndVacuumsOldRows` is a standalone top-level regression instead of a named subtest.
  - Root cause: the new migration/vacuum scenario was added directly rather than under the repo's default `t.Run("Should...")` pattern.
  - Fix plan: wrap the existing body in a named subtest and colocate any new vacuum-failure regression under the same parent test file.
