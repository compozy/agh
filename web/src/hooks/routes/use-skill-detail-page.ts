import { useCallback, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";

import {
  useDisableSkill,
  useEnableSkill,
  useSkill,
  useSkillContent,
  useSkillShadows,
  useSkills,
} from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";

export interface SkillDetailRouteSearch {
  content?: string;
}

function useSkillDetailPage(name: string, search: SkillDetailRouteSearch = {}) {
  const navigate = useNavigate();
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";

  const requestedContent = search.content === name;

  const skillsQuery = useSkills(workspaceId);
  const skills = useMemo(() => skillsQuery.data ?? [], [skillsQuery.data]);
  const listSkill = useMemo(() => skills.find(skill => skill.name === name), [name, skills]);

  const {
    data: selectedSkill,
    isLoading: isLoadingDetail,
    error: detailError,
  } = useSkill(name, workspaceId);
  const {
    data: selectedSkillShadows,
    error: shadowsError,
    isLoading: isLoadingShadows,
  } = useSkillShadows(name, workspaceId);

  const {
    data: selectedSkillContent,
    isLoading: isLoadingContent,
    error: contentError,
    refetch: refetchSkillContent,
  } = useSkillContent(name, workspaceId, requestedContent);

  const disableMutation = useDisableSkill();
  const enableMutation = useEnableSkill();

  const handleDisable = useCallback(() => {
    disableMutation.mutate({ name, workspace: workspaceId });
  }, [disableMutation, name, workspaceId]);

  const handleEnable = useCallback(() => {
    enableMutation.mutate({ name, workspace: workspaceId });
  }, [enableMutation, name, workspaceId]);

  const handleViewContent = useCallback(() => {
    void navigate({
      to: "/skills/$name",
      params: { name },
      search: prev => ({
        ...(prev as SkillDetailRouteSearch),
        content: name,
      }),
    });
  }, [name, navigate]);

  const handleRetryContent = useCallback(() => {
    void refetchSkillContent();
  }, [refetchSkillContent]);

  const handleBack = useCallback(() => {
    void navigate({ to: "/skills" });
  }, [navigate]);

  return {
    contentError: requestedContent ? contentError : null,
    detailError,
    handleBack,
    handleDisable,
    handleEnable,
    handleRetryContent,
    handleViewContent,
    isActionPending: disableMutation.isPending || enableMutation.isPending,
    isContentLoading: requestedContent && isLoadingContent,
    isLoadingDetail,
    isLoadingShadows,
    selectedSkill: selectedSkill ?? listSkill,
    selectedSkillContent: requestedContent ? selectedSkillContent : undefined,
    selectedSkillShadows,
    shadowsError,
    workspaceId,
  };
}

export { useSkillDetailPage };
