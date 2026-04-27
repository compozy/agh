---
status: resolved
file: web/src/systems/agent/components/agent-stats-grid.test.tsx
line: 6
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59tCVx,comment:PRRC_kwDOR5y4QM67bPPh
---

# Issue 007: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use the session system public barrel for cross-system imports.**

Line 4 and Line 5 import from another system’s internals (`@/systems/session/mocks`, `@/systems/session/types`). Please consume these through `@/systems/session` (and re-export what you need there if missing).  


As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/agent-stats-grid.test.tsx` around lines 4 -
6, The test is importing session internals directly (primarySessionFixture,
SessionPayload) instead of the public session barrel; change the imports in
agent-stats-grid.test.tsx to import these symbols from "@/systems/session" (e.g.
import { primarySessionFixture, type SessionPayload } from "@/systems/session")
and if those symbols are not currently re-exported by the session barrel, add
them to the session system's public index (re-export primarySessionFixture and
SessionPayload from the session module) so cross-system imports use only the
public barrel; keep the AgentStatsGrid import as-is.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `agent-stats-grid.test.tsx` imports `primarySessionFixture` and `SessionPayload` from session internals instead of the public session barrel.
  - The same production-bundle concern from issue 004 applies here, so the fixture value should not be exposed through the main session barrel.
  - The scoped change imports `SessionPayload` from `@/systems/session` and `primarySessionFixture` from the dedicated public test barrel `@/systems/session/testing`.

## Resolution

- Updated `web/src/systems/agent/components/agent-stats-grid.test.tsx` to import `SessionPayload` from `@/systems/session` and `primarySessionFixture` from `@/systems/session/testing`.
- Reused the public session testing barrel added for issue 004.
- Verified with targeted Vitest and full `make verify`.
