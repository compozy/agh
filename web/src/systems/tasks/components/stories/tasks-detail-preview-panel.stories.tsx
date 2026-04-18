import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { TasksDetailPreviewPanel } from "../tasks-detail-preview-panel";
import { buildDetailFixture, buildTaskFixture } from "./fixtures";

const meta: Meta<typeof TasksDetailPreviewPanel> = {
  title: "systems/tasks/TasksDetailPreviewPanel",
  component: TasksDetailPreviewPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {
  render: () => (
    <PanelSurface>
      <TasksDetailPreviewPanel detail={null} task={null} />
    </PanelSurface>
  ),
};

export const Loading: Story = {
  render: () => (
    <PanelSurface>
      <TasksDetailPreviewPanel detail={null} isLoading task={buildTaskFixture()} />
    </PanelSurface>
  ),
};

export const WithError: Story = {
  render: () => (
    <PanelSurface>
      <TasksDetailPreviewPanel
        detail={null}
        errorMessage="Task preview could not be loaded."
        task={buildTaskFixture()}
      />
    </PanelSurface>
  ),
};

export const Populated: Story = {
  render: () => {
    const task = buildTaskFixture();
    return (
      <PanelSurface>
        <TasksDetailPreviewPanel detail={buildDetailFixture()} task={task} />
      </PanelSurface>
    );
  },
};

export const LongTitle: Story = {
  render: () => {
    const task = buildTaskFixture({
      title:
        "Investigate edge-case regression in the persisted tool-state store that leaks streaming chunks across sessions",
    });
    const detail = buildDetailFixture();
    detail.task = { ...detail.task, title: task.title };
    return (
      <PanelSurface>
        <TasksDetailPreviewPanel detail={detail} task={task} />
      </PanelSurface>
    );
  },
};
