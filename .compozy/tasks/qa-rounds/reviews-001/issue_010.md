---
status: resolved
file: web/src/systems/agent/components/stories/agent-sessions-list.stories.tsx
line: 33
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59sdE_,comment:PRRC_kwDOR5y4QM67ae4_
---

# Issue 010: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Don’t seed the failure story from `codexSessions[0]` unguarded.**

If the fixtures stop containing `"codex-agent"`, this spread throws during module evaluation and breaks the whole story file. Build the failure row from an explicit base fixture or guard the lookup first.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/agent/components/stories/agent-sessions-list.stories.tsx`
around lines 13 - 33, codexSessionsWithFailure currently spreads
codexSessions[0] without checking its existence which will throw during module
evaluation if codexSessions no longer contains the expected fixture; update the
construction of codexSessionsWithFailure (and any use of codexSessions[0]) to
either (a) build the failure session from an explicit base fixture object (copy
required fields and override id/name/state/etc.) instead of spreading
codexSessions[0], or (b) first guard the lookup by checking that codexSessions
&& codexSessions.length > 0 and fall back to a safe default baseSession when
missing; adjust the identifier referenced (codexSessionsWithFailure,
codexSessions[0]) accordingly so the story never throws at import time.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `codexSessionsWithFailure` spreads `codexSessions[0]` at module evaluation time.
  - If the fixture set no longer includes a `codex-agent` session, the story file can crash during import before Storybook renders anything.
  - Fix by using an explicit fallback `SessionPayload` base before creating the failure row.
  - Resolution: added an explicit fallback codex session fixture and imported the story in the story regression test.
  - Verification: targeted Vitest passed; `make verify` passed.
