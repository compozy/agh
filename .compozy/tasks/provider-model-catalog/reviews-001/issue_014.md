---
provider: coderabbit
pr: "118"
round: 1
round_created_at: 2026-05-07T16:19:53.268066Z
status: resolved
file: internal/daemon/model_catalog_test.go
line: 166
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AX6sc,comment:PRRC_kwDOR5y4QM6-6bsC
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

**Release the fake refresh before waiting on `refreshErrCh`.**

`manuallyReleasedModelCatalogService.Refresh` does not return until `service.release` is closed, so `waitForCatalogTestError()` cannot complete first. This subtest will hang here instead of checking the shutdown path.
 

<details>
<summary>Suggested reorder</summary>

```diff
-		refreshErr := waitForCatalogTestError(t, refreshErrCh, "manual refresh shutdown cancellation")
-		if !errors.Is(refreshErr, context.Canceled) {
-			t.Fatalf("Refresh(shutdown) error = %v, want context.Canceled", refreshErr)
-		}
-
 		close(service.release)
 		waitForCatalogTestSignal(t, service.released, "manual refresh release")
+		refreshErr := waitForCatalogTestError(t, refreshErrCh, "manual refresh shutdown cancellation")
+		if !errors.Is(refreshErr, context.Canceled) {
+			t.Fatalf("Refresh(shutdown) error = %v, want context.Canceled", refreshErr)
+		}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/model_catalog_test.go` around lines 154 - 166, The test is
hanging because manuallyReleasedModelCatalogService.Refresh blocks until
service.release is closed; close service.release before waiting on refreshErrCh.
Move the close(service.release) call to precede the call to
waitForCatalogTestError(t, refreshErrCh, ...) so that Refresh can return and
refreshErrCh can be read; keep the shutdown context/abort checks (shutdownCtx,
cancelShutdown, runtime.Shutdown) as-is but reorder so close(service.release)
happens before waiting for refreshErrCh and before waitForCatalogTestSignal on
service.released.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The shutdown test currently waits on `refreshErrCh` before releasing the fake service, but `Refresh(...)` does not return until `service.release` closes.
  - That ordering can deadlock the test instead of exercising the intended cancellation path.
  - Fix: release the fake service first, then await the refresh error/result assertions.
