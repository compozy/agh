---
status: resolved
file: web/e2e/automation.spec.ts
line: 91
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59dk1x,comment:PRRC_kwDOR5y4QM67HMXC
---

# Issue 020: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Avoid `force: true` here; it can mask broken edit UX.**

Line 91 bypasses actionability checks, so this test may pass when the user cannot actually click Edit.


<details>
<summary>Suggested fix</summary>

```diff
-  await automationUI.editAutomationButton.click({ force: true });
+  await expect(automationUI.editAutomationButton).toBeEnabled();
+  await automationUI.editAutomationButton.click();
```
</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/e2e/automation.spec.ts` at line 91, Remove the use of force:true on
automationUI.editAutomationButton.click so the test doesn't bypass actionability
checks; instead wait/assert the button is actionable (e.g.,
automationUI.editAutomationButton.should('be.visible').and('not.be.disabled') or
equivalent) and then call automationUI.editAutomationButton.click() normally,
ensuring any overlays/modals are closed before clicking and keeping the selector
automationUI.editAutomationButton as the target.
```

</details>

<!-- fingerprinting:phantom:poseidon:hawk -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `VALID`
- Notes:
  - `automationUI.editAutomationButton.click({ force: true })` bypasses Playwright actionability checks.
  - The fix is to assert the edit button is enabled and click it normally.
