export const sessionKeys = {
  all: ["sessions"] as const,
  lists: () => [...sessionKeys.all, "list"] as const,
  list: (workspace: string | null = null) => [...sessionKeys.lists(), workspace ?? "all"] as const,
  detail: (id: string) => [...sessionKeys.all, "detail", id] as const,
  events: (id: string) => [...sessionKeys.all, "detail", id, "events"] as const,
  history: (id: string) => [...sessionKeys.all, "detail", id, "history"] as const,
  transcript: (id: string) => [...sessionKeys.all, "detail", id, "transcript"] as const,
  ledger: (id: string) => [...sessionKeys.all, "detail", id, "ledger"] as const,
};
