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
const reducedMotionMock = vi.fn<() => boolean>().mockReturnValue(false);

const mockSetActiveWorkspaceId = vi.fn();
const mockCreateSessionMutate = vi.fn();

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
  Outlet: () => <div data-testid="outlet" />,
  useLocation: <T,>(opts?: { select?: (location: { pathname: string }) => T }) => {
    const location = { pathname: mockPathname };
    return opts?.select ? opts.select(location) : location;
  },
}));

vi.mock("motion/react", () => ({
  AnimatePresence: ({
    mode,
    initial,
    children,
  }: {
    mode?: string;
    initial?: boolean;
    children: ReactNode;
  }) => (
    <div
      data-testid="animate-presence"
      data-mode={mode}
      data-initial={initial === undefined ? "true" : String(initial)}
    >
      {children}
    </div>
  ),
  motion: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    div: ({ children, transition, initial, animate, exit, ...rest }: any) => (
      <div
        data-motion-duration={transition?.duration}
        data-motion-ease={transition?.ease}
        data-motion-initial={JSON.stringify(initial ?? null)}
        data-motion-animate={JSON.stringify(animate ?? null)}
        data-motion-exit={JSON.stringify(exit ?? null)}
        {...rest}
      >
        {children}
      </div>
    ),
  },
  useReducedMotionConfig: () => reducedMotionMock(),
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

import { Route, resolveRouteTransitionDuration, ROUTE_FADE_DURATION } from "./_app";

describe("AppLayout", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const AppLayout = (Route as any).component as () => ReactNode;

  beforeEach(() => {
    mockHasWorkspaces = true;
    mockActiveWorkspaceId = workspaceFixture.id;
    mockPathname = "/tasks";
    reducedMotionMock.mockReset();
    reducedMotionMock.mockReturnValue(false);
    mockSetActiveWorkspaceId.mockReset();
    mockCreateSessionMutate.mockReset();
  });

  it("renders sidebar, content column, and outlet wrapped in the route motion shell", () => {
    render(<AppLayout />);
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("app-content")).toBeInTheDocument();
    const presence = screen.getByTestId("animate-presence");
    expect(presence).toHaveAttribute("data-mode", "wait");
    expect(presence).toContainElement(screen.getByTestId("app-route-motion"));
    expect(screen.getByTestId("app-route-motion")).toContainElement(screen.getByTestId("outlet"));
  });

  it("keys the motion shell by location pathname so a route swap replaces it", () => {
    mockPathname = "/tasks";
    const first = render(<AppLayout />);
    expect(first.getByTestId("app-route-motion")).toHaveAttribute("data-route-key", "/tasks");
    first.unmount();

    mockPathname = "/session/abc";
    const second = render(<AppLayout />);
    expect(second.getByTestId("app-route-motion")).toHaveAttribute(
      "data-route-key",
      "/session/abc"
    );
  });

  it("uses the 200ms ease-out fade under default motion preferences", () => {
    render(<AppLayout />);
    const motionEl = screen.getByTestId("app-route-motion");
    expect(motionEl.dataset.motionDuration).toBe(String(ROUTE_FADE_DURATION));
    expect(motionEl.dataset.motionEase).toBe("easeOut");
    expect(JSON.parse(motionEl.dataset.motionInitial ?? "null")).toEqual({ opacity: 0 });
    expect(JSON.parse(motionEl.dataset.motionAnimate ?? "null")).toEqual({ opacity: 1 });
    expect(JSON.parse(motionEl.dataset.motionExit ?? "null")).toEqual({ opacity: 0 });
  });

  it("collapses route transitions to duration 0 under prefers-reduced-motion: reduce", () => {
    reducedMotionMock.mockReturnValue(true);
    render(<AppLayout />);
    const motionEl = screen.getByTestId("app-route-motion");
    expect(motionEl.dataset.motionDuration).toBe("0");
  });

  it("exposes resolveRouteTransitionDuration so consumers can test the gating logic", () => {
    expect(resolveRouteTransitionDuration(false)).toBe(ROUTE_FADE_DURATION);
    expect(resolveRouteTransitionDuration(true)).toBe(0);
  });

  it("renders onboarding instead of the shell when no workspaces exist", () => {
    mockHasWorkspaces = false;
    mockActiveWorkspaceId = null;

    render(<AppLayout />);

    expect(screen.getByTestId("workspace-onboarding")).toBeInTheDocument();
    expect(screen.queryByTestId("app-sidebar")).not.toBeInTheDocument();
    expect(screen.queryByTestId("animate-presence")).not.toBeInTheDocument();
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
