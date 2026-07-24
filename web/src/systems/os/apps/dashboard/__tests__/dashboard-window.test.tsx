import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { TopbarSlotProvider } from "@agh/ui";

const connection = { status: "connected" as "connected" | "disconnected" };

vi.mock("@/systems/status", () => ({
  useDaemonHealth: () => ({
    connectionStatus: connection.status,
    health: undefined,
    isInitialLoading: false,
  }),
}));

vi.mock("@/systems/session", () => ({
  useSessionCreate: () => ({
    openForAgent: vi.fn(),
    isCreating: false,
    pendingAgentName: null,
    hasActiveWorkspace: true,
  }),
}));

vi.mock("@/systems/dashboard", () => ({
  HomeDashboard: () => <div data-testid="home-body" />,
}));

import { DashboardWindow } from "../dashboard-window";

function renderWindow() {
  return render(
    <TopbarSlotProvider>
      <DashboardWindow windowId="win-1" />
    </TopbarSlotProvider>
  );
}

describe("DashboardWindow", () => {
  it("Should render the home dashboard body while connected", () => {
    connection.status = "connected";
    renderWindow();
    expect(screen.getByTestId("home-body")).toBeInTheDocument();
  });

  it("Should render the disconnect surface when the daemon is unreachable", () => {
    connection.status = "disconnected";
    renderWindow();
    expect(screen.getByTestId("home-error")).toBeInTheDocument();
    expect(screen.queryByTestId("home-body")).toBeNull();
  });
});
