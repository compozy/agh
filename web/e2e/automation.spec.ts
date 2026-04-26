import { fileURLToPath } from "node:url";
import path from "node:path";

import { automationOperatorSelectors, sessionLifecycleSelectors } from "./fixtures/selectors";
import {
  browserAutomationOperatorFlowScenario,
  seedBrowserAutomationOperatorFlow,
} from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "./fixtures/workspace";

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

test("operator can inspect automation, trigger a real run, and inspect the linked session transcript", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const automationUI = automationOperatorSelectors(appPage);
  const sessionUI = sessionLifecycleSelectors(appPage);
  const seeded = await seedBrowserAutomationOperatorFlow(runtime, {
    agentName: automationAgentName,
  });

  await useGlobalWorkspaceIfPrompted(automationUI);

  await expect(automationUI.appSidebar).toBeVisible();
  await expect(automationUI.navJobs).toBeVisible();
  await automationUI.navJobs.click();

  await expect(appPage).toHaveURL(/\/jobs$/);
  await expect(automationUI.jobsShell).toBeVisible();
  await expect(automationUI.jobsScopeAll).toHaveAttribute("aria-pressed", "true");
  await expect(automationUI.listPanel).toBeVisible();
  await expect(automationUI.item(seeded.job.id)).toBeVisible();
  await expect(automationUI.detailPanel).toContainText(seeded.job.name);
  await expect(automationUI.detailPanel).toContainText(
    browserAutomationOperatorFlowScenario.job.prompt
  );
  await expect(automationUI.runHistory).toBeVisible();
  await expect(automationUI.run(seeded.baselineRun.id)).toBeVisible();
  await expect(automationUI.run(seeded.baselineRun.id)).toContainText(/completed/i);
  await expect(automationUI.runSessionLink(seeded.baselineRun.id)).toBeVisible();
  await expect(automationUI.runSessionLink(seeded.baselineRun.id)).toHaveAttribute(
    "href",
    `/session/${seeded.baselineRun.session_id}`
  );

  await automationUI.navTriggers.click();
  await expect(appPage).toHaveURL(/\/triggers$/);
  await expect(automationUI.triggersShell).toBeVisible();
  await expect(automationUI.triggersScopeAll).toHaveAttribute("aria-pressed", "true");
  await expect(automationUI.item(seeded.trigger.id)).toBeVisible();
  await expect(automationUI.detailPanel).toContainText(seeded.trigger.name);
  await expect(automationUI.detailPanel).toContainText(
    browserAutomationOperatorFlowScenario.trigger.webhookID
  );

  await automationUI.navJobs.click();
  await expect(appPage).toHaveURL(/\/jobs$/);
  await expect(automationUI.jobsShell).toBeVisible();
  await automationUI.item(seeded.job.id).click();

  await expect(automationUI.editAutomationButton).toBeVisible();
  await expect(automationUI.editAutomationButton).toBeEnabled();
  await automationUI.editAutomationButton.click();
  await expect(automationUI.jobForm).toBeVisible();
  await expect(automationUI.jobNameInput).toHaveValue(seeded.job.name);
  await expect(automationUI.jobScheduleExpr).toHaveValue(
    browserAutomationOperatorFlowScenario.job.scheduleExpr
  );
  await appPage.keyboard.press("Escape");
  await expect(automationUI.jobForm).toBeHidden();

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
