---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/bridgesdk/runtime_refac_test.go
line: 83
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AbUs9,comment:PRRC_kwDOR5y4QM6-_G35
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the first shutdown error specifically.**

Lines 92-94 only check that *some* error was returned, so this test still passes if `handleShutdown` starts failing for an unrelated reason. Use a sentinel error and assert it with `errors.Is`.

<details>
<summary>Suggested fix</summary>

```diff
 		shutdownCalls := 0
+		shutdownFailure := errors.New("provider shutdown failed")
 		runtime, err := NewRuntime(RuntimeConfig{
 			ExtensionInfo: subprocess.InitializeExtensionInfo{
 				Name:    "telegram-adapter",
 				Version: "1.0.0",
 			},
 			Deliver: func(_ context.Context, session *Session, request bridgepkg.DeliveryRequest) (bridgepkg.DeliveryAck, error) {
 				return session.AckDelivery(request, "remote-1", "")
 			},
 			Shutdown: func(context.Context, *Session, subprocess.ShutdownRequest) error {
 				shutdownCalls++
 				if shutdownCalls == 1 {
-					return errors.New("provider shutdown failed")
+					return shutdownFailure
 				}
 				return nil
 			},
 		})
@@
-		if _, err := runtime.handleShutdown(t.Context(), json.RawMessage(`{"reason":"test"}`)); err == nil {
-			t.Fatal("first handleShutdown() error = nil, want provider failure")
+		if _, err := runtime.handleShutdown(t.Context(), json.RawMessage(`{"reason":"test"}`)); !errors.Is(err, shutdownFailure) {
+			t.Fatalf("first handleShutdown() error = %v, want shutdownFailure", err)
 		}
```
</details>

 
As per coding guidelines, "MUST have specific error assertions (ErrorContains, ErrorAs)".


Also applies to: 92-94

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/bridgesdk/runtime_refac_test.go` around lines 79 - 83, The test's
mocked Shutdown handler increments shutdownCalls and returns an error on the
first call, but the assertion only checks that some error occurred; update the
mock to return a sentinel error variable (e.g., var errProviderShutdown =
errors.New("provider shutdown failed")) from the Shutdown closure (referencing
Shutdown and shutdownCalls) and change the assertion to use
errors.Is(returnedErr, errProviderShutdown) so the test specifically verifies
that the first shutdown failure is the expected sentinel error.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/bridgesdk/runtime_refac_test.go:92-94` only asserts that some error occurred on the first shutdown attempt.
  - Because the test already controls the injected failure, the assertion should verify the specific sentinel with `errors.Is`.
  - Fix plan: introduce a sentinel shutdown error in the test and assert it explicitly.
  - Resolved: the shutdown retry test now asserts the injected sentinel failure with `errors.Is`.
