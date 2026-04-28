/**
 * Wire-protocol kind → leading-dot color map.
 * Each protocol kind (`say`, `greet`, `direct`, …) is identified visually by
 * a 7px colored dot rendered ahead of the kind label. Unknown kinds (platform
 * names, event ids) render without a dot.
 */
export const KIND_COLORS: Record<string, string> = {
  say: "var(--color-kind-say)",
  greet: "var(--color-kind-greet)",
  direct: "var(--color-kind-direct)",
  receipt: "var(--color-kind-receipt)",
  capability: "var(--color-kind-capability)",
  trace: "var(--color-kind-trace)",
  whois: "var(--color-kind-whois)",
};

export function kindColorFor(kind: string): string | undefined {
  return KIND_COLORS[kind.toLowerCase()];
}
