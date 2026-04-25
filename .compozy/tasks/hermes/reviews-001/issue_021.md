---
status: resolved
file: internal/memory/store.go
line: 466
severity: major
author: coderabbitai[bot]
provider_ref: review:4175534665,nitpick_hash:d583059689f4
review_hash: d583059689f4
source_review_id: "4175534665"
source_review_submitted_at: "2026-04-25T12:34:13Z"
---

# Issue 021: Scope operation health metrics to the same visible workspaces.
## Review Comment

`operationStats(ctx)` is aggregated across the entire shared catalog, while the entry/orphan counts above are filtered to `workspaces`. In a multi-workspace daemon, `HealthStats(..., []string{workspaceA})` will report operation counts and last-operation timestamps produced by workspaceB.

## Triage

- Decision: `valid`
- Root cause: `Store.HealthStats` builds catalog filters for global memory plus the caller-visible workspace roots, and uses those filters for entry/orphan counts, but then calls `catalog.operationStats(ctx)` with no filters. That query counts every row in the shared `memory_operation_log`, including operations from unrelated workspace scopes.
- Fix approach: pass the same catalog filters into operation-stat collection, count only global/legacy rows plus the requested workspace roots, and add a regression test with two workspace roots sharing one catalog to prove workspace B operations no longer affect workspace A health counts or timestamps.
