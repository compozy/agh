import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyWorkspaceIds, storyWorkspaceNames } from "@/storybook/fintech-scenario";
import { CenteredSurface } from "@/storybook/story-layout";
import type { TaskScope } from "@/systems/tasks";

import { ModeToolbar, type TaskFormMode } from "../../task-form/mode-toolbar";

const meta: Meta<typeof ModeToolbar> = {
  title: "systems/tasks/task-form/ModeToolbar",
  component: ModeToolbar,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const storyWorkspaces = [
  { id: storyWorkspaceIds.hq, name: storyWorkspaceNames.hq },
  { id: storyWorkspaceIds.risk, name: storyWorkspaceNames.risk },
  { id: storyWorkspaceIds.growth, name: storyWorkspaceNames.growth },
];

function ModeToolbarHarness({ initialMode }: { initialMode: TaskFormMode }) {
  const [mode, setMode] = useState<TaskFormMode>(initialMode);
  const [scope, setScope] = useState<TaskScope>("workspace");
  const [workspaceId, setWorkspaceId] = useState<string | null>(storyWorkspaceIds.hq);

  return (
    <CenteredSurface className="items-start justify-center p-6">
      <div className="w-full max-w-(--width-modal-md) overflow-hidden rounded-xl border border-line bg-canvas-soft">
        <ModeToolbar
          mode={mode}
          onModeChange={setMode}
          onScopeChange={setScope}
          onWorkspaceChange={setWorkspaceId}
          scope={scope}
          workspaceId={workspaceId}
          workspaces={storyWorkspaces}
        />
      </div>
    </CenteredSurface>
  );
}

export const Simple: Story = {
  args: {},
  render: () => <ModeToolbarHarness initialMode="simple" />,
};

export const Advanced: Story = {
  args: {},
  render: () => <ModeToolbarHarness initialMode="advanced" />,
};
