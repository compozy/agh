---
status: resolved
file: internal/memory/store_test.go
line: 682
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrX6,comment:PRRC_kwDOR5y4QM62twdX
---

# Issue 014: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Convert these helper assertions into table-driven subtests.**

This block still bundles multiple cases into one test, so failures are less isolated and it does not match the repo’s default test shape. Please move these assertions into `t.Run("Should ...")` table cases, and keep `t.Parallel()` on independent subtests.



<details>
<summary>Example refactor</summary>

```diff
-	if got := ageDays(today, now); got != 0 {
-		t.Fatalf("ageDays(today) = %d, want 0", got)
-	}
-	if got := ageDays(yesterday, now); got != 1 {
-		t.Fatalf("ageDays(yesterday) = %d, want 1", got)
-	}
-	if got := ageText(today, now); got != "today" {
-		t.Fatalf("ageText(today) = %q, want %q", got, "today")
-	}
-	if got := ageText(yesterday, now); got != "yesterday" {
-		t.Fatalf("ageText(yesterday) = %q, want %q", got, "yesterday")
-	}
-	if got := ageText(threeDaysAgo, now); got != "3 days ago" {
-		t.Fatalf("ageText(threeDaysAgo) = %q, want %q", got, "3 days ago")
-	}
-	if got := freshnessWarning(today, now); got != "" {
-		t.Fatalf("freshnessWarning(today) = %q, want empty", got)
-	}
-	if got := freshnessWarning(yesterday, now); got != "" {
-		t.Fatalf("freshnessWarning(yesterday) = %q, want empty", got)
-	}
-	if got := freshnessWarning(threeDaysAgo, now); !strings.Contains(got, "3 days old") {
-		t.Fatalf("freshnessWarning(threeDaysAgo) = %q, want age caveat", got)
-	}
+	tests := []struct {
+		name string
+		run  func(*testing.T)
+	}{
+		{
+			name: "Should return zero days for today",
+			run: func(t *testing.T) {
+				t.Parallel()
+				if got := ageDays(today, now); got != 0 {
+					t.Fatalf("ageDays(today) = %d, want 0", got)
+				}
+			},
+		},
+		{
+			name: "Should return one day for yesterday",
+			run: func(t *testing.T) {
+				t.Parallel()
+				if got := ageDays(yesterday, now); got != 1 {
+					t.Fatalf("ageDays(yesterday) = %d, want 1", got)
+				}
+			},
+		},
+		{
+			name: "Should render relative age text",
+			run: func(t *testing.T) {
+				t.Parallel()
+				if got := ageText(threeDaysAgo, now); got != "3 days ago" {
+					t.Fatalf("ageText(threeDaysAgo) = %q, want %q", got, "3 days ago")
+				}
+			},
+		},
+		{
+			name: "Should emit warning only for stale memories",
+			run: func(t *testing.T) {
+				t.Parallel()
+				if got := freshnessWarning(threeDaysAgo, now); !strings.Contains(got, "3 days old") {
+					t.Fatalf("freshnessWarning(threeDaysAgo) = %q, want age caveat", got)
+				}
+			},
+		},
+	}
+
+	for _, tt := range tests {
+		tt := tt
+		t.Run(tt.name, func(t *testing.T) {
+			tt.run(t)
+		})
+	}
```
</details>

As per coding guidelines, `**/*_test.go`: `Use table-driven tests with subtests (t.Run) as default in Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store_test.go` around lines 660 - 682, Split the block of
assertions into a table-driven set of subtests using t.Run("Should ...") entries
that each test one expectation for ageDays, ageText, and freshnessWarning;
create a testCases slice referencing the inputs (today, yesterday, threeDaysAgo,
now) and expected outputs, loop over it and for each case call t.Run with a
descriptive "Should ..." name, run t.Parallel() inside each subtest, and perform
the single assertion (use equality checks for ageDays/ageText and
strings.Contains for freshnessWarning's "3 days old" expectation) against the
functions ageDays, ageText, and freshnessWarning so failures are isolated.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: The `TestStalenessHelpers` assertions are bundled into one block, which weakens failure isolation and diverges from the workspace testing convention for table-driven subtests. This is a test-structure issue rather than a product bug, but it is a legitimate scoped cleanup request.
- Fix approach: Convert the block into `t.Run("Should ...")` table-driven subtests with one assertion per case while preserving the current expectations.
