import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useSidebarStore } from "@/stores/sidebar-store";

// ---------------------------------------------------------------------------
// Test state
// ---------------------------------------------------------------------------

const mutate = vi.fn();
let matchedRoute: Record<string, boolean> = {};

let agentsState = {
  data: [] as Array<{ name: string; provider: string; prompt: string }>,
  isLoading: false,
  isError: false,
};

let sessionsState = {
  data: [] as Array<{
    id: string;
    name: string;
    agent_name: string;
    workspace_id: string;
    state: string;
    updated_at: string;
    created_at: string;
  }>,
};

let workspacesState = {
  data: [
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
  ],
  isLoading: false,
  isError: false,
};

let healthState = {
  health: { version: "0.1.0" } as { version: string } | undefined,
  connectionStatus: "connected" as "connected" | "disconnected" | "reconnecting",
};

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("lucide-react", () => ({
  Book: () => <span data-testid="icon-book">book</span>,
  Bot: () => <span>bot</span>,
  ChevronRight: () => <span>chevron</span>,
  Loader2: () => <span>loader</span>,
  PanelLeftClose: () => <span>panel-close</span>,
  PanelLeftOpen: () => <span>panel-open</span>,
  Plus: () => <span>plus</span>,
  Search: () => <span>search</span>,
  Settings: () => <span>settings</span>,
  Terminal: () => <span data-testid="icon-terminal">terminal</span>,
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
  useMatchRoute: () => (opts: { to: string }) => matchedRoute[opts.to] ?? false,
}));

vi.mock("@/components/ui/collapsible", () => ({
  Collapsible: ({ children, className }: { children: ReactNode; className?: string }) => (
    <div className={className} data-state="open">
      {children}
    </div>
  ),
  CollapsibleContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  CollapsibleTrigger: ({ children, className }: { children: ReactNode; className?: string }) => (
    <button className={className}>{children}</button>
  ),
}));

vi.mock("@/components/ui/kbd", () => ({
  Kbd: ({ children }: { children: ReactNode }) => <kbd>{children}</kbd>,
}));

vi.mock("@/systems/agent/components/agent-icon", () => ({
  AgentIcon: ({ provider }: { provider: string }) => (
    <span data-testid={`agent-icon-${provider}`} />
  ),
}));

vi.mock("@/systems/agent/hooks/use-agents", () => ({
  useAgents: () => agentsState,
}));

vi.mock("@/systems/daemon/components/connection-status", () => ({
  ConnectionStatus: ({ status }: { status: string }) => (
    <span data-testid="connection-status">{status}</span>
  ),
}));

vi.mock("@/systems/daemon/hooks/use-daemon-health", () => ({
  useDaemonHealth: () => healthState,
}));

vi.mock("@/systems/session/hooks/use-session-actions", () => ({
  useCreateSession: () => ({
    mutate,
    isPending: false,
  }),
}));

vi.mock("@/systems/session/hooks/use-sessions", () => ({
  useSessions: () => sessionsState,
}));

vi.mock("@/systems/workspace", () => ({
  useWorkspaces: () => workspacesState,
}));

import { AppSidebar } from "./app-sidebar";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("AppSidebar", () => {
  beforeEach(() => {
    matchedRoute = {};
    agentsState = {
      data: [],
      isLoading: false,
      isError: false,
    };
    sessionsState = { data: [] };
    workspacesState = {
      data: [
        {
          id: "ws_alpha",
          root_dir: "/workspace/alpha",
          add_dirs: [],
          name: "alpha",
          created_at: "2026-04-06T10:00:00Z",
          updated_at: "2026-04-06T10:00:00Z",
        },
        {
          id: "ws_beta",
          root_dir: "/workspace/beta",
          add_dirs: [],
          name: "beta",
          created_at: "2026-04-06T10:00:00Z",
          updated_at: "2026-04-06T10:00:00Z",
        },
      ],
      isLoading: false,
      isError: false,
    };
    healthState = {
      health: { version: "0.1.0" },
      connectionStatus: "connected",
    };
    useSidebarStore.setState({ collapsed: false });
    mutate.mockReset();
  });

  // -------------------------------------------------------------------------
  // Icon Rail
  // -------------------------------------------------------------------------

  describe("Icon Rail", () => {
    it("renders the icon rail", () => {
      render(<AppSidebar />);
      expect(screen.getByTestId("icon-rail")).toBeInTheDocument();
    });

    it("renders workspace circle avatars with single-letter labels", () => {
      render(<AppSidebar />);
      const avatarAlpha = screen.getByTestId("workspace-avatar-ws_alpha");
      const avatarBeta = screen.getByTestId("workspace-avatar-ws_beta");
      expect(avatarAlpha).toHaveTextContent("A");
      expect(avatarBeta).toHaveTextContent("B");
    });

    it("renders app logo with accent background", () => {
      render(<AppSidebar />);
      const logo = screen.getByTestId("app-logo");
      expect(logo).toBeInTheDocument();
      expect(logo.className).toContain("bg-[#E8572A]");
    });

    it("highlights active workspace with accent ring border", () => {
      render(<AppSidebar />);
      // First workspace is active by default
      const activeAvatar = screen.getByTestId("workspace-avatar-ws_alpha");
      expect(activeAvatar.className).toContain("ring-[#E8572A]");
    });

    it("does not highlight inactive workspaces", () => {
      render(<AppSidebar />);
      const inactiveAvatar = screen.getByTestId("workspace-avatar-ws_beta");
      expect(inactiveAvatar.className).not.toContain("ring-[#E8572A]");
    });

    it("switches active workspace on click", () => {
      render(<AppSidebar />);
      const betaAvatar = screen.getByTestId("workspace-avatar-ws_beta");
      fireEvent.click(betaAvatar);
      // After clicking beta, beta should have ring
      expect(betaAvatar.className).toContain("ring-[#E8572A]");
    });
  });

  // -------------------------------------------------------------------------
  // Agent List
  // -------------------------------------------------------------------------

  describe("Agent List", () => {
    it("renders agents with session counts", () => {
      agentsState = {
        data: [
          { name: "coder", provider: "claude", prompt: "code" },
          { name: "writer", provider: "openai", prompt: "write" },
        ],
        isLoading: false,
        isError: false,
      };
      sessionsState = {
        data: [
          {
            id: "s1",
            name: "Session 1",
            agent_name: "coder",
            workspace_id: "ws_alpha",
            state: "active",
            updated_at: "2026-04-06T10:00:00Z",
            created_at: "2026-04-06T10:00:00Z",
          },
          {
            id: "s2",
            name: "Session 2",
            agent_name: "coder",
            workspace_id: "ws_alpha",
            state: "stopped",
            updated_at: "2026-04-06T09:00:00Z",
            created_at: "2026-04-06T09:00:00Z",
          },
        ],
      };

      render(<AppSidebar />);
      // Coder agent should show count 2
      expect(screen.getByText("coder")).toBeInTheDocument();
      expect(screen.getByText("2")).toBeInTheDocument();
      // Writer agent should show count 0
      expect(screen.getByText("writer")).toBeInTheDocument();
      expect(screen.getByText("0")).toBeInTheDocument();
    });

    it("shows bootstrap hint when no agents are loaded", () => {
      render(<AppSidebar />);
      expect(screen.getByText("Run `agh install` to bootstrap AGH")).toBeInTheDocument();
    });

    it("creates sessions in the selected workspace", () => {
      agentsState = {
        data: [{ name: "claude-agent", provider: "anthropic", prompt: "You are helpful." }],
        isLoading: false,
        isError: false,
      };

      render(<AppSidebar />);
      fireEvent.click(screen.getByTestId("new-session-claude-agent"));

      expect(mutate).toHaveBeenCalledWith({
        agent_name: "claude-agent",
        workspace: "ws_alpha",
      });
    });
  });

  // -------------------------------------------------------------------------
  // Navigation
  // -------------------------------------------------------------------------

  describe("Navigation", () => {
    it("renders Knowledge nav item linking to /_app/knowledge", () => {
      render(<AppSidebar />);
      const knowledgeLink = screen.getByTestId("nav-knowledge");
      expect(knowledgeLink).toBeInTheDocument();
      expect(knowledgeLink).toHaveAttribute("href", "/_app/knowledge");
    });

    it("renders Skills nav item linking to /_app/skills", () => {
      render(<AppSidebar />);
      const skillsLink = screen.getByTestId("nav-skills");
      expect(skillsLink).toBeInTheDocument();
      expect(skillsLink).toHaveAttribute("href", "/_app/skills");
    });

    it("shows active indicator (3px accent bar) on active Knowledge nav", () => {
      matchedRoute["/_app/knowledge"] = true;
      render(<AppSidebar />);
      const indicator = screen.getByTestId("nav-active-knowledge");
      expect(indicator).toBeInTheDocument();
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[#E8572A]");
    });

    it("shows active indicator (3px accent bar) on active Skills nav", () => {
      matchedRoute["/_app/skills"] = true;
      render(<AppSidebar />);
      const indicator = screen.getByTestId("nav-active-skills");
      expect(indicator).toBeInTheDocument();
      expect(indicator.className).toContain("w-[3px]");
      expect(indicator.className).toContain("bg-[#E8572A]");
    });

    it("does not show active indicator when nav is not active", () => {
      render(<AppSidebar />);
      expect(screen.queryByTestId("nav-active-knowledge")).not.toBeInTheDocument();
      expect(screen.queryByTestId("nav-active-skills")).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  // Collapse / Expand
  // -------------------------------------------------------------------------

  describe("Collapse Toggle", () => {
    it("panel is visible when not collapsed", () => {
      render(<AppSidebar />);
      const panel = screen.getByTestId("sidebar-panel");
      expect(panel.className).toContain("w-[220px]");
      expect(panel.className).not.toContain("w-0");
    });

    it("clicking collapse hides the panel but icon rail remains", () => {
      render(<AppSidebar />);
      fireEvent.click(screen.getByTestId("collapse-toggle"));
      const panel = screen.getByTestId("sidebar-panel");
      expect(panel.className).toContain("w-0");
      // Icon rail still present
      expect(screen.getByTestId("icon-rail")).toBeInTheDocument();
    });

    it("expand button appears when collapsed and restores panel", () => {
      useSidebarStore.setState({ collapsed: true });
      render(<AppSidebar />);
      const expandBtn = screen.getByTestId("expand-toggle");
      expect(expandBtn).toBeInTheDocument();
      fireEvent.click(expandBtn);
      const panel = screen.getByTestId("sidebar-panel");
      expect(panel.className).toContain("w-[220px]");
    });
  });

  // -------------------------------------------------------------------------
  // System Footer
  // -------------------------------------------------------------------------

  describe("System Footer", () => {
    it("shows connection status", () => {
      render(<AppSidebar />);
      expect(screen.getByTestId("connection-status")).toBeInTheDocument();
      expect(screen.getByTestId("connection-status")).toHaveTextContent("connected");
    });

    it("shows version from daemon health", () => {
      render(<AppSidebar />);
      expect(screen.getByText("v0.1.0")).toBeInTheDocument();
    });

    it("shows settings button", () => {
      render(<AppSidebar />);
      expect(screen.getByText("Settings")).toBeInTheDocument();
    });
  });
});
