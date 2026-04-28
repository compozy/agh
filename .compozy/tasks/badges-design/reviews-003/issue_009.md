---
status: pending
file: packages/ui/src/tokens.css
line: 49
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM5-O8FB,comment:PRRC_kwDOR5y4QM68JGPu
---

# Issue 009: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Align the protocol-kind token names with the active kind enum.**

This block introduces `--color-kind-recipe`, but the rest of this PR still renders a `capability` kind. With those names out of sync, `capability` has no dedicated token to resolve against and will either fall back or inherit the wrong color. Please make the token name and the runtime kind name consistent before shipping.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@packages/ui/src/tokens.css` around lines 42 - 49, The CSS defines a
protocol-kind token --color-kind-recipe but the runtime enum uses "capability",
so update the token name to match the active kind: replace or add
--color-kind-capability (instead of --color-kind-recipe) in the protocol-kind
colors block so the runtime lookup for "capability" resolves to the intended
color; ensure any usages reference --color-kind-capability consistently.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
