export const modelCatalogKeys = {
  all: ["model-catalog"] as const,
  providers: () => [...modelCatalogKeys.all, "providers"] as const,
  providerRoot: (providerId: string) =>
    [...modelCatalogKeys.providers(), providerId.trim()] as const,
  providerModels: (providerId: string, sourceId?: string, includeStale?: boolean) =>
    [
      ...modelCatalogKeys.providerRoot(providerId),
      "models",
      sourceId ?? "",
      Boolean(includeStale),
    ] as const,
  providerStatus: (providerId: string) =>
    [...modelCatalogKeys.providerRoot(providerId), "status"] as const,
};
