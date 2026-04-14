---
status: resolved
file: internal/daemon/daemon_test.go
line: 783
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564Lf6,comment:PRRC_kwDOR5y4QM63o2Pv
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use a `t.Run("Should...")` subtest wrapper for this new test case.**

This new test is added as a top-level body only, but this repo requires the “Should...” subtest pattern for test cases.

<details>
<summary>🔧 Minimal refactor shape</summary>

```diff
 func TestBootExtensionsKeepsHealthyRegisteredExtensionsAfterPartialStartFailure(t *testing.T) {
 	t.Parallel()
+	t.Run("ShouldKeepHealthyRegisteredExtensionsAfterPartialStartFailure", func(t *testing.T) {
+		t.Parallel()
 
-	// current test body...
+		// current test body...
+	})
 }
```
</details>


As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases" and "Use table-driven tests with subtests (t.Run) as default in Go tests".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestBootExtensionsKeepsHealthyRegisteredExtensionsAfterPartialStartFailure(t *testing.T) {
	t.Parallel()
	t.Run("ShouldKeepHealthyRegisteredExtensionsAfterPartialStartFailure", func(t *testing.T) {
		t.Parallel()

		db := openDaemonTestGlobalDB(t)
		installDaemonTestExtension(t, db, "ext-healthy", daemonTestExtensionOptions{}, true)
		installDaemonTestExtension(t, db, "ext-bad", daemonTestExtensionOptions{}, true)

		var logBuffer bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
		runtime := &fakeExtensionRuntime{
			startErr: errors.New("boom"),
			getFn: func(name string) (*extensionpkg.Extension, error) {
				switch name {
				case "ext-healthy":
					return &extensionpkg.Extension{
						Info: extensionpkg.ExtensionInfo{
							Name:    "ext-healthy",
							Enabled: true,
						},
						Status: extensionpkg.ExtensionStatus{
							Name:       "ext-healthy",
							Enabled:    true,
							Registered: true,
						},
					}, nil
				case "ext-bad":
					return nil, extensionpkg.ErrExtensionNotFound
				default:
					return nil, extensionpkg.ErrExtensionNotFound
				}
			},
		}
		homePaths := testHomePaths(t)
		d := newTestDaemon(t, homePaths, testConfig(t, homePaths))
		d.newExtensionManager = func(extensionManagerDeps) extensionRuntime {
			return runtime
		}

		rebuilds := 0
		state := &bootState{
			logger:   logger,
			registry: db,
			sessions: &fakeSessionManager{},
			observer: &fakeObserver{},
			bridges:  &bridgeRuntime{broker: bridgepkg.NewBroker(nil)},
			hooks: &fakeHookRuntime{
				onRebuild: func(context.Context) error {
					rebuilds++
					return nil
				},
			},
		}
		cleanup := &bootCleanup{}

		if err := d.bootExtensions(testutil.Context(t), state, cleanup); err != nil {
			t.Fatalf("bootExtensions() error = %v, want nil", err)
		}

		if runtime.startCount != 1 {
			t.Fatalf("extension runtime start count = %d, want 1", runtime.startCount)
		}
		if rebuilds != 1 {
			t.Fatalf("hook rebuild count = %d, want 1 after partial start", rebuilds)
		}
		if len(cleanup.fns) != 1 {
			t.Fatalf("cleanup fns = %d, want 1", len(cleanup.fns))
		}
		if state.currentExtensionRuntime() != runtime {
			t.Fatalf("state.extensions = %#v, want runtime", state.currentExtensionRuntime())
		}
		if state.deps.Extensions == nil {
			t.Fatal("state.deps.Extensions = nil, want extension service")
		}
		if state.bridges.extensions != runtime {
			t.Fatalf("state.bridges.extensions = %#v, want runtime", state.bridges.extensions)
		}
		healthy, err := state.deps.Extensions.Status(testutil.Context(t), "ext-healthy")
		if err != nil {
			t.Fatalf("Extensions.Status(ext-healthy) error = %v", err)
		}
		if got, want := healthy.State, "registered"; got != want {
			t.Fatalf("ext-healthy state = %q, want %q", got, want)
		}
		bad, err := state.deps.Extensions.Status(testutil.Context(t), "ext-bad")
		if err != nil {
			t.Fatalf("Extensions.Status(ext-bad) error = %v", err)
		}
		if got, want := bad.State, "enabled"; got != want {
			t.Fatalf("ext-bad state = %q, want %q", got, want)
		}
		if !strings.Contains(logBuffer.String(), "healthy extensions only") {
			t.Fatalf("log output = %q, want partial start continuation message", logBuffer.String())
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

In `@internal/daemon/daemon_test.go` around lines 690 - 783, Wrap the entire test
body of
TestBootExtensionsKeepsHealthyRegisteredExtensionsAfterPartialStartFailure in a
t.Run subtest using the "Should..." naming convention (e.g. t.Run("Should keep
healthy registered extensions after partial start failure", func(t *testing.T) {
... })); move t.Parallel() into that subtest and keep all setup/assertions (db
:= openDaemonTestGlobalDB, installDaemonTestExtension, runtime :=
&fakeExtensionRuntime{...}, d.newExtensionManager, state := &bootState{...},
cleanup := &bootCleanup{}, call to d.bootExtensions, and all checks) inside the
subtest closure so the top-level Test... function only invokes t.Run with the
described name.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The new daemon test body was added directly under the top-level `Test...` function without the repo-standard `t.Run("Should...")` wrapper.
  Root cause: the new case skipped the required subtest structure used throughout the Go test suite.
  Planned fix: wrap the test body in a `Should...` subtest and keep `t.Parallel()` inside that closure.

## Resolution

- Wrapped the daemon extension boot regression case in a `t.Run("Should...")` subtest and kept `t.Parallel()` inside that closure so the test follows the suite’s required structure.
