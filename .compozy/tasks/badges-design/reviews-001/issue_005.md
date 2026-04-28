---
status: resolved
file: web/src/components/design-system-showcase.test.tsx
line: 205
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-Mtcw,comment:PRRC_kwDOR5y4QM68GE0G
---

# Issue 005: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Check if kind-chip is exported from the network system's public barrel
rg -n "export.*KindChip" web/src/systems/network/index.ts 2>/dev/null || echo "No barrel export found"

# Check how kind-chip is imported in the showcase
rg -n "kind-chip" web/src/components/design-system-showcase.tsx
```

Repository: compozy/agh

Length of output: 149

---

🏁 Script executed:

```shell
#!/bin/bash
# Check the actual structure of the network system
echo "=== Network system public barrel content ==="
cat web/src/systems/network/index.ts 2>/dev/null || echo "File not found"

echo -e "\n=== Check if kind-chip file exists ==="
find web/src/systems/network -name "*kind-chip*" -type f

echo -e "\n=== Check what's exported from kind-chip ==="
if [ -f "web/src/systems/network/components/kind-chip.tsx" ]; then
  rg -n "^export" web/src/systems/network/components/kind-chip.tsx
fi
```

Repository: compozy/agh

Length of output: 3435

---



</details>

**Fix cross-system import to use public barrel.**

The showcase imports `KindChip` from `@/systems/network/components/kind-chip`, which violates the guideline: "Cross-system imports: Only through the public barrel (`@/systems/<domain>`). Never reach into another system's internals."

`KindChip` is not currently exported from `@/systems/network`'s public barrel. Either:
1. Export `KindChip` from `web/src/systems/network/index.ts` and update the showcase to import from `@/systems/network`
2. Move `KindChip` to a shared location outside the network system if it's a general-purpose component

Update the test's allowed imports list to match the corrected import path once resolved.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/components/design-system-showcase.test.tsx` around lines 199 - 205,
The showcase is importing KindChip directly from
"@/systems/network/components/kind-chip" which breaks cross-system import rules;
either export KindChip from the network public barrel (add an export in
web/src/systems/network/index.ts) and change the showcase import to
"@/systems/network", or move KindChip to a shared location and import from that
new public barrel; after doing that, update the allowed Set in
design-system-showcase.test.tsx to replace
"@/systems/network/components/kind-chip" with the new public import path
"@/systems/network" (or the new shared barrel path) so the test reflects the
corrected import.
```

</details>

<!-- fingerprinting:phantom:poseidon:ocelot -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `design-system-showcase.tsx` imports `KindChip` from `@/systems/network/components/kind-chip`, so the test's allowed import list preserves a cross-system internal import.
- Fix approach: export `KindChip` from the network public barrel, change the showcase import to `@/systems/network`, and update this test's allowed import set. This requires the minimal adjacent edit to `web/src/systems/network/index.ts` because the barrel currently does not expose `KindChip`.
