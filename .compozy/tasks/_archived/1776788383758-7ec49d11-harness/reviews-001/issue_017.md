---
status: resolved
file: internal/daemon/prompt_input_composite.go
line: 174
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57-dM4,comment:PRRC_kwDOR5y4QM65IPEN
---

# Issue 017: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Propagate the real prompt metadata into `ResolvePrompt`.**

This always resolves policies against `acp.PromptMeta{}`. As a result, metadata-driven selection cannot distinguish synthetic reentry, network prompts, or any other prompt-scoped flags, so the chosen augmenters can diverge from the actual submission context.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/prompt_input_composite.go` around lines 171 - 174, The code
is always calling c.resolver.ResolvePrompt with an empty acp.PromptMeta{} which
prevents metadata-driven selection; change the call to pass the actual prompt
metadata extracted from the current context (e.g. use any existing metadata on
info or session rather than acp.PromptMeta{}). Locate the call to
c.resolver.ResolvePrompt(info, sess.CurrentTurnSource(), acp.PromptMeta{}) and
construct/forward the real PromptMeta (from info, sess, or the request payload)
so ResolvePrompt receives the true prompt-scoped flags instead of an empty
struct.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `promptInputComposite.Augment` always resolves policy with `acp.PromptMeta{}`, which drops the real prompt metadata for the active turn.
  - This is not just a theoretical gap: for synthetic turns the resolver requires synthetic metadata, so resolving with an empty meta can fail augmentation entirely once the composite is active on synthetic dispatch.
  - A correct fix requires minimal out-of-scope session plumbing to expose the active prompt metadata alongside `CurrentTurnSource`, then forwarding that metadata into `ResolvePrompt`; I will add regression coverage through the composite integration test.
