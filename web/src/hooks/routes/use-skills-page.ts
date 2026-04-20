import { useCallback, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";

import {
  useDisableSkill,
  useEnableSkill,
  useSkill,
  useSkillContent,
  useSkills,
} from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";

type Tab = "installed" | "marketplace";

export interface SkillsRouteSearch {
  tab?: Tab;
  skill?: string;
  content?: string;
  q?: string;
}

function normalizeSearchValue(value: string | null | undefined): string | undefined {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

function useSkillsPage(search: SkillsRouteSearch = {}) {
  const navigate = useNavigate({ from: "/skills" });

  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";

  const activeTab = search.tab ?? "installed";
  const selectedSkillName = search.skill ?? null;
  const requestedSkillContentName = search.content ?? null;
  const searchQuery = search.q ?? "";

  const skillsQuery = useSkills(workspaceId);
  const skills = skillsQuery.data ?? [];
  const marketplaceSkills = useMemo(
    () => skills.filter(skill => skill.source === "marketplace"),
    [skills]
  );

  const effectiveSelectedName = useMemo(() => {
    if (selectedSkillName && skills.some(skill => skill.name === selectedSkillName)) {
      return selectedSkillName;
    }

    return skills[0]?.name ?? null;
  }, [selectedSkillName, skills]);

  const {
    data: selectedSkill,
    isLoading: isLoadingDetail,
    error: detailError,
  } = useSkill(effectiveSelectedName ?? "", workspaceId);

  const disableMutation = useDisableSkill();
  const enableMutation = useEnableSkill();

  const installedSkillNames = useMemo(() => {
    return new Set(skills.map(skill => skill.name));
  }, [skills]);

  const shouldLoadSelectedContent =
    effectiveSelectedName !== null && requestedSkillContentName === effectiveSelectedName;

  const {
    data: selectedSkillContent,
    isLoading: isLoadingContent,
    error: contentError,
    refetch: refetchSkillContent,
  } = useSkillContent(effectiveSelectedName ?? "", workspaceId, shouldLoadSelectedContent);

  const handleDisable = (name: string) => {
    disableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleEnable = (name: string) => {
    enableMutation.mutate({ name, workspace: workspaceId });
  };

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
      }));
    },
    [updateSearch]
  );

  const setSelectedSkillName = useCallback(
    (nextSkillName: string | null) => {
      updateSearch(current => ({
        ...current,
        content: current.content === nextSkillName ? current.content : undefined,
        skill: normalizeSearchValue(nextSkillName),
      }));
    },
    [updateSearch]
  );

  const setSearchQuery = useCallback(
    (nextQuery: string) => {
      updateSearch(current => ({
        ...current,
        q: normalizeSearchValue(nextQuery),
      }));
    },
    [updateSearch]
  );

  const handleViewContent = (name: string) => {
    updateSearch(current => ({
      ...current,
      skill: normalizeSearchValue(name),
      content: normalizeSearchValue(name),
    }));
  };

  const handleRetryContent = () => {
    void refetchSkillContent();
  };

  const hasSkills = skills.length > 0;
  const error = skillsQuery.error && !hasSkills ? skillsQuery.error : null;
  const backgroundError = skillsQuery.error && hasSkills ? skillsQuery.error : null;

  return {
    activeTab,
    backgroundError,
    contentError: shouldLoadSelectedContent ? contentError : null,
    detailError,
    effectiveSelectedName,
    error,
    handleDisable,
    handleEnable,
    handleRetryContent,
    handleViewContent,
    installedSkillNames,
    isActionPending: disableMutation.isPending || enableMutation.isPending,
    isContentLoading: shouldLoadSelectedContent && isLoadingContent,
    isLoading: skillsQuery.isLoading && !hasSkills,
    isLoadingDetail: isLoadingDetail && effectiveSelectedName !== null,
    marketplaceSkillCount: marketplaceSkills.length,
    marketplaceSkills,
    searchQuery,
    selectedSkill: effectiveSelectedName
      ? (selectedSkill ?? skills.find(skill => skill.name === effectiveSelectedName))
      : undefined,
    selectedSkillContent: shouldLoadSelectedContent ? selectedSkillContent : undefined,
    setActiveTab,
    setSearchQuery,
    setSelectedSkillName,
    skillCount: skills.length,
    skills,
  };
}

export { useSkillsPage };
