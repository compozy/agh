---
status: resolved
file: internal/api/core/handlers.go
line: 550
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-IGMH,comment:PRRC_kwDOR5y4QM67_zc2
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

rg -n -C3 'func StatusForWorkspaceError|StatusForWorkspaceError\(' --type go
```

Repository: compozy/agh

Length of output: 8499

---

🏁 Script executed:

```shell
cat -n internal/api/core/errors.go | sed -n '40,100p'
```

Repository: compozy/agh

Length of output: 2509

---

🏁 Script executed:

```shell
rg -n 'func statusForWorkspaceError' internal/api/core/errors.go -A 20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
rg -n 'func statusForWorkspaceError' --type go
```

Repository: compozy/agh

Length of output: 229

---

🏁 Script executed:

```shell
sed -n '105,150p' internal/api/core/session_workspace.go
```

Repository: compozy/agh

Length of output: 1763

---



</details>

**Return a stable dependency-unavailable response when workspace resolver is missing.**

When `h.Workspaces` is nil, `workspaceAgentDefs` returns a plain error that isn't a workspace package sentinel. The default case of `statusForWorkspaceError` maps this to `http.StatusInternalServerError` (500), which doesn't reflect a missing injected dependency. `GET /api/agents?workspace=...` and `GET /api/agents/:name?workspace=...` should return a deterministic, capability-aware status (e.g., 503) rather than a generic internal error.

Consider defining a dedicated sentinel (e.g., `ErrWorkspaceResolverUnavailable`) in the workspace package and mapping it explicitly in `statusForWorkspaceError`, or use a different status code path for nil-check failures that indicates unavailability rather than server error.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/handlers.go` around lines 538 - 550, The nil-check in
BaseHandlers.workspaceAgentDefs currently returns a generic fmt.Errorf when
h.Workspaces is nil; instead introduce and return a dedicated sentinel error
(e.g., workspace.ErrWorkspaceResolverUnavailable) from this function (or a
well-named package-level sentinel in the workspace package) so callers can
detect a missing injected dependency, and update statusForWorkspaceError to map
that sentinel to a 503 (or other capability-unavailable status) so GET
/api/agents... yields a deterministic dependency-unavailable response; locate
the check in workspaceAgentDefs (h.Workspaces) and the error-mapping in
statusForWorkspaceError to implement the sentinel and mapping.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `BaseHandlers.workspaceAgentDefs` returns a generic formatted error when `h.Workspaces` is nil. `statusForAgentWorkspaceError` delegates generic errors to `StatusForWorkspaceError`, which currently maps them to HTTP 500. The root cause is the missing domain sentinel for a resolver dependency that is unavailable. Fix by adding a workspace sentinel, returning it from the nil dependency path, and mapping it to HTTP 503 with regression coverage. Minimal support edits outside the issue's primary file are required in `internal/workspace/workspace.go` for the sentinel and `internal/api/core/session_workspace.go` plus tests for the HTTP status mapping.

## Resolution

- Added `workspace.ErrWorkspaceResolverUnavailable`, returned it from the nil resolver path, and mapped it to HTTP 503.
- Added regression coverage for `/agents?workspace=...` and the workspace status mapping.
- Verified through targeted Go tests and `make verify`.
