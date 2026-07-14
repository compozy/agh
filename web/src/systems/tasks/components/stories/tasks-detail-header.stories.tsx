import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { TasksDetailHeader } from "../tasks-detail-header";
import { buildDetailFixture } from "./fixtures";

const meta: Meta<typeof TasksDetailHeader> = {
  title: "systems/tasks/components/TasksDetailHeader",
  component: TasksDetailHeader,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <TasksDetailHeader detail={buildDetailFixture()} onCancel={() => undefined} />
    </PanelSurface>
  ),
};

export const Draft: Story = {
  render: () => {
    const detail = buildDetailFixture();
    detail.task = { ...detail.task, status: "draft" };
    return (
      <PanelSurface>
        <TasksDetailHeader detail={detail} onPublish={() => undefined} />
      </PanelSurface>
    );
  },
};

export const LongTitle: Story = {
  render: () => {
    const detail = buildDetailFixture();
    detail.task = {
      ...detail.task,
      title:
        "Investigate edge-case regression in the persisted tool-state store that leaks streaming chunks across sessions",
    };
    return (
      <PanelSurface>
        <TasksDetailHeader detail={detail} onCancel={() => undefined} />
      </PanelSurface>
    );
  },
};

export const Paused: Story = {
  render: () => {
    const detail = buildDetailFixture();
    detail.task = {
      ...detail.task,
      paused: true,
      paused_reason: "provider incident",
    };
    detail.summary = {
      ...detail.summary,
      effective_paused: true,
      paused_by_task_id: detail.task.id,
    };
    return (
      <PanelSurface>
        <TasksDetailHeader
          detail={detail}
          onEnqueueRun={() => undefined}
          onResume={() => undefined}
        />
      </PanelSurface>
    );
  },
};

/** Ready Task projection whose active run requires operator recovery. */
export const ActiveRunNeedsAttention: Story = {
  args: {},
  render: () => {
    const detail = buildDetailFixture();
    const activeRun = {
      ...detail.summary.active_run!,
      status: "needs_attention" as const,
      session_id: undefined,
      started_at: undefined,
      error: "No capable agent claimed this run before escalation.",
    };

    detail.task = { ...detail.task, status: "ready", max_attempts: 3 };
    detail.summary = { ...detail.summary, status: "ready", active_run: activeRun };

    return (
      <PanelSurface>
        <TasksDetailHeader
          detail={detail}
          onEnqueueRun={() => undefined}
          onRecover={() => undefined}
        />
      </PanelSurface>
    );
  },
};

/** Ready Task projection whose active run has no continuation attempts left. */
export const ExhaustedActiveRunNeedsAttention: Story = {
  args: {},
  render: () => {
    const detail = buildDetailFixture();
    const activeRun = {
      ...detail.summary.active_run!,
      attempt: 1,
      status: "needs_attention" as const,
      session_id: undefined,
      started_at: undefined,
      error: "Run exhausted max_attempts=1.",
    };

    detail.task = { ...detail.task, status: "ready", max_attempts: 1 };
    detail.summary = { ...detail.summary, status: "ready", active_run: activeRun };

    return (
      <PanelSurface>
        <TasksDetailHeader
          detail={detail}
          onEnqueueRun={() => undefined}
          onRecover={() => undefined}
        />
      </PanelSurface>
    );
  },
};
