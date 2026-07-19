import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { StorybookUserHomeDirSetup } from "@/storybook/route-story";
import { storyWorkspacePaths } from "@/storybook/fintech-scenario";
import { workspaceFixtures } from "@/systems/workspace/mocks";

import { AppSidebar, type AppSidebarProps } from "../app-sidebar";

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[640px] bg-canvas text-fg" style={{ width: 960 }}>
      {children}
      <div className="flex min-h-0 flex-1 items-center justify-center px-10 text-sm text-muted">
        Outlet content
      </div>
    </div>
  );
}

type StoryArgs = Omit<AppSidebarProps, "collapsed" | "onCollapseChange" | "onSelectWorkspace"> & {
  defaultCollapsed?: boolean;
  defaultWorkspaceId?: string | null;
  userHomeDir?: string | null;
};

function AppSidebarHarness({
  defaultCollapsed = false,
  defaultWorkspaceId,
  activeWorkspaceId,
  userHomeDir = null,
  ...rest
}: StoryArgs) {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);
  const [workspaceId, setWorkspaceId] = useState<string | null>(
    defaultWorkspaceId ?? activeWorkspaceId ?? null
  );

  return (
    <Frame>
      <StorybookUserHomeDirSetup userHomeDir={userHomeDir} />
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
  title: "systems/runtime/components/AppSidebar",
  component: AppSidebarHarness,
  parameters: {
    layout: "fullscreen",
    router: { kind: "stub" },
    docs: {
      description: {
        component:
          "Runtime shell sidebar. The rail owns the brand logo plus workspace avatars; inactive workspaces can expose an exact active-session count and direct return link. The body holds Dashboard plus Operate (Agents first with live/total badge), Catalog, and System; the footer mounts the single `RuntimeConnectionIndicator` (no rail LED) alongside the Restart-daemon control. The wordmark lives in the app-shell header one level up.",
      },
    },
  },
  args: {
    workspaces: workspaceFixtures,
    activeWorkspaceId: workspaceFixtures[1].id,
    onAddWorkspace: () => undefined,
    agentsCount: { live: 2, total: 6 },
    activeSessionCount: 2,
    workspaceSessionActivity: {},
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const NetworkActive: Story = {
  parameters: {
    router: { kind: "stub", initialEntries: ["/network"] },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByTestId("nav-network")).toHaveAttribute("data-active", "true");
    await expect(canvas.getByTestId("nav-active-network")).toBeInTheDocument();
  },
};

export const WithHomeWorkspace: Story = {
  args: {
    // Mark a non-first fixture as the home/global workspace (root_dir === user_home_dir).
    userHomeDir: workspaceFixtures[3].root_dir,
    activeWorkspaceId: workspaceFixtures[3].id,
    defaultWorkspaceId: workspaceFixtures[3].id,
  },
  parameters: {
    docs: {
      description: {
        story:
          "The home/global workspace (its `root_dir` equals the daemon `user_home_dir`) is pinned to the top of the rail with a home glyph instead of a letter avatar, and a hairline divider separates it from the project workspaces below.",
      },
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const home = await canvas.findByTestId(`workspace-avatar-${workspaceFixtures[3].id}`);
    await expect(home).toHaveAttribute("data-home", "true");
    await expect(home.querySelector("svg")).not.toBeNull();

    const rail = canvasElement.querySelector<HTMLElement>("[data-testid=icon-rail]");
    const avatarIds = Array.from(
      rail?.querySelectorAll<HTMLElement>('[data-testid^="workspace-avatar-"]') ?? []
    ).map(node => node.getAttribute("data-testid"));
    await expect(avatarIds[0]).toBe(`workspace-avatar-${workspaceFixtures[3].id}`);
    await expect(canvas.getByTestId("rail-home-divider")).toBeInTheDocument();
  },
};

export const AgentsOperateBadge: Story = {
  args: {
    agentsCount: { live: 3, total: 8 },
  },
  parameters: {
    docs: {
      description: {
        story:
          "Agents is the first Operate nav item. Its live/total badge uses the exact workspace catalog facets supplied by the shell; the per-agent category tree no longer lives in the sidebar.",
      },
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const agentsNav = await canvas.findByTestId("nav-agents");
    await expect(agentsNav).toHaveAttribute("href", "/agents");
    await expect(canvas.getByTestId("agents-live-count")).toBeInTheDocument();
    await expect(canvas.queryByTestId("sidebar-create-agent")).toBeNull();
  },
};

/**
 * Background user sessions remain visible from another workspace and link back to the latest one.
 */
export const BackgroundSessionActivity: Story = {
  args: {
    activeWorkspaceId: workspaceFixtures[1].id,
    defaultWorkspaceId: workspaceFixtures[1].id,
    workspaceSessionActivity: {
      [workspaceFixtures[0].id]: {
        state: "ready",
        count: 2,
        returnTarget: {
          sessionId: "sess_reconcile_payouts",
          agentName: "fraud-investigator",
          title: "Reconcile payout ledger",
        },
      },
    },
  },
  parameters: {
    docs: {
      description: {
        story:
          "An inactive workspace shows its exact active-session count. The workspace avatar is a direct, accessible return link to its latest active user session.",
      },
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const returnLink = await canvas.findByRole("link", {
      name: `Return to ${workspaceFixtures[0].name}: 2 active sessions. Latest: Reconcile payout ledger`,
    });
    await expect(returnLink).toHaveAttribute(
      "href",
      "/agents/fraud-investigator/sessions/sess_reconcile_payouts"
    );
    await expect(
      canvas.getByTestId(`workspace-active-session-count-${workspaceFixtures[0].id}`)
    ).toHaveTextContent("2");
  },
};

export const BackgroundSessionActivityError: Story = {
  args: {
    activeWorkspaceId: workspaceFixtures[1].id,
    defaultWorkspaceId: workspaceFixtures[1].id,
    workspaceSessionActivity: {
      [workspaceFixtures[0].id]: {
        state: "error",
        message: "Session activity is unavailable for this workspace.",
      },
    },
  },
  parameters: {
    docs: {
      description: {
        story:
          "A failed workspace activity query renders a warning badge instead of a false zero count.",
      },
    },
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(
      canvas.getByTestId(`workspace-session-activity-error-${workspaceFixtures[0].id}`)
    ).toBeInTheDocument();
    await expect(
      canvas.getByRole("button", {
        name: `Workspace: ${workspaceFixtures[0].name}, session activity unavailable`,
      })
    ).toBeEnabled();
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
    agentsCount: { live: 0, total: 0 },
    activeSessionCount: 0,
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

export const SwitchesWorkspaceViaHeader: Story = {
  args: {},
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const target = workspaceFixtures[0];
    if (!target) {
      throw new Error("Expected at least one workspace fixture");
    }

    await userEvent.click(canvas.getByTestId("workspace-switcher"));
    await userEvent.click(canvas.getByTestId(`workspace-command-item-${target.id}`));

    await waitFor(() =>
      expect(canvas.getByTestId(`workspace-avatar-${target.id}`)).toHaveAttribute(
        "data-active",
        "true"
      )
    );
    await expect(canvas.getByTestId("workspace-switcher-name")).toHaveTextContent(target.name);
  },
};
