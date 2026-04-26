---
status: resolved
file: internal/daemon/harness_context_test.go
line: 129
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vJ,comment:PRRC_kwDOR5y4QM67Z0ND
---

# Issue 005: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use `Should...` names for the newly added matrix subtests.**

The new case names should follow the required subtest naming pattern used by `t.Run(tc.name, ...)`.

<details>
<summary>Proposed diff</summary>

```diff
-            name: "coordinator startup session resolves coordinator policy",
+            name: "Should resolve coordinator policy for coordinator startup session",
...
-            name: "spawned worker network turn resolves spawned policy",
+            name: "Should resolve spawned policy for spawned worker network turn",
```
</details>



As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".


Also applies to: 128-162

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/harness_context_test.go` around lines 98 - 129, The test-case
name strings in the table-driven tests (the name field used by t.Run(tc.name,
...)) do not follow the required "Should..." subtest naming pattern; update the
name values for the new matrix entries (e.g., the case currently titled
"coordinator startup session resolves coordinator policy" and the subsequent
"spawned worker network turn resolves spawned policy") to begin with "Should"
(for example "Should resolve coordinator policy on startup session" and "Should
resolve spawned policy for spawned worker network turn"), leaving the rest of
the HarnessResolutionInput, HarnessSessionInput, HarnessTurnRequest,
wantSections/wantTags, and other fields (symbols: name, HarnessResolutionInput,
HarnessSessionInput, HarnessTurnRequest, t.Run) unchanged so t.Run(tc.name, ...)
uses the new "Should..." names.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: The table-driven cases in `TestHarnessContextResolverResolve` are executed through `t.Run(tc.name, ...)`, but the two new case names do not start with `Should`. Fix by renaming them to descriptive `Should ...` names while leaving inputs and expected outputs unchanged.
