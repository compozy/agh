---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/extension/host_api_test.go
line: 4958
severity: minor
author: coderabbitai[bot]
provider_ref: review:4233115469,nitpick_hash:b3b234b931d1
review_hash: b3b234b931d1
source_review_id: "4233115469"
source_review_submitted_at: "2026-05-06T04:12:03Z"
---

# Issue 018: Avoid a hard 2s wall-clock timeout in this cleanup poll.
## Review Comment

A fixed 2-second budget is easy to trip under `-race` or a busy CI runner, even when the prompts would settle shortly after. Reuse the cleanup context deadline here instead of a separate short constant.

As per coding guidelines, `Run tests with -race flag and ensure CGO_ENABLED=1 for race-sensitive packages`.

## Triage

- Decision: `valid`
- Notes:
  - `waitForHostAPIPromptsToSettle` still uses a fixed `2s` wall-clock deadline regardless of the cleanup context budget.
  - That is scheduler-sensitive and can fail spuriously under `-race` or busy CI even when cleanup is making progress.
  - Planned fix: derive the polling deadline from the enclosing cleanup context instead of an unrelated constant.
  - Resolved: the cleanup poll now derives its deadline from the provided cleanup context, and the host API test passes that context through instead of relying on a fixed 2-second timeout.
