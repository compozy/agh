import { http, HttpResponse, type HttpHandler } from "msw";

import type {
  CreateTaskRequest,
  TaskInboxItem,
  TaskListItem,
  TaskRecord,
  TaskRun,
  TaskSummary,
  TaskTriageState,
  UpdateTaskRequest,
} from "../types";
import {
  TASK_FIXTURES,
  buildCreatedTaskFixture,
  buildDetailFixture,
  buildTaskRunRecordFixture,
  buildTaskRunDetailFixture,
  buildTaskTreeFixture,
  taskDashboardFixture,
  taskDetailFixture,
  taskInboxFixture,
  taskRunDetailFixture,
  taskTimelineFixture,
  taskTriageStateFixture,
} from "./fixtures";

function resolveTask(id: string): TaskListItem | null {
  return TASK_FIXTURES.find(task => task.id === id) ?? null;
}

function resolveTaskRecord(id: string): TaskRecord | null {
  const task = resolveTask(id);
  return task ? ({ ...task } as TaskRecord) : null;
}

function summaryFromTask(task: TaskListItem): TaskSummary {
  return {
    ...(resolveTaskRecord(task.id) ?? (task as unknown as TaskRecord)),
    active_run: task.active_run ?? null,
    child_count: task.child_count ?? 0,
    dependency_count: task.dependency_count ?? 0,
  } as TaskSummary;
}

function runRecordFromActiveRun(task: TaskListItem): TaskRun | null {
  if (!task.active_run) {
    return null;
  }

  return buildTaskRunRecordFixture({
    id: task.active_run.id,
    task_id: task.active_run.task_id,
    attempt: task.active_run.attempt,
    status: task.active_run.status,
    queued_at: task.active_run.queued_at,
    started_at: task.active_run.started_at,
    ended_at: task.active_run.ended_at,
    claimed_by: task.active_run.claimed_by,
    error: task.active_run.error,
    session_id: task.active_run.session_id,
  });
}

function resolveTaskDetail(id: string) {
  const task = resolveTask(id);
  if (!task) {
    return null;
  }

  if (id === taskDetailFixture.task.id) {
    return taskDetailFixture;
  }

  return buildDetailFixture({
    task: {
      ...(resolveTaskRecord(task.id) ?? (task as unknown as TaskRecord)),
      description: `${task.title} detail for Storybook route coverage.`,
    } as TaskRecord,
    summary: summaryFromTask(task),
    children: [],
    dependency_references: [],
    runs: runRecordFromActiveRun(task) ? [runRecordFromActiveRun(task)!] : [],
  });
}

function resolveTaskRuns(taskId: string): TaskRun[] {
  if (taskId === taskDetailFixture.task.id) {
    return taskDetailFixture.runs ?? [];
  }

  const task = resolveTask(taskId);
  const run = task ? runRecordFromActiveRun(task) : null;
  return run ? [run] : [];
}

function resolveTaskTree(taskId: string) {
  if (taskId === taskDetailFixture.task.id) {
    return buildTaskTreeFixture();
  }

  const task = resolveTask(taskId);
  if (!task) {
    return null;
  }

  return buildTaskTreeFixture({
    root: {
      task: task as unknown as TaskRecord,
      active_run: task.active_run ?? null,
      depth: 0,
      parent_task_id: undefined,
      child_count: 0,
      last_activity_at: task.last_activity_at ?? task.updated_at,
    },
    descendants: [],
  });
}

function resolveTaskRun(runId: string) {
  if (runId === taskRunDetailFixture.run.id) {
    return taskRunDetailFixture;
  }

  const primaryRun = (taskDetailFixture.runs ?? []).find(run => run.id === runId);
  if (primaryRun) {
    return buildTaskRunDetailFixture({
      run: primaryRun,
      task: taskDetailFixture.task,
      summary: taskRunDetailFixture.summary,
      session:
        primaryRun.session_id === undefined
          ? null
          : {
              session_id: primaryRun.session_id,
              created_at: taskRunDetailFixture.session?.created_at ?? "2026-04-17T09:58:00Z",
              updated_at: taskRunDetailFixture.session?.updated_at ?? "2026-04-17T10:01:00Z",
              agent_name: taskRunDetailFixture.session?.agent_name,
              channel: taskRunDetailFixture.session?.channel,
              name: taskRunDetailFixture.session?.name,
              state: taskRunDetailFixture.session?.state,
              workspace_id: taskRunDetailFixture.session?.workspace_id,
            },
    });
  }

  for (const task of TASK_FIXTURES) {
    if (task.active_run?.id === runId) {
      const run = runRecordFromActiveRun(task);
      return buildTaskRunDetailFixture({
        run: run ?? undefined,
        task: task as unknown as TaskRecord,
        session: task.active_run.session_id
          ? {
              ...(taskRunDetailFixture.session ?? {
                created_at: "2026-04-17T09:58:00Z",
                updated_at: "2026-04-17T10:01:00Z",
              }),
              session_id: task.active_run.session_id,
            }
          : null,
      });
    }
  }

  return null;
}

function filterTasks(requestUrl: URL) {
  const scope = requestUrl.searchParams.get("scope");
  const status = requestUrl.searchParams.get("status");
  const ownerRef = requestUrl.searchParams.get("owner_ref");
  const query = requestUrl.searchParams.get("query")?.trim().toLowerCase() ?? "";

  return TASK_FIXTURES.filter(task => {
    if (scope && task.scope !== scope) return false;
    if (status && task.status !== status) return false;
    if (ownerRef && task.owner?.ref !== ownerRef) return false;
    if (query && !`${task.title} ${task.identifier ?? ""}`.toLowerCase().includes(query)) {
      return false;
    }
    return true;
  });
}

function filterInboxItems(items: TaskInboxItem[], requestUrl: URL) {
  const lane = requestUrl.searchParams.get("lane");
  const unread = requestUrl.searchParams.get("unread");
  const query = requestUrl.searchParams.get("query")?.trim().toLowerCase() ?? "";

  return items.filter(item => {
    if (lane && item.lane !== lane) return false;
    if (unread === "true" && item.triage.read) return false;
    if (
      query &&
      !`${item.task.title} ${item.task.identifier ?? ""}`.toLowerCase().includes(query)
    ) {
      return false;
    }
    return true;
  });
}

function buildInboxResponse(requestUrl: URL) {
  const flatItems = (taskInboxFixture.groups ?? []).flatMap(group => group.items ?? []);
  const filteredItems = filterInboxItems(flatItems, requestUrl);

  const grouped = new Map<string, TaskInboxItem[]>();
  for (const item of filteredItems) {
    const existing = grouped.get(item.lane) ?? [];
    existing.push(item);
    grouped.set(item.lane, existing);
  }

  const groups = Array.from(grouped.entries()).map(([lane, items]) => ({
    lane,
    count: items.length,
    unread_count: items.filter(item => !item.triage.read).length,
    items,
  }));

  return {
    ...taskInboxFixture,
    total: filteredItems.length,
    unread_total: filteredItems.filter(item => !item.triage.read).length,
    archived_total: filteredItems.filter(item => item.triage.archived).length,
    groups,
  };
}

function filterRuns(runs: TaskRun[], requestUrl: URL) {
  const status = requestUrl.searchParams.get("status");
  const sessionId = requestUrl.searchParams.get("session_id");

  return runs.filter(run => {
    if (status && run.status !== status) return false;
    if (sessionId && run.session_id !== sessionId) return false;
    return true;
  });
}

function withTriageState(taskId: string, overrides: Partial<TaskTriageState> = {}) {
  return {
    ...taskTriageStateFixture,
    task_id: taskId,
    updated_at: "2026-04-17T10:05:00Z",
    ...overrides,
  } as TaskTriageState;
}

function notFound(entity: string, id: string) {
  return HttpResponse.json({ error: `${entity} not found: ${id}` }, { status: 404 });
}

export const handlers: HttpHandler[] = [
  http.get("/api/tasks", ({ request }) =>
    HttpResponse.json({ tasks: filterTasks(new URL(request.url)) })
  ),
  http.get("/api/tasks/:id", ({ params }) => {
    const id = String(params.id);
    const detail = resolveTaskDetail(id);

    if (!detail) {
      return notFound("Task", id);
    }

    return HttpResponse.json({ task: detail });
  }),
  http.get("/api/tasks/:id/runs", ({ params, request }) => {
    const id = String(params.id);
    if (!resolveTask(id)) {
      return notFound("Task", id);
    }

    return HttpResponse.json({ runs: filterRuns(resolveTaskRuns(id), new URL(request.url)) });
  }),
  http.get("/api/tasks/:id/timeline", ({ params, request }) => {
    const id = String(params.id);
    if (!resolveTask(id)) {
      return notFound("Task", id);
    }

    const limit = Number(new URL(request.url).searchParams.get("limit") ?? "0");
    const timeline =
      Number.isFinite(limit) && limit > 0
        ? taskTimelineFixture.slice(0, limit)
        : taskTimelineFixture;

    return HttpResponse.json({ timeline });
  }),
  http.get("/api/tasks/:id/tree", ({ params }) => {
    const id = String(params.id);
    const tree = resolveTaskTree(id);

    if (!tree) {
      return notFound("Task", id);
    }

    return HttpResponse.json({ tree });
  }),
  http.get("/api/task-runs/:id", ({ params }) => {
    const id = String(params.id);
    const run = resolveTaskRun(id);

    if (!run) {
      return notFound("Task run", id);
    }

    return HttpResponse.json({ run });
  }),
  http.get("/api/observe/tasks/dashboard", () =>
    HttpResponse.json({ dashboard: taskDashboardFixture })
  ),
  http.get("/api/observe/tasks/inbox", ({ request }) =>
    HttpResponse.json({ inbox: buildInboxResponse(new URL(request.url)) })
  ),
  http.post("/api/tasks", async ({ request }) => {
    const body = (await request.json()) as Partial<CreateTaskRequest>;

    return HttpResponse.json({ task: buildCreatedTaskFixture(body) }, { status: 201 });
  }),
  http.patch("/api/tasks/:id", async ({ params, request }) => {
    const id = String(params.id);
    const task = resolveTaskRecord(id);
    if (!task) {
      return notFound("Task", id);
    }

    const body = (await request.json()) as Partial<UpdateTaskRequest>;
    return HttpResponse.json({
      task: {
        ...task,
        title: body.title ?? task.title,
        description: body.description ?? task.description,
        priority: body.priority ?? task.priority,
        owner: body.clear_owner ? null : (body.owner ?? task.owner),
        max_attempts: body.max_attempts ?? task.max_attempts,
        approval_policy:
          body.approval_policy === "none"
            ? undefined
            : (body.approval_policy ?? task.approval_policy),
        network_channel: body.network_channel ?? task.network_channel,
      },
    });
  }),
  http.post("/api/tasks/:id/publish", ({ params }) => {
    const id = String(params.id);
    const task = resolveTaskRecord(id);
    if (!task) {
      return notFound("Task", id);
    }

    return HttpResponse.json({
      task: {
        ...task,
        status: "ready",
      },
    });
  }),
  http.post("/api/tasks/:id/cancel", ({ params }) => {
    const id = String(params.id);
    const task = resolveTaskRecord(id);
    if (!task) {
      return notFound("Task", id);
    }

    return HttpResponse.json({
      task: {
        ...task,
        status: "canceled",
      },
    });
  }),
  http.post("/api/tasks/:id/approve", ({ params }) => {
    const id = String(params.id);
    const task = resolveTaskRecord(id);
    if (!task) {
      return notFound("Task", id);
    }

    return HttpResponse.json({
      task: {
        ...task,
        status: "ready",
        approval_state: "approved",
      },
    });
  }),
  http.post("/api/tasks/:id/reject", ({ params }) => {
    const id = String(params.id);
    const task = resolveTaskRecord(id);
    if (!task) {
      return notFound("Task", id);
    }

    return HttpResponse.json({
      task: {
        ...task,
        status: "blocked",
        approval_state: "rejected",
      },
    });
  }),
  http.post("/api/tasks/:id/runs", ({ params }) => {
    const id = String(params.id);
    const task = resolveTask(id);
    if (!task) {
      return notFound("Task", id);
    }

    return HttpResponse.json(
      {
        run: buildTaskRunRecordFixture({
          id: "run_created",
          task_id: id,
          attempt: 1,
          status: "queued",
          queued_at: "2026-04-17T10:05:00Z",
          started_at: null,
          session_id: undefined,
        }),
      },
      { status: 201 }
    );
  }),
  http.post("/api/tasks/:id/triage/read", ({ params }) =>
    HttpResponse.json({ triage: withTriageState(String(params.id), { read: true }) })
  ),
  http.post("/api/tasks/:id/triage/archive", ({ params }) =>
    HttpResponse.json({
      triage: withTriageState(String(params.id), { archived: true, read: true }),
    })
  ),
  http.post("/api/tasks/:id/triage/dismiss", ({ params }) =>
    HttpResponse.json({
      triage: withTriageState(String(params.id), { dismissed: true, read: true }),
    })
  ),
];
