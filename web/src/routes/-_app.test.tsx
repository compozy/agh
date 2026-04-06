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

vi.mock("@/components/ui/sidebar", () => ({
  SidebarProvider: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sidebar-provider">{children}</div>
  ),
  SidebarInset: ({ children, className }: { children: React.ReactNode; className?: string }) => (
    <main data-testid="sidebar-inset" className={className}>
      {children}
    </main>
  ),
}));

import { Route } from "./_app";

describe("AppLayout", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const AppLayout = (Route as any).component as () => React.ReactNode;

  it("renders sidebar provider wrapping all content", () => {
    render(<AppLayout />);
    expect(screen.getByTestId("sidebar-provider")).toBeInTheDocument();
  });

  it("renders sidebar and outlet", () => {
    render(<AppLayout />);
    expect(screen.getByTestId("app-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("renders header inside sidebar inset", () => {
    render(<AppLayout />);
    const inset = screen.getByTestId("sidebar-inset");
    const header = screen.getByTestId("app-header");
    expect(inset).toContainElement(header);
  });

  it("renders outlet inside sidebar inset for child routes", () => {
    render(<AppLayout />);
    const inset = screen.getByTestId("sidebar-inset");
    const outlet = screen.getByTestId("outlet");
    expect(inset).toContainElement(outlet);
  });
});
