---
status: resolved
file: internal/store/globaldb/global_db_task_test.go
line: 1112
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vR,comment:PRRC_kwDOR5y4QM67Z0NM
---

# Issue 019: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap these new test cases in `t.Run("Should...")` subtests.**

Both new cases are written as direct top-level test bodies. This repo’s test policy requires `t.Run("Should...")` for all test cases.



<details>
<summary>Suggested structure</summary>

```diff
 func TestGlobalDBUpdateTaskRunAllowsQueuedSessionRelease(t *testing.T) {
     t.Parallel()
+    t.Run("Should allow clearing lease fields when transitioning claimed run back to queued", func(t *testing.T) {
+        t.Parallel()
+        // existing test body
+    })
 }

 func TestGlobalDBUpdateTaskRunRejectsActiveSessionClear(t *testing.T) {
     t.Parallel()
+    t.Run("Should reject clearing session binding for active runs", func(t *testing.T) {
+        t.Parallel()
+        // existing test body
+    })
 }
```
</details>

As per coding guidelines, "**MUST use t.Run("Should...") pattern for ALL test cases**."


Also applies to: 1114-1136

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_task_test.go` around lines 1070 - 1112, The
test body in TestGlobalDBUpdateTaskRunAllowsQueuedSessionRelease must be wrapped
as a named subtest per project policy; replace the current top-level assertions
with t.Run("Should release queued session when requeued", func(t *testing.T) {
t.Parallel(); /* move the current body here unchanged */ }) so the existing
setup, CreateTask/CreateTaskRun, UpdateTaskRun and assertions remain identical
but run as a subtest; do the same for the other new test referenced around lines
1114-1136 (wrap its body in a t.Run("Should ...", func(t *testing.T) {
t.Parallel(); ... })) so every test case uses the t.Run("Should...") pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestGlobalDBUpdateTaskRunAllowsQueuedSessionRelease` and `TestGlobalDBUpdateTaskRunRejectsActiveSessionClear` are direct test bodies. Fix by wrapping each body in a named `Should ...` subtest while preserving the existing task/run setup and assertions.
