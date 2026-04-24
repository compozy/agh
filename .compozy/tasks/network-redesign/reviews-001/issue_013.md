---
status: resolved
file: internal/store/globaldb/global_db_network_messages_test.go
line: 128
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4166737115,nitpick_hash:e9dd9f7a7931
review_hash: e9dd9f7a7931
source_review_id: "4166737115"
source_review_submitted_at: "2026-04-23T23:14:00Z"
---

# Issue 013: Cover cursor tie-breaks with equal timestamps.
## Review Comment

The cursor pagination test only exercises strictly increasing timestamp values. To fully validate the `(timestamp, message_id)` ordering contract, add at least one pair of entries with the same timestamp but different message IDs (e.g., `msg-2a` and `msg-2b` at `recordedAt.Add(time.Minute)`). This ensures entries sharing a timestamp aren't skipped or duplicated during pagination.

## Triage

- Decision: `valid`
- Reasoning: the cursor pagination test only covers strictly increasing timestamps. Since `ListNetworkMessages()` orders by `(timestamp, message_id)`, the current test misses the tie-break case where multiple rows share a timestamp.
- Fix plan: add equal-timestamp entries with distinct `message_id` values and update the before/after assertions to verify the full `(timestamp, message_id)` ordering contract.
- Resolution: expanded the cursor pagination fixtures with equal-timestamp message IDs and updated the assertions to verify deterministic `(timestamp, message_id)` ordering.
- Verification: `go test ./internal/store/globaldb` and `make verify`
