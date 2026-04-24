---
status: resolved
file: internal/api/spec/spec.go
line: 1559
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6P,comment:PRRC_kwDOR5y4QM66Aomf
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**`deleteTask` response codes are out of sync with runtime mapping.**

The spec advertises `422`, but `StatusForTaskError` maps task validation errors to `400` and conflict cases to `409` (see `internal/api/core/errors.go:180-213`). This will mislead generated clients.

<details>
<summary>🛠️ Suggested response alignment</summary>

```diff
 		Responses: []ResponseSpec{
 			{Status: 204, Description: "No Content"},
 			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
-			{Status: 422, Description: "Invalid task delete", Body: contract.ErrorPayload{}},
+			{Status: 400, Description: "Invalid task delete", Body: contract.ErrorPayload{}},
+			{Status: 409, Description: "Task delete conflict", Body: contract.ErrorPayload{}},
 			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
 			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
 		},
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
		Responses: []ResponseSpec{
			{Status: 204, Description: "No Content"},
			{Status: 404, Description: "Task not found", Body: contract.ErrorPayload{}},
			{Status: 400, Description: "Invalid task delete", Body: contract.ErrorPayload{}},
			{Status: 409, Description: "Task delete conflict", Body: contract.ErrorPayload{}},
			{Status: 503, Description: "Task service is not configured", Body: contract.ErrorPayload{}},
			{Status: 500, Description: "Internal server error", Body: contract.ErrorPayload{}},
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/spec/spec.go` around lines 1554 - 1559, The OpenAPI spec for
deleteTask is out of sync with runtime error mapping from StatusForTaskError;
replace the single 422 response entry with responses that match the function's
mappings: add a 400 "Invalid task request" (Body: contract.ErrorPayload) for
validation errors and a 409 "Task conflict" (Body: contract.ErrorPayload) for
conflict cases, keeping the existing 404, 503 and 500 entries; update the
ResponseSpec list in the deleteTask definition so generated clients match
StatusForTaskError behavior.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The spec is out of sync on the validation status code: the delete path returns `task.ErrValidation`, which `StatusForTaskError` maps to `400`, not `422`. I will align the delete-task response table to advertise `400` for invalid delete requests. I am not planning to add `409` because the current delete implementation does not emit a conflict sentinel on this route.
