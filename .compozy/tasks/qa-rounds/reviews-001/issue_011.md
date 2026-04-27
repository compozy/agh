---
status: resolved
file: web/src/systems/agent/components/stories/agent-sessions-list.stories.tsx
line: 59
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59sdFA,comment:PRRC_kwDOR5y4QM67ae5A
---

# Issue 011: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
# First, let's locate and examine the file
fd -t f "agent-sessions-list.stories.tsx" web/
```

Repository: compozy/agh

Length of output: 128

---

🏁 Script executed:

```shell
# Read the file to see the actual imports and the Frame function
cat -n web/src/systems/agent/components/stories/agent-sessions-list.stories.tsx | head -70
```

Repository: compozy/agh

Length of output: 2379

---

🏁 Script executed:

```shell
# Check if React is imported anywhere in the file
rg "^import.*React" web/src/systems/agent/components/stories/agent-sessions-list.stories.tsx
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check the tsconfig or TypeScript configuration to understand how React types are resolved
fd -t f "tsconfig.json" | head -5
```

Repository: compozy/agh

Length of output: 265

---

🏁 Script executed:

```shell
# Check how React 19 is configured in the project - look for package.json react version
rg '"react":\s*"' package.json
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Check if there are other Storybook story files that might show the pattern for handling children types
rg "React\.ReactNode" web/src/systems/ -A 1 -B 1 | head -30
```

Repository: compozy/agh

Length of output: 2453

---

🌐 Web query:

`React 19 JSX runtime ReactNode type reference without import TypeScript`

💡 Result:

In React 19 with the new JSX runtime ("jsx": "react-jsx" in tsconfig.json), there is no global ReactNode type available without importing React. The ReactNode type is defined in `@types/react`'s index types as type ReactNode = ReactChild | ReactFragment | ReactPortal | boolean | null | undefined, but to use it (e.g., for component return types), you must import it from 'react': import { ReactNode } from 'react'; function MyComponent: ReactNode { return <div>Hello</div>; } This is because React 19 scoped the JSX namespace to React.JSX (in react/jsx-runtime.d.ts for automatic runtime), removing the global JSX namespace pollution. TypeScript falls back to global JSX only if the scoped one is unavailable, but ReactNode is not part of JSX—it's a React-specific type requiring the React module. The common build error "The inferred type cannot be named without a reference to '@types/react'" occurs when TypeScript cannot emit declarations for inferred ReactElement/JSX.Element types without explicit React types installed/updated to ^19.0.0. Explicitly typing components as ReactNode (with import) resolves this portably. No configuration makes ReactNode globally available without import in React 19 JSX runtime; the design requires importing React types for use.

Citations:

- 1: https://github.com/DefinitelyTyped/DefinitelyTyped/pull/69022
- 2: https://ar.react.dev/blog/2024/04/25/react-19-upgrade-guide
- 3: https://www.typescriptlang.org/docs/handbook/jsx.html
- 4: https://github.com/facebook/react/releases/tag/v19.0.0
- 5: https://react.dev/blog/2024/04/25/react-19-upgrade-guide
- 6: https://github.com/DefinitelyTyped/DefinitelyTyped/discussions/71395
- 7: https://www.jsdocs.io/package/@types/react

---

🏁 Script executed:

```shell
# Check the web package's tsconfig.json
cat -n web/tsconfig.json 2>/dev/null || cat -n web/tsconfig.app.json 2>/dev/null || echo "No tsconfig found in web/"
```

Repository: compozy/agh

Length of output: 983

---

🏁 Script executed:

```shell
# Check the root tsconfig.json
cat -n tsconfig.json | head -50
```

Repository: compozy/agh

Length of output: 602

---

🏁 Script executed:

```shell
# Verify if the package.json shows React version
rg '"react"' package.json -A 1 -B 1
```

Repository: compozy/agh

Length of output: 37

---

🏁 Script executed:

```shell
# Look at one of the other story files to see if they have React imports
head -20 web/src/systems/tasks/components/stories/tasks-list-panel.stories.tsx
```

Repository: compozy/agh

Length of output: 672

---



</details>

**Import `ReactNode` explicitly — React 19 requires it.**

Using `React.ReactNode` without importing `React` or `ReactNode` fails TypeScript strict mode under React 19's automatic JSX runtime (`"jsx": "react-jsx"` in tsconfig). This same issue exists in multiple story files (e.g., `tasks-kanban-board.stories.tsx`, `tasks-list-panel.stories.tsx`). Use an explicit type import instead.

<details>
<summary>🛠️ Suggested fix</summary>

```diff
+import type { ReactNode } from "react";
 import type { Meta, StoryObj } from "@storybook/react-vite";
@@
-function Frame({ children }: { children: React.ReactNode }) {
+function Frame({ children }: { children: ReactNode }) {
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/stories/agent-sessions-list.stories.tsx`
around lines 53 - 59, The Frame component uses React.ReactNode but
React/ReactNode isn't explicitly imported which breaks TypeScript with the
react-jsx runtime; update the top of the file to add an explicit type import
(e.g., import type { ReactNode } from "react") and then change the component
signature to use ReactNode ({ children }: { children: ReactNode }) — apply the
same explicit type import and replacement in other story files that use
React.ReactNode (e.g., tasks-kanban-board.stories.tsx,
tasks-list-panel.stories.tsx).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The story file references `React.ReactNode` without importing `React`, which is unsafe under the React 19 automatic JSX runtime.
  - The batch scope covers the agent story file, not the unrelated task story examples mentioned in the prompt, so the fix is limited to the scoped file.
  - Fix by importing `ReactNode` as a type and using it through a named `FrameProps` interface.
  - Resolution: added an explicit `ReactNode` import and `FrameProps` interface in the scoped story file.
  - Verification: `make web-typecheck` passed; `make verify` passed.
