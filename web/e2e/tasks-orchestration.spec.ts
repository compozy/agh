import path from "node:path";
import { fileURLToPath } from "node:url";

import { tasksOperatorSelectors } from "./fixtures/selectors";
import { seedBrowserTasksOperatorFlow } from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";
import { useGlobalWorkspaceIfPrompted } from "./fixtures/workspace";

const browserLifecycleFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "browser_session_lifecycle_fixture.json"
);

const tasksSessionAgentName = "browser-lifecycle-agent";

test.use({
  runtimeOptions: {
    seed: {
      mockAgents: [
        {
          agentName: tasksSessionAgentName,
          fixtureAgent: tasksSessionAgentName,
          fixturePath: browserLifecycleFixture,
        },
      ],
    },
  },
});

test("operator inspects orchestration tab on a real seeded task", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const tasksUI = tasksOperatorSelectors(appPage);
  const seeded = await seedBrowserTasksOperatorFlow(runtime, {
    sessionAgentName: tasksSessionAgentName,
  });

  await useGlobalWorkspaceIfPrompted(tasksUI);

  await tasksUI.navTasks.click();
  await expect(appPage).toHaveURL(/\/tasks$/);

  await appPage.goto(runtime.url(`/tasks/${seeded.referenceTask.id}`), {
    waitUntil: "domcontentloaded",
  });
  await expect(tasksUI.detailContent).toBeVisible();

  await tasksUI.detailTabOrchestration.click();
  await expect(tasksUI.orchestrationPanel).toBeVisible();
  await expect(tasksUI.orchestrationProfileCard).toBeVisible();
  await expect(tasksUI.orchestrationReviewsCard).toBeVisible();
  await expect(tasksUI.orchestrationNotificationsCard).toBeVisible();
  await expect(tasksUI.orchestrationStreamCard).toBeVisible();

  // Seeded tasks land with a default execution profile (all `inherit`); reviews and bridge
  // notification subscriptions are unset, so those cards stay in their empty branches.
  await expect(tasksUI.orchestrationProfileSummary).toBeVisible();
  await expect(tasksUI.orchestrationReviewsEmpty).toBeVisible();
  await expect(tasksUI.orchestrationNotificationsEmpty).toBeVisible();

  await expect(tasksUI.orchestrationStreamLatest).toBeVisible();
  await expect(tasksUI.orchestrationStreamSeed).toBeVisible();
  await expect(tasksUI.orchestrationStreamStatus).toBeVisible();

  await browserArtifacts.captureScreenshot("tasks-orchestration-tab", appPage);
});
