import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@agh/ui", async () => {
  const actual = await vi.importActual<typeof import("@agh/ui")>("@agh/ui");
  return {
    ...actual,
    Toaster: () => <div data-testid="toaster" />,
    TooltipProvider: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="tooltip-provider">{children}</div>
    ),
  };
});

vi.mock("@tanstack/react-router", () => ({
  createRootRoute: (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
  Outlet: () => <div data-testid="outlet" />,
}));

import { Route } from "./__root";

describe("RootComponent", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const RootComponent = (Route as any).component as () => React.ReactNode;

  it("renders the sticky app header with wordmark, ALPHA chip, and placeholder nav", () => {
    render(<RootComponent />);
    const header = screen.getByTestId("app-header");
    expect(header).toBeInTheDocument();
    expect(screen.getByTestId("app-header-wordmark")).toHaveTextContent("agh");
    expect(screen.getByTestId("app-header-alpha-chip")).toHaveTextContent(/alpha/i);
    expect(screen.getByTestId("app-header-nav")).toBeInTheDocument();
  });

  it("wraps the shell in a TooltipProvider", () => {
    render(<RootComponent />);
    const tooltipProvider = screen.getByTestId("tooltip-provider");
    expect(tooltipProvider).toBeInTheDocument();
    expect(tooltipProvider).toContainElement(screen.getByTestId("app-shell"));
  });

  it("renders the Outlet below the header", () => {
    render(<RootComponent />);
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("mounts the global Toaster once", () => {
    render(<RootComponent />);
    expect(screen.getByTestId("toaster")).toBeInTheDocument();
  });
});
