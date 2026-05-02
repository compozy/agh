---
provider: coderabbit
pr: "88"
round: 2
round_created_at: 2026-05-02T22:54:45.308545Z
status: resolved
file: internal/api/core/authored_context.go
line: 230
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_IrdJ,comment:PRRC_kwDOR5y4QM69XbzI
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Soul CAS endpoints should also reject `If-Match`.**

Heartbeat mutations already reject header-based CAS, but these Soul put/delete/rollback/refresh handlers silently ignore `If-Match` and only honor `expected_digest`. That makes conditional requests behave differently across authored-context endpoints and can turn a stale precondition into an unexpected write.
 


Also applies to: 233-265, 303-364

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/authored_context.go` around lines 198 - 230, The
PutAgentSoul handler (and the other authored-context Soul handlers in the same
file for the ranges covering delete/rollback/refresh) must reject requests that
use the If-Match header the same way heartbeat mutations do: detect presence of
the If-Match header early (before calling
decodeAuthoredJSONBody/resolveAuthoredAgentTarget and before calling
h.SoulAuthoring.* methods), and respond with the same error path used for
unsupported header-based CAS (use h.respondError with the same status/error used
by heartbeat mutations) rather than silently ignoring it; apply the same
check-and-reject logic to the other Soul handlers in this file (the
delete/rollback/refresh handlers around lines 233-265 and 303-364) so
conditional requests behave consistently.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - Heartbeat mutations already reject header-based CAS through `rejectHeartbeatIfMatch`, but the Soul put/delete/rollback handlers and session Soul refresh path currently skip that guard.
  - A request carrying `If-Match` can therefore be silently accepted or ignored on Soul endpoints, which is inconsistent with the authored-context CAS contract.
  - Resolved by adding the shared `rejectExpectedDigestHeader` / `rejectSoulIfMatch` guard in `internal/api/core/authored_context.go` before decoding or service calls for Soul put/delete/rollback and session Soul refresh.
  - Verification: added `internal/api/core/authored_context_test.go` to assert the `400` authored-context validation error path and confirmed the full repo gate with `make verify`.
