---
status: resolved
file: internal/session/provider_lifecycle_integration_test.go
line: 61
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581azr,comment:PRRC_kwDOR5y4QM66RFPT
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Don't swallow cleanup failures.**

Both cleanups discard `Stop` errors with `_`. If resume leaves a live session behind, these tests can still pass and contaminate later integration cases.


<details>
<summary>Suggested change</summary>

```diff
 t.Cleanup(func() {
-	_ = h.manager.Stop(testutil.Context(t), resumed.ID)
+	if err := h.manager.Stop(testutil.Context(t), resumed.ID); err != nil {
+		t.Errorf("Stop(%q) cleanup error = %v", resumed.ID, err)
+	}
 })
```
</details>
As per coding guidelines, `Never ignore errors with _ — every error must be handled or have a written justification`.


Also applies to: 122-124

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/provider_lifecycle_integration_test.go` around lines 59 -
61, The cleanup currently swallows errors from h.manager.Stop (called with
testutil.Context(t) and resumed.ID); capture the returned error in each
t.Cleanup and handle it (e.g., if err := h.manager.Stop(testutil.Context(t),
resumed.ID); err != nil { t.Fatalf("failed to stop session %s: %v", resumed.ID,
err) }) so failures during Stop do not go unnoticed — apply the same change to
both occurrences referenced in the review.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: both integration-test cleanups discard `Stop` errors, so leaked sessions during cleanup can be hidden and contaminate later cases.
- Fix plan: handle cleanup `Stop` errors explicitly in each `t.Cleanup` and report them through the test.
