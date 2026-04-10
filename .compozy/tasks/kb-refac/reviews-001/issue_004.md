---
status: resolved
file: internal/api/httpapi/routes.go
line: 21
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56LrWW,comment:PRRC_kwDOR5y4QM62twbc
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Add nil guard for handlers parameter.**

If `handlers` is nil, calls like `handlers.CreateWorkspace` will panic. While the `NoRoute` setup has a nil check, the route registration calls don't.


<details>
<summary>🛡️ Proposed fix</summary>

```diff
 func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
+	if handlers == nil {
+		return
+	}
 	api := router.Group("/api")
 
 	registerWorkspaceRoutes(api, handlers)
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	if handlers == nil {
		return
	}
	api := router.Group("/api")

	registerWorkspaceRoutes(api, handlers)
	registerSessionRoutes(api, handlers)
	registerAgentRoutes(api, handlers)
	registerObserveRoutes(api, handlers)
	registerHookRoutes(api, handlers)
	registerSkillRoutes(api, handlers)
	registerMemoryRoutes(api, handlers)
	registerDaemonRoutes(api, handlers)

	if engine, ok := router.(*gin.Engine); ok && handlers != nil {
		engine.NoRoute(handlers.serveStaticRoute)
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/httpapi/routes.go` around lines 6 - 21, The RegisterRoutes
function calls multiple route registration helpers (registerWorkspaceRoutes,
registerSessionRoutes, registerAgentRoutes, registerObserveRoutes,
registerHookRoutes, registerSkillRoutes, registerMemoryRoutes,
registerDaemonRoutes) using the handlers pointer without guarding for nil, which
can cause panics when handlers is nil; update RegisterRoutes to check that
handlers != nil before invoking those register* functions (either return early
or skip registration when handlers is nil) so calls like
handlers.CreateWorkspace won't be dereferenced, while preserving the existing
NoRoute handler setup that already checks for nil.
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Reasoning: `RegisterRoutes` registers handler methods even when `handlers` is nil. Method values on a nil receiver do not panic immediately, but they still register broken route handlers that will fail when invoked. The function already treats `nil` specially for `NoRoute`, so guarding the route registration path is consistent.
- Fix approach: Return early when `handlers` is nil and add a regression test that confirms no API routes are registered in that configuration.
