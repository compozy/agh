---
status: resolved
file: internal/api/spec/spec.go
line: 2246
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59qlsP,comment:PRRC_kwDOR5y4QM67YHCp
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't document `/api/tasks/{id}/approve` as requiring a JSON body.**

`buildOperation` marks every non-nil `RequestBody` as required, but the integration test exercises this route with an empty POST and expects `201 Created`. The generated spec will reject a valid call path unless this request body is modeled as optional.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/spec/spec.go` around lines 2235 - 2246, The OpenAPI spec
currently lists RequestBody: contract.TaskExecutionRequest{} for the approveTask
operation which causes buildOperation to treat a body as required; change the
operation's RequestBody to be nil/absent or a nil pointer (e.g., set RequestBody
to nil or (*contract.TaskExecutionRequest)(nil)) for the Method "POST" Path
"/api/tasks/{id}/approve" / OperationID "approveTask" so buildOperation will not
mark the body required and the empty-POST test can succeed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes: `approveTask` accepts an empty POST via `decodeOptionalJSON`, but `OperationSpec.RequestBody` is non-nil and `buildOperation` marks every body required. The OpenAPI document therefore rejects a valid transport path. Fix by allowing optional request bodies in the spec builder and marking approve's `TaskExecutionRequest` body optional rather than required.
