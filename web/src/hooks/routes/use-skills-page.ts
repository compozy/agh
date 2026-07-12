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
  const skills = skillsQuery.data ?? [];

  const disableMutation = useDisableSkill();
  const enableMutation = useEnableSkill();
  const installMutation = useInstallSkillMarketplace();
  const updateMutation = useUpdateSkillMarketplace();
  const removeMutation = useRemoveSkillMarketplace();

  const installedSkillNames = new Set(skills.map(skill => skill.name));

  const marketplaceQueryActive = activeTab === "marketplace" && searchQuery.trim() !== "";
  const marketplaceSearchQuery = useSkillMarketplaceSearch(
    marketplaceQueryActive ? searchQuery : ""
  );
  const marketplaceListings = marketplaceQueryActive ? (marketplaceSearchQuery.data ?? []) : [];
  const marketplaceListingCount = marketplaceListings.length;

  const handleDisable = (name: string) => {
    disableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleEnable = (name: string) => {
    enableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleInstallMarketplace = (slug: string) => {
    installMutation.mutate({ body: { slug }, workspace: workspaceId });
  };

  const handleUpdateMarketplace = (name: string) => {
    updateMutation.mutate({ body: { name }, workspace: workspaceId });
  };

  const handleRemoveMarketplace = (name: string) => {
    removeMutation.mutate({ name, workspace: workspaceId });
  };

  const updateSearch = (updater: (current: SkillsRouteSearch) => SkillsRouteSearch) => {
    void navigate({
      search: current => updater((current as SkillsRouteSearch | undefined) ?? {}),
      to: "/skills",
    });
  };

  const setActiveTab = (nextTab: Tab) => {
    updateSearch(current => ({
      ...current,
      tab: nextTab === "installed" ? undefined : nextTab,
      // Clear installed-only filters when switching to marketplace.
      source: nextTab === "marketplace" ? undefined : current.source,
      enabled: nextTab === "marketplace" ? undefined : current.enabled,
    }));
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

  const setSourceFilter = (next: SkillSourceFilter | null) => {
    updateSearch(current => ({
      ...current,
      source: next ?? undefined,
    }));
  };

  const setEnabledFilter = (next: SkillEnabledFilter | null) => {
    updateSearch(current => ({
      ...current,
      enabled: next ?? undefined,
    }));
  };

  const clearFilters = () => {
    updateSearch(current => ({
      ...current,
      q: undefined,
      source: undefined,
      enabled: undefined,
    }));
  };

  const handleRefresh = () => {
    void skillsQuery.refetch();
    if (marketplaceQueryActive) {
      void marketplaceSearchQuery.refetch();
    }
  };

  const browseMarketplace = () => {
    setActiveTab("marketplace");
  };

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
