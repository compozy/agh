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
  const skills = skillsQuery.data ?? [];
  const listSkill = skills.find(skill => skill.name === name);

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

  const handleDisable = () => {
    disableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleEnable = () => {
    enableMutation.mutate({ name, workspace: workspaceId });
  };

  const handleViewContent = () => {
    void navigate({
      to: "/skills/$name",
      params: { name },
      search: prev => ({
        ...(prev as SkillDetailRouteSearch),
        content: name,
      }),
    });
  };

  const handleRetryContent = () => {
    void refetchSkillContent();
  };

  return {
    contentError: requestedContent ? contentError : null,
    detailError,
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
