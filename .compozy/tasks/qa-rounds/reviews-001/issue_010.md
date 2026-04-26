---
status: resolved
file: internal/network/router.go
line: 639
severity: minor
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:335034e78607
review_hash: 335034e78607
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 010: Directed self-WHOIS is now suppressed, but the manager still audits it as received.
## Review Comment

Line 640 drops the local responder when the directed target is also the sender. That means a self-directed WHOIS now returns with no `Generated` response, but `Manager.controlMessageReceivers()` still records the target session as having received the request. This will skew received audits/stats for a message that was intentionally skipped.

## Triage

- Decision: `VALID`
- Notes: A directed WHOIS whose target is also the sender is intentionally skipped by `whoisRequestResponders`, but the route result remains non-ignored. `Manager.recordInboundAudit` suppresses control-message receiver audit only for ignored/rejected results, so the skipped self-WHOIS can still be audited as received. Fix within `internal/network/router.go` by marking directed self-WHOIS requests as `Ignored`, and add coverage in `router_test.go`.
