import { useQuery } from "@tanstack/react-query";

import {
  skillContentOptions,
  skillDetailOptions,
  skillMarketplaceInfoOptions,
  skillMarketplaceSearchOptions,
  skillsListOptions,
} from "@/systems/skill/lib/query-options";

export function useSkills(workspace: string) {
  return useQuery(skillsListOptions(workspace));
}

export function useSkill(name: string, workspace: string) {
  return useQuery(skillDetailOptions(name, workspace));
}

export function useSkillContent(name: string, workspace: string, enabled = false) {
  return useQuery(skillContentOptions(name, workspace, enabled));
}

export function useSkillMarketplaceSearch(query: string, limit?: number) {
  return useQuery(skillMarketplaceSearchOptions(query, limit));
}

export function useSkillMarketplaceInfo(slug: string, enabled = true) {
  return useQuery(skillMarketplaceInfoOptions(slug, enabled));
}
