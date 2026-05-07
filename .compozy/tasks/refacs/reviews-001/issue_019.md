---
provider: coderabbit
pr: "120"
round: 1
round_created_at: 2026-05-07T18:15:52.56459Z
status: resolved
file: internal/api/udsapi/bridges_test.go
line: 56
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4245882823,nitpick_hash:a3e695369a82
review_hash: a3e695369a82
source_review_id: "4245882823"
source_review_submitted_at: "2026-05-07T16:38:59Z"
---

# Issue 019: Assert response.Bridge.Status == BridgeStatusStarting to cover the contract change.
## Review Comment

The PR's stated behavioral change is that `status` is no longer client-supplied but is now derived from the `enabled` flag (mapped to `BridgeStatusStarting`). The stub at line 40 echoes `req.Status` back verbatim — after removing `status` from the request JSON the handler must be setting it before calling `CreateInstance`. Neither the stub assertions (lines 19-32) nor the response assertions (lines 65-70) verify that this mapping actually occurred, leaving the key guarantee of this change untested.

---

## Triage

- Decision: `VALID`
- Notes:
  The current UDS bridge create test never proves that the handler derived `Status` from `Enabled` before calling `CreateInstance`. The stub echoes `req.Status`, so both the request and response should assert the expected `BridgeStatusStarting` mapping.
