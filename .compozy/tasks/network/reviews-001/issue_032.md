---
status: resolved
file: internal/store/globaldb/global_db_network_audit.go
line: 125
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:4fcbc27c3c95
review_hash: 4fcbc27c3c95
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 032: Wrap timestamp parse failures with audit-row context.
## Review Comment

Returning the raw `store.ParseTimestamp` error makes it harder to tell which scan path failed when audit reads break. Please wrap it here the same way the other database errors are wrapped.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf(\"context: %w\", err)`".

## Triage

- Decision: `valid`
- Notes:
  `scanNetworkAudit` wraps scan failures but returns bare `store.ParseTimestamp` errors. That loses the fact that the failure happened while decoding a network-audit row from the global DB. The fix is to wrap timestamp parse failures with the same network-audit scan context used for the other row-decoding errors.
  Resolved by wrapping timestamp parse failures in `internal/store/globaldb/global_db_network_audit.go`. Because the relevant regression coverage lives in `internal/store/globaldb/global_db_network_audit_test.go`, I made a minimal out-of-scope test addition there. Verified with package tests and a clean `make verify`.
