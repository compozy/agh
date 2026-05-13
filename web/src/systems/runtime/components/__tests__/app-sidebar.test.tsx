import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { AppSidebar, type AppSidebarProps } from "../app-sidebar";
import { computeAgentsCount } from "../app-sidebar";

const onSelectWorkspace = vi.fn();
const onCollapseChange = vi.fn();
const onAddWorkspace = vi.fn();
let matchedRoute: Record<string, boolean> = {};
let matchedRouteFuzzy: Record<string, boolean> = {};

type MatchRouteParams = Record<string, string>;

function routeMatchKey(to: string, params?: MatchRouteParams): string {
  if (!params) return to;
  const serializedParams = Object.entries(params)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => `${key}=${value}`)
    .join("&");
  return `${to}?${serializedParams}`;
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: ReactNode;
    to: string;
    params?: MatchRouteParams;
    [key: string]: unknown;
  }) => {
    const href = params
      ? Object.entries(params).reduce((acc, [key, value]) => acc.replace(`$${key}`, value), to)
      : to;
    return (
      <a href={href} {...props}>
        {children}
      </a>
    );
  },
  useMatchRoute: () => (opts: { to: string; params?: MatchRouteParams; fuzzy?: boolean }) => {
    const matchKey = routeMatchKey(opts.to, opts.params);
    if (opts.fuzzy) {
      return matchedRouteFuzzy[matchKey] ?? matchedRoute[matchKey] ?? false;
    }
    return matchedRoute[matchKey] ?? false;
  },
}));

vi.mock("@/systems/daemon/hooks/use-daemon-connection-status", () => ({
  useDaemonConnectionStatus: () => mockConnectionStatus,
}));

vi.mock("@/systems/runtime/hooks/use-nav-counts", () => ({
  useNavCounts: () => ({
    counts: {},
    refresh: () => {},
    status: "connected",
  }),
}));

const mockTriggerAsync = vi.fn();
const mockToastError = vi.fn();

vi.mock("sonner", () => ({
  toast: {
    error: (...args: unknown[]) => mockToastError(...args),
  },
}));

vi.mock("@/systems/settings", () => ({
  useSettingsRestart: () => ({
    trigger: vi.fn(),
    triggerAsync: mockTriggerAsync,
    isTriggerPending: mockRestartFlags.isTriggerPending,
    isPolling: mockRestartFlags.isPolling,
    triggerError: null,
  }),
}));

let mockConnectionStatus: "connected" | "connecting" | "disconnected" | "error" = "connected";
const mockRestartFlags = {
  isTriggerPending: false,
  isPolling: false,
};

function renderSidebar(props: AppSidebarProps) {
  return render(
    <UIProvider reducedMotion="always">
      <AppSidebar {...props} />
    </UIProvider>
  );
}

function makeProps(overrides: Partial<AppSidebarProps> = {}): AppSidebarProps {
  const workspaces = [
    {
      id: "ws_alpha",
      root_dir: "/workspace/alpha",
      add_dirs: [] as string[],
      name: "alpha",
      created_at: "2026-04-06T10:00:00Z",
      updated_at: "2026-04-06T10:00:00Z",
    },
    {
      id: "ws_beta",
      root_dir: "/workspace/beta",
      add_dirs: [] as string[],
      name: "beta",
      created_at: "2026-04-06T10:00:00Z",
      updated_at: "2026-04-06T10:00:00Z",
    },
  ];

  return {
    collapsed: false,
    onCollapseChange,
    workspaces,
    activeWorkspaceId: "ws_alpha",
    activeWorkspace: workspaces[0],
    onSelectWorkspace,
    onAddWorkspace,
    agents: [],
    agentsLoading: false,
    agentsError: false,
    sessions: [],
    ...overrides,
  };
}

describe("AppSidebar", () => {
  beforeEach(() => {
    matchedRoute = {};
    matchedRouteFuzzy = {};
    mockConnectionStatus = "connected";
    mockRestartFlags.isTriggerPending = false;
    mockRestartFlags.isPolling = false;
    mockTriggerAsync.mockReset();
    mockTriggerAsync.mockResolvedValue({
      operation_id: "op-1",
      status: "started",
      active_session_count: 0,
    });
    mockToastError.mockReset();
    onSelectWorkspace.mockReset();
    onCollapseChange.mockReset();
    onAddWorkspace.mockReset();
  });

  describe("Should render the header slot", () => {
    it("Should render the workspace switcher inside the panel header", () => {
      renderSidebar(makeProps());
      const header = document.querySelector('[data-slot="sidebar-header"]');
      expect(header).not.toBeNull();
      expect(screen.getByTestId("workspace-switcher")).toBeInTheDocument();
      expect(screen.getByTestId("workspace-switcher-avatar")).toHaveTextContent("A");
      expect(screen.getByTestId("workspace-switcher-name")).toHaveTextContent("alpha");
      expect(screen.getByTestId("workspace-switcher-chevron")).toBeInTheDocument();
    });
  });

  describe("Should render the rail composition", () => {
    it("Should render the icon rail wrapper", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("icon-rail")).toBeInTheDocument();
    });

    it("Should render workspace squircle avatars with single-letter labels", () => {
      renderSidebar(makeProps());
      const alpha = screen.getByTestId("workspace-avatar-ws_alpha");
      expect(alpha).toHaveTextContent("A");
      expect(screen.getByTestId("workspace-avatar-ws_beta")).toHaveTextContent("B");
    });

    it("Should render the brand logo at the top of the rail", () => {
      renderSidebar(makeProps());
      const link = screen.getByTestId("app-logo");
      const logo = link.querySelector('[data-slot="logo"]');

      expect(logo).not.toBeNull();
      expect(logo).toHaveAttribute("data-variant", "symbol");
      expect(logo).toHaveAttribute("viewBox", "0 0 355 355");
    });

    it("Should link the app logo back to the dashboard", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("app-logo")).toHaveAttribute("href", "/");
      expect(screen.getByTestId("app-logo")).toHaveAttribute("aria-label", "Go to dashboard");
    });

    it("Should highlight the active workspace with an accent border", () => {
      renderSidebar(makeProps());
      const active = screen.getByTestId("workspace-avatar-ws_alpha");
      expect(active).toHaveAttribute("data-active", "true");
    });

    it("Should not render the deleted workspace-badge slot anywhere in the sidebar", () => {
      renderSidebar(makeProps());
      const sidebar = screen.getByTestId("app-sidebar");
      const wsBadgeQuery = `[class*="${"side"}__${"ws-badge"}"]`;
      expect(sidebar.querySelector(wsBadgeQuery)).toBeNull();
      expect(sidebar.querySelector('[data-slot="ws-badge"]')).toBeNull();
    });

    it("Should not render the rail-bottom connection LED (footer is single owner)", () => {
      renderSidebar(makeProps());
      const rail = screen.getByTestId("icon-rail");
      expect(rail.querySelector('[data-slot="connection-indicator"]')).toBeNull();
      const railConnectionQuery = `[class*="${"rail"}__${"connection"}"]`;
      expect(rail.querySelector(railConnectionQuery)).toBeNull();
    });

    it("Should not render the Bell/Cmd/Settings triplet at the rail bottom", () => {
      renderSidebar(makeProps());
      const rail = screen.getByTestId("icon-rail");
      expect(rail.querySelector('[data-testid="rail-bell"]')).toBeNull();
      expect(rail.querySelector('[data-testid="rail-cmd"]')).toBeNull();
      expect(rail.querySelector('[data-testid="rail-settings"]')).toBeNull();
    });

    it("Should select a workspace on avatar click", () => {
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("workspace-avatar-ws_beta"));
      expect(onSelectWorkspace).toHaveBeenCalledWith("ws_beta");
    });

    it("Should open the workspace setup flow from the add button", () => {
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("add-workspace-btn"));
      expect(onAddWorkspace).toHaveBeenCalledOnce();
    });

    it("Should keep the + add affordance when there are no workspaces", () => {
      renderSidebar(makeProps({ workspaces: [], activeWorkspaceId: null }));
      expect(screen.getByTestId("add-workspace-btn")).toBeInTheDocument();
      expect(screen.queryByTestId(/^workspace-avatar-/)).not.toBeInTheDocument();
    });
  });

  describe("Should render the agent tree", () => {
    it("Should render each agent as a flat link to /agents/$name", () => {
      renderSidebar(
        makeProps({
          agents: [
            { name: "coder", provider: "claude", prompt: "code" },
            { name: "writer", provider: "openai", prompt: "write" },
          ],
        })
      );

      const coderRow = screen.getByTestId("agent-row-coder");
      const writerRow = screen.getByTestId("agent-row-writer");
      expect(coderRow).toHaveAttribute("href", "/agents/coder");
      expect(writerRow).toHaveAttribute("href", "/agents/writer");
    });

    it("Should not render session counts, expand toggles, or new-session buttons inside the agent tree", () => {
      renderSidebar(
        makeProps({
          agents: [{ name: "coder", provider: "claude", prompt: "code" }],
          sessions: [
            {
              id: "s1",
              name: "Session 1",
              agent_name: "coder",
              provider: "claude",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
          ],
        })
      );

      expect(screen.queryByTestId("new-session-coder")).not.toBeInTheDocument();
      expect(screen.queryByTestId("agent-trigger-coder")).not.toBeInTheDocument();
      expect(screen.queryByTestId("session-row-s1")).not.toBeInTheDocument();
    });

    it("Should show a status dot only on agents with at least one active session", () => {
      renderSidebar(
        makeProps({
          agents: [
            { name: "coder", provider: "claude", prompt: "code" },
            { name: "writer", provider: "openai", prompt: "write" },
          ],
          sessions: [
            {
              id: "s_active",
              name: "Live",
              agent_name: "coder",
              provider: "claude",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
            {
              id: "s_done",
              name: "Done",
              agent_name: "writer",
              provider: "openai",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "stopped",
              updated_at: "2026-04-06T09:00:00Z",
              created_at: "2026-04-06T09:00:00Z",
            },
          ],
        })
      );

      expect(screen.getByTestId("agent-status-dot-coder")).toBeInTheDocument();
      expect(screen.queryByTestId("agent-status-dot-writer")).not.toBeInTheDocument();
    });

    it("Should highlight the agent row whose route is active (fuzzy: covers nested session route)", () => {
      matchedRouteFuzzy[routeMatchKey("/agents/$name", { name: "coder" })] = true;
      renderSidebar(
        makeProps({
          agents: [
            { name: "coder", provider: "claude", prompt: "code" },
            { name: "writer", provider: "openai", prompt: "write" },
          ],
        })
      );
      expect(screen.getByTestId("agent-row-coder")).toHaveAttribute("data-active", "true");
      expect(screen.getByTestId("agent-active-coder")).toBeInTheDocument();
      expect(screen.getByTestId("agent-row-writer")).toHaveAttribute("data-active", "false");
      expect(screen.queryByTestId("agent-active-writer")).not.toBeInTheDocument();
    });

    it("Should show the bootstrap hint when no agents are loaded", () => {
      renderSidebar(makeProps());
      expect(screen.getByText("Run `agh install` to bootstrap AGH")).toBeInTheDocument();
    });

    it("Should show the loading state when agents are loading", () => {
      renderSidebar(makeProps({ agentsLoading: true, agents: undefined }));
      expect(screen.getByText("Loading agents...")).toBeInTheDocument();
    });

    it("Should render categorized agents grouped by category_path", () => {
      matchedRouteFuzzy[routeMatchKey("/agents/$name", { name: "deals" })] = true;
      renderSidebar(
        makeProps({
          agents: [
            {
              name: "deals",
              provider: "claude",
              prompt: "deals",
              category_path: ["Marketing", "Sales"],
            },
            {
              name: "outreach",
              provider: "claude",
              prompt: "outreach",
              category_path: ["Operations"],
            },
            { name: "writer", provider: "openai", prompt: "write" },
          ],
          sessions: [
            {
              id: "s_active",
              name: "Live",
              agent_name: "deals",
              provider: "claude",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
          ],
        })
      );

      expect(screen.getByTestId("agent-category-Marketing")).toBeInTheDocument();
      expect(screen.getByTestId("agent-category-Marketing/Sales")).toBeInTheDocument();
      expect(screen.getByTestId("agent-category-Operations")).toBeInTheDocument();

      const dealsRow = screen.getByTestId("agent-row-deals");
      expect(dealsRow).toHaveAttribute("href", "/agents/deals");
      expect(dealsRow).toHaveAttribute("data-active", "true");
      expect(screen.getByTestId("agent-active-deals")).toBeInTheDocument();
      expect(screen.getByTestId("agent-status-dot-deals")).toBeInTheDocument();

      expect(screen.getByTestId("agent-row-writer")).toHaveAttribute("href", "/agents/writer");

      expect(screen.getByTestId("agent-category-Marketing")).toHaveAttribute(
        "data-expanded",
        "true"
      );
      expect(screen.getByTestId("agent-category-Marketing/Sales")).toHaveAttribute(
        "data-expanded",
        "true"
      );
      expect(screen.getByTestId("agent-category-Operations")).toHaveAttribute(
        "data-expanded",
        "false"
      );
    });
  });

  describe("Should render the AGENTS whole-tree live count", () => {
    it("Should render the live/total label when agents are present", () => {
      renderSidebar(
        makeProps({
          agents: [
            { name: "coder", provider: "claude", prompt: "code" },
            { name: "writer", provider: "openai", prompt: "write" },
            { name: "researcher", provider: "openai", prompt: "research" },
          ],
          sessions: [
            {
              id: "s_active_1",
              name: "Live coder",
              agent_name: "coder",
              provider: "claude",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
          ],
        })
      );
      expect(screen.getByTestId("agents-live-count")).toHaveTextContent("1/3 live");
    });

    it("Should render 2/2 when every agent has an active session", () => {
      renderSidebar(
        makeProps({
          agents: [
            { name: "coder", provider: "claude", prompt: "code" },
            { name: "writer", provider: "openai", prompt: "write" },
          ],
          sessions: [
            {
              id: "s1",
              name: "L1",
              agent_name: "coder",
              provider: "claude",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
            {
              id: "s2",
              name: "L2",
              agent_name: "writer",
              provider: "openai",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
          ],
        })
      );
      expect(screen.getByTestId("agents-live-count")).toHaveTextContent("2/2 live");
    });

    it("Should not render the count chip when there are no agents", () => {
      renderSidebar(makeProps({ agents: [] }));
      expect(screen.queryByTestId("agents-live-count")).not.toBeInTheDocument();
    });
  });

  describe("Should render the nav section structure", () => {
    it("Should render Agents, Operate, Catalog, and System section labels in order", () => {
      renderSidebar(makeProps());
      const labels = screen.getAllByTestId("sidebar-section-label");
      expect(labels.map(node => node.textContent)).toEqual([
        "Agents",
        "Operate",
        "Catalog",
        "System",
      ]);
    });

    it("Should use the canonical Inter UC eyebrow utility for section headers", () => {
      renderSidebar(makeProps());
      const label = screen.getAllByTestId("sidebar-section-label")[0];
      const classes = label.className.split(/\s+/);
      expect(classes).toContain("eyebrow");
      expect(classes).not.toContain("eyebrow-micro");
    });

    it("Should render Dashboard above Agents as the first nav item", () => {
      renderSidebar(makeProps());
      const nav = screen.getByTestId("sidebar-nav");
      const firstNavLink = nav.querySelector<HTMLAnchorElement>('a[data-testid^="nav-"]');
      expect(firstNavLink?.getAttribute("data-testid")).toBe("nav-dashboard");
      expect(firstNavLink).toHaveAttribute("href", "/");
    });

    it("Should render the grouped nav items in order (Operate → Catalog → System)", () => {
      renderSidebar(makeProps());
      const nav = screen.getByTestId("sidebar-nav");
      const navLinks = Array.from(
        nav.querySelectorAll<HTMLAnchorElement>('a[data-testid^="nav-"]')
      ).map(link => link.getAttribute("data-testid"));

      expect(navLinks).toEqual([
        "nav-dashboard",
        "nav-network",
        "nav-tasks",
        "nav-jobs",
        "nav-triggers",
        "nav-knowledge",
        "nav-skills",
        "nav-bridges",
        "nav-sandbox",
        "nav-settings",
      ]);
    });

    it.each([
      ["dashboard", "/"],
      ["network", "/network"],
      ["tasks", "/tasks"],
      ["jobs", "/jobs"],
      ["triggers", "/triggers"],
      ["knowledge", "/knowledge"],
      ["skills", "/skills"],
      ["bridges", "/bridges"],
      ["sandbox", "/sandbox"],
      ["settings", "/settings"],
    ])("Should render the %s nav item linking to %s", (testKey, href) => {
      renderSidebar(makeProps());
      expect(screen.getByTestId(`nav-${testKey}`)).toHaveAttribute("href", href);
    });

    it("Should render the Settings nav item inside the panel (not the footer)", () => {
      renderSidebar(makeProps());
      const nav = screen.getByTestId("sidebar-nav");
      const footer = screen.getByTestId("sidebar-footer");
      expect(nav).toContainElement(screen.getByTestId("nav-settings"));
      expect(footer).not.toContainElement(screen.queryByTestId("nav-settings"));
    });

    it.each([
      ["dashboard", "/"],
      ["network", "/network"],
      ["jobs", "/jobs"],
      ["triggers", "/triggers"],
      ["knowledge", "/knowledge"],
      ["skills", "/skills"],
      ["bridges", "/bridges"],
      ["sandbox", "/sandbox"],
    ])("Should render the 2px accent bar on active %s nav", (testKey, path) => {
      matchedRoute[path] = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId(`nav-active-${testKey}`)).toBeInTheDocument();
    });

    it("Should keep Tasks active for task detail and run detail deep links (fuzzy)", () => {
      matchedRouteFuzzy["/tasks"] = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId("nav-active-tasks")).toBeInTheDocument();
    });

    it("Should mark Settings active when the settings route matches (fuzzy)", () => {
      matchedRouteFuzzy["/settings"] = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId("nav-active-settings")).toBeInTheDocument();
    });

    it("Should not show active indicators when no route matches", () => {
      renderSidebar(makeProps());
      expect(screen.queryByTestId("nav-active-dashboard")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-tasks")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-jobs")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-triggers")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-settings")).not.toBeInTheDocument();
    });
  });

  describe("Should support the collapse trigger", () => {
    it("Should flip aria-expanded and notify onCollapseChange via the built-in trigger", () => {
      renderSidebar(makeProps());
      const trigger = screen.getByRole("button", { name: "Toggle sidebar" });
      expect(trigger).toHaveAttribute("aria-expanded", "true");

      fireEvent.click(trigger);
      expect(onCollapseChange).toHaveBeenCalledWith(true);
    });

    it("Should reflect a controlled collapsed state", () => {
      renderSidebar(makeProps({ collapsed: true }));
      const trigger = screen.getByRole("button", { name: "Toggle sidebar" });
      expect(trigger).toHaveAttribute("aria-expanded", "false");
    });
  });

  describe("Should render the footer connection LED", () => {
    it("Should render exactly one RuntimeConnectionIndicator in the footer (single owner)", () => {
      renderSidebar(makeProps());
      const indicators = document.querySelectorAll('[data-testid="runtime-connection-indicator"]');
      expect(indicators.length).toBe(1);
      const footer = screen.getByTestId("sidebar-footer");
      expect(footer).toContainElement(
        document.querySelector('[data-testid="runtime-connection-indicator"]') as HTMLElement
      );
    });

    it("Should render the success solid tone when the daemon is reachable", () => {
      mockConnectionStatus = "connected";
      renderSidebar(makeProps());
      const indicator = screen.getByTestId("runtime-connection-indicator");
      expect(indicator).toHaveAttribute("data-tone", "success");
      expect(indicator).toHaveAttribute("data-pulse", "false");
    });

    it("Should render the danger solid tone when the daemon is unreachable", () => {
      mockConnectionStatus = "disconnected";
      renderSidebar(makeProps());
      const indicator = screen.getByTestId("runtime-connection-indicator");
      expect(indicator).toHaveAttribute("data-tone", "danger");
      expect(indicator).toHaveAttribute("data-pulse", "false");
    });

    it("Should not render the daemon version badge in the footer", () => {
      renderSidebar(makeProps());
      expect(screen.queryByTestId("sidebar-version")).not.toBeInTheDocument();
    });
  });

  describe("Should render the restart daemon control", () => {
    it("Should mount the restart button in the footer with an accessible label", () => {
      renderSidebar(makeProps());
      const footer = screen.getByTestId("sidebar-footer");
      const button = screen.getByTestId("sidebar-restart-daemon");
      expect(footer).toContainElement(button);
      expect(button).toHaveAttribute("aria-label", "Restart daemon");
    });

    it("Should disable the restart button while the daemon is reconnecting", () => {
      mockConnectionStatus = "connecting";
      renderSidebar(makeProps());
      expect(screen.getByTestId("sidebar-restart-daemon")).toBeDisabled();
    });

    it("Should disable the restart button when the daemon is unreachable", () => {
      mockConnectionStatus = "disconnected";
      renderSidebar(makeProps());
      expect(screen.getByTestId("sidebar-restart-daemon")).toBeDisabled();
    });

    it("Should disable the restart button while a restart operation is polling", () => {
      mockRestartFlags.isPolling = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId("sidebar-restart-daemon")).toBeDisabled();
    });

    it("Should open the confirm dialog with the active-session impact line", () => {
      renderSidebar(
        makeProps({
          sessions: [
            {
              id: "s1",
              name: "L1",
              agent_name: "coder",
              provider: "claude",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
            {
              id: "s2",
              name: "L2",
              agent_name: "writer",
              provider: "openai",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "active",
              updated_at: "2026-04-06T10:00:00Z",
              created_at: "2026-04-06T10:00:00Z",
            },
          ],
        })
      );

      fireEvent.click(screen.getByTestId("sidebar-restart-daemon"));
      expect(screen.getByTestId("sidebar-restart-confirm-detail")).toHaveTextContent(
        "2 active sessions will be interrupted."
      );
    });

    it("Should describe a zero-session restart explicitly", () => {
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("sidebar-restart-daemon"));
      expect(screen.getByTestId("sidebar-restart-confirm-detail")).toHaveTextContent(
        "No active sessions will be interrupted."
      );
    });

    it("Should call triggerAsync when the user confirms the restart", async () => {
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("sidebar-restart-daemon"));
      fireEvent.click(screen.getByTestId("sidebar-restart-confirm-button"));
      await waitFor(() => expect(mockTriggerAsync).toHaveBeenCalledTimes(1));
      expect(mockToastError).not.toHaveBeenCalled();
    });

    it("Should toast an error when triggerAsync rejects", async () => {
      mockTriggerAsync.mockRejectedValueOnce(new Error("network"));
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("sidebar-restart-daemon"));
      await act(async () => {
        fireEvent.click(screen.getByTestId("sidebar-restart-confirm-button"));
      });
      await waitFor(() => expect(mockToastError).toHaveBeenCalledWith("Failed to restart daemon."));
    });

    it("Should dismiss the dialog without calling triggerAsync on cancel", () => {
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("sidebar-restart-daemon"));
      fireEvent.click(screen.getByTestId("sidebar-restart-cancel"));
      expect(mockTriggerAsync).not.toHaveBeenCalled();
    });
  });
});

describe("computeAgentsCount", () => {
  it("Should return zero counts for an empty agent list", () => {
    expect(computeAgentsCount([], [])).toEqual({ live: 0, total: 0 });
    expect(computeAgentsCount(undefined, undefined)).toEqual({ live: 0, total: 0 });
  });

  it("Should only count agents whose name has at least one active session", () => {
    const result = computeAgentsCount(
      [
        { name: "alpha", provider: "claude", prompt: "" },
        { name: "beta", provider: "claude", prompt: "" },
        { name: "gamma", provider: "claude", prompt: "" },
      ],
      [
        {
          id: "s1",
          name: "alpha-live",
          agent_name: "alpha",
          provider: "claude",
          workspace_id: "ws",
          workspace_path: "/",
          state: "active",
          updated_at: "2026-04-06T10:00:00Z",
          created_at: "2026-04-06T10:00:00Z",
        },
        {
          id: "s2",
          name: "beta-stopped",
          agent_name: "beta",
          provider: "claude",
          workspace_id: "ws",
          workspace_path: "/",
          state: "stopped",
          updated_at: "2026-04-06T10:00:00Z",
          created_at: "2026-04-06T10:00:00Z",
        },
      ]
    );
    expect(result).toEqual({ live: 1, total: 3 });
  });

  it("Should de-duplicate by agent name when an agent has multiple active sessions", () => {
    const result = computeAgentsCount(
      [{ name: "alpha", provider: "claude", prompt: "" }],
      [
        {
          id: "s1",
          name: "alpha-1",
          agent_name: "alpha",
          provider: "claude",
          workspace_id: "ws",
          workspace_path: "/",
          state: "active",
          updated_at: "2026-04-06T10:00:00Z",
          created_at: "2026-04-06T10:00:00Z",
        },
        {
          id: "s2",
          name: "alpha-2",
          agent_name: "alpha",
          provider: "claude",
          workspace_id: "ws",
          workspace_path: "/",
          state: "active",
          updated_at: "2026-04-06T10:00:00Z",
          created_at: "2026-04-06T10:00:00Z",
        },
      ]
    );
    expect(result).toEqual({ live: 1, total: 1 });
  });

  it("Should ignore sessions whose agent_name is not in the tree", () => {
    const result = computeAgentsCount(
      [{ name: "alpha", provider: "claude", prompt: "" }],
      [
        {
          id: "s1",
          name: "phantom",
          agent_name: "ghost",
          provider: "claude",
          workspace_id: "ws",
          workspace_path: "/",
          state: "active",
          updated_at: "2026-04-06T10:00:00Z",
          created_at: "2026-04-06T10:00:00Z",
        },
      ]
    );
    expect(result).toEqual({ live: 0, total: 1 });
  });
});
