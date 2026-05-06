export const agentKeys = {
  all: ["agents"] as const,
  list: (workspace?: string | null) => [...agentKeys.all, "list", workspace ?? null] as const,
  detail: (name: string, workspace?: string | null) =>
    [...agentKeys.all, "detail", name, workspace ?? null] as const,
};
