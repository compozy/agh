import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const workspaceFixture = {
  id: "ws_alpha",
  root_dir: "/workspace/alpha",
  add_dirs: [],
  name: "alpha",
  created_at: "2026-04-06T10:00:00Z",
  updated_at: "2026-04-06T10:00:00Z",
};

let mockHasWorkspaces = true;
let mockActiveWorkspaceId: string | null = "ws_alpha";

const mockSetActiveWorkspaceId = vi.fn();
const mockCreateSessionMutate = vi.fn();

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
  Outlet: () => <div data-testid="outlet" />,
}));

vi.mock("@/components/app-sidebar", () => ({
  AppSidebar: ({ onAddWorkspace }: { onAddWorkspace: () => void }) => (
    <button data-testid="app-sidebar" onClick={onAddWorkspace} type="button">
      Sidebar
    </button>
  ),
}));

vi.mock("@/stores/sidebar-store", () => ({
  useSidebarStore: (selector: (state: { collapsed: boolean; toggle: () => void }) => unknown) =>
    selector({ collapsed: false, toggle: vi.fn() }),
}));

vi.mock("@/systems/daemon", () => ({
  useDaemonHealth: () => ({
    health: { version: "0.1.0" },
    connectionStatus: "connected",
  }),
}));

vi.mock("@/systems/agent", () => ({
  useAgents: () => ({
    data: [],
    isLoading: false,
    isError: false,
  }),
}));

vi.mock("@/systems/session", () => ({
  useCreateSession: () => ({
    mutate: mockCreateSessionMutate,
    isPending: false,
  }),
  useSessions: () => ({
    data: [],
  }),
}));

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({
    workspaces: mockHasWorkspaces ? [workspaceFixture] : [],
    hasWorkspaces: mockHasWorkspaces,
    activeWorkspace: mockHasWorkspaces ? workspaceFixture : undefined,
    activeWorkspaceId: mockActiveWorkspaceId,
    setActiveWorkspaceId: mockSetActiveWorkspaceId,
    isLoading: false,
    isError: false,
  }),
  WorkspaceOnboarding: ({
    onWorkspaceResolved,
  }: {
    onWorkspaceResolved: (workspaceId: string) => void;
  }) => (
    <button
      data-testid="workspace-onboarding"
      onClick={() => onWorkspaceResolved("ws_home")}
      type="button"
    >
      Workspace onboarding
    </button>
  ),
  WorkspaceSetupDialog: ({
    open,
    onWorkspaceResolved,
  }: {
    open: boolean;
    onWorkspaceResolved: (workspaceId: string) => void;
  }) =>
    open ? (
      <button
        data-testid="workspace-setup-dialog"
        onClick={() => onWorkspaceResolved("ws_new")}
        type="button"
      >
        Workspace setup
      </button>
    ) : null,
}));

import { Route } from "./_app";

describe("AppLayout", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const AppLayout = (Route as any).component as () => ReactNode;

  beforeEach(() => {
    mockHasWorkspaces = true;
    mockActiveWorkspaceId = workspaceFixture.id;
    mockSetActiveWorkspaceId.mockReset();
    mockCreateSessionMutate.mockReset();
  });

  it("renders sidebar and outlet", () => {
    render(<AppLayout />);
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("renders outlet", () => {
    render(<AppLayout />);
    const outlet = screen.getByTestId("outlet");
    expect(outlet).toBeInTheDocument();
  });

  it("renders onboarding instead of the shell when no workspaces exist", () => {
    mockHasWorkspaces = false;
    mockActiveWorkspaceId = null;

    render(<AppLayout />);

    expect(screen.getByTestId("workspace-onboarding")).toBeInTheDocument();
    expect(screen.queryByTestId("app-sidebar")).not.toBeInTheDocument();
  });

  it("propagates resolved workspace ids from onboarding", () => {
    mockHasWorkspaces = false;
    mockActiveWorkspaceId = null;

    render(<AppLayout />);
    fireEvent.click(screen.getByTestId("workspace-onboarding"));

    expect(mockSetActiveWorkspaceId).toHaveBeenCalledWith("ws_home");
  });

  it("opens workspace setup from the sidebar trigger and selects the resolved workspace", () => {
    render(<AppLayout />);

    fireEvent.click(screen.getByTestId("app-sidebar"));
    expect(screen.getByTestId("workspace-setup-dialog")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("workspace-setup-dialog"));
    expect(mockSetActiveWorkspaceId).toHaveBeenCalledWith("ws_new");
  });
});
