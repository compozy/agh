---
status: resolved
file: web/src/systems/bridges/components/bridge-list-panel.tsx
line: 7
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-MtdO,comment:PRRC_kwDOR5y4QM68GE0o
---

# Issue 012: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Import `KindChip` via the network system public barrel.**

This reaches into another system’s internals. Keep cross-system boundaries by importing through `@/systems/network` (and export `KindChip` from that barrel if needed).

<details>
<summary>♻️ Suggested change</summary>

```diff
-import { KindChip } from "@/systems/network/components/kind-chip";
+import { KindChip } from "@/systems/network";
```
</details>



As per coding guidelines: `web/src/**/*.{ts,tsx}`: Cross-system imports: Only through the public barrel (`@/systems/<domain>`). Never reach into another system's internals.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/bridges/components/bridge-list-panel.tsx` around lines 6 - 7,
Replace the direct import of KindChip from the network system's internals with
the public barrel export: change the import to pull KindChip from
"@/systems/network" (e.g., import { KindChip } from "@/systems/network"); if
KindChip is not currently re-exported from that barrel, add it to the network
system's public exports so the component can be imported only via the public
barrel and avoid reaching into components/kind-chip internals.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `bridge-list-panel.tsx` imports `KindChip` from a network internal component path instead of the public `@/systems/network` barrel.
- Fix approach: import through `@/systems/network` and add the required minimal barrel export.
