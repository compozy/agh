import { useMutation, useQueryClient } from "@tanstack/react-query";

import { enableSkill, disableSkill } from "../adapters/skill-api";
import { skillKeys } from "../lib/query-keys";

interface SkillActionParams {
  name: string;
  workspace: string;
}

export function useEnableSkill() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, workspace }: SkillActionParams) => enableSkill(name, workspace),
    onSettled: (_data, _error, { workspace }) => {
      queryClient.invalidateQueries({ queryKey: skillKeys.list(workspace) });
    },
  });
}

export function useDisableSkill() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, workspace }: SkillActionParams) => disableSkill(name, workspace),
    onSettled: (_data, _error, { workspace }) => {
      queryClient.invalidateQueries({ queryKey: skillKeys.list(workspace) });
    },
  });
}
