---
provider: coderabbit
pr: "113"
round: 1
round_created_at: 2026-05-06T20:42:04.329549Z
status: resolved
file: web/src/systems/agent/components/agent-command-list.tsx
line: 109
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM6AH9A3,comment:PRRC_kwDOR5y4QM6-k_Qa
---

# Issue 009: _🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_
## Review Comment

_🛠️ Refactor suggestion_ | _🟠 Major_ | _⚡ Quick win_

**Replace ad-hoc typography values with design tokens/classes**

Line 101 and Line 108 use arbitrary values (`text-[10px]`, `tracking-[0.12em]`). Please switch to tokenized/standardized typography utilities to keep web UI styling consistent.

 

<details>
<summary>Proposed change</summary>

```diff
- className="font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground"
+ className="font-mono text-xs uppercase tracking-wide text-muted-foreground"
```

```diff
- className="ml-auto truncate font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground"
+ className="ml-auto truncate font-mono text-xs uppercase tracking-wide text-muted-foreground"
```
</details>

As per coding guidelines, "Pull every color, font, radius, spacing step, and motion value from `DESIGN.md` — never invent tokens."

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
                      className="font-mono text-xs uppercase tracking-wide text-muted-foreground"
                      data-testid={`agent-command-provider-${agent.name}`}
                    >
                      {agent.provider}
                    </span>
                    {categoryLabel ? (
                      <span
                        className="ml-auto truncate font-mono text-xs uppercase tracking-wide text-muted-foreground"
                        data-testid={`agent-command-category-${agent.name}`}
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against current code. Fix only still-valid issues, skip the
rest with a brief reason, keep changes minimal, and validate.

In `@web/src/systems/agent/components/agent-command-list.tsx` around lines 101 -
109, The spans rendering agent.provider and the conditional categoryLabel in
AgentCommandList (the span elements with data-testid
agent-command-provider-{agent.name} and agent-command-category-{agent.name}) use
ad-hoc typography classes text-[10px] and tracking-[0.12em]; replace those with
the standardized design tokens/classes from your design system (e.g., the
tokenized font-size and letter-spacing utility names defined in DESIGN.md / your
tailwind config) so both className values use the approved typography utilities
instead of arbitrary values, keeping the rest of the existing classes
(font-mono, uppercase, text-muted-foreground, truncate, ml-auto) intact.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `web/src/systems/agent/components/agent-command-list.tsx` still uses arbitrary `text-[10px]` and `tracking-[0.12em]` utilities for provider/category metadata.
  - Root cause: these spans bypass the repo’s standardized typography utilities for mono metadata labels.
  - Fix approach: replace the ad-hoc values with the existing tokenized utility classes used for uppercase mono metadata.
