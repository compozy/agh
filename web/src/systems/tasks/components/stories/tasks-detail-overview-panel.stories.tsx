import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { TasksDetailOverviewPanel } from "../tasks-detail-overview-panel";
import { buildDetailFixture } from "./fixtures";

const meta: Meta<typeof TasksDetailOverviewPanel> = {
  title: "systems/tasks/TasksDetailOverviewPanel",
  component: TasksDetailOverviewPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <TasksDetailOverviewPanel detail={buildDetailFixture()} />
    </PanelSurface>
  ),
};

export const WithoutActiveRun: Story = {
  render: () => {
    const detail = buildDetailFixture();
    detail.summary = { ...detail.summary, active_run: null } as typeof detail.summary;
    return (
      <PanelSurface>
        <TasksDetailOverviewPanel detail={detail} />
      </PanelSurface>
    );
  },
};

export const NoDescription: Story = {
  render: () => {
    const detail = buildDetailFixture();
    detail.task = { ...detail.task, description: "" };
    return (
      <PanelSurface>
        <TasksDetailOverviewPanel detail={detail} />
      </PanelSurface>
    );
  },
};
