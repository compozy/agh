import { useState } from "react";

import {
  useDisableSkill,
  useEnableSkill,
  useSkill,
  useSkillContent,
  useSkillShadows,
} from "@/systems/skill";
import { useActiveWorkspace } from "@/systems/workspace";

function useMarketplaceDetailSkillManage(name: string) {
  const { activeWorkspaceId } = useActiveWorkspace();
  const workspaceId = activeWorkspaceId ?? "";
  const [contentRequested, setContentRequested] = useState(false);
  const skillQuery = useSkill(name, workspaceId);
  const contentQuery = useSkillContent(name, workspaceId, contentRequested);
  const shadowsQuery = useSkillShadows(name, workspaceId);
  const enableMutation = useEnableSkill();
  const disableMutation = useDisableSkill();
  const skill = skillQuery.data;
  const acting = enableMutation.isPending || disableMutation.isPending;

  return {
    acting,
    contentQuery,
    contentRequested,
    setContentRequested,
    shadowsQuery,
    skill,
    toggleEnabled: (next: boolean) => {
      if (acting || !skill) return;
      if (next) enableMutation.mutate({ name, workspace: workspaceId });
      else disableMutation.mutate({ name, workspace: workspaceId });
    },
  };
}

export { useMarketplaceDetailSkillManage };
