import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { agentFixtures } from "@/systems/agent/mocks";
import { sessionFixtures } from "@/systems/session/mocks";
import { workspaceFixtures } from "@/systems/workspace/mocks";

import { AppSidebar, type AppSidebarProps } from "../app-sidebar";

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="flex min-h-[640px] bg-[color:var(--color-canvas)] text-[color:var(--color-text-primary)]"
      style={{ width: 960 }}
    >
      {children}
      <div className="flex min-h-0 flex-1 items-center justify-center px-10 text-sm text-[color:var(--color-text-secondary)]">
        Outlet content
      </div>
    </div>
  );
}

type StoryArgs = Omit<AppSidebarProps, "collapsed" | "onCollapseChange" | "onSelectWorkspace"> & {
  defaultCollapsed?: boolean;
  defaultWorkspaceId?: string | null;
};

function AppSidebarHarness({
  defaultCollapsed = false,
  defaultWorkspaceId,
  activeWorkspaceId,
  activeWorkspace,
  ...rest
}: StoryArgs) {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);
  const [workspaceId, setWorkspaceId] = useState<string | null>(
    defaultWorkspaceId ?? activeWorkspaceId ?? null
  );
  const resolvedActive =
    rest.workspaces?.find(ws => ws.id === workspaceId) ?? activeWorkspace ?? undefined;

  return (
    <Frame>
      <AppSidebar
        {...rest}
        activeWorkspace={resolvedActive}
        activeWorkspaceId={workspaceId}
        onSelectWorkspace={setWorkspaceId}
        collapsed={collapsed}
        onCollapseChange={setCollapsed}
      />
    </Frame>
  );
}

const meta: Meta<typeof AppSidebarHarness> = {
  title: "app/AppSidebar",
  component: AppSidebarHarness,
  parameters: {
    layout: "fullscreen",
    router: { kind: "stub" },
    docs: {
      description: {
        component:
          "Thin composition over `@agh/ui` `Sidebar`. The rail owns the workspace switcher, the header owns the wordmark, the nav owns the agent tree and workspace nav, and the footer owns the connection indicator + settings link.",
      },
    },
  },
  args: {
    workspaces: workspaceFixtures,
    activeWorkspaceId: workspaceFixtures[1].id,
    activeWorkspace: workspaceFixtures[1],
    onAddWorkspace: () => undefined,
    health: { version: "0.4.1" },
    connectionStatus: "connected",
    agents: agentFixtures,
    agentsLoading: false,
    agentsError: false,
    sessions: sessionFixtures,
    onNewSession: () => undefined,
    isCreatingSession: false,
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Collapsed: Story = {
  args: {
    defaultCollapsed: true,
  },
  parameters: {
    docs: {
      description: {
        story:
          "Sidebar starts collapsed. The panel animates to zero width while the rail stays fully visible.",
      },
    },
  },
};

export const NoWorkspaces: Story = {
  args: {
    workspaces: [],
    activeWorkspace: undefined,
    activeWorkspaceId: null,
    defaultWorkspaceId: null,
    agents: [],
    sessions: [],
  },
  parameters: {
    docs: {
      description: {
        story: "Empty state: no workspaces, no agents — only the + add-workspace affordance.",
      },
    },
  },
};

export const ManyWorkspaces: Story = {
  args: {
    workspaces: [
      ...workspaceFixtures,
      {
        id: "ws_research",
        root_dir: "/workspaces/research",
        add_dirs: [],
        name: "research",
        created_at: "2026-04-13T09:00:00Z",
        updated_at: "2026-04-17T10:00:00Z",
      },
      {
        id: "ws_ops",
        root_dir: "/workspaces/ops",
        add_dirs: [],
        name: "ops",
        created_at: "2026-04-10T09:00:00Z",
        updated_at: "2026-04-17T10:05:00Z",
      },
      {
        id: "ws_docs",
        root_dir: "/workspaces/docs",
        add_dirs: [],
        name: "docs",
        created_at: "2026-04-12T09:00:00Z",
        updated_at: "2026-04-17T10:10:00Z",
      },
    ],
  },
};

export const Disconnected: Story = {
  args: {
    connectionStatus: "disconnected",
  },
  parameters: {
    docs: {
      description: {
        story: "Connection indicator shows Disconnected when the daemon is not reachable.",
      },
    },
  },
};

export const Reconnecting: Story = {
  args: {
    connectionStatus: "reconnecting",
  },
};

export const TogglesCollapse: Story = {
  args: {},
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const trigger = await canvas.findByRole("button", { name: "Toggle sidebar" });
    await expect(trigger).toHaveAttribute("aria-expanded", "true");

    const sidebar = canvasElement.querySelector<HTMLElement>("[data-slot=sidebar]");
    const rail = canvasElement.querySelector<HTMLElement>("[data-slot=sidebar-rail]");
    await expect(sidebar).not.toBeNull();

    await userEvent.click(trigger);
    await waitFor(() => expect(trigger).toHaveAttribute("aria-expanded", "false"));
    await expect(sidebar).toHaveAttribute("data-state", "collapsed");
    await expect(rail?.offsetWidth).toBeGreaterThan(0);
  },
};

export const SwitchesWorkspace: Story = {
  args: {},
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const targetAvatar = await canvas.findByTestId(`workspace-avatar-${workspaceFixtures[0].id}`);
    await expect(targetAvatar).toHaveAttribute("data-active", "false");

    await userEvent.click(targetAvatar);

    await waitFor(() =>
      expect(canvas.getByTestId(`workspace-avatar-${workspaceFixtures[0].id}`)).toHaveAttribute(
        "data-active",
        "true"
      )
    );
    await expect(canvas.getByTestId(`workspace-avatar-${workspaceFixtures[1].id}`)).toHaveAttribute(
      "data-active",
      "false"
    );
  },
};
