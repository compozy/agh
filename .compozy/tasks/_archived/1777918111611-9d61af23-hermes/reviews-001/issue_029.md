---
status: resolved
file: internal/skills/provenance_test.go
line: 80
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:f856bd528cc9
review_hash: f856bd528cc9
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 029: Add t.Run() subtest and replace string error matching with typed error assertion
## Review Comment

This test violates two coding guidelines: it bypasses the required `t.Run("ShouldRejectSymlinkEscape", ...)` subtest pattern and relies on brittle `strings.Contains(err.Error(), "reject hashed symlink")` matching at line 103.

**Required changes:**
1. Wrap test logic in `t.Run("ShouldRejectSymlinkEscape", func(t *testing.T) { ... })`
2. Define a sentinel error (e.g., `var ErrSymlinkEscape = errors.New(...)`) in `provenance.go` and return it from `ComputeDirectoryHash()`
3. Replace line 103's string check with `errors.Is(err, ErrSymlinkEscape)` or a typed error assertion via `errors.As()`

The production code currently wraps errors with message text only (`fmt.Errorf("skills: reject hashed symlink %q: %w", ...)`), but must expose a comparable error identity for proper testing per coding guidelines: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings."

## Triage

- Decision: `valid`
- Root cause: `ComputeDirectoryHash` wraps symlink escape rejection with descriptive text only, so tests must match `err.Error()` text to distinguish that failure. The test also lacks the required `ShouldRejectSymlinkEscape` subtest wrapper.
- Fix approach: add a comparable `ErrSymlinkEscape` error identity in `internal/skills/provenance.go` and wrap it when rejecting an escaping symlink, then update the test to use `t.Run("ShouldRejectSymlinkEscape", ...)` plus `errors.Is`. This requires the minimal production edit to `internal/skills/provenance.go`; changing only the scoped test file cannot create a structured error identity.
