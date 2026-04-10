---
status: resolved
file: internal/api/spec/spec_test.go
line: 23
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56QU6B,comment:PRRC_kwDOR5y4QM620Apx
---

# Issue 008: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Guard `sessions.items` before dereference to avoid panic.**

At Line 22, `propertySchema(...).Items.Value` can panic if `sessions` stops being an array or `items` is omitted. Prefer an explicit assertion failure for clearer diagnostics.

<details>
<summary>Proposed change</summary>

```diff
-		sessionSchema := propertySchema(t, listSessionsSchema, "sessions").Items.Value
+		sessions := propertySchema(t, listSessionsSchema, "sessions")
+		if sessions.Items == nil || sessions.Items.Value == nil {
+			t.Fatal("expected sessions to be an array with items schema")
+		}
+		sessionSchema := sessions.Items.Value
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		sessions := propertySchema(t, listSessionsSchema, "sessions")
		if sessions.Items == nil || sessions.Items.Value == nil {
			t.Fatal("expected sessions to be an array with items schema")
		}
		sessionSchema := sessions.Items.Value
		assertRequired(t, sessionSchema, "id", "agent_name", "state", "created_at", "updated_at")
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/spec/spec_test.go` around lines 22 - 23, The test dereferences
propertySchema(t, listSessionsSchema, "sessions").Items.Value which can panic if
the "sessions" schema is not an array or omits items; update the test to first
retrieve the Items (e.g. sessionItems := propertySchema(t, listSessionsSchema,
"sessions").Items), assert it is not nil (fail with a clear message if nil),
optionally assert the "sessions" schema has type "array" if available, then use
sessionItems.Value when calling assertRequired(t, sessionSchema, ...); reference
propertySchema, listSessionsSchema, and assertRequired in the change.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The finding is accurate. The test dereferences `.Items.Value` directly, so a schema regression from array to non-array would panic instead of failing with a clear assertion.
  - Root cause: missing structural guard before dereferencing the OpenAPI schema's array item reference.
  - Fix approach: capture the `sessions` property schema first, assert that its items schema is present, and then continue with the required-field assertions.
  - Resolution: implemented in `internal/api/spec/spec_test.go` and verified with focused package tests plus `make verify`.
