---
status: resolved
file: web/src/hooks/routes/use-session-page-controls.ts
line: 12
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM58pG6s,comment:PRRC_kwDOR5y4QM66AonF
---

# Issue 013: _🛠️ Refactor suggestion_ | _🟠 Major_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_

**Use the session public barrel instead of internal cross-system paths.**

This route hook imports from `@/systems/session/adapters/...`, `.../hooks/...`, and `.../types`. Please route cross-system usage through `@/systems/session` to preserve system boundaries.



As per coding guidelines, "Cross-system imports MUST only go through the public barrel (`@/systems/<domain>`). Never reach into another system's internals."

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/hooks/routes/use-session-page-controls.ts` around lines 6 - 12, The
imports in this hook currently reach into session internals; replace the
internal paths with the session public barrel by importing
useClearSessionConversation, useDeleteSession, useResumeSession, useStopSession
and SessionPayload from "@/systems/session" (the system's public barrel) so all
cross-system usage goes through the public API; update the import statement to
reference "@/systems/session" and remove any direct references to
adapters/hooks/types paths.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  `use-session-page-controls.ts` reaches into `session` internals through adapter, hook, and type paths instead of using the system barrel. I will switch the hook to import session APIs from `@/systems/session` and add the minimal missing barrel export in `web/src/systems/session/index.ts` so the cross-system boundary is clean.
