---
status: resolved
file: cmd/agh-codegen/main_test.go
line: 48
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QU57,comment:PRRC_kwDOR5y4QM620Apq
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Refactor tests to required `t.Run("Should...")` subtest style**

Line 10, Line 24, and Line 38 define standalone cases instead of the required subtest pattern. Please convert these into table-driven subtests with `t.Run("Should...")`.

<details>
<summary>Suggested refactor</summary>

```diff
-func TestCheckJSONFileIgnoresFormattingDifferences(t *testing.T) {
-	t.Parallel()
-
-	path := filepath.Join(t.TempDir(), "openapi.json")
-	if err := os.WriteFile(path, []byte("{\n  \"z\": 1,\n  \"nested\": {\"b\": 2, \"a\": [1, 2]}\n}\n"), 0o644); err != nil {
-		t.Fatalf("os.WriteFile() error = %v", err)
-	}
-
-	want := []byte("{\"nested\":{\"a\":[1,2],\"b\":2},\"z\":1}")
-	if err := checkJSONFile(path, want); err != nil {
-		t.Fatalf("checkJSONFile() error = %v, want nil", err)
-	}
-}
-
-func TestCheckJSONFileRejectsContentDifferences(t *testing.T) {
+func TestCheckJSONFile(t *testing.T) {
 	t.Parallel()
 
-	path := filepath.Join(t.TempDir(), "openapi.json")
-	if err := os.WriteFile(path, []byte("{\"version\":1}\n"), 0o644); err != nil {
-		t.Fatalf("os.WriteFile() error = %v", err)
-	}
-
-	err := checkJSONFile(path, []byte("{\"version\":2}\n"))
-	if err == nil || !strings.Contains(err.Error(), "stale") {
-		t.Fatalf("checkJSONFile() error = %v, want stale", err)
-	}
+	tests := []struct {
+		name          string
+		fileContent   []byte
+		want          []byte
+		wantErrSubstr string
+	}{
+		{
+			name:        "ShouldIgnoreFormattingDifferences",
+			fileContent: []byte("{\n  \"z\": 1,\n  \"nested\": {\"b\": 2, \"a\": [1, 2]}\n}\n"),
+			want:        []byte("{\"nested\":{\"a\":[1,2],\"b\":2},\"z\":1}"),
+		},
+		{
+			name:          "ShouldRejectContentDifferences",
+			fileContent:   []byte("{\"version\":1}\n"),
+			want:          []byte("{\"version\":2}\n"),
+			wantErrSubstr: "stale",
+		},
+	}
+
+	for _, tt := range tests {
+		tt := tt
+		t.Run(tt.name, func(t *testing.T) {
+			t.Parallel()
+			path := filepath.Join(t.TempDir(), "openapi.json")
+			if err := os.WriteFile(path, tt.fileContent, 0o644); err != nil {
+				t.Fatalf("os.WriteFile() error = %v", err)
+			}
+
+			err := checkJSONFile(path, tt.want)
+			if tt.wantErrSubstr == "" && err != nil {
+				t.Fatalf("checkJSONFile() error = %v, want nil", err)
+			}
+			if tt.wantErrSubstr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr)) {
+				t.Fatalf("checkJSONFile() error = %v, want %q", err, tt.wantErrSubstr)
+			}
+		})
+	}
 }
 
 func TestFormatTypeScriptMatchesRepositoryFormatter(t *testing.T) {
 	t.Parallel()
-
-	formatted, err := formatTypeScript("sdk/typescript/src/generated/contracts.ts", []byte("export type Value =\n  | \"a\"\n  | \"b\";\n"))
-	if err != nil {
-		t.Fatalf("formatTypeScript() error = %v", err)
-	}
-	if got, want := string(formatted), "export type Value = \"a\" | \"b\";\n"; got != want {
-		t.Fatalf("formatTypeScript() = %q, want %q", got, want)
-	}
+	t.Run("ShouldMatchRepositoryFormatter", func(t *testing.T) {
+		t.Parallel()
+		formatted, err := formatTypeScript("sdk/typescript/src/generated/contracts.ts", []byte("export type Value =\n  | \"a\"\n  | \"b\";\n"))
+		if err != nil {
+			t.Fatalf("formatTypeScript() error = %v", err)
+		}
+		if got, want := string(formatted), "export type Value = \"a\" | \"b\";\n"; got != want {
+			t.Fatalf("formatTypeScript() = %q, want %q", got, want)
+		}
+	})
 }
```
</details>


As per coding guidelines, `**/*_test.go`: "Use table-driven tests with subtests (`t.Run`) as default" and "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@cmd/agh-codegen/main_test.go` around lines 10 - 48, Convert the three
standalone tests (TestCheckJSONFileIgnoresFormattingDifferences,
TestCheckJSONFileRejectsContentDifferences,
TestFormatTypeScriptMatchesRepositoryFormatter) into table-driven subtests using
t.Run with the "Should..." naming pattern; for each existing scenario wrap the
setup/assertion inside a t.Run("Should ...", func(t *testing.T){ ... }) and, if
multiple cases are combined, create a slice of cases and iterate calling
t.Run(case.name, func(t *testing.T){ ... }); keep the existing use of
checkJSONFile and formatTypeScript but move their invocation/assertions into the
subtest bodies and preserve t.Parallel() where appropriate (call t.Parallel()
inside each subtest rather than at top-level test if you want parallelism).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding is reasonable for this file. `cmd/agh-codegen/main_test.go` currently uses three one-off top-level tests where the repo guidance prefers subtests and table-driven structure by default.
  - Root cause: the initial test file was written in a minimal standalone style instead of the prevailing subtest pattern.
  - Fix approach: consolidate the JSON-file scenarios into a table-driven test with `t.Run("Should...")` cases and wrap the TypeScript formatter assertion in a `Should...` subtest while preserving the current behavior checks.
  - Resolution: implemented in `cmd/agh-codegen/main_test.go` and verified with focused package tests plus `make verify`.
