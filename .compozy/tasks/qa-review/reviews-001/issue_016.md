---
status: resolved
file: web/src/components/app-sidebar.tsx
line: 241
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59oaQ2,comment:PRRC_kwDOR5y4QM67VX7V
---

# Issue 016: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify derived-state effect usage in AgentItem.
rg -n -C3 'shouldAutoOpen|useEffect\(|onOpenChange=' web/src/components/app-sidebar.tsx
```

Repository: compozy/agh

Length of output: 641

---



</details>

**Remove `useEffect` for derived `open` state; use computed value with user override instead.**

The `shouldAutoOpen` prop-derived value is synced into local state via `useEffect`, violating the rule that effects are only for external system sync. Instead, compute `open` directly and maintain only a user override state:

<details>
<summary>Refactor</summary>

```diff
-import { useEffect, useState } from "react";
+import { useState } from "react";
@@
-  const shouldAutoOpen = count > 0 || showPendingSessionRow;
-  const [open, setOpen] = useState(shouldAutoOpen);
-
-  useEffect(() => {
-    if (shouldAutoOpen) {
-      setOpen(true);
-    }
-  }, [shouldAutoOpen]);
+  const shouldAutoOpen = count > 0 || showPendingSessionRow;
+  const [userOpenOverride, setUserOpenOverride] = useState<boolean | null>(null);
+  const open = userOpenOverride ?? shouldAutoOpen;
@@
-    <Collapsible open={open} onOpenChange={setOpen} className="group/agent">
+    <Collapsible open={open} onOpenChange={setUserOpenOverride} className="group/agent">
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/components/app-sidebar.tsx` around lines 231 - 241, The component
currently copies the derived boolean shouldAutoOpen into local state via
useEffect; remove that effect and instead keep only a user override state (e.g.,
replace useState(shouldAutoOpen) with const [openOverride, setOpenOverride] =
useState<boolean | null>(null) or similar), compute the actual open value as
const open = openOverride !== null ? openOverride : shouldAutoOpen, remove the
useEffect block that sets open from shouldAutoOpen, and pass the computed open
and the setter (setOpenOverride or a wrapper that toggles the override) into
<Collapsible open={open} onOpenChange={...}> so the component uses the derived
shouldAutoOpen unless the user has explicitly overridden it; reference symbols:
shouldAutoOpen, open, setOpen (replace with openOverride/setOpenOverride),
useState, useEffect, Collapsible.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `AgentItem` copies the derived `shouldAutoOpen` value into local state and re-synchronizes it with `useEffect`.
  - Root cause: the component is using an effect for derived UI state rather than computing the open state from props plus an explicit user override.
  - Fix plan: replace the mirrored state with a nullable user override, compute `open` during render, remove the effect, and keep the existing regression around late-arriving sessions green.
