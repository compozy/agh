---
status: resolved
file: internal/registry/extract_test.go
line: 150
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM567HkC,comment:PRRC_kwDOR5y4QM63sxYC
---

# Issue 012: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "1) Find string-based error assertions in this test file:"
rg -n 'strings\.Contains\(.*err\.Error\(\)' internal/registry/extract_test.go

echo
echo "2) List current sentinel errors in extract implementation:"
rg -n 'Err[A-Za-z0-9_]+' internal/registry/extract.go

echo
echo "3) Check whether extract.go wraps gzip/path errors with %w:"
rg -n 'fmt\.Errorf\(.*%w' internal/registry/extract.go
```

Repository: compozy/agh

Length of output: 8428

---

🏁 Script executed:

```shell
cat -n internal/registry/extract.go | sed -n '361,371p'
```

Repository: compozy/agh

Length of output: 476

---



</details>

**Fix string-based error assertions; use `errors.Is()` for sentinel errors — but the symlink case requires implementation refactoring first.**

The gzip case (line 174) can be refactored to use `errors.Is()` since the underlying error is already wrapped with `%w` at line 76 in the implementation.

However, the symlink case (line 149) cannot use the suggested `errors.As()` approach. The implementation at line 368 creates a formatted error message without wrapping any underlying error. To enable typed error matching, define and use a sentinel error in the implementation:

```go
// In extract.go
var ErrPathTraversesSymlink = errors.New("path traverses symlink")

// At line 368, wrap it:
return fmt.Errorf("%w: extraction path %q", ErrPathTraversesSymlink, path)
```

Then the test can assert:
```go
var symlinkErr *fs.PathError
if !errors.Is(err, registry.ErrPathTraversesSymlink) {
    t.Fatalf("ExtractArchive(symlinked parent) error = %v, want symlink guard", err)
}
```

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the symlink traversal guard returns a plain formatted error string, forcing brittle string matching in tests instead of sentinel-based assertions.
- Evidence: [`internal/registry/extract.go`](internal/registry/extract.go) line 368 does not wrap a sentinel error, and the matching test in [`internal/registry/extract_test.go`](internal/registry/extract_test.go) line 149 falls back to `strings.Contains`.
- Fix plan: introduce a sentinel for symlink traversal, wrap it in the extractor, and update the tests to use `errors.Is`.
- Resolution: Added `ErrPathTraversesSymlink`, wrapped the extractor error with it, and updated tests to use `errors.Is`. Verified with package tests and `make verify`.
