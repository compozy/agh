import path from "node:path";
import { fileURLToPath } from "node:url";

import { sessionLifecycleSelectors, tasksOperatorSelectors } from "./fixtures/selectors";
import { seedBrowserTasksOperatorFlow } from "./fixtures/runtime";
import { expect, test } from "./fixtures/test";

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
const createdDraftDescription =
  "Use the shared browser lane to capture fresh Tasks evidence for task_19.";
const createdDraftTitle = "Draft Tasks browser evidence rollout";

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

test("operator can execute the shipped Tasks flow through the shared daemon-served browser lane", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const sessionUI = sessionLifecycleSelectors(appPage);
  const tasksUI = tasksOperatorSelectors(appPage);
  const seeded = await seedBrowserTasksOperatorFlow(runtime, {
    sessionAgentName: tasksSessionAgentName,
  });

  if (await tasksUI.workspaceOnboarding.isVisible()) {
    await tasksUI.workspaceUseGlobal.click();
  }

  await expect(tasksUI.appSidebar).toBeVisible();
  await expect(tasksUI.navTasks).toBeVisible();
  await tasksUI.navTasks.click();

  await expect(appPage).toHaveURL(/\/tasks$/);
  await expect(tasksUI.modeList).toHaveAttribute("aria-pressed", "true");
  await expect(tasksUI.taskCard(seeded.referenceTask.id)).toBeVisible();
  await expect(tasksUI.taskCard(seeded.approvalTask.id)).toBeVisible();
  await expect(tasksUI.taskCard(seeded.runningTask.id)).toBeVisible();
  await expect(tasksUI.detailPreviewPanel).toBeVisible();
  await browserArtifacts.captureScreenshot("tasks-list-seeded", appPage);

  await tasksUI.openCreate.click();
  await expect(appPage).toHaveURL(/\/tasks\/new$/);
  await expect(tasksUI.createEditorSurface).toBeVisible();
  await tasksUI.createPriority("high").click();
  await tasksUI.createTitle.fill(createdDraftTitle);
  await tasksUI.createDescription.fill(createdDraftDescription);
  await expect(tasksUI.createSaveDraft).toBeEnabled();
  await tasksUI.createSaveDraft.click();
  await expect(tasksUI.createEditorSurface).toBeHidden();

  let createdDraftId = "";
  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{
        tasks: Array<{ id: string; status: string; title: string }>;
      }>(`/api/tasks?include_drafts=true&query=${encodeURIComponent(createdDraftTitle)}&limit=10`);
      const createdTask = payload.tasks.find(task => task.title === createdDraftTitle);
      createdDraftId = createdTask?.id ?? "";
      return createdTask?.status ?? "";
    })
    .toBe("draft");

  if (createdDraftId === "") {
    throw new Error(`Expected a created draft task for "${createdDraftTitle}".`);
  }

  await expect(tasksUI.taskCard(createdDraftId)).toBeVisible();
  await tasksUI.taskCard(createdDraftId).click();
  await expect(tasksUI.detailContent).toContainText(createdDraftTitle);
  await expect(tasksUI.detailPublish).toBeVisible();
  await browserArtifacts.captureScreenshot("tasks-draft-created", appPage);

  const publishResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(createdDraftId)}/publish`)
    );
  });
  await tasksUI.detailPublish.click();
  const publishResponse = await publishResponsePromise;
  expect(publishResponse.ok()).toBeTruthy();
  await expect(publishResponse.json()).resolves.toMatchObject({
    task: {
      status: "ready",
    },
  });

  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{
        task: {
          summary?: { status?: string | null };
          task?: { status?: string | null };
        };
      }>(`/api/tasks/${encodeURIComponent(createdDraftId)}`);
      return payload.task.summary?.status ?? payload.task.task?.status ?? "";
    })
    .toBe("ready");
  await expect(tasksUI.detailPublish).toBeHidden();
  await browserArtifacts.captureScreenshot("tasks-draft-published", appPage);

  await expect(tasksUI.detailContent).toBeVisible();
  await expect(tasksUI.detailContent).toContainText(createdDraftTitle);
  await expect(tasksUI.detailTab("timeline")).toBeVisible();
  await browserArtifacts.captureScreenshot("tasks-detail-route", appPage);

  await tasksUI.detailTabAgents.click();
  await expect
    .poll(async () => {
      return (
        (await tasksUI.multiAgentEmpty.isVisible()) ||
        (await tasksUI.multiAgentNoActive.isVisible()) ||
        (await tasksUI.multiAgentDisconnected.isVisible())
      );
    })
    .toBe(true);
  await browserArtifacts.captureScreenshot("tasks-live-fallback", appPage);

  await tasksUI.detailBreadcrumbTasks.click();
  await expect(appPage).toHaveURL(/\/tasks$/);
  await tasksUI.modeDashboard.click();
  await expect(tasksUI.dashboardView).toBeVisible();
  await expect(tasksUI.dashboardActiveRun(seeded.runningRun.id)).toBeVisible();
  await browserArtifacts.captureScreenshot("tasks-dashboard", appPage);

  const activeRunPath = `/tasks/${seeded.runningTask.id}/runs/${seeded.runningRun.id}`;
  const activeRunLink = tasksUI.dashboardActiveRunLink(seeded.runningRun.id);
  await expect(activeRunLink).toBeVisible();
  await expect(activeRunLink).toHaveAttribute("href", activeRunPath);
  await appPage.goto(runtime.url(activeRunPath), {
    waitUntil: "domcontentloaded",
  });
  await expect(tasksUI.runDetailContent).toBeVisible();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(activeRunPath);
  await expect(tasksUI.runSessionDrilldown).toBeVisible();
  await browserArtifacts.captureScreenshot("tasks-run-detail", appPage);

  await tasksUI.runSessionDrilldown.click();
  await expect.poll(() => new URL(appPage.url()).pathname).toBe(`/session/${seeded.session.id}`);
  await expect(sessionUI.chatHeader).toBeVisible();
  await browserArtifacts.captureScreenshot("tasks-linked-session", appPage);

  await appPage.goto(runtime.url("/tasks"), {
    waitUntil: "domcontentloaded",
  });
  await expect(tasksUI.modeList).toBeVisible();
  await tasksUI.modeInbox.click();
  await expect(tasksUI.inboxView).toBeVisible();
  await expect(tasksUI.inboxLane("approvals")).toBeVisible();
  await expect(tasksUI.inboxItem(seeded.approvalTask.id)).toBeVisible();
  await browserArtifacts.captureScreenshot("tasks-inbox-approval-pending", appPage);

  const approveResponsePromise = appPage.waitForResponse(response => {
    return (
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(seeded.approvalTask.id)}/approve`)
    );
  });
  await tasksUI.inboxApprove(seeded.approvalTask.id).click();
  const approveResponse = await approveResponsePromise;
  expect(approveResponse.ok()).toBeTruthy();
  await expect(approveResponse.json()).resolves.toMatchObject({
    task: {
      approval_state: "approved",
    },
  });

  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{
        task: {
          summary?: { approval_state?: string | null };
          task?: { approval_state?: string | null };
        };
      }>(`/api/tasks/${encodeURIComponent(seeded.approvalTask.id)}`);
      return payload.task.summary?.approval_state ?? payload.task.task?.approval_state ?? "";
    })
    .toBe("approved");
  await expect
    .poll(async () => {
      const payload = await runtime.requestJSON<{
        inbox: {
          groups?: Array<{
            items?: Array<{
              task: {
                id: string;
              };
            }>;
          }>;
        };
      }>("/api/observe/tasks/inbox?lane=approvals&limit=10");

      return (
        payload.inbox.groups?.some(group =>
          (group.items ?? []).some(item => item.task.id === seeded.approvalTask.id)
        ) ?? false
      );
    })
    .toBe(false);
  await browserArtifacts.captureScreenshot("tasks-inbox-approval-approved", appPage);
});
