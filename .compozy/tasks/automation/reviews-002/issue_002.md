---
status: resolved
file: internal/api/core/automation.go
line: 92
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56TZaA,comment:PRRC_kwDOR5y4QM623-TH
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't let `next_run` enrichment turn a successful write into an error response.**

Both handlers persist the create/update first and only then call `automationNextRunByJobID()`. If `manager.Status()` fails there, the API returns an error even though the mutation already succeeded, which is a bad client contract for write operations. For these paths, treat `next_run` lookup as best-effort and fall back to `nil` instead of failing the whole request. 


Also applies to: 161-167

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/api/core/automation.go` around lines 86 - 92, The call to
h.automationNextRunByJobID (used after persisting a job in the create/update
handlers) must be treated as best-effort so its failure doesn't turn a
successful write into an error response; update the code around the call (the
block that assigns nextRunByID and currently returns on err) to instead log the
error (or use h.logger) and set nextRunByID to nil/empty so
JobPayloadFromJob/timePointerFromMap receives nil for next_run; apply the same
change for the second occurrence (the similar block around lines 161-167) to
ensure manager.Status()/automationNextRunByJobID errors are non-fatal and the
handler still returns the created/updated job with next_run=null.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `CreateAutomationJob` and `UpdateAutomationJob` persist the mutation first and only then call `automationNextRunByJobID()`.
  - If `manager.Status()` fails inside that helper, the handler currently turns a successful write into an error response, which violates the write contract and can mislead clients into retrying an already-applied mutation.
  - Fix approach: make next-run enrichment best-effort only for the create/update job handlers, log the lookup failure, and return the created/updated job with no `next_run` value instead of failing the request.
  - Resolution: made job write-path next-run enrichment best-effort, added API regression tests for status lookup failures, and verified with focused `go test` runs plus `make verify`.
