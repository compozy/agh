---
status: resolved
file: internal/extension/registry.go
line: 271
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4107751611,nitpick_hash:b7efef7966ec
review_hash: b7efef7966ec
source_review_id: "4107751611"
source_review_submitted_at: "2026-04-14T16:56:20Z"
---

# Issue 006: Remove installWithSource method and update tests to use public API.
## Review Comment

The method is only called from test files (`registry_test.go`, `registry_integration_test.go`, `manager_test.go`) and duplicates functionality already available through the public `Install` method with the `WithInstallSource` option. Replace test calls with `Install(manifest, path, checksum, WithInstallSource(source))` to avoid maintaining test-only production surface code.

## Triage

- Decision: `valid`
- Root cause: `installWithSource` is redundant production surface; the public `Install(..., WithInstallSource(...))` path already provides the same behavior.
- Evidence: [`internal/extension/registry.go`](internal/extension/registry.go) lines 271-277 only wrap `installWithConfig`, and `rg` shows callers are test-only.
- Fix plan: remove `installWithSource` and update the affected tests to use the public `Install` API with `WithInstallSource`. This requires minimal edits in test files outside the scoped production-file list.
- Resolution: Removed `installWithSource` and updated the affected registry/manager tests to use the public install API. Verified with package tests and `make verify`.
