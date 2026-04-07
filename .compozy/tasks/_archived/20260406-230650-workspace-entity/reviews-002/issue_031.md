---
status: resolved
file: web/src/routes/_app/-session.$id.test.tsx
line: 94
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM55IoC6,comment:PRRC_kwDOR5y4QM61T6Id
---

# Issue 031: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**New workspace mock isn’t asserting the new behavior**

You now mock workspace data, but the test never verifies that `session.workspace_id` is translated into `ChatHeader`’s `workspaceName`. This leaves the new route logic untested.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/-session`.$id.test.tsx around lines 81 - 94, Test doesn't
assert that session.workspace_id maps to ChatHeader.workspaceName; update the
test in -session.$id.test.tsx to render the route (using the existing render
call) and assert the mocked workspace name ("alpha") appears where ChatHeader
should show workspaceName (or assert ChatHeader received the prop) so the
useWorkspaces mock is validated; reference the useWorkspaces mock,
session.workspace_id value, the render invocation in this test, and the
ChatHeader component when adding the assertion.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  The route test now mocks workspace data but never verifies that
  `session.workspace_id` is translated into `ChatHeader.workspaceName`. That
  leaves the new lookup logic untested. Plan: make the `ChatHeader` mock expose
  the received `workspaceName` and assert that `"alpha"` is passed through.
