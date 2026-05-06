---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/contract/memory.go
line: 507
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isb,comment:PRRC_kwDOR5y4QM6-UFVZ
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Use one provider identifier for lifecycle operations.**

`MemoryProviderLifecycleRequest` carries `Name`, but enable/disable already target a provider by path in the new client surface. That lets one request identify two different providers, and behavior then depends on which field the handler trusts. Drop `Name` from this body or make handlers reject mismatches explicitly.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/contract/memory.go` around lines 503 - 507, The
MemoryProviderLifecycleRequest struct currently contains Name and Reason which
allows two different identifiers for the same lifecycle call; remove the Name
field from MemoryProviderLifecycleRequest so lifecycle requests only carry
Reason (and rely on the provider path/ID from routing/client surface), then
update all handlers and callers that previously read
MemoryProviderLifecycleRequest.Name (e.g., your enable/disable provider handlers
and any client code constructing this request) to use the route/path provider
identifier instead; also update tests and any JSON serialization expectations
accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/core/memory.go` still resolves lifecycle targets with `firstNonEmptyString(req.Name, c.Param("provider_name"))`, and `internal/cli/memory.go` still sends `MemoryProviderLifecycleRequest{Name: name}`.
  - That means the request body can disagree with the route path and the handler will silently choose whichever field is non-empty first, which is the exact ambiguity the review called out.
  - Fix approach: remove `Name` from `MemoryProviderLifecycleRequest`, make enable/disable use the route param only, and update CLI callers plus route tests/spec expectations accordingly.
