---
status: resolved
file: web/src/routes/_app/stories/-network.stories.tsx
line: 12
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59r7vW,comment:PRRC_kwDOR5y4QM67Z0NS
---

# Issue 027: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use the network public barrel instead of a deep system import (Line 12).**

Please route this through `@/systems/network` to preserve system boundaries from routes.

<details>
<summary>Suggested change</summary>

```diff
-import { networkStatusFixture } from "@/systems/network/mocks";
+import { networkStatusFixture } from "@/systems/network";
```
</details>


As per coding guidelines, "Cross-system imports: Only through the public barrel (`@/systems/<domain>`). Never reach into another system's internals".

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/stories/-network.stories.tsx` at line 12, Importing
networkStatusFixture directly from the internals violates cross-system import
rules; update the import in _network.stories.tsx to re-export
networkStatusFixture from the public barrel by changing the source to the
systems network public barrel (use "@/systems/network") so the story consumes
networkStatusFixture via the public API rather than the deep path.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: The route story imports `networkStatusFixture` from `@/systems/network/mocks`, which crosses into system internals from a route-level file. The fix is to stop importing the mock fixture from the route story and use the public `NetworkStatus` type from `@/systems/network` for a local story override object.
