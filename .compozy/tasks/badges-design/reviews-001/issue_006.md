---
status: resolved
file: web/src/components/design-system-showcase.tsx
line: 139
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-Mtc1,comment:PRRC_kwDOR5y4QM68GE0M
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Import `KindChip` via the network public barrel, not an internal path.**

Line 139 pulls from another system’s internal component path, which breaks system boundaries.


<details>
<summary>Suggested fix</summary>

```diff
-import { KindChip } from "@/systems/network/components/kind-chip";
+import { KindChip } from "@/systems/network";
```
</details>
As per coding guidelines, `web/src/**/*.{ts,tsx}`: Cross-system imports: Only through the public barrel (`@/systems/<domain>`). Never reach into another system's internals.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/components/design-system-showcase.tsx` at line 139, The import of
KindChip in design-system-showcase.tsx uses an internal path and violates
cross-system rules; update the import to use the network system's public barrel
instead (replace the current "@/systems/network/components/kind-chip" import
with an import from "@/systems/network" that exposes KindChip) so the component
is consumed via the network public API rather than an internal file.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `design-system-showcase.tsx` directly reaches into `network/components/kind-chip`, violating the system boundary rule for cross-system imports.
- Fix approach: re-export `KindChip` from `@/systems/network` and import through that public barrel. The required barrel export is a minimal adjacent edit to `web/src/systems/network/index.ts`.
