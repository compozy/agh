---
status: resolved
file: internal/session/provider_lifecycle_test.go
line: 36
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59R11n,comment:PRRC_kwDOR5y4QM663fB7
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify ignored assignments in this test file (and nearby integration parity file).
rg -nP --type=go '^\s*_\s*=\s*' internal/session/provider_lifecycle_test.go internal/api/httpapi/transport_parity_integration_test.go
```

Repository: compozy/agh

Length of output: 439

---

🏁 Script executed:

```shell
#!/bin/bash
# Read the context around line 35 in the test file
sed -n '30,40p' internal/session/provider_lifecycle_test.go | cat -n

echo "---"

# Check if there are any inline comments explaining the ignored error
sed -n '32,38p' internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 574

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for other cleanup patterns in test files to understand the pattern
rg -n 'Cleanup.*Stop' -A 2 internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for all t.Cleanup patterns in the test file to see error handling approaches
rg -n 't.Cleanup' internal/session/provider_lifecycle_test.go -A 3
```

Repository: compozy/agh

Length of output: 146

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for cleanup patterns across all test files to understand idiomatic error handling
rg -n 't.Cleanup' --type=go -A 2 | head -40
```

Repository: compozy/agh

Length of output: 2425

---

🏁 Script executed:

```shell
#!/bin/bash
# Search for error handling patterns inside t.Cleanup blocks
rg -n 't.Cleanup' --type=go -A 5 | grep -E '(t.Cleanup|if err|t.Errorf|t.Fatalf)' | head -50
```

Repository: compozy/agh

Length of output: 4164

---

🏁 Script executed:

```shell
#!/bin/bash
# Look at the reconcile_test.go cleanup patterns more closely to see if errors are handled
sed -n '148,155p' internal/resources/reconcile_test.go | cat -n
```

Repository: compozy/agh

Length of output: 339

---

🏁 Script executed:

```shell
#!/bin/bash
# Check what other cleanup Stop() calls do in the codebase
rg -n '\.Stop\(' --type=go -B 2 -A 2 | grep -E '(t.Cleanup|Stop|Fatalf|Errorf|_ =)' | head -30
```

Repository: compozy/agh

Length of output: 2515

---

🏁 Script executed:

```shell
#!/bin/bash
# Verify if there's any documentation or justification for _ = patterns in cleanup
rg -n 'TODO|FIXME|NOTE|XXX' internal/session/provider_lifecycle_test.go
```

Repository: compozy/agh

Length of output: 37

---



</details>

**Handle cleanup `Stop` error instead of discarding it with `_`.**

Line 35 suppresses the error from `h.manager.Stop()`, violating the guideline: "Never ignore errors with `_` — every error must be handled or have a written justification." Other cleanup Stop operations in the codebase use `t.Fatalf()` (e.g., `automation/schedule_test.go`, `e2e/runtime_harness.go`).

<details>
<summary>Proposed fix</summary>

```diff
 	t.Cleanup(func() {
-		_ = h.manager.Stop(testutil.Context(t), session.ID)
+		if stopErr := h.manager.Stop(testutil.Context(t), session.ID); stopErr != nil {
+			t.Fatalf("cleanup Stop() error = %v", stopErr)
+		}
 	})
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	t.Cleanup(func() {
		if stopErr := h.manager.Stop(testutil.Context(t), session.ID); stopErr != nil {
			t.Fatalf("cleanup Stop() error = %v", stopErr)
		}
	})
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/provider_lifecycle_test.go` around lines 34 - 36, The
cleanup currently discards the error returned by
h.manager.Stop(testutil.Context(t), session.ID); change it to handle the error
and fail the test on error—call t.Cleanup with a closure that calls
h.manager.Stop(...) and if it returns a non-nil error invoke t.Fatalf (or
t.Fatalf-like helper) including the error and context (session.ID) so the test
fails loudly instead of ignoring the error. Ensure you reference the same
symbols: t.Cleanup, h.manager.Stop, testutil.Context, and session.ID when making
the change.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- The cleanup in `TestCreateWithProviderOverridePropagatesToSessionRuntime` currently ignores the `Stop` error with `_ =`, which violates the repo rule against discarding errors and can hide shutdown regressions.
- The cleanup should fail loudly when teardown fails so the test reports the underlying lifecycle issue.
- Resolved by handling the cleanup `Stop` error explicitly and failing the test with session context if teardown ever breaks.
