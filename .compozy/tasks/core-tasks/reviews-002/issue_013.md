---
status: resolved
file: internal/extension/host_api_test.go
line: 1770
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564LgU,comment:PRRC_kwDOR5y4QM63o2QR
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use `t.Run("Should...")` naming for all subtests in new task test tables.**

Several new subtest names (`CreateDenied`, `UnknownWorkspace`, `GetTask`, etc.) don’t follow the required `Should...` convention.

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".



Also applies to: 2308-2331, 2344-2396, 2421-2448, 2472-2499, 2525-2547

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_test.go` around lines 1740 - 1770, The subtest
names in the table (the slice named tests and its entries used in t.Run(tt.name,
...)) don't follow the required "Should..." pattern; update each test case's
name field (e.g., "CreateDenied", "UpdateDenied", "RunStartDenied" and all
similar names in the other blocks) to be prefixed with "Should" and a
descriptive action (e.g., "Should deny create", "Should deny update", "Should
deny run start"), and keep the test body unchanged (the call to env.call and the
assertCapabilityDenied assertions). Ensure you update the string used in
t.Run(...) to the new "Should..." value so all subtests comply with the naming
convention.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  Several new table-driven subtests in `internal/extension/host_api_test.go` use names like `CreateDenied`, `UnknownWorkspace`, and `GetTask`, which do not follow the repository's `Should...` convention.
  Root cause: the new task-related table rows were added with terse identifiers rather than full `Should...` test-case names.
  Planned fix: rename the affected table-entry names so every `t.Run(tt.name, ...)` in the touched task tables uses a `Should...` label.

## Resolution

- Renamed the affected task-related table-driven subtests in `internal/extension/host_api_test.go` so each `t.Run(tt.name, ...)` now uses the required `Should...` naming convention.
