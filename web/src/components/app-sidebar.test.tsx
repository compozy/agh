import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mutate = vi.fn();

let agentsState = {
  data: [] as Array<{ name: string; provider: string; prompt: string }>,
  isLoading: false,
  isError: false,
};

let workspacesState = {
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

vi.mock("lucide-react", () => ({
  AlertCircle: () => <span>alert</span>,
  Bot: () => <span>bot</span>,
  Loader2: () => <span>loader</span>,
  Search: () => <span>search</span>,
  Settings: () => <span>settings</span>,
  Terminal: () => <span>terminal</span>,
}));

vi.mock("@/components/ui/sidebar", () => ({
  Sidebar: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarGroup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarGroupContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarGroupLabel: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarMenu: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarMenuButton: ({ children, ...props }: { children: ReactNode; tooltip?: string }) => (
    <button {...props}>{children}</button>
  ),
  SidebarMenuItem: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SidebarRail: () => <div data-testid="sidebar-rail" />,
  SidebarSeparator: () => <hr />,
}));

vi.mock("@/components/ui/kbd", () => ({
  Kbd: ({ children }: { children: ReactNode }) => <kbd>{children}</kbd>,
}));

vi.mock("@/systems/agent/components/agent-sidebar-group", () => ({
  AgentSidebarGroup: ({
    agent,
    onNewSession,
    newSessionDisabled,
    children,
  }: {
    agent: { name: string };
    onNewSession?: (agentName: string) => void;
    newSessionDisabled?: boolean;
    children?: ReactNode;
  }) => (
    <div>
      <button
        data-testid={`new-session-${agent.name}`}
        onClick={() => onNewSession?.(agent.name)}
        disabled={newSessionDisabled}
      >
        New Session
      </button>
      {children}
    </div>
  ),
}));

vi.mock("@/systems/agent/hooks/use-agents", () => ({
  useAgents: () => agentsState,
}));

vi.mock("@/systems/daemon/components/connection-status", () => ({
  ConnectionStatus: ({ status }: { status: string }) => <span>{status}</span>,
}));

vi.mock("@/systems/daemon/hooks/use-daemon-health", () => ({
  useDaemonHealth: () => ({
    connectionStatus: "connected",
  }),
}));

vi.mock("@/systems/session/components/session-sidebar-item", () => ({
  SessionSidebarItem: ({ workspaceName }: { workspaceName?: string }) => (
    <div>{workspaceName ?? "session-item"}</div>
  ),
}));

vi.mock("@/systems/session/hooks/use-session-actions", () => ({
  useCreateSession: () => ({
    mutate,
    isPending: false,
  }),
}));

vi.mock("@/systems/session/hooks/use-sessions", () => ({
  useSessions: () => ({
    data: [],
  }),
}));

vi.mock("@/systems/workspace", () => ({
  WorkspaceSelector: ({
    workspaces,
    value,
    onValueChange,
  }: {
    workspaces: Array<{ id: string; name: string }>;
    value: string | null;
    onValueChange: (value: string) => void;
  }) => (
    <select
      aria-label="Workspace"
      value={value ?? ""}
      onChange={event => onValueChange(event.currentTarget.value)}
    >
      {workspaces.map(workspace => (
        <option key={workspace.id} value={workspace.id}>
          {workspace.name}
        </option>
      ))}
    </select>
  ),
  useWorkspaces: () => workspacesState,
}));

import { AppSidebar } from "./app-sidebar";

describe("AppSidebar", () => {
  beforeEach(() => {
    agentsState = {
      data: [],
      isLoading: false,
      isError: false,
    };
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
    mutate.mockReset();
  });

  it("prompts the user to run agh install when no agents are loaded", () => {
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

    fireEvent.change(screen.getByLabelText("Workspace"), {
      target: { value: "ws_beta" },
    });
    fireEvent.click(screen.getByTestId("new-session-claude-agent"));

    expect(mutate).toHaveBeenCalledWith({
      agent_name: "claude-agent",
      workspace: "ws_beta",
    });
  });

  it("shows a workspace registration hint when no workspaces are available", () => {
    workspacesState = {
      data: [],
      isLoading: false,
      isError: false,
    };

    render(<AppSidebar />);

    expect(
      screen.getByText("Run `agh workspace add <path>` to register a workspace")
    ).toBeInTheDocument();
  });
});
