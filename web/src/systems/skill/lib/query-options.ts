import { queryOptions } from "@tanstack/react-query";

import {
  getSkill,
  getSkillContent,
  getSkillMarketplaceInfo,
  listSkills,
  searchSkillMarketplace,
} from "../adapters/skill-api";
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

export function skillContentOptions(name: string, workspace: string, enabled: boolean) {
  return queryOptions({
    queryKey: skillKeys.content(name, workspace),
    queryFn: ({ signal }) => getSkillContent(name, workspace, signal),
    staleTime: 30_000,
    enabled: enabled && !!name && !!workspace,
  });
}

export function skillMarketplaceSearchOptions(query: string, limit?: number) {
  const trimmed = query.trim();
  return queryOptions({
    queryKey: skillKeys.marketplaceSearch(trimmed, limit),
    queryFn: ({ signal }) => searchSkillMarketplace({ query: trimmed, limit }, signal),
    staleTime: 30_000,
    enabled: trimmed !== "",
  });
}

export function skillMarketplaceInfoOptions(slug: string, enabled = true) {
  const normalizedSlug = slug.trim();
  return queryOptions({
    queryKey: skillKeys.marketplaceInfo(normalizedSlug),
    queryFn: ({ signal }) => getSkillMarketplaceInfo(normalizedSlug, signal),
    staleTime: 30_000,
    enabled: enabled && normalizedSlug !== "",
  });
}
