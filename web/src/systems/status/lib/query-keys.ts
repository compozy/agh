export const daemonKeys = {
  all: ["daemon"] as const,
  health: () => [...daemonKeys.all, "health"] as const,
  status: () => [...daemonKeys.all, "status"] as const,
};
