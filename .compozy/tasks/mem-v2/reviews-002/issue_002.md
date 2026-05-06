---
provider: coderabbit
pr: "108"
round: 2
round_created_at: 2026-05-06T04:43:32.489895Z
status: resolved
file: internal/api/core/memory.go
line: 1567
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2cOi,comment:PRRC_kwDOR5y4QM6-Uf9n
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Don't route explicit global selectors through a workspace store.**

`memoryRecallStoreForSelector` drops into `ForWorkspace(...)` whenever `resolved.Workspace` is set, even if the resolved selector is `global` or `agent:global`. That silently changes search/reindex/reset/promote requests from global scope to workspace-local state.




<details>
<summary>Suggested fix</summary>

```diff
 func (h *BaseHandlers) memoryRecallStoreForSelector(
 	ctx context.Context,
 	selector memorySelector,
 ) (*memory.Store, error) {
@@
 	store := h.MemoryStore
-	if strings.TrimSpace(resolved.Workspace) != "" {
+	if strings.TrimSpace(resolved.Workspace) != "" && (
+		resolved.Scope == memcontract.ScopeWorkspace ||
+		resolved.Scope == "" ||
+		(resolved.Scope == memcontract.ScopeAgent && resolved.AgentTier == memcontract.AgentTierWorkspace)
+	) {
 		store = store.ForWorkspace(resolved.Workspace)
 	}
 	if strings.TrimSpace(resolved.AgentName) != "" && resolved.AgentTier.Normalize() != "" {
 		store = store.ForAgent(resolved.WorkspaceID, resolved.AgentName, resolved.AgentTier)
 	}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/core/memory.go` around lines 1549 - 1567,
memoryRecallStoreForSelector currently calls
store.ForWorkspace(resolved.Workspace) and store.ForAgent(...) whenever
resolved.Workspace or resolved.AgentName are set, which incorrectly routes
explicit global selectors into a workspace-local store; update the logic after
resolveMemorySelector so you only call ForWorkspace when resolved.Workspace is
non-empty AND not equal to the global sentinel (e.g., "global"), and only call
ForAgent when resolved.AgentName is non-empty AND not equal to the agent-global
sentinel (e.g., "global"); keep using resolved.WorkspaceID/resolved.AgentTier as
before but guard the ForWorkspace and ForAgent calls against the global values
to preserve true global scope.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `memoryRecallStoreForSelector` unconditionally applies `ForWorkspace` whenever `resolved.Workspace` is non-empty, even for selectors that resolved to global or agent-global scope.
- Evidence: the current function resolves the selector and then immediately scopes the base store to the workspace before checking whether the selector is actually workspace-scoped.
- Fix plan: only apply `ForWorkspace` when the resolved selector requires workspace-local recall state, while preserving agent-global and explicit global selectors on the root store.
- Resolution: implemented in `internal/api/core/memory.go` and validated with targeted handler coverage in `internal/api/httpapi/memory_test.go` because the scoped production file had no local test seam in the batch scope.
- Verification: targeted `go test` for `internal/api/httpapi` passed, and fresh `make verify` passed on 2026-05-06.
