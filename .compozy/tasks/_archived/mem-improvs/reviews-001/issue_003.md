---
status: resolved
file: internal/daemon/daemon_memory_e2e_integration_test.go
line: 21
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4132976935,nitpick_hash:a216b942a23f
review_hash: a216b942a23f
source_review_id: "4132976935"
source_review_submitted_at: "2026-04-18T00:19:15Z"
---

# Issue 003: Please wrap these E2E scenarios in t.Run("Should...") subtests.
## Review Comment

Both tests pack several independent contract checks into one large flow, which makes failures harder to localize and drifts from the repository’s test shape. Splitting the main assertions into `t.Run("Should ...")` blocks would make regressions much easier to pinpoint.

As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases`.

Also applies to: 225-225

## Triage

- Decision: `valid`
- Notes:
  - Both E2E tests currently bundle several independent contract checks into single large flows, which makes failures harder to localize and does not match the repository’s `t.Run("Should ...")` style.
  - I will keep the scenario setup intact and split the assertion groups into explicit `Should ...` subtests so regressions point at the failing contract directly.

## Resolution

- Split both daemon memory E2E scenarios into focused `t.Run("Should ...")` subtests around search parity, reindex/health, stored message integrity, prompt recall injection, and stale index preservation.
