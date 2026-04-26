import type {
  CreateTaskRequest,
  TaskChildSummary,
  TaskDashboardView,
  TaskDetailView,
  TaskInboxItem,
  TaskInboxView,
  TaskListItem,
  TaskRecord,
  TaskRun,
  TaskRunDetailView,
  TaskSummary,
  TaskTimelineItem,
  TaskTreeNode,
  TaskTreeView,
  TaskTriageState,
} from "../types";

type TaskDependencyReference = NonNullable<TaskDetailView["dependency_references"]>[number];
type TaskActiveRun = NonNullable<TaskListItem["active_run"]>;

const STORYBOOK_WORKSPACE_ID = "ws_storybook";
const STORYBOOK_CHANNEL = "storybook";

export function buildTaskRunFixture(overrides: Partial<TaskActiveRun> = {}): TaskActiveRun {
  return {
    id: "run_001",
    task_id: "task_001",
    attempt: 2,
    max_attempts: 3,
    status: "running",
    queued_at: "2026-04-17T09:58:00Z",
    started_at: "2026-04-17T09:59:00Z",
    session_id: "sess-storybook",
    claimed_by: { kind: "agent_session", ref: "Coder" },
    claim_token_hash: "sha256:storybook-run",
    coordination_channel_id: "coord-task-001",
    ...overrides,
  } as TaskActiveRun;
}

export function buildTaskRunRecordFixture(overrides: Partial<TaskRun> = {}): TaskRun {
  return {
    id: "run_001",
    task_id: "task_001",
    attempt: 2,
    status: "running",
    queued_at: "2026-04-17T09:58:00Z",
    started_at: "2026-04-17T09:59:00Z",
    session_id: "sess-storybook",
    claimed_by: { kind: "agent_session", ref: "Coder" },
    origin: { kind: "cli", ref: "op" },
    claim_token_hash: "sha256:storybook-run",
    coordination_channel_id: "coord-task-001",
    coordination_channel: {
      id: "coord-task-001",
      display_name: "TASK-1 coordination",
      workspace_id: STORYBOOK_WORKSPACE_ID,
      task_id: "task_001",
      run_id: "run_001",
      allowed_message_kinds: [
        "status",
        "request",
        "reply",
        "blocker",
        "handoff",
        "result",
        "review_request",
      ],
    },
    ...overrides,
  } as TaskRun;
}

export function buildTaskFixture(overrides: Partial<TaskListItem> = {}): TaskListItem {
  return {
    id: "task_001",
    identifier: "TASK-1",
    title: "Refactor event mapper",
    status: "in_progress",
    scope: "workspace",
    workspace_id: STORYBOOK_WORKSPACE_ID,
    origin: { kind: "web", ref: "op" },
    created_at: "2026-04-17T09:00:00Z",
    updated_at: "2026-04-17T10:02:00Z",
    last_activity_at: "2026-04-17T10:01:00Z",
    created_by: { kind: "human", ref: "pedro@" },
    owner: { kind: "agent_session", ref: "Coder" },
    priority: "high",
    child_count: 2,
    dependency_count: 1,
    active_run: buildTaskRunFixture(),
    ...overrides,
  } as TaskListItem;
}

export const TASK_FIXTURES: TaskListItem[] = [
  buildTaskFixture(),
  buildTaskFixture({
    id: "task_002",
    identifier: "TASK-2",
    title: "Add NATS retry backoff",
    status: "pending",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
  }),
  buildTaskFixture({
    id: "task_003",
    identifier: "TASK-3",
    title: "Streaming buffer leak",
    status: "failed",
    priority: "urgent",
    active_run: buildTaskRunFixture({
      id: "run_003",
      task_id: "task_003",
      attempt: 3,
      max_attempts: 3,
      status: "failed",
      started_at: "2026-04-17T08:03:00Z",
      ended_at: "2026-04-17T08:14:00Z",
      session_id: "sess-fail",
      error: "rate-limited by upstream",
    }),
  }),
  buildTaskFixture({
    id: "task_004",
    identifier: "TASK-4",
    title: "Bridge health telemetry",
    status: "completed",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
  }),
  buildTaskFixture({
    id: "task_005",
    identifier: "TASK-5",
    title: "Rewrite permission prompt",
    status: "blocked",
    priority: "high",
    active_run: null,
  }),
  buildTaskFixture({
    id: "task_006",
    identifier: "TASK-6",
    title: "Approve remote bridge rollout",
    status: "blocked",
    priority: "high",
    approval_policy: "manual",
    approval_state: "pending",
    active_run: null,
  }),
  buildTaskFixture({
    id: "task_007",
    identifier: "TASK-7",
    title: "Archive stale draft spec",
    status: "ready",
    priority: "low",
    active_run: null,
  }),
];

export function buildTaskRecordFixture(
  task: TaskListItem = TASK_FIXTURES[0]!,
  overrides: Partial<TaskRecord> = {}
): TaskRecord {
  return {
    ...task,
    workspace_id:
      task.scope === "workspace"
        ? ((task as { workspace_id?: string }).workspace_id ?? STORYBOOK_WORKSPACE_ID)
        : undefined,
    description: "Pull CodeRabbit review on PR 341 and post a summary.",
    approval_policy: task.approval_policy,
    approval_state: task.approval_state,
    network_channel: STORYBOOK_CHANNEL,
    max_attempts: task.active_run?.max_attempts ?? 3,
    ...overrides,
  } as TaskRecord;
}

export function buildTaskChildFixture(overrides: Partial<TaskChildSummary> = {}): TaskChildSummary {
  return {
    id: "task_child_001",
    identifier: "TASK-43",
    status: "ready",
    scope: "workspace",
    title: "Write migration",
    priority: "medium",
    owner: { kind: "agent_session", ref: "Coder" },
    last_activity_at: "2026-04-17T09:30:00Z",
    ...overrides,
  } as TaskChildSummary;
}

export function buildTaskDependencyReferenceFixture(
  overrides: Partial<TaskDependencyReference> = {}
): TaskDependencyReference {
  return {
    depends_on: buildTaskRecordFixture(
      buildTaskFixture({
        id: "task_dep_001",
        identifier: "TASK-44",
        title: "Land API schema",
        status: "ready",
        active_run: null,
        child_count: 0,
        dependency_count: 0,
      })
    ),
    ...overrides,
  } as TaskDependencyReference;
}

function buildTaskSummaryFixture(
  listTask: TaskListItem,
  task: TaskRecord,
  overrides: Partial<TaskSummary> = {}
): TaskSummary {
  return {
    ...task,
    active_run: listTask.active_run ?? null,
    child_count: listTask.child_count ?? 0,
    dependency_count: listTask.dependency_count ?? 0,
    ...overrides,
  } as TaskSummary;
}

export function buildDetailFixture(overrides: Partial<TaskDetailView> = {}): TaskDetailView {
  const listTask = TASK_FIXTURES[0]!;
  const task = buildTaskRecordFixture(listTask);
  const runs = [
    buildTaskRunRecordFixture(),
    buildTaskRunRecordFixture({
      id: "run_000",
      task_id: task.id,
      attempt: 1,
      status: "completed",
      queued_at: "2026-04-17T09:00:00Z",
      started_at: "2026-04-17T09:01:00Z",
      ended_at: "2026-04-17T09:05:00Z",
      session_id: "sess-prev",
    }),
  ];
  const detail: TaskDetailView = {
    task,
    summary: buildTaskSummaryFixture(listTask, task),
    children: [
      buildTaskChildFixture(),
      buildTaskChildFixture({
        id: "task_child_002",
        identifier: "TASK-45",
        title: "Verify CLI parity",
        status: "in_progress",
        priority: "high",
      }),
    ],
    dependency_references: [buildTaskDependencyReferenceFixture()],
    runs,
  } as TaskDetailView;

  return {
    ...detail,
    ...overrides,
    task: {
      ...detail.task,
      ...overrides.task,
    },
    summary: {
      ...detail.summary,
      ...overrides.summary,
    },
    children: overrides.children ?? detail.children,
    dependency_references: overrides.dependency_references ?? detail.dependency_references,
    runs: overrides.runs ?? detail.runs,
  } as TaskDetailView;
}

export function buildDashboardFixture(
  overrides: Partial<TaskDashboardView> = {}
): TaskDashboardView {
  return {
    active_runs: {
      claimed: 0,
      queued: 1,
      running: 1,
      starting: 0,
      total: 2,
      items: [
        {
          age_ms: 45_000,
          attempt: 2,
          health_status: "ok",
          last_activity_at: "2026-04-17T10:00:00Z",
          max_attempts: 3,
          run_id: "run_001",
          run_status: "running",
          scope: "workspace",
          stuck: false,
          task_id: "task_001",
          task_identifier: "TASK-1",
          task_status: "in_progress",
          task_title: "Refactor event mapper",
        },
        {
          age_ms: 180_000,
          attempt: 1,
          health_status: "warning",
          last_activity_at: "2026-04-17T09:57:00Z",
          max_attempts: 3,
          run_id: "run_006",
          run_status: "queued",
          scope: "workspace",
          stuck: false,
          task_id: "task_006",
          task_identifier: "TASK-6",
          task_status: "blocked",
          task_title: "Approve remote bridge rollout",
        },
      ],
    },
    cards: {
      blocked: {
        awaiting_approval: 1,
        awaiting_dependencies: 1,
        health_status: "warning",
        tasks: 2,
      },
      failed: {
        failed_runs: 1,
        forced_stops: 0,
        health_status: "warning",
        tasks: 1,
      },
      in_progress: {
        active_runs: 2,
        claimed_runs: 0,
        queued_runs: 1,
        running_runs: 1,
        starting_runs: 0,
        tasks: 2,
        health_status: "ok",
      },
      latency: {
        claim_latency_ms: { average_ms: 1_000, maximum_ms: 1_800, samples: 4 },
        start_latency_ms: { average_ms: 500, maximum_ms: 1_200, samples: 4 },
      },
    },
    freshness: {
      age_ms: 500,
      has_live_work: true,
      latest_activity_at: "2026-04-17T10:00:00Z",
      observed_at: "2026-04-17T10:00:01Z",
      stale: false,
      stale_after_ms: 60_000,
      status: "fresh",
    },
    health: {
      active_orphan_runs: 0,
      queue_backlog: false,
      status: "ok",
      stuck_runs: 0,
    },
    queue: {
      backlog_status: "ok",
      backlog_threshold_ms: 60_000,
      backlog_warning: false,
      oldest_queue_age_ms: 14_000,
      oldest_queued_at: "2026-04-17T09:59:45Z",
      total: 1,
    },
    status_breakdown: [
      { count: 2, share_percent: 29, status: "completed" },
      { count: 1, share_percent: 14, status: "failed" },
      { count: 1, share_percent: 14, status: "blocked" },
      { count: 2, share_percent: 29, status: "in_progress" },
      { count: 1, share_percent: 14, status: "pending" },
    ],
    totals: {
      active_runs: 2,
      awaiting_approval_tasks: 1,
      blocked_tasks: 2,
      canceled_runs: 0,
      canceled_tasks: 0,
      claimed_runs: 0,
      completed_runs: 4,
      completed_tasks: 2,
      dependency_blocked_tasks: 1,
      draft_tasks: 0,
      failed_runs: 1,
      failed_tasks: 1,
      in_progress_tasks: 2,
      pending_tasks: 1,
      queued_runs: 1,
      ready_tasks: 1,
      running_runs: 1,
      runs_total: 7,
      starting_runs: 0,
      tasks_total: TASK_FIXTURES.length,
    },
    ...overrides,
  } as TaskDashboardView;
}

export function buildInboxItemFixture(overrides: Partial<TaskInboxItem> = {}): TaskInboxItem {
  return {
    lane: "my_work",
    latest_activity_at: "2026-04-17T10:00:00Z",
    task: buildTaskRecordFixture(
      buildTaskFixture({
        id: "task_inbox_001",
        identifier: "TASK-8",
        title: "Inbox item",
        status: "ready",
        priority: "medium",
        active_run: null,
      })
    ),
    triage: {
      actor: { kind: "human", ref: "op" },
      archived: false,
      dismissed: false,
      read: false,
      task_id: "task_inbox_001",
      updated_at: "2026-04-17T10:00:00Z",
    },
    ...overrides,
  } as TaskInboxItem;
}

function buildPopulatedInboxGroups(): NonNullable<TaskInboxView["groups"]> {
  return [
    {
      lane: "approvals",
      count: 1,
      unread_count: 1,
      items: [
        buildInboxItemFixture({
          lane: "approvals",
          approval_policy: "manual",
          approval_state: "pending",
          task: buildTaskRecordFixture(TASK_FIXTURES[5]!),
          triage: {
            actor: { kind: "human", ref: "op" },
            archived: false,
            dismissed: false,
            read: false,
            task_id: "task_006",
            updated_at: "2026-04-17T10:00:00Z",
          },
        }),
      ],
    },
    {
      lane: "failed_runs",
      count: 1,
      unread_count: 1,
      items: [
        buildInboxItemFixture({
          lane: "failed_runs",
          run: buildTaskRunFixture({
            id: "run_003",
            task_id: "task_003",
            attempt: 3,
            max_attempts: 3,
            status: "failed",
            ended_at: "2026-04-17T08:14:00Z",
            error: "rate-limited by upstream",
          }),
          task: buildTaskRecordFixture(TASK_FIXTURES[2]!),
          triage: {
            actor: { kind: "agent_session", ref: "Coder" },
            archived: false,
            dismissed: false,
            read: false,
            task_id: "task_003",
            updated_at: "2026-04-17T08:14:00Z",
          },
        }),
      ],
    },
    {
      lane: "my_work",
      count: 1,
      unread_count: 1,
      items: [
        buildInboxItemFixture({
          lane: "my_work",
          task: buildTaskRecordFixture(TASK_FIXTURES[0]!),
          triage: {
            actor: { kind: "agent_session", ref: "Coder" },
            archived: false,
            dismissed: false,
            read: false,
            task_id: "task_001",
            updated_at: "2026-04-17T10:01:00Z",
          },
        }),
      ],
    },
    {
      lane: "archived",
      count: 1,
      unread_count: 0,
      items: [
        buildInboxItemFixture({
          lane: "archived",
          task: buildTaskRecordFixture(TASK_FIXTURES[6]!),
          triage: {
            actor: { kind: "human", ref: "pedro@" },
            archived: true,
            dismissed: false,
            read: true,
            task_id: "task_007",
            updated_at: "2026-04-17T07:55:00Z",
          },
        }),
      ],
    },
  ];
}

export function buildInboxFixture(overrides: Partial<TaskInboxView> = {}): TaskInboxView {
  return {
    archived_total: 0,
    total: 0,
    unread_total: 0,
    groups: [],
    ...overrides,
  } as TaskInboxView;
}

export function buildTaskTimelineItemFixture(
  overrides: Partial<TaskTimelineItem> = {}
): TaskTimelineItem {
  return {
    event_id: "evt_001",
    sequence: 101,
    timestamp: "2026-04-17T10:00:30Z",
    event_type: "task.run_progress",
    origin: { kind: "agent_session", ref: "Coder" },
    task: buildTaskRecordFixture(TASK_FIXTURES[0]!),
    run: buildTaskRunFixture(),
    payload: { message: "Applying the new mapper across the event reducers." },
    ...overrides,
  } as TaskTimelineItem;
}

export const taskTimelineFixture: TaskTimelineItem[] = [
  buildTaskTimelineItemFixture({
    event_id: "evt_000",
    sequence: 100,
    timestamp: "2026-04-17T09:58:00Z",
    event_type: "task.run_started",
    payload: { message: "Run started for TASK-1." },
  }),
  buildTaskTimelineItemFixture(),
  buildTaskTimelineItemFixture({
    event_id: "evt_002",
    sequence: 102,
    timestamp: "2026-04-17T10:01:00Z",
    event_type: "task.dependency_added",
    run: null,
    payload: { message: "Linked TASK-44 as a dependency." },
  }),
  buildTaskTimelineItemFixture({
    event_id: "evt_003",
    sequence: 103,
    timestamp: "2026-04-17T10:02:00Z",
    event_type: "task.run_failed",
    run: buildTaskRunFixture({
      id: "run_004",
      task_id: "task_001",
      attempt: 3,
      status: "failed",
      ended_at: "2026-04-17T10:02:00Z",
      error: "temporary provider outage",
    }),
    payload: { message: "Run failed after provider outage." },
  }),
];

export function buildTaskTreeNodeFixture(overrides: Partial<TaskTreeNode> = {}): TaskTreeNode {
  return {
    task: buildTaskRecordFixture(TASK_FIXTURES[0]!) as unknown as TaskTreeNode["task"],
    active_run: buildTaskRunFixture(),
    depth: 0,
    parent_task_id: undefined,
    child_count: 2,
    last_activity_at: "2026-04-17T10:01:00Z",
    ...overrides,
  } as TaskTreeNode;
}

export function buildTaskTreeFixture(overrides: Partial<TaskTreeView> = {}): TaskTreeView {
  return {
    root: buildTaskTreeNodeFixture(),
    descendants: [
      buildTaskTreeNodeFixture({
        task: buildTaskRecordFixture(
          buildTaskFixture({
            id: "task_child_001",
            identifier: "TASK-43",
            title: "Write migration",
            status: "in_progress",
            active_run: buildTaskRunFixture({
              id: "run_child_001",
              task_id: "task_child_001",
              attempt: 1,
              max_attempts: 2,
              status: "running",
              session_id: "sess-child-1",
            }),
            child_count: 0,
            dependency_count: 0,
          })
        ) as unknown as TaskTreeNode["task"],
        active_run: buildTaskRunFixture({
          id: "run_child_001",
          task_id: "task_child_001",
          attempt: 1,
          max_attempts: 2,
          status: "running",
          session_id: "sess-child-1",
        }),
        depth: 1,
        parent_task_id: "task_001",
        child_count: 0,
      }),
      buildTaskTreeNodeFixture({
        task: buildTaskRecordFixture(
          buildTaskFixture({
            id: "task_child_002",
            identifier: "TASK-45",
            title: "Verify CLI parity",
            status: "pending",
            active_run: buildTaskRunFixture({
              id: "run_child_002",
              task_id: "task_child_002",
              attempt: 1,
              max_attempts: 1,
              status: "queued",
              session_id: undefined,
            }),
            child_count: 0,
            dependency_count: 0,
          })
        ) as unknown as TaskTreeNode["task"],
        active_run: buildTaskRunFixture({
          id: "run_child_002",
          task_id: "task_child_002",
          attempt: 1,
          max_attempts: 1,
          status: "queued",
          session_id: undefined,
        }),
        depth: 1,
        parent_task_id: "task_001",
        child_count: 0,
      }),
    ],
    ...overrides,
  } as TaskTreeView;
}

export function buildTaskRunDetailFixture(
  overrides: Partial<TaskRunDetailView> = {}
): TaskRunDetailView {
  const session =
    overrides.session === null
      ? null
      : ({
          session_id: "sess-storybook",
          created_at: "2026-04-17T09:58:00Z",
          updated_at: "2026-04-17T10:01:00Z",
          agent_name: "Coder",
          workspace_id: STORYBOOK_WORKSPACE_ID,
          state: "active",
          ...overrides.session,
        } as TaskRunDetailView["session"]);

  return {
    run: buildTaskRunRecordFixture({
      id: "run_001",
      task_id: "task_001",
      attempt: 2,
      status: "running",
      queued_at: "2026-04-17T09:58:00Z",
      started_at: "2026-04-17T09:59:00Z",
      origin: { kind: "cli", ref: "op" },
      session_id: "sess-storybook",
      idempotency_key: "storybook-run",
      claimed_by: { kind: "agent_session", ref: "Coder" },
      ...overrides.run,
    }),
    task: buildTaskRecordFixture(
      TASK_FIXTURES[0]!,
      overrides.task
    ) as unknown as TaskRunDetailView["task"],
    summary: {
      last_activity_at: "2026-04-17T10:01:00Z",
      last_event_type: "task.run_progress",
      tool_call_count: 4,
      input_tokens: 14_281,
      output_tokens: 3_046,
      total_tokens: 17_327,
      turn_count: 6,
      total_cost: 0.18,
      cost_currency: "USD",
      ...overrides.summary,
    },
    session,
  } as TaskRunDetailView;
}

export const taskDashboardFixture = buildDashboardFixture();
export const taskInboxFixture = buildInboxFixture({
  archived_total: 1,
  total: 4,
  unread_total: 3,
  groups: buildPopulatedInboxGroups(),
});
export const taskDetailFixture = buildDetailFixture();
export const taskTreeFixture = buildTaskTreeFixture();
export const taskRunDetailFixture = buildTaskRunDetailFixture();

export const taskTriageStateFixture: TaskTriageState = {
  actor: { kind: "human", ref: "op" },
  archived: false,
  dismissed: false,
  read: true,
  task_id: "task_001",
  updated_at: "2026-04-17T10:01:00Z",
} as TaskTriageState;

export function buildCreatedTaskFixture(body?: Partial<CreateTaskRequest>): TaskRecord {
  return buildTaskRecordFixture(
    buildTaskFixture({
      id: "task_created",
      identifier: "TASK-NEW",
      title: body?.title?.trim() || "Created Storybook task",
      status: body?.draft ? "draft" : "ready",
      scope: body?.scope ?? "workspace",
      active_run: null,
      child_count: 0,
      dependency_count: 0,
      priority: body?.priority ?? "medium",
      approval_policy: body?.approval_policy,
      network_channel: body?.network_channel ?? STORYBOOK_CHANNEL,
    }),
    {
      description: body?.description ?? "",
    }
  );
}
