---
status: resolved
file: internal/fileutil/atomic_test.go
line: 84
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:43fbbd438dc3
review_hash: 43fbbd438dc3
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 008: Guard this test on Windows to avoid false failures.
## Review Comment

On Windows, the underlying Win32 APIs normalize file paths by trimming trailing spaces before file creation, regardless of whether Go uses the `\\?\` prefix. This means when `AtomicWriteFile` is called with a path containing trailing whitespace like `"target.txt "`, the file is created as `target.txt` (trimmed). The test's assertion that the trimmed path does not exist will fail on Windows, even if the function works correctly.

Add a platform guard at the start of the test:

```diff
func TestAtomicWriteFilePreservesLiteralWhitespaceInPath(t *testing.T) {
t.Parallel()
+ if runtime.GOOS == "windows" {
+ t.Skip("trailing-space filenames are normalized by Win32 APIs")
+ }

path := filepath.Join(t.TempDir(), "target.txt ")
```

## Triage

- Decision: `VALID`
- Notes:
  The trailing-space pathname assertion is not portable to Windows because the
  platform normalizes trailing spaces during file creation. Plan: guard the test
  with a Windows skip so the literal-path behavior is only asserted on
  platforms that preserve the filename.
