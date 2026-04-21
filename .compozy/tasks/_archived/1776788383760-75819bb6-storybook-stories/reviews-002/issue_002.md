---
status: resolved
file: web/src/components/ui/stories/combobox.stories.tsx
line: 30
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575wGz,comment:PRRC_kwDOR5y4QM65CLB3
---

# Issue 002: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

<details>
<summary>🧩 Analysis chain</summary>

🏁 Script executed:

```shell
#!/bin/bash
# Verify object-shape aliases in web TS/TSX files (guideline expects interface)
rg -nP --type=ts --type=tsx -C2 '\btype\s+[A-Za-z0-9_]+\s*=\s*\{' web/src
```

Repository: compozy/agh

Length of output: 82

---

🏁 Script executed:

```shell
cat -n web/src/components/ui/stories/combobox.stories.tsx | head -40
```

Repository: compozy/agh

Length of output: 1361

---



</details>

**Use `interface` instead of `type` for object shape definitions.**

Per coding guidelines (`web/**/*.ts?(x)`), object shapes must use `interface` rather than `type` aliases.

<details>
<summary>♻️ Proposed fix</summary>

```diff
-type CityOption = { value: string; label: string };
+interface CityOption {
+  value: string;
+  label: string;
+}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
interface CityOption {
  value: string;
  label: string;
}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/components/ui/stories/combobox.stories.tsx` at line 30, Replace the
type alias CityOption with an interface declaration to follow project
conventions: change "type CityOption = { value: string; label: string }" to
"interface CityOption { value: string; label: string }" and update any local
usages in the combobox.stories.tsx story (e.g., props, variables, function
signatures referencing CityOption) to ensure the interface name is used
consistently.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Notes:
  - Verified the current story file and searched the repository instructions for a rule that object-shape aliases in `web/**/*.ts(x)` must use `interface`.
  - No such repository rule is present in the applicable `AGENTS.md`/`CLAUDE.md` files, and the local `CityOption` alias is a small story-local shape with no behavioral defect or type-safety issue.
  - Changing this to `interface` would be style churn only, with no correctness or maintainability gain required by the current project rules.
  - Analysis complete; no code change was warranted.
