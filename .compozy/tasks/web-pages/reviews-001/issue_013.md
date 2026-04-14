---
status: resolved
file: web/src/routes/_app/-network.test.tsx
line: 76
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4M,comment:PRRC_kwDOR5y4QM63ZMH9
---

# Issue 013: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

<details>
<summary>🧩 Analysis chain</summary>

🌐 Web query:

`In TypeScript with the modern React JSX transform, can a file reference the type `React.ReactNode` without importing the `React` namespace, or should it use an imported `ReactNode`/`import type * as React from "react"`?`

💡 Result:

No, in TypeScript with the modern React JSX transform ("jsx": "react-jsx" or "react-jsxdev"), a file cannot reference React.ReactNode without importing the React namespace. You must import React (typically via import * as React from 'react';) to access React.ReactNode, or use import type { ReactNode } from 'react'; for the type-only import. The new JSX transform eliminates the need to import React for JSX syntax itself (no more import React from 'react'; at the top for JSX to work), as it uses automatic runtime importing from react/jsx-runtime. However, React-specific types like ReactNode, which are defined in the main React types (index.d.ts in `@types/react`), are part of the React namespace/module and are not re-exported or made globally available by the jsx-runtime types. TypeScript docs confirm that for classic "react" mode, React must be imported for the JSX namespace (React.JSX). For modern react-jsx modes, the JSX namespace comes from react/jsx-runtime, which re-exports JSX types by extending React.JSX (e.g., interface Element extends React.JSX.Element {}), but does not include ReactNode (a top-level type for renderable content). ReactNode remains in the main React module. Official React docs and cheatsheets consistently show children: React.ReactNode with React imported. No sources indicate ReactNode becomes usable without import in modern setups.

Citations:

- 1: https://www.typescriptlang.org/docs/handbook/jsx.html
- 2: https://beta.reactjs.org/learn/typescript
- 3: https://react.dev/learn/typescript
- 4: https://github.com/DefinitelyTyped/DefinitelyTyped/blob/HEAD/types/react/index.d.ts
- 5: https://github.com/DefinitelyTyped/DefinitelyTyped/blob/master/types/react/v17/jsx-runtime.d.ts
- 6: https://github.com/DefinitelyTyped/DefinitelyTyped/blob/0b728411cd1dfb4bd26992bb35a73cf8edaa22e7/types/react/jsx-runtime.d.ts

---

🏁 Script executed:

```shell
cat -n web/src/routes/_app/-network.test.tsx | head -100
```

Repository: compozy/agh

Length of output: 3616

---

🏁 Script executed:

```shell
cat -n web/src/routes/_app/-network.test.tsx | sed -n '1,10p; 70,80p; 300,315p'
```

Repository: compozy/agh

Length of output: 1346

---



</details>

**Replace `React.ReactNode` with the imported `ReactNode` type.**

The file imports `ReactNode` from React (line 3) but references `React.ReactNode` at lines 74 and 307 without importing the `React` namespace. This causes TypeScript errors that block `make web-typecheck`.

<details>
<summary>Proposed fix</summary>

```diff
-  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
+  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
     component: opts.component,
   }),
```

```diff
-const NetworkPage = (Route as unknown as { component: () => React.ReactNode }).component;
+const NetworkPage = (Route as unknown as { component: () => ReactNode }).component;
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/routes/_app/-network.test.tsx` around lines 74 - 76, The type
reference uses the React namespace (React.ReactNode) but the file imports the
bare ReactNode type; update the type annotations to use the imported ReactNode
instead of React.ReactNode wherever they appear (for example in the
createFileRoute factory return type and the other route/component type
declarations that reference React.ReactNode) so TypeScript uses the imported
symbol ReactNode consistently.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: `make web-typecheck` passes on the current tree, so the claimed compiler failure does not reproduce in this repository.
- Reasoning: sibling route tests in the same codebase use the same `React.ReactNode` pattern without importing the React namespace and still pass the project typecheck, so this is not a batch blocker.
