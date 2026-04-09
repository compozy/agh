import { queryOptions } from "@tanstack/react-query";

import { getSkill, listSkills } from "../adapters/skill-api";
import { skillKeys } from "./query-keys";

export function skillsListOptions(workspace: string) {
  return queryOptions({
    queryKey: skillKeys.list(workspace),
    queryFn: ({ signal }) => listSkills(workspace, signal),
    staleTime: 30_000,
    refetchInterval: 60_000,
    enabled: !!workspace,
  });
}

export function skillDetailOptions(name: string, workspace: string) {
  return queryOptions({
    queryKey: skillKeys.detail(name, workspace),
    queryFn: ({ signal }) => getSkill(name, workspace, signal),
    staleTime: 30_000,
    enabled: !!name && !!workspace,
  });
}
