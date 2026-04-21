---
status: resolved
file: internal/procutil/procutil_test.go
line: 86
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58jpRA,comment:PRRC_kwDOR5y4QM655WE5
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cd internal/procutil && ls -la && head -100 procutil_test.go
```

Repository: compozy/agh

Length of output: 2969

---



</details>

**Use required `t.Run("Should...")` subtests for these test cases.**

Both `TestStartedAtCurrentProcess` and `TestMatchesStartTimeCurrentProcess` contain meaningful behavioral checks but violate the repository's explicit requirement to use `t.Run()` subtests for all test cases. The first should wrap its checks in a single subtest; the second should convert to table-driven format with subtests for each scenario.

<details>
<summary>Proposed refactor</summary>

```diff
 func TestStartedAtCurrentProcess(t *testing.T) {
 	t.Parallel()
-
-	startedAt, err := StartedAt(os.Getpid())
-	if err != nil {
-		t.Fatalf("StartedAt(current pid) error = %v", err)
-	}
-	if startedAt.IsZero() {
-		t.Fatal("StartedAt(current pid) = zero, want non-zero start time")
-	}
-	if startedAt.After(time.Now().UTC().Add(time.Second)) {
-		t.Fatalf("StartedAt(current pid) = %v, want a past timestamp", startedAt)
-	}
+	t.Run("ShouldReturnNonZeroPastTimestampForCurrentProcess", func(t *testing.T) {
+		t.Parallel()
+		startedAt, err := StartedAt(os.Getpid())
+		if err != nil {
+			t.Fatalf("StartedAt(current pid) error = %v", err)
+		}
+		if startedAt.IsZero() {
+			t.Fatal("StartedAt(current pid) = zero, want non-zero start time")
+		}
+		if startedAt.After(time.Now().UTC().Add(time.Second)) {
+			t.Fatalf("StartedAt(current pid) = %v, want a past timestamp", startedAt)
+		}
+	})
 }
 
 func TestMatchesStartTimeCurrentProcess(t *testing.T) {
 	t.Parallel()
-
-	startedAt, err := StartedAt(os.Getpid())
-	if err != nil {
-		t.Fatalf("StartedAt(current pid) error = %v", err)
-	}
-	if !MatchesStartTime(os.Getpid(), startedAt) {
-		t.Fatalf("MatchesStartTime(current pid, %v) = false, want true", startedAt)
-	}
-	if MatchesStartTime(os.Getpid(), startedAt.Add(-time.Hour)) {
-		t.Fatalf("MatchesStartTime(current pid, mismatched start) = true, want false")
-	}
+	startedAt, err := StartedAt(os.Getpid())
+	if err != nil {
+		t.Fatalf("StartedAt(current pid) error = %v", err)
+	}
+	testCases := []struct {
+		name      string
+		input     time.Time
+		wantMatch bool
+	}{
+		{name: "ShouldMatchCurrentProcessStartTime", input: startedAt, wantMatch: true},
+		{name: "ShouldRejectMismatchedStartTime", input: startedAt.Add(-time.Hour), wantMatch: false},
+	}
+	for _, tc := range testCases {
+		tc := tc
+		t.Run(tc.name, func(t *testing.T) {
+			t.Parallel()
+			got := MatchesStartTime(os.Getpid(), tc.input)
+			if got != tc.wantMatch {
+				t.Fatalf("MatchesStartTime(current pid, %v) = %v, want %v", tc.input, got, tc.wantMatch)
+			}
+		})
+	}
 }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/procutil/procutil_test.go` around lines 58 - 86, Wrap
TestStartedAtCurrentProcess's assertions inside a single t.Run subtest (e.g.,
t.Run("ShouldReturnNonZeroPastStartTime", ...)) and call StartedAt(os.Getpid())
inside that subtest; convert TestMatchesStartTimeCurrentProcess into a
table-driven test that iterates scenarios (e.g., {"matches", startedAt, want
true}, {"mismatch", startedAt.Add(-time.Hour), want false}) and run each row as
its own t.Run subtest, invoking MatchesStartTime(os.Getpid(), ...) inside each
subtest and asserting the expected boolean result; keep references to StartedAt
and MatchesStartTime and preserve existing error handling/assert messages within
the new subtests.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestStartedAtCurrentProcess` and `TestMatchesStartTimeCurrentProcess` are meaningful coverage additions, but they bypass the repository's required `Should...` subtest structure.
- Fix plan: wrap the `StartedAt` assertions in a single subtest and convert the `MatchesStartTime` assertions into named table-driven subtests while preserving current behavior coverage.
- Resolution: refactored the current-process `StartedAt` and `MatchesStartTime` coverage into named `Should...` subtests, with table-driven cases for matching and mismatched start times.
- Verification: `go test ./internal/procutil` and `make verify` passed after the change.
