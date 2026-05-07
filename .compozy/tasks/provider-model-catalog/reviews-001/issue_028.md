---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: pending
file: internal/store/globaldb/global_db.go
line: 687
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6ty,comment:PRRC_kwDOR5y4QM6-6bty
---

# Issue 028: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Do not mutate migration v1 schema payload**

Line 687 injects model-catalog DDL into `globalSchemaStatements`, which is the statement body for `Version: 1`. That changes historical migration contents and risks drift across upgrade histories. Keep migration v1 immutable and introduce this schema only through the new tail migration.

 

<details>
<summary>Suggested fix</summary>

```diff
 	bridgeTaskSubscriptionSchemaStatements(),
 	resources.SchemaStatements(),
-	modelCatalogSchemaStatements(),
 )
```
</details>

As per coding guidelines: SQLite migration registries are append-only; never change an existing migration identity after it may have been applied.

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

In `@internal/store/globaldb/global_db.go` at line 687, The migration v1 payload
is being mutated by appending modelCatalogSchemaStatements() into
globalSchemaStatements (Version: 1); revert that change so
globalSchemaStatements remains exactly as originally defined, and instead
introduce the model-catalog DDL as a new, appended migration (create a new
migration entry that references modelCatalogSchemaStatements() or include it in
the tail migration list) so the schema is added via a new migration identity
rather than modifying Version: 1.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
