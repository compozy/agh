import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => React.ReactNode }) => ({
    component: opts.component,
  }),
  Outlet: () => <div data-testid="outlet" />,
}));

vi.mock("@/components/app-header", () => ({
  AppHeader: () => <header data-testid="app-header">Header</header>,
}));

vi.mock("@/components/app-sidebar", () => ({
  AppSidebar: () => <nav data-testid="app-sidebar">Sidebar</nav>,
}));

import { Route } from "./_app";

describe("AppLayout", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const AppLayout = (Route as any).component as () => React.ReactNode;

  it("renders sidebar and outlet", () => {
    render(<AppLayout />);
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("renders header alongside sidebar", () => {
    render(<AppLayout />);
    expect(screen.getByTestId("app-header")).toBeInTheDocument();
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
  });

  it("renders outlet inside the content area", () => {
    render(<AppLayout />);
    const outlet = screen.getByTestId("outlet");
    expect(outlet).toBeInTheDocument();
  });
});
