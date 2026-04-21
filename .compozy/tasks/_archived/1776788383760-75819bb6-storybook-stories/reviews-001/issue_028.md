---
status: resolved
file: web/src/systems/session/components/tool-renderers/stories/read-content.stories.tsx
line: 19
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575nQD,comment:PRRC_kwDOR5y4QM65B_KZ
---

# Issue 028: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify potential namespace/type import issue and tsconfig JSX/type settings.
rg -nP --type=tsx 'React\.ReactNode' web/src/systems/session/components/tool-renderers/stories
rg -nP --type=tsx '^import\s+type\s+\{[^}]*ReactNode' web/src/systems/session/components/tool-renderers/stories
fd -i 'tsconfig*.json' web | xargs -I{} sh -c 'echo "---- {} ----"; rg -n "\"jsx\"|\"types\"|\"jsxImportSource\"" "{}"'
```

Repository: compozy/agh

Length of output: 229

---

🏁 Script executed:

```shell
cat -n web/src/systems/session/components/tool-renderers/stories/read-content.stories.tsx
```

Repository: compozy/agh

Length of output: 1451

---

🏁 Script executed:

```shell
cd web && npm run typecheck 2>&1 | grep -A5 -B5 "read-content.stories"
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
cd web && npm run typecheck 2>&1 | head -100
```

Repository: compozy/agh

Length of output: 420

---

🏁 Script executed:

```shell
rg 'React\.ReactNode' web/src/systems --type ts --type tsx -A 2 -B 2
```

Repository: compozy/agh

Length of output: 82

---

🏁 Script executed:

```shell
rg 'import.*ReactNode' web/src/systems --type ts --type tsx | head -20
```

Repository: compozy/agh

Length of output: 82

---

🏁 Script executed:

```shell
rg 'React\.ReactNode' web/src/systems -g '*.tsx' -g '*.ts' -A 1 -B 1
```

Repository: compozy/agh

Length of output: 16936

---

🏁 Script executed:

```shell
rg 'import.*type.*ReactNode|import.*ReactNode' web/src/systems -g '*.tsx' -g '*.ts' | head -20
```

Repository: compozy/agh

Length of output: 1796

---



</details>

**Import `ReactNode` and define props with an interface.**

Line 19 references `React.ReactNode` without importing `React` or `ReactNode`. With `"jsx": "react-jsx"` in tsconfig, the React import is not needed for JSX, but the namespace reference `React.ReactNode` requires an explicit import. Define the props shape using an interface per coding guidelines.

<details>
<summary>Proposed change</summary>

```diff
+import type { ReactNode } from "react";
 import type { Meta, StoryObj } from "@storybook/react-vite";
@@
-function ReadFrame({ children }: { children: React.ReactNode }) {
+interface ReadFrameProps {
+  children: ReactNode;
+}
+
+function ReadFrame({ children }: ReadFrameProps) {
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In
`@web/src/systems/session/components/tool-renderers/stories/read-content.stories.tsx`
at line 19, The ReadFrame component uses React.ReactNode but React/ReactNode is
not imported and the project prefers an explicit props interface; fix by
importing { ReactNode } from 'react' (or import React and use ReactNode) and
replace the inline type annotation with a named interface (e.g., interface
ReadFrameProps { children: ReactNode }) then update the function signature to
function ReadFrame({ children }: ReadFrameProps) so the type reference is
explicit and follows the coding guideline.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes: Current `web` typecheck passes, so this is not a live compiler break in the current toolchain, but the story still relies on `React.ReactNode` through the ambient namespace and uses an inline props shape. Fix by importing `ReactNode` explicitly and moving the wrapper props to a named interface to match the codebase’s explicit type style.

## Resolution

- Imported `ReactNode`, added a named `ReadFrameProps` interface, and updated the story to use the aliased `ReadContent` import with explicit empty `args`.

## Verification

- Covered by `web/src/storybook/web-storybook-stories-and-fixtures.test.tsx`, `make web-typecheck`, `make web-test`, and `make verify`.
