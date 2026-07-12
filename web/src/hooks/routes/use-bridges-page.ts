import { useCallback, useMemo } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";

import type { ListingViewMode } from "@agh/ui";

import { normalizeListingSearchValue } from "@/lib/listing-search";
import {
  bridgeListFilterForScope,
  useBridgeHealthStream,
  useBridgeProviders,
  useBridges,
} from "@/systems/bridges";
import type { BridgeScopeFilter } from "@/systems/bridges";
import {
  parseBridgePlatformFilter,
  parseBridgeScopeFilter,
  parseBridgeStatusFilter,
  type BridgePlatformFilter,
  type BridgeStatusFilter,
} from "@/systems/bridges/lib/bridge-list-filters";
import { useActiveWorkspace } from "@/systems/workspace";
import { useBridgeCreateFlow } from "./use-bridge-create-flow";

export interface BridgesRouteSearch {
  q?: string;
  view?: ListingViewMode;
  scope?: BridgeScopeFilter;
  platform?: BridgePlatformFilter;
  status?: BridgeStatusFilter;
}

function useBridgesPage(search: BridgesRouteSearch = {}) {
  const navigate = useNavigate({ from: "/bridges" });
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;

  const { activeWorkspace, activeWorkspaceId } = useActiveWorkspace();

  const searchQuery = search.q ?? "";
  const view: ListingViewMode = search.view ?? "rows";
  const scopeFilter: BridgeScopeFilter = search.scope ?? "all";
  const platformFilter = search.platform ?? null;
  const statusFilter = search.status ?? null;

  const bridgeListFilters = useMemo(
    () => ({
      ...bridgeListFilterForScope(scopeFilter, activeWorkspaceId),
      q: searchQuery,
      platform: platformFilter ?? undefined,
      status: statusFilter ?? undefined,
      sort: "name" as const,
      limit: 50,
    }),
    [activeWorkspaceId, platformFilter, scopeFilter, searchQuery, statusFilter]
  );
  const bridgeListEnabled = scopeFilter !== "workspace" || Boolean(activeWorkspaceId);

  const bridgesQuery = useBridges(bridgeListFilters, { enabled: bridgeListEnabled });
  useBridgeHealthStream({
    bridgeIds: bridgesQuery.bridges.map(bridge => bridge.id),
    enabled: bridgeListEnabled,
    filters: bridgeListFilters,
  });
  const providersQuery = useBridgeProviders();

  const bridges = bridgesQuery.bridges;
  const bridgeHealth = bridgesQuery.bridgeHealth;
  const providers = providersQuery.data ?? [];
  const createFlow = useBridgeCreateFlow({
    activeWorkspaceId,
    activeWorkspaceName: activeWorkspace?.name,
    providers,
  });

  const platforms = Object.keys(bridgesQuery.facets?.platforms ?? {});
  const statuses: BridgeStatusFilter[] = [];
  for (const [status, count] of Object.entries(bridgesQuery.facets?.statuses ?? {})) {
    if (count > 0 || status === statusFilter) statuses.push(status as BridgeStatusFilter);
  }
  const totalBridgeCount = bridgesQuery.total;
  const hasActiveFilters =
    searchQuery.trim() !== "" ||
    scopeFilter !== "all" ||
    platformFilter !== null ||
    statusFilter !== null;

  const isInitialLoading =
    (bridgesQuery.isLoading && !bridgesQuery.data) ||
    (providersQuery.isLoading && !providersQuery.data);
  const fatalError =
    (!bridgesQuery.data && bridgesQuery.error) || (!providersQuery.data && providersQuery.error);
  const backgroundError =
    bridgesQuery.error && bridgesQuery.data
      ? bridgesQuery.error
      : providersQuery.error && providersQuery.data
        ? providersQuery.error
        : null;

  const updateSearch = useCallback(
    (updater: (current: BridgesRouteSearch) => BridgesRouteSearch) => {
      void navigate({
        search: current => updater((current as BridgesRouteSearch | undefined) ?? {}),
        to: "/bridges",
      });
    },
    [navigate]
  );

  const setSearchQuery = useCallback(
    (nextQuery: string) => {
      updateSearch(current => ({
        ...current,
        q: normalizeListingSearchValue(nextQuery),
      }));
    },
    [updateSearch]
  );

  const setView = useCallback(
    (nextView: ListingViewMode) => {
      updateSearch(current => ({
        ...current,
        view: nextView === "rows" ? undefined : nextView,
      }));
    },
    [updateSearch]
  );

  const setScopeFilter = useCallback(
    (next: BridgeScopeFilter) => {
      updateSearch(current => ({
        ...current,
        scope: next === "all" ? undefined : next,
      }));
    },
    [updateSearch]
  );

  const setPlatformFilter = useCallback(
    (next: BridgePlatformFilter | null) => {
      updateSearch(current => ({
        ...current,
        platform: next ?? undefined,
      }));
    },
    [updateSearch]
  );

  const setStatusFilter = useCallback(
    (next: BridgeStatusFilter | null) => {
      updateSearch(current => ({
        ...current,
        status: next ?? undefined,
      }));
    },
    [updateSearch]
  );

  const clearFilters = useCallback(() => {
    updateSearch(current => ({
      ...current,
      platform: undefined,
      q: undefined,
      scope: undefined,
      status: undefined,
    }));
  }, [updateSearch]);

  const handleRefresh = useCallback(() => {
    void bridgesQuery.refetch();
    void providersQuery.refetch();
  }, [bridgesQuery, providersQuery]);

  return {
    activeWorkspaceId,
    backgroundError,
    bridgeHealth,
    bridges,
    canCreateBridge: createFlow.canCreateBridge,
    clearFilters,
    createDialogProps: createFlow.createDialogProps,
    fatalError,
    hasActiveFilters,
    hasNextPage: bridgesQuery.hasNextPage,
    handleRefresh,
    hasChildMatch,
    isInitialLoading,
    isFetchingNextPage: bridgesQuery.isFetchingNextPage,
    loadMore: bridgesQuery.fetchNextPage,
    openCreateDialog: createFlow.openCreateDialog,
    platformFilter,
    platforms,
    providers,
    scopeFilter,
    searchQuery,
    setPlatformFilter,
    setScopeFilter,
    setSearchQuery,
    setStatusFilter,
    setView,
    statusFilter,
    statuses,
    totalBridgeCount,
    view,
  };
}

export {
  parseBridgePlatformFilter,
  parseBridgeScopeFilter,
  parseBridgeStatusFilter,
  useBridgesPage,
};
