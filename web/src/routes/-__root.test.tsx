import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const mockInvalidate = vi.fn();
const mockReset = vi.fn();

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
  createRootRoute: (opts: {
    component: () => React.ReactNode;
    errorComponent?: (props: { error: Error; reset: () => void }) => React.ReactNode;
    notFoundComponent?: (props: { isNotFound: true; routeId: string }) => React.ReactNode;
  }) => ({
    component: opts.component,
    errorComponent: opts.errorComponent,
    notFoundComponent: opts.notFoundComponent,
  }),
  Outlet: () => <div data-testid="outlet" />,
  Link: ({
    children,
    to,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
  } & React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
  useRouter: () => ({
    invalidate: mockInvalidate,
  }),
  useMatchRoute: () => (opts: { to: string }) => opts.to === "/",
}));

import { Route } from "./__root";

describe("RootComponent", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const RootComponent = (Route as any).component as () => React.ReactNode;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const RootErrorBoundary = (Route as any).errorComponent as (props: {
    error: Error;
    reset: () => void;
  }) => React.ReactNode;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const RootNotFoundBoundary = (Route as any).notFoundComponent as (props: {
    isNotFound: true;
    routeId: string;
  }) => React.ReactNode;

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

  it("renders a root-level not-found fallback with a clear path home", () => {
    render(<RootNotFoundBoundary isNotFound routeId="__root__" />);

    expect(screen.getByTestId("root-route-not-found")).toBeInTheDocument();
    expect(screen.getByText("Route not found")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go home" })).toHaveAttribute("href", "/");
  });

  it("renders a root-level error fallback that resets and invalidates the router", () => {
    render(<RootErrorBoundary error={new Error("route failed")} reset={mockReset} />);

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(mockReset).toHaveBeenCalledTimes(1);
    expect(mockInvalidate).toHaveBeenCalledWith({ forcePending: true });
    expect(screen.getByText("route failed")).toBeInTheDocument();
  });
});
