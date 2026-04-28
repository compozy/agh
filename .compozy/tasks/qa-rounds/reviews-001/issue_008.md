---
status: resolved
file: internal/cli/config.go
line: 97
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4188693296,nitpick_hash:519534b735ab
review_hash: 519534b735ab
source_review_id: "4188693296"
source_review_submitted_at: "2026-04-28T12:24:35Z"
---

# Issue 008: Add coverage for the new defaults.sandbox mutation key.
## Review Comment

Line 97 adds new mutable surface (`defaults.sandbox`), but this change should be pinned with a direct CLI regression test (accept `defaults.sandbox`, reject legacy `defaults.environment`) to avoid rename drift.

As per coding guidelines, "`**/*.go`: Maintain 80% code coverage per Go package".

## Triage

- Decision: `VALID`
- Notes: `defaults.sandbox` is now accepted by `config set`, but `config_test.go` only covers `defaults.provider` and sandbox profile paths. Add a direct CLI regression that sets `defaults.sandbox`, verifies JSON output, and verifies the persisted config value.

## Resolution

- Added CLI coverage that sets `defaults.sandbox`, checks JSON output, and verifies persisted config state.
- Verified through targeted CLI tests and `make verify`.
