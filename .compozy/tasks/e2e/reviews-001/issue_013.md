---
status: resolved
file: internal/daemon/tool_mcp_resources_test.go
line: 266
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEb4,comment:PRRC_kwDOR5y4QM640q0Q
---

# Issue 013: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use `errors.Is()` instead of comparing error strings.**

The coding guidelines state: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings." String comparison is fragile and breaks if error messages change.



<details>
<summary>🔧 Proposed fix using errors.Is</summary>

```diff
+var errTriggerFailure = errors.New("trigger failure")
+
 // In the test setup:
 func(context.Context, resources.ResourceKind, resources.ReconcileReason) error {
-	return assertErr("trigger failure")
+	return errTriggerFailure
 },

 // In the assertion:
 err := syncer.Sync(context.Background())
-if err == nil {
-	t.Fatal("syncer.Sync() error = nil, want trigger failure")
-}
-if got, want := err.Error(), "trigger failure"; got != want {
-	t.Fatalf("syncer.Sync() error = %q, want %q", got, want)
-}
+if !errors.Is(err, errTriggerFailure) {
+	t.Fatalf("syncer.Sync() error = %v, want %v", err, errTriggerFailure)
+}
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/tool_mcp_resources_test.go` around lines 260 - 266, The test
must not compare error strings; change the assertions to use errors.Is: keep the
nil check for err (if err == nil { t.Fatal(...) }) then replace the string
comparison with errors.Is(err, ErrTriggerFailure) (or the actual sentinel error
exported by the code that represents "trigger failure"); import "errors" at top
if needed and assert like if !errors.Is(err, ErrTriggerFailure) {
t.Fatalf("syncer.Sync() error = %v, want ErrTriggerFailure", err) } so the test
matches wrapped errors reliably (ensure ErrTriggerFailure is the correct
sentinel from the package under test or add/export one if missing).
```

</details>

<!-- fingerprinting:phantom:medusa:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Root cause: the test compares `err.Error()` to a literal even though the failure is intentionally represented by a sentinel-style test error.
- Fix plan: switch the test to a real sentinel error value and assert it with `errors.Is`.
- Resolution: introduced a real sentinel trigger error and switched the assertion to `errors.Is`.
- Verification: `go test ./internal/daemon` passed. Historical note: the earlier `driver/dist/index.js` blocker was stale; the shipped mock driver is `internal/testutil/acpmock/cmd/acpmock-driver`.
