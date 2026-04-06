---
status: resolved
file: internal/httpapi/handlers_test.go
line: 416
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoCK,comment:PRRC_kwDOR5y4QM61T6Hh
---

# Issue 013: _🧹 Nitpick_ | _🔵 Trivial_
## Review Comment

_🧹 Nitpick_ | _🔵 Trivial_

**Table-drive the workspace handler coverage.**

These cases all repeat the same router/workspace setup and then vary only request payloads and assertions. Folding them into a `t.Run` table will shorten the suite and make it easier to keep the HTTP and UDS variants in sync.

As per coding guidelines, "Use table-driven tests with subtests (`t.Run`) as default in Go tests".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/httpapi/handlers_test.go` around lines 191 - 416, Collapse the six
nearly-identical tests (TestCreateWorkspaceHandlerRegistersWorkspace,
TestListWorkspacesHandlerReturnsRegisteredRows,
TestGetWorkspaceHandlerReturnsDetail,
TestUpdateWorkspaceHandlerUpdatesWorkspace,
TestDeleteWorkspaceHandlerReturnsNoContent,
TestResolveWorkspaceHandlerReturnsWorkspace) into a single table-driven test
that shares the common setup (homePaths := newTestHomePaths, engine :=
newTestRouter(... newTestHandlersWithWorkspace(...)), and helper calls
performRequest/decodeJSONResponse) and runs each case as a subtest via t.Run;
for each table entry provide only the per-case workspace stub behavior, HTTP
method/path/body, and expected status/response assertions, and invoke the
existing helpers (newTestRouter, newTestHandlersWithWorkspace, performRequest,
decodeJSONResponse) inside the shared setup so the variants only supply
case-specific data and assertions.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `INVALID`
- Notes:
  - This is a broad structural refactor suggestion, not a defect report.
  - The six handler tests exercise distinct request/response shapes and already have focused assertions with readable failures.
  - Rewriting them into one table in this remediation batch would create unrelated churn without fixing behavior.
