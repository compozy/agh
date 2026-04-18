---
status: resolved
file: internal/memory/store.go
line: 632
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575b1Z,comment:PRRC_kwDOR5y4QM65BvYQ
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Empty scopes will reindex on every request.**

`scopeEntryCount(...) == 0` is being used as the only readiness signal. For a legitimately empty global/workspace scope, that stays zero forever, so every `Search`/`HealthStats` call re-scans disk and rewrites the catalog. Persist a per-scope/workspace synced marker in `memory_catalog_state` and check that instead of row existence alone.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 618 - 632, ensureCatalogFilterReady
currently treats scopeEntryCount(...) == 0 as the only readiness signal and will
call reindexScopes every time a legitimately empty scope/workspace has zero
rows; instead add a persisted per-scope/workspace readiness marker in the
memory_catalog_state table and consult that marker in ensureCatalogFilterReady
before deciding to reindex. Specifically, modify ensureCatalogFilterReady to
first read a synced/readiness flag for the given filter.scope and
filter.workspaceRoot from memory_catalog_state, only fall back to calling
scopeEntryCount if the marker is missing, and after reindexScopes completes
successfully set the marker for that scope/workspace; update any codepaths that
create or refresh the catalog (reindexScopes) to write the marker (and
clear/update it on failures) so empty but-synced scopes do not reindex on every
Search/HealthStats call.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `ensureCatalogFilterReady` treats `scopeEntryCount(...) > 0` as the only ready signal, so a legitimately empty but already-synced scope keeps looking cold and reindexes on every `Search`/`HealthStats` call.
  - The underlying root cause is missing persisted readiness state in `memory_catalog_state`; empty scopes need a durable synced marker independent of row count.
  - Completed fix: added persisted per-scope/workspace sync markers in `memory_catalog_state`, wrote them during scope replacement, and taught `ensureCatalogFilterReady` to trust the marker first and backfill it from existing rows when needed.
  - Completed validation: added regression coverage proving empty synced scopes stay warm across later `HealthStats` reads in `internal/memory/store_test.go`.
  - Verification: `make verify` passed after the change set.
