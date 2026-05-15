export const agentKeys = {
  all: ["agents"] as const,
  lists: () => [...agentKeys.all, "list"] as const,
  list: (workspace?: string | null) => [...agentKeys.lists(), workspace ?? null] as const,
  detail: (name: string, workspace?: string | null) =>
    [...agentKeys.all, "detail", name, workspace ?? null] as const,
};
