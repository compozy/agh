---
status: resolved
file: internal/registry/extract_test.go
line: 111
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107316563,nitpick_hash:5fed7df21da7
review_hash: 5fed7df21da7
source_review_id: "4107316563"
source_review_submitted_at: "2026-04-14T15:47:27Z"
---

# Issue 006: Prefer typed/sentinel error assertions here.
## Review Comment

These checks are pinned to free-form error text, so harmless context changes will break the tests. The unsafe-entry paths should expose sentinel or typed errors so these cases can assert with `errors.Is` / `errors.As` instead of `strings.Contains(err.Error(), ...)`. As per coding guidelines, "Use errors.Is() and errors.As() for error matching — never compare error strings" and "MUST have specific error assertions (ErrorContains, ErrorAs)".

Also applies to: 125-127, 148-150, 163-165, 173-175

## Triage

- Decision: `valid`
- Root cause: several archive/path validation paths currently return only free-form error text, which forced the tests in [internal/registry/extract_test.go](/Users/pedronauck/Dev/compozy/_worktrees/ext-registry/internal/registry/extract_test.go) to use brittle `strings.Contains` checks.
- Why this is a real defect: the repository guidance explicitly prefers `errors.Is` / `errors.As` over string matching, and the current tests would break on harmless message rewording.
- Intended fix: introduce sentinel errors for the unsafe archive/path validation branches, wrap them with `%w`, and convert the affected tests to `errors.Is`.
- Resolution: Added sentinel errors in `internal/registry/extract.go` for destination/path/unsupported-entry validation and updated the affected assertions in `internal/registry/extract_test.go` to use `errors.Is`.
- Verification: Ran `go test ./internal/registry` and then `make verify`.
