---
status: pending
file: web/src/systems/network/components/network-create-channel-dialog.tsx
line: 127
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM56sg4c,comment:PRRC_kwDOR5y4QM63ZMIS
---

# Issue 023: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Expose the agent selection state to assistive tech.**

Each row behaves like a toggle, but the selected state is only visual. Add `aria-pressed={isSelected}` or switch to checkbox semantics so screen readers can tell which agents are selected. 

<details>
<summary>Suggested fix</summary>

```diff
                     <button
+                      aria-pressed={isSelected}
                       className={cn(
                         "flex w-full items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3 text-left transition-colors last:border-b-0",
                         "hover:bg-[color:var(--color-surface)]",
                         isSelected && "bg-[color:var(--color-surface)]"
                       )}
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
                    <button
                      aria-pressed={isSelected}
                      className={cn(
                        "flex w-full items-center gap-3 border-b border-[color:var(--color-divider)] px-4 py-3 text-left transition-colors last:border-b-0",
                        "hover:bg-[color:var(--color-surface)]",
                        isSelected && "bg-[color:var(--color-surface)]"
                      )}
                      data-testid={`network-agent-option-${agent.name}`}
                      key={agent.name}
                      onClick={() => onToggleAgent(agent.name)}
                      type="button"
                    >
                      <span
                        className={cn(
                          "flex size-4 shrink-0 items-center justify-center rounded border",
                          isSelected
                            ? "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]"
                            : "border-[color:var(--color-divider)] bg-transparent text-transparent"
                        )}
                      >
                        <Check className="size-3" />
                      </span>
                      <AgentIcon
                        className="size-4 text-[color:var(--color-text-tertiary)]"
                        provider={agent.provider}
                      />
                      <span className="min-w-0 flex-1 truncate text-sm text-[color:var(--color-text-primary)]">
                        {agent.name}
                      </span>
                      <span className="shrink-0 font-mono text-[0.64rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
                        {agent.provider}
                      </span>
                    </button>
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/network/components/network-create-channel-dialog.tsx` around
lines 96 - 127, The toggle button for each agent (the <button> that uses
onToggleAgent and data-testid={`network-agent-option-${agent.name}`}) currently
only conveys selection visually via isSelected; update the element to expose
state to assistive tech by adding aria-pressed={isSelected} (or replace the
button with a semantic checkbox/input and corresponding label if you prefer
checkbox semantics) and ensure the accessible name/label includes
agent.name/provider so screen readers can identify which agent is toggled; keep
the existing onToggleAgent handler and visual classes unchanged.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Notes:
