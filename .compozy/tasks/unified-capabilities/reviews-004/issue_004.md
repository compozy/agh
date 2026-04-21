---
status: resolved
file: internal/codegen/openapits/generate.go
line: 77
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58jpQC,comment:PRRC_kwDOR5y4QM655WDx
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify string-based error matching and os.IsNotExist usage in Go code.
rg -n --type go 'os\.IsNotExist\(|err\.Error\(\)|strings\.Contains\(\s*err\.Error\(' -C2
```

Repository: compozy/agh

Length of output: 50368

---

🏁 Script executed:

```shell
cat -n internal/codegen/openapits/generate.go | head -100
```

Repository: compozy/agh

Length of output: 3632

---



</details>

**Replace `os.IsNotExist()` with `errors.Is()` and wrap missing-file error with sentinel.**

Line 75 uses the deprecated `os.IsNotExist()` pattern; use `errors.Is(err, os.ErrNotExist)` instead. Line 76 returns a plain-text error, which forces string matching in callers. Introduce a sentinel error `ErrMissingGeneratedFile` (following the pattern of `ErrStaleGeneratedFile` at line 21) and wrap it for robust error checking.

<details>
<summary>Proposed fix</summary>

```diff
 var ErrStaleGeneratedFile = errors.New("generated file is stale")
+var ErrMissingGeneratedFile = errors.New("generated file is missing")
 
 func checkGeneratedFile(path string, want []byte) error {
 	got, err := os.ReadFile(path)
 	if err != nil {
-		if os.IsNotExist(err) {
-			return fmt.Errorf("%s is missing; run codegen", path)
+		if errors.Is(err, os.ErrNotExist) {
+			return fmt.Errorf("%s: %w; run codegen", path, ErrMissingGeneratedFile)
 		}
 		return fmt.Errorf("read %q: %w", path, err)
 	}
```
</details>

Coding guideline: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
var ErrStaleGeneratedFile = errors.New("generated file is stale")
var ErrMissingGeneratedFile = errors.New("generated file is missing")

func checkGeneratedFile(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: %w; run codegen", path, ErrMissingGeneratedFile)
		}
		return fmt.Errorf("read %q: %w", path, err)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/codegen/openapits/generate.go` around lines 72 - 77, In
checkGeneratedFile, replace the os.IsNotExist(err) usage with errors.Is(err,
os.ErrNotExist) and introduce a new sentinel error ErrMissingGeneratedFile
(mirroring ErrStaleGeneratedFile) to represent a missing generated file; when
the file read fails due to not-exist, return a wrapped error using the sentinel
(e.g., fmt.Errorf("%w: %s", ErrMissingGeneratedFile, path)) so callers can use
errors.Is to detect the missing-file case instead of string matching.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `checkGeneratedFile()` uses `os.IsNotExist(err)` and returns a plain string error for missing output, which prevents callers from using typed `errors.Is` matching and forces string inspection.
- Fix plan: add a dedicated missing-file sentinel, switch the filesystem-not-found branch to `errors.Is(err, os.ErrNotExist)`, and wrap the sentinel in the returned error.
- Resolution: added `ErrMissingGeneratedFile`, switched the missing-file branch to `errors.Is(err, os.ErrNotExist)`, and wrapped the sentinel in the returned error.
- Verification: `go test ./internal/codegen/openapits` and `make verify` passed after the change.
