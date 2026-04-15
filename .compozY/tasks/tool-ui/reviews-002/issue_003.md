---
status: resolved
file: web/src/systems/session/components/message-markdown.tsx
line: 19
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57JF8q,comment:PRRC_kwDOR5y4QM63_uH7
---

# Issue 003: _⚠️ Potential issue_ | _🟡 Minor_

## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Use `@/*` alias import for `CopyButton`.**

This relative import should follow the project alias convention.

<details>
<summary>🔧 Proposed fix</summary>

```diff
-import { CopyButton } from "./copy-button";
+import { CopyButton } from "@/systems/session/components/copy-button";
```

</details>

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
import { CopyButton } from "@/systems/session/components/copy-button";
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/message-markdown.tsx` at line 19, Replace
the relative import of CopyButton in message-markdown.tsx with the project path
alias: change the import from "./copy-button" to the equivalent alias path under
@ (e.g. "@/systems/session/components/copy-button") so the CopyButton symbol is
imported via the `@/`* mapping instead of a relative path.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `invalid`
- Reasoning: `message-markdown.tsx` imports `CopyButton` from the same component directory, which matches existing local component patterns such as `message-bubble.tsx` and other neighboring files. The codebase does not enforce alias imports for same-directory references, so changing this import would be stylistic churn rather than a corrective fix.
- Resolution: No code change required. Closed after validating the local import conventions used throughout the surrounding session components.
