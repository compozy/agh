---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/config/provider.go
line: 1028
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245741930,nitpick_hash:f2855e8e28bf
review_hash: f2855e8e28bf
source_review_id: "4245741930"
source_review_submitted_at: "2026-05-07T16:19:15Z"
---

# Issue 012: Misleading error message for whitespace-only default.
## Review Comment

The condition `strings.TrimSpace(m.Default) == "" && m.Default != ""` correctly catches whitespace-only defaults, but the error message "is required" is confusing—it implies the field is missing rather than malformed.

## Triage

- Decision: `valid`
- Notes:
  - `ProviderModelsConfig.Validate(...)` correctly rejects whitespace-only defaults, but the current message says the field "is required".
  - That message is misleading because the field is present but malformed.
  - Fix: keep the validation logic and change the error text to describe whitespace-only content accurately.
