---
status: pending
file: internal/extension/registry.go
line: 165
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM563lMJ,comment:PRRC_kwDOR5y4QM63oCsx
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Verify whether older `extensions` tables get an explicit migration path.
rg -n -C2 --type go --type sql --glob '!**/*_test.go' \
  'CREATE TABLE IF NOT EXISTS extensions|ALTER TABLE\s+extensions|registry_slug|registry_name|remote_version'
```

Repository: compozy/agh

Length of output: 2649

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Search for migration files and directories
echo "=== Looking for migration directories/files ==="
find . -type d \( -name "*migrat*" -o -name "*schema*" \) 2>/dev/null | head -20

echo ""
echo "=== Looking for migration-related functions ==="
rg -n --type go 'func.*[Mm]igrat|func.*[Uu]pgrade|func.*[Ss]chema' --max-count=20

echo ""
echo "=== Searching for ALTER TABLE statements ==="
rg -n -i 'ALTER TABLE' --type go --type sql

echo ""
echo "=== Checking internal/store directory structure ==="
find internal/store -type f -name '*.go' | head -20
```

Repository: compozy/agh

Length of output: 6449

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Check migrateGlobalSchema function ==="
sed -n '56,150p' internal/store/globaldb/migrate_workspace.go

echo ""
echo "=== Check if extensions table is mentioned in migrate_workspace.go ==="
rg -n -C3 'extensions' internal/store/globaldb/migrate_workspace.go

echo ""
echo "=== Check schema.go to understand migration flow ==="
cat -n internal/store/schema.go

echo ""
echo "=== Check global_db.go to see how migrations are called ==="
sed -n '1,200p' internal/store/globaldb/global_db.go | cat -n
```

Repository: compozy/agh

Length of output: 3560

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Search for ANY mention of extensions migration/alter ==="
rg -n -C2 'extensions' internal/store/globaldb/migrate_workspace.go

echo ""
echo "=== Check where migrateGlobalSchema is called ==="
rg -n 'migrateGlobalSchema' internal/store/globaldb/

echo ""
echo "=== Check global_db.go initialization to see migration flow ==="
rg -n -B5 -A10 'func.*Open.*[Gg]lobal|func.*New.*[Gg]lobal' internal/store/globaldb/global_db.go | head -100

echo ""
echo "=== Search for 'registry_slug|registry_name|remote_version' migration ==="
rg -n 'registry_slug|registry_name|remote_version' internal/store/globaldb/migrate_workspace.go
```

Repository: compozy/agh

Length of output: 115

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Check where migrateGlobalSchema is called ==="
rg -n 'migrateGlobalSchema' internal/store/globaldb/

echo ""
echo "=== Check entire global_db.go initialization ==="
head -300 internal/store/globaldb/global_db.go | cat -n

echo ""
echo "=== Search for any extensions column migration anywhere ==="
rg -n -i 'ALTER TABLE extensions|ADD COLUMN.*registry_slug|ADD COLUMN.*registry_name|ADD COLUMN.*remote_version' --type go --type sql
```

Repository: compozy/agh

Length of output: 13380

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Check context around line 320 in global_db.go ==="
sed -n '300,350p' internal/store/globaldb/global_db.go | cat -n

echo ""
echo "=== Check if there's any separate extensions migration file ==="
find internal -name '*extension*' -type f | grep -i migrat

echo ""
echo "=== Check internal/extension/registry.go initialization ==="
head -50 internal/extension/registry.go

echo ""
echo "=== Look for installWithConfig and see if there's error handling for missing columns ==="
rg -n -B5 -A15 'func.*installWithConfig|func.*Install.*extension' internal/extension/registry.go | head -80
```

Repository: compozy/agh

Length of output: 1213

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

echo "=== Verify the three queries that would fail ==="
echo "Query 1 - List (lines 162-165):"
sed -n '160,166p' internal/extension/registry.go

echo ""
echo "Query 2 - Get (lines 200-203):"
sed -n '198,205p' internal/extension/registry.go

echo ""
echo "Query 3 - Install (lines 364-366):"
sed -n '362,368p' internal/extension/registry.go

echo ""
echo "=== Confirm migration gap: extensions NOT in migrateGlobalSchema ==="
grep -c 'extensions' internal/store/globaldb/migrate_workspace.go || echo "NO MATCHES - extensions migration missing"
```

Repository: compozy/agh

Length of output: 1078

---



</details>

**Add a migration for the new provenance columns.**

These queries now assume `registry_slug`, `registry_name`, and `remote_version` already exist. The provided schema context only shows `CREATE TABLE IF NOT EXISTS extensions (...)`, which does not upgrade older databases. The migration logic in `migrateGlobalSchema()` handles other tables (sessions, event_summaries, token_stats, permission_log, network_audit_log) but does not include the extensions table. Existing installations will fail with `no such column` errors when `List()`, `Get()`, or `Install()` executes against a pre-existing extensions table lacking these columns.

Also applies to: 200-203, 364-366

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/registry.go` around lines 162 - 165, The extensions table
needs a migration to add the new provenance columns referenced by List(), Get(),
and Install(); update migrateGlobalSchema() to perform an idempotent schema
change that adds registry_slug, registry_name, and remote_version to the
extensions table (use ALTER TABLE … ADD COLUMN if-not-exists semantics or a safe
existence check) so older databases are upgraded before queries run; ensure the
migration runs alongside the other table migrations and that the new columns
have appropriate types/defaults to match how registry.go uses them.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
