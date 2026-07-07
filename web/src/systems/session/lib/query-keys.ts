export const sessionKeys = {
  all: ["sessions"] as const,
  lists: () => [...sessionKeys.all, "list"] as const,
  list: (workspace: string | null = null) => [...sessionKeys.lists(), workspace ?? "all"] as const,
  workspace: (workspace: string) => [...sessionKeys.all, "workspace", workspace] as const,
  detail: (workspace: string, id: string) =>
    [...sessionKeys.workspace(workspace), "detail", id] as const,
  events: (workspace: string, id: string) =>
    [...sessionKeys.detail(workspace, id), "events"] as const,
  history: (workspace: string, id: string) =>
    [...sessionKeys.detail(workspace, id), "history"] as const,
  transcript: (workspace: string, id: string) =>
    [...sessionKeys.detail(workspace, id), "transcript"] as const,
  recap: (workspace: string, id: string, limit?: number) =>
    [...sessionKeys.detail(workspace, id), "recap", limit ?? "default"] as const,
  ledger: (workspace: string, id: string) =>
    [...sessionKeys.detail(workspace, id), "ledger"] as const,
};
