---
status: resolved
file: internal/extension/host_api_test.go
line: 1019
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM564LgJ,comment:PRRC_kwDOR5y4QM63o2QD
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Increase the notifier wait timeout to reduce CI flakiness.**

A `500ms` timeout can be too tight under loaded CI and cause nondeterministic failures despite correct behavior.

<details>
<summary>💡 Suggested tweak</summary>

```diff
-	case <-time.After(500 * time.Millisecond):
+	case <-time.After(2 * time.Second):
 		t.Fatal("timed out waiting for prompt completion")
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
	select {
	case notifiedSessionID := <-turnEnded:
		if got, want := notifiedSessionID, sess.ID; got != want {
			t.Fatalf("turn end notifier session id = %q, want %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for prompt completion")
	}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/extension/host_api_test.go` around lines 1012 - 1019, The test's
select that waits on the turnEnded notifier is using a 500ms timeout which is
too short for CI; update the time.After(500 * time.Millisecond) to a larger
duration (e.g. 2s or make it configurable) so the case reading from the
turnEnded channel (notifiedSessionID compared to sess.ID) has more leeway and
reduces flakiness in CI.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The notifier wait is condition-based, but the current `500ms` timeout is still tighter than the rest of the async integration tests and can become flaky under loaded CI.
  Root cause: the completion-notification timeout is overly aggressive for an asynchronous delivery path.
  Planned fix: increase the timeout to a safer value so legitimate notifier delivery is not misclassified as a failure.

## Resolution

- Increased the host API notifier wait timeout from `500ms` to `2s` in `internal/extension/host_api_test.go` to reduce CI flakiness on the async prompt-completion path.
