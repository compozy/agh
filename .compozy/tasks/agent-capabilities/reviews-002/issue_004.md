---
status: pending
file: internal/daemon/daemon_test.go
line: 4464
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135966430,nitpick_hash:80cdc7e5234c
review_hash: 80cdc7e5234c
source_review_id: "4135966430"
source_review_submitted_at: "2026-04-19T12:48:57Z"
---

# Issue 004: Deep-clone recorded capabilities in the fake join log.
## Review Comment

`Line 4471` only copies the top-level slice. `session.NetworkPeerCapability` still carries nested slice aliases, so later mutation can bleed into recorded assertions. This fake will be more stable if it snapshots the nested fields too, like the session test fake does.

## Triage

- Decision: `UNREVIEWED`
- Notes:
