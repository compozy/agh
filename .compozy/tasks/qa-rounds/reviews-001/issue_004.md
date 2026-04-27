---
status: resolved
file: web/src/routes/_app/agents.$name.tsx
line: 85
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59sdE-,comment:PRRC_kwDOR5y4QM67ae4-
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid showing zeroed stats while session data is unresolved.**

`AgentStatsGrid` always receives `page.sessions`, so during `sessionsLoading` or `sessionsError` it renders `0`/`—` metrics that look authoritative even though the data is unavailable. Gate the grid behind the same state handling as the table, or give the grid explicit loading/error variants.


As per coding guidelines, "Truthful UI > plausible UI — don't render controls or metrics the runtime doesn't actually support. When Paper artboards conflict with daemon truth, daemon wins".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/agents`.$name.tsx around lines 79 - 85, AgentStatsGrid is
currently rendered with page.sessions even when session data is loading or
errored, causing misleading zero/placeholder metrics; update the render logic so
AgentStatsGrid is only shown when sessions are successfully loaded (i.e.,
!page.sessionsLoading && !page.sessionsError && page.sessions) or modify
AgentStatsGrid to accept and respect explicit loading/error props (e.g.,
sessionsLoading/sessionsError) and render loading/error variants instead—mirror
the same gating used by AgentSessionsList (agentName, sessions, isLoading,
isError) so the grid never displays authoritative metrics while data is
unresolved.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `AgentStatsGrid` receives `page.sessions` even while the sessions query is loading or errored, so it can show `0` and `—` as if they were authoritative runtime metrics.
  - This violates the truthful UI rule because unresolved data is rendered as a real empty data set.
  - Fix by gating the stats grid behind the same sessions success state used by the sessions list. Add a route test for the loading branch so the grid is absent while session data is unresolved.
  - Resolution: render `AgentStatsGrid` only after session data resolves successfully, while the sessions list retains loading/error UI.
  - Verification: targeted Vitest passed; `make verify` passed.
