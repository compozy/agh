---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/acp/client_integration_test.go
line: 16
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:a36e016c6a58
review_hash: a36e016c6a58
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 001: Mark the independent subtests parallel.
## Review Comment

These subtests each build their own temp dir, driver, and helper process, so they can call `t.Parallel()`. Keeping them serial slows the integration suite and skips the default test-concurrency discipline this repo asks for. The network guardrails case can stay serial because it uses `t.Setenv`.

As per coding guidelines, "Default to `t.Parallel` in Go tests unless there is a specific reason to disable it (opt-out with `t.Setenv`)".

Also applies to: 40-67, 71-104, 108-179, 183-223

## Triage

- Decision: `VALID`
- Notes:
  The current subtests are independent and use isolated temp dirs, drivers, and helper processes. They can safely opt into `t.Parallel()`. The network-guardrails case remains serial because it uses `t.Setenv`, which is the documented opt-out.
