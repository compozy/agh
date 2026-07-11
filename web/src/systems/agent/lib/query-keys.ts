export interface AgentHeartbeatStatusKeyOptions {
  workspaceId?: string | null;
  sessionId?: string | null;
  includeSessionHealth?: boolean;
  includeRecentWakeEvents?: boolean;
}

export const agentKeys = {
  all: ["agents"] as const,
  lists: () => [...agentKeys.all, "list"] as const,
  list: (workspace?: string | null) => [...agentKeys.lists(), workspace ?? null] as const,
  detail: (name: string, workspace?: string | null) =>
    [...agentKeys.all, "detail", name, workspace ?? null] as const,
  soul: (name: string, workspace?: string | null) =>
    [...agentKeys.detail(name, workspace), "soul"] as const,
  soulHistory: (name: string, workspace?: string | null) =>
    [...agentKeys.soul(name, workspace), "history"] as const,
  heartbeat: (name: string, workspace?: string | null) =>
    [...agentKeys.detail(name, workspace), "heartbeat"] as const,
  heartbeatHistory: (name: string, workspace?: string | null) =>
    [...agentKeys.heartbeat(name, workspace), "history"] as const,
  heartbeatStatus: (name: string, options: AgentHeartbeatStatusKeyOptions = {}) =>
    [
      ...agentKeys.heartbeat(name, options.workspaceId),
      "status",
      options.sessionId ?? null,
      options.includeSessionHealth ?? null,
      options.includeRecentWakeEvents ?? null,
    ] as const,
};
