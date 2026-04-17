---
status: resolved
file: packages/site/components/landing/install-section.tsx
line: 99
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57hDAl,comment:PRRC_kwDOR5y4QM64gE5r
---

# Issue 018: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**ARIA tab semantics are incomplete and should be finished (or removed).**

The component declares a tab pattern but does not fully implement tab behavior (tabpanel linkage and arrow-key navigation). That creates inconsistent accessibility semantics.


<details>
<summary>♿ Suggested fix (complete tab pattern)</summary>

```diff
-import { useState } from "react";
+import { useState, type KeyboardEvent } from "react";
@@
 export function InstallSection() {
   const [tab, setTab] = useState<TabId>("brew");
   const active = INSTALL_TABS.find(t => t.id === tab) ?? INSTALL_TABS[0];
+
+  const onTabKeyDown = (event: KeyboardEvent<HTMLButtonElement>, current: TabId) => {
+    if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") return;
+    event.preventDefault();
+    const index = INSTALL_TABS.findIndex(t => t.id === current);
+    const delta = event.key === "ArrowRight" ? 1 : -1;
+    const next = (index + delta + INSTALL_TABS.length) % INSTALL_TABS.length;
+    setTab(INSTALL_TABS[next]!.id);
+  };
@@
           {INSTALL_TABS.map(t => (
             <button
               key={t.id}
               type="button"
               role="tab"
+              id={`install-tab-${t.id}`}
+              aria-controls={`install-panel-${t.id}`}
               aria-selected={t.id === tab}
+              tabIndex={t.id === tab ? 0 : -1}
               onClick={() => setTab(t.id)}
+              onKeyDown={e => onTabKeyDown(e, t.id)}
               className={cn(
@@
-        <div className="mt-4">
+        <div
+          id={`install-panel-${active.id}`}
+          role="tabpanel"
+          aria-labelledby={`install-tab-${active.id}`}
+          className="mt-4"
+        >
           <CodeBlock code={active.command} caption={active.note} shell />
         </div>
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import { useState, type KeyboardEvent } from "react";

export function InstallSection() {
  const [tab, setTab] = useState<TabId>("brew");
  const active = INSTALL_TABS.find(t => t.id === tab) ?? INSTALL_TABS[0];

  const onTabKeyDown = (event: KeyboardEvent<HTMLButtonElement>, current: TabId) => {
    if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") return;
    event.preventDefault();
    const index = INSTALL_TABS.findIndex(t => t.id === current);
    const delta = event.key === "ArrowRight" ? 1 : -1;
    const next = (index + delta + INSTALL_TABS.length) % INSTALL_TABS.length;
    setTab(INSTALL_TABS[next]!.id);
  };

  return (
    <div
      role="tablist"
      aria-label="Install methods"
      className="flex flex-wrap gap-1 rounded-[8px] border border-(--color-divider) bg-(--color-canvas) p-1"
    >
      {INSTALL_TABS.map(t => (
        <button
          key={t.id}
          type="button"
          role="tab"
          id={`install-tab-${t.id}`}
          aria-controls={`install-panel-${t.id}`}
          aria-selected={t.id === tab}
          tabIndex={t.id === tab ? 0 : -1}
          onClick={() => setTab(t.id)}
          onKeyDown={e => onTabKeyDown(e, t.id)}
          className={cn(
            buttonVariants({
              variant: t.id === tab ? "secondary" : "ghost",
              size: "sm",
            }),
            "flex-1 font-mono text-[12px] tracking-[0.02em]",
            t.id === tab &&
              "bg-(--color-accent-tint) text-(--color-accent) hover:bg-(--color-accent-tint)"
          )}
        >
          {t.label}
        </button>
      ))}
    </div>

    <div
      id={`install-panel-${active.id}`}
      role="tabpanel"
      aria-labelledby={`install-tab-${active.id}`}
      className="mt-4"
    >
      <CodeBlock code={active.command} caption={active.note} shell />
    </div>
  );
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/site/components/landing/install-section.tsx` around lines 70 - 99,
The tab pattern is incomplete: finish ARIA semantics by giving each tab button
an id and aria-controls that points to the corresponding tabpanel, make the
CodeBlock wrapper a role="tabpanel" with id matching aria-controls and
aria-labelledby pointing back to the active tab id, and ensure tab buttons use
aria-selected (already used) plus tabIndex (0 for selected, -1 for others); add
keyboard handling on the tab buttons (e.g., a handleKeyDown referenced from
INSTALL_TABS mapping) to implement Left/Right/Home/End arrow navigation that
moves focus and calls setTab; update references to INSTALL_TABS, tab, setTab,
active, and CodeBlock to wire these attributes and the keydown handler so the
component implements a complete tab pattern.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - The install-method selector declares tab semantics (`role="tablist"` / `role="tab"`) but does not complete the rest of the tab pattern, so assistive tech gets partial semantics and no keyboard navigation.
  - Root cause: tabs are missing `id` / `aria-controls`, inactive tabs stay in the tab order, the content region is not marked as a `tabpanel`, and arrow/home/end key handling is absent.
  - Fix plan: implement the complete tab contract with panel linkage, roving tab index, and keyboard navigation, then add an interaction test for the keyboard path.
  - Resolution: completed the tab contract in `install-section.tsx` with linked tab/panel IDs, roving tab index, and arrow/home/end keyboard navigation, and added a landing interaction test.
  - Verification: `packages/site` `bun run test`, `bun run typecheck`, and `bun run build` passed.
