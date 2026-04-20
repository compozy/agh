import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import type { WorkspacePayload } from "@/systems/workspace";

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

const SINGLE: WorkspacePayload[] = [
  {
    id: "ws_home",
    root_dir: "/Users/pedro",
    add_dirs: [],
    name: "home",
    created_at: "2026-04-10T12:00:00Z",
    updated_at: "2026-04-10T12:00:00Z",
  },
];

const MANY: WorkspacePayload[] = [
  SINGLE[0],
  {
    id: "ws_agh",
    root_dir: "/Users/pedro/Dev/agh",
    add_dirs: [],
    name: "agh",
    created_at: "2026-04-10T12:00:00Z",
    updated_at: "2026-04-10T12:00:00Z",
  },
  {
    id: "ws_kit",
    root_dir: "/Users/pedro/Dev/design-kit",
    add_dirs: [],
    name: "design-kit",
    created_at: "2026-04-10T12:00:00Z",
    updated_at: "2026-04-10T12:00:00Z",
  },
];

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-[22rem]">{children}</div>
    </CenteredSurface>
  );
}

export const Empty: Story = {
  render: () => (
    <Frame>
      <WorkspaceSelector
        workspaces={[]}
        activeWorkspaceId={null}
        onSelectWorkspace={() => undefined}
      />
    </Frame>
  ),
};

export const Single: Story = {
  render: () => (
    <Frame>
      <WorkspaceSelector
        workspaces={SINGLE}
        activeWorkspaceId={SINGLE[0].id}
        globalWorkspaceId={SINGLE[0].id}
        onSelectWorkspace={() => undefined}
      />
    </Frame>
  ),
};

export const Many: Story = {
  render: () => (
    <Frame>
      <WorkspaceSelector
        workspaces={MANY}
        activeWorkspaceId={null}
        globalWorkspaceId="ws_home"
        onSelectWorkspace={() => undefined}
      />
    </Frame>
  ),
};

export const Active: Story = {
  render: () => (
    <Frame>
      <WorkspaceSelector
        workspaces={MANY}
        activeWorkspaceId="ws_agh"
        globalWorkspaceId="ws_home"
        onSelectWorkspace={() => undefined}
      />
    </Frame>
  ),
};
