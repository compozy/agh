---
status: resolved
file: web/src/systems/daemon/hooks/use-daemon-health.ts
line: 3
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-Mtdk,comment:PRRC_kwDOR5y4QM68GE1G
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid importing a UI component type into a system hook.**

`useDaemonHealth` now depends on `@/components/connection-indicator`, which reverses the intended layer direction and couples daemon domain logic to a view module.

<details>
<summary>♻️ Proposed fix</summary>

```diff
-import type { ConnectionStatus } from "@/components/connection-indicator";
+type ConnectionStatus = "connected" | "reconnecting" | "disconnected";
```
</details>

As per coding guidelines: `web/src/systems/**/*.{ts,tsx}` must keep dependency flow unidirectional (`adapters → lib → hooks → components`).

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
type ConnectionStatus = "connected" | "reconnecting" | "disconnected";
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/daemon/hooks/use-daemon-health.ts` at line 3, The hook
useDaemonHealth currently imports the UI type ConnectionStatus from the
component, which breaks layer boundaries; instead remove that import and replace
it with a system-level type (either define a local ConnectionStatus enum/type
inside useDaemonHealth or move a shared status type to a lower-level module such
as a lib/adapters/types module and import that). Update all references inside
useDaemonHealth (and any callers in systems) to use the new system-level type so
the hook no longer depends on "@/components/connection-indicator".
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `useDaemonHealth` imports a UI component type from `@/components/connection-indicator`, making a daemon system hook depend on a view-layer component.
- Fix approach: define the connection status union at the hook/system level so the hook no longer imports from the component layer.
