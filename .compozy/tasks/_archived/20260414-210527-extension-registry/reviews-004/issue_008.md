---
status: resolved
file: internal/registry/extract.go
line: 176
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107849208,nitpick_hash:7595035c5b73
review_hash: 7595035c5b73
source_review_id: "4107849208"
source_review_submitted_at: "2026-04-14T17:14:35Z"
---

# Issue 008: Clarify unlimited write behavior when limit is zero or negative.
## Review Comment

When `limit <= 0`, the writer simply counts bytes without enforcing any limit. This may be intentional (for testing or optional limit enforcement), but a comment would help clarify this design choice.

## Triage

- Decision: `invalid`
- Root cause analysis: there is no correctness defect here. `countingLimitWriter` intentionally treats `limit <= 0` as unbounded, and the behavior is already covered by `TestCountingLimitWriter` in the same package.
- Evidence: [`internal/registry/extract.go`](internal/registry/extract.go) lines 176-178 are straightforward, and production extraction normalizes non-positive limits before the writer is used.
- Reason not fixing: adding a comment would be documentation churn only; the current code and tests already make the behavior explicit.
- Resolution: No code change required. Verified during package tests and the final `make verify` pass.
