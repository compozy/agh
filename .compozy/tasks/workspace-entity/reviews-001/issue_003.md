---
status: pending
file: internal/store/schema.go
line: 656
severity: medium
author: claude-code
provider_ref:
---

# Issue 003: Duplicate uniqueWorkspaceName across workspace/ and store/

## Review Comment

The `uniqueWorkspaceName` function is identically implemented in two places:

- `internal/workspace/resolver.go:1016`
- `internal/store/schema.go:656`

Both take a `rootDir` and a `taken map[string]struct{}` and return a deduplicated name using `filepath.Base` with `-N` suffixes. The store copy is used during legacy schema migration; the workspace copy is used during normal registration.

Having two identical functions risks them diverging silently (e.g., if the naming algorithm changes in one but not the other), producing inconsistent workspace names between migration and runtime flows.

**Suggested fix:** Extract the function to a shared location. Since `workspace/` defines the domain types and the store imports `workspace/`, the function naturally belongs in `workspace/` as an exported helper (e.g., `workspace.UniqueWorkspaceName`). The store migration code can then import it.

## Triage

- Decision: `UNREVIEWED`
- Notes:
