---
status: resolved
file: web/src/systems/agent/components/stories/agent-stats-grid.stories.tsx
line: 63
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59sdFC,comment:PRRC_kwDOR5y4QM67ae5D
---

# Issue 012: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Potential runtime error if fixture is empty.**

Line 48 accesses `richSessions[0]` without a guard. If `sessionFixtures` has no sessions with `agent_name === "codex-agent"`, this will spread `undefined` and cause unexpected behavior or errors.



<details>
<summary>🛡️ Proposed fix with guard</summary>

```diff
+const baseSession = richSessions[0];
+
 const failedSessions: SessionPayload[] = [
   ...richSessions,
-  {
-    ...richSessions[0],
+  ...(baseSession ? [{
+    ...baseSession,
     id: "sess-failure",
     state: "stopped",
     stop_reason: "agent_crashed",
     failure: { kind: "agent_crashed", summary: "broker disconnect" },
     activity: {
       elapsed_seconds: 412,
       idle_seconds: 0,
       iteration_current: 2,
       iteration_max: 6,
       last_activity_at: "2026-04-17T18:55:00Z",
       last_activity_kind: "tool",
       last_progress_at: "2026-04-17T18:55:00Z",
     },
-  },
+  }] : []),
 ];
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/stories/agent-stats-grid.stories.tsx` around
lines 45 - 63, The test fixture builds failedSessions by spreading
richSessions[0] without ensuring richSessions is non-empty, which can spread
undefined and crash; update the failedSessions construction (the failedSessions
constant that references richSessions[0]) to guard against an empty richSessions
– e.g., check richSessions.length and if empty use a safe fallback object (a
minimal session shape or the first item from sessionFixtures) or skip adding the
failure case so the fixture remains valid when no "codex-agent" sessions exist.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `failedSessions` spreads `richSessions[0]` without proving the filtered fixture array is non-empty.
  - If the `codex-agent` fixture is removed, the story can fail during module import.
  - Fix by introducing an explicit fallback `SessionPayload` base and using it when `richSessions[0]` is absent. Also convert the story frame props to the same explicit `ReactNode`/interface pattern used by the other scoped stories.
  - Resolution: added an explicit fallback rich session, guarded the failure base, and converted the frame props to `ReactNode` plus `FrameProps`.
  - Verification: targeted Vitest passed; `make verify` passed.
