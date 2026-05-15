export const skillKeys = {
  all: ["skills"] as const,
  list: (workspace: string) => [...skillKeys.all, "list", workspace] as const,
  detail: (name: string, workspace: string) =>
    [...skillKeys.all, "detail", name, workspace] as const,
  content: (name: string, workspace: string) =>
    [...skillKeys.all, "content", name, workspace] as const,
  marketplace: () => [...skillKeys.all, "marketplace"] as const,
  marketplaceSearch: (query: string, limit?: number) =>
    [...skillKeys.marketplace(), "search", query, limit ?? null] as const,
  marketplaceInfo: (slug: string) => [...skillKeys.marketplace(), "info", slug] as const,
};
