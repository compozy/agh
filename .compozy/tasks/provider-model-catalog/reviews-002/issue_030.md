---
provider: coderabbit
pr: "118"
round: 2
round_created_at: 2026-05-07T18:16:18.885242Z
status: resolved
file: internal/store/globaldb/global_db_test.go
line: 434
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AYaq8,comment:PRRC_kwDOR5y4QM6-7HZg
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**`assertAppliedGlobalMigrationOrder` only validates the pinned prefix.**

This helper stops at `expectedGlobalMigrationPrefix()`, so the newer tail migration(s) are no longer identity-checked by the tests that switched from explicit assertions to this shared helper. A renamed, reordered, or checksum-changed tail migration can now pass these tests unnoticed. Please either extend the expected identities to the current tail or rename/split the helper so callers do not treat it as a full-order assertion.  

As per coding guidelines: "Migration drift fixes require observed-history tests — cover fresh DB, upgrade/reopen, and the real recorded migration prefix that failed".


Also applies to: 453-475

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db_test.go` around lines 384 - 427, The helper
expectedGlobalMigrationPrefix only asserts a pinned prefix so tests using
assertAppliedGlobalMigrationOrder miss validating newer migrations; fix by
either (A) extending expectedGlobalMigrationPrefix() to include the current full
tail of migration identities (add the missing version/name/checksum entries) and
update any callers to expect the full sequence, or (B) rename/split the helper
(e.g., expectedGlobalMigrationPrefix -> expectedGlobalMigrationPrefixOnly and
create expectedFullGlobalMigrationOrder()) and update
assertAppliedGlobalMigrationOrder callers to use the new full-order helper so
tests validate the complete migration identity sequence (refer to
expectedGlobalMigrationPrefix and assertAppliedGlobalMigrationOrder to locate
the change).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - Despite the helper name, `expectedGlobalMigrationPrefix()` currently contains the full shipped migration identity list through the current tail migration.
  - `assertAppliedGlobalMigrationOrder` also checks the exact record count and every version/name/checksum tuple, so tail drift is already covered.
  - No code change is needed; the concern no longer matches the current helper behavior.
