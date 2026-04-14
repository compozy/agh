---
status: resolved
file: web/src/routes/_app/network.tsx
line: 33
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4Z,comment:PRRC_kwDOR5y4QM63ZMIO
---

# Issue 017: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Avoid reaching into the workspace system's internals from this route.**

`WorkspacePageShell` is imported from `@/systems/workspace/components/...`, which bypasses the workspace public API and couples the network route to another system's internal layout. Please re-export it from `@/systems/workspace` or move the shared shell to a neutral module.

As per coding guidelines, `web/src/**/*.{ts,tsx}`: Only import from cross-system dependencies through the public barrel export (`@/systems/<domain>`), never reach into another system's internals.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/network.tsx` around lines 32 - 33, This route is
importing WorkspacePageShell from an internal components path which breaches the
public API; update the workspace system to re-export WorkspacePageShell from its
public barrel (so it can be imported as WorkspacePageShell from
"@/systems/workspace") or move the shared shell into a neutral module, then
change the route import to use the public export (alongside
useActiveWorkspace/useWorkspace) so the network route only depends on the
workspace public API and no longer reaches into internals.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: the network route also reaches into the workspace system internals for `WorkspacePageShell`, which violates the same public-barrel boundary as the bridges route.
- Fix approach: consume `WorkspacePageShell` from `@/systems/workspace` after adding the minimal barrel export noted in issue 015. This requires one small out-of-scope barrel change to satisfy the boundary.
