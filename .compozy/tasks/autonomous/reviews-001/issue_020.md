---
status: resolved
file: internal/api/udsapi/agent_identity_test.go
line: 145
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:fbf102a3039e
review_hash: fbf102a3039e
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 020: Handle the json.Marshal failure here instead of discarding it.
## Review Comment

If `contract.AgentMeResponse` stops serializing cleanly, this failure path will hide the real problem behind `_` and make the assertion output less trustworthy.

As per coding guidelines, "Never ignore errors with `_` in Go — every error must be handled or have a written justification."

## Triage

- Decision: `VALID`
- Notes: `TestAgentMeReturnsValidatedCallerIdentity` discards the `json.Marshal` error when formatting a failure. If the payload becomes non-serializable, the test would hide that problem. Fix by checking the marshal error before using the encoded payload in the failure message.
