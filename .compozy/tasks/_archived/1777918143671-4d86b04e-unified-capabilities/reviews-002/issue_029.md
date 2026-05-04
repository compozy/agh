---
status: resolved
file: internal/store/globaldb/global_db_session_test.go
line: 29
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:94b277ae2ca3
review_hash: 94b277ae2ca3
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 029: Assert the two new liveness timestamps too.
## Review Comment

This query now includes `subprocess_started_at` and `last_update_at`, but the test only checks PID/stall fields. A column-order regression can mis-wire either timestamp and still pass, so please assert both parsed values as part of the new scan contract.

As per coding guidelines, `Focus on critical paths: workflow execution, state management, error handling` and `Ensure tests verify behavior outcomes, not just function calls`.

Also applies to: 70-81

## Triage

- Decision: `valid`
- Notes:
  `scanSessionInfo()` now reads both `subprocess_started_at` and `last_update_at`, but the focused scan test only asserts PID and stall fields. A scan-order regression could swap or drop one of those timestamps without failing the test.
  I will add assertions for both parsed liveness timestamps in the existing scan coverage.
  Fixed and verified with targeted package tests plus `make verify`.
