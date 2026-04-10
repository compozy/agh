---
status: resolved
file: internal/acp/client_test.go
line: 677
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QU5_,comment:PRRC_kwDOR5y4QM620Apu
---

# Issue 006: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Wrap this test body in a `t.Run("Should...")` subtest to satisfy test conventions.**

This test currently skips the required subtest pattern for Go tests in this repo.

<details>
<summary>♻️ Proposed refactor</summary>

```diff
 func TestStopManagedProcessRespectsContext(t *testing.T) {
 	t.Parallel()
-
-	driver := New(WithStopTimeout(5 * time.Second))
-	managed, err := subprocess.Launch(context.Background(), subprocess.LaunchConfig{
-		Command:          "sh",
-		Args:             []string{"-c", "sleep 30"},
-		DisableTransport: true,
-		ShutdownTimeout:  time.Second,
-	})
-	if err != nil {
-		t.Fatalf("Launch() error = %v", err)
-	}
-
-	proc := &AgentProcess{
-		managed: managed,
-		done:    make(chan struct{}),
-	}
-	go proc.waitForExit()
-	t.Cleanup(func() {
-		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
-		defer cancel()
-		_ = managed.Shutdown(cleanupCtx)
-		<-proc.Done()
-	})
-
-	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
-	defer cancel()
-
-	startedAt := time.Now()
-	err = driver.Stop(stopCtx, proc)
-	if !errors.Is(err, context.DeadlineExceeded) {
-		t.Fatalf("Stop() error = %v, want context deadline exceeded", err)
-	}
-	if elapsed := time.Since(startedAt); elapsed > time.Second {
-		t.Fatalf("Stop() elapsed = %v, want <= 1s", elapsed)
-	}
+	t.Run("Should return deadline exceeded when managed process shutdown exceeds stop context", func(t *testing.T) {
+		t.Parallel()
+
+		driver := New(WithStopTimeout(5 * time.Second))
+		managed, err := subprocess.Launch(context.Background(), subprocess.LaunchConfig{
+			Command:          "sh",
+			Args:             []string{"-c", "sleep 30"},
+			DisableTransport: true,
+			ShutdownTimeout:  time.Second,
+		})
+		if err != nil {
+			t.Fatalf("Launch() error = %v", err)
+		}
+
+		proc := &AgentProcess{
+			managed: managed,
+			done:    make(chan struct{}),
+		}
+		go proc.waitForExit()
+		t.Cleanup(func() {
+			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
+			defer cancel()
+			_ = managed.Shutdown(cleanupCtx)
+			<-proc.Done()
+		})
+
+		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
+		defer cancel()
+
+		startedAt := time.Now()
+		err = driver.Stop(stopCtx, proc)
+		if !errors.Is(err, context.DeadlineExceeded) {
+			t.Fatalf("Stop() error = %v, want context deadline exceeded", err)
+		}
+		if elapsed := time.Since(startedAt); elapsed > time.Second {
+			t.Fatalf("Stop() elapsed = %v, want <= 1s", elapsed)
+		}
+	})
 }
```
</details>

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (t.Run) as default in Go tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client_test.go` around lines 640 - 677, Wrap the existing test
body of TestStopManagedProcessRespectsContext in a t.Run subtest (e.g.,
t.Run("Should stop managed process respecting context", func(t *testing.T) { ...
})) and move t.Parallel() into that subtest; keep all setup
(New(WithStopTimeout...), subprocess.Launch, AgentProcess creation, go
proc.waitForExit(), t.Cleanup shutdown, stopCtx/cancel, timing and assertions
against driver.Stop and context.DeadlineExceeded) unchanged and inside the
subtest closure so the test follows the repo's t.Run("Should...") convention
while still exercising driver.Stop, subprocess.Launch, AgentProcess.waitForExit
and proc.Done as before.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding is reasonable. This test is a single scenario and can follow the same `t.Run("Should...")` convention already used elsewhere in the repo without changing its behavior.
  - Root cause: the test was added as a direct top-level body rather than a named subtest.
  - Fix approach: wrap the existing body in one `Should...` subtest and move `t.Parallel()` into the subtest so the assertions and cleanup remain unchanged.
  - Resolution: implemented in `internal/acp/client_test.go` and verified with focused package tests plus `make verify`.
