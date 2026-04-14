function normalizeText(value?: string) {
  return value ?? "";
}

export const bridgeKeys = {
  all: ["bridges"] as const,
  lists: () => [...bridgeKeys.all, "list"] as const,
  list: () => [...bridgeKeys.lists(), "all"] as const,
  providers: () => [...bridgeKeys.all, "providers"] as const,
  details: () => [...bridgeKeys.all, "detail"] as const,
  detail: (id: string) => [...bridgeKeys.details(), normalizeText(id)] as const,
  routesRoot: () => [...bridgeKeys.all, "routes"] as const,
  routes: (id: string) => [...bridgeKeys.routesRoot(), normalizeText(id)] as const,
};
