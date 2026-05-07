---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/store/globaldb/schema_model_catalog.go
line: 40
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYarF,comment:PRRC_kwDOR5y4QM6-7HZu
---

# Issue 031: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Add FK from `model_catalog_rows` to `model_catalog_sources` to prevent orphan rows.**

`model_catalog_rows` is keyed by `(source_id, provider_id, model_id)` but doesn’t enforce that `(source_id, provider_id)` exists in `model_catalog_sources`. A parent delete can leave stale rows.
 
<details>
<summary>Proposed DDL change</summary>

```diff
 		`CREATE TABLE IF NOT EXISTS model_catalog_rows (
 			source_id                TEXT NOT NULL CHECK (trim(source_id) <> ''),
 			provider_id              TEXT NOT NULL CHECK (trim(provider_id) <> ''),
 			model_id                 TEXT NOT NULL CHECK (trim(model_id) <> ''),
@@
 			cost_output_per_million  REAL,
 			last_error               TEXT NOT NULL DEFAULT '',
-			PRIMARY KEY (source_id, provider_id, model_id)
+			PRIMARY KEY (source_id, provider_id, model_id),
+			FOREIGN KEY (source_id, provider_id)
+				REFERENCES model_catalog_sources(source_id, provider_id)
+				ON DELETE CASCADE
 		);`,
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/schema_model_catalog.go` around lines 19 - 40, Add a
foreign key on model_catalog_rows to ensure (source_id, provider_id) references
the parent table model_catalog_sources and avoid orphan rows: update the CREATE
TABLE for model_catalog_rows to include a FOREIGN KEY (source_id, provider_id)
REFERENCES model_catalog_sources(source_id, provider_id) clause (choose ON
DELETE CASCADE if you want child rows removed when a source is deleted, or ON
DELETE RESTRICT to prevent parent deletion while children exist), and ensure the
referenced columns in model_catalog_sources match types and NOT NULL
constraints.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The fresh schema for `model_catalog_rows` still lacks a foreign key back to `model_catalog_sources(source_id, provider_id)`.
  - That leaves the child table structurally capable of holding orphaned rows if parent/source status rows disappear, and the schema should enforce the relationship directly.
  - Fix plan: add the FK to the fresh schema and append a tail migration that rebuilds the projection tables into the corrected shape without mutating existing migration identities.
