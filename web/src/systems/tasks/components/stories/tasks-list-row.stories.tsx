import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { TasksListRow } from "../tasks-list-row";
import { buildTaskFixture } from "./fixtures";

const meta: Meta<typeof TasksListRow> = {
  title: "systems/tasks/TasksListRow",
  component: TasksListRow,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function Frame({ children }: { children: React.ReactNode }) {
  return <PanelSurface className="max-w-[340px] p-0">{children}</PanelSurface>;
}

export const Pending: Story = {
  render: () => (
    <Frame>
      <TasksListRow
        task={buildTaskFixture({ status: "pending", title: "Pending task", active_run: null })}
      />
    </Frame>
  ),
};

export const Running: Story = {
  render: () => (
    <Frame>
      <TasksListRow task={buildTaskFixture({ status: "in_progress", title: "Running task" })} />
    </Frame>
  ),
};

export const Done: Story = {
  render: () => (
    <Frame>
      <TasksListRow
        task={buildTaskFixture({ status: "completed", title: "Done task", active_run: null })}
      />
    </Frame>
  ),
};

export const Failed: Story = {
  render: () => (
    <Frame>
      <TasksListRow
        task={buildTaskFixture({ status: "failed", title: "Failed task", active_run: null })}
      />
    </Frame>
  ),
};

export const Blocked: Story = {
  render: () => (
    <Frame>
      <TasksListRow
        task={buildTaskFixture({ status: "blocked", title: "Blocked task", active_run: null })}
      />
    </Frame>
  ),
};

export const WithLaneAndSelection: Story = {
  render: () => (
    <Frame>
      <TasksListRow
        lane="approvals"
        selected
        task={buildTaskFixture({ status: "ready", title: "Awaiting approval" })}
      />
    </Frame>
  ),
};
