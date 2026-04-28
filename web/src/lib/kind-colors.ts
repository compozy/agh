/**
 * Wire-protocol kind → leading-dot color map.
 * Each protocol kind (`say`, `greet`, `direct`, …) is identified visually by
 * a 7px colored dot rendered ahead of the kind label. Unknown kinds (platform
 * names, event ids) render without a dot.
 */
export const KIND_COLORS: Record<string, string> = {
  say: "#8E8E93",
  greet: "#5BA6FF",
  direct: "var(--color-accent)",
  receipt: "var(--color-success)",
  recipe: "var(--color-warning)",
  trace: "#B892FF",
  whois: "#4FD1C5",
};

export function kindColorFor(kind: string): string | undefined {
  return KIND_COLORS[kind.toLowerCase()];
}
