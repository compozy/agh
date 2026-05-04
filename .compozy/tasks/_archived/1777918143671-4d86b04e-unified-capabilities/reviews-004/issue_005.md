---
status: resolved
file: internal/codegen/openapits/generate_test.go
line: 109
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58jpP8,comment:PRRC_kwDOR5y4QM655WDp
---

# Issue 005: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify brittle error-text assertions in Go tests.
rg -n --type go 'strings\.Contains\(\s*err\.Error\(\)' -C2
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
# Read the test file to see lines around 103-109 and 146-152
wc -l internal/codegen/openapits/generate_test.go
```

Repository: compozy/agh

Length of output: 102

---

🏁 Script executed:

```shell
# Read specific sections of the test file
sed -n '95,115p' internal/codegen/openapits/generate_test.go
sed -n '140,160p' internal/codegen/openapits/generate_test.go
```

Repository: compozy/agh

Length of output: 1253

---

🏁 Script executed:

```shell
# Check the corresponding implementation file for error handling
ls -la internal/codegen/openapits/generate.go
```

Repository: compozy/agh

Length of output: 142

---

🏁 Script executed:

```shell
# Search for error types and Check/checkGeneratedFile function implementations
grep -n "func Check\|func checkGeneratedFile\|var.*Err\|type.*Error" internal/codegen/openapits/generate.go | head -20
```

Repository: compozy/agh

Length of output: 252

---

🏁 Script executed:

```shell
# Read the full generate.go file to see all error handling
cat -n internal/codegen/openapits/generate.go
```

Repository: compozy/agh

Length of output: 3881

---

🏁 Script executed:

```shell
# Also check if there are other test files that test similar error scenarios
grep -r "is missing; run codegen" internal/codegen/ --include="*.go"
```

Repository: compozy/agh

Length of output: 368

---



</details>

**Replace missing-output string checks with typed error assertions.**

The code currently returns `fmt.Errorf("%s is missing; run codegen", path)` without wrapping a sentinel error, making test assertions brittle. Use `errors.Is()` with a sentinel (e.g., `ErrMissingGeneratedFile`) to align with the guideline: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings."

This requires both a production code change (add and wrap sentinel) and test updates.

<details>
<summary>Suggested changes</summary>

In `generate.go`, add the sentinel:
```go
var ErrMissingGeneratedFile = errors.New("generated file is missing")
```

Then wrap it at line 76:
```go
if os.IsNotExist(err) {
    return fmt.Errorf("%s: %w", path, ErrMissingGeneratedFile)
}
```

In `generate_test.go`, replace string assertions:
```diff
- if !strings.Contains(err.Error(), "is missing; run codegen") {
-   t.Fatalf("Check() error = %v, want missing output guidance", err)
- }
+ if !errors.Is(err, ErrMissingGeneratedFile) {
+   t.Fatalf("Check() error = %v, want ErrMissingGeneratedFile", err)
+ }
```

Apply the same change to line 150 in `TestCheckGeneratedFile`.
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/codegen/openapits/generate_test.go` around lines 103 - 109, Add a
sentinel error and wrap filesystem-not-found errors, then update tests to use
errors.Is: declare a package-level var ErrMissingGeneratedFile =
errors.New("generated file is missing") in the generate.go file, change the
os.IsNotExist(err) return to wrap that sentinel (e.g., return fmt.Errorf("%s:
%w", path, ErrMissingGeneratedFile)) in the code path used by Check, and update
the tests in generate_test.go (the TestCheck... assertions around
Check(context.Background(), artifact) and the TestCheckGeneratedFile case) to
use errors.Is(err, ErrMissingGeneratedFile) (and remove string-based contains
checks) so tests assert the typed wrapped error instead of matching substrings.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the missing-generated-output assertions in `generate_test.go` currently depend on `strings.Contains(err.Error(), ...)`, which is brittle and tied to the untyped error from `checkGeneratedFile()`.
- Fix plan: once the missing-file sentinel exists, update both missing-output tests to assert `errors.Is(err, ErrMissingGeneratedFile)` instead of matching substrings.
- Resolution: updated the missing-output tests in both `Check()` and `checkGeneratedFile()` to assert `errors.Is(err, ErrMissingGeneratedFile)` instead of matching error text.
- Verification: `go test ./internal/codegen/openapits` and `make verify` passed after the change.
