export const daemonKeys = {
  all: ["daemon"] as const,
  health: () => [...daemonKeys.all, "health"] as const,
};
