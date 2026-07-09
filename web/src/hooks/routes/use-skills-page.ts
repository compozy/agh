import { useCallback, useMemo } from "react";
import { useChildMatches, useNavigate } from "@tanstack/react-router";

import type { ListingViewMode } from "@agh/ui";

import {
  useDisableSkill,
  useEnableSkill,
  useInstallSkillMarketplace,
  useRemoveSkillMarketplace,
  useSkillMarketplaceSearch,
  useSkills,
  useUpdateSkillMarketplace,
} from "@/systems/skill";
import {
  parseSkillEnabledFilter,
  parseSkillSourceFilter,
  type SkillEnabledFilter,
  type SkillSourceFilter,
} from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";
import { normalizeListingSearchValue } from "@/lib/listing-search";

type Tab = "installed" | "marketplace";

export interface SkillsRouteSearch {
  tab?: Tab;
  q?: string;
  view?: ListingViewMode;
  source?: SkillSourceFilter;
  enabled?: SkillEnabledFilter;
}

function useSkillsPage(search: SkillsRouteSearch = {}) {
  const navigate = useNavigate({ from: "/skills" });
  const childMatches = useChildMatches();
  const hasChildMatch = childMatches.length > 0;

  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";

  const activeTab = search.tab ?? "installed";
  const searchQuery = search.q ?? "";
  const view: ListingViewMode = search.view ?? "rows";
  const sourceFilter = search.source ?? null;
  const enabledFilter = search.enabled ?? null;

  const skillsQuery = useSkills(workspaceId);
  const skills = useMemo(() => skillsQuery.data ?? [], [skillsQuery.data]);

  const disableMutation = useDisableSkill();
  const enableMutation = useEnableSkill();
  const installMutation = useInstallSkillMarketplace();
  const updateMutation = useUpdateSkillMarketplace();
  const removeMutation = useRemoveSkillMarketplace();

  const installedSkillNames = useMemo(() => {
    return new Set(skills.map(skill => skill.name));
  }, [skills]);

  const marketplaceQueryActive = activeTab === "marketplace" && searchQuery.trim() !== "";
  const marketplaceSearchQuery = useSkillMarketplaceSearch(
    marketplaceQueryActive ? searchQuery : ""
  );
  const marketplaceListings = useMemo(
    () => (marketplaceQueryActive ? (marketplaceSearchQuery.data ?? []) : []),
    [marketplaceQueryActive, marketplaceSearchQuery.data]
  );
  const marketplaceListingCount = marketplaceListings.length;

  const handleDisable = useCallback(
    (name: string) => {
      disableMutation.mutate({ name, workspace: workspaceId });
    },
    [disableMutation, workspaceId]
  );

  const handleEnable = useCallback(
    (name: string) => {
      enableMutation.mutate({ name, workspace: workspaceId });
    },
    [enableMutation, workspaceId]
  );

  const handleInstallMarketplace = useCallback(
    (slug: string) => {
      installMutation.mutate({ body: { slug }, workspace: workspaceId });
    },
    [installMutation, workspaceId]
  );

  const handleUpdateMarketplace = useCallback(
    (name: string) => {
      updateMutation.mutate({ body: { name }, workspace: workspaceId });
    },
    [updateMutation, workspaceId]
  );

  const handleRemoveMarketplace = useCallback(
    (name: string) => {
      removeMutation.mutate({ name, workspace: workspaceId });
    },
    [removeMutation, workspaceId]
  );

  const updateSearch = useCallback(
    (updater: (current: SkillsRouteSearch) => SkillsRouteSearch) => {
      void navigate({
        search: current => updater((current as SkillsRouteSearch | undefined) ?? {}),
        to: "/skills",
      });
    },
    [navigate]
  );

  const setActiveTab = useCallback(
    (nextTab: Tab) => {
      updateSearch(current => ({
        ...current,
        tab: nextTab === "installed" ? undefined : nextTab,
        // Clear installed-only filters when switching to marketplace.
        source: nextTab === "marketplace" ? undefined : current.source,
        enabled: nextTab === "marketplace" ? undefined : current.enabled,
      }));
    },
    [updateSearch]
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

  const setSourceFilter = useCallback(
    (next: SkillSourceFilter | null) => {
      updateSearch(current => ({
        ...current,
        source: next ?? undefined,
      }));
    },
    [updateSearch]
  );

  const setEnabledFilter = useCallback(
    (next: SkillEnabledFilter | null) => {
      updateSearch(current => ({
        ...current,
        enabled: next ?? undefined,
      }));
    },
    [updateSearch]
  );

  const clearFilters = useCallback(() => {
    updateSearch(current => ({
      ...current,
      q: undefined,
      source: undefined,
      enabled: undefined,
    }));
  }, [updateSearch]);

  const handleRefresh = useCallback(() => {
    void skillsQuery.refetch();
    if (marketplaceQueryActive) {
      void marketplaceSearchQuery.refetch();
    }
  }, [marketplaceQueryActive, marketplaceSearchQuery, skillsQuery]);

  const browseMarketplace = useCallback(() => {
    setActiveTab("marketplace");
  }, [setActiveTab]);

  const hasSkills = skills.length > 0;
  const error = skillsQuery.error && !hasSkills ? skillsQuery.error : null;
  const backgroundError = skillsQuery.error && hasSkills ? skillsQuery.error : null;

  return {
    activeTab,
    backgroundError,
    browseMarketplace,
    clearFilters,
    enabledFilter,
    error,
    handleDisable,
    handleEnable,
    handleInstallMarketplace,
    handleRefresh,
    handleRemoveMarketplace,
    handleUpdateMarketplace,
    hasChildMatch,
    installedSkillNames,
    isActionPending: disableMutation.isPending || enableMutation.isPending,
    isInstalling: installMutation.isPending,
    isLoading: skillsQuery.isLoading && !hasSkills,
    isMarketplaceSearchEnabled: marketplaceQueryActive,
    isMarketplaceSearching: marketplaceQueryActive && marketplaceSearchQuery.isFetching,
    isRemoving: removeMutation.isPending,
    isUpdating: updateMutation.isPending,
    marketplaceListingCount,
    marketplaceListings,
    marketplaceSearchError: marketplaceQueryActive ? (marketplaceSearchQuery.error ?? null) : null,
    searchQuery,
    setActiveTab,
    setEnabledFilter,
    setSearchQuery,
    setSourceFilter,
    setView,
    skillCount: skills.length,
    skills,
    sourceFilter,
    view,
    workspaceId,
  };
}

export { parseSkillEnabledFilter, parseSkillSourceFilter, useSkillsPage };
