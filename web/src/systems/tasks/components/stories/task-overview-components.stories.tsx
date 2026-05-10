import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";


import type { MultiAgentAgent } from "@/hooks/routes/use-task-detail-page";
import { PanelSurface } from "@/storybook/story-layout";
import {
  buildDashboardFixture,
  buildDetailFixture,
  buildTaskTreeFixture,
  buildInboxFixture,
  TASK_FIXTURES,
  taskBridgeNotificationSubscriptionsFixture,
  taskExecutionProfileFixture,
  taskRunReviewListFixture,
  taskTimelineFixture,
} from "@/systems/tasks/mocks";

import { TaskCard } from "../task-card";
import { TaskDeleteAction } from "../task-delete-action";
import { TasksBridgeNotificationsCard } from "../tasks-bridge-notifications-card";
import { TasksDashboardActiveRuns } from "../tasks-dashboard-active-runs";
import { TasksDashboardCards } from "../tasks-dashboard-cards";
import { TasksDashboardQueueHealth } from "../tasks-dashboard-queue-health";
import { TasksDashboardStatusBreakdown } from "../tasks-dashboard-status-breakdown";
import { TasksDetailChildrenPanel } from "../tasks-detail-children-panel";
import { TasksDetailDependenciesPanel } from "../tasks-detail-dependencies-panel";
import { TasksDetailRunsPanel } from "../tasks-detail-runs-panel";
import { TasksExecutionProfileCard } from "../tasks-execution-profile-card";
import { TasksInboxItem } from "../tasks-inbox-item";
import { TasksInboxLaneTabs } from "../tasks-inbox-lane-tabs";
import { TasksMultiAgentPanel } from "../tasks-multi-agent-panel";
import { TasksDetailOrchestrationPanel } from "../tasks-detail-orchestration-panel";
import { TasksReviewsCard } from "../tasks-reviews-card";
import { TasksStreamResumeCard } from "../tasks-stream-resume-card";
import { TasksTimelinePanel } from "../tasks-timeline-panel";

const dashboard = buildDashboardFixture();
const detail = buildDetailFixture();
const inbox = buildInboxFixture();
const tree = buildTaskTreeFixture();
const treeNodes: MultiAgentAgent[] = [tree.root, ...(tree.descendants ?? [])].map(
  (node, index) => ({
    node,
    isRoot: index === 0,
    isPrimary: index === 0,
    isLive: index < 2,
    label: node.task.owner?.ref ?? node.task.identifier ?? node.task.id,
  })
);
const inboxItems = inbox.groups?.flatMap(group => group.items ?? []) ?? [];

const meta: Meta<typeof TaskCard> = {
  title: "systems/tasks/TaskOverviewComponents",
  component: TaskCard,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Standalone task cards, dashboard cards, detail tables, inbox rows, reviews, stream, and notification cards.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[820px] p-6">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Task cards cover running, blocked, and failed actions.
 */
export const Cards: Story = {
  args: {},
  render: () => (
    <div className="grid max-w-2xl gap-3">
      <TaskCard task={TASK_FIXTURES[0]!} selected onSelect={fn()} />
      <TaskCard task={TASK_FIXTURES[4]!} onSelect={fn()} />
      <TaskCard task={TASK_FIXTURES[3]!} onSelect={fn()} onRetry={fn()} />
    </div>
  ),
};

/**
 * Dashboard subcomponents render the same payload used by the dashboard view.
 */
export const DashboardSections: Story = {
  args: {},
  render: () => (
    <div className="grid gap-5">
      <TasksDashboardCards dashboard={dashboard} />
      <div className="grid gap-5 xl:grid-cols-3">
        <TasksDashboardStatusBreakdown dashboard={dashboard} />
        <TasksDashboardQueueHealth dashboard={dashboard} />
        <TasksDashboardActiveRuns dashboard={dashboard} />
      </div>
    </div>
  ),
};

/**
 * Detail tables expose children, dependencies, and historical runs.
 */
export const DetailTables: Story = {
  args: {},
  render: () => (
    <div className="grid gap-5">
      <TasksDetailChildrenPanel items={detail.children ?? []} />
      <TasksDetailDependenciesPanel dependencies={detail.dependency_references ?? []} />
      <TasksDetailRunsPanel taskId={detail.task.id} runs={detail.runs ?? []} />
    </div>
  ),
};

/**
 * Inbox tabs and rows show lane counts, unread state, and row actions.
 */
export const Inbox: Story = {
  args: {},
  render: () => (
    <div className="grid max-w-3xl gap-4">
      <TasksInboxLaneTabs inbox={inbox} value="all" onChange={fn()} />
      {inboxItems.slice(0, 3).map(item => (
        <TasksInboxItem
          key={item.task.id}
          item={item}
          onApprove={fn()}
          onReject={fn()}
          onRetry={fn()}
          onArchive={fn()}
          onDismiss={fn()}
          onMarkRead={fn()}
          onOpen={fn()}
        />
      ))}
    </div>
  ),
};

/**
 * Delete action shows the destructive confirmation flow used by task detail
 * and editor surfaces.
 */
export const DeleteAction: StoryObj<typeof TaskDeleteAction> = {
  args: {},
  render: () => (
    <TaskDeleteAction
      taskId={detail.task.id}
      taskTitle={detail.task.title}
      onDelete={fn()}
      triggerTestId="story-task-delete-trigger"
    />
  ),
};

/**
 * Operational cards show timeline, review, stream, notification, and execution-profile states.
 */
export const OperationalCards: Story = {
  args: {},
  render: () => (
    <div className="grid gap-5">
      <TasksTimelinePanel items={taskTimelineFixture} isLive />
      <TasksReviewsCard reviews={taskRunReviewListFixture} />
      <TasksStreamResumeCard
        latestEventSeq={detail.task.latest_event_seq}
        hasLatestEventSeq
        streamSeedSequence={detail.task.latest_event_seq + 1}
        streamState="connected"
        streamErrorMessage={null}
      />
      <TasksBridgeNotificationsCard
        subscriptions={taskBridgeNotificationSubscriptionsFixture}
        onCreate={async () => undefined}
        onDelete={async () => undefined}
      />
      <TasksExecutionProfileCard
        taskId={detail.task.id}
        profile={taskExecutionProfileFixture}
        onSetProfile={async () => undefined}
        onDeleteProfile={async () => undefined}
      />
    </div>
  ),
};

/**
 * Multi-agent panel renders root and descendant agent cards from the task tree.
 */
export const MultiAgent: Story = {
  args: {},
  render: () => (
    <TasksMultiAgentPanel
      agents={treeNodes}
      state="ready"
      liveCount={2}
      descendantCount={Math.max(0, treeNodes.length - 1)}
      activeDescendants={1}
      timeline={taskTimelineFixture}
    />
  ),
};

/**
 * Orchestration panel composes stream, execution profile, reviews, and bridge notifications.
 */
export const Orchestration: Story = {
  args: {},
  render: () => (
    <TasksDetailOrchestrationPanel
      profile={{
        taskId: detail.task.id,
        profile: taskExecutionProfileFixture,
        onSetProfile: async () => undefined,
        onDeleteProfile: async () => undefined,
      }}
      reviews={{ reviews: taskRunReviewListFixture }}
      notifications={{
        subscriptions: taskBridgeNotificationSubscriptionsFixture,
        onCreate: async () => undefined,
        onDelete: async () => undefined,
      }}
      stream={{
        latestEventSeq: detail.task.latest_event_seq,
        hasLatestEventSeq: true,
        streamSeedSequence: detail.task.latest_event_seq + 1,
        streamState: "connected",
        streamErrorMessage: null,
      }}
    />
  ),
};
