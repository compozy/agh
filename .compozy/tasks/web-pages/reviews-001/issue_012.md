---
status: resolved
file: internal/store/globaldb/global_db_network_messages_test.go
line: 97
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:04a22edf08f2
review_hash: 04a22edf08f2
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 012: Convert the guard-clause matrix into table-driven subtests.
## Review Comment

These nil-receiver / nil-context / closed-store checks all follow the same pattern, so a `t.Run("Should...")` table would remove duplication and make future guard cases cheaper to add. As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default".

## Triage

- Decision: `valid`
- Root cause: the guard-clause coverage is correct, but the new cases are repetitive one-offs rather than a table-driven `t.Run("Should...")` matrix.
- Fix approach: collapse the guard cases into a small subtest table.
