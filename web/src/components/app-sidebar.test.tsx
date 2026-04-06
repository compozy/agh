import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("lucide-react", () => ({
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
  AgentSidebarGroup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/systems/agent/hooks/use-agents", () => ({
  useAgents: () => ({
    data: [],
    isLoading: false,
    isError: false,
  }),
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
  SessionSidebarItem: () => <div>session-item</div>,
}));

vi.mock("@/systems/session/hooks/use-session-actions", () => ({
  useCreateSession: () => ({
    mutate: vi.fn(),
  }),
}));

vi.mock("@/systems/session/hooks/use-sessions", () => ({
  useSessions: () => ({
    data: [],
  }),
}));

import { AppSidebar } from "./app-sidebar";

describe("AppSidebar", () => {
  it("prompts the user to run agh install when no agents are loaded", () => {
    render(<AppSidebar />);

    expect(screen.getByText("Run `agh install` to bootstrap AGH")).toBeInTheDocument();
  });
});
