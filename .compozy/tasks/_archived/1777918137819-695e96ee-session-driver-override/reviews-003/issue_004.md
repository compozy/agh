---
status: resolved
file: internal/store/globaldb/global_db_extra_test.go
line: 687
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RvvW,comment:PRRC_kwDOR5y4QM663WqB
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Do not discard cleanup errors with `_` in tests.**

Both cleanup closures ignore `db.Close()` errors. Please assert/log cleanup failure instead of discarding it.

<details>
<summary>🔧 Suggested fix</summary>

```diff
- t.Cleanup(func() { _ = db.Close() })
+ t.Cleanup(func() {
+ 	if err := db.Close(); err != nil {
+ 		t.Errorf("db.Close() error = %v", err)
+ 	}
+ })
```
</details>

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".



Also applies to: 783-783

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/globaldb/global_db_extra_test.go` at line 687, The test
cleanup currently discards errors from db.Close() in the t.Cleanup closures
(using `_ = db.Close()`); change these to capture and handle the error by
asserting or logging it (e.g., call require.NoError or t.Fatalf/t.Errorf inside
the cleanup) so failures in global_db_extra_test.go are reported; update both
occurrences that call t.Cleanup(func() { _ = db.Close() }) to check the returned
error from db.Close() and fail or log appropriately.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- Root cause confirmed: both cleanup closures call `_ = db.Close()`, which discards a real error path and conflicts with the repo rule against ignoring errors.
- These are test-only cleanups, so the right fix is to report close failures through the test, not to suppress them.
- Fix plan: replace both ignored close calls with `t.Cleanup(...)` functions that check `db.Close()` and report any failure via `t.Errorf(...)`.
- Implemented: both reviewed `db.Close()` cleanups now report close failures, and the adjacent `conn.Close()` cleanup in the same helper was brought into the same no-ignored-errors pattern for consistency with the repo rule.
- Verified with targeted `go test ./internal/store/globaldb` coverage for the touched tests and the full repository gate (`make verify`).
