export const skillKeys = {
  all: ["skills"] as const,
  list: (workspace: string) => [...skillKeys.all, "list", workspace] as const,
  detail: (name: string, workspace: string) =>
    [...skillKeys.all, "detail", name, workspace] as const,
};
