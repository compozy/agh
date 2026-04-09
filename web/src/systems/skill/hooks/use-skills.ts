import { useQuery } from "@tanstack/react-query";

import { skillDetailOptions, skillsListOptions } from "../lib/query-options";

export function useSkills(workspace: string) {
  return useQuery(skillsListOptions(workspace));
}

export function useSkill(name: string, workspace: string) {
  return useQuery(skillDetailOptions(name, workspace));
}
