import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { MouseEvent as ReactMouseEvent, ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import {
  resetUserHomeDirStore,
  useUserHomeDirStore,
} from "@/systems/workspace/hooks/use-user-home-dir-store";

import { AppSidebar, type AppSidebarProps } from "../app-sidebar";

const onSelectWorkspace = vi.fn();
const onCollapseChange = vi.fn();
const onAddWorkspace = vi.fn();
let matchedRoute: Record<string, boolean> = {};
let matchedRouteFuzzy: Record<string, boolean> = {};
let linkStates: Record<string, unknown> = {};

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
    activeOptions,
    activeProps,
    children,
    className,
    inactiveProps,
    to,
    params,
    onClick,
    state,
    ...props
  }: {
    activeOptions?: { exact?: boolean; includeSearch?: boolean };
    activeProps?: Record<string, string>;
    children: ReactNode | ((state: { isActive: boolean; isTransitioning: boolean }) => ReactNode);
    className?: string;
    inactiveProps?: Record<string, string>;
    to: string;
    params?: MatchRouteParams;
    onClick?: (event: ReactMouseEvent<HTMLAnchorElement>) => void;
    state?: unknown;
    [key: string]: unknown;
  }) => {
    const href = params
      ? Object.entries(params).reduce((acc, [key, value]) => acc.replace(`$${key}`, value), to)
      : to;
    linkStates[href] = state;
    const fuzzy = !(activeOptions?.exact ?? false);
    const matchKey = routeMatchKey(to, params);
    const isActive = fuzzy
      ? (matchedRouteFuzzy[matchKey] ?? matchedRoute[matchKey] ?? false)
      : (matchedRoute[matchKey] ?? false);
    const { className: stateClassName, ...stateProps } =
      (isActive ? activeProps : inactiveProps) ?? {};
    const resolvedChildren =
      typeof children === "function" ? children({ isActive, isTransitioning: false }) : children;
    return (
      <a
        href={href}
        onClick={event => {
          event.preventDefault();
          onClick?.(event);
        }}
        {...props}
        {...stateProps}
        aria-current={isActive ? "page" : undefined}
        className={[className, stateClassName].filter(Boolean).join(" ") || undefined}
      >
        {resolvedChildren}
      </a>
    );
  },
}));

vi.mock("@/systems/status/hooks/use-daemon-connection-status", () => ({
  useDaemonConnectionStatus: () => mockConnectionStatus,
}));

vi.mock("@/systems/status", () => ({
  useDaemonHealth: () => ({
    connectionStatus: mockConnectionStatus,
    health: { status: "ok" },
    isInitialLoading: false,
  }),
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
    agentsCount: undefined,
    activeSessionCount: 0,
    workspaceSessionActivity: {},
    ...overrides,
  };
}

describe("AppSidebar", () => {
  beforeEach(() => {
    resetUserHomeDirStore();
    matchedRoute = {};
    matchedRouteFuzzy = {};
    linkStates = {};
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
      const switcher = screen.getByTestId("workspace-switcher");
      expect(switcher).toBeInTheDocument();
      expect(switcher.tagName).toBe("BUTTON");
      expect(switcher).toHaveAttribute("aria-expanded", "false");
      expect(screen.getByTestId("workspace-switcher-avatar")).toHaveTextContent("A");
      expect(screen.getByTestId("workspace-switcher-name")).toHaveTextContent("alpha");
      expect(screen.getByTestId("workspace-switcher-chevron")).toBeInTheDocument();
    });

    it("Should select a workspace from the header command popover", async () => {
      const user = userEvent.setup();
      renderSidebar(makeProps());
      await user.click(screen.getByTestId("workspace-switcher"));
      await user.click(screen.getByTestId("workspace-command-item-ws_beta"));
      expect(onSelectWorkspace).toHaveBeenCalledWith("ws_beta");
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

    it("Should expose the exact inactive-workspace session count as an accessible return link", () => {
      renderSidebar(
        makeProps({
          workspaceSessionActivity: {
            ws_beta: {
              state: "ready",
              count: 2,
              returnTarget: {
                sessionId: "sess_reconcile",
                agentName: "codex-agent",
                title: "Reconcile payout ledger",
              },
            },
          },
        })
      );

      const returnLink = screen.getByRole("link", {
        name: "Return to beta: 2 active sessions. Latest: Reconcile payout ledger",
      });
      expect(returnLink).toHaveAttribute("href", "/agents/codex-agent/sessions/sess_reconcile");
      expect(screen.getByTestId("workspace-active-session-count-ws_beta")).toHaveTextContent("2");
      expect(linkStates["/agents/codex-agent/sessions/sess_reconcile"]).toEqual({
        sessionReturn: { sessionId: "sess_reconcile", workspaceId: "ws_beta" },
      });

      fireEvent.click(returnLink);
      expect(onSelectWorkspace).not.toHaveBeenCalled();
    });

    it("Should not render an activity count on the selected workspace", () => {
      renderSidebar(
        makeProps({
          workspaceSessionActivity: {
            ws_alpha: {
              state: "ready",
              count: 3,
              returnTarget: {
                sessionId: "sess_selected",
                agentName: "codex-agent",
                title: "Selected workspace task",
              },
            },
          },
        })
      );

      expect(
        screen.queryByTestId("workspace-active-session-count-ws_alpha")
      ).not.toBeInTheDocument();
      expect(screen.getByTestId("workspace-avatar-ws_alpha").tagName).toBe("BUTTON");
    });

    it("Should expose unavailable workspace activity without fabricating a zero count", () => {
      renderSidebar(
        makeProps({
          workspaceSessionActivity: {
            ws_beta: { state: "error", message: "session catalog unavailable" },
          },
        })
      );

      expect(
        screen.getByRole("button", { name: "Workspace: beta, session activity unavailable" })
      ).toBeInTheDocument();
      expect(screen.getByTestId("workspace-session-activity-error-ws_beta")).toHaveAttribute(
        "title",
        "session catalog unavailable"
      );
      expect(
        screen.queryByTestId("workspace-active-session-count-ws_beta")
      ).not.toBeInTheDocument();
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

  describe("Should distinguish the home/global workspace", () => {
    function railAvatarIds(): string[] {
      const rail = screen.getByTestId("icon-rail");
      return Array.from(
        rail.querySelectorAll<HTMLElement>('[data-testid^="workspace-avatar-"]')
      ).map(node => node.getAttribute("data-testid") ?? "");
    }

    it("Should render the workspace matching user_home_dir first, ahead of project workspaces", () => {
      // ws_beta lives at /workspace/beta; mark it as the home workspace.
      useUserHomeDirStore.getState().setUserHomeDir("/workspace/beta");
      renderSidebar(makeProps());
      expect(railAvatarIds()).toEqual(["workspace-avatar-ws_beta", "workspace-avatar-ws_alpha"]);
    });

    it("Should render a home icon (not a letter) for the home workspace", () => {
      useUserHomeDirStore.getState().setUserHomeDir("/workspace/beta");
      renderSidebar(makeProps());
      const home = screen.getByTestId("workspace-avatar-ws_beta");
      expect(home).toHaveAttribute("data-home", "true");
      // The home avatar carries the lucide home glyph instead of the "B" letter.
      expect(home.querySelector("svg")).not.toBeNull();
      expect(home).not.toHaveTextContent("B");
      expect(home).toHaveAccessibleName("Home workspace: beta");
    });

    it("Should render a divider between the home workspace and the project workspaces", () => {
      useUserHomeDirStore.getState().setUserHomeDir("/workspace/beta");
      renderSidebar(makeProps());
      const rail = screen.getByTestId("icon-rail");
      const divider = screen.getByTestId("rail-home-divider");
      expect(rail).toContainElement(divider);
    });

    it("Should keep letter avatars and skip the divider when no workspace matches user_home_dir", () => {
      useUserHomeDirStore.getState().setUserHomeDir("/somewhere/else");
      renderSidebar(makeProps());
      expect(railAvatarIds()).toEqual(["workspace-avatar-ws_alpha", "workspace-avatar-ws_beta"]);
      expect(screen.getByTestId("workspace-avatar-ws_alpha")).toHaveTextContent("A");
      expect(screen.getByTestId("workspace-avatar-ws_alpha")).not.toHaveAttribute("data-home");
      expect(screen.queryByTestId("rail-home-divider")).not.toBeInTheDocument();
    });
  });

  describe("Should place Agents as the first Operate nav item", () => {
    it("Should link Agents to /agents with fuzzy match coverage", () => {
      matchedRouteFuzzy["/agents"] = true;
      renderSidebar(makeProps());
      const agentsNav = screen.getByTestId("nav-agents");
      expect(agentsNav).toHaveAttribute("href", "/agents");
      expect(agentsNav).toHaveAttribute("data-active", "true");
      expect(screen.getByTestId("nav-active-agents")).toBeInTheDocument();
    });

    it("Should not render the deleted agent tree, section label, or sidebar create button", () => {
      renderSidebar(makeProps({ agentsCount: { live: 1, total: 3 } }));
      expect(screen.queryByTestId("agent-row-coder")).not.toBeInTheDocument();
      expect(screen.queryByTestId("sidebar-create-agent")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Create agent" })).not.toBeInTheDocument();
      expect(screen.queryByText("Run `agh install` to bootstrap AGH")).not.toBeInTheDocument();
      expect(screen.queryByText("Loading agents...")).not.toBeInTheDocument();
      const labels = screen.getAllByTestId("sidebar-section-label").map(node => node.textContent);
      expect(labels).toEqual(["Operate", "Catalog", "System"]);
    });
  });

  describe("Should render the Agents Operate badge from backend catalog facets", () => {
    it("Should render the exact live/total badge supplied by the shell", () => {
      renderSidebar(makeProps({ agentsCount: { live: 1, total: 3 } }));
      expect(screen.getByTestId("agents-live-count")).toHaveTextContent("1/3");
      expect(screen.getByTestId("nav-agents")).toContainElement(
        screen.getByTestId("agents-live-count")
      );
    });

    it("Should render 2/2 when every agent has a running session", () => {
      renderSidebar(makeProps({ agentsCount: { live: 2, total: 2 } }));
      expect(screen.getByTestId("agents-live-count")).toHaveTextContent("2/2");
    });

    it("Should render a zero-live catalog total", () => {
      renderSidebar(makeProps({ agentsCount: { live: 0, total: 1 } }));
      expect(screen.getByTestId("agents-live-count")).toHaveTextContent("0/1");
    });

    it("Should not render the count chip when there are no agents", () => {
      renderSidebar(makeProps({ agentsCount: { live: 0, total: 0 } }));
      expect(screen.queryByTestId("agents-live-count")).not.toBeInTheDocument();
    });
  });

  describe("Should render the nav section structure", () => {
    it("Should render Operate, Catalog, and System section labels in order", () => {
      renderSidebar(makeProps());
      const labels = screen.getAllByTestId("sidebar-section-label");
      expect(labels.map(node => node.textContent)).toEqual(["Operate", "Catalog", "System"]);
    });

    it("Should use the canonical Inter UC eyebrow utility for section headers", () => {
      renderSidebar(makeProps());
      const label = screen.getAllByTestId("sidebar-section-label")[0];
      const classes = label.className.split(/\s+/);
      expect(classes).toContain("eyebrow");
      expect(classes).not.toContain("eyebrow-micro");
    });

    it("Should render Dashboard above Operate as the first nav item", () => {
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
        "nav-agents",
        "nav-network",
        "nav-tasks",
        "nav-loops",
        "nav-jobs",
        "nav-triggers",
        "nav-marketplace",
        "nav-bridges",
        "nav-knowledge",
        "nav-sandbox",
        "nav-vault",
        "nav-settings",
      ]);
    });

    it.each([
      ["dashboard", "/"],
      ["agents", "/agents"],
      ["network", "/network"],
      ["tasks", "/tasks"],
      ["loops", "/loops"],
      ["jobs", "/jobs"],
      ["triggers", "/triggers"],
      ["marketplace", "/marketplace"],
      ["knowledge", "/knowledge"],
      ["bridges", "/bridges"],
      ["sandbox", "/sandbox"],
      ["vault", "/vault"],
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
      ["marketplace", "/marketplace"],
      ["knowledge", "/knowledge"],
      ["bridges", "/bridges"],
      ["sandbox", "/sandbox"],
      ["vault", "/vault"],
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

    it("Should keep Marketplace active for entry detail routes (fuzzy)", () => {
      matchedRouteFuzzy["/marketplace"] = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId("nav-active-marketplace")).toBeInTheDocument();
    });

    it("Should keep Loops active for loop detail and editor deep links (fuzzy)", () => {
      matchedRouteFuzzy["/loops"] = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId("nav-active-loops")).toBeInTheDocument();
    });

    it("Should keep Jobs active for job detail deep links (fuzzy)", () => {
      matchedRouteFuzzy["/jobs"] = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId("nav-active-jobs")).toBeInTheDocument();
    });

    it("Should keep Triggers active for trigger detail deep links (fuzzy)", () => {
      matchedRouteFuzzy["/triggers"] = true;
      renderSidebar(makeProps());
      expect(screen.getByTestId("nav-active-triggers")).toBeInTheDocument();
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
      renderSidebar(makeProps({ activeSessionCount: 2 }));

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
