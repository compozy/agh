---
status: resolved
file: internal/acp/types_test.go
line: 138
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIeD,comment:PRRC_kwDOR5y4QM66CAki
---

# Issue 003: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap the new cases in the required `t.Run("Should...")` form.**

These additions are useful, but they introduce more top-level tests instead of the required subtest style. Please move them under `t.Run("Should defer tool results until the tool call")` / `t.Run("Should flush deferred tool results before done")` so this file stays consistent with the enforced Go test pattern.


As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default pattern for Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/types_test.go` around lines 89 - 138, Wrap each new top-level
test body in a t.Run subtest using the required "Should..." titles: inside
TestEmitPromptEventDefersToolResultUntilToolCall call t.Run("Should defer tool
results until the tool call", func(t *testing.T) { t.Parallel(); /* move the
current body here */ }) and inside
TestEmitPromptEventFlushesDeferredToolResultsBeforeDone call t.Run("Should flush
deferred tool results before done", func(t *testing.T) { t.Parallel(); /* move
the current body here */ }), preserving the use of AgentProcess, beginPrompt,
emitPromptEvent and reads from active.events unchanged; ensure t.Parallel() is
invoked inside the subtest functions (not only at the top) so the tests follow
the project's t.Run("Should...") subtest pattern.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Reasoning: the two new ACP tests are top-level one-off bodies instead of the repo's required `t.Run("Should...")` structure. Bringing them into named subtests keeps the file aligned with the enforced Go test pattern and makes failures easier to localize.
- Fix plan: wrap each top-level body in a `t.Run("Should...")` subtest with its own `t.Parallel()` while preserving the existing assertions.
- Resolution: wrapped the ACP regression coverage in named `t.Run("Should...")` subtests so failures stay localized and the file matches the repo's Go test pattern.
- Verification: `go test ./internal/acp` and `make verify`
