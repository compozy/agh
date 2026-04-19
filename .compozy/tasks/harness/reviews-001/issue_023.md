---
status: resolved
file: internal/memory/recall_test.go
line: 125
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM7,comment:PRRC_kwDOR5y4QM65IPEP
---

# Issue 023: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Adopt the required Go test-case structure (`t.Run("Should...")`).**

These new tests are mostly top-level single-case functions; project test rules require table-driven/subtest style with `t.Run("Should...")` naming as the default format.

<details>
<summary>Suggested structure (example)</summary>

```diff
-func TestNewRecallAugmenterReturnsOriginalMessageWhenSessionOrQueryIsEmpty(t *testing.T) {
-	t.Parallel()
-	...
-}
+func TestNewRecallAugmenter(t *testing.T) {
+	t.Parallel()
+	tests := []struct {
+		name    string
+		session *session.Session
+		message string
+		want    string
+	}{
+		{name: "Should return original message for nil session", session: nil, message: "hello", want: "hello"},
+		{name: "Should return original message for blank query", session: &session.Session{Type: session.SessionTypeUser}, message: "   ", want: "   "},
+	}
+	for _, tc := range tests {
+		tc := tc
+		t.Run(tc.name, func(t *testing.T) {
+			t.Parallel()
+			...
+		})
+	}
+}
```
</details>

  
As per coding guidelines: "Use table-driven tests with subtests (t.Run) as default in Go tests" and "MUST use t.Run(\"Should...\") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/recall_test.go` around lines 13 - 125, The test functions
TestNewRecallAugmenterReturnsOriginalMessageWhenSessionOrQueryIsEmpty,
TestNewRecallAugmenterPrependsRecallAndPreservesUserMessage, and
TestBuildRecallBlockSkipsZeroScoreEntriesAndCapsResults must use the project's
required subtest pattern; wrap each test body in a t.Run("Should ...", func(t
*testing.T){ ... }) with the descriptive "Should..." name (e.g., "Should return
original message when session or query is empty", "Should prepend recall and
preserve user message", "Should skip zero-score entries and cap results"), move
t.Parallel() into the subtest function, and keep all existing setup and
assertions (augmenter creation, store writes, buildRecallBlock assertions)
unchanged inside those t.Run blocks so behavior is preserved while conforming to
the t.Run("Should...") requirement.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the new recall tests are still top-level single-case functions without the repo-default `t.Run("Should ...")` structure. The behavior coverage is useful, but the file does not currently follow the project’s required Go test layout.
- Fix approach: Restructure the file into `t.Run("Should ...")` subtests while preserving the existing setups and assertions unchanged.
