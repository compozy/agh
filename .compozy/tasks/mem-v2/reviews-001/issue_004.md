---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 1271
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Isi,comment:PRRC_kwDOR5y4QM6-UFVk
---

# Issue 004: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Assert the dream target, not just that any trigger ran.**

This only checks `Triggered` and `dream.calls`, so a handler that ignores `workspace_id` or silently defaults to global still passes. Include `scope` in the request and have the stub record/assert the workspace argument it received.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/api/httpapi/httpapi_integration_test.go` around lines 1254 - 1271,
The test currently only checks payload.Triggered and runtime.dream.calls, which
lets a handler ignore workspace_id; modify the POST to include the
scope/workspace in the JSON (add "scope" or "workspace_id" field alongside
workspace_id already present), extend memoryDreamTriggerResponse (or inspect the
decoded payload) to include the returned scope field if the endpoint echoes it,
and update the dream stub used by runtime.dream to record the workspace argument
it received (add a recordedWorkspace field or similar) so the test can assert
that runtime.dream.recordedWorkspace == runtime.workspace; keep the existing
checks for payload.Triggered and runtime.dream.calls but add an explicit
equality assertion for the recorded workspace to ensure the correct target was
triggered.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TestHTTPMemoryDreamTriggerIntegration` only proves that some trigger happened (`Triggered` + `dream.calls`) and never proves the workspace selector reached the dream trigger service.
  - The integration stub in the same file currently discards the workspace argument passed to `Trigger`.
  - Fix approach: record the workspace argument inside `integrationDreamTrigger`, send an explicit scoped request, and assert the recorded workspace matches the runtime workspace.
