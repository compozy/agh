import { useState } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import type { ListingViewMode } from "@agh/ui";

import { normalizeListingSearchValue } from "@/lib/listing-search";
import {
  buildBridgeCreateRequest,
  bridgeListFilterForScope,
  createBridgeCreateDraft,
  findBridgeProviderByKey,
  isBridgeProviderSelectable,
  useBridgeHealthStream,
  useBridgeProviders,
  useBridges,
  useCreateBridge,
} from "@/systems/bridges";
import type { BridgeCreateDraft, BridgeScopeFilter } from "@/systems/bridges";
import {
  parseBridgePlatformFilter,
  parseBridgeScopeFilter,
  parseBridgeStatusFilter,
  type BridgePlatformFilter,
  type BridgeStatusFilter,
} from "@/systems/bridges/lib/bridge-list-filters";
import { useActiveWorkspace } from "@/systems/workspace";

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

  const [isCreateDialogOpen, setCreateDialogOpen] = useState(false);
  const [createDraft, setCreateDraft] = useState<BridgeCreateDraft>(() =>
    createBridgeCreateDraft([], activeWorkspaceId)
  );

  const bridgeListFilters = {
    ...bridgeListFilterForScope(scopeFilter, activeWorkspaceId),
    q: searchQuery,
    platform: platformFilter ?? undefined,
    status: statusFilter ?? undefined,
    sort: "name" as const,
    limit: 50,
  };
  const bridgeListEnabled = scopeFilter !== "workspace" || Boolean(activeWorkspaceId);

  const bridgesQuery = useBridges(bridgeListFilters, { enabled: bridgeListEnabled });
  useBridgeHealthStream({
    bridgeIds: bridgesQuery.bridges.map(bridge => bridge.id),
    enabled: bridgeListEnabled,
    filters: bridgeListFilters,
  });
  const providersQuery = useBridgeProviders();
  const createBridgeMutation = useCreateBridge();

  const bridges = bridgesQuery.bridges;
  const bridgeHealth = bridgesQuery.bridgeHealth;
  const providers = providersQuery.data ?? [];
  const canCreateBridge = providers.some(isBridgeProviderSelectable);

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

  const updateSearch = (updater: (current: BridgesRouteSearch) => BridgesRouteSearch) => {
    void navigate({
      search: current => updater((current as BridgesRouteSearch | undefined) ?? {}),
      to: "/bridges",
    });
  };

  const setSearchQuery = (nextQuery: string) => {
    updateSearch(current => ({
      ...current,
      q: normalizeListingSearchValue(nextQuery),
    }));
  };

  const setView = (nextView: ListingViewMode) => {
    updateSearch(current => ({
      ...current,
      view: nextView === "rows" ? undefined : nextView,
    }));
  };

  const setScopeFilter = (next: BridgeScopeFilter) => {
    updateSearch(current => ({
      ...current,
      scope: next === "all" ? undefined : next,
    }));
  };

  const setPlatformFilter = (next: BridgePlatformFilter | null) => {
    updateSearch(current => ({
      ...current,
      platform: next ?? undefined,
    }));
  };

  const setStatusFilter = (next: BridgeStatusFilter | null) => {
    updateSearch(current => ({
      ...current,
      status: next ?? undefined,
    }));
  };

  const clearFilters = () => {
    updateSearch(current => ({
      ...current,
      platform: undefined,
      q: undefined,
      scope: undefined,
      status: undefined,
    }));
  };

  const handleRefresh = () => {
    void bridgesQuery.refetch();
    void providersQuery.refetch();
  };

  const openCreateDialog = () => {
    setCreateDraft(createBridgeCreateDraft(providers, activeWorkspaceId));
    setCreateDialogOpen(true);
  };

  const handleCreateDialogOpenChange = (open: boolean) => {
    setCreateDialogOpen(open);
  };

  const handleCreateBridge = async () => {
    const provider = findBridgeProviderByKey(providers, createDraft.selectedProviderKey);
    if (!provider || !isBridgeProviderSelectable(provider)) {
      toast.error("Select an available bridge provider before creating the bridge.");
      return;
    }

    if (createDraft.scope === "workspace" && !activeWorkspaceId) {
      toast.error("Select an active workspace before creating a workspace-scoped bridge.");
      return;
    }

    const requestResult = buildBridgeCreateRequest(createDraft, provider, activeWorkspaceId);
    if (!requestResult.ok) {
      toast.error(requestResult.error);
      return;
    }

    try {
      const result = await createBridgeMutation.mutateAsync(requestResult.data);
      setCreateDialogOpen(false);
      toast.success(`Created bridge ${result.bridge.display_name}.`);
      void navigate({
        params: { id: result.bridge.id },
        to: "/bridges/$id",
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to create bridge");
    }
  };

  const createDialogProps = {
    activeWorkspaceId,
    activeWorkspaceName: activeWorkspace?.name,
    draft: createDraft,
    isPending: createBridgeMutation.isPending,
    onDraftChange: setCreateDraft,
    onOpenChange: handleCreateDialogOpenChange,
    onSubmit: handleCreateBridge,
    open: isCreateDialogOpen,
    providers,
  };

  return {
    activeWorkspaceId,
    backgroundError,
    bridgeHealth,
    bridges,
    canCreateBridge,
    clearFilters,
    createDialogProps,
    fatalError,
    hasActiveFilters,
    hasNextPage: bridgesQuery.hasNextPage,
    handleRefresh,
    hasChildMatch,
    isInitialLoading,
    isFetchingNextPage: bridgesQuery.isFetchingNextPage,
    loadMore: bridgesQuery.fetchNextPage,
    openCreateDialog,
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
