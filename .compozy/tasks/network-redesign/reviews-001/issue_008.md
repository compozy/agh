---
status: resolved
file: internal/network/audit_test.go
line: 167
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4151559901,nitpick_hash:f76f09563762
review_hash: f76f09563762
source_review_id: "4151559901"
source_review_submitted_at: "2026-04-22T01:22:21Z"
---

# Issue 008: Run these subtests in parallel.
## Review Comment

Each case builds its own writer/store and does not share state, so adding `t.Parallel()` inside the subtests would keep this aligned with the repo's default test pattern.

As per coding guidelines, "Add `t.Parallel()` to independent subtests in Go tests".

## Triage

- Decision: `valid`
- Reasoning: the affected subtests each build isolated store/writer fixtures and do not share mutable state. Adding `t.Parallel()` inside those subtests is safe and brings the file back in line with the repo's default independent-subtest pattern.
- Fix plan: add `t.Parallel()` to the independent audit-writer subtests only.
- Resolution: added `t.Parallel()` to the independent audit-writer subtests while keeping shared setup unchanged.
- Verification: `go test ./internal/network` and `make verify`
