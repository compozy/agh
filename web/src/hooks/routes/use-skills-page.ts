import { useChildMatches, useNavigate } from "@tanstack/react-router";

import type { ListingViewMode } from "@agh/ui";

import { useDisableSkill, useEnableSkill, useSkills } from "@/systems/skill";
import {
  parseSkillEnabledFilter,
  parseSkillSourceFilter,
  type SkillEnabledFilter,
  type SkillSourceFilter,
} from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";
import { normalizeListingSearchValue } from "@/lib/listing-search";

export interface SkillsRouteSearch {
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

  const searchQuery = search.q ?? "";
  const view: ListingViewMode = search.view ?? "rows";
  const sourceFilter = search.source ?? null;
  const enabledFilter = search.enabled ?? null;

  const skillsQuery = useSkills(workspaceId);
  const skills = skillsQuery.data ?? [];

  const disableMutation = useDisableSkill();
  const enableMutation = useEnableSkill();

  const handleDisable = (name: string) => {
    disableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleEnable = (name: string) => {
    enableMutation.mutate({ name, workspace: workspaceId });
  };

  const updateSearch = (updater: (current: SkillsRouteSearch) => SkillsRouteSearch) => {
    void navigate({
      search: current => updater((current as SkillsRouteSearch | undefined) ?? {}),
      to: "/skills",
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
  };

  const hasSkills = skills.length > 0;
  const error = skillsQuery.error && !hasSkills ? skillsQuery.error : null;
  const backgroundError = skillsQuery.error && hasSkills ? skillsQuery.error : null;

  return {
    backgroundError,
    clearFilters,
    enabledFilter,
    error,
    handleDisable,
    handleEnable,
    handleRefresh,
    hasChildMatch,
    isActionPending: disableMutation.isPending || enableMutation.isPending,
    isLoading: skillsQuery.isLoading && !hasSkills,
    searchQuery,
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
