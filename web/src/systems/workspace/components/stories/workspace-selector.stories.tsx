import type { Meta, StoryObj } from "@storybook/react-vite";
import { Skeleton } from "@agh/ui";
import { http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { CenteredSurface } from "@/storybook/story-layout";
import { useWorkspaces } from "@/systems/workspace";

import { WorkspaceSelector } from "../workspace-selector";

const meta: Meta<typeof WorkspaceSelector> = {
  title: "systems/workspace/WorkspaceSelector",
  component: WorkspaceSelector,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function WorkspaceSelectorFromQuery() {
  const query = useWorkspaces();

  if (query.isLoading) {
    return (
      <CenteredSurface>
        <div className="w-[22rem] space-y-2">
          <Skeleton className="h-9 w-full rounded-lg" />
          <Skeleton className="h-5 w-32 rounded-md" />
        </div>
      </CenteredSurface>
    );
  }

  const workspaces = query.data ?? [];

  return (
    <CenteredSurface>
      <div className="w-[22rem]">
        <WorkspaceSelector
          onValueChange={() => undefined}
          value={workspaces[0]?.id ?? null}
          workspaces={workspaces}
        />
      </div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  render: () => <WorkspaceSelectorFromQuery />,
};

export const Empty: Story = {
  parameters: {
    ...storybookMswParameters({
      workspace: [http.get("/api/workspaces", () => HttpResponse.json({ workspaces: [] }))],
    }),
  },
  render: () => <WorkspaceSelectorFromQuery />,
};
