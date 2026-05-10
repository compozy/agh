export const NETWORK_IDENTITY_PALETTE: ReadonlyArray<readonly [string, string]> = [
  ["var(--accent-tint)", "var(--accent)"],
  ["var(--info-tint)", "var(--info)"],
  ["var(--success-tint)", "var(--success)"],
  ["var(--warning-tint)", "var(--warning)"],
  ["var(--danger-tint)", "var(--danger)"],
  ["var(--neutral-tint)", "var(--muted)"],
  ["var(--elevated)", "var(--fg)"],
  ["var(--canvas-soft)", "var(--muted)"],
];

export function pickIdentityPaletteIndex(seed: string): number {
  let hash = 0;
  for (let index = 0; index < seed.length; index += 1) {
    hash = (hash * 31 + seed.charCodeAt(index)) | 0;
  }
  return Math.abs(hash) % NETWORK_IDENTITY_PALETTE.length;
}

export function pickIdentityPaletteColors(seed: string): readonly [string, string] {
  const palette = NETWORK_IDENTITY_PALETTE[pickIdentityPaletteIndex(seed)];
  if (!palette) {
    throw new Error("network identity palette is empty");
  }
  return palette;
}

export function getIdentityInitial(value: string | null | undefined): string {
  const trimmed = value?.trim() ?? "";
  if (trimmed.length === 0) {
    return "?";
  }
  return trimmed.charAt(0).toUpperCase();
}
