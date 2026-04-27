---
status: resolved
file: web/src/systems/agent/components/agent-sessions-list.test.tsx
line: 6
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59tCVk,comment:PRRC_kwDOR5y4QM67bPPS
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid cross-system internal imports in tests as well.**

Line 5 and Line 6 pull from `@/systems/session/mocks` and `@/systems/session/types`. Please import through `@/systems/session` (and expose required members in that barrel if needed).  


As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/agent-sessions-list.test.tsx` around lines 5
- 6, Tests are importing internal symbols directly (primarySessionFixture and
SessionPayload) from "@/systems/session/mocks" and "@/systems/session/types";
update the test in agent-sessions-list.test.tsx to import these symbols from the
public barrel "@/systems/session" instead, and if those exports are not yet
re-exported from the barrel, add primarySessionFixture and SessionPayload to the
session module's public exports so the test can consume them via
"@/systems/session".
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `agent-sessions-list.test.tsx` imports `primarySessionFixture` and `SessionPayload` from session internals instead of the session public barrel.
  - `SessionPayload` is already exported by `@/systems/session`; fixture values are test-only and should not be exported from the production barrel because that leaks fixture data into the app bundle.
  - The fix imports `SessionPayload` from `@/systems/session` and `primarySessionFixture` from the dedicated public test barrel `@/systems/session/testing`.

## Resolution

- Updated `web/src/systems/agent/components/agent-sessions-list.test.tsx` to import `SessionPayload` from `@/systems/session` and `primarySessionFixture` from `@/systems/session/testing`.
- Added the public session testing barrel without changing the production session barrel.
- Verified with targeted Vitest and full `make verify`.
