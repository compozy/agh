import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("next-themes", () => ({
  ThemeProvider: ({ children, ...props }: { children: React.ReactNode; attribute?: string }) => (
    <div data-testid="theme-provider" data-attribute={props.attribute}>
      {children}
    </div>
  ),
}));

vi.mock("@/components/ui/sonner", () => ({
  Toaster: () => <div data-testid="toaster" />,
}));

vi.mock("@/components/ui/tooltip", () => ({
  TooltipProvider: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="tooltip-provider">{children}</div>
  ),
}));

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

  it("wraps app with ThemeProvider using class attribute", () => {
    render(<RootComponent />);
    const themeProvider = screen.getByTestId("theme-provider");
    expect(themeProvider).toBeInTheDocument();
    expect(themeProvider).toHaveAttribute("data-attribute", "class");
  });

  it("renders TooltipProvider inside ThemeProvider", () => {
    render(<RootComponent />);
    const themeProvider = screen.getByTestId("theme-provider");
    const tooltipProvider = screen.getByTestId("tooltip-provider");
    expect(themeProvider).toContainElement(tooltipProvider);
  });

  it("renders Outlet for child routes", () => {
    render(<RootComponent />);
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("renders Toaster for notifications", () => {
    render(<RootComponent />);
    expect(screen.getByTestId("toaster")).toBeInTheDocument();
  });
});
