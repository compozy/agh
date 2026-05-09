import { execFile } from "node:child_process";
import { readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

import { captureRouteState } from "../fixtures/browser-artifact-session";
import { tasksOperatorSelectors } from "../fixtures/selectors";
import { seedBrowserTasksOperatorFlow, type BrowserRuntime } from "../fixtures/runtime";
import { expect, test } from "../fixtures/test";
import { ensureGlobalWorkspace, useGlobalWorkspaceIfPrompted } from "../fixtures/workspace";

const execFileAsync = promisify(execFile);
const browserLifecycleFixture = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "..",
  "internal",
  "testutil",
  "acpmock",
  "testdata",
  "browser_session_lifecycle_fixture.json"
);
const tasksSessionAgentName = "browser-lifecycle-agent";
const sensitivePattern =
  /agh_claim_|["']claim_token["']\s*:|mcp[_-]?auth|telegram-bot-token|pkce|oauth|webhook_secret|provider[_-]?credential/i;

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

test("operator cancels a running task run and sees matching HTTP, UDS, CLI, and browser state", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const ui = tasksOperatorSelectors(appPage);
  const seeded = await seedBrowserTasksOperatorFlow(runtime, {
    sessionAgentName: tasksSessionAgentName,
  });
  await useGlobalWorkspaceIfPrompted(ui);

  const runPath = `/tasks/${encodeURIComponent(seeded.runningTask.id)}/runs/${encodeURIComponent(seeded.runningRun.id)}`;
  await appPage.goto(runtime.url(runPath), { waitUntil: "domcontentloaded" });
  await expect(ui.runDetailContent).toBeVisible();
  await expect(ui.runDetailCancel).toBeVisible();

  const cancelResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/task-runs/${encodeURIComponent(seeded.runningRun.id)}/cancel`)
  );
  await ui.runDetailCancel.click();
  expect((await cancelResponsePromise).ok()).toBe(true);

  await expect
    .poll(async () => {
      const detail = await getTaskRun(runtime, seeded.runningRun.id);
      return detail.run.status;
    })
    .toBe("canceled");
  await expect(ui.runDetailCancel).toBeHidden();
  await appPage.reload({ waitUntil: "domcontentloaded" });
  await expect(ui.runDetailContent).toContainText("canceled");
  await expect(ui.runDetailCancel).toBeHidden();

  const parity = await captureTaskRunParity(runtime, seeded.runningTask.id, seeded.runningRun.id);
  expect(parity.http.run.status).toBe("canceled");
  expect(parity.uds?.run.status).toBe("canceled");
  expect(findRun(parity.cliRuns, seeded.runningRun.id)?.status).toBe("canceled");

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", parity);
  await browserArtifacts.captureScreenshot("tasks-run-canceled", appPage);
  await browserArtifacts.persist(appPage);
  const routeState = await readRouteState(runtime);
  expect(routeState).toMatchObject({
    tasks_run_cancel_visible: false,
    tasks_run_detail_visible: true,
    tasks_selected_run: seeded.runningRun.id,
    tasks_selected_task: seeded.runningTask.id,
    tasks_view_visible: true,
  });
  await assertNoTaskSensitiveLeak(appPage, runtime, parity);
});

test("operator rejects a manual approval task without creating hidden work", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const ui = tasksOperatorSelectors(appPage);
  const seeded = await seedBrowserTasksOperatorFlow(runtime, {
    sessionAgentName: tasksSessionAgentName,
  });
  await useGlobalWorkspaceIfPrompted(ui);

  await appPage.goto(runtime.url("/tasks"), { waitUntil: "domcontentloaded" });
  await ui.modeInbox.click();
  await expect(ui.inboxLane("approvals")).toBeVisible();
  await expect(ui.inboxItem(seeded.approvalTask.id)).toBeVisible();

  const rejectResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(seeded.approvalTask.id)}/reject`)
  );
  await ui.inboxReject(seeded.approvalTask.id).click();
  expect((await rejectResponsePromise).ok()).toBe(true);

  await expect
    .poll(async () => {
      const detail = await getTask(runtime, seeded.approvalTask.id);
      return {
        approval: detail.task.approval_state,
        runs: taskDetailRuns(detail).length,
      };
    })
    .toEqual({ approval: "rejected", runs: 0 });
  await expect(ui.inboxItem(seeded.approvalTask.id)).toHaveAttribute("data-lane", "blocked");
  await expect(ui.inboxReject(seeded.approvalTask.id)).toBeHidden();
  await expect(ui.inboxApprove(seeded.approvalTask.id)).toBeHidden();

  await appPage.reload({ waitUntil: "domcontentloaded" });
  await ui.modeInbox.click();
  await expect(ui.modeInbox).toHaveAttribute("aria-pressed", "true");
  await expect(ui.inboxView).toBeVisible();
  await expect(ui.inboxItem(seeded.approvalTask.id)).toHaveAttribute("data-lane", "blocked");
  await expect(ui.inboxReject(seeded.approvalTask.id)).toBeHidden();
  await expect(ui.inboxApprove(seeded.approvalTask.id)).toBeHidden();

  const snapshot = {
    http: await getTask(runtime, seeded.approvalTask.id),
    uds: await requestOperatorJSONOrThrow<TaskDetailEnvelope>(
      runtime,
      `/api/tasks/${encodeURIComponent(seeded.approvalTask.id)}`
    ),
    cli: await taskCLI<TaskDetailView>(runtime, ["task", "get", seeded.approvalTask.id]),
  };
  expect(snapshot.http.task.approval_state).toBe("rejected");
  expect(snapshot.uds.task.task.approval_state).toBe("rejected");
  expect(snapshot.cli.task.approval_state).toBe("rejected");
  expect(taskDetailRuns(snapshot.cli)).toHaveLength(0);

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", snapshot);
  await browserArtifacts.captureScreenshot("tasks-approval-rejected", appPage);
  await browserArtifacts.persist(appPage);
  const routeState = await readRouteState(runtime);
  expect(routeState).toMatchObject({
    tasks_active_mode: "inbox",
    tasks_inbox_count: expect.any(Number),
    tasks_view_visible: true,
  });
  await assertNoTaskSensitiveLeak(appPage, runtime, snapshot);
});

test("operator retries failed work and sees an auditable run review gate", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const ui = tasksOperatorSelectors(appPage);
  const seeded = await seedBrowserTasksOperatorFlow(runtime, {
    sessionAgentName: tasksSessionAgentName,
  });
  const workerSession = await createSession(
    runtime,
    tasksSessionAgentName,
    seeded.session.workspace_id
  );
  await useGlobalWorkspaceIfPrompted(ui);

  const task = await createTask(runtime, {
    description: "Browser hardening coverage for retry and review gate behavior.",
    identifier: uniqueID("tasks-retry-review"),
    priority: "high",
    scope: "workspace",
    title: uniqueTitle("Retry and review hardening"),
    workspace: seeded.session.workspace_id,
  });
  const failedRun = await failNewRun(
    runtime,
    task.id,
    workerSession.id,
    tasksSessionAgentName,
    workerSession.channel
  );
  await expect
    .poll(async () => (await getTaskRun(runtime, failedRun.id)).run.status)
    .toBe("failed");

  await appPage.goto(runtime.url("/tasks"), { waitUntil: "domcontentloaded" });
  await ui.modeInbox.click();
  await expect(ui.inboxLane("failed_runs")).toBeVisible();
  await expect(ui.inboxItem(task.id)).toBeVisible();

  const retryResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(task.id)}/runs`)
  );
  await ui.inboxRetry(task.id).click();
  expect((await retryResponsePromise).ok()).toBe(true);
  await expect
    .poll(async () =>
      taskDetailRuns(await getTask(runtime, task.id))
        .map(run => run.status)
        .sort()
    )
    .toEqual(["failed", "queued"]);

  const requested = await runtime.requestJSON<TaskRunReviewRequestEnvelope>(
    `/api/task-runs/${encodeURIComponent(failedRun.id)}/reviews`,
    {
      method: "POST",
      body: JSON.stringify({
        policy: "always",
        reason: "browser hardening review gate evidence",
      }),
    }
  );
  expect(requested.created).toBe(true);
  const verdict = await runtime.requestJSON<TaskRunReviewVerdictEnvelope>(
    `/api/task-reviews/${encodeURIComponent(requested.review.review_id)}/verdict`,
    {
      method: "POST",
      body: JSON.stringify({
        run_id: failedRun.id,
        verdict: {
          confidence: 0.91,
          delivery_id: uniqueID("browser-review"),
          outcome: "approved",
          reason: "retry lane and artifact evidence verified",
          review_text: "Browser hardening review gate passed with retry evidence.",
        },
      }),
    }
  );
  expect(verdict.review.status).toBe("recorded");
  expect(verdict.review.outcome).toBe("approved");

  const invalidVerdictResponse = await fetch(
    runtime.url(`/api/task-reviews/${encodeURIComponent(requested.review.review_id)}/verdict`),
    {
      method: "POST",
      body: JSON.stringify({
        run_id: failedRun.id,
        verdict: {
          confidence: 0.5,
          delivery_id: uniqueID("browser-review-invalid"),
          outcome: "not_a_review_outcome",
          reason: "browser hardening invalid verdict probe",
          review_text: "This invalid outcome must be rejected.",
        },
      }),
    }
  );
  const invalidVerdictBody = await invalidVerdictResponse.text();
  expect(invalidVerdictResponse.ok).toBe(false);
  expect(invalidVerdictResponse.status).toBe(400);
  expect(invalidVerdictBody).toMatch(/invalid|outcome|validation|verdict/i);

  await appPage.goto(
    runtime.url(`/tasks/${encodeURIComponent(task.id)}/runs/${encodeURIComponent(failedRun.id)}`),
    { waitUntil: "domcontentloaded" }
  );
  await expect(ui.runDetailContent).toBeVisible();
  await expect(ui.runReviewRow(requested.review.review_id)).toBeVisible();
  await expect(ui.runReviewRow(requested.review.review_id)).toContainText("approved");

  const snapshot = {
    http: {
      detail: await getTask(runtime, task.id),
      failedRun: await getTaskRun(runtime, failedRun.id),
      reviews: await runtime.requestJSON<TaskRunReviewsEnvelope>(
        `/api/task-runs/${encodeURIComponent(failedRun.id)}/reviews`
      ),
    },
    uds: await requestOperatorJSONOrThrow<TaskRunReviewsEnvelope>(
      runtime,
      `/api/task-runs/${encodeURIComponent(failedRun.id)}/reviews`
    ),
    cliReviews: await taskCLI<TaskRunReview[]>(runtime, [
      "task",
      "review",
      "list",
      "--run",
      failedRun.id,
    ]),
    cliRuns: await taskCLI<TaskRun[]>(runtime, ["task", "run", "list", task.id, "--last", "10"]),
    invalidVerdict: {
      body: invalidVerdictBody,
      status: invalidVerdictResponse.status,
    },
  };
  expect(snapshot.http.reviews.reviews).toHaveLength(1);
  expect(snapshot.uds.reviews).toHaveLength(1);
  expect(snapshot.cliReviews).toHaveLength(1);
  expect(findRun(snapshot.cliRuns, failedRun.id)?.status).toBe("failed");

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", snapshot);
  await browserArtifacts.captureScreenshot("tasks-retry-review-gate", appPage);
  await browserArtifacts.persist(appPage);
  const routeState = await readRouteState(runtime);
  expect(routeState).toMatchObject({
    tasks_run_detail_visible: true,
    tasks_selected_run: failedRun.id,
    tasks_selected_task: task.id,
  });
  expect(Number(routeState.tasks_review_count)).toBeGreaterThanOrEqual(1);
  await assertNoTaskSensitiveLeak(appPage, runtime, snapshot);
});

test("operator inspects child and dependency graph, edits the task, and deletes disposable work", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const ui = tasksOperatorSelectors(appPage);
  await ensureGlobalWorkspace(runtime);

  const parent = await createTask(runtime, {
    description: "Parent task for graph hardening.",
    identifier: uniqueID("tasks-graph-parent"),
    scope: "global",
    title: uniqueTitle("Graph parent"),
  });
  const dependency = await createTask(runtime, {
    description: "Dependency task for graph hardening.",
    identifier: uniqueID("tasks-graph-dep"),
    scope: "global",
    title: uniqueTitle("Graph dependency"),
  });
  const child = await createChildTask(runtime, parent.id, {
    description: "Child task for graph hardening.",
    identifier: uniqueID("tasks-graph-child"),
    scope: "global",
    title: uniqueTitle("Graph child"),
  });
  await addDependency(runtime, child.id, dependency.id);

  await appPage.goto(runtime.url(`/tasks/${encodeURIComponent(parent.id)}`), {
    waitUntil: "domcontentloaded",
  });
  await expect(ui.detailContent).toBeVisible();
  await ui.detailTab("children").click();
  await expect(ui.detailChildItem(child.id)).toBeVisible();
  await expect(ui.detailChildLink(child.id)).toHaveAttribute(
    "href",
    `/tasks/${encodeURIComponent(child.id)}`
  );
  await runtime.artifactCollector.captureJSON(
    "browser_route_state",
    await captureRouteState(appPage)
  );
  const parentRouteState = await readRouteState(runtime);
  expect(parentRouteState).toMatchObject({
    tasks_children_count: 1,
    tasks_detail_visible: true,
    tasks_selected_task: parent.id,
  });

  await appPage.goto(runtime.url(`/tasks/${encodeURIComponent(child.id)}`), {
    waitUntil: "domcontentloaded",
  });
  await expect(ui.detailContent).toBeVisible();
  await ui.detailTab("dependencies").click();
  await expect(ui.detailDependencyItem(dependency.id)).toBeVisible();
  await expect(ui.detailDependencyLink(dependency.id)).toHaveAttribute(
    "href",
    `/tasks/${encodeURIComponent(dependency.id)}`
  );
  await runtime.artifactCollector.captureJSON(
    "browser_route_state",
    await captureRouteState(appPage)
  );
  const childRouteState = await readRouteState(runtime);
  expect(childRouteState).toMatchObject({
    tasks_dependencies_count: 1,
    tasks_detail_visible: true,
    tasks_selected_task: child.id,
  });

  const blockedRunResponse = await fetch(
    runtime.url(`/api/tasks/${encodeURIComponent(child.id)}/runs`),
    {
      method: "POST",
      body: JSON.stringify({
        idempotency_key: uniqueID("blocked-start"),
      }),
    }
  );
  expect(blockedRunResponse.ok).toBe(true);
  const blockedRunPayload = (await blockedRunResponse.json()) as TaskRunEnvelope;
  const blockedClaimResponse = await fetch(
    runtime.url(`/api/task-runs/${encodeURIComponent(blockedRunPayload.run.id)}/claim`),
    {
      method: "POST",
      body: JSON.stringify({
        idempotency_key: uniqueID("blocked-claim"),
      }),
    }
  );
  const blockedClaimBody = await blockedClaimResponse.text();
  expect(blockedClaimResponse.ok).toBe(false);
  expect(blockedClaimResponse.status).toBeGreaterThanOrEqual(400);
  expect(blockedClaimBody).toMatch(/blocked|dependency|invalid/i);

  const editedTitle = uniqueTitle("Graph child edited");
  await ui.detailEdit.click();
  await expect(ui.createEditorSurface).toBeVisible();
  await ui.createTitle.fill(editedTitle);
  const editResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "PATCH" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(child.id)}`)
  );
  await ui.createSubmit.click();
  expect((await editResponsePromise).ok()).toBe(true);
  await expect.poll(async () => (await getTask(runtime, child.id)).task.title).toBe(editedTitle);
  await expect(ui.detailContent).toContainText(editedTitle);

  const disposable = await createTask(runtime, {
    description: "Disposable task for delete hardening.",
    draft: true,
    identifier: uniqueID("tasks-delete"),
    scope: "global",
    title: uniqueTitle("Disposable delete"),
  });
  await appPage.goto(runtime.url(`/tasks/${encodeURIComponent(disposable.id)}`), {
    waitUntil: "domcontentloaded",
  });
  await expect(ui.detailDelete).toBeVisible();
  await ui.detailDelete.click();
  await expect(ui.detailDeleteDialog).toBeVisible();
  await runtime.artifactCollector.captureJSON(
    "browser_route_state",
    await captureRouteState(appPage)
  );
  const deleteDialogRouteState = await readRouteState(runtime);
  expect(deleteDialogRouteState).toMatchObject({
    tasks_detail_delete_dialog_open: true,
    tasks_detail_visible: true,
    tasks_selected_task: disposable.id,
  });
  const deleteResponsePromise = appPage.waitForResponse(
    response =>
      response.request().method() === "DELETE" &&
      response.url().endsWith(`/api/tasks/${encodeURIComponent(disposable.id)}`)
  );
  await ui.detailDeleteConfirm.click();
  expect((await deleteResponsePromise).ok()).toBe(true);
  await expect.poll(() => new URL(appPage.url()).pathname).toBe("/tasks");
  await expect(
    fetch(runtime.url(`/api/tasks/${encodeURIComponent(disposable.id)}`))
  ).resolves.toMatchObject({
    status: 404,
  });

  const snapshot = {
    http: {
      parent: await getTask(runtime, parent.id),
      child: await getTask(runtime, child.id),
      dependency: await getTask(runtime, dependency.id),
    },
    uds: await requestOperatorJSONOrThrow<TaskTreeEnvelope>(
      runtime,
      `/api/tasks/${encodeURIComponent(parent.id)}/tree`
    ),
    cliChild: await taskCLI<TaskDetailView>(runtime, ["task", "get", child.id]),
    blockedStart: {
      body: blockedClaimBody,
      run: blockedRunPayload.run,
      status: blockedClaimResponse.status,
    },
  };
  expect(snapshot.http.parent.children.map(item => item.id)).toContain(child.id);
  expect(snapshot.http.child.dependency_references.map(item => item.depends_on.id)).toContain(
    dependency.id
  );
  expect(JSON.stringify(snapshot.uds.tree)).toContain(child.id);
  expect(snapshot.cliChild.dependencies.map(edge => edge.depends_on_task_id)).toContain(
    dependency.id
  );

  await runtime.artifactCollector.captureJSON("browser_api_snapshots", snapshot);
  await browserArtifacts.captureScreenshot("tasks-graph-edit-delete", appPage);
  await browserArtifacts.persist(appPage);
  await assertNoTaskSensitiveLeak(appPage, runtime, snapshot);
});

test("tasks list, inbox, detail, and run detail stay usable across responsive breakpoints", async ({
  appPage,
  browserArtifacts,
  runtime,
}) => {
  const ui = tasksOperatorSelectors(appPage);
  const seeded = await seedBrowserTasksOperatorFlow(runtime, {
    sessionAgentName: tasksSessionAgentName,
  });
  await useGlobalWorkspaceIfPrompted(ui);

  for (const viewport of [
    { height: 820, name: "mobile", width: 375 },
    { height: 900, name: "tablet", width: 768 },
    { height: 900, name: "desktop", width: 1280 },
  ]) {
    await appPage.setViewportSize({ height: viewport.height, width: viewport.width });

    await appPage.goto(runtime.url("/tasks"), { waitUntil: "domcontentloaded" });
    await ui.modeList.click();
    await expect(ui.modeList).toHaveAttribute("aria-pressed", "true");
    await revealTasksListPanel(appPage);
    await expect(appPage.getByTestId("tasks-list-panel")).toBeVisible();
    await expect(appPage.getByTestId("tasks-list-lane-pills")).toBeVisible();
    const listSearch = appPage.getByTestId("tasks-list-search-input");
    await expect(listSearch).toBeVisible();
    await listSearch.fill(`no-task-${viewport.name}-${viewport.width}`);
    await expect(appPage.getByTestId("tasks-list-empty")).toBeVisible();
    await listSearch.fill("");
    await expect(ui.taskCard(seeded.referenceTask.id)).toBeVisible();

    await ui.modeInbox.click();
    await expect(ui.modeInbox).toHaveAttribute("aria-pressed", "true");
    await expect(ui.inboxView).toBeVisible();
    await expect(appPage.getByTestId("tasks-inbox-lane-tabs")).toBeVisible();
    const inboxSearch = appPage.getByTestId("tasks-inbox-search");
    await expect(inboxSearch).toBeVisible();
    await inboxSearch.fill(`no-inbox-${viewport.name}-${viewport.width}`);
    await expect(appPage.getByTestId("tasks-inbox-empty")).toBeVisible();

    await appPage.goto(runtime.url(`/tasks/${encodeURIComponent(seeded.referenceTask.id)}`), {
      waitUntil: "domcontentloaded",
    });
    await expect(ui.detailContent).toBeVisible();
    await expect(ui.detailTab("timeline")).toBeVisible();
    await expect(ui.detailTab("runs")).toBeVisible();
    await ui.detailTab("runs").click();
    await expect(ui.detailRunsEmpty).toBeVisible();

    await appPage.goto(
      runtime.url(
        `/tasks/${encodeURIComponent(seeded.runningTask.id)}/runs/${encodeURIComponent(
          seeded.runningRun.id
        )}`
      ),
      { waitUntil: "domcontentloaded" }
    );
    await expect(ui.runDetailContent).toBeVisible();
    await expect(ui.runDetailCancel).toBeVisible();
    await browserArtifacts.captureScreenshot(`tasks-responsive-${viewport.name}`, appPage);
  }
});

interface TaskRecord {
  id: string;
  approval_state?: string | null;
  status: string;
  title: string;
}

interface TaskRun {
  id: string;
  status: string;
  task_id: string;
}

interface SessionPayload {
  channel?: string | null;
  id: string;
  workspace_id?: string | null;
}

interface TaskRunDetailView {
  run: TaskRun;
}

interface TaskDependency {
  depends_on_task_id: string;
}

interface TaskDependencyReference {
  depends_on: TaskRecord;
}

interface TaskDetailView {
  task: TaskRecord;
  children: TaskRecord[];
  dependencies: TaskDependency[];
  dependency_references: TaskDependencyReference[];
  runs?: TaskRun[];
  task_runs?: TaskRun[];
}

interface TaskDetailEnvelope {
  task: TaskDetailView;
}

interface TaskRunEnvelope {
  run: TaskRun;
}

interface TaskRunDetailEnvelope {
  run: TaskRunDetailView;
}

interface TaskRunReviewsEnvelope {
  reviews: TaskRunReview[];
}

interface TaskRunReview {
  review_id: string;
  run_id: string;
  status: string;
  outcome?: string | null;
}

interface TaskRunReviewRequestEnvelope {
  created: boolean;
  review: TaskRunReview;
}

interface TaskRunReviewVerdictEnvelope {
  review: TaskRunReview;
}

interface TaskTreeEnvelope {
  tree: unknown;
}

interface TaskRunParitySnapshot {
  cliRuns: TaskRun[];
  http: TaskRunDetailView;
  uds?: TaskRunDetailView;
}

interface AgentTaskNextRecord {
  claimed: boolean;
  claim?: {
    run: TaskRun;
    lease?: {
      claim_token_hash?: string;
    };
  };
}

interface AgentTaskLeaseRecord {
  run_id: string;
  status: string;
}

async function createTask(
  runtime: BrowserRuntime,
  body: Record<string, unknown>
): Promise<TaskRecord> {
  return (
    await runtime.requestJSON<{ task: TaskRecord }>("/api/tasks", {
      method: "POST",
      body: JSON.stringify(body),
    })
  ).task;
}

async function createSession(
  runtime: BrowserRuntime,
  agentName: string,
  workspaceID: string
): Promise<SessionPayload> {
  return (
    await runtime.requestJSON<{ session: SessionPayload }>("/api/sessions", {
      method: "POST",
      body: JSON.stringify({
        agent_name: agentName,
        workspace: workspaceID,
      }),
    })
  ).session;
}

async function createChildTask(
  runtime: BrowserRuntime,
  parentID: string,
  body: Record<string, unknown>
): Promise<TaskRecord> {
  return (
    await runtime.requestJSON<{ task: TaskRecord }>(
      `/api/tasks/${encodeURIComponent(parentID)}/children`,
      {
        method: "POST",
        body: JSON.stringify(body),
      }
    )
  ).task;
}

async function addDependency(
  runtime: BrowserRuntime,
  taskID: string,
  dependsOnTaskID: string
): Promise<TaskDetailView> {
  return (
    await runtime.requestJSON<TaskDetailEnvelope>(
      `/api/tasks/${encodeURIComponent(taskID)}/dependencies`,
      {
        method: "POST",
        body: JSON.stringify({
          depends_on_task_id: dependsOnTaskID,
          kind: "blocks",
        }),
      }
    )
  ).task;
}

async function getTask(runtime: BrowserRuntime, taskID: string): Promise<TaskDetailView> {
  return (await runtime.requestJSON<TaskDetailEnvelope>(`/api/tasks/${encodeURIComponent(taskID)}`))
    .task;
}

async function getTaskRun(runtime: BrowserRuntime, runID: string): Promise<TaskRunDetailView> {
  return (
    await runtime.requestJSON<TaskRunDetailEnvelope>(`/api/task-runs/${encodeURIComponent(runID)}`)
  ).run;
}

async function enqueueRun(
  runtime: BrowserRuntime,
  taskID: string,
  networkChannel?: string | null
): Promise<TaskRun> {
  return (
    await runtime.requestJSON<TaskRunEnvelope>(`/api/tasks/${encodeURIComponent(taskID)}/runs`, {
      method: "POST",
      body: JSON.stringify({
        idempotency_key: uniqueID("enqueue"),
        network_channel: networkChannel ?? undefined,
      }),
    })
  ).run;
}

async function failNewRun(
  runtime: BrowserRuntime,
  taskID: string,
  sessionID: string,
  agentName: string,
  networkChannel?: string | null
): Promise<TaskRun> {
  const run = await enqueueRun(runtime, taskID, networkChannel);
  const next = await agentTaskCLI<AgentTaskNextRecord>(runtime, sessionID, agentName, [
    "task",
    "next",
  ]);
  expect(next.stdout).not.toMatch(sensitivePattern);
  if (!next.payload.claimed) {
    throw new Error(`task next did not claim run ${run.id}: ${next.stdout}`);
  }
  expect(next.payload.claim?.run.id).toBe(run.id);
  expect(next.payload.claim?.lease?.claim_token_hash).toBeTruthy();

  const failed = await agentTaskCLI<AgentTaskLeaseRecord>(runtime, sessionID, agentName, [
    "task",
    "fail",
    run.id,
    "--error",
    "browser hardening injected failure",
    "--metadata",
    JSON.stringify({ evidence: "retry-lane" }),
  ]);
  expect(failed.stdout).not.toMatch(sensitivePattern);
  expect(failed.payload).toMatchObject({
    run_id: run.id,
    status: "failed",
  });

  return (await getTaskRun(runtime, run.id)).run;
}

async function captureTaskRunParity(
  runtime: BrowserRuntime,
  taskID: string,
  runID: string
): Promise<TaskRunParitySnapshot> {
  const uds = await requestOperatorJSONOrThrow<TaskRunDetailEnvelope>(
    runtime,
    `/api/task-runs/${encodeURIComponent(runID)}`
  );
  return {
    cliRuns: await taskCLI<TaskRun[]>(runtime, ["task", "run", "list", taskID, "--last", "10"]),
    http: await getTaskRun(runtime, runID),
    uds: uds.run,
  };
}

async function taskCLI<T>(runtime: BrowserRuntime, args: string[]): Promise<T> {
  if (!runtime.paths) {
    throw new Error("task CLI checks require launch-mode runtime paths.");
  }
  const { stdout } = await execFileAsync(runtime.paths.cliShim, [...args, "-o", "json"], {
    env: cliEnv(runtime.paths),
  });
  expect(stdout).not.toMatch(sensitivePattern);
  return JSON.parse(stdout) as T;
}

async function agentTaskCLI<T>(
  runtime: BrowserRuntime,
  sessionID: string,
  agentName: string,
  args: string[]
): Promise<{ payload: T; stdout: string }> {
  if (!runtime.paths) {
    throw new Error("agent task CLI checks require launch-mode runtime paths.");
  }
  const { stdout } = await execFileAsync(runtime.paths.cliShim, [...args, "-o", "json"], {
    env: {
      ...cliEnv(runtime.paths),
      AGH_AGENT: agentName,
      AGH_AGENT_NAME: agentName,
      AGH_SESSION_ID: sessionID,
    },
  });
  return {
    payload: JSON.parse(stdout) as T,
    stdout,
  };
}

function taskDetailRuns(detail: TaskDetailView): TaskRun[] {
  return detail.runs ?? detail.task_runs ?? [];
}

async function requestOperatorJSONOrThrow<T>(
  runtime: BrowserRuntime,
  pathname: string,
  init?: RequestInit
): Promise<T> {
  if (!runtime.requestOperatorJSON) {
    throw new Error("operator UDS parity checks require requestOperatorJSON.");
  }
  return await runtime.requestOperatorJSON<T>(pathname, init);
}

function findRun(runs: TaskRun[], runID: string): TaskRun | undefined {
  return runs.find(run => run.id === runID);
}

function cliEnv(paths: { cliShim: string; homeDir: string }): NodeJS.ProcessEnv {
  return {
    ...process.env,
    AGH_HOME: paths.homeDir,
    HOME: paths.homeDir,
    PATH: [path.dirname(paths.cliShim), process.env.PATH ?? ""]
      .filter(Boolean)
      .join(path.delimiter),
  };
}

function uniqueID(prefix: string): string {
  return `${prefix}-${crypto.randomUUID().slice(0, 8)}`;
}

function uniqueTitle(prefix: string): string {
  return `${prefix} ${crypto.randomUUID().slice(0, 8)}`;
}

async function readRouteState(runtime: BrowserRuntime): Promise<Record<string, unknown>> {
  return JSON.parse(
    await readFile(runtime.artifactCollector.artifactPath("browser_route_state"), "utf8")
  ) as Record<string, unknown>;
}

async function revealTasksListPanel(page: import("@playwright/test").Page): Promise<void> {
  const listPanel = page.getByTestId("tasks-list-panel");
  if (await listPanel.isVisible().catch(() => false)) {
    return;
  }

  const backButton = page.getByRole("button", { name: "Back" });
  if (await backButton.isVisible().catch(() => false)) {
    await backButton.click();
  }
}

async function assertNoTaskSensitiveLeak(
  page: import("@playwright/test").Page,
  runtime: BrowserRuntime,
  snapshot: unknown
): Promise<void> {
  await expect(page.locator("body")).not.toContainText(sensitivePattern);
  const payloads = [
    JSON.stringify(snapshot),
    await readFile(runtime.artifactCollector.artifactPath("browser_console"), "utf8"),
    await readFile(runtime.artifactCollector.artifactPath("browser_network"), "utf8"),
    await readFile(runtime.artifactCollector.artifactPath("browser_route_state"), "utf8"),
    await readFile(runtime.artifactCollector.artifactPath("browser_api_snapshots"), "utf8"),
  ];
  for (const payload of payloads) {
    expect(payload).not.toMatch(sensitivePattern);
  }
  if (runtime.paths?.daemonLog) {
    expect(await readFile(runtime.paths.daemonLog, "utf8")).not.toMatch(sensitivePattern);
  }
}
