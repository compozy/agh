---
status: resolved
file: internal/api/udsapi/handlers_test.go
line: 501
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177143389,nitpick_hash:c1e21af4470a
review_hash: c1e21af4470a
source_review_id: "4177143389"
source_review_submitted_at: "2026-04-26T16:15:24Z"
---

# Issue 009: Extend the handler-binding map to cover the rest of the new agent routes.
## Review Comment

This map only asserts bindings for `reply`, `claim-next`, and `complete`. The newly added `send`, `spawn`, `heartbeat`, `fail`, and `release` routes can still be wired to the wrong handler while `TestRegisterRoutesCoversTechSpecEndpoints` passes, because that test only checks method/path registration.

As per coding guidelines, `Focus on critical paths: workflow execution, state management, error handling`.

## Triage

- Decision: `VALID`
- Notes: `TestRegisterTaskRoutesUseSharedHandlerBindings` only asserts handler binding substrings for `reply`, `claim-next`, and `complete` among the new agent routes. `send`, `spawn`, `heartbeat`, `fail`, and `release` can be registered to the wrong handler while method/path registration still passes. The fix is to extend the expected binding map for the missing agent route handlers.
- Resolution: Extended UDS route binding assertions across the new agent channel/task/spawn routes and verified with focused tests plus full `make verify`.
