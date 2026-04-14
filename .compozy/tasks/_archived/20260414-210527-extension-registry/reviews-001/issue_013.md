---
status: resolved
file: internal/registry/multi_test.go
line: 334
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562WU3,comment:PRRC_kwDOR5y4QM63maeI
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Handle the download reader close error explicitly.**

Line 333 drops `result.Reader.Close()` with `_`, so this test would miss a cleanup failure on the delegated source. Fail the test if closing the reader returns an error.

<details>
<summary>💡 Proposed fix</summary>

```diff
-	_ = result.Reader.Close()
+	if err := result.Reader.Close(); err != nil {
+		t.Fatalf("Reader.Close() error = %v", err)
+	}
```
</details>


As per coding guidelines, Never ignore errors with `_` — every error must be handled or have a written justification.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/registry/multi_test.go` around lines 320 - 334, The test currently
discards the error from result.Reader.Close() which hides cleanup failures;
update the tail of the Test (where result is obtained) to call
result.Reader.Close(), check its returned error, and fail the test (e.g.,
t.Fatalf or t.Fatalf-like assertion) if Close() returns a non-nil error so
cleanup errors on the delegated source are surfaced; locate the call that
currently does `_ = result.Reader.Close()` and replace it with explicit error
handling using the existing t instance and result variable.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `TestMultiRegistryDownloadDelegatesToResolvedSource()` currently discards `result.Reader.Close()` with `_`, so the test would hide cleanup failures from the delegated source. I will handle the close error explicitly in the in-scope `internal/registry/multi_test.go`.
- Resolution: Replaced the ignored `result.Reader.Close()` call with explicit error handling in `internal/registry/multi_test.go`.
- Verification: `go test ./internal/registry/...`; `make verify`
