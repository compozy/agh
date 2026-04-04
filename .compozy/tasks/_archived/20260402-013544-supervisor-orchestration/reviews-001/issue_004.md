---
status: resolved
file: internal/prompt/tools_test.go
line: 59
severity: low
author: claude-code
---

# Issue 004: tools_test.go uses reflect.DeepEqual instead of slices.Equal



## Review Comment

`TestToolsForAgentType` uses `reflect.DeepEqual` for comparing string slices:

```go
if !reflect.DeepEqual(got, tc.want) {
```

The rest of the codebase (e.g., `api_lifecycle_test.go:431`, `session_manager_test.go:300`) uses `slices.Equal` from the standard library for the same purpose. `slices.Equal` is type-safe and does not require the `reflect` import.

Replace with:

```go
if !slices.Equal(got, tc.want) {
```

And update the import from `"reflect"` to `"slices"`.

## Triage

- Decision: `valid`
- Notes:
  This is low severity, but it is a legitimate cleanup in the touched test surface. `slices.Equal` is the standard-library slice comparator used elsewhere in this package area, removes the unnecessary `reflect` import, and keeps the test idiom consistent with the rest of the repository.
  Resolved by switching `internal/prompt/tools_test.go` from `reflect.DeepEqual` to `slices.Equal`.
