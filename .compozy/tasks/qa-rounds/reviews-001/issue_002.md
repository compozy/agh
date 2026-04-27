---
status: resolved
file: web/src/routes/_app/agents.$name.sessions.$id.tsx
line: 136
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59sdE9,comment:PRRC_kwDOR5y4QM67ae48
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use canonical agent name for post-delete navigation.**

`onDeleteSuccess` currently routes with the URL param `name`, which can be stale if the path is manually edited or mismatched. Use the resolved session agent name for consistent redirect behavior.

<details>
<summary>🐛 Suggested fix</summary>

```diff
   const workspaceName = workspaces?.find(workspace => workspace.id === session.workspace_id)?.name;
+  const resolvedAgentName = session.agent_name ?? name;
@@
       <SessionPageContent
-        agentName={session.agent_name ?? name}
+        agentName={resolvedAgentName}
         sessionId={id}
         session={session}
         workspaceName={workspaceName}
         onDeleteSuccess={() => {
-          void navigate({ to: "/agents/$name", params: { name } });
+          void navigate({ to: "/agents/$name", params: { name: resolvedAgentName } });
         }}
       />
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/agents`.$name.sessions.$id.tsx around lines 125 - 136,
The onDeleteSuccess handler uses the possibly-stale route param name when
navigating after deletion; change it to use the resolved agent name from the
session (e.g., session.agent_name ?? name) so the redirect is canonical. Update
the onDeleteSuccess in SessionPageContent's props to call navigate({ to:
"/agents/$name", params: { name: session.agent_name ?? name } }) (or compute a
resolvedAgentName variable) to ensure consistent post-delete routing.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The route passes `session.agent_name ?? name` to the page content but still navigates after delete with the raw URL param `name`.
  - If the URL param is stale or manually edited, delete success redirects to a non-canonical agent route even though the resolved session payload contains the correct agent name.
  - Fix by computing a single resolved agent name from the session and using it for both rendering and post-delete navigation. Add a route test that exercises delete success with a mismatched URL param.
  - Resolution: reused `resolvedAgentName` for both content and delete-success navigation, with a regression test covering a mismatched route param.
  - Verification: targeted Vitest passed; `make verify` passed.
