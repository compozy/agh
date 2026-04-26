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
let mockPathname = "/tasks";
let mockLatestPathname = "/tasks";
const mockInvalidate = vi.fn();
const mockReset = vi.fn();
const mockNavigate = vi.fn();

const mockSetActiveWorkspaceId = vi.fn();
const mockCreateSessionMutateAsync = vi.fn();

vi.mock("@tanstack/react-router", () => ({
  createFileRoute:
    () =>
    (opts: {
      component: () => ReactNode;
      errorComponent?: (props: { error: Error; reset: () => void }) => ReactNode;
      notFoundComponent?: (props: { isNotFound: true; routeId: string }) => ReactNode;
    }) => ({
      component: opts.component,
      errorComponent: opts.errorComponent,
      notFoundComponent: opts.notFoundComponent,
    }),
  Outlet: () => <div data-testid="outlet" />,
  useLocation: <T,>(opts?: { select?: (location: { pathname: string }) => T }) => {
    const location = { pathname: mockPathname };
    return opts?.select ? opts.select(location) : location;
  },
  Link: ({
    children,
    to,
    ...props
  }: {
    children: ReactNode;
    to: string;
  } & React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
  useRouter: () => ({
    invalidate: mockInvalidate,
    latestLocation: { pathname: mockLatestPathname },
  }),
  useNavigate: () => mockNavigate,
}));

vi.mock("@/components/app-sidebar", () => ({
  AppSidebar: ({ onAddWorkspace }: { onAddWorkspace: () => void }) => (
    <button data-testid="app-sidebar" onClick={onAddWorkspace} type="button">
      Sidebar
    </button>
  ),
}));

vi.mock("@/stores/sidebar-store", () => ({
  useSidebarStore: (
    selector: (state: {
      collapsed: boolean;
      toggle: () => void;
      setCollapsed: (next: boolean) => void;
    }) => unknown
  ) => selector({ collapsed: false, toggle: vi.fn(), setCollapsed: vi.fn() }),
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
    mutateAsync: mockCreateSessionMutateAsync,
    isPending: false,
  }),
  useSessions: () => ({
    data: [],
  }),
  useSessionCreateDialog: () => ({
    open: false,
    agents: [],
    workspace: undefined,
    providerOptions: [],
    providersLoading: false,
    providersError: null,
    selectedAgentName: "",
    selectedProvider: "",
    isSubmitting: false,
    submitError: null,
    pendingAgentName: null,
    pendingWorkspaceId: null,
    openForAgent: vi.fn(),
    setOpen: vi.fn(),
    onAgentChange: vi.fn(),
    onProviderChange: vi.fn(),
    submit: vi.fn(),
  }),
  SessionCreateDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="session-create-dialog" /> : null,
  SessionCreateProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
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
  useWorkspace: () => ({
    data: {
      workspace: workspaceFixture,
      agents: [],
      providers: [],
    },
    isLoading: false,
    isError: false,
    error: null,
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const AppErrorBoundary = (Route as any).errorComponent as (props: {
    error: Error;
    reset: () => void;
  }) => ReactNode;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const AppNotFoundBoundary = (Route as any).notFoundComponent as (props: {
    isNotFound: true;
    routeId: string;
  }) => ReactNode;

  beforeEach(() => {
    mockHasWorkspaces = true;
    mockActiveWorkspaceId = workspaceFixture.id;
    mockPathname = "/tasks";
    mockLatestPathname = "/tasks";
    mockInvalidate.mockReset();
    mockNavigate.mockReset();
    mockReset.mockReset();
    mockSetActiveWorkspaceId.mockReset();
    mockCreateSessionMutateAsync.mockReset();
  });

  it("renders sidebar and outlet directly inside the content column without a motion shell", () => {
    render(<AppLayout />);
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    const content = screen.getByTestId("app-content");
    expect(content).toContainElement(screen.getByTestId("outlet"));
    expect(screen.queryByTestId("animate-presence")).not.toBeInTheDocument();
    expect(screen.queryByTestId("app-route-motion")).not.toBeInTheDocument();
  });

  it("renders onboarding instead of the shell when no workspaces exist", () => {
    mockHasWorkspaces = false;
    mockActiveWorkspaceId = null;

    render(<AppLayout />);

    expect(screen.getByTestId("workspace-onboarding")).toBeInTheDocument();
    expect(screen.queryByTestId("app-sidebar")).not.toBeInTheDocument();
    expect(screen.queryByTestId("outlet")).not.toBeInTheDocument();
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

  it("renders an app-level not-found fallback with a path back home", () => {
    render(<AppNotFoundBoundary isNotFound routeId="/_app" />);

    expect(screen.getByTestId("app-route-not-found")).toBeInTheDocument();
    expect(screen.getByText("Page not found")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go home" })).toHaveAttribute("href", "/");
  });

  it("renders an app-level error fallback that resets and invalidates the router", () => {
    render(<AppErrorBoundary error={new Error("app route failed")} reset={mockReset} />);

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(mockReset).toHaveBeenCalledTimes(1);
    expect(mockInvalidate).toHaveBeenCalledWith({ forcePending: true });
    expect(screen.getByText("app route failed")).toBeInTheDocument();
  });
});
