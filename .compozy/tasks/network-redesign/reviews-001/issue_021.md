---
status: resolved
file: web/src/systems/network/components/network-workspace-shell.tsx
line: 567
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58qIer,comment:PRRC_kwDOR5y4QM66CAlW
---

# Issue 021: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Don’t hard-code the header status dot to success.**

The dot always renders with `tone="success"`, so `"degraded"`/`"stopped"` states still look healthy even though `getNetworkStatusTone()` exists for this mapping. That makes the workspace header report the wrong runtime health.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/components/network-workspace-shell.tsx` around lines
565 - 569, The StatusDot in network-workspace-shell.tsx is hard-coded to
tone="success"; change it to use the mapping function instead (call
getNetworkStatusTone with the current status value, e.g.
getNetworkStatusTone(status.status)) so the dot reflects
"running"/"online"/"degraded"/"stopped" correctly; ensure getNetworkStatusTone
is imported/available in the file and remove the literal "success" prop.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
- `web/src/systems/network/components/network-workspace-shell.tsx:564-567` hard-codes the header `StatusDot` to `tone="success"` while the same module already imports status-formatting helpers.
- The rendered header therefore reports degraded or stopped runtimes as healthy, which is a real UI-state mismatch rather than a stylistic preference.
- Fix approach: derive the tone from `status.status` via `getNetworkStatusTone(...)` and add regression coverage that exercises a non-healthy status through the rendered network page/component surface.

## Resolution

- Updated `network-workspace-shell.tsx` to derive the workspace header dot tone from `getNetworkStatusTone(status.status)` instead of a hard-coded success state.
- Added minimal out-of-scope regression coverage in `web/src/systems/network/components/network-workspace-shell.test.tsx` because the batch scope did not include an existing component test surface for this header behavior.
- Verified with `bun x vitest run src/systems/network/components/network-workspace-shell.test.tsx`, `make web-test`, and `make verify`.
