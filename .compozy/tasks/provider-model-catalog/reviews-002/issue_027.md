---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/store/globaldb/global_db.go
line: 687
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaq-,comment:PRRC_kwDOR5y4QM6-7HZk
---

# Issue 027: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Do not mutate the Version 1 schema payload with new DDL.**

Adding `modelCatalogSchemaStatements()` into `globalSchemaStatements` changes what migration Version 1 does over time. Keep new schema only in the newly appended migration to preserve append-only migration history.

 

<details>
<summary>Suggested fix</summary>

```diff
 	bridgeTaskSubscriptionSchemaStatements(),
 	resources.SchemaStatements(),
-	modelCatalogSchemaStatements(),
 )
```
</details>

As per coding guidelines: `internal/store/**/*.go`: "SQLite migration registries are append-only" and "New schema work appends at the registry tail."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	bridgeTaskSubscriptionSchemaStatements(),
	resources.SchemaStatements(),
)
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db.go` at line 687, Remove the call to
modelCatalogSchemaStatements() from the initial globalSchemaStatements so
Version 1's payload is not mutated; instead create a new migration entry at the
end of the registry that contains modelCatalogSchemaStatements() and append it
to whatever slice/registry holds migrations (the same registry where
globalSchemaStatements are registered), keeping the existing
globalSchemaStatements and Version 1 migration intact and unmodified.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - `globalSchemaStatements` in the current file already excludes `modelCatalogSchemaStatements()`.
  - The model-catalog schema is introduced by the appended tail migration (`add_model_catalog_persistence`), which preserves the append-only migration contract the review comment asks for.
  - No code change is needed; this finding is stale against the current migration registry.
