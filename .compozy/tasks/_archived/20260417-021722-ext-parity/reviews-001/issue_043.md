---
status: resolved
file: internal/daemon/bridge_resources.go
line: 68
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:a04d85a2b541
review_hash: a04d85a2b541
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 043: Silent nil return may mask configuration issues.
## Review Comment

The function returns `nil, nil` when either `raw` or `codecs` is nil. This silent behavior could mask misconfiguration during daemon boot, making debugging harder. Consider returning an explicit error or logging a warning when this occurs.

## Triage

- Decision: `invalid`
- Notes:
  - `bridgeInstanceResourceStore` returning `nil, nil` is the intentional signal that bridge resource definitions are disabled for this runtime, matching the caller’s `bridgeResources != nil` guard in `bootRuntimeServices`.
  - The bridge runtime already falls back to the service-backed lifecycle path when `resourceDefinitionsEnabled()` is false, so this does not silently create a partially configured bridge subsystem.
  - There is no reachable daemon-boot misconfiguration here to surface: the nil store represents "resource-backed bridge mode unavailable", not an unexpected broken state.
  - Resolution: no production change required; repository verification passed after resolving the valid issues in this batch.
