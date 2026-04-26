---
status: resolved
file: internal/task/manager_integration_test.go
line: 1153
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vU,comment:PRRC_kwDOR5y4QM67Z0NQ
---

# Issue 022: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap this case in a `t.Run("Should ...")` subtest.**

The new test is top-level only; this repo requires the explicit `Should...` subtest pattern.

<details>
<summary>Proposed fix</summary>

```diff
 func TestTaskManagerRecoverRunOnBootRequeuesBoundRunWithGlobalDB(t *testing.T) {
  t.Parallel()
+ t.Run("Should requeue a bound run and clear session binding on boot recovery", func(t *testing.T) {
+   t.Parallel()

-	ctx := testutil.Context(t)
-	db := openTaskManagerGlobalDB(t)
-	manager := newTaskManagerIntegration(t, db)
-	operator, err := taskpkg.DeriveHumanActorContext("operator", taskpkg.OriginKindCLI, "agh task run")
-	if err != nil {
-		t.Fatalf("DeriveHumanActorContext() error = %v", err)
-	}
-	agent, err := taskpkg.DeriveAgentSessionActorContext("sess-stale-boot")
-	if err != nil {
-		t.Fatalf("DeriveAgentSessionActorContext() error = %v", err)
-	}
-	daemon, err := taskpkg.DeriveDaemonActorContext("boot-recovery", "daemon.boot")
-	if err != nil {
-		t.Fatalf("DeriveDaemonActorContext() error = %v", err)
-	}
+		ctx := testutil.Context(t)
+		db := openTaskManagerGlobalDB(t)
+		manager := newTaskManagerIntegration(t, db)
+		operator, err := taskpkg.DeriveHumanActorContext("operator", taskpkg.OriginKindCLI, "agh task run")
+		if err != nil {
+			t.Fatalf("DeriveHumanActorContext() error = %v", err)
+		}
+		agent, err := taskpkg.DeriveAgentSessionActorContext("sess-stale-boot")
+		if err != nil {
+			t.Fatalf("DeriveAgentSessionActorContext() error = %v", err)
+		}
+		daemon, err := taskpkg.DeriveDaemonActorContext("boot-recovery", "daemon.boot")
+		if err != nil {
+			t.Fatalf("DeriveDaemonActorContext() error = %v", err)
+		}

-	// ...existing assertions...
+		// ...existing assertions...
+	})
 }
```
</details>



As per coding guidelines, `**/*_test.go`: "MUST use t.Run('Should...') pattern for ALL test cases".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager_integration_test.go` around lines 1091 - 1153, The test
function TestTaskManagerRecoverRunOnBootRequeuesBoundRunWithGlobalDB must be
wrapped in a t.Run("Should ...") subtest: replace the top-level t.Parallel() and
direct test body with a single t.Run("Should requeue bound run on boot and
release session binding", func(t *testing.T) { t.Parallel(); /* existing test
body */ }), keeping all existing setup and assertions intact (references:
TestTaskManagerRecoverRunOnBootRequeuesBoundRunWithGlobalDB, manager.CreateTask,
manager.EnqueueRun, manager.ClaimNextRun, manager.RecoverRunOnBoot,
db.GetTaskRun) so the repo-wide "Should..." subtest pattern is satisfied.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `TestTaskManagerRecoverRunOnBootRequeuesBoundRunWithGlobalDB` has a top-level body with no `Should ...` subtest. The required test shape is to wrap the existing setup and assertions in one named subtest while preserving the integration flow and parallelism.
