---
status: resolved
file: web/src/routes/_app/-agents.$name.test.tsx
line: 7
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59tCVc,comment:PRRC_kwDOR5y4QM67bPPJ
---

# Issue 001: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use public system barrels instead of internal `mocks` paths**

These imports cross system boundaries through internal paths. Please import test fixtures via each system’s public barrel (or re-export them there first).

<details>
<summary>Suggested change</summary>

```diff
-import { primaryAgentFixture } from "@/systems/agent/mocks";
-import { primarySessionFixture } from "@/systems/session/mocks";
+import { primaryAgentFixture } from "@/systems/agent";
+import { primarySessionFixture } from "@/systems/session";
```
</details>


As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/-agents`.$name.test.tsx around lines 6 - 7, Replace the
direct imports from internal mock modules with the public system barrels: stop
importing primaryAgentFixture from "@/systems/agent/mocks" and
primarySessionFixture from "@/systems/session/mocks" and instead import those
fixtures from their public barrels (e.g., "@/systems/agent" and
"@/systems/session" or from the barrel export you add). Update the import
statements that reference primaryAgentFixture and primarySessionFixture so they
come from the respective public barrel exports; if the fixtures aren’t yet
re-exported from the public barrel, add re-exports for primaryAgentFixture and
primarySessionFixture in the systems' index barrel files and then import them
from those barrels.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The route test imports `primaryAgentFixture` from `@/systems/agent/mocks` and `primarySessionFixture` from `@/systems/session/mocks`, which bypasses each system public barrel.
  - The first implementation re-exported fixtures from the main system barrels, but production build inspection showed fixture data leaking into `web/dist/assets/session-*.js`.
  - The final fix uses dedicated public test barrels, `@/systems/agent/testing` and `@/systems/session/testing`, so cross-system tests no longer reach into `mocks` internals while production barrels stay fixture-free.

## Resolution

- Updated `web/src/routes/_app/-agents.$name.test.tsx` to import fixtures from the public testing barrels instead of internal `mocks` paths.
- Added `web/src/systems/agent/testing.ts` and `web/src/systems/session/testing.ts` as explicit public test fixture surfaces.
- Verified the production bundle no longer contains fixture markers (`Storybook rollout`, `sess-storybook`, `tool_bash`) after `make verify`.
