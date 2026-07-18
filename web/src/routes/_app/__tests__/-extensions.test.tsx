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

function normalizeSearch(value: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value)) {
    if (entry !== undefined) out[key] = entry;
  }
  return out;
}

function searchMatchesExactly(
  current: Record<string, unknown>,
  next: Record<string, unknown>
): boolean {
  const currentNorm = normalizeSearch(current);
  const nextNorm = normalizeSearch(next);
  const nextKeys = Object.keys(nextNorm);
  return (
    Object.keys(currentNorm).length === nextKeys.length &&
    nextKeys.every(key => currentNorm[key] === nextNorm[key])
  );
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    activeOptions,
    children,
    search,
    to,
    ...props
  }: {
    activeOptions?: { exact?: boolean; includeSearch?: boolean };
    children?: ReactNode;
    search?:
      | Record<string, unknown>
      | ((current: Record<string, unknown>) => Record<string, unknown>);
    to: string;
  }) => {
    const current = (router.validateSearch?.(router.search) ?? router.search) as Record<
      string,
      unknown
    >;
    const next =
      typeof search === "function" ? search(current) : ((search ?? {}) as Record<string, unknown>);
    const exact = activeOptions?.exact ?? false;
    const includeSearch = activeOptions?.includeSearch ?? true;
    const isActive =
      includeSearch && exact
        ? searchMatchesExactly(current, next)
        : includeSearch
          ? Object.keys(normalizeSearch(next)).every(
              key => normalizeSearch(current)[key] === normalizeSearch(next)[key]
            )
          : true;
    const query = new URLSearchParams(
      Object.fromEntries(
        Object.entries(normalizeSearch(next)).map(([key, value]) => [key, String(value)])
      )
    ).toString();
    return (
      <a
        href={`${to}${query ? `?${query}` : ""}`}
        aria-current={isActive ? "page" : undefined}
        {...props}
      >
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
    renderWithTopbar(<ExtensionsPage />);

    expect(screen.getByTestId("extensions-inventory-tab")).toHaveTextContent("bundles");
    expect(screen.getByRole("link", { name: "Bundles" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Extensions" })).not.toHaveAttribute("aria-current");
    expect(screen.getByRole("button", { name: "Browse marketplace" })).toHaveAttribute(
      "href",
      "/marketplace?kind=bundles"
    );
  });
});
