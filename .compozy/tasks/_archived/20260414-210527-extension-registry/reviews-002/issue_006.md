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
- Notes:
  Marked completed (resolved).
