import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { HomePageView } from "@/hooks/routes/use-home-page";

let mockHome: HomePageView;

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/hooks/routes/use-home-page", async () => {
  const actual = await vi.importActual<typeof import("@/hooks/routes/use-home-page")>(
    "@/hooks/routes/use-home-page"
  );
  return {
    ...actual,
    useHomePage: () => mockHome,
  };
});

import { Route } from "../index";

const HomePage = (Route as unknown as { component: () => React.ReactNode }).component;

function makeHome(overrides: Partial<HomePageView> = {}): HomePageView {
  return {
    isLoading: false,
    hasFatalError: false,
    errorMessage: null,
    connectionStatus: "connected",
    daemonStatus: {
      key: "healthy",
      tone: "success",
      label: "Healthy",
      description: "All subsystems are reporting healthy status.",
    },
    daemonVersion: "0.1.0-test",
    metrics: [
      { key: "active-sessions", label: "Active Sessions", value: "3", detail: "in main" },
      { key: "workspaces", label: "Workspaces", value: "2" },
      { key: "agents", label: "Agents", value: "5" },
      { key: "uptime", label: "Daemon Uptime", value: "2h" },
    ],
    hasWorkspaces: true,
    activeWorkspaceName: "main",
    ...overrides,
  };
}

describe("AppHomePage", () => {
  beforeEach(() => {
    mockHome = makeHome();
  });

  it("renders the page header with title 'Home' and the connection indicator", () => {
    render(<HomePage />);

    expect(screen.getByTestId("home-page-header")).toBeInTheDocument();
    expect(screen.getByTestId("home-page-title")).toHaveTextContent("Home");
    const indicator = screen.getByTestId("home-connection-indicator");
    expect(indicator).toHaveAttribute("data-status", "connected");
  });

  it("renders the daemon status card with the matching StatusDot tone for healthy", () => {
    render(<HomePage />);

    const card = screen.getByTestId("home-daemon-card");
    expect(card).toHaveAttribute("data-status", "healthy");
    expect(within(card).getByTestId("home-daemon-status-label")).toHaveTextContent("Healthy");
    expect(within(card).getByTestId("home-daemon-status-dot")).toHaveAttribute(
      "data-tone",
      "success"
    );
  });

  it.each([
    ["healthy", "success"],
    ["degraded", "warning"],
    ["unknown", "neutral"],
  ] as const)("maps the %s daemon status to the %s StatusDot tone", (key, tone) => {
    mockHome = makeHome({
      daemonStatus: {
        key,
        tone,
        label: key,
        description: ",",
      },
    });

    render(<HomePage />);

    const dot = screen.getByTestId("home-daemon-status-dot");
    expect(dot).toHaveAttribute("data-tone", tone);
  });

  it("renders all four metrics in the overview grid", () => {
    render(<HomePage />);

    const grid = screen.getByTestId("home-metric-grid");
    expect(within(grid).getByTestId("home-metric-active-sessions")).toHaveTextContent("3");
    expect(within(grid).getByTestId("home-metric-workspaces")).toHaveTextContent("2");
    expect(within(grid).getByTestId("home-metric-agents")).toHaveTextContent("5");
    expect(within(grid).getByTestId("home-metric-uptime")).toHaveTextContent("2h");
  });

  it("renders the daemon version badge in the daemon section header", () => {
    render(<HomePage />);
    expect(screen.getByTestId("home-daemon-version")).toHaveTextContent("v0.1.0-test");
  });

  it("hides the daemon version badge when the daemon has not reported a version", () => {
    mockHome = makeHome({ daemonVersion: null });
    render(<HomePage />);
    expect(screen.queryByTestId("home-daemon-version")).not.toBeInTheDocument();
  });

  it("renders skeletons for each metric while loading", () => {
    mockHome = makeHome({ isLoading: true });

    render(<HomePage />);

    expect(screen.getByTestId("home-loading")).toBeInTheDocument();
    expect(screen.getByTestId("home-metric-skeleton")).toBeInTheDocument();
    expect(screen.getByTestId("home-metric-skeleton-active-sessions")).toBeInTheDocument();
    expect(screen.getByTestId("home-metric-skeleton-workspaces")).toBeInTheDocument();
    expect(screen.getByTestId("home-metric-skeleton-agents")).toBeInTheDocument();
    expect(screen.getByTestId("home-metric-skeleton-uptime")).toBeInTheDocument();
    expect(screen.queryByTestId("home-metric-grid")).not.toBeInTheDocument();
  });

  it("renders an Empty error region with the daemon error message", () => {
    mockHome = makeHome({
      hasFatalError: true,
      errorMessage: "Workspaces could not be loaded",
    });

    render(<HomePage />);

    const errorRegion = screen.getByTestId("home-error");
    expect(errorRegion).toBeInTheDocument();
    expect(errorRegion).toHaveTextContent("Unable to load dashboard");
    expect(errorRegion).toHaveTextContent("Workspaces could not be loaded");
    expect(screen.queryByTestId("home-metric-grid")).not.toBeInTheDocument();
  });

  it("renders a fallback error message when none is provided", () => {
    mockHome = makeHome({ hasFatalError: true, errorMessage: null });

    render(<HomePage />);

    expect(screen.getByTestId("home-error")).toHaveTextContent(
      "Unable to load workspace data from the daemon."
    );
  });

  it("renders a disconnected daemon section with a recovery hint", () => {
    mockHome = makeHome({
      connectionStatus: "disconnected",
      daemonStatus: {
        key: "disconnected",
        tone: "danger",
        label: "Disconnected",
        description: "The daemon is unreachable. Start it with `agh daemon`.",
      },
    });

    render(<HomePage />);

    const disconnected = screen.getByTestId("home-daemon-disconnected");
    expect(disconnected).toBeInTheDocument();
    const indicator = within(disconnected).getByTestId("home-daemon-disconnected-indicator");
    expect(indicator).toHaveAttribute("data-status", "disconnected");
    expect(disconnected).toHaveTextContent("agh daemon");
    expect(screen.queryByTestId("home-daemon-card")).not.toBeInTheDocument();
  });

  it("renders the connection indicator pill in the header in disconnected tone", () => {
    mockHome = makeHome({
      connectionStatus: "disconnected",
    });

    render(<HomePage />);

    expect(screen.getByTestId("home-connection-indicator")).toHaveAttribute(
      "data-status",
      "disconnected"
    );
  });
});
