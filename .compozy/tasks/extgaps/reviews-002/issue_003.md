---
status: resolved
file: internal/bridges/managed_sync.go
line: 46
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4110597069,nitpick_hash:d06a891443a8
review_hash: d06a891443a8
source_review_id: "4110597069"
source_review_submitted_at: "2026-04-15T03:35:44Z"
---

# Issue 003: Constructor silently returns nil for nil store.
## Review Comment

Returning `nil` from a constructor when a required dependency is missing is an unusual pattern that shifts error discovery to runtime. While the nil-receiver check at line 82 catches this, it produces a generic error message that doesn't indicate the root cause was a `nil` store passed to the constructor.

Consider returning `(*ManagedSyncService, error)` or panicking during construction if this is truly a programmer error, rather than returning a nil service that fails later.

## Triage

- Decision: `valid`
- Root cause: `NewManagedSyncer(...)` returns a nil service when `store == nil`, which obscures the real configuration problem and turns it into a later nil-service failure at the call site.
- Why this is valid: the service already has explicit runtime validation for a missing store in `SyncManagedInstances(...)`, so returning nil from the constructor is unnecessary and makes the failure mode less precise.
- Fix approach: always return a concrete `*ManagedSyncService` and let `SyncManagedInstances(...)` report the explicit `managed sync store is required` error if construction was misconfigured. Add regression coverage in `internal/bridges/managed_sync_test.go` even though that test file is outside the six listed code files, because the review workflow allows targeted test edits needed to validate a fix.
- Resolution: `internal/bridges/managed_sync.go` now always returns a concrete service, and `internal/bridges/managed_sync_test.go` now verifies the explicit missing-store failure path.
- Verification: targeted `go test ./internal/bridges -run 'TestManagedSyncerReconcilesCreateUpdateDelete|TestManagedSyncerWrapsStoreErrors|TestManagedSyncerRejectsDuplicateDesiredIDs|TestManagedSyncerReturnsExplicitErrorForNilStoreConstruction' -count=1` passed, followed by a clean `make verify`.
