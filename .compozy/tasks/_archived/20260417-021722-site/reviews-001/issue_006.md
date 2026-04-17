---
status: resolved
file: internal/cli/doc_test.go
line: 197
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hC--,comment:PRRC_kwDOR5y4QM64gE3f
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# Check if the file exists and view its actual content
fd -t f "doc_test.go" | head -5
```

Repository: compozy/agh

Length of output: 79

---

🏁 Script executed:

```shell
# View the actual test file content around the flagged lines
wc -l internal/cli/doc_test.go
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
# Search for table-driven test patterns in other test files in the same directory
fd -t f "*_test.go" internal/cli/ | head -10
```

Repository: compozy/agh

Length of output: 343

---

🏁 Script executed:

```shell
# Look for patterns of how other tests are structured in this repo
rg -t go 't\.Run\(' internal/cli --max-count=5 -A 2
```

Repository: compozy/agh

Length of output: 6091

---

🏁 Script executed:

```shell
# Check for ignored error patterns in surrounding code
rg -t go 'filepath\.Rel.*_' --max-count=10
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
cat -n internal/cli/doc_test.go
```

Repository: compozy/agh

Length of output: 6651

---

🏁 Script executed:

```shell
# Check for ignored errors in doc_test.go specifically
rg -t go '_, ' internal/cli/doc_test.go -A 1 -B 1
```

Repository: compozy/agh

Length of output: 227

---

🏁 Script executed:

```shell
# Look at how errors are handled in the findMDX helper
sed -n '35,65p' internal/cli/doc_test.go
```

Repository: compozy/agh

Length of output: 845

---

🏁 Script executed:

```shell
# Check the help assertion context more closely
sed -n '18,30p' internal/cli/doc_test.go
```

Repository: compozy/agh

Length of output: 349

---



</details>

**Use table-driven tests with subtests and fix ignored errors.**

Tests must follow the `t.Run("Should...")` subtest pattern as the standard across this repository. Additionally, errors at lines 114 and 143 are discarded with `_` in `filepath.Rel()` calls; per coding standards, all errors must be handled explicitly or justified. Compare line 54-56 in the same file, which properly checks the error.

Line 25's assertion `strings.Contains(help, "doc")` is overly broad and may match unintended strings like "documentation"—use an exact pattern or word boundary check instead.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/doc_test.go` around lines 11 - 197, Convert each top-level test
(TestNewDocCommand_Hidden, TestNewDocCommand_NotInHelp,
TestNewDocCommand_DefaultOutputDir, TestNewDocCommand_GeneratesDocs,
TestNewDocCommand_CreatesOutputDir, TestNewDocCommand_GeneratesAllCommands) to
use a t.Run("Should ...") subtest wrapper and call t.Parallel() inside that
subtest; keep the same assertions but moved into the subtest closure. Replace
the broad strings.Contains(help, "doc") check in TestNewDocCommand_NotInHelp
with an exact-word check (e.g. use a regexp like
regexp.MustCompile(`\bdoc\b`).MatchString(help) or iterate root.Commands() and
check command.Use == "doc") to avoid matching substrings like "documentation".
Fix the ignored errors from filepath.Rel in the WalkDir closures in
TestNewDocCommand_GeneratesDocs (where rel, _ := filepath.Rel(outputDir, p)) and
the subsequent WalkDir (where rel, _ := filepath.Rel(outputDir, p)) by capturing
the error, handling it (t.Fatalf or returning the error) instead of discarding
it, similar to the error handling used in findMDX.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - The substantive defects in this comment are already covered by Issue 004 (over-broad help assertion) and Issue 005 (ignored `filepath.Rel` errors), and those will be fixed directly.
  - Wrapping each already-independent top-level test in a single-case subtest would add ceremony without improving coverage, failure localization, or behavior.
  - No separate code change is needed for the stylistic subtest request.
