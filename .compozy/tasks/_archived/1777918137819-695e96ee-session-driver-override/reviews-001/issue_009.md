---
status: resolved
file: internal/procutil/process_group_unix_test.go
line: 41
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581azQ,comment:PRRC_kwDOR5y4QM66RFO3
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Convert to table-driven subtests using `t.Run("Should...")`.**

These cases are valid, but the current shape (three top-level tests) violates the repo’s required Go test pattern.


<details>
<summary>✅ Suggested refactor</summary>

```diff
-func TestJoinProcessGroupKillResultSuppressesEPERMWhenWaitSucceeds(t *testing.T) {
-	t.Parallel()
-
-	signalErr := fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM)
-	if err := joinProcessGroupKillResult(signalErr, nil); err != nil {
-		t.Fatalf("joinProcessGroupKillResult(EPERM, nil) error = %v, want nil", err)
-	}
-}
-
-func TestJoinProcessGroupKillResultPreservesWaitFailure(t *testing.T) {
-	t.Parallel()
-
-	waitErr := errors.New("wait for process group exit: deadline exceeded")
-	signalErr := fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM)
-
-	err := joinProcessGroupKillResult(signalErr, waitErr)
-	if !errors.Is(err, waitErr) {
-		t.Fatalf("joinProcessGroupKillResult(EPERM, waitErr) = %v, want wrapped waitErr", err)
-	}
-}
-
-func TestJoinProcessGroupKillResultPreservesNonEPERMSignalFailure(t *testing.T) {
-	t.Parallel()
-
-	signalErr := fmt.Errorf("signal process group members: %w", syscall.ESRCH)
-	err := joinProcessGroupKillResult(signalErr, nil)
-	if !errors.Is(err, syscall.ESRCH) {
-		t.Fatalf("joinProcessGroupKillResult(ESRCH, nil) = %v, want wrapped ESRCH", err)
-	}
-}
+func TestJoinProcessGroupKillResult(t *testing.T) {
+	t.Parallel()
+
+	waitErr := errors.New("wait for process group exit: deadline exceeded")
+	tests := []struct {
+		name      string
+		signalErr error
+		waitErr   error
+		wantNil   bool
+		wantIs    []error
+	}{
+		{
+			name:      "Should suppress EPERM when wait succeeds",
+			signalErr: fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM),
+			waitErr:   nil,
+			wantNil:   true,
+		},
+		{
+			name:      "Should preserve wait failure when signal returns EPERM",
+			signalErr: fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM),
+			waitErr:   waitErr,
+			wantIs:    []error{waitErr},
+		},
+		{
+			name:      "Should preserve non-EPERM signal failure",
+			signalErr: fmt.Errorf("signal process group members: %w", syscall.ESRCH),
+			waitErr:   nil,
+			wantIs:    []error{syscall.ESRCH},
+		},
+	}
+
+	for _, tt := range tests {
+		tt := tt
+		t.Run(tt.name, func(t *testing.T) {
+			t.Parallel()
+
+			err := joinProcessGroupKillResult(tt.signalErr, tt.waitErr)
+			if tt.wantNil {
+				if err != nil {
+					t.Fatalf("joinProcessGroupKillResult() error = %v, want nil", err)
+				}
+				return
+			}
+			for _, target := range tt.wantIs {
+				if !errors.Is(err, target) {
+					t.Fatalf("joinProcessGroupKillResult() error = %v, want wrapped %v", err, target)
+				}
+			}
+		})
+	}
+}
```
</details>

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern for Go tests" and "MUST use t.Run("Should...") pattern for ALL test cases".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestJoinProcessGroupKillResult(t *testing.T) {
	t.Parallel()

	waitErr := errors.New("wait for process group exit: deadline exceeded")
	tests := []struct {
		name      string
		signalErr error
		waitErr   error
		wantNil   bool
		wantIs    []error
	}{
		{
			name:      "Should suppress EPERM when wait succeeds",
			signalErr: fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM),
			waitErr:   nil,
			wantNil:   true,
		},
		{
			name:      "Should preserve wait failure when signal returns EPERM",
			signalErr: fmt.Errorf("signal process group (pid 123, sig killed): %w", syscall.EPERM),
			waitErr:   waitErr,
			wantIs:    []error{waitErr},
		},
		{
			name:      "Should preserve non-EPERM signal failure",
			signalErr: fmt.Errorf("signal process group members: %w", syscall.ESRCH),
			waitErr:   nil,
			wantIs:    []error{syscall.ESRCH},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := joinProcessGroupKillResult(tt.signalErr, tt.waitErr)
			if tt.wantNil {
				if err != nil {
					t.Fatalf("joinProcessGroupKillResult() error = %v, want nil", err)
				}
				return
			}
			for _, target := range tt.wantIs {
				if !errors.Is(err, target) {
					t.Fatalf("joinProcessGroupKillResult() error = %v, want wrapped %v", err, target)
				}
			}
		})
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/procutil/process_group_unix_test.go` around lines 12 - 41, Convert
the three top-level tests into a single table-driven test that iterates over
cases and uses t.Run("Should ...") subtests; keep the overall Test... function
and call t.Parallel() once, then for each case (name like "Should suppress EPERM
when wait succeeds", "Should preserve wait failure", "Should preserve non-EPERM
signal failure") run t.Run(caseName, func(t *testing.T){ t.Parallel(); call
joinProcessGroupKillResult with the case's signalErr and waitErr and assert
using errors.Is or nil checks as in the originals }); reference the existing
symbols joinProcessGroupKillResult, syscall.EPERM, syscall.ESRCH, and use the
same expected behaviors (nil, wrapped waitErr, wrapped ESRCH) in the table
entries.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the new `joinProcessGroupKillResult` coverage is split into three top-level tests instead of one scenario table, which conflicts with the repo's required Go test pattern.
- Fix plan: collapse the cases into a single table-driven test with `t.Run("Should ...")` subtests and the same `errors.Is` assertions.
