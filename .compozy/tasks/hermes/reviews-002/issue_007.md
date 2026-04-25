---
status: resolved
file: internal/daemon/daemon_test.go
line: 2276
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59mLiY,comment:PRRC_kwDOR5y4QM67SmDd
---

# Issue 007: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Wrap this test case in a `t.Run("Should...")` subtest.**

Line 2236 adds a new test case without the required `t.Run("Should...")` pattern for test cases.



<details>
<summary>Proposed adjustment</summary>

```diff
 func TestRunShutsDownWhenObserverRetentionStartFails(t *testing.T) {
	t.Parallel()
-
-	homePaths := testHomePaths(t)
-	cfg := testConfig(t, homePaths)
-	retentionErr := errors.New("retention start failed")
-	observer := &failingRetentionObserver{startErr: retentionErr}
-	httpShutdown := false
-	udsShutdown := false
-
-	d := newTestDaemon(t, homePaths, &cfg)
-	d.acquireLock = func(path string, _ int) (*Lock, error) {
-		return &Lock{path: path}, nil
-	}
-	d.openRegistry = func(context.Context, string) (Registry, error) {
-		return &recordingRegistry{path: homePaths.DatabaseFile}, nil
-	}
-	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
-		return &fakeSessionManager{}, nil
-	}
-	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
-		return observer, nil
-	}
-	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
-		return &fakeServer{name: "http", onShutdown: func() { httpShutdown = true }}, nil
-	}
-	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
-		return &fakeServer{name: "uds", onShutdown: func() { udsShutdown = true }}, nil
-	}
-
-	err := d.Run(context.Background())
-	if !errors.Is(err, retentionErr) {
-		t.Fatalf("Run() error = %v, want retention start failure", err)
-	}
-	if !observer.shutdownCalled {
-		t.Fatal("observer.ShutdownRetention() was not called")
-	}
-	if !httpShutdown || !udsShutdown {
-		t.Fatalf("server shutdown flags = http:%v uds:%v, want both true", httpShutdown, udsShutdown)
-	}
+	t.Run("ShouldShutDownWhenObserverRetentionStartFails", func(t *testing.T) {
+		t.Parallel()
+
+		homePaths := testHomePaths(t)
+		cfg := testConfig(t, homePaths)
+		retentionErr := errors.New("retention start failed")
+		observer := &failingRetentionObserver{startErr: retentionErr}
+		httpShutdown := false
+		udsShutdown := false
+
+		d := newTestDaemon(t, homePaths, &cfg)
+		d.acquireLock = func(path string, _ int) (*Lock, error) {
+			return &Lock{path: path}, nil
+		}
+		d.openRegistry = func(context.Context, string) (Registry, error) {
+			return &recordingRegistry{path: homePaths.DatabaseFile}, nil
+		}
+		d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
+			return &fakeSessionManager{}, nil
+		}
+		d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
+			return observer, nil
+		}
+		d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
+			return &fakeServer{name: "http", onShutdown: func() { httpShutdown = true }}, nil
+		}
+		d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
+			return &fakeServer{name: "uds", onShutdown: func() { udsShutdown = true }}, nil
+		}
+
+		err := d.Run(context.Background())
+		if !errors.Is(err, retentionErr) {
+			t.Fatalf("Run() error = %v, want retention start failure", err)
+		}
+		if !observer.shutdownCalled {
+			t.Fatal("observer.ShutdownRetention() was not called")
+		}
+		if !httpShutdown || !udsShutdown {
+			t.Fatalf("server shutdown flags = http:%v uds:%v, want both true", httpShutdown, udsShutdown)
+		}
+	})
 }
```
</details>

As per coding guidelines, "MUST use t.Run("Should...") pattern for ALL test cases".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestRunShutsDownWhenObserverRetentionStartFails(t *testing.T) {
	t.Parallel()
	t.Run("ShouldShutDownWhenObserverRetentionStartFails", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		cfg := testConfig(t, homePaths)
		retentionErr := errors.New("retention start failed")
		observer := &failingRetentionObserver{startErr: retentionErr}
		httpShutdown := false
		udsShutdown := false

		d := newTestDaemon(t, homePaths, &cfg)
		d.acquireLock = func(path string, _ int) (*Lock, error) {
			return &Lock{path: path}, nil
		}
		d.openRegistry = func(context.Context, string) (Registry, error) {
			return &recordingRegistry{path: homePaths.DatabaseFile}, nil
		}
		d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
			return &fakeSessionManager{}, nil
		}
		d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
			return observer, nil
		}
		d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
			return &fakeServer{name: "http", onShutdown: func() { httpShutdown = true }}, nil
		}
		d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
			return &fakeServer{name: "uds", onShutdown: func() { udsShutdown = true }}, nil
		}

		err := d.Run(context.Background())
		if !errors.Is(err, retentionErr) {
			t.Fatalf("Run() error = %v, want retention start failure", err)
		}
		if !observer.shutdownCalled {
			t.Fatal("observer.ShutdownRetention() was not called")
		}
		if !httpShutdown || !udsShutdown {
			t.Fatalf("server shutdown flags = http:%v uds:%v, want both true", httpShutdown, udsShutdown)
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

In `@internal/daemon/daemon_test.go` around lines 2236 - 2276, Wrap the entire
test body of TestRunShutsDownWhenObserverRetentionStartFails in a t.Run subtest
with a descriptive name starting with "Should..." (e.g., t.Run("Should shutdown
when observer retention start fails", func(t *testing.T) { ... })), ensuring you
move the existing t.Parallel() and all setup/assertions into that subtest block
so the test follows the required t.Run("Should...") pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestRunShutsDownWhenObserverRetentionStartFails` has a direct top-level body instead of a named `t.Run("Should...")` subtest, diverging from the repository test-case pattern.
- Fix approach: wrap the existing setup and assertions in a `Should...` subtest and keep parallelism scoped inside the subtest.
