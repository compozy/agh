---
status: resolved
file: internal/daemon/task_runtime_test.go
line: 2019
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-uUK,comment:PRRC_kwDOR5y4QM65IlPN
---

# Issue 011: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Add cleanup for the reentry bridge created by the shared test helper.**

`newDetachedHarnessTaskRuntimeForTest` constructs a live `harnessReentryBridge` (Line 1987) but never shuts it down. Because this helper is reused heavily, leaked bridge goroutines can cause cross-test interference/flakes.



<details>
<summary>Proposed fix</summary>

```diff
 	reentry, err := newHarnessReentryBridge(
 		testutil.Context(t),
 		harnessResolver,
 		nil,
 		db,
 		sessions,
 		discardLogger(),
 	)
 	if err != nil {
 		t.Fatalf("newHarnessReentryBridge() error = %v", err)
 	}
+	t.Cleanup(reentry.shutdown)
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	reentry, err := newHarnessReentryBridge(
		testutil.Context(t),
		harnessResolver,
		nil,
		db,
		sessions,
		discardLogger(),
	)
	if err != nil {
		t.Fatalf("newHarnessReentryBridge() error = %v", err)
	}
	t.Cleanup(reentry.shutdown)
	manager, err := taskpkg.NewManager(
		taskpkg.WithStore(db),
		taskpkg.WithSessionExecutor(sessionBridge),
		taskpkg.WithEventObserver(reentry),
		taskpkg.WithNetworkChannelValidator(network.ValidateChannel),
		taskpkg.WithCancelGracePeriod(defaultTaskCancelGrace),
	)
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}
	detached, err := newHarnessDetachedWorkBridge(manager, db, sessions)
	if err != nil {
		t.Fatalf("newHarnessDetachedWorkBridge() error = %v", err)
	}

	return &taskRuntime{
		manager:  manager,
		store:    db,
		detached: detached,
		reentry:  reentry,
	}, resolver, homePaths
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/task_runtime_test.go` around lines 1987 - 2019,
newDetachedHarnessTaskRuntimeForTest creates a live harnessReentryBridge via
newHarnessReentryBridge but never stops it; register a test cleanup to stop the
bridge by calling its shutdown method (e.g. reentry.Close() or
reentry.Shutdown() as provided) — add t.Cleanup(func(){ _ = reentry.Close() })
immediately after the reentry is created (or use the correct method name on the
returned harnessReentryBridge type) so the reentry goroutines are stopped and
won't leak across tests; ensure the taskRuntime.reentry field remains set.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `newDetachedHarnessTaskRuntimeForTest` creates a live `harnessReentryBridge` backed by a worker goroutine.
  - The helper uses `testutil.Context(t)`, which cancels the bridge context during test cleanup, but it never waits for the bridge worker to exit.
  - Because this helper is reused across many detached-runtime tests, the missing shutdown join can leak background work past individual test cleanup and create cross-test interference.
  - Fix approach: register `t.Cleanup(reentry.shutdown)` immediately after bridge creation so cleanup both cancels and waits for the worker goroutine.
  - Resolved by adding the cleanup hook in `internal/daemon/task_runtime_test.go`, then verifying with targeted package tests and a fresh `make verify`.
