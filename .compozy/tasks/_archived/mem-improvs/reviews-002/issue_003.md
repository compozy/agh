---
status: resolved
file: internal/memory/store_test.go
line: 791
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133125556,nitpick_hash:edcb88e696e9
review_hash: edcb88e696e9
source_review_id: "4133125556"
source_review_submitted_at: "2026-04-18T01:12:18Z"
---

# Issue 003: Please add a partial-indexed health case here.
## Review Comment

This test only exercises the cold-start path where both scopes are indexed together. It won't catch the case where the catalog already contains global rows and `HealthStats()` is asked about a workspace for the first time, which is where scope readiness can be skipped and stats get understated. A follow-up case that indexes global first and then checks a newly introduced workspace would protect that regression.

## Triage

- Decision: `valid`
- Root cause: `TestStoreSearchAndReindex` only covers the empty-catalog path, so it misses the regression where global rows already exist and a brand-new workspace is queried later.
- Impact: the workspace-scope readiness bug can regress without a failing test.
- Fix plan: extend the store catalog regression coverage with a partial-index scenario that seeds global entries first, then exercises a new workspace and verifies the workspace scope gets indexed before reporting health.
- Resolution: added a partial-index regression subtest that seeds global catalog rows first, then creates a fresh workspace and verifies both `Search()` and `HealthStats()` include the newly indexed workspace scope.
- Verification: `go test ./internal/memory`; `make verify`
