---
provider: coderabbit
pr: "113"
round: 3
round_created_at: 2026-05-06T21:26:19.336933Z
status: resolved
file: web/src/systems/agent/components/stories/agent-command-select.stories.tsx
line: 46
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AIm32,comment:PRRC_kwDOR5y4QM6-l2-c
---

# Issue 002: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify inline object-shape props in Frame declaration.
rg -n 'function\s+Frame\s*\(\{\s*children\s*\}:\s*\{\s*children:\s*React\.ReactNode\s*\}\)' web/src/systems/agent/components/stories/agent-command-select.stories.tsx
```

Repository: compozy/agh

Length of output: 120

---



</details>

**Extract `Frame` props into an `interface`.**

Inline object shapes should be defined as interfaces per coding guidelines. Define `FrameProps` and use it in the function parameter.

<details>
<summary>Diff</summary>

```diff
+interface FrameProps {
+  children: React.ReactNode;
+}
+
-function Frame({ children }: { children: React.ReactNode }) {
+function Frame({ children }: FrameProps) {
   return (
     <CenteredSurface>
       <div className="w-full max-w-md">{children}</div>
     </CenteredSurface>
   );
 }
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
interface FrameProps {
  children: React.ReactNode;
}

function Frame({ children }: FrameProps) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-md">{children}</div>
    </CenteredSurface>
  );
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@web/src/systems/agent/components/stories/agent-command-select.stories.tsx` at
line 46, The function Frame currently types its props inline; extract that
inline type into a named interface FrameProps and use it in the function
signature. Create interface FrameProps { children: React.ReactNode } (or
equivalent) and update function Frame({ children }: FrameProps) to reference the
new interface; ensure any exports or usages still compile with the renamed type.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `Frame` currently declares its props with an inline object type, and the review request matches the repository preference for named interfaces in reusable component props.
  - The fix is local and low-risk: extract a `FrameProps` interface with `children: React.ReactNode` and use it in the function signature without changing behavior.
  - No dedicated test file change was warranted because this is a type-only story refactor with no behavior change; fresh full-repo verification passed with `make verify`.
