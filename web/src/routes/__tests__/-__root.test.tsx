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

import { routeComponent, routeErrorComponent, routeNotFoundComponent } from "@/test/route-options";

import { Route } from "../__root";

const RootComponent = routeComponent(Route);
const RootErrorBoundary: (props: { error: Error; reset: () => void }) => React.ReactNode =
  routeErrorComponent(Route);
const RootNotFoundBoundary: (props: { isNotFound: true; routeId: string }) => React.ReactNode =
  routeNotFoundComponent(Route);

describe("RootComponent", () => {
  it("renders the Outlet inside the shell", () => {
    render(<RootComponent />);
    expect(screen.getByTestId("outlet")).toBeInTheDocument();
  });

  it("renders a skip-to-content link that targets the app-content main", () => {
    render(<RootComponent />);
    const link = screen.getByTestId("skip-to-content");
    expect(link).toHaveAttribute("href", "#app-content");
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
