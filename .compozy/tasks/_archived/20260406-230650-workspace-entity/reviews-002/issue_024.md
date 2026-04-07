---
status: resolved
file: internal/store/store_helpers_test.go
line: 551
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCu,comment:PRRC_kwDOR5y4QM61T6IO
---

# Issue 024: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Handle `Close`/`Rollback` failures instead of discarding them.**

These cleanup paths currently swallow DB errors, so a broken migration can still look green. Surface them with `t.Errorf(...)`, allowing `sql.ErrTxDone` after a successful commit.


<details>
<summary>🔧 Suggested fix</summary>

```diff
  t.Cleanup(func() {
-		_ = db.Close()
+		if closeErr := db.Close(); closeErr != nil {
+			t.Errorf("db.Close() error = %v", closeErr)
+		}
  })
…
  defer func() {
-		_ = tx.Rollback()
+		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
+			t.Errorf("tx.Rollback() error = %v", rollbackErr)
+		}
  }()
```
</details>
As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification".


Also applies to: 588-590

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/store/store_helpers_test.go` around lines 549 - 551, The cleanup
currently discards errors from db.Close() (and similarly tx.Rollback() in the
other cleanup) which can hide migration/test failures; update the t.Cleanup
handlers that call db.Close() and tx.Rollback() to check the returned error and
call t.Errorf(...) when err != nil, but treat sql.ErrTxDone as a
non-fatal/ignored case (i.e., do not report sql.ErrTxDone after a successful
commit). Locate the cleanup closures using t.Cleanup and the symbols db.Close
and tx.Rollback to implement this error handling.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  The test cleanups currently discard `db.Close()` and `tx.Rollback()` errors,
  which can hide migration failures and violate the repository rule against
  ignoring errors. Plan: report cleanup failures with `t.Errorf(...)` and treat
  only `sql.ErrTxDone` as ignorable after a successful commit.
