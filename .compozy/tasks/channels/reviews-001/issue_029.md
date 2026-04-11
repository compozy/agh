---
status: resolved
file: internal/extension/describe_test.go
line: 45
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TBLx,comment:PRRC_kwDOR5y4QM623eJM
---

# Issue 029: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Refactor these tests to table-driven subtests using `t.Run("Should...")`.**

Both cases are currently standalone tests. Repo test standards require table-driven structure with `t.Run("Should...")` per case. Please consolidate these into a single table with named subtests (and keep parallelization at subtest level where independent).  
 

<details>
<summary>Proposed refactor shape</summary>

```diff
-func TestDescribeExtensionReportsActiveSubprocessRuntime(t *testing.T) {
-	t.Parallel()
-	...
-}
-
-func TestDescribeExtensionReportsRegisteredResourceHealth(t *testing.T) {
-	t.Parallel()
-	...
-}
+func TestDescribeExtension(t *testing.T) {
+	t.Parallel()
+
+	now := time.Date(2026, 4, 10, 18, 30, 0, 0, time.UTC)
+	tests := []struct {
+		name   string
+		ext    *Extension
+		active bool
+		now    time.Time
+		wantTy string
+		wantSt string
+		wantHl string
+		wantUp int64
+	}{
+		{
+			name: "Should report active subprocess runtime",
+			ext: &Extension{ /* existing subprocess fixture */ },
+			active: true, now: now,
+			wantTy: "subprocess", wantSt: "active", wantHl: "healthy", wantUp: 900,
+		},
+		{
+			name: "Should report registered resource health",
+			ext: &Extension{ /* existing resource fixture */ },
+			active: true, now: now,
+			wantTy: "resource", wantSt: "registered", wantHl: "healthy",
+		},
+	}
+
+	for _, tt := range tests {
+		tt := tt
+		t.Run(tt.name, func(t *testing.T) {
+			t.Parallel()
+			payload := DescribeExtension(tt.ext, tt.active, tt.now)
+			if payload.Type != tt.wantTy { t.Fatalf("Type = %q, want %q", payload.Type, tt.wantTy) }
+			if payload.State != tt.wantSt { t.Fatalf("State = %q, want %q", payload.State, tt.wantSt) }
+			if payload.Health != tt.wantHl { t.Fatalf("Health = %q, want %q", payload.Health, tt.wantHl) }
+			if tt.wantUp != 0 && payload.UptimeSeconds != tt.wantUp {
+				t.Fatalf("UptimeSeconds = %d, want %d", payload.UptimeSeconds, tt.wantUp)
+			}
+		})
+	}
+}
```
</details>

As per coding guidelines, "`**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default" and "`MUST use t.Run("Should...") pattern for ALL test cases`".


Also applies to: 47-71

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/describe_test.go` around lines 8 - 45, Refactor the two
standalone tests (e.g. TestDescribeExtensionReportsActiveSubprocessRuntime and
the other test around lines 47-71 that also calls DescribeExtension) into a
single table-driven test: create a slice of test cases with a descriptive Name
string starting with "Should..." and fields for input Extension, isSubprocess
bool, now time, and expected outputs (Type, State, Health, UptimeSeconds); then
iterate cases with for _, tc := range cases { tc := tc; t.Run(tc.Name, func(t
*testing.T) { t.Parallel(); payload := DescribeExtension(&tc.Extension,
tc.isSubprocess, tc.now); assert each expected field using the existing checks
(Type, State, Health, UptimeSeconds) }) } so each case runs as a subtest and
preserves parallelization and original assertions.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Why: The file has two standalone tests for the same function even though the repo's preferred pattern is table-driven `t.Run("Should...")` subtests for comparable cases.
- Root cause: The tests were added as separate functions instead of one table-driven structure.
- Fix plan: Consolidate the existing cases into a single table-driven test with parallel subtests and the same field-level assertions.
- Resolution: Converted the file to a table-driven `TestDescribeExtension` with `Should...` subtests and verified the package and repo gate.
