---
status: resolved
file: web/e2e/automation.spec.ts
line: 135
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57wEcj,comment:PRRC_kwDOR5y4QM640q1I
---

# Issue 034: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**The triggered run is not the one whose session link gets verified.**

After finding `uiTriggeredRun`, the test only checks that it appears in history, then it clicks `runSessionLink(seeded.baselineRun.id)`. A regression in the newly triggered run's linked session would still pass this test.

<details>
<summary>💡 Suggested fix</summary>

```diff
   const runsPayload = await runtime.requestJSON<{
     runs: Array<{ id: string; session_id?: string | null }>;
   }>(`/api/automation/jobs/${encodeURIComponent(seeded.job.id)}/runs?limit=10`);
   const uiTriggeredRun = runsPayload.runs.find(run => run.id !== seeded.baselineRun.id);

   expect(uiTriggeredRun).toBeTruthy();
+  expect(uiTriggeredRun?.session_id).toBeTruthy();
   await expect(automationUI.run(uiTriggeredRun?.id ?? "")).toBeVisible();
   await browserArtifacts.captureScreenshot("automation-operator-history", appPage);

-  await automationUI.runSessionLink(seeded.baselineRun.id).click();
+  await automationUI.runSessionLink(uiTriggeredRun?.id ?? "").click();

   await expect(appPage).toHaveURL(
     new RegExp(
-      `/session/${(seeded.baselineRun.session_id ?? "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`
+      `/session/${(uiTriggeredRun?.session_id ?? "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`
     )
   );
```
</details>

<!-- suggestion_start -->

<details>
<summary>📝 Committable suggestion</summary>

> ‼️ **IMPORTANT**
> Carefully review the code before committing. Ensure that it accurately replaces the highlighted code, contains no missing lines, and has no issues with indentation. Thoroughly test & benchmark the code to ensure it meets the requirements.

```suggestion
  const runsPayload = await runtime.requestJSON<{
    runs: Array<{ id: string; session_id?: string | null }>;
  }>(`/api/automation/jobs/${encodeURIComponent(seeded.job.id)}/runs?limit=10`);
  const uiTriggeredRun = runsPayload.runs.find(run => run.id !== seeded.baselineRun.id);

  expect(uiTriggeredRun).toBeTruthy();
  expect(uiTriggeredRun?.session_id).toBeTruthy();
  await expect(automationUI.run(uiTriggeredRun?.id ?? "")).toBeVisible();
  await browserArtifacts.captureScreenshot("automation-operator-history", appPage);

  await automationUI.runSessionLink(uiTriggeredRun?.id ?? "").click();

  await expect(appPage).toHaveURL(
    new RegExp(
      `/session/${(uiTriggeredRun?.session_id ?? "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`
    )
  );
  await expect(sessionUI.chatHeader).toBeVisible();
  await expect(sessionUI.chatView).toContainText(browserAutomationOperatorFlowScenario.job.prompt);
  await expect(sessionUI.chatView).toContainText(
    browserAutomationOperatorFlowScenario.transcript.assistant
  );

  await browserArtifacts.captureScreenshot("automation-linked-session", appPage);
```

</details>

<!-- suggestion_end -->

<details>
<summary>🧰 Tools</summary>

<details>
<summary>🪛 ast-grep (0.42.1)</summary>

[warning] 124-126: Regular expression constructed from variable input detected. This can lead to Regular Expression Denial of Service (ReDoS) attacks if the variable contains malicious patterns. Use libraries like 'recheck' to validate regex safety or use static patterns.
Context: new RegExp(
      `/session/${(seeded.baselineRun.session_id ?? "").replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`
    )
Note: [CWE-1333] Inefficient Regular Expression Complexity [REFERENCES]
    - https://owasp.org/www-community/attacks/Regular_expression_Denial_of_Service_-_ReDoS
    - https://cwe.mitre.org/data/definitions/1333.html

(regexp-from-variable)

</details>

</details>

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@web/e2e/automation.spec.ts` around lines 113 - 135, The test currently
verifies the baseline run's session link instead of the newly found run: after
locating uiTriggeredRun in runsPayload.runs, change the subsequent actions to
use uiTriggeredRun (e.g., call automationUI.runSessionLink(uiTriggeredRun.id)
and navigate/verify using uiTriggeredRun.session_id) rather than
seeded.baselineRun.id; also assert uiTriggeredRun exists and has a session_id
before using it so the URL and chat assertions target the actual triggered run
(references: uiTriggeredRun, runsPayload, automationUI.runSessionLink,
automationUI.run, seeded.baselineRun).
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  The test correctly discovers the UI-triggered run, but then follows the
  baseline run's session link. That means a regression in the newly triggered
  run's linked session would go undetected. The assertions need to target the
  triggered run and its `session_id`.

## Resolution

- The automation flow now waits for the UI-triggered run to publish a
  `session_id` and uses that exact run for the session navigation assertion.
