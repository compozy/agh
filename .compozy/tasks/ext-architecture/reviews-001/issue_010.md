---
status: resolved
file: internal/cli/client.go
line: 513
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4092736828,nitpick_hash:6a5ee75f8109
review_hash: 6a5ee75f8109
source_review_id: "4092736828"
source_review_submitted_at: "2026-04-10T22:18:10Z"
---

# Issue 010: Minor: Redundant strings.TrimSpace on name parameter.
## Review Comment

The callers `EnableExtension` and `DisableExtension` already call `strings.TrimSpace(name)` before passing to `extensionAction`. The additional `TrimSpace` on line 517 is harmless but redundant.

## Triage

- Decision: `valid`
- Notes: `EnableExtension` and `DisableExtension` already normalize the name before delegating to `extensionAction`, so the second trim in `extensionAction` is redundant. I will remove the duplicate normalization and keep a single canonical trim point.
