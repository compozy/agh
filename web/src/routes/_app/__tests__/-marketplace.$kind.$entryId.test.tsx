import { useEffect, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useTopbarSlotValue, type TopbarSlotValue } from "@agh/ui";
import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeBeforeLoad, routeComponent } from "@/test/route-options";
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
      beforeLoad?: (args: unknown) => unknown;
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => Record<string, unknown>;
    }) => ({
      beforeLoad: options.beforeLoad,
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

function TopbarSlotProbe({ onSlot }: { onSlot: (slot: TopbarSlotValue | null) => void }) {
  const slot = useTopbarSlotValue();
  useEffect(() => {
    onSlot(slot);
  }, [onSlot, slot]);
  return null;
}

function renderRoute(ui: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  let slotValue: TopbarSlotValue | null = null;
  const utils = renderWithTopbar(
    <QueryClientProvider client={client}>
      {ui}
      <TopbarSlotProbe onSlot={slot => (slotValue = slot)} />
    </QueryClientProvider>
  );
  return { ...utils, getSlot: () => slotValue };
}

beforeEach(() => {
  router.childMatches = [];
  router.params = { kind: "skill", entryId: "git-flow" };
  router.search = {};
});

describe("Marketplace detail topbar ownership", () => {
  it("Should register the Marketplace topbar crumb contract", () => {
    expect(routeBeforeLoad(MarketplaceRoute)()).toMatchObject({
      topbar: { crumb: { label: "Marketplace", to: "/marketplace" } },
    });
  });

  it("Should register the Marketplace kind entry parent-crumb and leaf-crumb contract", () => {
    expect(
      routeBeforeLoad<{ params: { kind: string; entryId: string } }>(MarketplaceDetailRoute)({
        params: { kind: "skill", entryId: "git-flow" },
      })
    ).toMatchObject({
      topbar: {
        parentCrumb: { label: "Skills", search: { kind: "skills" }, to: "/marketplace" },
        crumb: { label: "git-flow" },
      },
    });
  });

  it("Should clear the parent topbar slot when a detail child match is active", () => {
    router.childMatches = [{ id: "marketplace-detail" }];
    const { getSlot } = renderRoute(<MarketplacePage />);

    expect(screen.queryByTestId("marketplace-kind-navigation")).not.toBeInTheDocument();
    expect(screen.queryByTestId("marketplace-refresh")).not.toBeInTheDocument();
    expect(screen.getByTestId("marketplace-outlet")).toBeInTheDocument();
    expect(getSlot()).toBeNull();
  });

  it("Should publish the entry display name as the leaf breadcrumb override from the detail route", () => {
    const { getSlot } = renderRoute(<MarketplaceDetailPage />);

    expect(getSlot()?.crumb).toBe("git-flow");
    expect(screen.getByTestId("marketplace-detail")).toBeInTheDocument();
  });
});
