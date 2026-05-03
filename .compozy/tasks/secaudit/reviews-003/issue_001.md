---
provider: coderabbit
pr: "90"
round: 3
round_created_at: 2026-05-03T04:20:21.439202Z
status: resolved
file: internal/network/validate_test.go
line: 407
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_KbNG,comment:PRRC_kwDOR5y4QM69Zptx
---

# Issue 001: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Use `Should ...` names for the newly added table cases.**

At Line 328 and the other newly added entries in this block, the case names don’t follow the required `t.Run("Should...")` convention.

<details>
<summary>Suggested rename pattern</summary>

```diff
- name: "future timestamp outside replay window",
+ name: "Should reject future timestamp outside replay window",

- name: "greet task write without proof",
+ name: "Should reject greet task write without proof",

- name: "raw secret in body",
+ name: "Should reject raw secret in body",
```

</details>

   
As per coding guidelines, `**/*_test.go`: MUST use `t.Run("Should...")` pattern for ALL test cases.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/validate_test.go` around lines 327 - 407, The table-driven
test entries use plain names like "future timestamp outside replay window" and
must follow the t.Run "Should ..." pattern; update each case's name field (the
struct entries with name: "...") to a descriptive "Should ..." form (e.g.,
"Should reject future timestamp outside replay window", "Should reject greet
task write without proof", "Should reject raw secret in body", etc.) so all
cases in the table (the entries that mutate Envelope, set wantErr/wantMatch)
conform to the required t.Run naming convention.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding is valid against the current file: `internal/network/validate_test.go` uses table-driven `t.Run(tc.name, ...)` subtests, and the affected block still carries plain labels instead of the required `Should ...` form.
  - Fixing only the three newly added entries would leave the same table mixed and still non-compliant, so the in-scope remediation is to normalize the relevant table-driven case names in this file to `Should ...` descriptions.
  - The scoped remediation also wrapped the remaining direct assertion test in the same file with a `Should ...` subtest after the AGH test-shape checker flagged it during verification.
  - Verification passed with `python3 .agents/skills/agh-test-conventions/scripts/check-test-conventions.py internal/network/validate_test.go`, `go test ./internal/network -count=1 -race`, and full `make verify`.
