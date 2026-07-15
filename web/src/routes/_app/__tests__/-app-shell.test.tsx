import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { mockUseAppLayout, mockAppSidebar } = vi.hoisted(() => ({
  mockUseAppLayout: vi.fn(),
  mockAppSidebar: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  Outlet: () => <div>route outlet</div>,
}));

vi.mock("@/hooks/routes/use-app-layout", () => ({
  useAppLayout: mockUseAppLayout,
}));

vi.mock("@/components/topbar-shell", () => ({
  TopbarShell: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/systems/onboarding", () => ({
  useOnboardingStatus: () => ({ data: { completed: true } }),
  OnboardingWizard: () => null,
}));

vi.mock("@/systems/runtime", () => ({
  AppSidebar: (props: unknown) => {
    mockAppSidebar(props);
    return <div data-testid="app-sidebar" />;
  },
}));

vi.mock("@/systems/agent", () => ({
  AgentCreateDialog: () => null,
  AgentCreateHostProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/systems/session", () => ({
  SessionCreateDialog: () => null,
  SessionCreateProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/systems/workspace", () => ({
  WorkspaceOnboarding: () => null,
  WorkspaceSetupDialog: () => null,
}));

import { AppLayout } from "../-app-shell";

describe("AppShell session activity boundary", () => {
  beforeEach(() => {
    mockAppSidebar.mockReset();
    mockUseAppLayout.mockReset();
  });

  it("Should always pass the materialized workspace activity map to AppSidebar", () => {
    const workspaceSessionActivity = {};
    mockUseAppLayout.mockReturnValue({
      areWorkspacesLoading: false,
      workspacesError: false,
      hasWorkspaces: true,
      activeWorkspace: undefined,
      activeWorkspaceId: null,
      workspaces: [],
      collapsed: false,
      setCollapsed: vi.fn(),
      setActiveWorkspaceId: vi.fn(),
      openWorkspaceSetup: vi.fn(),
      agentsCount: undefined,
      activeSessionCount: 0,
      workspaceSessionActivity,
      handleNewSession: vi.fn(),
      isCreatingSession: false,
      pendingSessionAgentName: undefined,
      isWorkspaceSetupOpen: false,
      setWorkspaceSetupOpen: vi.fn(),
      agentCreate: {},
      sessionCreate: {},
    });

    render(<AppLayout />);

    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    expect(mockAppSidebar).toHaveBeenCalledWith(
      expect.objectContaining({ workspaceSessionActivity })
    );
    expect(mockAppSidebar.mock.calls[0]?.[0]).not.toHaveProperty(
      "workspaceSessionActivity",
      undefined
    );
  });
});
