import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeComponent } from "@/test/route-options";
import { marketplaceDetails } from "@/systems/marketplace/mocks";

const router = vi.hoisted(() => ({
  childMatches: [] as unknown[],
  params: { kind: "skill", entryId: "git-flow" },
  search: {} as Record<string, unknown>,
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
  Outlet: () => <div data-testid="marketplace-outlet" />,
  createFileRoute:
    () =>
    (options: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => ({
      component: options.component,
      useParams: () => router.params,
      useSearch: () => options.validateSearch?.(router.search) ?? router.search,
    }),
  useChildMatches: () => router.childMatches,
  useNavigate: () => vi.fn(),
}));

vi.mock("@/systems/marketplace", async importOriginal => {
  const actual = await importOriginal<typeof import("@/systems/marketplace")>();
  return {
    ...actual,
    MarketplaceDetail: () => <div data-testid="marketplace-detail" />,
    MarketplaceDetailSkeleton: () => <div data-testid="marketplace-detail-skeleton" />,
    useMarketplaceActionController: () => ({
      dialogs: null,
      handleAction: vi.fn(),
      isEntryPending: () => false,
    }),
    useMarketplaceEntry: () => ({
      data: marketplaceDetails["skill:git-flow"],
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    }),
    useRefreshMarketplaceCatalog: () => ({
      isPending: false,
      mutate: vi.fn(),
    }),
  };
});

vi.mock("@/systems/workspace", () => ({
  useActiveWorkspace: () => ({ activeWorkspaceId: "ws-test" }),
}));

import { Route as MarketplaceRoute } from "../marketplace";
import { Route as MarketplaceDetailRoute } from "../marketplace.$kind.$entryId";

const MarketplacePage = routeComponent(MarketplaceRoute);
const MarketplaceDetailPage = routeComponent(MarketplaceDetailRoute);

function renderRoute(ui: ReactNode, title: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithTopbar(<QueryClientProvider client={client}>{ui}</QueryClientProvider>, {
    title,
  });
}

beforeEach(() => {
  router.childMatches = [];
  router.params = { kind: "skill", entryId: "git-flow" };
  router.search = {};
});

describe("Marketplace detail topbar ownership", () => {
  it("Should clear the parent topbar slot when a detail child match is active", () => {
    router.childMatches = [{ id: "marketplace-detail" }];
    renderRoute(<MarketplacePage />, "Marketplace");

    expect(screen.queryByTestId("marketplace-kind-navigation")).not.toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-refresh")).not.toBeInTheDocument();
    expect(screen.getByTestId("marketplace-outlet")).toBeInTheDocument();
    expect(screen.getByTestId("topbar-title-text")).toHaveTextContent("Marketplace");
  });

  it("Should publish the Marketplace kind entry breadcrumb from the detail route", () => {
    renderRoute(<MarketplaceDetailPage />, "git-flow");

    expect(screen.getByLabelText("Marketplace, Skills, git-flow")).toBeInTheDocument();
    expect(screen.getByTestId("topbar-title-text")).toHaveTextContent("Marketplace");
    expect(screen.getByTestId("topbar-title-text")).toHaveTextContent("Skills");
    expect(screen.getByTestId("topbar-title-text")).toHaveTextContent("git-flow");
    expect(screen.getByTestId("marketplace-detail")).toBeInTheDocument();
  });
});
