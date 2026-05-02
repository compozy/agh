---
provider: coderabbit
pr: "88"
round: 1
round_created_at: 2026-05-02T18:22:40.559088Z
status: pending
file: internal/automation/trigger_test.go
line: 809
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4215360648,nitpick_hash:4940d7b08a0e
review_hash: 4940d7b08a0e
source_review_id: "4215360648"
source_review_submitted_at: "2026-05-02T18:22:08Z"
---

# Issue 016: Make the test resolver fail on unexpected secret refs.
## Review Comment

Right now the resolver returns `"shared-secret"` for every lookup, so these tests won't catch the engine resolving the wrong `WebhookSecretRef`. A ref-keyed stub would exercise the new contract much more directly.

## Triage

- Decision: `UNREVIEWED`
- Notes:
