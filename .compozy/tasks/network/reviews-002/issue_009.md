---
status: resolved
file: internal/network/delivery.go
line: 647
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093857291,nitpick_hash:e1d66d0523de
review_hash: e1d66d0523de
source_review_id: "4093857291"
source_review_submitted_at: "2026-04-11T14:15:44Z"
---

# Issue 009: Consider reusing the strings.Replacer instance.
## Review Comment

Creating a new `strings.Replacer` on every call to `xmlEscape` is inefficient for a hot path. Consider making it a package-level variable.

## Triage

- Decision: `invalid`
- Notes: `xmlEscape` is behaviorally correct today, and this comment is a micro-optimization request without profiling evidence that the per-call `strings.Replacer` allocation is a meaningful bottleneck. The batch is focused on correctness and testability regressions rather than speculative performance tuning.
