import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { PanelSurface } from "@/storybook/story-layout";
import { TasksListPanel } from "../tasks-list-panel";
import { TASK_FIXTURES } from "./fixtures";

const meta: Meta<typeof TasksListPanel> = {
  title: "systems/tasks/TasksListPanel",
  component: TasksListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function Frame({ children }: { children: React.ReactNode }) {
  return <PanelSurface className="max-w-[360px] p-0">{children}</PanelSurface>;
}

function ControlledListPanel(props: Partial<Parameters<typeof TasksListPanel>[0]>) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  return (
    <TasksListPanel
      onSearchChange={setQuery}
      onSelectTask={setSelectedId}
      searchQuery={query}
      selectedTaskId={selectedId}
      tasks={TASK_FIXTURES}
      totalCount={TASK_FIXTURES.length}
      {...props}
    />
  );
}

export const Populated: Story = {
  render: () => (
    <Frame>
      <ControlledListPanel />
    </Frame>
  ),
};

export const Empty: Story = {
  render: () => (
    <Frame>
      <ControlledListPanel tasks={[]} totalCount={0} />
    </Frame>
  ),
};

export const Loading: Story = {
  render: () => (
    <Frame>
      <ControlledListPanel isLoading tasks={[]} totalCount={0} />
    </Frame>
  ),
};

export const WithError: Story = {
  render: () => (
    <Frame>
      <ControlledListPanel
        errorMessage="Workspace offline. Retrying in a few seconds."
        tasks={[]}
        totalCount={0}
      />
    </Frame>
  ),
};

export const LaneSwitch: Story = {
  tags: ["play-fn"],
  render: () => (
    <Frame>
      <ControlledListPanel onCreateTask={() => undefined} />
    </Frame>
  ),
};
