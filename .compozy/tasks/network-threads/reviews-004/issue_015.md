---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/daemon/daemon_acpmock_faults_integration_test.go
line: 294
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_0Drx,comment:PRRC_kwDOR5y4QM6-RRYv
---

# Issue 015: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Don't treat an unstructured `error` event as acceptable here.**

`event.Failure == nil` currently bypasses the failure, so this test can still pass when the runtime emits a generic fatal `error` event without structured failure metadata.

 
<details>
<summary>Possible tightening</summary>

```diff
 	for _, event := range events {
 		if event.Type != "error" {
 			continue
 		}
-		if event.Failure == nil || event.Failure.Kind == store.FailureCanceled {
+		if event.Failure != nil && event.Failure.Kind == store.FailureCanceled {
 			continue
 		}
 		t.Fatalf("events = %#v, want no fatal error after explicit blocked prompt cancellation", events)
 	}
```
</details>
As per coding guidelines, `**/*_test.go`: `MUST have specific error assertions (ErrorContains, ErrorAs)`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func assertNoFatalBlockedCancelError(t testing.TB, events []aghcontract.AgentEventPayload) {
	t.Helper()

	for _, event := range events {
		if event.Type != "error" {
			continue
		}
		if event.Failure != nil && event.Failure.Kind == store.FailureCanceled {
			continue
		}
		t.Fatalf("events = %#v, want no fatal error after explicit blocked prompt cancellation", events)
	}
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/daemon/daemon_acpmock_faults_integration_test.go` around lines 282 -
294, The test helper assertNoFatalBlockedCancelError currently skips events with
event.Type == "error" when event.Failure == nil, letting an unstructured fatal
error slip by; update the logic in assertNoFatalBlockedCancelError to treat any
event with Type == "error" as a test failure (i.e., remove the event.Failure ==
nil bypass) and, where structured failure metadata is available (event.Failure
!= nil), assert its Kind equals store.FailureCanceled using a specific assertion
(ErrorContains/ErrorAs or equivalent) so the test fails on generic error events
and only accepts explicit canceled failures; reference the event.Type check and
the event.Failure.Kind usage in assertNoFatalBlockedCancelError when making the
change.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: `assertNoFatalBlockedCancelError` currently skips every `error` event with `event.Failure == nil`, which means an unstructured fatal runtime error is treated as acceptable after blocked-prompt cancellation. Tighten the helper so only explicitly structured `FailureCanceled` events are ignored.
