---
status: resolved
file: web/src/systems/bridges/components/bridge-provider-card.tsx
line: 4
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-MtdV,comment:PRRC_kwDOR5y4QM68GE0z
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Use the network public barrel, not internal component paths.**

This imports another system’s internal file directly, which breaks the cross-system boundary contract.


<details>
<summary>Suggested boundary-safe import</summary>

```diff
-import { KindChip } from "@/systems/network/components/kind-chip";
+import { KindChip } from "@/systems/network";
```
</details>
As per coding guidelines `web/src/**/*.{ts,tsx}: Cross-system imports: Only through the public barrel (`@/systems/`<domain>). Never reach into another system's internals`.

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import { KindChip } from "@/systems/network";
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/bridges/components/bridge-provider-card.tsx` at line 4, The
import of KindChip is reaching into another system's internals; update the
import in bridge-provider-card.tsx to use the network system's public barrel
instead of "@/systems/network/components/kind-chip"—i.e., import KindChip from
the "@/systems/network" barrel so the cross-system boundary contract is
respected and internal paths are not referenced.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `bridge-provider-card.tsx` imports `KindChip` through another system's internal file.
- Fix approach: import `KindChip` from `@/systems/network` and rely on the public barrel export.
