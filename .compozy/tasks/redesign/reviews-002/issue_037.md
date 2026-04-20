---
status: resolved
file: packages/ui/src/components/split-pane.tsx
line: 95
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57_lcT,comment:PRRC_kwDOR5y4QM65JoyV
---

# Issue 037: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Back navigation can become a dead-end in narrow mode.**

On Line 87-Line 95, the Back button is rendered even when `onDetailClose` is undefined (Line 18), which can leave users unable to return to the list view on narrow screens.


<details>
<summary>🔧 Suggested direction</summary>

```diff
 export interface SplitPaneProps extends Omit<React.ComponentProps<"div">, "onChange"> {
   list: React.ReactNode;
   detail?: React.ReactNode;
   listWidth?: number;
   detailEmpty?: React.ReactNode;
-  onDetailClose?: () => void;
+  onDetailClose?: () => void; // make required when detail is shown in narrow mode, or provide an internal fallback
   narrowBreakpoint?: number;
   backLabel?: string;
 }
```

At minimum, gate rendering/behavior so Back always has a working close path (either required callback or internal fallback state).
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/components/split-pane.tsx` around lines 82 - 95, The Back
button in the split pane is rendered when narrow && hasDetail even if
onDetailClose is undefined, which can leave users stranded; update the
render/behavior so the button is only shown or active when a working close path
exists by either (A) gating the button render on onDetailClose (e.g., render
split-pane-back only if onDetailClose is provided) or (B) implement an internal
fallback close handler in the SplitPane component (use component state to hide
the detail and call that internal handler when onDetailClose is absent), and
ensure the button's onClick uses onDetailClose ?? fallbackClose and has
appropriate aria-disabled/disabled state when no handler exists; reference
symbols: narrow, hasDetail, onDetailClose, backLabel,
data-slot="split-pane-back".
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - On narrow viewports `SplitPane` hides the list whenever `detail` is present, but if `onDetailClose` is absent there is no guaranteed path back to the list view.
  - Fix by adding a narrow-mode fallback that does not strand the user when a close callback is missing.
  - Regression coverage requires touching adjacent existing test file `packages/ui/src/components/split-pane.test.tsx`, which is outside the listed batch code files but is the colocated test surface for this component.
