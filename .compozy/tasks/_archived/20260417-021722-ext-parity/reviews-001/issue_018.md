---
status: resolved
file: internal/api/httpapi/server.go
line: 240
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQU,comment:PRRC_kwDOR5y4QM64dqGd
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Fail fast when resource auth is configured without a resource service.**

`/api/resources` registration is gated by `resourceAuth`, not by `resources`. If a caller passes `WithResourceOperatorAuth(...)` but forgets `WithResourceService(...)`, the server can start with resource routes wired to a nil backend and only fail at request time. Please reject that combination in `validateRequired()`.


<details>
<summary>Suggested fix</summary>

```diff
func (s *Server) validateRequired() error {
	switch {
+	case len(s.resourceAuth) > 0 && s.resources == nil:
+		return errors.New("httpapi: resource service is required when resource operator auth is configured")
 	case s.sessions == nil:
 		return errors.New("httpapi: session manager is required")
 	case s.tasks == nil:
```
</details>


Also applies to: 390-390

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/server.go` around lines 235 - 240, If resource operator
auth is configured (WithResourceOperatorAuth sets server.resourceAuth) but no
resource service was provided (WithResourceService sets server.resources),
startup should fail fast; update validateRequired() to check that if
server.resourceAuth is non-nil/has handlers and server.resources is nil then
return a clear error (e.g., "resource auth configured but no resource service
provided"). Modify validateRequired() to perform this guard so routes under
/api/resources cannot be wired to a nil backend at runtime.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `validateRequired` currently checks sessions, tasks, observer, and workspace dependencies but does not reject `resourceAuth` without `resources`. That allows the server to start with HTTP resource routes configured against a nil backend. The fix is to fail fast during server construction and add a unit test.
