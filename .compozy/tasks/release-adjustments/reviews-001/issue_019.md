---
status: resolved
file: web/e2e/automation.spec.ts
line: 68
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1s,comment:PRRC_kwDOR5y4QM67HMW8
---

# Issue 019: _⚠️ Potential issue_ | _🟡 Minor_
## Review Comment

_⚠️ Potential issue_ | _🟡 Minor_

**Scope the completion assertion to the seeded run row.**

Line 68 currently validates completion at history-panel level, so it can pass for the wrong run.


<details>
<summary>Suggested fix</summary>

```diff
-  await expect(automationUI.runHistory).toContainText(/completed/i);
+  await expect(automationUI.run(seeded.baselineRun.id)).toContainText(/completed/i);
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  await expect(automationUI.run(seeded.baselineRun.id)).toContainText(/completed/i);
```

</details>

<!-- suggestion_end -->

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/e2e/automation.spec.ts` at line 68, The assertion currently checks
automationUI.runHistory at the panel level and can match the wrong run; instead
scope the check to the specific seeded run row by locating the seeded row
element (e.g., a helper or selector like automationUI.getSeededRunRow(seedId) or
find the row by the seeded run name/id) and assert that that single row contains
/completed/i (replace the top-level automationUI.runHistory reference with the
scoped seeded-run element such as automationUI.getSeededRunRow(seedId) and call
toContainText(/completed/i) on it).
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - The current completion assertion checks the entire run history panel, so a different completed run could satisfy it.
  - The fix is to assert `/completed/i` against `automationUI.run(seeded.baselineRun.id)` after confirming the seeded row is visible.
