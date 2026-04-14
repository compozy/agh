---
status: resolved
file: internal/network/tasks_integration_test.go
line: 247
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564Lg1,comment:PRRC_kwDOR5y4QM63o2Q5
---

# Issue 016: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use `context.Background()` in cleanup to avoid cancelled context.**

The cleanup function uses `ctx` from `testutil.Context(t)`, but test contexts are typically cancelled when the test completes. This could cause `db.Close()` to fail if the context is already cancelled. The cleanup at line 280 correctly uses `context.Background()` for the same reason.

<details>
<summary>🔧 Suggested fix for consistent cleanup context</summary>

```diff
 	t.Cleanup(func() {
-		if err := db.Close(ctx); err != nil {
+		if err := db.Close(context.Background()); err != nil {
 			t.Fatalf("GlobalDB.Close() error = %v", err)
 		}
 	})
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/network/tasks_integration_test.go` around lines 243 - 247, The test
cleanup currently calls db.Close(ctx) using the test context obtained via
testutil.Context(t), which may be cancelled when the test finishes; change the
cleanup to call db.Close(context.Background()) so the close runs with a
non-cancelled context—update the t.Cleanup anonymous function to call
db.Close(context.Background()) instead of db.Close(ctx) (referencing the
existing t.Cleanup(func() { ... }) and db.Close(...) call).
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The cleanup in `openNetworkTaskIngressDB()` closes the database with the per-test context captured earlier in the test, which may already be canceled when cleanup runs.
  Root cause: cleanup reused a test-scoped context instead of a fresh background context for teardown.
  Planned fix: use `context.Background()` for the close call, matching the other cleanup path in the same file.

## Resolution

- Updated the cleanup path in `internal/network/tasks_integration_test.go` to close the database with `context.Background()` so teardown does not depend on a possibly canceled test context.
