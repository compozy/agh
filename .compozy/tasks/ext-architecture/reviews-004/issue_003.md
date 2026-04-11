---
status: resolved
file: internal/acp/client_test.go
line: 501
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093048586,nitpick_hash:1dff8daa829b
review_hash: 1dff8daa829b
source_review_id: "4093048586"
source_review_submitted_at: "2026-04-11T01:15:37Z"
---

# Issue 003: Prefer shared constant over magic error code literal in test fixtures.
## Review Comment

Using `-32002` directly makes this test drift-prone if the sentinel changes. Reuse `requestErrorResourceNotFoundCode` for tighter coupling to the contract under test.

## Triage

- Decision: `valid`
- Notes:
- The test fixture hardcodes ACP request error code `-32002` even though the production code already defines `requestErrorResourceNotFoundCode` as the contract constant.
- Reusing the shared constant keeps the test aligned with the production sentinel and avoids silent drift if the ACP contract changes.
- Fix approach: replace the magic literal with the shared constant and keep the resource-missing behavior asserted in the existing test.
