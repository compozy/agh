---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/store/globaldb/global_db_model_catalog.go
line: 143
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaq1,comment:PRRC_kwDOR5y4QM6-7HZa
---

# Issue 028: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Read rows and reasoning efforts from the same snapshot.**

`ListRows` issues two independent selects against `g.db`. If a concurrent `ReplaceSourceRows` commits between them, callers can get row metadata from one snapshot and `ReasoningEfforts` from another, which makes the merged catalog nondeterministically lose or gain effort values. Please pin both reads to one read transaction / connection, or collapse them into a single snapshot-consistent query.

 

Based on learnings "Keep execution paths deterministic and observable in Go backend."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db_model_catalog.go` around lines 113 - 143,
The two separate selects (the QueryContext call that builds catalogRows and the
call to listModelCatalogReasoningEfforts) must run on the same snapshot; wrap
both reads in one read-only transaction or single connection so they see a
consistent snapshot. Change the QueryContext usage to use a sql.Tx (BeginTx with
ReadOnly=true) or a single DB connection and then call
tx.QueryContext/tx.QueryRowContext for the catalog scan (scanModelCatalogRow)
and update listModelCatalogReasoningEfforts to accept the same executor (tx or
conn) instead of g.db so you can call listModelCatalogReasoningEfforts(ctx, tx,
opts); compute the modelCatalogKey and assign ReasoningEfforts while still
inside the transaction, then commit/close before returning.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - `GlobalDB.ListRows` already executes through `withModelCatalogReadTransaction`, and `listModelCatalogRows` receives the transaction-scoped executor for both the row scan and the reasoning-effort scan.
  - The package already has a dedicated regression test asserting snapshot-consistent reads of rows and reasoning efforts from one transaction.
  - No code change is needed; this finding was already fixed.
