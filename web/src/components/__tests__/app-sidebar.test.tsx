import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { AppSidebar, type AppSidebarProps } from "@/components/app-sidebar";

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
    onSelectWorkspace,
    onAddWorkspace,
    health: { version: "0.1.0" },
    connectionStatus: "connected",
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
    onSelectWorkspace.mockReset();
    onCollapseChange.mockReset();
    onAddWorkspace.mockReset();
  });

  describe("Header", () => {
    it("does not render a sidebar header slot — workspace identity lives in the rail", () => {
      renderSidebar(makeProps());
      expect(screen.queryByTestId("sidebar-workspace-name")).not.toBeInTheDocument();
      expect(document.querySelector('[data-slot="sidebar-header"]')).toBeNull();
    });
  });

  describe("Workspace Rail", () => {
    it("renders the icon rail", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("icon-rail")).toBeInTheDocument();
    });

    it("renders workspace circle avatars with single-letter labels", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("workspace-avatar-ws_alpha")).toHaveTextContent("A");
      expect(screen.getByTestId("workspace-avatar-ws_beta")).toHaveTextContent("B");
    });

    it("renders the shared app symbol logo", () => {
      renderSidebar(makeProps());
      const link = screen.getByTestId("app-logo");
      const logo = link.querySelector('[data-slot="logo"]');

      expect(logo).not.toBeNull();
      expect(logo).toHaveAttribute("data-variant", "symbol");
      expect(logo).toHaveAttribute("viewBox", "0 0 355 355");
    });

    it("links the app logo back to the dashboard", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("app-logo")).toHaveAttribute("href", "/");
      expect(screen.getByTestId("app-logo")).toHaveAttribute("aria-label", "Go to dashboard");
      expect(screen.getByTestId("app-logo").className).toContain("focus-visible:ring-2");
    });

    it("highlights active workspace with accent border", () => {
      renderSidebar(makeProps());
      const active = screen.getByTestId("workspace-avatar-ws_alpha");
      expect(active).toHaveAttribute("data-active", "true");
      expect(active.className).toContain("border-accent");
    });

    it("does not highlight inactive workspaces", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("workspace-avatar-ws_beta")).toHaveAttribute(
        "data-active",
        "false"
      );
    });

    it("selects a workspace on click", () => {
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("workspace-avatar-ws_beta"));
      expect(onSelectWorkspace).toHaveBeenCalledWith("ws_beta");
    });

    it("opens workspace setup from the add button", () => {
      renderSidebar(makeProps());
      fireEvent.click(screen.getByTestId("add-workspace-btn"));
      expect(onAddWorkspace).toHaveBeenCalledOnce();
    });

    it("still renders the + affordance when there are no workspaces", () => {
      renderSidebar(makeProps({ workspaces: [], activeWorkspaceId: null }));
      expect(screen.getByTestId("add-workspace-btn")).toBeInTheDocument();
      expect(screen.queryByTestId(/^workspace-avatar-/)).not.toBeInTheDocument();
    });
  });

  describe("Agent List", () => {
    it("renders each agent as a flat link to /agents/$name", () => {
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

    it("does not render session counts, expand toggles, or new-session buttons in the sidebar", () => {
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

    it("shows a status dot only on agents that have at least one active session", () => {
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

    it("highlights the agent row whose route is active (fuzzy: covers nested session route)", () => {
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

    it("shows bootstrap hint when no agents are loaded", () => {
      renderSidebar(makeProps());
      expect(screen.getByText("Run `agh install` to bootstrap AGH")).toBeInTheDocument();
    });

    it("shows the loading state when agents are loading", () => {
      renderSidebar(makeProps({ agentsLoading: true, agents: undefined }));
      expect(screen.getByText("Loading agents...")).toBeInTheDocument();
    });

    it("renders categorized agents grouped by category_path through the real AppSidebar", () => {
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

      // Active leaf preserves existing testIds and surfaces the active indicator + status dot.
      const dealsRow = screen.getByTestId("agent-row-deals");
      expect(dealsRow).toHaveAttribute("href", "/agents/deals");
      expect(dealsRow).toHaveAttribute("data-active", "true");
      expect(screen.getByTestId("agent-active-deals")).toBeInTheDocument();
      expect(screen.getByTestId("agent-status-dot-deals")).toBeInTheDocument();

      // Root-level agent still renders alongside categories without an Uncategorized folder.
      expect(screen.getByTestId("agent-row-writer")).toHaveAttribute("href", "/agents/writer");

      // Active agent's ancestor folders expand on initial render; unrelated branches do not.
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

  describe("Nav — Section structure", () => {
    it("renders Dashboard, Agents, Operate, Catalog, and System section labels in order", () => {
      renderSidebar(makeProps());
      const labels = screen.getAllByTestId("sidebar-section-label");
      expect(labels.map(node => node.textContent)).toEqual([
        "Agents",
        "Operate",
        "Catalog",
        "System",
      ]);
    });

    it("uses JetBrains mono 9px uppercase for section headers", () => {
      renderSidebar(makeProps());
      const label = screen.getAllByTestId("sidebar-section-label")[0];
      expect(label.className).toContain("font-mono");
      expect(label.className).toContain("text-micro");
      expect(label.className).toContain("uppercase");
      expect(label.className).toContain("tracking-mono");
    });

    it("renders Dashboard above the Agents section as the first nav item", () => {
      renderSidebar(makeProps());
      const nav = screen.getByTestId("sidebar-nav");
      const firstNavLink = nav.querySelector<HTMLAnchorElement>('a[data-testid^="nav-"]');
      expect(firstNavLink?.getAttribute("data-testid")).toBe("nav-dashboard");
      expect(firstNavLink).toHaveAttribute("href", "/");
    });

    it("renders nav items in the new grouped order (Operate → Catalog → System)", () => {
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
    ])("renders %s nav item linking to %s", (testKey, href) => {
      renderSidebar(makeProps());
      expect(screen.getByTestId(`nav-${testKey}`)).toHaveAttribute("href", href);
    });

    it("renders the Settings nav item inside the panel (not the footer)", () => {
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
    ])("renders 2px accent bar on active %s nav", (testKey, path) => {
      matchedRoute[path] = true;
      renderSidebar(makeProps());
      const indicator = screen.getByTestId(`nav-active-${testKey}`);
      expect(indicator.className).toContain("w-[2px]");
      expect(indicator.className).toContain("bg-accent");
    });

    it("keeps Tasks nav active for task detail and run detail deep links (fuzzy match)", () => {
      matchedRouteFuzzy["/tasks"] = true;
      renderSidebar(makeProps());
      const indicator = screen.getByTestId("nav-active-tasks");
      expect(indicator.className).toContain("w-[2px]");
      expect(indicator.className).toContain("bg-accent");
    });

    it("marks Settings active when the settings route matches (fuzzy)", () => {
      matchedRouteFuzzy["/settings"] = true;
      renderSidebar(makeProps());
      const indicator = screen.getByTestId("nav-active-settings");
      expect(indicator.className).toContain("w-[2px]");
      expect(indicator.className).toContain("bg-accent");
    });

    it("does not show active indicators when no route matches", () => {
      renderSidebar(makeProps());
      expect(screen.queryByTestId("nav-active-dashboard")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-tasks")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-jobs")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-triggers")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-settings")).not.toBeInTheDocument();
    });
  });

  describe("Collapse", () => {
    it("flips aria-expanded and notifies onCollapseChange via the built-in trigger", () => {
      renderSidebar(makeProps());
      const trigger = screen.getByRole("button", { name: "Toggle sidebar" });
      expect(trigger).toHaveAttribute("aria-expanded", "true");

      fireEvent.click(trigger);
      expect(onCollapseChange).toHaveBeenCalledWith(true);
    });

    it("reflects a collapsed controlled state", () => {
      renderSidebar(makeProps({ collapsed: true }));
      const trigger = screen.getByRole("button", { name: "Toggle sidebar" });
      expect(trigger).toHaveAttribute("aria-expanded", "false");
    });
  });

  describe("Footer", () => {
    it("shows the connection indicator label", () => {
      renderSidebar(makeProps());
      expect(screen.getByText("Connected")).toBeInTheDocument();
    });

    it("shows version from daemon health", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("sidebar-version")).toHaveTextContent("v0.1.0");
    });

    it("reflects a disconnected daemon", () => {
      renderSidebar(makeProps({ connectionStatus: "disconnected" }));
      expect(screen.getByText("Disconnected")).toBeInTheDocument();
    });

    it("omits the version when health is missing", () => {
      renderSidebar(makeProps({ health: undefined }));
      expect(screen.queryByTestId("sidebar-version")).not.toBeInTheDocument();
    });
  });
});
