---
status: resolved
file: internal/registry/extract_test.go
line: 139
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM567Hj-,comment:PRRC_kwDOR5y4QM63sxX9
---

# Issue 011: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "Check CI runners for non-Linux platforms:"
if [ -d ".github/workflows" ]; then
  rg -n --iglob '*.yml' --iglob '*.yaml' 'runs-on:.*(windows|macos)|matrix:|go-version' .github/workflows || true
else
  echo "No .github/workflows directory found."
fi
```

Repository: compozy/agh

Length of output: 134

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Read the full test file to understand context
echo "=== Full test file context ==="
wc -l internal/registry/extract_test.go
echo ""

# Read around lines 131-139
echo "=== Context around lines 131-139 ==="
sed -n '120,150p' internal/registry/extract_test.go

echo ""
echo "=== Context around lines 252-259 ==="
sed -n '240,270p' internal/registry/extract_test.go
```

Repository: compozy/agh

Length of output: 2223

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check for build constraints in test files
echo "=== Build constraints in extract_test.go ==="
head -20 internal/registry/extract_test.go

# Search for existing platform guards in the test file
echo ""
echo "=== Platform-specific code in test files ==="
rg -n 'build.*tag|//\s*\+build|runtime\.GOOS|build\.Default\.GOOS' internal/registry/ || echo "No platform guards found"

# Check project's go.mod or documentation for platform info
echo ""
echo "=== Go version and module info ==="
head -5 go.mod
```

Repository: compozy/agh

Length of output: 490

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Look for CI configurations beyond GitHub Actions
echo "=== CI configurations ==="
find . -maxdepth 2 -type f \( -name '.gitlab-ci.yml' -o -name '.circleci' -o -name 'Makefile' -o -name 'tox.ini' -o -name '.travis.yml' \) 2>/dev/null | head -20

# Check README for platform information
echo ""
echo "=== README platform mentions ==="
if [ -f README.md ]; then
  rg -i 'platform|windows|macos|linux|supported|os' README.md | head -10
else
  echo "No README.md found"
fi

# Check if go.mod mentions any platform-specific dependencies
echo ""
echo "=== Check for known OS-specific imports ==="
rg -n 'golang.org/x/sys' internal/registry/extract_test.go || echo "No sys imports in test file"
```

Repository: compozy/agh

Length of output: 1410

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check Makefile for CI/testing configuration
echo "=== Makefile content ==="
cat Makefile

# Also search for any test-related scripts
echo ""
echo "=== Test-related scripts ==="
find . -maxdepth 1 -type f -name '*test*' -o -name '*ci*' 2>/dev/null || echo "No test scripts found"
```

Repository: compozy/agh

Length of output: 992

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find magefile
echo "=== Looking for magefile ==="
find . -maxdepth 2 -name 'magefile.*' -o -name 'magefiles' -type d

# Check for any CI files
echo ""
echo "=== All CI/workflow files ==="
find . -path '*/.git' -prune -o -type f \( -name '*.yml' -o -name '*.yaml' \) -print | grep -E '(ci|workflow|github|gitlab|circleci)' || echo "No workflow files found"

# Look at actual test invocation
echo ""
echo "=== Search for test execution pattern ==="
rg -n 'go test' . --type go --type txt --type md 2>/dev/null | head -20 || echo "No explicit go test patterns found"
```

Repository: compozy/agh

Length of output: 273

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Read magefile to understand test target
echo "=== Magefile test target ==="
cat magefile.go | head -100

# Also look for build constraints in the main package
echo ""
echo "=== Search for build constraints in production code ==="
rg -n '//go:build|// \+build' . --type go | grep -v test | head -20 || echo "No build constraints found in production code"

# Check if project explicitly supports Windows/macOS
echo ""
echo "=== Platform support in docs ==="
find docs -type f \( -name '*.md' \) -exec grep -l -i 'platform\|windows\|macos\|linux' {} \; 2>/dev/null || echo "No platform documentation found"
```

Repository: compozy/agh

Length of output: 4119

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Read the full test around symlinked-parent to understand what it's testing
echo "=== Full symlinked-parent test ==="
sed -n '131,160p' internal/registry/extract_test.go

# Read the full test around chmod to understand what it's testing
echo ""
echo "=== Full chmod test ==="
sed -n '240,280p' internal/registry/extract_test.go

# Check what these tests are verifying
echo ""
echo "=== Function being tested - ExtractArchive ==="
rg -A 5 'func ExtractArchive' internal/registry/ | head -20

# Check if there are any existing skip patterns in tests
echo ""
echo "=== Existing test skips ==="
rg -n 't\.Skip|runtime\.GOOS|build.Default' internal/registry/extract_test.go || echo "No test skips found"
```

Repository: compozy/agh

Length of output: 2972

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Get the exact test names and line numbers for these problematic tests
echo "=== Test names and line numbers ==="
rg -n 't\.Run\(' internal/registry/extract_test.go | grep -A 1 -B 1 'symlinked-parent'

# Find the second test with chmod
echo ""
echo "=== Test containing chmod ==="
rg -B 10 'os.Chmod.*0o555' internal/registry/extract_test.go | head -20

# Verify these are the only symlink/chmod tests
echo ""
echo "=== All symlink/chmod tests in extract_test.go ==="
rg -n 'os\.Symlink|os\.Chmod' internal/registry/extract_test.go

# Check if there's any platform guard pattern used elsewhere in tests
echo ""
echo "=== Patterns for platform-aware tests in the codebase ==="
rg -n 'runtime\.GOOS|t\.Skip.*windows|t\.Skip.*darwin' . --type go | head -10 || echo "No platform guards found in test patterns"
```

Repository: compozy/agh

Length of output: 1700

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find the test function name containing the chmod test at line 252
echo "=== Test function containing chmod test ==="
sed -n '220,260p' internal/registry/extract_test.go | head -30

# Look at the exact skip pattern from atomic_test.go as an example
echo ""
echo "=== Example skip pattern from atomic_test.go ==="
sed -n '40,45p' internal/fileutil/atomic_test.go
```

Repository: compozy/agh

Length of output: 1170

---



</details>

**Add platform-specific test skips for os.Symlink and os.Chmod operations.**

The "symlinked-parent" test (line 131) uses `os.Symlink()`, which requires elevated privileges or Dev Mode on Windows. The `TestCleanupArchiveFileJoinsRemoveFailure` test (lines 240-280) uses `os.Chmod()` with 0o555 semantics that differ on Windows—directory permissions don't prevent deletion the same way as on Unix.

Follow the established pattern in `internal/fileutil/atomic_test.go:41-42`:

```go
if runtime.GOOS == "windows" {
    t.Skip("symlink and chmod semantics are platform-specific on windows")
}
```

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/extract_test.go` around lines 131 - 139, Add
platform-specific skips for Windows to avoid running os.Symlink and
os.Chmod-dependent tests: in the t.Run("symlinked-parent", ...) block add a
runtime.GOOS == "windows" check and call t.Skip with a message about
symlink/chmod semantics, and likewise add the same runtime check and t.Skip at
the start of the TestCleanupArchiveFileJoinsRemoveFailure test; follow the
existing pattern used in internal/fileutil/atomic_test.go to detect windows and
skip these platform-specific tests.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `extract_test.go` contains Windows-sensitive `os.Symlink` and directory-permission tests without the platform guard pattern already used elsewhere in the repository.
- Evidence: [`internal/registry/extract_test.go`](internal/registry/extract_test.go) has no `runtime.GOOS == "windows"` skips, while [`internal/fileutil/atomic_test.go`](internal/fileutil/atomic_test.go) already uses that pattern for permission-sensitive tests.
- Fix plan: add targeted Windows skips to the symlink and chmod-dependent tests only.
- Resolution: Added targeted Windows skips to the symlink and chmod-sensitive extractor tests. Verified with package tests and `make verify`.
