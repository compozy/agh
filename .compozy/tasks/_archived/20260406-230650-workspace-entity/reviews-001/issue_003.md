---
status: resolved
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

- Decision: `valid`
- Root cause: `uniqueWorkspaceName` is duplicated verbatim in `internal/workspace/resolver.go` and `internal/store/schema.go`. The migration path and runtime registration path therefore rely on separate copies of the same naming rule.
- Fix plan: extract the helper into the `workspace` package as a shared exported function and switch both call sites to the shared implementation so migration and runtime registration stay consistent.

## Resolution

- Extracted the naming algorithm into `workspace.UniqueWorkspaceName(...)` and switched both the resolver and the schema migration flow to the shared helper.
- Added direct helper coverage in the workspace test suite and updated store helper coverage to use the shared function.
- Verified with targeted package tests and `make verify`.
