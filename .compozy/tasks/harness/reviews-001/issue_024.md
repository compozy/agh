---
status: resolved
file: internal/observe/observer_test.go
line: 283
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:d9fcbd8a0357
review_hash: d9fcbd8a0357
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 024: This bypasses the recorder path the PR is adding.
## Review Comment

Writing summaries straight into `registry` only proves `QueryEvents` can read stored rows. It will still pass if `harnessLifecycleRecorder` emits the wrong type/summary or never writes at all, so it doesn’t really protect the new observability integration. As per coding guidelines, focus on critical paths and ensure tests verify behavior outcomes, not just function calls.

## Triage

- Decision: `invalid`
- Reasoning: this unit test is intentionally scoped to the `observe.QueryEvents` read path over stored summaries. The actual harness-lifecycle recorder path is package-private to `internal/daemon` and is already exercised end-to-end by the transport parity integration coverage that waits for `harness.context_resolved`, `harness.section_selected`, and `harness.augmenter_applied` through the observe HTTP/UDS surface. Reworking `internal/observe/observer_test.go` would duplicate broader integration coverage without improving the protected behavior in this package.
- Resolution approach: leave the unit test focused on `QueryEvents`; no code change is required for this finding.
