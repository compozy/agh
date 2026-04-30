---
provider: coderabbit
pr: "85"
round: 1
round_created_at: 2026-04-30T14:00:14.99254Z
status: resolved
file: internal/cli/tool_integration_test.go
line: 50
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-ulKj,comment:PRRC_kwDOR5y4QM680KJK
---

# Issue 024: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Handle `server.Shutdown` failures in cleanup.**

Swallowing this error can hide leaked goroutines or socket teardown problems, and it violates the repo rule against discarding errors in tests.


<details>
<summary>Suggested fix</summary>

```diff
 	t.Cleanup(func() {
 		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
 		defer cancel()
-		_ = server.Shutdown(ctx)
+		if err := server.Shutdown(ctx); err != nil {
+			t.Errorf("server.Shutdown() error = %v", err)
+		}
 	})
```
</details>

As per coding guidelines, "Never ignore errors with `_` in production code or tests — every error must be handled or explicitly justified".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("server.Shutdown() error = %v", err)
		}
	})
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/cli/tool_integration_test.go` around lines 47 - 50, In the t.Cleanup
block that creates ctx,cancel via context.WithTimeout and calls
server.Shutdown(ctx), stop discarding the returned error from server.Shutdown;
capture it and handle it (e.g., if err != nil { t.Fatalf("server.Shutdown
failed: %v", err) } or t.Errorf(...) ) so test failures or leaked resources are
surfaced; ensure you still call cancel() via defer cancel() and reference the
server.Shutdown(ctx) call by name when adding the conditional error handling.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `TestCLIToolCommandsMatchUDSContractsIntegration` discards `server.Shutdown(ctx)` with `_`, which violates the repo-wide test error handling rule and can hide teardown failures. The fix is to check the shutdown error in `t.Cleanup` and report it through the test.
- Resolution: Changed cleanup to check and report `server.Shutdown(ctx)` errors; verified with focused integration tests and `make verify`.
