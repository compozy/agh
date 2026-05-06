import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { storyAgentNames, storyWorkspacePaths } from "@/storybook/fintech-scenario";
import { agentFixtures } from "@/systems/agent/mocks";
import type { AgentPayload } from "@/systems/agent/types";
import { sessionFixtures } from "@/systems/session/mocks";
import { workspaceFixtures } from "@/systems/workspace/mocks";

import { AppSidebar, type AppSidebarProps } from "../app-sidebar";

const sidebarStoryCategoryByName: Record<string, string[]> = {
  [storyAgentNames.cto]: ["Engineering", "Leadership"],
  [storyAgentNames.platform]: ["Engineering", "Platform"],
  [storyAgentNames.frontend]: ["Engineering", "Platform"],
  [storyAgentNames.release]: ["Engineering", "Platform"],
  [storyAgentNames.cfo]: ["Finance"],
  [storyAgentNames.marketing]: ["Marketing", "Campaigns"],
  [storyAgentNames.copywriter]: ["Marketing", "Campaigns"],
  [storyAgentNames.product]: ["Product"],
  [storyAgentNames.support]: ["Support"],
  [storyAgentNames.fraud]: ["Risk", "Fraud"],
  [storyAgentNames.compliance]: ["Risk", "Compliance"],
};

const categorizedAgentFixtures: AgentPayload[] = agentFixtures.map(agent => {
  const category_path = sidebarStoryCategoryByName[agent.name];
  return category_path ? { ...agent, category_path } : agent;
});

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
  ...rest
}: StoryArgs) {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);
  const [workspaceId, setWorkspaceId] = useState<string | null>(
    defaultWorkspaceId ?? activeWorkspaceId ?? null
  );

  return (
    <Frame>
      <AppSidebar
        {...rest}
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
          "Thin composition over `@agh/ui` `Sidebar`. The rail owns the workspace switcher; the nav holds Dashboard plus four labeled sections: Agents, Operate (Network/Tasks/Jobs/Triggers), Catalog (Knowledge/Skills/Bridges), and System (Sandbox/Settings); the footer keeps only the connection indicator and version badge. The global `agh` wordmark lives in the app-shell header one level up.",
      },
    },
  },
  args: {
    workspaces: workspaceFixtures,
    activeWorkspaceId: workspaceFixtures[1].id,
    onAddWorkspace: () => undefined,
    health: { version: "0.4.1" },
    connectionStatus: "connected",
    agents: agentFixtures,
    agentsLoading: false,
    agentsError: false,
    sessions: sessionFixtures,
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Categorized: Story = {
  args: {
    agents: categorizedAgentFixtures,
  },
  parameters: {
    docs: {
      description: {
        story:
          "Agents grouped by `category_path`. Top-level folders (Engineering, Marketing, Risk, ...) expand by default, multi-level branches (Engineering / Platform, Marketing / Campaigns, Risk / Fraud) demonstrate the nested tree, and root-level agents (Finance, Product, Support) sit alongside the folders.",
      },
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByTestId("agent-category-Engineering")).toBeInTheDocument();
    await expect(canvas.getByTestId("agent-category-Marketing")).toBeInTheDocument();
    await expect(canvas.getByTestId("agent-category-Risk")).toBeInTheDocument();
    await expect(canvas.getByTestId("agent-category-Engineering/Platform")).toBeInTheDocument();
    await expect(canvas.getByTestId("agent-category-Marketing/Campaigns")).toBeInTheDocument();
  },
};

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
    activeWorkspaceId: null,
    defaultWorkspaceId: null,
    agents: [],
    sessions: [],
  },
  parameters: {
    docs: {
      description: {
        story: "Empty state: no workspaces, no agents; only the + add-workspace affordance.",
      },
    },
  },
};

export const ManyWorkspaces: Story = {
  args: {
    workspaces: [
      ...workspaceFixtures,
      {
        id: "ws_merchant_success",
        root_dir: "/workspaces/northstar-pay/merchant-success",
        add_dirs: [],
        name: "merchant-success",
        created_at: "2026-04-13T09:00:00Z",
        updated_at: "2026-04-17T10:00:00Z",
      },
      {
        id: "ws_partner_ops",
        root_dir: "/workspaces/northstar-pay/partner-ops",
        add_dirs: [],
        name: "partner-ops",
        created_at: "2026-04-10T09:00:00Z",
        updated_at: "2026-04-17T10:05:00Z",
      },
      {
        id: "ws_collections_lab",
        root_dir: "/workspaces/northstar-pay/collections-lab",
        add_dirs: [storyWorkspacePaths.sharedPolicies],
        name: "collections-lab",
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
