---
status: resolved
file: internal/registry/source_test.go
line: 25
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WU_,comment:PRRC_kwDOR5y4QM63maeT
---

# Issue 014: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Refactor tests to table-driven subtests with `t.Run("Should...")`.**

Current tests are direct top-level cases; this misses the repository’s required test shape for `*_test.go` files.

<details>
<summary>♻️ Proposed refactor</summary>

```diff
-func TestSourceCapsZeroValue(t *testing.T) {
-	t.Parallel()
-
-	var caps SourceCaps
-	if caps.Search {
-		t.Fatal("SourceCaps zero value Search = true, want false")
-	}
-}
-
-func TestErrNotSupportedMatchesWrappedError(t *testing.T) {
-	t.Parallel()
-
-	err := fmt.Errorf("wrapped: %w", ErrNotSupported)
-	if !errors.Is(err, ErrNotSupported) {
-		t.Fatalf("errors.Is(%v, ErrNotSupported) = false, want true", err)
-	}
-}
+func TestRegistrySourcePrimitives(t *testing.T) {
+	t.Parallel()
+
+	tests := []struct {
+		name string
+		run  func(t *testing.T)
+	}{
+		{
+			name: "Should keep SourceCaps zero value search disabled",
+			run: func(t *testing.T) {
+				var caps SourceCaps
+				if caps.Search {
+					t.Fatal("SourceCaps zero value Search = true, want false")
+				}
+			},
+		},
+		{
+			name: "Should match ErrNotSupported when wrapped",
+			run: func(t *testing.T) {
+				err := fmt.Errorf("wrapped: %w", ErrNotSupported)
+				if !errors.Is(err, ErrNotSupported) {
+					t.Fatalf("errors.Is(%v, ErrNotSupported) = false, want true", err)
+				}
+			},
+		},
+	}
+
+	for _, tt := range tests {
+		tt := tt
+		t.Run(tt.name, func(t *testing.T) {
+			t.Parallel()
+			tt.run(t)
+		})
+	}
+}
```
</details>


As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/source_test.go` around lines 9 - 25, Refactor the two tests
TestSourceCapsZeroValue and TestErrNotSupportedMatchesWrappedError into
table-driven subtests using the t.Run("Should...") pattern: create a slice of
test cases (with name and inputs/expected) for each test, iterate over them and
call t.Run("Should <describe expectation>", func(t *testing.T) { t.Parallel();
...assertions... }), keeping the existing checks (zero-value Search boolean and
errors.Is behavior) but moving them into the subtest bodies; reference the
existing test identifiers TestSourceCapsZeroValue and
TestErrNotSupportedMatchesWrappedError to locate where to replace the direct
checks with the table-driven t.Run subtests.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `internal/registry/source_test.go` currently uses two direct top-level tests for simple primitives. Converting them to a small table-driven `t.Run("Should...")` shape keeps the assertions identical while aligning the file with the repository’s test structure conventions. I will refactor those tests in-place.
- Resolution: Refactored `internal/registry/source_test.go` into table-driven `t.Run("Should...")` subtests while preserving the original assertions.
- Verification: `go test ./internal/registry/...`; `make verify`
