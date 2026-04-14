---
status: resolved
file: internal/session/stop_reason.go
line: 81
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM562ao4,comment:PRRC_kwDOR5y4QM63mgSR
---

# Issue 029: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**Guard nil receivers before calling `prepareStopWithCause()`.**

Both public methods dereference `m` immediately, so a typed-nil `*Manager` will panic instead of returning a normal error.



<details>
<summary>🛡️ Proposed fix</summary>

```diff
 func (m *Manager) RequestStopWithCause(ctx context.Context, id string, cause StopCause, detail string) error {
+	if m == nil {
+		return errors.New("session: manager is required")
+	}
 	if ctx == nil {
 		return errors.New("session: request stop context is required")
 	}
 	if cause == CauseNone {
 		cause = CauseUserRequested
@@
 func (m *Manager) StopWithCause(ctx context.Context, id string, cause StopCause, detail string) error {
+	if m == nil {
+		return errors.New("session: manager is required")
+	}
 	if ctx == nil {
 		return errors.New("session: stop context is required")
 	}
 	if cause == CauseNone {
 		cause = CauseUserRequested
```
</details>


Also applies to: 84-116

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/stop_reason.go` around lines 54 - 81, RequestStopWithCause
(and the sibling method around lines 84-116) dereference the receiver m before
checking for a typed nil, causing a panic for a nil *Manager; add an initial
guard like "if m == nil { return errors.New(\"session: manager is required\") }"
at the top of RequestStopWithCause and the other public stop method. Keep the
rest of the flow (calls to m.prepareStopWithCause, m.finalizeStopped,
m.driver.Cancel and isProcessDone) unchanged so the nil-receive case returns a
proper error instead of panicking.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  Both public methods dereference the receiver by calling `m.prepareStopWithCause(...)` before checking whether `m` is a typed-nil `*Manager`. That will panic instead of returning a normal validation error.
  I will add explicit nil-receiver guards at the top of both public stop methods and add a regression test. That test is not covered by the current batch file list, so a minimal session test file update outside the listed code files will be required and is justified by the lack of any scoped session test file for this path.
  Resolution: Added nil-receiver guards to both public stop methods and extended `internal/session/stop_reason_test.go` with `Should ...` regression coverage for the nil-manager path.
