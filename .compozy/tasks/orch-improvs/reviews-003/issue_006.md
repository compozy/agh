---
provider: coderabbit
pr: "106"
round: 3
round_created_at: 2026-05-06T06:28:14.497092Z
status: resolved
file: internal/session/query_test.go
line: 834
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_3hZu,comment:PRRC_kwDOR5y4QM6-V-Lm
---

# Issue 006: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Strengthen event equality checks in the helper.**

Comparing only `ID` and `Sequence` lets content/type regressions pass. This test should also assert the stable business fields that `openQueryRecorder` is expected to preserve.

 
<details>
<summary>🔍 Suggested change</summary>

```diff
 func compareQueriedSessionEvents(want []store.SessionEvent, got []store.SessionEvent) error {
 	if len(got) != len(want) {
 		return fmt.Errorf("count = %d, want %d", len(got), len(want))
 	}
 	for index := range want {
 		if got[index].ID != want[index].ID {
 			return fmt.Errorf("event[%d].id = %q, want %q", index, got[index].ID, want[index].ID)
 		}
 		if got[index].Sequence != want[index].Sequence {
 			return fmt.Errorf(
 				"event[%d].sequence = %d, want %d",
 				index,
 				got[index].Sequence,
 				want[index].Sequence,
 			)
 		}
+		if got[index].Type != want[index].Type {
+			return fmt.Errorf("event[%d].type = %q, want %q", index, got[index].Type, want[index].Type)
+		}
+		if got[index].TurnID != want[index].TurnID {
+			return fmt.Errorf("event[%d].turn_id = %q, want %q", index, got[index].TurnID, want[index].TurnID)
+		}
+		if got[index].AgentName != want[index].AgentName {
+			return fmt.Errorf("event[%d].agent_name = %q, want %q", index, got[index].AgentName, want[index].AgentName)
+		}
+		if got[index].Content != want[index].Content {
+			return fmt.Errorf("event[%d].content = %q, want %q", index, got[index].Content, want[index].Content)
+		}
 	}
 	return nil
 }
```
</details>
As per coding guidelines, "Ensure tests can fail when business logic changes".

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
func compareQueriedSessionEvents(want []store.SessionEvent, got []store.SessionEvent) error {
	if len(got) != len(want) {
		return fmt.Errorf("count = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index].ID != want[index].ID {
			return fmt.Errorf("event[%d].id = %q, want %q", index, got[index].ID, want[index].ID)
		}
		if got[index].Sequence != want[index].Sequence {
			return fmt.Errorf(
				"event[%d].sequence = %d, want %d",
				index,
				got[index].Sequence,
				want[index].Sequence,
			)
		}
		if got[index].Type != want[index].Type {
			return fmt.Errorf("event[%d].type = %q, want %q", index, got[index].Type, want[index].Type)
		}
		if got[index].TurnID != want[index].TurnID {
			return fmt.Errorf("event[%d].turn_id = %q, want %q", index, got[index].TurnID, want[index].TurnID)
		}
		if got[index].AgentName != want[index].AgentName {
			return fmt.Errorf("event[%d].agent_name = %q, want %q", index, got[index].AgentName, want[index].AgentName)
		}
		if got[index].Content != want[index].Content {
			return fmt.Errorf("event[%d].content = %q, want %q", index, got[index].Content, want[index].Content)
		}
	}
	return nil
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/session/query_test.go` around lines 816 - 834, The helper
compareQueriedSessionEvents currently only checks ID and Sequence which lets
business-field regressions slip; update compareQueriedSessionEvents to assert
the full business payload for each event (e.g., Type, Content/Payload,
Metadata/Attributes, timestamps or any stable business fields that
openQueryRecorder should preserve) instead of only ID/Sequence — either by doing
explicit field comparisons for those stable fields or by using a deep-equality
check (reflect.DeepEqual or cmp.Diff) between want[index] and got[index] after
normalizing any non-deterministic fields, and return a descriptive error showing
the mismatch when they differ.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause analysis: `compareQueriedSessionEvents` only compares `ID` and `Sequence`, so regressions in preserved event business data can slip through unnoticed.
- Why this is valid: `openQueryRecorder` is expected to reopen persisted session history faithfully, and the current helper would still pass if event type/content/turn/agent data changed while IDs remained stable.
- Fix approach: strengthen the helper in `internal/session/query_test.go` to compare the stable preserved fields, then keep the existing reopened-recorder tests as the regression harness.

## Resolution

- Strengthened `compareQueriedSessionEvents` in `internal/session/query_test.go` to assert the stable preserved event fields rather than only `ID` and `Sequence`.

## Verification

- Focused regression: `go test ./internal/session -run 'TestOpenQueryRecorder|TestReadMetaAndQueryHelpers' -count=1 -race`
- Fresh full gate: `make verify` exited `0` in this session.
