---
status: resolved
file: internal/api/contract/tasks.go
line: 108
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dMk,comment:PRRC_kwDOR5y4QM65IPD2
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Expose `TaskRunPayload.Metadata` consistently across transports.**

This field is now part of the shared contract, but `internal/extension/host_api_tasks.go:taskRunPayloadFromRun` still never copies `run.Metadata`. HTTP callers will receive `metadata`, while extension callers silently lose it for the same run.


<details>
<summary>🔧 Suggested follow-up</summary>

```diff
diff --git a/internal/extension/host_api_tasks.go b/internal/extension/host_api_tasks.go
@@
         EndedAt:        optionalTime(run.EndedAt),
         Error:          run.Error,
+        Metadata:       cloneRawMessage(run.Metadata),
         Result:         cloneRawMessage(run.Result),
     }
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/contract/tasks.go` at line 108, taskRunPayloadFromRun in
internal/extension/host_api_tasks.go fails to propagate run.Metadata into the
shared TaskRunPayload.Metadata (defined as json.RawMessage in
internal/api/contract/tasks.go), causing metadata to be present for HTTP callers
but lost for extensions; update taskRunPayloadFromRun to set
TaskRunPayload.Metadata = run.Metadata (or a safe copy/clone if necessary) when
constructing the payload so the Metadata field is consistently exposed across
transports.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/api/contract/tasks.go` exposes `TaskRunPayload.Metadata`, but `internal/extension/host_api_tasks.go:taskRunPayloadFromRun` currently drops `run.Metadata` while copying the rest of the run payload.
  - That creates a real transport parity bug: HTTP/UDS callers can receive run metadata, while extension callers cannot for the same run.
  - Fixing this requires a minimal out-of-scope change in `internal/extension/host_api_tasks.go` and a focused regression assertion in `internal/extension/host_api_test.go`, because the scoped contract file only declares the payload shape.
