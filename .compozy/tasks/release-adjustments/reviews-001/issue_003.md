---
status: resolved
file: internal/acp/handlers_test.go
line: 1147
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk02,comment:PRRC_kwDOR5y4QM67HMV3
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Make the new negative-heartbeat case a named subtest with a specific error assertion.**

This currently only checks non-nil error, so an unrelated validation failure would still pass.


<details>
<summary>✅ Suggested test tightening</summary>

```diff
-	if err := (PromptRequest{
-		TurnID:                    "turn-negative-heartbeat",
-		Message:                   "hello",
-		ActivityHeartbeatInterval: -time.Second,
-	}).Validate(); err == nil {
-		t.Fatal("PromptRequest.Validate(negative heartbeat) error = nil, want validation error")
-	}
+	t.Run("ShouldRejectNegativeHeartbeatInterval", func(t *testing.T) {
+		t.Parallel()
+
+		err := (PromptRequest{
+			TurnID:                    "turn-negative-heartbeat",
+			Message:                   "hello",
+			ActivityHeartbeatInterval: -time.Second,
+		}).Validate()
+		if err == nil {
+			t.Fatal("PromptRequest.Validate(negative heartbeat) error = nil, want validation error")
+		}
+		if !strings.Contains(err.Error(), "heartbeat") {
+			t.Fatalf("PromptRequest.Validate(negative heartbeat) error = %v, want heartbeat-specific validation", err)
+		}
+	})
```
</details>

As per coding guidelines, `**/*_test.go`: `MUST use t.Run("Should...") pattern for ALL test cases` and `MUST have specific error assertions (ErrorContains, ErrorAs)`.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/acp/handlers_test.go` around lines 1141 - 1147, Convert the
anonymous negative-heartbeat check into a named subtest (e.g.,
t.Run("ShouldRejectNegativeHeartbeat", func(t *testing.T) { ... })) and call
PromptRequest{TurnID:"turn-negative-heartbeat", Message:"hello",
ActivityHeartbeatInterval:-time.Second}.Validate() inside it; then assert the
returned error specifically using ErrorContains/ErrorAs (for example
require.ErrorContains(t, err, "heartbeat") or errors.Is/As as appropriate)
instead of only checking err != nil so the test ensures the validation failed
for the negative ActivityHeartbeatInterval via PromptRequest.Validate.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `TestAccessorsAndValidationHelpers` validates the negative heartbeat case inline and only checks for a non-nil error.
  - A different validation failure could satisfy the current assertion, so the fix is a named `Should...` subtest plus a heartbeat-specific error assertion.
