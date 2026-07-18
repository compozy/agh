import { useEffect, type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useTopbarSlotValue, type TopbarSlotValue } from "@agh/ui";
import { renderWithTopbar } from "@/test/render-with-topbar";
import { routeBeforeLoad, routeComponent } from "@/test/route-options";
import { marketplaceDetails } from "@/systems/marketplace/mocks";

const router = vi.hoisted(() => ({
  childMatches: [] as { routeId: string }[],
  params: { kind: "skill", entryId: "git-flow" },
  pathname: "/marketplace/skills",
  search: {} as Record<string, unknown>,
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    activeOptions: _activeOptions,
    children,
    search,
    to,
    ...props
  }: {
    children?: ReactNode;
    activeOptions?: Record<string, unknown>;
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
  redirect: (opts: { to: string }) => {
    const error = new Error("REDIRECT") as Error & { to: string };
    error.to = opts.to;
    throw error;
  },
  useChildMatches: () => router.childMatches,
  useNavigate: () => vi.fn(),
  useRouterState: ({ select }: { select: (state: { location: { pathname: string } }) => string }) =>
    select({ location: { pathname: router.pathname } }),
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
      handleDeactivate: vi.fn(),
      handleRemove: vi.fn(),
      handleToggleEnabled: vi.fn(),
      isEntryFlashing: () => false,
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
import { Route as MarketplaceBundlesRoute } from "../marketplace.bundles";
import { Route as MarketplaceBundleActivationRoute } from "../marketplace.bundles.activations.$id";
import { Route as MarketplaceExtensionsRoute } from "../marketplace.extensions";
import { Route as MarketplaceIndexRoute } from "../marketplace.index";
import { Route as MarketplaceMcpsRoute } from "../marketplace.mcps";
import { Route as MarketplaceSkillsRoute } from "../marketplace.skills";

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
  router.pathname = "/marketplace/skills";
  router.search = {};
});

describe("Marketplace layout and detail topbar ownership", () => {
  it("Should redirect only the dedicated /marketplace index route to /marketplace/skills", () => {
    expect(() => routeBeforeLoad(MarketplaceIndexRoute)()).toThrow();
  });

  it("Should register the stable Marketplace layout crumb without redirect behavior", () => {
    expect(routeBeforeLoad(MarketplaceRoute)()).toMatchObject({
      topbar: { crumb: { label: "Marketplace", to: "/marketplace" } },
    });
  });

  it("Should return the same Marketplace layout context reference across invocations", () => {
    const beforeLoad = routeBeforeLoad(MarketplaceRoute);
    const first = beforeLoad();
    const second = beforeLoad();

    expect(first).toBe(second);
  });

  it.each([
    [MarketplaceSkillsRoute, "Skills"],
    [MarketplaceMcpsRoute, "MCPs"],
    [MarketplaceExtensionsRoute, "Extensions"],
    [MarketplaceBundlesRoute, "Bundles"],
  ])("Should register a stable kind crumb for %s", (route, label) => {
    const beforeLoad = routeBeforeLoad(route);
    const first = beforeLoad();
    const second = beforeLoad();

    expect(first).toBe(second);
    expect(first).toMatchObject({ topbar: { crumb: { label } } });
  });

  it("Should register the Marketplace kind entry parent-crumb and leaf-crumb contract", () => {
    expect(
      routeBeforeLoad<{ params: { kind: string; entryId: string } }>(MarketplaceDetailRoute)({
        params: { kind: "skill", entryId: "git-flow" },
      })
    ).toMatchObject({
      topbar: {
        parentCrumb: { label: "Skills", to: "/marketplace/skills" },
        crumb: { label: "git-flow" },
      },
    });
  });

  it("Should let the nested Bundles route own the activation parent crumb", () => {
    expect(
      routeBeforeLoad<{ params: { id: string } }>(MarketplaceBundleActivationRoute)({
        params: { id: "activation-ops-starter" },
      })
    ).toEqual({
      topbar: { crumb: { label: "activation-ops-starter" } },
    });
  });

  it("Should publish RouteNav when rendering a kind layout page", () => {
    const { getSlot } = renderRoute(<MarketplacePage />);
    expect(screen.getByTestId("marketplace-kind-navigation")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-refresh")).toBeInTheDocument();
    expect(screen.getByTestId("marketplace-outlet")).toBeInTheDocument();
    expect(getSlot()?.routeNav).toBeTruthy();
  });

  it("Should clear the parent topbar slot when a detail child match is active", () => {
    router.childMatches = [{ routeId: "/_app/marketplace/$kind/$entryId" }];
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
