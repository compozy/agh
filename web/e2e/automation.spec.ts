import { fileURLToPath } from "node:url";
import path from "node:path";

import { automationOperatorSelectors, sessionLifecycleSelectors } from "./fixtures/selectors";
import {
  browserAutomationOperatorFlowScenario,
  seedBrowserAutomationOperatorFlow,
} from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";

const automationTaskFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "automation_task_fixture.json"
);

const automationAgentName = "browser-automation-runner";

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          fixturePath: automationTaskFixture,
          fixtureAgent: "automation-runner",
          agentName: automationAgentName,
        },
      ],
    },
  },
});

test("operator can edit automation, trigger a real run, and inspect the linked session transcript", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const automationUI = automationOperatorSelectors(appPage);
  const sessionUI = sessionLifecycleSelectors(appPage);
  const seeded = await seedBrowserAutomationOperatorFlow(runtime, {
    agentName: automationAgentName,
  });

  if (await automationUI.workspaceOnboarding.isVisible()) {
    await automationUI.workspaceUseGlobal.click();
  }

  await expect(automationUI.appSidebar).toBeVisible();
  await expect(automationUI.navAutomation).toBeVisible();
  await automationUI.navAutomation.click();

  await expect(appPage).toHaveURL(/\/automation$/);
  await expect(automationUI.kindJobs).toHaveAttribute("aria-pressed", "true");
  await expect(automationUI.listPanel).toBeVisible();
  await expect(automationUI.item(seeded.job.id)).toBeVisible();
  await expect(automationUI.detailPanel).toContainText(seeded.job.name);
  await expect(automationUI.detailPanel).toContainText(
    browserAutomationOperatorFlowScenario.job.prompt
  );
  await expect(automationUI.runHistory).toBeVisible();
  await expect(automationUI.run(seeded.baselineRun.id)).toBeVisible();
  await expect(automationUI.runHistory).toContainText("Completed");
  await expect(automationUI.runHistory).toContainText(seeded.baselineRun.session_id ?? "");
  await expect(automationUI.runSessionLink(seeded.baselineRun.id)).toBeVisible();

  await automationUI.kindTriggers.click();
  await expect(automationUI.kindTriggers).toHaveAttribute("aria-pressed", "true");
  await expect(automationUI.item(seeded.trigger.id)).toBeVisible();
  await expect(automationUI.detailPanel).toContainText(seeded.trigger.name);
  await expect(automationUI.detailPanel).toContainText(
    browserAutomationOperatorFlowScenario.trigger.webhookID
  );

  await automationUI.kindJobs.click();
  await expect(automationUI.kindJobs).toHaveAttribute("aria-pressed", "true");
  await automationUI.item(seeded.job.id).click();

  await expect(automationUI.editAutomationButton).toBeVisible();
  await automationUI.editAutomationButton.click();
  await expect(automationUI.jobForm).toBeVisible();

  await automationUI.jobNameInput.fill(browserAutomationOperatorFlowScenario.job.editedName);
  await automationUI.jobScheduleExpr.fill(
    browserAutomationOperatorFlowScenario.job.updatedScheduleExpr
  );
  await expect(automationUI.submitJobForm).toBeEnabled();
  await automationUI.submitJobForm.click();

  await expect(automationUI.jobForm).toBeHidden();
  await expect(automationUI.detailPanel).toContainText(
    browserAutomationOperatorFlowScenario.job.editedName
  );
  await expect(automationUI.item(seeded.job.id)).toContainText(
    browserAutomationOperatorFlowScenario.job.editedName
  );

  await automationUI.triggerJobButton.click();

  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{
        runs: Array<{ id: string }>;
      }>(`/api/automation/jobs/${encodeURIComponent(seeded.job.id)}/runs?limit=10`);
      return payload.runs.length;
    })
    .toBe(2);

  let uiTriggeredRun:
    | {
        id: string;
        session_id?: string | null;
      }
    | undefined;
  await expect
    .poll(async () => {
      const runsPayload = await runtime.requestJSON<{
        runs: Array<{ id: string; session_id?: string | null }>;
      }>(`/api/automation/jobs/${encodeURIComponent(seeded.job.id)}/runs?limit=10`);
      uiTriggeredRun = runsPayload.runs.find(
        run => run.id !== seeded.baselineRun.id && run.session_id
      );
      return uiTriggeredRun?.session_id ?? "";
    })
    .not.toBe("");

  if (!uiTriggeredRun?.session_id) {
    throw new Error("Expected the UI-triggered automation run to include a linked session.");
  }

  await expect(automationUI.run(uiTriggeredRun.id)).toBeVisible();
  await browserArtifacts.captureScreenshot("automation-operator-history", appPage);

  await automationUI.runSessionLink(uiTriggeredRun.id).click();

  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(`/session/${uiTriggeredRun.session_id}`);
  await expect(sessionUI.chatHeader).toBeVisible();
  await expect(sessionUI.chatView).toContainText(browserAutomationOperatorFlowScenario.job.prompt);
  await expect(sessionUI.chatView).toContainText(
    browserAutomationOperatorFlowScenario.transcript.assistant
  );

  await browserArtifacts.captureScreenshot("automation-linked-session", appPage);
});
