/**
 * Deterministic UI-side owner palette per PRD G3.
 *
 * Hash is FNV-1a over the label. Collisions are intentional because the palette
 * is a glance-aid, not an identifier; owner labels remain authoritative. The
 * 7-color human + 4-color agent split keeps cross-kind collisions rare even in
 * mixed workspaces. No daemon round-trip and no collision-avoidance fallback.
 */
export type OwnerKind = "agent" | "human" | "system";

export const AGENT_PALETTE: ReadonlyArray<string> = [
  "#7a8aa3",
  "#9b8a72",
  "#7c9a8c",
  "#a18a8a",
] as const;

export const HUMAN_PALETTE: ReadonlyArray<string> = ["#c79f7a", "#b8826a", "#a47265"] as const;

export const SYSTEM_PALETTE: ReadonlyArray<string> = ["#9a9a9f"] as const;

const FNV_OFFSET_BASIS = 0x811c9dc5;
const FNV_PRIME = 0x01000193;

function fnv1aHash(value: string): number {
  let hash = FNV_OFFSET_BASIS;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, FNV_PRIME);
  }
  return hash >>> 0;
}

function paletteFor(kind: OwnerKind): ReadonlyArray<string> {
  switch (kind) {
    case "agent":
      return AGENT_PALETTE;
    case "human":
      return HUMAN_PALETTE;
    case "system":
      return SYSTEM_PALETTE;
  }
}

export function ownerColor(label: string, kind: OwnerKind): string {
  const palette = paletteFor(kind);
  const hash = fnv1aHash(label);
  return palette[hash % palette.length];
}
