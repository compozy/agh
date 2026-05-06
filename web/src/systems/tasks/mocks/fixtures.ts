import type {
  AgentContextView,
  CreateTaskRequest,
  TaskBridgeNotificationCursor,
  TaskBridgeNotificationSubscription,
  TaskChildSummary,
  TaskContextBundle,
  TaskDashboardView,
  TaskDetailView,
  TaskExecutionProfile,
  TaskInboxItem,
  TaskInboxView,
  TaskListItem,
  TaskRecord,
  TaskRun,
  TaskRunDetailView,
  TaskRunReview,
  TaskRunReviewVerdictResult,
  TaskSummary,
  TaskTimelineItem,
  TaskTreeNode,
  TaskTreeView,
  TaskTriageState,
} from "../types";
import {
  storyAgentNames,
  storyCoordinatorAgentName,
  storyDefaultWorkspaceId,
  storyHeroNetworkChannel,
  storyPeople,
  storySessionIds,
} from "@/storybook/fintech-scenario";

type TaskDependencyReference = NonNullable<TaskDetailView["dependency_references"]>[number];
type TaskActiveRun = NonNullable<TaskListItem["active_run"]>;
type TaskDashboardActiveRuns = TaskDashboardView["active_runs"];
type TaskDashboardActiveRun = NonNullable<TaskDashboardActiveRuns["items"]>[number];
type TaskDashboardFixtureOverrides = Omit<Partial<TaskDashboardView>, "active_runs"> & {
  active_runs?: Omit<Partial<TaskDashboardActiveRuns>, "items"> & {
    items?: Partial<TaskDashboardActiveRun>[];
  };
};
type TaskInboxItemFixtureOverrides = Omit<Partial<TaskInboxItem>, "task"> & {
  task?: Partial<TaskInboxItem["task"]>;
};

const STORYBOOK_WORKSPACE_ID = storyDefaultWorkspaceId;
const STORYBOOK_CHANNEL = storyHeroNetworkChannel;

export function buildTaskRunFixture(overrides: Partial<TaskActiveRun> = {}): TaskActiveRun {
  return {
    id: "run_001",
    task_id: "task_001",
    attempt: 2,
    max_attempts: 3,
    status: "running",
    queued_at: "2026-04-17T09:58:00Z",
    started_at: "2026-04-17T09:59:00Z",
    session_id: storySessionIds.product,
    claimed_by: { kind: "agent_session", ref: storyAgentNames.product },
    claim_token_hash: "sha256:launch-command-run",
    coordination_channel_id: "coord-launch-001",
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
    session_id: storySessionIds.product,
    claimed_by: { kind: "agent_session", ref: storyAgentNames.product },
    origin: { kind: "cli", ref: storyPeople.primaryOperator },
    claim_token_hash: "sha256:launch-command-run",
    coordination_channel_id: "coord-launch-001",
    coordination_channel: {
      id: "coord-launch-001",
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
    title: "Lock launch blockers for the 18:30 UTC cutover",
    status: "in_progress",
    scope: "workspace",
    workspace_id: STORYBOOK_WORKSPACE_ID,
    latest_event_seq: 1,
    origin: { kind: "web", ref: storyPeople.primaryOperator },
    created_at: "2026-04-17T09:00:00Z",
    updated_at: "2026-04-17T18:02:00Z",
    last_activity_at: "2026-04-17T18:01:00Z",
    created_by: { kind: "human", ref: storyPeople.primaryOperator },
    owner: { kind: "agent_session", ref: storyAgentNames.product },
    priority: "high",
    child_count: 3,
    dependency_count: 2,
    active_run: buildTaskRunFixture(),
    ...overrides,
  } as TaskListItem;
}

export const TASK_FIXTURES: TaskListItem[] = [
  buildTaskFixture(),
  buildTaskFixture({
    id: "task_002",
    identifier: "TASK-2",
    title: "Validate landing-page hero on mobile breakpoints",
    status: "in_progress",
    priority: "urgent",
    active_run: buildTaskRunFixture({
      id: "run_002",
      task_id: "task_002",
      attempt: 1,
      max_attempts: 2,
      status: "running",
      queued_at: "2026-04-17T17:40:00Z",
      started_at: "2026-04-17T17:42:00Z",
      session_id: storySessionIds.frontend,
      claimed_by: { kind: "agent_session", ref: storyAgentNames.frontend },
      claim_token_hash: "sha256:hero-qa-run",
      coordination_channel_id: "coord-launch-002",
    }),
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.frontend },
  }),
  buildTaskFixture({
    id: "task_003",
    identifier: "TASK-3",
    title: "Prepare final pricing claims for ads and site",
    status: "pending",
    priority: "high",
    active_run: null,
    child_count: 1,
    dependency_count: 1,
    owner: { kind: "agent_session", ref: storyAgentNames.copywriter },
  }),
  buildTaskFixture({
    id: "task_004",
    identifier: "TASK-4",
    title: "Reconcile BR settlement replay ETA with the partner bank",
    status: "failed",
    priority: "urgent",
    owner: { kind: "agent_session", ref: storyAgentNames.platform },
    active_run: buildTaskRunFixture({
      id: "run_004",
      task_id: "task_004",
      attempt: 3,
      max_attempts: 3,
      status: "failed",
      started_at: "2026-04-17T17:03:00Z",
      ended_at: "2026-04-17T17:14:00Z",
      session_id: storySessionIds.platform,
      claimed_by: { kind: "agent_session", ref: storyAgentNames.platform },
      error: "partner-bank replay detail was stale and blocked the public timeout copy",
    }),
  }),
  buildTaskFixture({
    id: "task_005",
    identifier: "TASK-5",
    title: "Publish launch CRM batch after release control clears 25%",
    status: "blocked",
    priority: "high",
    active_run: null,
    child_count: 0,
    dependency_count: 1,
    owner: { kind: "agent_session", ref: storyAgentNames.marketing },
  }),
  buildTaskFixture({
    id: "task_006",
    identifier: "TASK-6",
    title: "Approve public timeout copy for BR merchants",
    status: "blocked",
    priority: "urgent",
    approval_policy: "manual",
    approval_state: "pending",
    active_run: null,
    owner: { kind: "human", ref: storyPeople.productLead },
  }),
  buildTaskFixture({
    id: "task_007",
    identifier: "TASK-7",
    title: "Update support macros for launch-day pricing questions",
    status: "ready",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.support },
  }),
  buildTaskFixture({
    id: "task_008",
    identifier: "TASK-8",
    title: "Produce executive GMV and burn snapshot for go-live",
    status: "ready",
    priority: "high",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.cfo },
  }),
  buildTaskFixture({
    id: "task_009",
    identifier: "TASK-9",
    title: "Canary 25% rollout for checkout release",
    status: "completed",
    priority: "high",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.release },
  }),
  buildTaskFixture({
    id: "task_010",
    identifier: "TASK-10",
    title: "Review MX cashback wording on the landing page",
    status: "pending",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 1,
    owner: { kind: "agent_session", ref: storyAgentNames.copywriter },
  }),
  buildTaskFixture({
    id: "task_011",
    identifier: "TASK-11",
    title: "Confirm VIP merchant callback template for payout delays",
    status: "blocked",
    priority: "high",
    active_run: null,
    child_count: 0,
    dependency_count: 1,
    owner: { kind: "agent_session", ref: storyAgentNames.support },
  }),
  buildTaskFixture({
    id: "task_012",
    identifier: "TASK-12",
    title: "Audit reserve exposure after the pilot merchant batch",
    status: "ready",
    priority: "high",
    active_run: null,
    child_count: 1,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.fraud },
  }),
  buildTaskFixture({
    id: "task_013",
    identifier: "TASK-13",
    title: "Refresh launch FAQ knowledge base",
    status: "completed",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.marketing },
  }),
  buildTaskFixture({
    id: "task_014",
    identifier: "TASK-14",
    title: "Verify webhook replay backlog at the partner bank",
    status: "in_progress",
    priority: "high",
    active_run: buildTaskRunFixture({
      id: "run_014",
      task_id: "task_014",
      attempt: 1,
      max_attempts: 2,
      status: "queued",
      queued_at: "2026-04-17T17:44:00Z",
      started_at: null,
      session_id: undefined,
      claimed_by: undefined,
      claim_token_hash: "sha256:partner-backlog-run",
      coordination_channel_id: "coord-launch-014",
    }),
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.platform },
  }),
  buildTaskFixture({
    id: "task_015",
    identifier: "TASK-15",
    title: "Draft launch-room recap for board observers",
    status: "ready",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.cto },
  }),
  buildTaskFixture({
    id: "task_016",
    identifier: "TASK-16",
    title: "Sign off launch-banner claims compliance",
    status: "completed",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.compliance },
  }),
  buildTaskFixture({
    id: "task_017",
    identifier: "TASK-17",
    title: "Compare onboarding funnel metrics against paid spend ramp",
    status: "pending",
    priority: "medium",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.marketing },
  }),
  buildTaskFixture({
    id: "task_018",
    identifier: "TASK-18",
    title: "Finalize launch-room owner matrix and escalation routing",
    status: "ready",
    priority: "low",
    active_run: null,
    child_count: 0,
    dependency_count: 0,
    owner: { kind: "agent_session", ref: storyAgentNames.product },
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
    description:
      "Coordinate the launch-week decision, capture the blocker context, and leave a crisp operator-ready outcome.",
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
    title: "Lock fallback banner copy",
    priority: "medium",
    owner: { kind: "agent_session", ref: storyAgentNames.copywriter },
    latest_event_seq: 1,
    last_activity_at: "2026-04-17T17:30:00Z",
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
        title: "Finalize compliance sign-off on public timeout copy",
        status: "ready",
        active_run: null,
        child_count: 0,
        dependency_count: 0,
        owner: { kind: "agent_session", ref: storyAgentNames.compliance },
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
      ended_at: "2026-04-17T09:08:00Z",
      session_id: storySessionIds.product,
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
        title: "Verify partner settlement replay closeout",
        status: "in_progress",
        priority: "high",
        owner: { kind: "agent_session", ref: storyAgentNames.platform },
      }),
      buildTaskChildFixture({
        id: "task_child_003",
        identifier: "TASK-46",
        title: "Sync support fallback ownership",
        status: "pending",
        priority: "medium",
        owner: { kind: "agent_session", ref: storyAgentNames.support },
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
  overrides: TaskDashboardFixtureOverrides = {}
): TaskDashboardView {
  const { active_runs: activeRunsOverrides, ...viewOverrides } = overrides;
  const activeRuns: TaskDashboardActiveRuns = {
    claimed: activeRunsOverrides?.claimed ?? 0,
    queued: activeRunsOverrides?.queued ?? 1,
    running: activeRunsOverrides?.running ?? 2,
    starting: activeRunsOverrides?.starting ?? 0,
    total: activeRunsOverrides?.total ?? 3,
    items: (activeRunsOverrides?.items ?? defaultTaskDashboardActiveRuns()).map(
      (item, index) =>
        ({
          latest_event_seq: index + 1,
          ...item,
        }) as TaskDashboardActiveRun
    ),
  };
  return {
    active_runs: activeRuns,
    cards: {
      blocked: {
        awaiting_approval: 1,
        awaiting_dependencies: 2,
        health_status: "warning",
        tasks: 3,
      },
      failed: {
        failed_runs: 1,
        forced_stops: 0,
        health_status: "warning",
        tasks: 1,
      },
      in_progress: {
        active_runs: 3,
        claimed_runs: 0,
        queued_runs: 1,
        running_runs: 2,
        starting_runs: 0,
        tasks: 3,
        health_status: "ok",
      },
      latency: {
        claim_latency_ms: { average_ms: 920, maximum_ms: 1_600, samples: 7 },
        start_latency_ms: { average_ms: 460, maximum_ms: 980, samples: 7 },
      },
    },
    freshness: {
      age_ms: 500,
      has_live_work: true,
      latest_activity_at: "2026-04-17T18:00:00Z",
      observed_at: "2026-04-17T18:00:01Z",
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
      oldest_queue_age_ms: 42_000,
      oldest_queued_at: "2026-04-17T17:57:45Z",
      total: 1,
    },
    status_breakdown: [
      { count: 3, share_percent: 17, status: "completed" },
      { count: 1, share_percent: 6, status: "failed" },
      { count: 3, share_percent: 17, status: "blocked" },
      { count: 3, share_percent: 17, status: "in_progress" },
      { count: 3, share_percent: 17, status: "pending" },
      { count: 5, share_percent: 28, status: "ready" },
    ],
    totals: {
      active_runs: 3,
      awaiting_approval_tasks: 1,
      blocked_tasks: 3,
      canceled_runs: 0,
      canceled_tasks: 0,
      claimed_runs: 0,
      completed_runs: 5,
      completed_tasks: 3,
      dependency_blocked_tasks: 2,
      draft_tasks: 0,
      failed_runs: 1,
      failed_tasks: 1,
      in_progress_tasks: 3,
      pending_tasks: 3,
      queued_runs: 1,
      ready_tasks: 5,
      running_runs: 2,
      runs_total: 9,
      starting_runs: 0,
      tasks_total: TASK_FIXTURES.length,
    },
    ...viewOverrides,
  } as TaskDashboardView;
}

function defaultTaskDashboardActiveRuns(): TaskDashboardActiveRun[] {
  return [
    {
      age_ms: 45_000,
      attempt: 2,
      health_status: "ok",
      last_activity_at: "2026-04-17T18:00:00Z",
      latest_event_seq: 1,
      max_attempts: 3,
      run_id: "run_001",
      run_status: "running",
      scope: "workspace",
      stuck: false,
      task_id: "task_001",
      task_identifier: "TASK-1",
      task_status: "in_progress",
      task_title: "Lock launch blockers for the 18:30 UTC cutover",
    },
    {
      age_ms: 67_000,
      attempt: 1,
      health_status: "ok",
      last_activity_at: "2026-04-17T17:58:00Z",
      latest_event_seq: 2,
      max_attempts: 2,
      run_id: "run_002",
      run_status: "running",
      scope: "workspace",
      stuck: false,
      task_id: "task_002",
      task_identifier: "TASK-2",
      task_status: "in_progress",
      task_title: "Validate landing-page hero on mobile breakpoints",
    },
    {
      age_ms: 180_000,
      attempt: 1,
      health_status: "warning",
      last_activity_at: "2026-04-17T17:57:00Z",
      latest_event_seq: 3,
      max_attempts: 2,
      run_id: "run_014",
      run_status: "queued",
      scope: "workspace",
      stuck: false,
      task_id: "task_014",
      task_identifier: "TASK-14",
      task_status: "in_progress",
      task_title: "Verify webhook replay backlog at the partner bank",
    },
  ];
}

export function buildInboxItemFixture(
  overrides: TaskInboxItemFixtureOverrides = {}
): TaskInboxItem {
  const { task, ...itemOverrides } = overrides;
  const defaultTask = buildTaskRecordFixture(
    buildTaskFixture({
      id: "task_inbox_001",
      identifier: "TASK-8",
      title: "Inbox item",
      status: "ready",
      priority: "medium",
      active_run: null,
    })
  );
  return {
    lane: "my_work",
    latest_activity_at: "2026-04-17T18:00:00Z",
    triage: {
      actor: { kind: "human", ref: storyPeople.primaryOperator },
      archived: false,
      dismissed: false,
      read: false,
      task_id: "task_inbox_001",
      updated_at: "2026-04-17T18:00:00Z",
    },
    ...itemOverrides,
    task: task === undefined ? defaultTask : buildTaskRecordFixture(undefined, task),
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
            actor: { kind: "human", ref: storyPeople.primaryOperator },
            archived: false,
            dismissed: false,
            read: false,
            task_id: "task_006",
            updated_at: "2026-04-17T18:00:00Z",
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
            id: "run_004",
            task_id: "task_004",
            attempt: 3,
            max_attempts: 3,
            status: "failed",
            ended_at: "2026-04-17T17:14:00Z",
            session_id: storySessionIds.platform,
            claimed_by: { kind: "agent_session", ref: storyAgentNames.platform },
            error: "partner-bank replay detail was stale and blocked the public timeout copy",
          }),
          task: buildTaskRecordFixture(TASK_FIXTURES[3]!),
          triage: {
            actor: { kind: "agent_session", ref: storyAgentNames.platform },
            archived: false,
            dismissed: false,
            read: false,
            task_id: "task_004",
            updated_at: "2026-04-17T17:14:00Z",
          },
        }),
      ],
    },
    {
      lane: "my_work",
      count: 2,
      unread_count: 2,
      items: [
        buildInboxItemFixture({
          lane: "my_work",
          task: buildTaskRecordFixture(TASK_FIXTURES[0]!),
          triage: {
            actor: { kind: "agent_session", ref: storyAgentNames.product },
            archived: false,
            dismissed: false,
            read: false,
            task_id: "task_001",
            updated_at: "2026-04-17T18:01:00Z",
          },
        }),
        buildInboxItemFixture({
          lane: "my_work",
          task: buildTaskRecordFixture(TASK_FIXTURES[1]!),
          triage: {
            actor: { kind: "agent_session", ref: storyAgentNames.frontend },
            archived: false,
            dismissed: false,
            read: false,
            task_id: "task_002",
            updated_at: "2026-04-17T17:59:00Z",
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
          task: buildTaskRecordFixture(TASK_FIXTURES[12]!),
          triage: {
            actor: { kind: "human", ref: storyPeople.primaryOperator },
            archived: true,
            dismissed: false,
            read: true,
            task_id: "task_013",
            updated_at: "2026-04-17T17:25:00Z",
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
    timestamp: "2026-04-17T18:00:30Z",
    event_type: "task.run_progress",
    origin: { kind: "agent_session", ref: storyAgentNames.product },
    task: buildTaskRecordFixture(TASK_FIXTURES[0]!),
    run: buildTaskRunFixture(),
    payload: {
      message: "Tracking the last blockers across copy, canary status, and partner-bank replay.",
    },
    ...overrides,
  } as TaskTimelineItem;
}

export const taskTimelineFixture: TaskTimelineItem[] = [
  buildTaskTimelineItemFixture({
    event_id: "evt_000",
    sequence: 100,
    timestamp: "2026-04-17T17:58:00Z",
    event_type: "task.run_started",
    payload: { message: "Run started for the launch command checkpoint." },
  }),
  buildTaskTimelineItemFixture(),
  buildTaskTimelineItemFixture({
    event_id: "evt_002",
    sequence: 102,
    timestamp: "2026-04-17T18:01:00Z",
    event_type: "task.dependency_added",
    run: null,
    payload: { message: "Linked compliance sign-off on public timeout copy as a dependency." },
  }),
  buildTaskTimelineItemFixture({
    event_id: "evt_003",
    sequence: 103,
    timestamp: "2026-04-17T18:02:00Z",
    event_type: "task.run_failed",
    run: buildTaskRunFixture({
      id: "run_004",
      task_id: "task_001",
      attempt: 3,
      status: "failed",
      ended_at: "2026-04-17T18:02:00Z",
      error: "partner-bank replay detail was stale during the checkpoint",
    }),
    payload: { message: "Checkpoint failed after partner-bank replay detail went stale." },
  }),
];

export function buildTaskTreeNodeFixture(overrides: Partial<TaskTreeNode> = {}): TaskTreeNode {
  return {
    task: buildTaskRecordFixture(TASK_FIXTURES[0]!) as unknown as TaskTreeNode["task"],
    active_run: buildTaskRunFixture(),
    depth: 0,
    parent_task_id: undefined,
    child_count: 3,
    last_activity_at: "2026-04-17T18:01:00Z",
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
            title: "Lock fallback banner copy",
            status: "in_progress",
            active_run: buildTaskRunFixture({
              id: "run_child_001",
              task_id: "task_child_001",
              attempt: 1,
              max_attempts: 2,
              status: "running",
              session_id: storySessionIds.copywriter,
              claimed_by: { kind: "agent_session", ref: storyAgentNames.copywriter },
            }),
            child_count: 0,
            dependency_count: 0,
            owner: { kind: "agent_session", ref: storyAgentNames.copywriter },
          })
        ) as unknown as TaskTreeNode["task"],
        active_run: buildTaskRunFixture({
          id: "run_child_001",
          task_id: "task_child_001",
          attempt: 1,
          max_attempts: 2,
          status: "running",
          session_id: storySessionIds.copywriter,
          claimed_by: { kind: "agent_session", ref: storyAgentNames.copywriter },
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
            title: "Verify partner settlement replay closeout",
            status: "pending",
            active_run: buildTaskRunFixture({
              id: "run_child_002",
              task_id: "task_child_002",
              attempt: 1,
              max_attempts: 1,
              status: "queued",
              session_id: undefined,
              claimed_by: undefined,
            }),
            child_count: 0,
            dependency_count: 0,
            owner: { kind: "agent_session", ref: storyAgentNames.platform },
          })
        ) as unknown as TaskTreeNode["task"],
        active_run: buildTaskRunFixture({
          id: "run_child_002",
          task_id: "task_child_002",
          attempt: 1,
          max_attempts: 1,
          status: "queued",
          session_id: undefined,
          claimed_by: undefined,
        }),
        depth: 1,
        parent_task_id: "task_001",
        child_count: 0,
      }),
      buildTaskTreeNodeFixture({
        task: buildTaskRecordFixture(
          buildTaskFixture({
            id: "task_child_003",
            identifier: "TASK-46",
            title: "Sync support fallback ownership",
            status: "ready",
            active_run: null,
            child_count: 0,
            dependency_count: 0,
            owner: { kind: "agent_session", ref: storyAgentNames.support },
          })
        ) as unknown as TaskTreeNode["task"],
        active_run: null,
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
          session_id: storySessionIds.product,
          created_at: "2026-04-17T09:58:00Z",
          updated_at: "2026-04-17T18:01:00Z",
          agent_name: storyAgentNames.product,
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
      origin: { kind: "cli", ref: storyPeople.primaryOperator },
      session_id: storySessionIds.product,
      idempotency_key: "launch-command-run",
      claimed_by: { kind: "agent_session", ref: storyAgentNames.product },
      ...overrides.run,
    }),
    task: buildTaskRecordFixture(
      TASK_FIXTURES[0]!,
      overrides.task
    ) as unknown as TaskRunDetailView["task"],
    summary: {
      last_activity_at: "2026-04-17T18:01:00Z",
      last_event_type: "task.run_progress",
      tool_call_count: 7,
      input_tokens: 21_842,
      output_tokens: 4_918,
      total_tokens: 26_760,
      turn_count: 9,
      total_cost: 0.31,
      cost_currency: "USD",
      ...overrides.summary,
    },
    session,
  } as TaskRunDetailView;
}

export const taskDashboardFixture = buildDashboardFixture();
export const taskInboxFixture = buildInboxFixture({
  archived_total: 1,
  total: 5,
  unread_total: 4,
  groups: buildPopulatedInboxGroups(),
});
export const taskDetailFixture = buildDetailFixture();
export const taskTreeFixture = buildTaskTreeFixture();
export const taskRunDetailFixture = buildTaskRunDetailFixture();

export const taskTriageStateFixture: TaskTriageState = {
  actor: { kind: "human", ref: storyPeople.primaryOperator },
  archived: false,
  dismissed: false,
  read: true,
  task_id: "task_001",
  updated_at: "2026-04-17T18:01:00Z",
} as TaskTriageState;

export function buildCreatedTaskFixture(body?: Partial<CreateTaskRequest>): TaskRecord {
  return buildTaskRecordFixture(
    buildTaskFixture({
      id: "task_created",
      identifier: "TASK-NEW",
      title: body?.title?.trim() || "Created launch coordination task",
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

/**
 * Saved-intent fixture: a user-created task that has not been published or
 * started yet. Renders as `saved_intent` in the lifecycle pill, no active run,
 * and no coordination channel binding.
 */
export const savedIntentTaskFixture: TaskListItem = buildTaskFixture({
  id: "task_saved_intent",
  identifier: "TASK-SAVED",
  title: "Draft a partner-facing launch recap",
  status: "draft",
  draft: true,
  active_run: null,
  child_count: 0,
  dependency_count: 0,
  priority: "medium",
  owner: { kind: "human", ref: storyPeople.primaryOperator },
  origin: { kind: "web", ref: storyPeople.primaryOperator },
});

/**
 * Agent-created approval-pending fixture: agent drafted the work and the
 * operator must approve before any run is enqueued.
 */
export const awaitingApprovalTaskFixture: TaskListItem = buildTaskFixture({
  id: "task_awaiting_approval",
  identifier: "TASK-APPROVE",
  title: "Approve coordinator-suggested BR timeout copy update",
  status: "blocked",
  draft: false,
  active_run: null,
  approval_policy: "manual",
  approval_state: "pending",
  child_count: 0,
  dependency_count: 0,
  priority: "high",
  owner: { kind: "human", ref: storyPeople.productLead },
  origin: { kind: "agent_session", ref: storyAgentNames.product },
  created_by: { kind: "agent_session", ref: storyAgentNames.product },
});

/**
 * Queued-with-coordination fixture: a coordinator-handoff run was enqueued and
 * is bound to a stable coordination channel, but no worker has claimed it yet.
 */
export const queuedCoordinatedTaskFixture: TaskListItem = buildTaskFixture({
  id: "task_queued_coordinated",
  identifier: "TASK-QUEUED",
  title: "Coordinate partner settlement replay closeout",
  status: "in_progress",
  active_run: buildTaskRunFixture({
    id: "run_queued_coordinated",
    task_id: "task_queued_coordinated",
    attempt: 1,
    max_attempts: 3,
    status: "queued",
    queued_at: "2026-04-17T09:55:00Z",
    started_at: null,
    session_id: undefined,
    claim_token_hash: "sha256:queued-coordinated",
    coordination_channel_id: "coord-task-queued",
    coordination_channel: {
      id: "coord-task-queued",
      display_name: "TASK-QUEUED coordination",
      workspace_id: STORYBOOK_WORKSPACE_ID,
      task_id: "task_queued_coordinated",
      run_id: "run_queued_coordinated",
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
  }),
  child_count: 0,
  dependency_count: 0,
  priority: "high",
});

/**
 * Coordinator-enabled workspace fixture: indicates that a coordinator is
 * configured + active for the workspace. Consumed by Storybook variations and
 * any future coordinator-aware UI affordances. Channel availability is the
 * UI-visible signal — it never implies channel messages own task status.
 */
export interface CoordinatorEnabledWorkspaceFixture {
  workspaceId: string;
  coordinatorEnabled: boolean;
  coordinatorAgentName: string;
  defaultChannelDisplayName: string;
}

export const coordinatorEnabledWorkspaceFixture: CoordinatorEnabledWorkspaceFixture = {
  workspaceId: STORYBOOK_WORKSPACE_ID,
  coordinatorEnabled: true,
  coordinatorAgentName: storyCoordinatorAgentName,
  defaultChannelDisplayName: "Launch War Room coordination",
};

export function buildTaskExecutionProfileFixture(
  overrides: Partial<TaskExecutionProfile> = {}
): TaskExecutionProfile {
  const taskId = overrides.task_id ?? "task_001";
  return {
    task_id: taskId,
    coordinator: {
      mode: "guided",
      agent_name: storyCoordinatorAgentName,
      guidance: "Lead the launch room through the 18:30 UTC cutover decisions.",
      provider: "anthropic",
      model: "claude-opus-4-7",
      ...overrides.coordinator,
    },
    worker: {
      mode: "select",
      allowed_agent_names: [storyAgentNames.product, storyAgentNames.platform],
      preferred_agent_names: [storyAgentNames.product],
      preferred_capabilities: ["task.execute"],
      required_capabilities: ["task.execute"],
      provider: "anthropic",
      model: "claude-opus-4-7",
      ...overrides.worker,
    },
    review: {
      allowed_agent_names: [storyAgentNames.compliance],
      preferred_agent_names: [storyAgentNames.compliance],
      preferred_capabilities: ["review.run"],
      required_capabilities: ["review.run"],
      ...overrides.review,
    },
    sandbox: {
      mode: "ref",
      sandbox_ref: "fintech-launch",
      ...overrides.sandbox,
    },
    participants: {
      preferred_agent_names: [storyAgentNames.copywriter],
      ...overrides.participants,
    },
    created_at: "2026-04-17T08:00:00Z",
    updated_at: "2026-04-17T17:30:00Z",
    ...overrides,
  };
}

export const taskExecutionProfileFixture: TaskExecutionProfile = buildTaskExecutionProfileFixture();

export function buildTaskRunReviewFixture(overrides: Partial<TaskRunReview> = {}): TaskRunReview {
  return {
    review_id: "review_001",
    run_id: "run_001",
    task_id: "task_001",
    review_round: 1,
    attempt: 1,
    policy: "on_success",
    status: "in_review",
    requested_at: "2026-04-17T18:00:00Z",
    routed_at: "2026-04-17T18:00:30Z",
    started_at: "2026-04-17T18:01:00Z",
    reviewed_at: "2026-04-17T18:00:00Z",
    deadline_at: "2026-04-17T19:00:00Z",
    created_at: "2026-04-17T18:00:00Z",
    updated_at: "2026-04-17T18:01:00Z",
    reviewer_agent_name: storyAgentNames.compliance,
    reviewer_session_id: storySessionIds.product,
    delivery_id: "delivery_review_001",
    ...overrides,
  } as TaskRunReview;
}

export const taskRunReviewFixture: TaskRunReview = buildTaskRunReviewFixture();

export const taskRunReviewListFixture: TaskRunReview[] = [
  taskRunReviewFixture,
  buildTaskRunReviewFixture({
    review_id: "review_002",
    review_round: 2,
    status: "recorded",
    outcome: "rejected",
    reason: "missing partner-bank reconciliation evidence",
    next_round_guidance: "Attach the partner-bank reconciliation artifacts before the next round.",
    confidence: 0.62,
    reviewer_agent_name: storyAgentNames.compliance,
    reviewed_at: "2026-04-17T18:30:00Z",
    updated_at: "2026-04-17T18:30:30Z",
  }),
];

export function buildTaskRunReviewVerdictResultFixture(
  overrides: Partial<TaskRunReviewVerdictResult> = {}
): TaskRunReviewVerdictResult {
  return {
    review: buildTaskRunReviewFixture({
      review_id: "review_002",
      status: "recorded",
      outcome: "rejected",
      reason: "missing partner-bank reconciliation evidence",
      next_round_guidance:
        "Attach the partner-bank reconciliation artifacts before the next round.",
      confidence: 0.62,
      reviewed_at: "2026-04-17T18:30:00Z",
      updated_at: "2026-04-17T18:30:30Z",
      ...overrides.review,
    }),
    continuation_run: buildTaskRunRecordFixture({
      id: "run_continuation_001",
      task_id: "task_001",
      attempt: 3,
      status: "queued",
      queued_at: "2026-04-17T18:30:30Z",
      started_at: null,
      session_id: undefined,
      claim_token_hash: "sha256:continuation-launch-command",
      coordination_channel_id: "coord-launch-001",
    }) as TaskRunReviewVerdictResult["continuation_run"],
    circuit_opened: false,
    ...overrides,
  } as TaskRunReviewVerdictResult;
}

export const taskRunReviewVerdictResultFixture: TaskRunReviewVerdictResult =
  buildTaskRunReviewVerdictResultFixture();

export function buildBridgeNotificationCursorFixture(
  overrides: Partial<TaskBridgeNotificationCursor> = {}
): TaskBridgeNotificationCursor {
  return {
    consumer_id: "bridge_task_subscription:bsub_001",
    stream_name: "task_events",
    subject_id: "task_001",
    last_sequence: 14,
    last_delivery_id: "delivery_evt_014",
    last_delivered_at: "2026-04-17T18:01:00Z",
    last_error: undefined,
    updated_at: "2026-04-17T18:01:00Z",
    ...overrides,
  } as TaskBridgeNotificationCursor;
}

export function buildTaskBridgeNotificationSubscriptionFixture(
  overrides: Partial<TaskBridgeNotificationSubscription> = {}
): TaskBridgeNotificationSubscription {
  return {
    subscription_id: overrides.subscription_id ?? "bsub_001",
    task_id: overrides.task_id ?? "task_001",
    bridge_instance_id: overrides.bridge_instance_id ?? "bridge_instance_alpha",
    delivery_mode: overrides.delivery_mode ?? "direct-send",
    scope: overrides.scope ?? "workspace",
    workspace_id: overrides.workspace_id ?? STORYBOOK_WORKSPACE_ID,
    peer_id: overrides.peer_id ?? "peer_launch_observer",
    group_id: overrides.group_id,
    thread_id: overrides.thread_id,
    created_by: overrides.created_by ?? {
      kind: "human",
      ref: storyPeople.primaryOperator,
    },
    created_at: overrides.created_at ?? "2026-04-17T16:00:00Z",
    updated_at: overrides.updated_at ?? "2026-04-17T18:01:00Z",
    cursor:
      overrides.cursor ??
      buildBridgeNotificationCursorFixture({
        subject_id: overrides.task_id ?? "task_001",
        consumer_id: `bridge_task_subscription:${overrides.subscription_id ?? "bsub_001"}`,
      }),
  } as TaskBridgeNotificationSubscription;
}

export const taskBridgeNotificationSubscriptionFixture: TaskBridgeNotificationSubscription =
  buildTaskBridgeNotificationSubscriptionFixture();

export const taskBridgeNotificationSubscriptionsFixture: TaskBridgeNotificationSubscription[] = [
  taskBridgeNotificationSubscriptionFixture,
  buildTaskBridgeNotificationSubscriptionFixture({
    subscription_id: "bsub_002",
    bridge_instance_id: "bridge_instance_beta",
    delivery_mode: "reply",
    scope: "global",
    workspace_id: undefined,
    peer_id: "peer_partner_observer",
    group_id: "launch_observers",
    thread_id: "thread_launch_partner",
    cursor: buildBridgeNotificationCursorFixture({
      consumer_id: "bridge_task_subscription:bsub_002",
      subject_id: "task_001",
      last_sequence: 0,
      last_delivery_id: undefined,
      last_delivered_at: null,
      updated_at: null,
    }),
  }),
];

export function buildTaskContextBundleFixture(
  overrides: Partial<TaskContextBundle> = {}
): TaskContextBundle {
  const taskId = overrides.task?.id ?? "task_001";
  return {
    task: {
      id: taskId,
      identifier: overrides.task?.identifier ?? "TASK-1",
      title: overrides.task?.title ?? "Lock launch blockers for the 18:30 UTC cutover",
      status: overrides.task?.status ?? "in_progress",
      scope: overrides.task?.scope ?? "workspace",
      priority: overrides.task?.priority ?? "high",
      latest_event_seq: overrides.task?.latest_event_seq ?? 14,
      workspace_id: overrides.task?.workspace_id ?? STORYBOOK_WORKSPACE_ID,
      owner: overrides.task?.owner ?? {
        kind: "agent_session",
        ref: storyAgentNames.product,
      },
    },
    current_run: overrides.current_run ?? {
      id: "run_001",
      task_id: taskId,
      attempt: 2,
      status: "running",
      queued_at: "2026-04-17T09:58:00Z",
      claimed_at: "2026-04-17T09:58:30Z",
      heartbeat_at: "2026-04-17T18:00:00Z",
      lease_until: "2026-04-17T18:05:00Z",
      ended_at: "2026-04-17T18:00:00Z",
      started_at: "2026-04-17T09:59:00Z",
      max_attempts: 3,
      session_id: storySessionIds.product,
      claim_token_hash: "sha256:launch-command-run",
      coordination_channel_id: "coord-launch-001",
    },
    execution_profile: overrides.execution_profile ?? buildTaskExecutionProfileFixture(),
    latest_event_seq: overrides.latest_event_seq ?? 14,
    limits: overrides.limits ?? {
      context_body_max_bytes: 65_536,
      max_runtime_seconds: 1_800,
      summary_max_bytes: 4_096,
    },
    prior_attempts: overrides.prior_attempts ?? [],
    recent_events: overrides.recent_events ?? [],
    review_history: overrides.review_history ?? [],
    handoff_summary: overrides.handoff_summary,
    review_continuation: overrides.review_continuation ?? null,
  } as TaskContextBundle;
}

export const taskContextBundleFixture: TaskContextBundle = buildTaskContextBundleFixture();

export function buildAgentContextFixture(
  overrides: Partial<AgentContextView> = {}
): AgentContextView {
  return {
    self: {
      session_id: storySessionIds.product,
      agent_name: storyAgentNames.product,
      provider: "anthropic",
      model: "claude-opus-4-7",
    },
    session: {
      id: storySessionIds.product,
      created_at: "2026-04-17T09:58:00Z",
      updated_at: "2026-04-17T18:01:00Z",
      state: "active",
      name: "launch command session",
    },
    workspace: {
      id: STORYBOOK_WORKSPACE_ID,
      name: "Fintech Launch",
      root_dir: "/workspaces/fintech-launch",
    },
    capabilities: {
      capabilities: [{ id: "task.execute", summary: "Execute tasks" }],
      section: { limit: 64, returned: 1, truncated: false },
    },
    coordination_channel: {
      available: true,
      channel: {
        id: "coord-launch-001",
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
    },
    inbox_summary: {
      unread_count: 0,
      items: [],
      section: { limit: 32, returned: 0, truncated: false },
    },
    limits: {
      context_section_limit: 32,
      max_active_task_leases: 4,
      max_children: 8,
      max_spawn_depth: 3,
    },
    peer_roster: {
      peers: [],
      section: { limit: 16, returned: 0, truncated: false },
    },
    provenance: {
      generated_at: "2026-04-17T18:01:00Z",
      source: "test",
    },
    soul: {
      active: true,
      enabled: true,
      present: true,
      principles: [],
      tone: [],
      valid: true,
    },
    task: {
      available: true,
      task: {
        id: "task_001",
        identifier: "TASK-1",
        title: "Lock launch blockers for the 18:30 UTC cutover",
        status: "in_progress",
        scope: "workspace",
        priority: "high",
        latest_event_seq: 14,
        workspace_id: STORYBOOK_WORKSPACE_ID,
        owner: { kind: "agent_session", ref: storyAgentNames.product },
      },
      bundle: buildTaskContextBundleFixture(),
    },
    ...overrides,
  } as AgentContextView;
}

export const agentContextFixture: AgentContextView = buildAgentContextFixture();
