---
status: resolved
file: internal/api/core/conversions.go
line: 592
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:9186c00bba42
review_hash: 9186c00bba42
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 003: These task mappers are duplicated across transports.
## Review Comment

This block now mirrors the task mapping code in `internal/extension/host_api_tasks.go` almost field-for-field. With timeline/tree/run-detail/dashboard/inbox all growing together, keeping both copies aligned will be brittle. Consider centralizing the task→payload transforms and letting each transport only wrap its transport-specific response shape.

## Triage

- Decision: `invalid`
- Notes: The duplication observation is fair, but I did not find a concrete behavioral mismatch in the current code. Fixing it cleanly would require a broader cross-package refactor between `internal/api/core` and `internal/extension` mapping layers, which is outside the scope of this review-remediation batch and not required to correct a live defect.
