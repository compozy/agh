import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyWorkspaceNames } from "@/storybook/fintech-scenario";
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

function ModeToolbarHarness({ initialMode }: { initialMode: TaskFormMode }) {
  const [mode, setMode] = useState<TaskFormMode>(initialMode);
  const [scope, setScope] = useState<TaskScope>("workspace");

  return (
    <CenteredSurface className="items-start justify-center p-6">
      <div className="w-full max-w-(--width-modal-md) overflow-hidden rounded-xl border border-line bg-canvas-soft">
        <ModeToolbar
          mode={mode}
          onModeChange={setMode}
          onScopeChange={setScope}
          scope={scope}
          workspaceName={storyWorkspaceNames.hq}
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
