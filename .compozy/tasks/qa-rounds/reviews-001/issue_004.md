---
status: resolved
file: internal/daemon/daemon_integration_test.go
line: 2483
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vI,comment:PRRC_kwDOR5y4QM67Z0NC
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Wrap this scenario in a `t.Run("Should...")` test case to match test policy.**

The scenario is good, but the new test case should follow the required `Should...` subtest pattern for consistency with repository standards.

<details>
<summary>Minimal structure change</summary>

```diff
 func TestBootRunsWorkspaceTaskRunHookWithRelativeScriptPath(t *testing.T) {
-    homePaths := integrationHomePaths(t)
-    ...
+    t.Run("ShouldRunWorkspaceTaskRunHookWithRelativeScriptPath", func(t *testing.T) {
+        homePaths := integrationHomePaths(t)
+        ...
+    })
 }
```
</details>


As per coding guidelines, `**/*_test.go`: "MUST use t.Run("Should...") pattern for ALL test cases".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func TestBootRunsWorkspaceTaskRunHookWithRelativeScriptPath(t *testing.T) {
    t.Run("ShouldRunWorkspaceTaskRunHookWithRelativeScriptPath", func(t *testing.T) {
        homePaths := integrationHomePaths(t)
        cfg := testConfig(t, homePaths)
        cfg.Memory.Enabled = false
        cfg.Skills.Enabled = false

        workspaceRoot := filepath.Join(t.TempDir(), "workspace")
        if err := os.MkdirAll(filepath.Join(workspaceRoot, aghconfig.DirName, "hooks"), 0o755); err != nil {
            t.Fatalf(
                "os.MkdirAll(%q) error = %v",
                filepath.Join(workspaceRoot, aghconfig.DirName, "hooks"),
                err,
            )
        }
        writeDaemonFile(
            t,
            filepath.Join(workspaceRoot, aghconfig.DirName, "hooks", "capture-task-run.sh"),
            "#!/bin/sh\ncat > \"$1\"\n",
        )
        if err := os.Chmod(
            filepath.Join(workspaceRoot, aghconfig.DirName, "hooks", "capture-task-run.sh"),
            0o755,
        ); err != nil {
            t.Fatalf("os.Chmod(capture-task-run.sh) error = %v", err)
        }
        writeDaemonFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, "config.toml"), `
[[hooks.declarations]]
name = "workspace-task-run"
event = "task.run.enqueued"
mode = "sync"
command = "/bin/sh"
args = [".agh/hooks/capture-task-run.sh", ".agh/task-run-enqueued.json"]
`)

        resolvedWorkspace := seedDaemonWorkspace(t, homePaths, workspaceRoot)

        d, err := New(
            WithHomePaths(homePaths),
            WithConfig(&cfg),
            WithLogger(discardLogger()),
        )
        if err != nil {
            t.Fatalf("New() error = %v", err)
        }
        d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
            return &fakeSessionManager{}, nil
        }
        d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
            return &fakeObserver{}, nil
        }
        d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
            return &fakeServer{name: "http"}, nil
        }
        d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
            return &fakeServer{name: "uds"}, nil
        }

        if err := d.boot(testutil.Context(t)); err != nil {
            t.Fatalf("boot() error = %v", err)
        }
        t.Cleanup(func() {
            if err := d.Shutdown(testutil.Context(t)); err != nil {
                t.Fatalf("Shutdown() error = %v", err)
            }
        })
        if d.hooks == nil {
            t.Fatal("boot() did not initialize daemon hooks")
        }

        payload := hookspkg.TaskRunEnqueuedPayload{
            PayloadBase: hookspkg.PayloadBase{
                Event:     hookspkg.HookTaskRunEnqueued,
                Timestamp: time.Date(2026, 4, 26, 19, 30, 0, 0, time.UTC),
            },
            TaskRunContext: hookspkg.TaskRunContext{
                TaskID:                "task-1",
                RunID:                 "run-1",
                WorkspaceID:           resolvedWorkspace.ID,
                CoordinationChannelID: "operations",
                NetworkChannel:        "operations",
                AgentName:             "qa",
                TaskStatus:            "ready",
                RunStatus:             "queued",
            },
            IdempotencyKey: "task.start.task-1",
        }

        if _, err := d.hooks.DispatchTaskRunEnqueued(testutil.Context(t), payload); err != nil {
            t.Fatalf("DispatchTaskRunEnqueued() error = %v", err)
        }

        outputPath := filepath.Join(workspaceRoot, aghconfig.DirName, "task-run-enqueued.json")
        body, err := os.ReadFile(outputPath)
        if err != nil {
            t.Fatalf("os.ReadFile(%q) error = %v", outputPath, err)
        }

        var captured hookspkg.TaskRunEnqueuedPayload
        if err := json.Unmarshal(body, &captured); err != nil {
            t.Fatalf("json.Unmarshal(task run hook payload) error = %v; body=%s", err, string(body))
        }
        if captured.Event != hookspkg.HookTaskRunEnqueued ||
            captured.WorkspaceID != resolvedWorkspace.ID ||
            captured.RunID != "run-1" {
            t.Fatalf("captured payload = %#v, want enqueued payload for the seeded workspace run", captured)
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

In `@internal/daemon/daemon_integration_test.go` around lines 2377 - 2483, The
test function TestBootRunsWorkspaceTaskRunHookWithRelativeScriptPath must be
wrapped in a t.Run subtest using the repository's "Should..." naming convention;
update the body of TestBootRunsWorkspaceTaskRunHookWithRelativeScriptPath so its
existing implementation is executed inside t.Run("Should run workspace task-run
hook with relative script path", func(t *testing.T) { ... }), keeping all
existing setup, payload creation, hook dispatch, and assertions intact
(references: TestBootRunsWorkspaceTaskRunHookWithRelativeScriptPath, d.boot,
d.hooks.DispatchTaskRunEnqueued, seedDaemonWorkspace,
hookspkg.TaskRunEnqueuedPayload).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestBootRunsWorkspaceTaskRunHookWithRelativeScriptPath` is a new direct test scenario without the required `Should ...` subtest wrapper. Fix by wrapping the existing body in `t.Run("Should run workspace task-run hook with relative script path", ...)` without changing the boot, hook dispatch, or assertions.
