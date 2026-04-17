---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 15
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:c79b6f2e1181
review_hash: c79b6f2e1181
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 011: Consider adopting t.Run("Should...") pattern throughout.
## Review Comment

The coding guidelines specify: "MUST use `t.Run("Should...")` pattern for ALL test cases." While the current tests verify meaningful behavior, restructuring with subtests would improve:
- Test output clarity (each subtest named descriptively)
- Ability to run specific scenarios (`go test -run TestX/Should_handle_Y`)
- Parallel execution of independent scenarios within a test

This applies to all test functions in this file.

As per coding guidelines: "MUST use t.Run('Should...') pattern for ALL test cases"

---

## Triage

- Decision: `INVALID`
- Reasoning: the repository guidance requires subtests by default for multi-scenario tests, but it does not require wrapping every standalone top-level test in an extra `t.Run("Should...")` shell. Blanket nesting here would add ceremony without strengthening coverage.
- Resolution: closed as non-actionable. The genuinely overgrown test in this file is handled separately in issue 012.
