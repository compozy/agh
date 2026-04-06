---
status: resolved
file: internal/httpapi/workspaces.go
line: 118
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCU,comment:PRRC_kwDOR5y4QM61T6Hu
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't call multi-reference `Resolve` directly from the `:id` route.**

`GET /api/workspaces/:id` currently accepts name/path-style refs, while `PATCH` and `DELETE` on the same route require the real workspace ID. That makes the route inconsistent and can return the wrong workspace when a name collides with another workspace's ID. Look up the record by ID first, then resolve details from that known workspace.


<details>
<summary>🔧 Suggested fix</summary>

```diff
 func (h *Handlers) getWorkspace(c *gin.Context) {
-	resolved, err := h.workspaces.Resolve(c.Request.Context(), c.Param("id"))
+	workspace, err := h.workspaces.Get(c.Request.Context(), c.Param("id"))
+	if err != nil {
+		respondError(c, statusForWorkspaceError(err), err)
+		return
+	}
+
+	resolved, err := h.workspaces.Resolve(c.Request.Context(), workspace.ID)
 	if err != nil {
 		respondError(c, statusForWorkspaceError(err), err)
 		return
 	}
```
</details>
As per coding guidelines, "Keep execution paths deterministic and observable."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/httpapi/workspaces.go` around lines 100 - 118, The route
getWorkspace currently calls h.workspaces.Resolve directly with c.Param("id")
which allows name/path refs and causes inconsistency with PATCH/DELETE; instead
first fetch the workspace record by its real ID (e.g. call h.workspaces.Get or
the repository method that returns the workspace by ID using c.Param("id")),
handle not-found/error as before, and then call Resolve (or a
ResolveDetails-style method) using the known workspace.ID to populate
Agents/Skills/etc; update subsequent uses such as
filterSessionInfosByWorkspaceID(sessions, resolved.ID) to use the confirmed ID
and keep error handling the same.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - The review comment’s premise is incorrect for the current code.
  - `PATCH` and `DELETE` do not require a raw workspace ID today; both routes also call `h.workspaces.Get(c.Request.Context(), c.Param("id"))`, and `Get` is a multi-reference lookup just like `Resolve`.
  - Changing only `getWorkspace` would introduce a new inconsistency instead of fixing one, and the current interface does not expose an ID-only lookup to implement the suggested behavior cleanly inside this batch.
