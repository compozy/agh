---
status: resolved
file: internal/diagnostics/redact_test.go
line: 40
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLia,comment:PRRC_kwDOR5y4QM67SmDf
---

# Issue 008: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Convert looped `maxBytes` cases into per-case subtests.**

On Line 36, the loop creates multiple test cases without `t.Run("Should...")`, so failures are less isolated and it misses the required test pattern.

<details>
<summary>Proposed refactor</summary>

```diff
 import (
+	"fmt"
	"strings"
	"testing"
 )
@@
	t.Run("Should keep non positive byte budgets bounded", func(t *testing.T) {
		t.Parallel()

		for _, maxBytes := range []int{0, -1} {
-			if got := RedactAndBound("token=super-secret", maxBytes); got != "" {
-				t.Fatalf("RedactAndBound(maxBytes=%d) = %q, want empty bounded result", maxBytes, got)
-			}
+			maxBytes := maxBytes
+			t.Run(fmt.Sprintf("Should return empty bounded result for maxBytes=%d", maxBytes), func(t *testing.T) {
+				t.Parallel()
+				if got := RedactAndBound("token=super-secret", maxBytes); got != "" {
+					t.Fatalf("RedactAndBound(maxBytes=%d) = %q, want empty bounded result", maxBytes, got)
+				}
+			})
		}
	})
```
</details>

As per coding guidelines: `MUST use t.Run("Should...") pattern for ALL test cases` and `Use table-driven tests with subtests (t.Run) as default pattern for Go tests`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/diagnostics/redact_test.go` around lines 36 - 40, The loop over
maxBytes in the test should be converted into table-driven subtests that call
t.Run for each case so failures are isolated; replace the for _, maxBytes :=
range []int{0, -1} loop with a slice of test cases (e.g., name and maxBytes) and
run each case with t.Run("Should ...", func(t *testing.T) { if got :=
RedactAndBound("token=super-secret", tc.maxBytes); got != "" { t.Fatalf(...) }
}), ensuring each subtest name follows the "Should..." pattern and references
RedactAndBound so results are checked per-case.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestRedactHandlesQuotedJSONSecretsAndBounds` loops over multiple `maxBytes` values inside one subtest, so each case lacks isolated `t.Run("Should...")` failure reporting.
- Fix approach: convert the non-positive byte-budget cases into table-driven subtests with names that start with `Should`.
