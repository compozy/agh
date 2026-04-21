---
status: resolved
file: internal/network/capability_catalog_test.go
line: 63
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58PQ7E,comment:PRRC_kwDOR5y4QM65eHnc
---

# Issue 003: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use `Should...` subtest names to match the required test-case pattern.**

Line 62 and Line 121 currently use non-`Should...` subtest names (for example, `"filter absent"`). Please rename all subtests in these tables to `Should...` format for consistency with repo standards.


As per coding guidelines, `MUST use t.Run("Should...") pattern for ALL test cases`.


Also applies to: 120-123

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/capability_catalog_test.go` around lines 61 - 63, Rename the
subtest names to the required "Should..." pattern by updating the test table
entries and the t.Run call that iterates over tests; locate the tests table
(variable tests) and the t.Run(tc.name, func(t *testing.T) { ... }) invocation
in capability_catalog_test.go and ensure each tc.name is a "Should ..." string
(or change the t.Run to use fmt.Sprintf("Should %s", tc.name) so all subtests
conform), including entries referenced around the current tc loop and the ones
at lines ~120-123.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the new table-driven subtests in `internal/network/capability_catalog_test.go` use lowercase descriptive names like `"filter absent"` instead of the repository's required `Should...` naming convention.
- Why this is valid: the repo standards explicitly require `t.Run("Should...")`-style test cases, and this file is already using table-driven subtests where renaming the cases is the minimal conforming change.
- Fix approach: rename the table case names to `Should...` strings while preserving the current assertions and test structure.

## Resolution

- Renamed every table case in `internal/network/capability_catalog_test.go` to `Should...` form without changing the assertions or coverage.
- Verification:
  - `go test ./internal/network -run 'TestParseWhoisCapabilityDiscoveryRequestCapabilityFilterPresence|TestProjectWhoisCapabilityCatalogDistinguishesAbsentAndExplicitEmptyFilters' -count=1`
  - `make verify`
