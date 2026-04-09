import { useMutation, useQueryClient } from "@tanstack/react-query";

import { enableSkill, disableSkill } from "../adapters/skill-api";
import { skillKeys } from "../lib/query-keys";

interface SkillActionParams {
  name: string;
  workspace: string;
}

function invalidateSkillQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  name: string,
  workspace: string
) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: skillKeys.list(workspace) }),
    queryClient.invalidateQueries({ queryKey: skillKeys.detail(name, workspace) }),
    queryClient.invalidateQueries({ queryKey: skillKeys.content(name, workspace) }),
  ]);
}

export function useEnableSkill() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, workspace }: SkillActionParams) => enableSkill(name, workspace),
    onSettled: (_data, _error, { name, workspace }) =>
      invalidateSkillQueries(queryClient, name, workspace),
  });
}

export function useDisableSkill() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, workspace }: SkillActionParams) => disableSkill(name, workspace),
    onSettled: (_data, _error, { name, workspace }) =>
      invalidateSkillQueries(queryClient, name, workspace),
  });
}
