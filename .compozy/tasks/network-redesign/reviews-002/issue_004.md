---
status: resolved
file: internal/store/globaldb/global_db_network_channels_test.go
line: 223
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59ReBH,comment:PRRC_kwDOR5y4QM662-g5
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Handle `db.Close()` errors in cleanup.**

Discarding the close error with `_` breaks the repo’s Go error-handling rule. Please report it from the cleanup closure instead.



<details>
<summary>Suggested fix</summary>

```diff
-	t.Cleanup(func() { _ = db.Close() })
+	t.Cleanup(func() {
+		if err := db.Close(); err != nil {
+			t.Errorf("db.Close() error = %v", err)
+		}
+	})
```
</details>

As per coding guidelines, `**/*.go`: Never ignore errors with `_` — every error must be handled or have a written justification.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_network_channels_test.go` at line 223, The
cleanup currently discards the error from db.Close() with `_`, violating the
no-ignored-errors rule; update the t.Cleanup closure to call db.Close(), capture
its error, and report it via the testing.T instance (e.g., t.Errorf or t.Fatalf)
so failures are surfaced — locate the t.Cleanup(func() { _ = db.Close() }) line
and replace it with a closure that checks err := db.Close() and reports it using
t (preserving test context).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `TestMigrateGlobalSchemaRebuildsNetworkChannelsWithWorkspaceForeignKey` currently discards `db.Close()` with `_`, which violates the repository rule against ignored errors in Go code.
- Fix plan: Replace the cleanup closure in `internal/store/globaldb/global_db_network_channels_test.go` with one that checks `db.Close()` and reports failures through `t.Errorf`.
- Outcome: The migration test cleanup now reports `db.Close()` failures via `t.Errorf`. Verified with `go test ./internal/store/globaldb -count=1` and `make verify`.
