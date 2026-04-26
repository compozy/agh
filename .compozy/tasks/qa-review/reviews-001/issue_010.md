---
status: resolved
file: internal/store/globaldb/global_db_task_test.go
line: 725
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:86ff1de48631
review_hash: 86ff1de48631
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 010: Cover the other persisted open-run states here as subtests.
## Review Comment

This validates the queued case against SQLite, but the storage guard rejects *any* non-terminal stored status. Adding claimed/running/starting variants here would catch SQL-level status normalization regressions that the in-memory manager tests would miss.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default pattern" and "Focus on critical paths: workflow execution, state management, error handling."

## Triage

- Decision: `valid`
- Notes:
  - The current SQLite regression only exercises the queued open-run case, but the storage guard rejects any non-terminal stored status.
  - Root cause: coverage does not currently pin claimed, starting, or running rows, so a future SQL/status normalization change could regress the guard without failing this test.
  - Fix plan: convert the open-run guard test into table-driven subtests for queued, claimed, starting, and running statuses while preserving the idempotency assertions.
