import type { ReactNode } from "react";
import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithTopbar } from "@/test/render-with-topbar";

const router = vi.hoisted(() => ({
  search: { tab: "bundles" as "bundles" | undefined },
  validateSearch: undefined as
    | ((search: Record<string, unknown>) => { tab?: "bundles" })
    | undefined,
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    search,
    to,
    ...props
  }: {
    children?: ReactNode;
    search?: Record<string, string>;
    to: string;
  }) => {
    const query = search ? new URLSearchParams(search).toString() : "";
    return (
      <a href={`${to}${query ? `?${query}` : ""}`} {...props}>
        {children}
      </a>
    );
  },
  Outlet: () => <div data-testid="extensions-outlet" />,
  createFileRoute:
    () =>
    (options: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => { tab?: "bundles" };
    }) => {
      router.validateSearch = options.validateSearch;
      return {
        component: options.component,
        useSearch: () => router.validateSearch?.(router.search) ?? router.search,
      };
    },
  useChildMatches: () => [],
  useNavigate: () => vi.fn(),
}));

vi.mock("@/systems/extensions", () => ({
  ExtensionsInventory: ({ tab }: { tab: string }) => (
    <output data-testid="extensions-inventory-tab">{tab}</output>
  ),
  useBundleActivations: () => ({ data: [{ id: "bundle-a" }] }),
  useExtensionInventory: () => ({ data: [] }),
}));

import { routeComponent } from "@/test/route-options";
import { Route } from "../extensions";

const ExtensionsPage = routeComponent(Route);

beforeEach(() => {
  router.search = { tab: "bundles" };
});

describe("Extensions route", () => {
  it("Should honor the bundles deep-link and keep its marketplace CTA plural", () => {
    renderWithTopbar(<ExtensionsPage />, { title: "Extensions" });

    expect(screen.getByTestId("extensions-inventory-tab")).toHaveTextContent("bundles");
    expect(screen.getByRole("button", { name: "Bundles" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Browse marketplace" })).toHaveAttribute(
      "href",
      "/marketplace?kind=bundles"
    );
  });
});
