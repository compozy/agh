import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { storyWorkspaceIds, storyWorkspaceNames } from "@/storybook/fintech-scenario";
import { CenteredSurface } from "@/storybook/story-layout";

import { ScopeSelector, type ScopeSelectorScope } from "../scope-selector";

const storyWorkspaces = [
  { id: storyWorkspaceIds.hq, name: storyWorkspaceNames.hq },
  { id: storyWorkspaceIds.risk, name: storyWorkspaceNames.risk },
  { id: storyWorkspaceIds.growth, name: storyWorkspaceNames.growth },
];

interface ScopeSelectorHarnessProps {
  initialScope?: ScopeSelectorScope;
  initialWorkspaceId?: string | null;
  workspaceDisabled?: boolean;
  emptyRegistry?: boolean;
}

function ScopeSelectorHarness({
  initialScope = "workspace",
  initialWorkspaceId = storyWorkspaceIds.hq,
  workspaceDisabled = false,
  emptyRegistry = false,
}: ScopeSelectorHarnessProps) {
  const [scope, setScope] = useState<ScopeSelectorScope>(initialScope);
  const [workspaceId, setWorkspaceId] = useState<string | null>(initialWorkspaceId);

  return (
    <CenteredSurface className="items-start justify-center p-6">
      <div className="w-full max-w-[520px] border border-line bg-canvas-soft p-4">
        <ScopeSelector
          scope={scope}
          workspaceId={workspaceId}
          workspaces={emptyRegistry ? [] : storyWorkspaces}
          onScopeChange={setScope}
          onWorkspaceChange={setWorkspaceId}
          testIdPrefix="scope"
          workspaceDisabled={workspaceDisabled}
        />
      </div>
    </CenteredSurface>
  );
}

const meta: Meta<typeof ScopeSelectorHarness> = {
  title: "systems/workspace/ScopeSelector",
  component: ScopeSelectorHarness,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shared global/workspace scope control for task, job, and trigger creation surfaces. Workspace mode uses the compact workspace command selector.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Workspace: Story = {};

export const Global: Story = {
  args: {
    initialScope: "global",
    initialWorkspaceId: null,
  },
};

export const SwitchesWorkspace: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByTestId("scope-workspace-select"));
    await userEvent.click(canvas.getByTestId("scope-workspace-item-" + storyWorkspaceIds.risk));

    await waitFor(() =>
      expect(canvas.getByTestId("workspace-switcher-name")).toHaveTextContent(
        storyWorkspaceNames.risk
      )
    );
  },
};

export const EmptyRegistry: Story = {
  args: {
    emptyRegistry: true,
  },
};

export const WorkspaceDisabled: Story = {
  args: {
    initialScope: "global",
    initialWorkspaceId: null,
    workspaceDisabled: true,
  },
};
