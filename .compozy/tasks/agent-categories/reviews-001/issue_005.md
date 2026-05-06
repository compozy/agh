---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: packages/ui/src/components/reui/tree.tsx
line: 129
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AH9AW,comment:PRRC_kwDOR5y4QM6-k_Pw
---

# Issue 005: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
cat -n packages/ui/src/components/reui/tree.tsx | head -150 | tail -50
```

Repository: compozy/agh

Length of output: 2082

---

🏁 Script executed:

```shell
sed -n '100,140p' packages/ui/src/components/reui/tree.tsx
```

Repository: compozy/agh

Length of output: 1502

---

🏁 Script executed:

```shell
rg "mergeProps" packages/ui/src/components/reui/tree.tsx -B 2 -A 2
```

Repository: compozy/agh

Length of output: 443

---

🏁 Script executed:

```shell
rg "function mergeProps|const mergeProps|export.*mergeProps" -t ts -t tsx
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
rg 'type.*:.*"button"' packages/ui -t ts -t tsx
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
fd -e test.ts -e test.tsx -e spec.ts -e spec.tsx packages/ui | xargs rg -l "TreeItem" 2>/dev/null | head -5
```

Repository: compozy/agh

Length of output: 640

---

🏁 Script executed:

```shell
head -30 packages/ui/src/components/reui/tree.tsx
```

Repository: compozy/agh

Length of output: 1131

---

🏁 Script executed:

```shell
rg "defaultTagName.*button" packages/ui -B 3 -A 3
```

Repository: compozy/agh

Length of output: 570

---

🏁 Script executed:

```shell
cat packages/ui/src/components/stories/tree.stories.tsx
```

Repository: compozy/agh

Length of output: 2357

---

🏁 Script executed:

```shell
rg "type.*button" packages/ui/src/components --max-count=20
```

Repository: compozy/agh

Length of output: 3108

---

🏁 Script executed:

```shell
rg "TreeItem|agent-category-tree" web/src -A 5 -B 5 | head -80
```

Repository: compozy/agh

Length of output: 7089

---

🏁 Script executed:

```shell
rg -i "<form" web/src/systems/agent -A 10 | head -50
```

Repository: compozy/agh

Length of output: 37

---

🌐 Web query:

`Base UI React mergeProps function documentation`

💡 Result:

Base UI React mergeProps documentation (official): - The utility is documented at https://base-ui.com/react/utils/merge-props [1]. What mergeProps does - mergeProps merges multiple sets of React props into a single props object you can spread onto an element [1]. - It follows an Object.assign-like rule where later arguments overwrite earlier ones for overlapping fields (rightmost wins) [1]. How it handles specific prop types - Event handlers: Base UI merges/chains event handlers and executes them right-to-left (rightmost first) [1]. For React synthetic events, Base UI wraps handlers so you can prevent Base UI’s internal handler logic by calling event.preventBaseUIHandler; this does not call preventDefault or stopPropagation [1]. - className and style: the merge logic is special-cased (not pure Object.assign); the API documentation and source note that merging behavior differs from simple field overwriting for these props [1][2]. Argument forms - You can pass multiple arguments (props objects and/or functions that receive the merged props so far and return the next props object) [1]. - mergeProps accepts up to 5 arguments; if you need to merge more than 5 prop sets, Base UI provides mergePropsN instead [1]. Function-render callback note - When using the function form of a Base UI component’s render prop, Base UI does not automatically merge props; you should use mergeProps to combine Base UI’s props with your own [1]. Source code reference (implementation) - The underlying implementation lives in the Base UI repo at packages/react/src/merge-props/mergeProps.ts [2]. The file comments describe the Object.assign-like “rightmost wins” behavior and call out that it doesn’t apply to event handlers, className, and style props [2]. Related docs (where mergeProps commonly appears) - Base UI’s useRender page explains that event handlers are merged and className/style are joined/merged when building render props [3]. - Base UI’s composition handbook provides general guidance for composing Base UI components via render props and prop forwarding/spreading [4].

Citations:

- 1: https://base-ui.com/react/utils/merge-props
- 2: https://github.com/mui/base-ui/blob/d81ec002/packages/react/src/merge-props/mergeProps.ts
- 3: https://base-ui.com/react/utils/use-render
- 4: https://base-ui.com/react/handbook/composition

---

🏁 Script executed:

```shell
rg "defaultTagName" packages/ui -B 5 -A 5
```

Repository: compozy/agh

Length of output: 4671

---

🏁 Script executed:

```shell
cat -n packages/ui/src/components/button.tsx | head -50
```

Repository: compozy/agh

Length of output: 3450

---



</details>

**Set the default tree row button type explicitly.**

`TreeItem` renders a real `<button>` by default. HTML buttons without an explicit `type` attribute default to `type="submit"`, which causes unintended form submission if the row is placed inside a form. Add `type: "button" as const` to `defaultProps` to prevent this.

<details>
<summary>Suggested fix</summary>

```diff
   const defaultProps = {
     "data-slot": "tree-item",
+    type: "button" as const,
     style: mergedStyle,
     className: cn(
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  const defaultProps = {
    "data-slot": "tree-item",
    type: "button" as const,
    style: mergedStyle,
    className: cn(
      "z-10 ps-(--tree-padding) outline-hidden select-none not-last:pb-0.5 focus:z-20 data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
      className
    ),
    "data-focus": focused,
    "data-folder": isFolder,
    "data-selected": selected,
    "data-drag-target": dragTarget,
    "data-search-match": searchMatch,
    "aria-expanded": isFolder ? item.isExpanded() : undefined,
  };

  return (
    <TreeContext.Provider value={{ ...parentContext, currentItem: item }}>
      {useRender({
        defaultTagName: "button",
        render,
        props: mergeProps<"button">(defaultProps, otherProps),
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@packages/ui/src/components/reui/tree.tsx` around lines 109 - 129, The
defaultProps for the TreeItem row uses a real <button> (see defaultProps and the
useRender/mergeProps call) but doesn't set an explicit type, so it defaults to
type="submit" and can submit surrounding forms; add type: "button" (as const) to
the defaultProps object to ensure the rendered button is non-submitting.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `TreeItem` still renders a default `<button>` without an explicit `type`.
  - Root cause: the Base UI render path defaults to a real button element, and HTML buttons default to `type="submit"` inside forms.
  - Fix approach: add `type: "button"` to the default tree-item props and cover it with a regression test so tree rows stay non-submitting when embedded in forms.
