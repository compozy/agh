import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";

import { AppSidebar, type AppSidebarProps } from "@/components/app-sidebar";

const onSelectWorkspace = vi.fn();
const onCollapseChange = vi.fn();
const onNewSession = vi.fn();
const onAddWorkspace = vi.fn();
let matchedRoute: Record<string, boolean> = {};
let matchedRouteFuzzy: Record<string, boolean> = {};

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: ReactNode;
    to: string;
    params?: Record<string, string>;
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
  useMatchRoute: () => (opts: { to: string; fuzzy?: boolean }) => {
    if (opts.fuzzy) {
      return matchedRouteFuzzy[opts.to] ?? matchedRoute[opts.to] ?? false;
    }
    return matchedRoute[opts.to] ?? false;
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
    activeWorkspace: workspaces[0],
    activeWorkspaceId: "ws_alpha",
    onSelectWorkspace,
    onAddWorkspace,
    health: { version: "0.1.0" },
    connectionStatus: "connected",
    agents: [],
    agentsLoading: false,
    agentsError: false,
    sessions: [],
    onNewSession,
    isCreatingSession: false,
    pendingSessionAgentName: null,
    pendingSessionWorkspaceId: null,
    ...overrides,
  };
}

describe("AppSidebar", () => {
  beforeEach(() => {
    matchedRoute = {};
    matchedRouteFuzzy = {};
    onSelectWorkspace.mockReset();
    onCollapseChange.mockReset();
    onNewSession.mockReset();
    onAddWorkspace.mockReset();
  });

  describe("Header", () => {
    it("surfaces the active workspace name", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("sidebar-workspace-name")).toHaveTextContent("alpha");
    });

    it("removes the non-functional sidebar search affordances", () => {
      renderSidebar(makeProps());
      expect(screen.queryByRole("button", { name: "Search" })).not.toBeInTheDocument();
      expect(screen.queryByText("Search…")).not.toBeInTheDocument();
    });

    it("no longer carries the wordmark (now owned by the global app shell)", () => {
      renderSidebar(makeProps());
      expect(screen.queryByTestId("sidebar-wordmark")).not.toBeInTheDocument();
      expect(screen.queryByTestId("sidebar-alpha-chip")).not.toBeInTheDocument();
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

    it("renders the app logo with accent background", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("app-logo").className).toContain("bg-[color:var(--color-accent)]");
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
      expect(active.className).toContain("border-[color:var(--color-accent)]");
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
      renderSidebar(
        makeProps({ workspaces: [], activeWorkspace: undefined, activeWorkspaceId: null })
      );
      expect(screen.getByTestId("add-workspace-btn")).toBeInTheDocument();
      expect(screen.queryByTestId(/^workspace-avatar-/)).not.toBeInTheDocument();
    });
  });

  describe("Agent List", () => {
    it("renders agents with session counts", () => {
      renderSidebar(
        makeProps({
          agents: [
            { name: "coder", provider: "claude", prompt: "code" },
            { name: "writer", provider: "openai", prompt: "write" },
          ],
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
            {
              id: "s2",
              name: "Session 2",
              agent_name: "coder",
              provider: "claude",
              workspace_id: "ws_alpha",
              workspace_path: "/workspace/alpha",
              state: "stopped",
              updated_at: "2026-04-06T09:00:00Z",
              created_at: "2026-04-06T09:00:00Z",
            },
          ],
        })
      );

      expect(screen.getByText("coder")).toBeInTheDocument();
      expect(screen.getByText("writer")).toBeInTheDocument();
      expect(screen.getByText("2")).toBeInTheDocument();
      expect(screen.getByText("0")).toBeInTheDocument();
    });

    it("shows bootstrap hint when no agents are loaded", () => {
      renderSidebar(makeProps());
      expect(screen.getByText("Run `agh install` to bootstrap AGH")).toBeInTheDocument();
    });

    it("shows the loading state when agents are loading", () => {
      renderSidebar(makeProps({ agentsLoading: true, agents: undefined }));
      expect(screen.getByText("Loading agents...")).toBeInTheDocument();
    });

    it("creates sessions via the agent's + button", () => {
      renderSidebar(
        makeProps({
          agents: [{ name: "claude-agent", provider: "anthropic", prompt: "You are helpful." }],
        })
      );

      fireEvent.click(screen.getByTestId("new-session-claude-agent"));
      expect(onNewSession).toHaveBeenCalledWith("claude-agent");
    });

    it("disables the new-session button when no workspace is active", () => {
      renderSidebar(
        makeProps({
          activeWorkspace: undefined,
          activeWorkspaceId: null,
          agents: [{ name: "claude-agent", provider: "anthropic", prompt: "help" }],
        })
      );

      expect(screen.getByTestId("new-session-claude-agent")).toBeDisabled();
    });

    it("shows a spinner and temporary starting row for the pending agent", () => {
      renderSidebar(
        makeProps({
          agents: [
            { name: "claude-agent", provider: "anthropic", prompt: "help" },
            { name: "general", provider: "openai", prompt: "general" },
          ],
          isCreatingSession: true,
          pendingSessionAgentName: "claude-agent",
          pendingSessionWorkspaceId: "ws_alpha",
        })
      );

      expect(screen.getByTestId("new-session-claude-agent")).toBeDisabled();
      expect(screen.getByTestId("new-session-general")).toBeDisabled();
      expect(screen.getByTestId("new-session-spinner-claude-agent")).toBeInTheDocument();
      expect(screen.queryByTestId("new-session-spinner-general")).not.toBeInTheDocument();
      expect(screen.getByTestId("pending-session-row-claude-agent")).toHaveTextContent(
        "starting..."
      );
    });

    it("does not render the temporary row when the pending session belongs to another workspace", () => {
      renderSidebar(
        makeProps({
          agents: [{ name: "claude-agent", provider: "anthropic", prompt: "help" }],
          isCreatingSession: true,
          pendingSessionAgentName: "claude-agent",
          pendingSessionWorkspaceId: "ws_beta",
        })
      );

      expect(screen.queryByTestId("pending-session-row-claude-agent")).not.toBeInTheDocument();
    });
  });

  describe("Nav — Workspace section", () => {
    it("renders a Workspace section label", () => {
      renderSidebar(makeProps());
      const labels = screen.getAllByTestId("sidebar-section-label");
      expect(labels.map(node => node.textContent)).toEqual(
        expect.arrayContaining(["Agents", "Workspace"])
      );
    });

    it("uses JetBrains mono 11px uppercase for section headers", () => {
      renderSidebar(makeProps());
      const label = screen.getAllByTestId("sidebar-section-label")[0];
      expect(label.className).toContain("font-mono");
      expect(label.className).toContain("text-[11px]");
      expect(label.className).toContain("uppercase");
    });

    it("renders the workspace navigation in the expected order", () => {
      renderSidebar(makeProps());
      const nav = screen.getByTestId("sidebar-nav");
      const workspaceLinks = Array.from(
        nav.querySelectorAll<HTMLAnchorElement>('a[data-testid^="nav-"]')
      )
        .map(link => link.getAttribute("data-testid"))
        .filter((testId): testId is string => testId !== null && testId !== "nav-settings");

      expect(workspaceLinks).toEqual([
        "nav-network",
        "nav-tasks",
        "nav-bridges",
        "nav-jobs",
        "nav-triggers",
        "nav-knowledge",
        "nav-skills",
      ]);
    });

    it.each([
      ["network", "/network"],
      ["tasks", "/tasks"],
      ["bridges", "/bridges"],
      ["jobs", "/jobs"],
      ["triggers", "/triggers"],
      ["knowledge", "/knowledge"],
      ["skills", "/skills"],
    ])("renders %s nav item linking to %s", (testKey, href) => {
      renderSidebar(makeProps());
      expect(screen.getByTestId(`nav-${testKey}`)).toHaveAttribute("href", href);
    });

    it("renders the Settings nav item in the footer", () => {
      renderSidebar(makeProps());
      expect(screen.getByTestId("nav-settings")).toHaveAttribute("href", "/settings");
    });

    it.each([
      ["network", "/network"],
      ["bridges", "/bridges"],
      ["jobs", "/jobs"],
      ["triggers", "/triggers"],
      ["knowledge", "/knowledge"],
      ["skills", "/skills"],
    ])("renders 3px accent bar on active %s nav", (testKey, path) => {
      matchedRoute[path] = true;
      renderSidebar(makeProps());
      const indicator = screen.getByTestId(`nav-active-${testKey}`);
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("keeps Tasks nav active for task detail and run detail deep links (fuzzy match)", () => {
      matchedRouteFuzzy["/tasks"] = true;
      renderSidebar(makeProps());
      const indicator = screen.getByTestId("nav-active-tasks");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("marks Settings active when the settings route matches", () => {
      matchedRouteFuzzy["/settings"] = true;
      renderSidebar(makeProps());
      const indicator = screen.getByTestId("nav-active-settings");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("does not show active indicators when no route matches", () => {
      renderSidebar(makeProps());
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
