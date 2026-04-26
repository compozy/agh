---
status: resolved
file: internal/daemon/notifier_test.go
line: 471
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vL,comment:PRRC_kwDOR5y4QM67Z0NF
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Convert this test into `t.Run("Should...")` cases per policy.**

Line 412 currently uses a single monolithic test case; please split assertions into named `Should...` subtests (ideally table-driven) to align with repo test standards.

<details>
<summary>Refactor outline</summary>

```diff
 func TestScopeWorkspaceHookDeclsOnlyInjectsSupportedMatcherFields(t *testing.T) {
     t.Parallel()
-    // single scenario with many assertions
+    testCases := []struct {
+        name  string
+        event hookspkg.HookEvent
+        // expected matcher behavior fields...
+    }{
+        {name: "ShouldInjectWorkspaceIDAndRootForSessionHooks", event: hookspkg.HookSessionPostCreate},
+        {name: "ShouldInjectOnlyWorkspaceIDForTaskRunHooks", event: hookspkg.HookTaskRunEnqueued},
+        {name: "ShouldNotInjectWorkspaceFieldsForMessageHooks", event: hookspkg.HookMessageDelta},
+    }
+    for _, tc := range testCases {
+        tc := tc
+        t.Run(tc.name, func(t *testing.T) {
+            t.Parallel()
+            // scenario setup + focused assertions
+        })
+    }
 }
```
</details>


As per coding guidelines, `**/*_test.go`: "Table-driven tests with subtests (t.Run) as default pattern" and "MUST use t.Run("Should...") pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/notifier_test.go` around lines 412 - 471, Split the
monolithic TestScopeWorkspaceHookDeclsOnlyInjectsSupportedMatcherFields into
table-driven subtests using t.Run("Should ...") entries: build a table of cases
(e.g., "Should inject workspace fields for session", "Should only inject
WorkspaceID for task-run", "Should not inject workspace fields for message",
"Should not mutate original decls") that each call scopeWorkspaceHookDecls with
the same inputs and assert the specific Matcher fields and
hookspkg.ValidateMatcherForEvent results; keep the original decls and resolved
values for reuse, reference the function under test scopeWorkspaceHookDecls and
use hookspkg.ValidateMatcherForEvent in each subtest, and ensure the final case
verifies the original decls were not mutated.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestScopeWorkspaceHookDeclsOnlyInjectsSupportedMatcherFields` combines multiple independent assertions in one monolithic body. Fix by splitting the session, task-run, message, and immutability checks into table-driven `Should ...` subtests while reusing the same source declarations.
