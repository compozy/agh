---
status: resolved
file: internal/memory/store.go
line: 424
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575hox,comment:PRRC_kwDOR5y4QM65B3LS
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Normalize `HealthStats` workspace filters before deduping them.**

This path bypasses `normalizeScopeAndWorkspace` and only trims the incoming strings. Supplying both `/repo/app` and `/repo/app/.agh/memory` will create two filters for the same workspace, and the memory-dir form will not match catalog rows keyed by workspace root.  


<details>
<summary>🐛 Proposed fix</summary>

```diff
 	for _, workspace := range workspaces {
-		trimmed := strings.TrimSpace(workspace)
+		trimmed := deriveWorkspaceRoot(cleanDirPath(workspace))
 		if trimmed == "" {
 			continue
 		}
 		if _, exists := seen[trimmed]; exists {
 			continue
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/memory/store.go` around lines 415 - 424, The loop building
catalogFilter for ScopeWorkspace should normalize each incoming workspace string
before deduping: call normalizeScopeAndWorkspace (or the same normalization
logic) on the trimmed workspace to obtain the canonical workspaceRoot/scope,
handle/skip any errors or empty results, then use that normalized workspaceRoot
for the seen map and for constructing catalogFilter{scope: ScopeWorkspace,
workspaceRoot: normalized}. Replace the current raw-trim-and-dedupe flow
(variables seen, trimmed, filters append) so duplicates like "/repo/app" and
"/repo/app/.agh/memory" collapse to the same normalized workspaceRoot.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `Store.HealthStats` currently trims workspace inputs but does not canonicalize them before deduping or building `catalogFilter` values.
  - The catalog keys workspace rows by workspace root, while callers can provide either the workspace root or the workspace memory directory (`<workspace>/.agh/memory`).
  - That mismatch can produce duplicate filters for the same workspace and can miss ready-state or catalog rows keyed by the canonical root.
  - Fix approach: normalize each incoming workspace path to a canonical workspace root before dedupe/filter construction, then use that normalized root for `seen` and `catalogFilter`.
  - A focused regression test is required in `internal/memory/store_test.go` even though it is outside the listed code files, because the production bug is in `internal/memory/store.go` and needs coverage for root-vs-memory-dir inputs.
  - Resolved by introducing canonical workspace-root normalization in `internal/memory/store.go` before HealthStats dedupe/filter construction.
  - Verified with `go test ./internal/memory -run TestStoreNormalizesExplicitWorkspacePaths` and the full `make verify` gate.
