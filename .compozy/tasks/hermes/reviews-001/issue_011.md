---
status: resolved
file: internal/daemon/daemon.go
line: 979
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59lV1x,comment:PRRC_kwDOR5y4QM67Ri1I
---

# Issue 011: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Tear the daemon down if retention startup fails.**

By this point `boot(ctx)` has already completed and `dreamRuntime.Start(ctx)` may already be running. Returning directly here leaves the daemon partially started and can leak runtime state or the lockfile on retention init failures.



<details>
<summary>🧹 Suggested fix</summary>

```diff
    if err := d.startObserverRetention(ctx); err != nil {
-		return err
+		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
+		defer cancel()
+		shutdownErr := d.Shutdown(shutdownCtx)
+		return errors.Join(
+			fmt.Errorf("daemon: start observability retention: %w", err),
+			shutdownErr,
+		)
    }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	if err := d.startObserverRetention(ctx); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()
		shutdownErr := d.Shutdown(shutdownCtx)
		return errors.Join(
			fmt.Errorf("daemon: start observability retention: %w", err),
			shutdownErr,
		)
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/daemon.go` around lines 977 - 979, If
d.startObserverRetention(ctx) fails after boot(ctx) and after
dreamRuntime.Start(ctx) may be running, tear the daemon down instead of
returning: call the daemon's teardown routine (e.g., d.shutdown(ctx) or the
existing stop/Stop/StopAsync method that stops dreamRuntime and releases the
lockfile) before returning the error; ensure you stop the runtime, release any
lockfile/state, and propagate the original error. The change should be placed
where startObserverRetention is invoked so that on error you perform the
teardown (using the daemon's existing shutdown/stop method) and then return the
error.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `Daemon.Run` returns immediately when `startObserverRetention` fails after `boot` and optional `dreamRuntime.Start`. That leaves already-started daemon resources to their callers. On retention startup failure, call the normal shutdown path with a bounded context and return the startup error joined with any shutdown error.
