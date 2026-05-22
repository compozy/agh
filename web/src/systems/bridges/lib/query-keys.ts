import type { BridgeListFilter, BridgeTargetsQuery } from "../types";

function normalizeKeyValue(value?: string | number | null) {
  return value == null ? "" : String(value);
}

export const bridgeKeys = {
  all: ["bridges"] as const,
  lists: () => [...bridgeKeys.all, "list"] as const,
  list: (filters: BridgeListFilter = {}) =>
    [
      ...bridgeKeys.lists(),
      filters.scope ?? "all",
      normalizeKeyValue(filters.workspace_id),
      normalizeKeyValue(filters.workspace),
    ] as const,
  providers: () => [...bridgeKeys.all, "providers"] as const,
  details: () => [...bridgeKeys.all, "detail"] as const,
  detail: (id: string) => [...bridgeKeys.details(), normalizeKeyValue(id)] as const,
  routesRoot: () => [...bridgeKeys.all, "routes"] as const,
  routes: (id: string) => [...bridgeKeys.routesRoot(), normalizeKeyValue(id)] as const,
  targetsRoot: () => [...bridgeKeys.all, "targets"] as const,
  targets: (id: string, query: BridgeTargetsQuery = {}) =>
    [
      ...bridgeKeys.targetsRoot(),
      normalizeKeyValue(id),
      normalizeKeyValue(query.q),
      normalizeKeyValue(query.limit),
    ] as const,
  secretBindingsRoot: () => [...bridgeKeys.all, "secret-bindings"] as const,
  secretBindings: (id: string) =>
    [...bridgeKeys.secretBindingsRoot(), normalizeKeyValue(id)] as const,
};
