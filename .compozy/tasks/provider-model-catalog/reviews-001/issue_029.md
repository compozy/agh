---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: pending
file: internal/store/globaldb/global_db_model_catalog.go
line: 142
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6th,comment:PRRC_kwDOR5y4QM6-6btf
---

# Issue 029: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Keep the row list and reasoning-effort lookup on one snapshot.**

`ListRows` runs the base row query and `listModelCatalogReasoningEfforts` as two independent reads. A concurrent `ReplaceSourceRows` can commit between them, so callers can observe model rows from one catalog revision and reasoning efforts from another. Please execute both reads inside the same read transaction/connection, or fold efforts into the main query.

 
Based on learnings, Keep execution paths deterministic and observable.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db_model_catalog.go` around lines 113 - 142,
The two reads (the initial rows query using g.db.QueryContext and the subsequent
listModelCatalogReasoningEfforts call) must be executed on the same DB
connection/transaction to avoid observing mixed revisions; wrap both reads in a
single read-only transaction or use the same connection/Tx for both operations
(e.g., begin a Tx via g.db.BeginTx and run the query that calls
scanModelCatalogRow and then call listModelCatalogReasoningEfforts using that
Tx/connection), then iterate catalogRows, compute keys with modelCatalogKey and
assign ReasoningEfforts, and ensure rows are closed and the Tx is
committed/rolled back properly so both reads are consistent.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
