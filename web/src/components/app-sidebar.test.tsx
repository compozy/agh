import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AppSidebar, type AppSidebarProps } from "@/components/app-sidebar";

const onSelectWorkspace = vi.fn();
const onToggleCollapsed = vi.fn();
const onNewSession = vi.fn();
const onAddWorkspace = vi.fn();
let matchedRoute: Record<string, boolean> = {};
let matchedRouteFuzzy: Record<string, boolean> = {};

vi.mock("lucide-react", () => ({
  Book: () => <span data-testid="icon-book">book</span>,
  Bot: () => <span>bot</span>,
  ChevronRight: () => <span>chevron</span>,
  ListChecks: () => <span data-testid="icon-list-checks">list-checks</span>,
  Loader2: () => <span>loader</span>,
  Network: () => <span data-testid="icon-network">network</span>,
  PanelLeftClose: () => <span>panel-close</span>,
  PanelLeftOpen: () => <span>panel-open</span>,
  Plus: () => <span>plus</span>,
  Search: () => <span>search</span>,
  Settings: () => <span>settings</span>,
  Terminal: () => <span data-testid="icon-terminal">terminal</span>,
  Waypoints: () => <span data-testid="icon-waypoints">waypoints</span>,
  Wrench: () => <span data-testid="icon-wrench">wrench</span>,
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    to,
    ...props
  }: {
    children: ReactNode;
    to: string;
    [key: string]: unknown;
  }) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
  useMatchRoute: () => (opts: { to: string; fuzzy?: boolean }) => {
    if (opts.fuzzy) {
      return matchedRouteFuzzy[opts.to] ?? matchedRoute[opts.to] ?? false;
    }
    return matchedRoute[opts.to] ?? false;
  },
}));

vi.mock("@/components/ui/collapsible", () => ({
  Collapsible: ({
    children,
    className,
    defaultOpen = true,
  }: {
    children: ReactNode;
    className?: string;
    defaultOpen?: boolean;
  }) => (
    <div className={className} data-state={defaultOpen ? "open" : "closed"}>
      {children}
    </div>
  ),
  CollapsibleContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  CollapsibleTrigger: ({ children, className }: { children: ReactNode; className?: string }) => (
    <button className={className}>{children}</button>
  ),
}));

vi.mock("@agh/ui", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
  Kbd: ({ children }: { children: ReactNode }) => <kbd>{children}</kbd>,
}));

vi.mock("@/systems/agent", () => ({
  AgentIcon: ({ provider }: { provider: string }) => (
    <span data-testid={`agent-icon-${provider}`} />
  ),
}));

vi.mock("@/systems/daemon", () => ({
  ConnectionStatus: ({ status }: { status: string }) => (
    <span data-testid="connection-status">{status}</span>
  ),
}));

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
    onToggleCollapsed,
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
    ...overrides,
  };
}

describe("AppSidebar", () => {
  beforeEach(() => {
    matchedRoute = {};
    matchedRouteFuzzy = {};
    onSelectWorkspace.mockReset();
    onToggleCollapsed.mockReset();
    onNewSession.mockReset();
    onAddWorkspace.mockReset();
  });

  describe("Icon Rail", () => {
    it("renders the icon rail", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("icon-rail")).toBeInTheDocument();
    });

    it("renders workspace circle avatars with single-letter labels", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("workspace-avatar-ws_alpha")).toHaveTextContent("A");
      expect(screen.getByTestId("workspace-avatar-ws_beta")).toHaveTextContent("B");
    });

    it("renders app logo with accent background", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("app-logo").className).toContain("bg-[color:var(--color-accent)]");
    });

    it("highlights active workspace with accent border", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("workspace-avatar-ws_alpha").className).toContain(
        "border-[color:var(--color-accent)]"
      );
    });

    it("does not highlight inactive workspaces", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("workspace-avatar-ws_beta").className).not.toContain(
        "border-[color:var(--color-accent)]"
      );
    });

    it("selects a workspace on click", () => {
      render(<AppSidebar {...makeProps()} />);
      fireEvent.click(screen.getByTestId("workspace-avatar-ws_beta"));
      expect(onSelectWorkspace).toHaveBeenCalledWith("ws_beta");
    });

    it("opens workspace setup from the add button", () => {
      render(<AppSidebar {...makeProps()} />);
      fireEvent.click(screen.getByTestId("add-workspace-btn"));
      expect(onAddWorkspace).toHaveBeenCalledOnce();
    });
  });

  describe("Agent List", () => {
    it("renders agents with session counts", () => {
      render(
        <AppSidebar
          {...makeProps({
            agents: [
              { name: "coder", provider: "claude", prompt: "code" },
              { name: "writer", provider: "openai", prompt: "write" },
            ],
            sessions: [
              {
                id: "s1",
                name: "Session 1",
                agent_name: "coder",
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
                workspace_id: "ws_alpha",
                workspace_path: "/workspace/alpha",
                state: "stopped",
                updated_at: "2026-04-06T09:00:00Z",
                created_at: "2026-04-06T09:00:00Z",
              },
            ],
          })}
        />
      );

      expect(screen.getByText("coder")).toBeInTheDocument();
      expect(screen.getByText("writer")).toBeInTheDocument();
      expect(screen.getByText("2")).toBeInTheDocument();
      expect(screen.getByText("0")).toBeInTheDocument();
    });

    it("shows bootstrap hint when no agents are loaded", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByText("Run `agh install` to bootstrap AGH")).toBeInTheDocument();
    });

    it("starts agents with zero sessions collapsed by default", () => {
      render(
        <AppSidebar
          {...makeProps({
            agents: [{ name: "writer", provider: "openai", prompt: "write" }],
          })}
        />
      );

      expect(screen.getByText("writer").closest('[data-state="closed"]')).toBeInTheDocument();
    });

    it("creates sessions in the selected workspace", () => {
      render(
        <AppSidebar
          {...makeProps({
            agents: [{ name: "claude-agent", provider: "anthropic", prompt: "You are helpful." }],
          })}
        />
      );

      fireEvent.click(screen.getByTestId("new-session-claude-agent"));
      expect(onNewSession).toHaveBeenCalledWith("claude-agent");
    });
  });

  describe("Navigation", () => {
    it("renders the outer sidebar with a right border", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("app-sidebar").className).toContain("border-r");
    });

    it("renders Tasks nav item linking to /tasks", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-tasks")).toHaveAttribute("href", "/tasks");
    });

    it("renders Knowledge nav item linking to /knowledge", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-knowledge")).toHaveAttribute("href", "/knowledge");
    });

    it("renders Bridges nav item linking to /bridges", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-bridges")).toHaveAttribute("href", "/bridges");
    });

    it("renders Network nav item linking to /network", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-network")).toHaveAttribute("href", "/network");
    });

    it("renders Automation nav item linking to /automation", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-automation")).toHaveAttribute("href", "/automation");
    });

    it("renders Skills nav item linking to /skills", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-skills")).toHaveAttribute("href", "/skills");
    });

    it("renders Settings nav item linking to /settings", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-settings")).toHaveAttribute("href", "/settings");
    });

    it("shows active indicator on active Settings nav", () => {
      matchedRoute["/settings"] = true;
      render(<AppSidebar {...makeProps()} />);
      const indicator = screen.getByTestId("nav-active-settings");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("shows active indicator on active Automation nav", () => {
      matchedRoute["/automation"] = true;
      render(<AppSidebar {...makeProps()} />);
      const indicator = screen.getByTestId("nav-active-automation");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("shows active indicator on active Knowledge nav", () => {
      matchedRoute["/knowledge"] = true;
      render(<AppSidebar {...makeProps()} />);
      const indicator = screen.getByTestId("nav-active-knowledge");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("shows active indicator on active Bridges nav", () => {
      matchedRoute["/bridges"] = true;
      render(<AppSidebar {...makeProps()} />);
      const indicator = screen.getByTestId("nav-active-bridges");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("shows active indicator on active Network nav", () => {
      matchedRoute["/network"] = true;
      render(<AppSidebar {...makeProps()} />);
      const indicator = screen.getByTestId("nav-active-network");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("shows active indicator on active Skills nav", () => {
      matchedRoute["/skills"] = true;
      render(<AppSidebar {...makeProps()} />);
      const indicator = screen.getByTestId("nav-active-skills");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("shows active indicator on active Tasks nav at the base path", () => {
      matchedRouteFuzzy["/tasks"] = true;
      render(<AppSidebar {...makeProps()} />);
      const indicator = screen.getByTestId("nav-active-tasks");
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[color:var(--color-accent)]");
    });

    it("keeps Tasks nav active for task detail and run detail deep links (fuzzy match)", () => {
      matchedRouteFuzzy["/tasks"] = true;
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("nav-active-tasks")).toBeInTheDocument();
    });

    it("does not activate Tasks nav without a /tasks match", () => {
      matchedRoute["/automation"] = true;
      render(<AppSidebar {...makeProps()} />);
      expect(screen.queryByTestId("nav-active-tasks")).not.toBeInTheDocument();
    });

    it("does not show active indicator when nav is not active", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.queryByTestId("nav-active-automation")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-bridges")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-network")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-knowledge")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-skills")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-settings")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-tasks")).not.toBeInTheDocument();
    });
  });

  describe("Collapse Toggle", () => {
    it("panel is visible when not collapsed", () => {
      render(<AppSidebar {...makeProps()} />);
      const panel = screen.getByTestId("sidebar-panel");
      expect(panel.className).toContain("w-[220px]");
      expect(panel.className).not.toContain("w-0");
    });

    it("clicking collapse delegates to the route owner", () => {
      render(<AppSidebar {...makeProps()} />);
      fireEvent.click(screen.getByTestId("collapse-toggle"));
      expect(onToggleCollapsed).toHaveBeenCalledTimes(1);
    });

    it("expand button appears when collapsed and delegates toggle", () => {
      render(<AppSidebar {...makeProps({ collapsed: true })} />);
      fireEvent.click(screen.getByTestId("expand-toggle"));
      expect(onToggleCollapsed).toHaveBeenCalledTimes(1);
    });
  });

  describe("System Footer", () => {
    it("shows connection status", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByTestId("connection-status")).toHaveTextContent("connected");
    });

    it("shows version from daemon health", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByText("v0.1.0")).toBeInTheDocument();
    });

    it("shows settings button", () => {
      render(<AppSidebar {...makeProps()} />);
      expect(screen.getByText("Settings")).toBeInTheDocument();
    });
  });
});
