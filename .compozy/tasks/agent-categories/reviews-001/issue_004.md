---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: packages/ui/src/components/reui/tree.tsx
line: 66
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AH9AQ,comment:PRRC_kwDOR5y4QM6-k_Po
---

# Issue 004: _⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's look at the actual tree.tsx file to confirm the code at the specified lines
find . -name "tree.tsx" -type f | head -5
```

Repository: compozy/agh

Length of output: 98

---

🏁 Script executed:

```shell
# Once we find the file, let's examine it
cat -n packages/ui/src/components/reui/tree.tsx | sed -n '50,110p'
```

Repository: compozy/agh

Length of output: 2442

---

🏁 Script executed:

```shell
# Search for mergeProps function definition
rg "mergeProps" --type ts --type tsx -A 5 | head -50
```

Repository: compozy/agh

Length of output: 83

---

🏁 Script executed:

```shell
# Check if mergeProps is imported or defined in tree.tsx
rg "import.*mergeProps|function mergeProps" packages/ui/src/components/reui/tree.tsx
```

Repository: compozy/agh

Length of output: 112

---

🏁 Script executed:

```shell
# Look at the file structure around the Tree and TreeItem components
cat -n packages/ui/src/components/reui/tree.tsx | head -100
```

Repository: compozy/agh

Length of output: 3768

---

🏁 Script executed:

```shell
# Search for mergeProps usage in the codebase to understand expected behavior
rg "mergeProps" packages/ui/src/components/reui/tree.tsx -B 2 -A 2
```

Repository: compozy/agh

Length of output: 443

---

🏁 Script executed:

```shell
# Look for tests related to tree component and prop handling
find packages -name "*.test.*" -o -name "*.spec.*" | grep -i tree
```

Repository: compozy/agh

Length of output: 101

---

🏁 Script executed:

```shell
# Check if there are any tests in a nearby location
find packages/ui/src/components/reui -name "*.test.*" -o -name "*.spec.*"
```

Repository: compozy/agh

Length of output: 101

---

🏁 Script executed:

```shell
# Look at the imports and see if we can find documentation or source of mergeProps
rg "@base-ui/react" packages/ -l | head -10
```

Repository: compozy/agh

Length of output: 439

---

🏁 Script executed:

```shell
# Check package.json to see what version of base-ui is being used
find . -name "package.json" -path "*/packages/*" -exec grep -l "@base-ui/react" {} \;
```

Repository: compozy/agh

Length of output: 82

---

🏁 Script executed:

```shell
# Look at actual usage of mergeProps in other files to understand its API
rg "mergeProps\(" packages/ui --type ts -B 1 -A 3 | head -60
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Let's examine the test file to understand expected behavior
cat -n packages/ui/src/components/reui/tree.test.tsx | head -100
```

Repository: compozy/agh

Length of output: 2656

---

🏁 Script executed:

```shell
# Let's also look at the rest of the Tree component to see the full context
cat -n packages/ui/src/components/reui/tree.tsx | sed -n '100,150p'
```

Repository: compozy/agh

Length of output: 2090

---

🏁 Script executed:

```shell
# Check package.json for base-ui version and any related info
cat packages/ui/package.json | grep -A 3 -B 3 "base-ui"
```

Repository: compozy/agh

Length of output: 301

---

🏁 Script executed:

```shell
# Let's search for how mergeProps is supposed to work by checking base-ui imports or comments
rg "mergeProps" packages/ui -B 3 -A 5 | head -100
```

Repository: compozy/agh

Length of output: 6706

---

🏁 Script executed:

```shell
# Let's check the base-ui documentation or find examples showing mergeProps parameter order
rg "mergeProps" packages/ui/src/components --type ts -B 2 -A 2 | grep -A 5 "mergeProps<" | head -40
```

Repository: compozy/agh

Length of output: 2046

---

🏁 Script executed:

```shell
# Look at the full TreeItem implementation to see the flow
cat -n packages/ui/src/components/reui/tree.tsx | sed -n '89,132p'
```

Repository: compozy/agh

Length of output: 2080

---

🏁 Script executed:

```shell
# Let's understand the mergeProps behavior better by checking if there's any documentation or comments
rg "mergeProps" --type ts -B 5 -A 5 packages/ui/src/components/item.tsx | head -30
```

Repository: compozy/agh

Length of output: 622

---

🏁 Script executed:

```shell
# Check if there are any prop override tests or examples
rg "onClick|aria-" packages/ui/src/components/reui/tree.test.tsx
```

Repository: compozy/agh

Length of output: 340

---

🏁 Script executed:

```shell
# Let's verify the actual problem by checking what itemProps contains
rg "getProps\(\)" --type ts -B 2 -A 2
```

Repository: compozy/agh

Length of output: 419

---

🏁 Script executed:

```shell
# Let's search for web information about `@base-ui/react` mergeProps to confirm its parameter order
# First check if there's a README or docs in the repo mentioning this
find . -name "README*" -o -name "CONTRIBUTING*" | head -5
```

Repository: compozy/agh

Length of output: 242

---

🏁 Script executed:

```shell
# Let's look for any comments or documentation about the prop merging strategy in the tree implementation
cat -n packages/ui/src/components/reui/tree.tsx | sed -n '1,100p' | grep -A 2 -B 2 "merge\|override\|props"
```

Repository: compozy/agh

Length of output: 1339

---

🏁 Script executed:

```shell
# Check the actual Tree component return to see if the props are being used correctly
cat -n packages/ui/src/components/reui/tree.tsx | sed -n '57,82p'
```

Repository: compozy/agh

Length of output: 852

---

🏁 Script executed:

```shell
# Let's check if there's any web documentation about base-ui mergeProps parameter order
# by looking at how it's documented in typical usage patterns
rg "mergeProps" packages/ui/src/components --type ts -A 3 | head -50
```

Repository: compozy/agh

Length of output: 3295

---

🏁 Script executed:

```shell
# Let's look at the complete context of how defaultProps and otherProps are used in TreeItem
cat -n packages/ui/src/components/reui/tree.tsx | sed -n '109,132p'
```

Repository: compozy/agh

Length of output: 986

---

🏁 Script executed:

```shell
# Let me search for web documentation on base-ui mergeProps to confirm the parameter order
# by checking if there are any inline comments or TypeScript definitions hinting at behavior
rg "mergeProps" -B 1 -A 1 packages/ui/package.json
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Let's check the actual Tree component's return JSX to confirm how otherProps is used
cat -n packages/ui/src/components/reui/tree.tsx | sed -n '72,82p'
```

Repository: compozy/agh

Length of output: 394

---

🌐 Web query:

`base-ui react mergeProps parameter order precedence`

💡 Result:

In Base UI’s React `mergeProps`, argument order is left-to-right: for most props, the value from the rightmost argument wins (i.e., later arguments override earlier ones) [1][2]. Key precedence rules: 1) Default / “most keys” precedence: rightmost wins [1]. 2) `ref`: not merged; only the rightmost `ref` is kept [1]. 3) `className`: concatenated (right-to-left), so the rightmost `className` appears first in the joined result [1]. 4) `style`: merged; keys from the rightmost `style` overwrite earlier ones [1]. 5) Event handlers: merged into a chain executed right-to-left (rightmost first) [1]. About parameter order with function-arguments: Each argument can be a props object or a function. Those functions are called left-to-right, each receiving the merged props up to that point (left to right), and returning props to merge next [1].

Citations:

- 1: https://base-ui.com/react/utils/merge-props
- 2: https://base-ui.com/react/utils/use-render

---



</details>

**Use `mergeProps` to preserve caller props instead of plain object spread.**

The spread order at lines 65 and 94 allows `containerProps` and `itemProps` to overwrite caller-provided values (onClick, style, aria-*, etc.) with the same keys. Once lost in the spread, these props cannot be recovered by the `mergeProps` call later. Compose using `mergeProps<"div">(containerProps, props)` and `mergeProps<"button">(itemProps, { ...props, children })` so caller props take precedence, then extract and merge the style separately.

<details>
<summary>Suggested fix</summary>

```diff
 function Tree<T>({
   indent = 20,
   tree,
   className,
   toggleIconType = "chevron",
   ...props
 }: TreeProps<T>) {
   const containerProps = tree.getContainerProps();
-  const { style: propStyle, ...otherProps } = { ...props, ...containerProps };
+  const { style: propStyle, ...otherProps } = mergeProps<"div">(containerProps, props);

   const mergedStyle = {
     ...propStyle,
     "--tree-indent": `${indent}px`,
   } as React.CSSProperties;
@@
 function TreeItem<T>({ item, className, render, children, ...props }: TreeItemProps<T>) {
@@
   const itemProps = item.getProps();
-  const { style: propStyle, ...otherProps } = { ...props, children, ...itemProps };
+  const { style: propStyle, ...otherProps } = mergeProps<"button">(itemProps, {
+    ...props,
+    children,
+  });
```
</details>

Also applies to: 93-95

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@packages/ui/src/components/reui/tree.tsx` around lines 64 - 66, Replace the
current object-spread composition that does "{ ...props, ...containerProps }"
and "{ ...props, children, ...itemProps }" so caller props aren't overwritten:
call mergeProps<"div">(containerProps, props) to compose container props and
mergeProps<"button">(itemProps, { ...props, children }) for item props, then
extract style from both prop sets and shallow-merge styles explicitly (e.g.,
combinedStyle = { ...callerStyle, ...containerStyle } / { ...callerStyle,
...itemStyle }) before passing to the element; update uses of
tree.getContainerProps(), containerProps, itemProps, props and children
accordingly so caller-provided onClick/aria/style values win.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `packages/ui/src/components/reui/tree.tsx` still builds `otherProps` with plain object spread for both the tree container and tree items.
  - Root cause: `{ ...props, ...containerProps }` and `{ ...props, children, ...itemProps }` let headless-tree props overwrite caller-provided handlers, ARIA attributes, and styles before `mergeProps` ever runs.
  - Fix approach: compose container/item props with `mergeProps` first, then merge styles explicitly so caller props keep Base UI merge semantics and precedence.
