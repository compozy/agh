---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: pending
file: internal/store/globaldb/global_db_test.go
line: 426
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6tr,comment:PRRC_kwDOR5y4QM6-6btq
---

# Issue 030: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Pin the current migration tail in this append-only contract.**

`expectedGlobalMigrationPrefix()` still stops at version 20, and the helpers only require `len(records) >= len(want)`. That means migrations added after 20 can be renamed, reordered, or have their checksums changed without this contract failing, which leaves the new model-catalog tail unprotected. Extend the expected identities through the current tail and assert the full shipped sequence exactly.

 

As per coding guidelines, "SQLite migration registries are append-only... never insert, reorder, rename, renumber, or change an existing migration identity after it may have been applied."


Also applies to: 429-503

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/store/globaldb/global_db_test.go` around lines 384 - 426, The test
uses expectedGlobalMigrationPrefix() which only lists migrations through version
20 and relies on a len(records) >= len(want) check, leaving newer migrations
unpinned; update expectedGlobalMigrationPrefix() to include the complete,
current shipped migration tail (all versions/checksums up to the latest) and
change the test assertion to require exact sequence equality (not just prefix
length) so identities cannot be renamed/reordered/changed; apply the same fix to
the other helper/test block referenced in the comment (the second
expected-migration helper around lines 429-503) so both migration registries are
pinned end-to-end.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
