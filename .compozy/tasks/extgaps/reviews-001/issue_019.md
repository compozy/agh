---
status: resolved
file: internal/store/globaldb/global_db_bundles_test.go
line: 33
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110509614,nitpick_hash:fc4be2088bd3
review_hash: fc4be2088bd3
source_review_id: "4110509614"
source_review_submitted_at: "2026-04-15T03:04:56Z"
---

# Issue 019: Seed a legacy row before asserting migration success.
## Review Comment

This currently proves the column exists after reopen, but not that the migration preserved existing activations. A destructive table rebuild would still pass. Insert one legacy row before `OpenGlobalDB()` and assert it survives with a sane `spec_content_hash` value after migration.

As per coding guidelines, `**/*_test.go`: Focus on critical paths: workflow execution, state management, error handling.

## Triage

- Decision: `valid`
- Root cause: the migration test currently checks only that `spec_content_hash` exists after reopen. It does not prove that legacy activation rows survive the migration intact.
- Fix plan: seed a legacy activation row before reopening, run the migration, and assert the activation still exists afterward with the expected core fields and an empty/default `spec_content_hash`.
- Resolution: seeded a legacy activation row before reopen and asserted the migrated row survives with the expected core fields and an empty `spec_content_hash`.
- Verification: updated `internal/store/globaldb/global_db_bundles_test.go` and passed `go test ./internal/store/globaldb` plus `make verify`.
