---
status: resolved
file: internal/acp/client_test.go
line: 710
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QcYq,comment:PRRC_kwDOR5y4QM620KiP
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Handle shutdown error explicitly and bound cleanup wait.**

Ignoring `managed.Shutdown` hides teardown failures, and the unconditional `<-proc.Done()` can hang cleanup if exit never arrives.

<details>
<summary>♻️ Proposed fix</summary>

```diff
 		t.Cleanup(func() {
 			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 			defer cancel()
-			_ = managed.Shutdown(cleanupCtx)
-			<-proc.Done()
+			if shutdownErr := managed.Shutdown(cleanupCtx); shutdownErr != nil {
+				t.Fatalf("managed.Shutdown() error = %v", shutdownErr)
+			}
+			select {
+			case <-proc.Done():
+			case <-cleanupCtx.Done():
+				t.Fatalf("process did not exit during cleanup: %v", cleanupCtx.Err())
+			}
 		})
```
</details>


As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutdownErr := managed.Shutdown(cleanupCtx); shutdownErr != nil {
				t.Fatalf("managed.Shutdown() error = %v", shutdownErr)
			}
			select {
			case <-proc.Done():
			case <-cleanupCtx.Done():
				t.Fatalf("process did not exit during cleanup: %v", cleanupCtx.Err())
			}
		})
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/client_test.go` around lines 660 - 665, Handle the shutdown
error from managed.Shutdown and avoid an unbounded wait on proc.Done: inside the
t.Cleanup closure (around cleanupCtx, cancel, managed.Shutdown and
<-proc.Done()), call managed.Shutdown and capture its error and surface it
(t.Fatalf/t.Errorf or t.Logf + fail) instead of discarding with `_`, and replace
the unconditional `<-proc.Done()` with a select that waits on proc.Done() and
also a timeout (use the same 5s cleanupCtx or a new time.After) so the cleanup
cannot hang forever; reference the cleanup closure that creates
cleanupCtx/cancel, managed.Shutdown, and proc.Done to locate and update the
code.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The cleanup block in `TestStopManagedProcessRespectsContext` currently ignores `managed.Shutdown(...)` errors and waits on `proc.Done()` without a bound, so teardown failures can be hidden and the cleanup can hang forever.
  - Root cause: the test assumes the managed process always shuts down cleanly and exits promptly during cleanup.
  - Fix plan: handle the shutdown error explicitly and replace the unconditional receive with a `select` bounded by the existing cleanup context.
  - Implemented: the cleanup path now fails the test on shutdown errors and uses a timeout-bounded `select` around `proc.Done()` to prevent indefinite hangs.
  - Verification: `go test ./cmd/agh-codegen ./internal/acp ./internal/api/core ./internal/api/httpapi ./internal/api/spec -count=1`; `make verify`.
