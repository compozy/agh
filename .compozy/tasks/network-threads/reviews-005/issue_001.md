---
provider: coderabbit
pr: "105"
round: 5
round_created_at: 2026-05-06T02:28:33.373448Z
status: resolved
file: internal/acp/client_test.go
line: 1125
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_033t,comment:PRRC_kwDOR5y4QM6-SXvV
---

# Issue 001: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Wrap this new test in a `t.Run("Should ...")` subtest and add guaranteed cleanup.**

This new case currently uses a flat top-level body and can leak the helper process if the test fails before Line 1117. Wrap it in a `Should...` subtest and register cleanup immediately after process start.

<details>
<summary>Suggested patch</summary>

```diff
 func TestPromptStopDoesNotEmitRuntimeError(t *testing.T) {
-	t.Parallel()
-
-	driver := New()
-	proc := startHelperProcess(t, driver, "block_prompt_until_cancel", "", StartOpts{})
-
-	eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
-		TurnID:  "turn-stop",
-		Message: "block until stopped",
-	})
-	if err != nil {
-		t.Fatalf("Prompt() error = %v", err)
-	}
-
-	select {
-	case event := <-eventsCh:
-		if got, want := event.Type, EventTypeAgentMessage; got != want {
-			t.Fatalf("first prompt event = %q, want %q", got, want)
-		}
-	case <-time.After(5 * time.Second):
-		t.Fatal("timeout waiting for blocking prompt to start")
-	}
-
-	if err := driver.Stop(testutil.Context(t), proc); err != nil {
-		t.Fatalf("Stop() error = %v", err)
-	}
-	for _, event := range collectEvents(t, eventsCh) {
-		if event.Type == EventTypeError {
-			t.Fatalf("prompt events contain %q after explicit stop: %#v", EventTypeError, event)
-		}
-	}
+	t.Run("Should not emit runtime error after explicit stop", func(t *testing.T) {
+		t.Parallel()
+
+		driver := New()
+		proc := startHelperProcess(t, driver, "block_prompt_until_cancel", "", StartOpts{})
+		t.Cleanup(func() { stopProcess(t, driver, proc) })
+
+		eventsCh, err := driver.Prompt(testutil.Context(t), proc, PromptRequest{
+			TurnID:  "turn-stop",
+			Message: "block until stopped",
+		})
+		if err != nil {
+			t.Fatalf("Prompt() error = %v", err)
+		}
+
+		select {
+		case event := <-eventsCh:
+			if got, want := event.Type, EventTypeAgentMessage; got != want {
+				t.Fatalf("first prompt event = %q, want %q", got, want)
+			}
+		case <-time.After(5 * time.Second):
+			t.Fatal("timeout waiting for blocking prompt to start")
+		}
+
+		if err := driver.Stop(testutil.Context(t), proc); err != nil {
+			t.Fatalf("Stop() error = %v", err)
+		}
+		for _, event := range collectEvents(t, eventsCh) {
+			if event.Type == EventTypeError {
+				t.Fatalf("prompt events contain %q after explicit stop: %#v", EventTypeError, event)
+			}
+		}
+	})
 }
```
</details>

 
As per coding guidelines, "Use `t.Run('Should ...')` pattern for Go test subtests instead of flat test structures."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/acp/client_test.go` around lines 1094 - 1125, The
TestPromptStopDoesNotEmitRuntimeError function should be converted to use a
t.Run("Should ...") subtest and register guaranteed cleanup right after the
helper process is started: after calling startHelperProcess(t, driver,
"block_prompt_until_cancel", "", StartOpts{}) immediately call t.Cleanup to
stop/terminate the proc (e.g., call driver.Stop or equivalent) to avoid leaking
the helper on failures, then keep the existing logic (Prompt, select, Stop,
collectEvents) inside the t.Run body; ensure references to New(),
startHelperProcess, driver.Stop, eventsCh and collectEvents are preserved and
used in the subtest so cleanup always runs even if the test fails early.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TestPromptStopDoesNotEmitRuntimeError` starts a helper process and only stops it on the happy path. If `Prompt()` or the first-event wait fails, the helper can leak until the test process exits.
  - The file already follows `t.Run("Should ...")` in nearby tests, so this case should match the local convention.
  - Fix plan: wrap the body in a named subtest, register `t.Cleanup` immediately after `startHelperProcess`, and keep the explicit `driver.Stop(...)` assertion in the test body.

## Resolution

- Wrapped the case in `t.Run("Should not emit runtime error after explicit stop", ...)` and registered cleanup immediately after starting the helper process.
- Kept the explicit `driver.Stop(...)` assertion in the test body while avoiding the nested parallel scheduling that reintroduced the stop race.
- Verified with fresh full `make verify` (passed).
