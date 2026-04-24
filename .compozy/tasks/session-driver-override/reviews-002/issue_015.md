---
status: resolved
file: web/src/systems/session/components/session-create-dialog.tsx
line: 65
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RcPg,comment:PRRC_kwDOR5y4QM6628EO
---

# Issue 015: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Validate selected agent/provider against current options before enabling submit.**

At Line 59, `canSubmit` only checks non-empty `selectedAgentName`/`selectedProvider`. If options refresh and either value becomes stale, submit can still proceed with an invalid payload.



<details>
<summary>Suggested fix</summary>

```diff
   const activeAgent = agents.find(agent => agent.name === selectedAgentName);
   const hasAgents = agents.length > 0;
   const hasProviderOptions = providerOptions.length > 0;
+  const hasSelectedAgent = agents.some(agent => agent.name === selectedAgentName);
+  const hasSelectedProvider = providerOptions.some(option => option.name === selectedProvider);
   const workspaceSelected = workspace !== undefined;
   const canSubmit =
     !isSubmitting &&
+    !providersLoading &&
     workspaceSelected &&
     hasAgents &&
-    selectedAgentName.trim().length > 0 &&
+    hasSelectedAgent &&
     hasProviderOptions &&
-    selectedProvider.trim().length > 0;
+    hasSelectedProvider;
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/src/systems/session/components/session-create-dialog.tsx` around lines 59
- 65, The canSubmit check allows stale selections because it only checks
non-empty strings; update it to also verify the selections exist in the current
options arrays (e.g., ensure selectedAgentName matches an entry in agentOptions
and selectedProvider matches an entry in providerOptions) before enabling
submit: modify the canSubmit boolean to include existence checks (for example
using agentOptions.some(a => a.name === selectedAgentName.trim()) and
providerOptions.some(p => p.id === selectedProvider.trim() or p.name ===
selectedProvider.trim() depending on your provider shape), keep the existing
workspaceSelected/hasAgents/hasProviderOptions/isSubmitting guards, and trim
values when comparing. Ensure these symbols (canSubmit, selectedAgentName,
selectedProvider, agentOptions, providerOptions, hasAgents, hasProviderOptions)
are referenced so the check remains correct after options refreshes.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `UNREVIEWED`
- Decision: `valid`
- Notes: `canSubmit` only checks for non-empty selection strings, so stale agent/provider values remain submittable after the available options refresh. I will require both selections to resolve against the current option lists and add component coverage for the stale-selection case.
