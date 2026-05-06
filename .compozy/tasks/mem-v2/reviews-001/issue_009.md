---
provider: coderabbit
pr: "108"
round: 1
round_created_at: 2026-05-06T04:07:28.010433Z
status: resolved
file: internal/cli/client_test.go
line: 1398
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5_2Iss,comment:PRRC_kwDOR5y4QM6-UFVy
---

# Issue 009: _⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_ | _⚡ Quick win_

**Tighten the new memory transport assertions.**

These stubs return happy-path responses but do not verify the request shape for `ShowMemory`, `CreateMemory`, `DeleteMemory`, or `TriggerMemoryDream`. A client regression that drops `scope`, `workspace_id`, or create fields would still keep this suite green. Please assert those inputs here, like the `SearchMemory` and `ReindexMemory` cases already do. As per coding guidelines, "MUST test meaningful business logic, not trivial operations".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@internal/cli/client_test.go` around lines 1376 - 1398, The stub responses for
ShowMemory, CreateMemory, DeleteMemory, and TriggerMemoryDream must validate
incoming request shape; update the matching cases in internal/cli/client_test.go
to assert the query params and JSON body fields (e.g., for ShowMemory ensure
req.URL.Query().Get("scope") and workspace_id when required; for CreateMemory
parse req.Body and assert required fields like frontmatter.name/type, scope,
workspace_id or content; for DeleteMemory assert scope query equals "workspace"
already shown and add any required workspace_id; for TriggerMemoryDream parse
body and assert scope and workspace_id/payload fields). Use the existing pattern
(read and decode req.Body, compare values and call t.Fatalf on mismatch)
consistent with the SearchMemory/ReindexMemory assertions to make the tests fail
on regressions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The memory transport table in `internal/cli/client_test.go` already validates request shape for search/reindex, but the show/create/delete/dream cases still return happy-path payloads without checking the incoming query/body.
  - That leaves the client transport green even if it drops `scope`, `workspace_id`, or request body fields for those calls.
  - Fix approach: add request-shape assertions in those stub branches so the client test fails on transport regressions.
