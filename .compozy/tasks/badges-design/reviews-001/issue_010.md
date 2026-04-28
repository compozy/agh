---
status: resolved
file: web/src/systems/automation/components/automation-detail-panel.tsx
line: 16
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-MtdH,comment:PRRC_kwDOR5y4QM68GE0h
---

# Issue 010: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use the network system public barrel import here.**

Line 16 imports `KindChip` from another system’s internal path, which couples automation to network internals.


<details>
<summary>Suggested fix</summary>

```diff
-import { KindChip } from "@/systems/network/components/kind-chip";
+import { KindChip } from "@/systems/network";
```

If `KindChip` is not exported yet, add a named export in the network system barrel.
</details>
As per coding guidelines, `web/src/**/*.{ts,tsx}`: Cross-system imports: Only through the public barrel (`@/systems/<domain>`). Never reach into another system's internals.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/automation/components/automation-detail-panel.tsx` at line
16, Replace the internal-path import of KindChip with the network system public
barrel import (import { KindChip } from "@/systems/network"); if KindChip is not
exported from the network barrel, add a named re-export in the network system
barrel (e.g., export { KindChip } from "./components/kind-chip") so
automation-detail-panel uses only the public "@/systems/network" surface.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `automation-detail-panel.tsx` imports `KindChip` from the network system's internal component path.
- Fix approach: import `KindChip` from `@/systems/network` after adding a minimal public barrel export in `web/src/systems/network/index.ts`.
