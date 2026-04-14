---
status: resolved
file: internal/registry/extract.go
line: 267
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lM5,comment:PRRC_kwDOR5y4QM63oCtr
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, locate and examine the PathWithinRoot function
fd extract.go --type f | head -5
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
# Check the actual file and function
cat -n internal/registry/extract.go | sed -n '240,270p'
```

Repository: compozy/agh

Length of output: 1395

---

🏁 Script executed:

```shell
# Find all usages of PathWithinRoot
rg 'PathWithinRoot' --type go -B 2 -A 2
```

Repository: compozy/agh

Length of output: 3081

---

🏁 Script executed:

```shell
# Check the ExtractArchive function to understand its validation
ast-grep --pattern 'func ExtractArchive($$$) {
  $$$
}'
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check how ExtractArchive validates destRoot
rg 'func ExtractArchive' --type go -A 30 | head -50
```

Repository: compozy/agh

Length of output: 1863

---

🏁 Script executed:

```shell
# Check the resolvePathWithinRoot wrapper in extension/manager.go
rg 'func resolvePathWithinRoot' --type go -A 10
```

Repository: compozy/agh

Length of output: 691

---

🏁 Script executed:

```shell
# Check the test cases for PathWithinRoot to understand expected behavior
rg 'TestPathWithinRoot' --type go -A 30 | head -60
```

Repository: compozy/agh

Length of output: 1973

---



</details>

**Add explicit validation to reject empty root in `PathWithinRoot`.**

This exported function lacks input validation that similar functions in the codebase provide (e.g., `extractArchive` and `resolvePathWithinRoot`). When `root` is empty or whitespace-only, `filepath.Abs` resolves to the process working directory, creating a silent failure mode. An exported helper should validate its own inputs rather than relying on callers to do so.

<details>
<summary>Proposed fix</summary>

```diff
 func PathWithinRoot(root string, child string) (string, error) {
+	if strings.TrimSpace(root) == "" {
+		return "", errors.New("root path is required")
+	}
 	absRoot, err := filepath.Abs(strings.TrimSpace(root))
 	if err != nil {
 		return "", fmt.Errorf("resolve root %q: %w", root, err)
 	}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func PathWithinRoot(root string, child string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", errors.New("root path is required")
	}
	absRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", root, err)
	}
	targetPath := filepath.Join(absRoot, child)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve target %q: %w", targetPath, err)
	}
	relative, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", fmt.Errorf("resolve target %q within %q: %w", absTarget, absRoot, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path must stay within the root directory")
	}
	return absTarget, nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract.go` around lines 250 - 267, PathWithinRoot
currently lets an empty or whitespace-only root proceed because filepath.Abs
turns it into the working directory; update PathWithinRoot to explicitly
validate the trimmed root (e.g., call strings.TrimSpace(root) and check for
empty) and return a clear error (e.g., "root path must not be empty") before
calling filepath.Abs; follow the same validation style as
resolvePathWithinRoot/extractArchive and ensure error messages reference
PathWithinRoot for easier debugging and add/adjust unit tests that exercise
empty/whitespace root inputs.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `PathWithinRoot()` trims the root only when passing it to `filepath.Abs`, so an empty or whitespace-only root resolves to the current working directory instead of being rejected.
- Why this is a real defect: the helper is exported and used as a safety boundary; silently substituting the process cwd weakens that boundary and can produce misleading path-resolution behavior.
- Intended fix: reject blank roots explicitly and expose a stable sentinel error so callers and tests can assert the failure with `errors.Is`.
- Resolution: `PathWithinRoot()` now rejects blank roots before resolution and returns a sentinel error that callers/tests can match.
- Verification: Extended `internal/registry/extract_test.go` to cover blank-root rejection, ran `go test ./internal/registry`, and then `make verify`.
