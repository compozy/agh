---
provider: coderabbit
pr: "106"
round: 1
round_created_at: 2026-05-06T04:12:39.763475Z
status: resolved
file: internal/store/globaldb/global_db.go
line: 667
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Lrr,comment:PRRC_kwDOR5y4QM6-UJdt
---

# Issue 025: _⚠️ Potential issue_ | _🔴 Critical_ | _🏗️ Heavy lift_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_ | _🏗️ Heavy lift_

**Do not mutate already-released migration 7.**

Line 664 rewrites an applied migration in place. `RunMigrations` validates the recorded version/name/checksum for every applied migration, so databases that already stored version 7 will fail startup on integrity mismatch. Even if you keep the old checksum, existing installs still will not receive any new DDL added here because version 7 never re-runs. Keep migration 7 immutable and move the new claim-lease delta into a brand-new migration version instead.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db.go` around lines 664 - 667, The migration
at Version 7 (Name "add_task_run_claim_lease_schema", function
migrateTaskRunClaimLeaseSchema) must not be mutated; revert any edits to that
migration and instead create a new migration entry with a new Version number
(>7), unique Name (e.g., "add_task_run_claim_lease_schema_v2" or similar), new
Checksum, and a new Up function (e.g., migrateTaskRunClaimLeaseSchemaDelta) that
contains the new DDL/claim-lease changes you added; then append this new
migration to the migrations list so RunMigrations sees it as a new migration.
Ensure the original migrateTaskRunClaimLeaseSchema and its checksum remain
exactly as before so existing databases do not fail integrity checks.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes: I verified the migration history against the current branch parent (`git show a02534a7^:internal/store/globaldb/global_db.go`). Migration `Version: 7 / add_task_run_claim_lease_schema` already contained the same claim-lease DDL and auxiliary tables before this review batch. The current code only refactors those existing statements into `migrateTaskRunClaimLeaseSchema` plus `taskRunClaimLeaseAuxiliarySchemaStatements` while keeping the same version/name/checksum, so this batch is not mutating a previously released migration’s schema contract or hiding new DDL behind an already-applied version.
- Resolution: Analysis only; no code change in this batch.
