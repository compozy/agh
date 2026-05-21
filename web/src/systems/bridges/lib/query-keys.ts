import type { BridgeListFilter, BridgeTargetsQuery } from "../types";

function normalizeText(value?: string) {
  return value ?? "";
}

export const bridgeKeys = {
  all: ["bridges"] as const,
  lists: () => [...bridgeKeys.all, "list"] as const,
  list: (filters: BridgeListFilter = {}) =>
    [
      ...bridgeKeys.lists(),
      filters.scope ?? "all",
      normalizeText(filters.workspace_id),
      normalizeText(filters.workspace),
    ] as const,
  providers: () => [...bridgeKeys.all, "providers"] as const,
  details: () => [...bridgeKeys.all, "detail"] as const,
  detail: (id: string) => [...bridgeKeys.details(), normalizeText(id)] as const,
  routesRoot: () => [...bridgeKeys.all, "routes"] as const,
  routes: (id: string) => [...bridgeKeys.routesRoot(), normalizeText(id)] as const,
  targetsRoot: () => [...bridgeKeys.all, "targets"] as const,
  targets: (id: string, query: BridgeTargetsQuery = {}) =>
    [
      ...bridgeKeys.targetsRoot(),
      normalizeText(id),
      normalizeText(query.q),
      normalizeText(query.limit),
    ] as const,
  secretBindingsRoot: () => [...bridgeKeys.all, "secret-bindings"] as const,
  secretBindings: (id: string) => [...bridgeKeys.secretBindingsRoot(), normalizeText(id)] as const,
};
