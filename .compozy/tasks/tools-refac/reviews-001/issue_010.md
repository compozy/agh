---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/api/httpapi/routes.go
line: 131
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulJn,comment:PRRC_kwDOR5y4QM680KH-
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Verify whether tool mutation routes are guarded at route-registration level
rg -n --type=go 'registerToolRoutes|privilegedMutationGuard|tools\.POST\("/:id/(approvals|invoke)"' internal/api/httpapi/routes.go

# Check whether tests cover non-loopback blocking/allow rules for tool mutation endpoints
rg -n --type=go '/api/tools|/api/toolsets|/api/sessions/:id/tools|non-loopback|loopback' internal/api/httpapi/*_test.go
```

Repository: compozy/agh

Length of output: 2344

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Read the full registerToolRoutes function to see current state
sed -n '126,145p' internal/api/httpapi/routes.go

# Check if other mutation routes use the guard to understand the pattern
sed -n '250,275p' internal/api/httpapi/routes.go

# Check the handlers being used for these routes
rg -n --type=go 'func.*CreateToolApproval|func.*InvokeTool' internal/api/httpapi/
```

Repository: compozy/agh

Length of output: 2026

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Search for test cases that validate non-loopback blocking specifically for tool routes
rg -A 10 -B 2 'Should.*tool.*non-loopback|Should.*approvals.*non-loopback|Should.*invoke.*non-loopback' internal/api/httpapi/handlers_test.go internal/api/httpapi/server_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check if CreateToolApproval and InvokeTool handlers have their own auth checks
rg -A 15 'func.*\(h \*Handlers\) CreateToolApproval|func.*\(h \*Handlers\) InvokeTool' internal/api/httpapi/handlers.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find handler implementations by searching the directory
find internal/api/httpapi -name "*.go" -type f | head -20

# Search more broadly for the handlers
rg -n 'CreateToolApproval|InvokeTool' internal/api/httpapi/
```

Repository: compozy/agh

Length of output: 1012

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check what privilegedMutationGuard does
rg -A 10 'func.*privilegedMutationGuard' internal/api/httpapi/handlers.go

# Search for any tests that test tool approval or invoke endpoints
rg -n 'CreateToolApproval|InvokeTool|/api/tools.*approvals|/api/tools.*invoke' internal/api/httpapi/
```

Repository: compozy/agh

Length of output: 550

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Look for tests that specifically check auth/access control on these endpoints
rg -B 5 -A 15 'non-loopback.*tool|tool.*non-loopback' internal/api/httpapi/ --type=go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find and read the handler implementations
rg -n 'func.*Handlers.*CreateToolApproval|func.*Handlers.*InvokeTool' internal/api/httpapi/ -A 20
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Find loopbackMutationGuard implementation
rg -n 'func.*loopbackMutationGuard' internal/api/httpapi/ -A 15
```

Repository: compozy/agh

Length of output: 770

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check the handlers_test.go context around tool endpoint tests
sed -n '190,210p' internal/api/httpapi/handlers_test.go
```

Repository: compozy/agh

Length of output: 832

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Get broader context around handlers_test.go line 198-199
sed -n '170,220p' internal/api/httpapi/handlers_test.go
```

Repository: compozy/agh

Length of output: 1783

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Search more carefully for these handler functions
rg -n 'CreateToolApproval|InvokeTool' internal/api/httpapi/handlers.go -A 25
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check if these handlers are perhaps in extensions.go or another file
rg -n 'CreateToolApproval|InvokeTool' internal/api/httpapi/ -B 2 -A 15
```

Repository: compozy/agh

Length of output: 1380

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Search all Go files in internal/api/httpapi for these handler definitions
rg -l 'CreateToolApproval|InvokeTool' internal/api/httpapi/*.go

# Also try searching with a pattern match
rg 'func.*\(.*\).*CreateToolApproval|func.*\(.*\).*InvokeTool' internal/api/httpapi/
```

Repository: compozy/agh

Length of output: 86

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check if handlers struct is embedded or if these are methods
rg -n 'type Handlers' internal/api/httpapi/ -A 30
```

Repository: compozy/agh

Length of output: 2027

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Search for test coverage validating non-loopback blocking
rg -B 5 -A 15 'Should.*block.*non-loopback' internal/api/httpapi/server_test.go
```

Repository: compozy/agh

Length of output: 716

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check if these handlers are defined in BaseHandlers (embedded in Handlers)
rg -n 'CreateToolApproval|InvokeTool' internal/ --type=go
```

Repository: compozy/agh

Length of output: 4140

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Let's check the routes.go file more carefully - maybe there's a comment or indication these don't exist
sed -n '120,135p' internal/api/httpapi/routes.go
```

Repository: compozy/agh

Length of output: 695

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check the test that lists all routes - what test function is that?
sed -n '130,150p' internal/api/httpapi/handlers_test.go
```

Repository: compozy/agh

Length of output: 708

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Read the CreateToolApproval and InvokeTool handler implementations
sed -n '75,123p' internal/api/core/tools.go
```

Repository: compozy/agh

Length of output: 1731

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Read InvokeTool handler implementation
sed -n '124,200p' internal/api/core/tools.go
```

Repository: compozy/agh

Length of output: 2663

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check if there are any tests specifically for non-loopback blocking on tool endpoints
rg -B 3 -A 15 'Should.*approvals|Should.*invoke.*tool|tool.*mutation' internal/api/httpapi/server_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Search for any test that attempts to access tool approval or invoke from non-loopback
rg -n 'approvals|invoke.*tool' internal/api/httpapi/server_test.go -i
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefill

# Check if there's any upstream guard that might protect these endpoints
# Look for authentication middleware applied at route group level
sed -n '1,50p' internal/api/httpapi/routes.go
```

Repository: compozy/agh

Length of output: 109

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check what api.Group() middleware is applied and whether tools group gets any auth
rg -B 5 -A 5 'api.Group.*"/(extensions|settings|tools)"' internal/api/httpapi/routes.go
```

Repository: compozy/agh

Length of output: 1433

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check the route registration in server setup
rg -n 'registerToolRoutes|registerExtensionRoutes|registerSettingsRoutes' internal/api/httpapi/routes.go -B 2 -A 1
```

Repository: compozy/agh

Length of output: 821

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Verify that api parameter in registerToolRoutes doesn't have middleware applied
# Check how the routes are called
rg -B 10 'registerToolRoutes\(api' internal/api/httpapi/routes.go | head -20
```

Repository: compozy/agh

Length of output: 719

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check if there's a global auth middleware that might protect all routes
sed -n '1,100p' internal/api/httpapi/server.go
```

Repository: compozy/agh

Length of output: 2608

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check what auth middleware is applied and to which groups
rg -n 'api.Group.*auth|auth.*=.*handlers' internal/api/httpapi/routes.go
```

Repository: compozy/agh

Length of output: 158

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Look at complete context around tools route registration
sed -n '10,35p' internal/api/httpapi/routes.go
```

Repository: compozy/agh

Length of output: 857

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check if there are ANY tests that verify non-loopback blocking for these specific endpoints
rg -B 10 -A 10 'POST.*approvals|POST.*invoke' internal/api/httpapi/server_test.go | head -60
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Verify one more time: confirm there are NO auth/loopback checks in either handler
rg -n 'loopback|Forbidden|privileged' internal/api/core/tools.go
```

Repository: compozy/agh

Length of output: 87

---

🏁 Script executed:

```shell
#!/bin/bash
set -euo pipefail

# Check the UDS API routes to see if they have the same issue
sed -n '150,160p' internal/api/udsapi/routes.go
```

Repository: compozy/agh

Length of output: 360

---



</details>

**Protect tool mutation routes with the same privileged guard used by settings/extensions.**

`POST /api/tools/:id/approvals` and `POST /api/tools/:id/invoke` are mutation/execution paths but lack `privilegedMutationGuard()`. The handlers `CreateToolApproval` and `InvokeTool` perform no loopback enforcement. Other mutation routes in settings and extensions consistently apply the guard at route registration; tool routes deviate from this pattern. Non-loopback HTTP binds will expose remote tool execution.

<details>
<summary>Suggested route-level hardening</summary>

```diff
 func registerToolRoutes(api gin.IRouter, handlers *Handlers) {
+	privileged := handlers.privilegedMutationGuard()
 	tools := api.Group("/tools")
 	tools.GET("", handlers.ListTools)
 	tools.POST("/search", handlers.SearchTools)
-	tools.POST("/:id/approvals", handlers.CreateToolApproval)
-	tools.POST("/:id/invoke", handlers.InvokeTool)
+	tools.POST("/:id/approvals", privileged, handlers.CreateToolApproval)
+	tools.POST("/:id/invoke", privileged, handlers.InvokeTool)
 	tools.GET("/:id", handlers.GetTool)

 	sessions := api.Group("/sessions")
 	sessions.GET("/:id/tools", handlers.ListSessionTools)
 	sessions.POST("/:id/tools/search", handlers.SearchSessionTools)

 	toolsets := api.Group("/toolsets")
 	toolsets.GET("", handlers.ListToolsets)
 	toolsets.GET("/:id", handlers.GetToolset)
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func registerToolRoutes(api gin.IRouter, handlers *Handlers) {
	privileged := handlers.privilegedMutationGuard()
	tools := api.Group("/tools")
	tools.GET("", handlers.ListTools)
	tools.POST("/search", handlers.SearchTools)
	tools.POST("/:id/approvals", privileged, handlers.CreateToolApproval)
	tools.POST("/:id/invoke", privileged, handlers.InvokeTool)
	tools.GET("/:id", handlers.GetTool)

	sessions := api.Group("/sessions")
	sessions.GET("/:id/tools", handlers.ListSessionTools)
	sessions.POST("/:id/tools/search", handlers.SearchSessionTools)

	toolsets := api.Group("/toolsets")
	toolsets.GET("", handlers.ListToolsets)
	toolsets.GET("/:id", handlers.GetToolset)
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/routes.go` around lines 130 - 131, The routes POST
/:id/approvals and POST /:id/invoke register handlers CreateToolApproval and
InvokeTool without the privilegedMutationGuard, leaving tool mutation/execution
endpoints unprotected; update the tools route registrations in routes.go to wrap
these handlers with privilegedMutationGuard() (the same guard used for
settings/extensions routes) so that CreateToolApproval and InvokeTool are only
reachable when the privileged guard allows it, matching the pattern used
elsewhere and preventing remote non-loopback tool invocation.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: HTTP `POST /api/tools/:id/approvals` and `POST /api/tools/:id/invoke` are mutation/execution routes but are registered without `privilegedMutationGuard()`. Existing settings/extension mutations use the guard for non-loopback HTTP binds. Add the guard to these tool mutation routes and extend server tests so the handlers are not called on non-loopback binds.
