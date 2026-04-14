---
status: resolved
file: internal/automation/manager_test.go
line: 622
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562an_,comment:PRRC_kwDOR5y4QM63mgRQ
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use required `t.Run("Should...")` structure for this new test case.**

`TestManagerSessionTaskActorLifecycle` is added as a flat test body. This repo requires test cases to be expressed with `t.Run("Should...")`.

<details>
<summary>♻️ Proposed refactor</summary>

```diff
 func TestManagerSessionTaskActorLifecycle(t *testing.T) {
 	t.Parallel()
-
-	h := newManagerHarness(t)
-	manager := h.newManager(t, aghconfig.AutomationConfig{
-		Enabled:           true,
-		Timezone:          DefaultTimezone,
-		MaxConcurrentJobs: DefaultMaxConcurrentJobs,
-		DefaultFireLimit:  DefaultFireLimitConfig(),
-	})
-
-	actor, err := taskpkg.DeriveAutomationLinkedAgentSessionActorContext("sess-actor-1", "run:run-1")
-	if err != nil {
-		t.Fatalf("DeriveAutomationLinkedAgentSessionActorContext() error = %v", err)
-	}
-	if err := manager.RecordAutomationSessionTaskActor("sess-actor-1", actor); err != nil {
-		t.Fatalf("RecordAutomationSessionTaskActor() error = %v", err)
-	}
-
-	loaded, err := manager.TaskActorContextForSession("sess-actor-1")
-	if err != nil {
-		t.Fatalf("TaskActorContextForSession() error = %v", err)
-	}
-	if loaded != actor {
-		t.Fatalf("TaskActorContextForSession() = %#v, want %#v", loaded, actor)
-	}
-
-	manager.DeleteAutomationSessionTaskActor("sess-actor-1")
-	if _, err := manager.TaskActorContextForSession("sess-actor-1"); !errors.Is(err, ErrSessionTaskActorNotFound) {
-		t.Fatalf("TaskActorContextForSession(after delete) error = %v, want ErrSessionTaskActorNotFound", err)
-	}
+	t.Run("Should record, load, and delete a session task actor", func(t *testing.T) {
+		t.Parallel()
+
+		h := newManagerHarness(t)
+		manager := h.newManager(t, aghconfig.AutomationConfig{
+			Enabled:           true,
+			Timezone:          DefaultTimezone,
+			MaxConcurrentJobs: DefaultMaxConcurrentJobs,
+			DefaultFireLimit:  DefaultFireLimitConfig(),
+		})
+
+		actor, err := taskpkg.DeriveAutomationLinkedAgentSessionActorContext("sess-actor-1", "run:run-1")
+		if err != nil {
+			t.Fatalf("DeriveAutomationLinkedAgentSessionActorContext() error = %v", err)
+		}
+		if err := manager.RecordAutomationSessionTaskActor("sess-actor-1", actor); err != nil {
+			t.Fatalf("RecordAutomationSessionTaskActor() error = %v", err)
+		}
+
+		loaded, err := manager.TaskActorContextForSession("sess-actor-1")
+		if err != nil {
+			t.Fatalf("TaskActorContextForSession() error = %v", err)
+		}
+		if loaded != actor {
+			t.Fatalf("TaskActorContextForSession() = %#v, want %#v", loaded, actor)
+		}
+
+		manager.DeleteAutomationSessionTaskActor("sess-actor-1")
+		if _, err := manager.TaskActorContextForSession("sess-actor-1"); !errors.Is(err, ErrSessionTaskActorNotFound) {
+			t.Fatalf("TaskActorContextForSession(after delete) error = %v, want ErrSessionTaskActorNotFound", err)
+		}
+	})
 }
```
</details>


As per coding guidelines, `**/*_test.go`: MUST use t.Run("Should...") pattern for ALL test cases.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestManagerSessionTaskActorLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("Should record, load, and delete a session task actor", func(t *testing.T) {
		t.Parallel()

		h := newManagerHarness(t)
		manager := h.newManager(t, aghconfig.AutomationConfig{
			Enabled:           true,
			Timezone:          DefaultTimezone,
			MaxConcurrentJobs: DefaultMaxConcurrentJobs,
			DefaultFireLimit:  DefaultFireLimitConfig(),
		})

		actor, err := taskpkg.DeriveAutomationLinkedAgentSessionActorContext("sess-actor-1", "run:run-1")
		if err != nil {
			t.Fatalf("DeriveAutomationLinkedAgentSessionActorContext() error = %v", err)
		}
		if err := manager.RecordAutomationSessionTaskActor("sess-actor-1", actor); err != nil {
			t.Fatalf("RecordAutomationSessionTaskActor() error = %v", err)
		}

		loaded, err := manager.TaskActorContextForSession("sess-actor-1")
		if err != nil {
			t.Fatalf("TaskActorContextForSession() error = %v", err)
		}
		if loaded != actor {
			t.Fatalf("TaskActorContextForSession() = %#v, want %#v", loaded, actor)
		}

		manager.DeleteAutomationSessionTaskActor("sess-actor-1")
		if _, err := manager.TaskActorContextForSession("sess-actor-1"); !errors.Is(err, ErrSessionTaskActorNotFound) {
			t.Fatalf("TaskActorContextForSession(after delete) error = %v, want ErrSessionTaskActorNotFound", err)
		}
	})
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/automation/manager_test.go` around lines 591 - 622, The new test
TestManagerSessionTaskActorLifecycle violates the repo test convention by being
a flat test; wrap its assertions inside a t.Run("Should ...") subtest. Modify
TestManagerSessionTaskActorLifecycle to call t.Run with a descriptive "Should
..." name and move the existing setup and assertions (including calls to
newManager(...), taskpkg.DeriveAutomationLinkedAgentSessionActorContext,
manager.RecordAutomationSessionTaskActor, manager.TaskActorContextForSession and
manager.DeleteAutomationSessionTaskActor) into the subtest function body,
preserving the existing error checks and t.Fatalf calls.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestManagerSessionTaskActorLifecycle` is written as one flat body instead of the required `t.Run("Should...")` structure.
- Fix approach: wrap the existing assertions in one descriptive `Should...` subtest without changing the covered behavior.

## Resolution

- Wrapped the automation session task-actor lifecycle assertions in one descriptive `Should...` subtest.
- Verified in the final `make verify` run.
