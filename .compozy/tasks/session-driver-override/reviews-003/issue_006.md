---
status: resolved
file: web/src/systems/session/components/session-resume-failure.tsx
line: 5
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167424608,nitpick_hash:cd4a9fe88708
review_hash: cd4a9fe88708
source_review_id: "4167424608"
source_review_submitted_at: "2026-04-24T02:13:16Z"
---

# Issue 006: cn is unnecessary here with only static class strings.
## Review Comment

You can inline the className and drop the extra import.

Also applies to: 33-36

## Triage

- Decision: `invalid`
- Notes:
- This is style-only churn with no behavioral, accessibility, or policy impact.
- `cn(...)` is used broadly across `web/src/systems/session/components`, including cases where a component currently has static classes but may grow conditional variants later; using it here is consistent with nearby component patterns.
- Removing a harmless utility import does not materially improve correctness, bundle behavior, or maintainability enough to justify touching the component for this batch.
- Resolution: no code change required.
