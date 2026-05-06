---
provider: coderabbit
pr: "108"
round: 2
round_created_at: 2026-05-06T04:43:32.489895Z
status: resolved
file: internal/api/core/memory.go
line: 83
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2cOa,comment:PRRC_kwDOR5y4QM6-Uf9Z
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Honor `include_system` on the list/show endpoints.**

The new CLI wires `--include-system` for `memory list` and `memory show`, but these handlers never read that signal and always surface `_system` entries. That makes the default behavior broader than advertised and exposes system-managed memory unless callers already know to avoid it.




<details>
<summary>Suggested fix</summary>

```diff
 type memorySelector struct {
 	Scope       memcontract.Scope
 	Workspace   string
 	WorkspaceID string
 	AgentName   string
 	AgentTier   memcontract.AgentTier
+	IncludeSystem bool
 }
@@
 func memorySelectorFromQuery(c *gin.Context) memorySelector {
 	workspaceID := firstNonEmptyString(c.Query("workspace_id"), c.Query("workspace"))
 	return memorySelector{
 		Scope:       memcontract.Scope(c.Query("scope")),
 		WorkspaceID: workspaceID,
 		AgentName:   c.Query("agent_name"),
 		AgentTier:   memcontract.AgentTier(c.Query("agent_tier")),
+		IncludeSystem: c.Query("include_system") == "true",
 	}
 }
@@
 	for _, currentScope := range scopes {
 		current := resolved
 		current.Scope = currentScope
 		store, err := h.memoryStoreForSelector(ctx, current)
@@
-		headers = append(headers, items...)
+		for _, item := range items {
+			if !current.IncludeSystem && memorySystemManaged(item.Filename) {
+				continue
+			}
+			headers = append(headers, item)
+		}
 	}
@@
 	location, err := h.resolveMemoryLocation(c.Request.Context(), c.Param("filename"), memorySelectorFromQuery(c))
 	if err != nil {
 		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
 		return
 	}
+	if !memorySelectorFromQuery(c).IncludeSystem && memorySystemManaged(location.Filename) {
+		err := fmt.Errorf("%w: memory %q not found", os.ErrNotExist, location.Filename)
+		h.respondMemoryError(c, StatusForMemoryError(err), err, nil)
+		return
+	}
```
</details>


Also applies to: 203-229, 1652-1659

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/memory.go` around lines 75 - 83, ListMemory currently
always returns system memories; update it (and the corresponding show handler)
to honor the include_system flag by reading the query param (or the flag set by
memorySelectorFromQuery) and filtering out entries whose name starts with
"_system" when include_system is false. Concretely: in BaseHandlers.ListMemory
(and the memory show handler) use the selector returned by
memorySelectorFromQuery or c.Query("include_system") to determine
include_system, call listMemoryHeaders as you already do, then if include_system
is false remove any headers with names starting with "_system" before converting
with memorySummaryPayloads and returning; ensure the same check is applied in
the single-memory show path so system-managed entries are only returned when
include_system is true.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `memorySelector` and `memorySelectorFromQuery` do not carry `include_system`, so `ListMemory` and `ReadMemory` always behave as if system-managed files are allowed.
- Evidence: `ListMemory` forwards the selector straight into `listMemoryHeaders`, which currently appends every scanned header without filtering, and `ReadMemory` resolves/reads the file without any `_system` gate.
- Fix plan: add `IncludeSystem` to the selector, parse it from query params, filter `_system` entries in list responses, and reject direct reads of system-managed files unless `include_system=true`.
- Resolution: implemented in `internal/api/core/memory.go` and validated with targeted handler coverage in `internal/api/httpapi/memory_test.go` because the scoped production file had no local test seam in the batch scope.
- Verification: targeted `go test` for `internal/api/httpapi` passed, and fresh `make verify` passed on 2026-05-06.
