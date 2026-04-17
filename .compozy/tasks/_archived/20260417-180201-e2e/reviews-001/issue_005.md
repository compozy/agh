---
status: resolved
file: internal/api/udsapi/transport_parity_integration_test.go
line: 144
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:f8b7f467cffb
review_hash: f8b7f467cffb
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 005: Replace the sleep-based polling helpers.
## Review Comment

These wait loops rely on fixed `time.Sleep` intervals, which makes the transport lane slower and flakier under CI load. Switch them to the shared wait helper or a context-aware ticker/select so the wait can stop immediately on success or cancellation. As per coding guidelines, Never use `time.Sleep()` in orchestration — use proper synchronization primitives.

Also applies to: 165-185, 222-242

## Triage

- Decision: `VALID`
- Root cause: the polling helpers use fixed `time.Sleep` delays, which makes cancellation slower and adds avoidable flake under CI contention.
- Fix plan: replace the sleep loops with ticker/select-based waiting that exits immediately on success or context timeout. The matching HTTP transport waiter will be updated to the same pattern as part of the root-cause fix.
- Resolution: replaced the UDS transport polling helpers with context-aware ticker loops and updated the matching HTTP waiter to the same pattern.
- Verification: `go test ./internal/api/httpapi ./internal/api/udsapi` passed. Historical note: the later blocker about a missing `driver/dist/index.js` was stale; the shipped mock driver is `internal/testutil/acpmock/cmd/acpmock-driver`.
