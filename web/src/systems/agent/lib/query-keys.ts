export const agentKeys = {
  all: ["agents"] as const,
  list: () => [...agentKeys.all, "list"] as const,
  detail: (name: string) => [...agentKeys.all, "detail", name] as const,
};
