---
status: resolved
file: internal/registry/extract.go
line: 146
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lMy,comment:PRRC_kwDOR5y4QM63oCtj
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's locate and read the file to verify the snippet
git ls-files | grep -E 'internal/registry/extract\.go'
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
# Get the file size to determine best reading approach
wc -l internal/registry/extract.go
```

Repository: compozy/agh

Length of output: 87

---

🏁 Script executed:

```shell
# Read the relevant lines with context
sed -n '120,160p' internal/registry/extract.go
```

Repository: compozy/agh

Length of output: 1374

---

🏁 Script executed:

```shell
# Check the full function containing these lines to understand context
sed -n '100,180p' internal/registry/extract.go
```

Repository: compozy/agh

Length of output: 2658

---



</details>

**Surface cleanup failures instead of ignoring them with underscore.**

The call to `os.Remove(targetPath)` at line 142 ignores the removal error, violating the coding guideline "Never ignore errors with `_` — every error must be handled or have a written justification". If the file deletion fails after a write/close error, a partial or truncated file remains on disk and the caller receives no signal of this cleanup failure. Capture and propagate the removal error via `errors.Join()` to provide complete error context.

<details>
<summary>Proposed fix</summary>

```diff
-			if _, err := io.Copy(file, teeReader); err != nil {
-				writeErr := fmt.Errorf("write archive file %q: %w", targetPath, err)
-				if closeErr := file.Close(); closeErr != nil {
-					writeErr = errors.Join(writeErr, fmt.Errorf("close archive file %q after write failure: %w", targetPath, closeErr))
-				}
-				_ = os.Remove(targetPath)
-				return writeErr
-			}
-			if err := file.Close(); err != nil {
-				return fmt.Errorf("close archive file %q: %w", targetPath, err)
-			}
+			cleanupPartial := func(baseErr error) error {
+				if closeErr := file.Close(); closeErr != nil {
+					baseErr = errors.Join(baseErr, fmt.Errorf("close archive file %q after failure: %w", targetPath, closeErr))
+				}
+				if removeErr := os.Remove(targetPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
+					baseErr = errors.Join(baseErr, fmt.Errorf("remove partial archive file %q: %w", targetPath, removeErr))
+				}
+				return baseErr
+			}
+			if _, err := io.Copy(file, teeReader); err != nil {
+				return cleanupPartial(fmt.Errorf("write archive file %q: %w", targetPath, err))
+			}
+			if err := file.Close(); err != nil {
+				return cleanupPartial(fmt.Errorf("close archive file %q: %w", targetPath, err))
+			}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract.go` around lines 137 - 146, The cleanup call
os.Remove(targetPath) currently ignores any error (using _), so modify the error
handling in the write failure branch to capture the removal error and join it
with the existing writeErr (using errors.Join) before returning; specifically,
in the block where writeErr is created after io.Copy failure (and after
attempting file.Close()), replace the ignored os.Remove with code that stores
the removal error (e.g., rmErr := os.Remove(targetPath)) and if rmErr != nil set
writeErr = errors.Join(writeErr, fmt.Errorf("remove archive file %q after write
failure: %w", targetPath, rmErr)) so the returned error includes both the
original write/close error and any removal failure.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the partial-file cleanup path after `io.Copy` failure explicitly discards `os.Remove(targetPath)` errors with `_ =`, so cleanup failures are invisible even when a partial file is left behind.
- Why this is a real defect: it violates the repo's error-handling rule and can hide a failed cleanup after a partial extraction write.
- Intended fix: factor cleanup into a helper that joins write, close, and remove failures, then add a regression test that exercises a removal failure after a partial write.
- Resolution: Added `cleanupArchiveFile()` in `internal/registry/extract.go` and routed both write-failure and close-failure cleanup through it so close/remove errors are joined instead of discarded.
- Verification: Added `TestCleanupArchiveFileJoinsRemoveFailure` in `internal/registry/extract_test.go`, ran `go test ./internal/registry`, and then `make verify`.
