---
status: resolved
file: internal/cli/daemon_wait_test.go
line: 143
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TjvR,comment:PRRC_kwDOR5y4QM624LJR
---

# Issue 008: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap this new test case in a `t.Run("Should...")` subtest.**

This newly added case is top-level only; project test rules require the `t.Run("Should...")` pattern for all test cases.

<details>
<summary>♻️ Proposed refactor</summary>

```diff
 func TestWaitForDaemonStopClearsStaleNetworkSnapshot(t *testing.T) {
 	t.Parallel()
+	t.Run("Should clear stale network snapshot when daemon stops", func(t *testing.T) {
+		t.Parallel()
 
-	deps := newTestDeps(t, stubClient{
+		deps := newTestDeps(t, stubClient{
 		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
 			return DaemonStatus{}, errors.New("daemon unavailable")
 		},
-	})
-	deps.pollInterval = time.Millisecond
-	deps.stopTimeout = 100 * time.Millisecond
-	deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
+		})
+		deps.pollInterval = time.Millisecond
+		deps.stopTimeout = 100 * time.Millisecond
+		deps.readDaemonInfo = func(string) (aghdaemon.Info, error) {
 		return aghdaemon.Info{
 			PID:       42,
 			StartedAt: fixedTestNow,
 			Network: &aghdaemon.NetworkInfo{
 				Enabled:      true,
 				Status:       "running",
 				ListenerHost: "127.0.0.1",
 				ListenerPort: 4522,
 			},
 		}, nil
-	}
+		}
 
-	aliveChecks := 0
-	deps.processAlive = func(int) bool {
+		aliveChecks := 0
+		deps.processAlive = func(int) bool {
 		aliveChecks++
 		return aliveChecks < 2
-	}
+		}
 
-	runtime, err := loadRuntimeContext(deps)
-	if err != nil {
+		runtime, err := loadRuntimeContext(deps)
+		if err != nil {
 		t.Fatalf("loadRuntimeContext() error = %v", err)
-	}
-	info := aghdaemon.Info{
+		}
+		info := aghdaemon.Info{
 		PID:       42,
 		StartedAt: fixedTestNow,
 		Network: &aghdaemon.NetworkInfo{
 			Enabled:      true,
 			Status:       "running",
 			ListenerHost: "127.0.0.1",
 			ListenerPort: 4522,
 		},
-	}
+		}
 
-	status, err := waitForDaemonStop(testutil.Context(t), deps, runtime, info)
-	if err != nil {
+		status, err := waitForDaemonStop(testutil.Context(t), deps, runtime, info)
+		if err != nil {
 		t.Fatalf("waitForDaemonStop() error = %v", err)
-	}
-	if status.Network != nil {
+		}
+		if status.Network != nil {
 		t.Fatalf("waitForDaemonStop() network = %#v, want nil after stop", status.Network)
-	}
+		}
+	})
 }
```
</details>

  
As per coding guidelines, "`**/*_test.go`: MUST use t.Run(\"Should...\") pattern for ALL test cases" and "`**/*_test.go`: Use table-driven tests with subtests (`t.Run`) as default in Go tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/daemon_wait_test.go` around lines 92 - 143, Wrap the existing
test body of TestWaitForDaemonStopClearsStaleNetworkSnapshot in a t.Run subtest
(e.g. t.Run("Should clear stale network snapshot when daemon stops", func(t
*testing.T) { ... })) and move the t.Parallel() call into that subtest function
so the subtest is run in parallel; keep all existing setup, assertions and
references to waitForDaemonStop, loadRuntimeContext, deps, info, and status
unchanged inside the subtest.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestWaitForDaemonStopClearsStaleNetworkSnapshot` is a standalone case without the repository’s preferred `Should...` subtest structure.
- Fix plan: Wrap the body in a `t.Run("Should...")` subtest and keep the strengthened stop-state assertions inside that focused case.
