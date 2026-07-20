import { fileURLToPath } from "node:url";
import path from "node:path";

import type { AutomationJob, AutomationSuggestion } from "@/systems/automation";
import { automationOperatorSelectors, sessionLifecycleSelectors } from "../fixtures/selectors";
import {
  browserAutomationOperatorFlowScenario,
  seedBrowserAutomationOperatorFlow,
} from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const automationTaskFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "automation_task_fixture.json"
);

const automationAgentName = "browser-automation-runner";

function automationSessionPath(sessionId: string): string {
  return `/agents/${automationAgentName}/sessions/${sessionId}`;
}

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

  await expect(automationUI.osDesktop).toBeVisible();
  await expect(automationUI.navJobs).toBeVisible();
  await automationUI.navJobs.click();

  await expect(appPage).toHaveURL(/\/jobs$/);
  await expect(automationUI.jobsShell).toBeVisible();
  await expect(automationUI.jobsListRows).toBeVisible();
  await expect(automationUI.item(seeded.job.id)).toBeVisible();
  await automationUI.itemLink(seeded.job.id).click();

  await expect(appPage).toHaveURL(new RegExp(`/jobs/${seeded.job.id}$`));
  await expect(automationUI.detailPanel).toBeVisible();
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
  await expect(automationUI.triggersListRows).toBeVisible();
  await expect(automationUI.item(seeded.trigger.id)).toBeVisible();
  await automationUI.itemLink(seeded.trigger.id).click();

  await expect(appPage).toHaveURL(new RegExp(`/triggers/${seeded.trigger.id}$`));
  await expect(automationUI.detailPanel).toContainText(seeded.trigger.name);
  await expect(automationUI.detailPanel).toContainText(
    browserAutomationOperatorFlowScenario.trigger.webhookID
  );

  await expect(automationUI.editAutomationButton).toBeVisible();
  await expect(automationUI.editAutomationButton).toBeEnabled();
  await automationUI.editAutomationButton.click();
  await expect(automationUI.triggerNameInput).toHaveValue(seeded.trigger.name);
  const triggerDialog = appPage.getByTestId("automation-editor-dialog");
  await expect(triggerDialog).toHaveAttribute("data-frame", "unframed");
  await expect(triggerDialog.locator('[data-slot="dialog-header"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
  await expect(triggerDialog.locator('[data-slot="dialog-footer"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
  await expect(appPage.getByTestId("trigger-retry-max")).toBeVisible();
  await appPage.getByTestId("trigger-governance-toggle").click();
  await expect(appPage.getByTestId("trigger-retry-max")).toBeHidden();
  await appPage.getByTestId("trigger-governance-toggle").click();
  await expect(appPage.getByTestId("trigger-retry-max")).toBeVisible();
  await appPage.keyboard.press("Escape");
  await expect(triggerDialog).toBeHidden();

  await automationUI.navJobs.click();
  await expect(appPage).toHaveURL(/\/jobs$/);
  await expect(automationUI.jobsShell).toBeVisible();
  await automationUI.itemLink(seeded.job.id).click();
  await expect(appPage).toHaveURL(new RegExp(`/jobs/${seeded.job.id}$`));

  await expect(automationUI.editAutomationButton).toBeVisible();
  await expect(automationUI.editAutomationButton).toBeEnabled();
  await automationUI.editAutomationButton.click();
  await expect(automationUI.jobForm).toBeVisible();
  const jobDialog = appPage.getByTestId("automation-editor-dialog");
  await expect(jobDialog).toHaveAttribute("data-frame", "unframed");
  await expect(jobDialog.locator('[data-slot="dialog-header"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
  await expect(jobDialog.locator('[data-slot="dialog-footer"]')).toHaveAttribute(
    "data-variant",
    "ruled"
  );
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
    .poll(
      async () => {
        const runsPayload = await runtime.requestJSON<{
          runs: Array<{ id: string; session_id?: string | null }>;
        }>(`/api/automation/jobs/${encodeURIComponent(seeded.job.id)}/runs?limit=10`);
        uiTriggeredRun = runsPayload.runs.find(
          run => run.id !== seeded.baselineRun.id && run.session_id
        );
        return uiTriggeredRun?.session_id ?? "";
      },
      {
        timeout: 20_000,
      }
    )
    .not.toBe("");

  if (!uiTriggeredRun?.session_id) {
    throw new Error("Expected the UI-triggered automation run to include a linked session.");
  }

  await expect(automationUI.run(uiTriggeredRun.id)).toBeVisible();
  await browserArtifacts.captureScreenshot("automation-operator-history", appPage);

  await automationUI.runSessionLink(uiTriggeredRun.id).click();

  await expect
    .poll(() => new URL(appPage.url()).pathname)
    .toBe(automationSessionPath(uiTriggeredRun.session_id));
  await expect(sessionUI.chatView).toBeVisible();
  await expect(sessionUI.chatView).toContainText(browserAutomationOperatorFlowScenario.job.prompt);
  await expect(sessionUI.chatView).toContainText(
    browserAutomationOperatorFlowScenario.transcript.assistant
  );

  await browserArtifacts.captureScreenshot("automation-linked-session", appPage);
});

test("operator can accept and dismiss workspace suggestions through the real daemon", async ({
  appPage,
  runtime,
}) => {
  const automationUI = automationOperatorSelectors(appPage);

  await useGlobalWorkspaceIfPrompted(automationUI);
  const workspaces = await runtime.requestJSON<{
    workspaces: Array<{ id: string }>;
  }>("/api/workspaces");
  const workspaceID = workspaces.workspaces[0]?.id;
  if (!workspaceID) {
    throw new Error("Expected the browser Automation scenario to have an active workspace.");
  }

  const suggestionPath = `/api/workspaces/${encodeURIComponent(workspaceID)}/automation/suggestions`;
  const initial = await runtime.requestJSON<{ suggestions: AutomationSuggestion[] }>(
    `${suggestionPath}?status=pending`
  );
  expect(initial.suggestions).toHaveLength(4);

  const acceptTarget = initial.suggestions.find(
    suggestion => suggestion.payload.name === "Daily workspace briefing"
  );
  const dismissTarget = initial.suggestions.find(
    suggestion => suggestion.payload.name === "Weekday standup draft"
  );
  if (!acceptTarget || !dismissTarget) {
    throw new Error("Expected the deterministic starter suggestion catalog.");
  }

  await automationUI.navJobs.click();
  await expect(appPage).toHaveURL(/\/jobs$/);
  await expect(automationUI.automationSuggestionsCard).toBeVisible();

  const acceptedRow = automationUI.suggestion(acceptTarget.id);
  await expect(acceptedRow).toContainText(acceptTarget.payload.prompt);
  await acceptedRow.getByRole("button", { name: "Create job" }).click();

  await expect(acceptedRow).toBeHidden();
  await expect(automationUI.item(acceptTarget.payload.id)).toBeVisible();
  const acceptedJob = await runtime.requestJSON<{ job: AutomationJob }>(
    `/api/automation/jobs/${encodeURIComponent(acceptTarget.payload.id)}`
  );
  expect(acceptedJob.job).toMatchObject({
    enabled: true,
    id: acceptTarget.payload.id,
    workspace_id: workspaceID,
  });

  const dismissedRow = automationUI.suggestion(dismissTarget.id);
  await dismissedRow.getByRole("button", { name: "Dismiss" }).click();
  await expect(dismissedRow).toBeHidden();

  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(automationUI.jobsShell).toBeVisible();
  await expect(automationUI.suggestion(acceptTarget.id)).toBeHidden();
  await expect(automationUI.suggestion(dismissTarget.id)).toBeHidden();

  const accepted = await runtime.requestJSON<{ suggestions: AutomationSuggestion[] }>(
    `${suggestionPath}?status=accepted`
  );
  const dismissed = await runtime.requestJSON<{ suggestions: AutomationSuggestion[] }>(
    `${suggestionPath}?status=dismissed`
  );
  expect(accepted.suggestions.map(suggestion => suggestion.id)).toContain(acceptTarget.id);
  expect(dismissed.suggestions.map(suggestion => suggestion.id)).toContain(dismissTarget.id);
});
