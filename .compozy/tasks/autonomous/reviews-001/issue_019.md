---
status: resolved
file: internal/api/udsapi/agent_channels_test.go
line: 302
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177060832,nitpick_hash:0f6e1f015154
review_hash: 0f6e1f015154
source_review_id: "4177060832"
source_review_submitted_at: "2026-04-26T14:53:33Z"
---

# Issue 019: Consider extracting shared session fixture.
## Review Comment

This `activeAgentSessionManager` function duplicates similar logic that appears in `internal/api/core/agent_channels_internal_test.go` (`agentCoreSessionManager`). Consider extracting a shared test fixture to avoid duplication across test packages.

## Triage

- Decision: `INVALID`
- Notes: The duplication is intentional package-local fixture setup. `internal/api/core/agent_channels_internal_test.go` returns a `sessionManagerStub`, while `internal/api/udsapi/agent_channels_test.go` returns the UDS package's aliased `stubSessionManager`; extracting this across packages would either expose package test internals or add a shared fixture abstraction only for two short test helpers. The current duplication keeps each package's tests independent and is lower risk than a cross-package fixture.
