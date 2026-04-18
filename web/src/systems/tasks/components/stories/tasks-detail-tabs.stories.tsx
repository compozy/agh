import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { PanelSurface } from "@/storybook/story-layout";
import type { TaskDetailPanel } from "@/hooks/routes/use-task-detail-page";
import { TasksDetailTabs } from "../tasks-detail-tabs";

const meta: Meta<typeof TasksDetailTabs> = {
  title: "systems/tasks/TasksDetailTabs",
  component: TasksDetailTabs,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const ITEMS = [
  { id: "overview" as const, label: "Overview" },
  { id: "runs" as const, label: "Runs", count: 3 },
  { id: "timeline" as const, label: "Events", live: true },
  { id: "agents" as const, label: "Agents", count: 2, live: true },
  { id: "children" as const, label: "Children", count: 2 },
  { id: "dependencies" as const, label: "Dependencies", count: 1 },
];

function Controlled() {
  const [active, setActive] = useState<TaskDetailPanel>("overview");
  return <TasksDetailTabs active={active} items={ITEMS} onChange={setActive} />;
}

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <Controlled />
    </PanelSurface>
  ),
};
