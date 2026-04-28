---
status: resolved
file: internal/api/core/tasks_terminal_integration_test.go
line: 219
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-JRQZ,comment:PRRC_kwDOR5y4QM68BYXZ
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Assert empty `payload.Run.Result` on fail/cancel paths.**

When `wantResultJSON` is empty, this block skips any result assertion. A regression that leaks stale result data in non-complete responses would still pass.



<details>
<summary>Patch suggestion</summary>

```diff
-			if tc.wantResultJSON != "" {
-				assertRawJSONEqual(t, "payload.Run.Result", payload.Run.Result, tc.wantResultJSON)
-			}
+			if tc.wantResultJSON == "" {
+				if len(payload.Run.Result) != 0 {
+					t.Fatalf("payload.Run.Result = %s, want empty result", string(payload.Run.Result))
+				}
+			} else {
+				assertRawJSONEqual(t, "payload.Run.Result", payload.Run.Result, tc.wantResultJSON)
+			}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
			if tc.wantResultJSON == "" {
				if len(payload.Run.Result) != 0 {
					t.Fatalf("payload.Run.Result = %s, want empty result", string(payload.Run.Result))
				}
			} else {
				assertRawJSONEqual(t, "payload.Run.Result", payload.Run.Result, tc.wantResultJSON)
			}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/tasks_terminal_integration_test.go` around lines 217 - 219,
The test currently only asserts payload.Run.Result when tc.wantResultJSON != "",
which lets stale results slip through on fail/cancel paths; modify the block
around tc.wantResultJSON to add an explicit assertion for the empty-case so that
when tc.wantResultJSON == "" the test asserts payload.Run.Result is empty (e.g.,
assert.Empty or assert.Equal(t, "", payload.Run.Result) / assert.Nil as
appropriate) instead of skipping; keep using the same variables
(tc.wantResultJSON, payload.Run.Result) and the existing assert helpers
(assertRawJSONEqual) for the non-empty branch.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `TestTaskRunTerminalHandlersPreserveHistoricalChannelBindingsIntegration` asserts `payload.Run.Result` only when `wantResultJSON` is non-empty.
  - Fail/cancel cases use an empty expected result, so a stale non-empty result payload could leak through without test coverage.
  - Fix approach: add an explicit empty-result assertion when `wantResultJSON == ""` and keep the existing `assertRawJSONEqual` path for non-empty results.

## Resolution

- Added an explicit empty-result assertion for cases where `wantResultJSON == ""`.
- Preserved `assertRawJSONEqual` for non-empty result payloads.
- Verified with targeted `go test -race -tags integration ./internal/api/core -run TestTaskRunTerminalHandlersPreserveHistoricalChannelBindingsIntegration -count=1`.
- Verified the repository gate with `make verify` after code changes.
