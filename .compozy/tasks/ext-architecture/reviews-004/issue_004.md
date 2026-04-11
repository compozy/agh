---
status: resolved
file: internal/api/core/handlers.go
line: 605
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56Q887,comment:PRRC_kwDOR5y4QM6200jM
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid returning the absolute home path from `/daemon/status`.**

`UserHomeDir` will usually contain the local account name (`/home/alice`, `/Users/bob`) and now gets returned to every caller of the status endpoint. Unless this route is guaranteed to be local/admin-only, that's unnecessary host/PII disclosure with little contract value. Prefer dropping this field from the HTTP payload or gating it behind an explicit privileged/debug mode.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/handlers.go` around lines 596 - 605, The handler is
returning the absolute user home path via UserHomeDir (h.daemonUserHomeDir()) in
the /daemon/status response, leaking local user/host PII; remove this field from
the response payload (contract.DaemonStatusPayload) or only populate it when an
explicit privileged/debug condition is true. Locate the construction of
contract.DaemonStatusResponse in handlers.go (where UserHomeDir:
h.daemonUserHomeDir() is set) and either delete the UserHomeDir entry entirely
or wrap it in a conditional (e.g., if h.isPrivilegedRequest() or
h.Config.PrivilegedStatusMode) so the field is only included for admin/debug
requests; adjust the contract/consumer expectations accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
- `daemon.user_home_dir` is an intentional, currently consumed contract. The web workspace onboarding flow uses it to implement the "Use global workspace" action, and both core/http API tests assert that behavior today.
- Removing or blanking the field in `handlers.go` alone would regress supported local-first functionality without introducing a replacement capability for the UI/CLI flow.
- A real hardening change here would need a broader transport/authz design decision rather than an isolated handler tweak, so this review item is not actionable as a standalone patch in the scoped files.
