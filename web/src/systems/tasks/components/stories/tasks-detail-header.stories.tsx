import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { TasksDetailHeader } from "../tasks-detail-header";
import { buildDetailFixture } from "./fixtures";

const meta: Meta<typeof TasksDetailHeader> = {
  title: "systems/tasks/TasksDetailHeader",
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
