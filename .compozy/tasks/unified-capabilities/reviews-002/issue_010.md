---
status: resolved
file: internal/daemon/daemon_network_collaboration_integration_test.go
line: 353
severity: minor
author: coderabbitai[bot]
provider_ref: review:4148870373,nitpick_hash:986aa43f6f14
review_hash: 986aa43f6f14
source_review_id: "4148870373"
source_review_submitted_at: "2026-04-21T15:20:42Z"
---

# Issue 010: The broadcast assertion only checks the sender transcript.
## Review Comment

If fan-out to `curatorSession` regresses, this scenario still passes because every later step is directed. Please add a recipient-side transcript check here so the `"capability say delivery"` step still validates broadcast delivery.

## Triage

- Decision: `valid`
- Root cause: the broadcast `"capability say delivery"` wait only checks channel storage and the sender transcript, so a regression that stops fan-out to `curatorSession` would not be caught before the scenario switches to directed traffic.
- Fix plan: add a recipient-side transcript assertion for the broadcast step in this integration test.
- Resolution: implemented and verified through targeted Go tests and a clean `make verify` run.
