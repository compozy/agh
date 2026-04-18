---
status: resolved
file: internal/session/stop_reason.go
line: 194
severity: minor
author: coderabbitai[bot]
provider_ref: review:4135189365,nitpick_hash:6cf9ae234f44
review_hash: 6cf9ae234f44
source_review_id: "4135189365"
source_review_submitted_at: "2026-04-18T22:38:10Z"
---

# Issue 027: Wrap stop-preparation failures with stage context.
## Review Comment

These raw `return err` paths make it hard to distinguish whether stop setup failed in pre-stop hooks, metadata persistence, or prompt setup synchronization.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

Also applies to: 199-201, 204-205, 208-209

## Triage

- Decision: `valid`
- Root cause: `prepareStopWithCause` still returns several stage errors raw, which hides whether stop setup failed in pre-stop hook dispatch, stop-state preparation, metadata persistence, or prompt-setup synchronization.
- Fix approach: wrap each stop-preparation failure with stage-specific context and add focused coverage so the wrapped errors remain distinguishable.
