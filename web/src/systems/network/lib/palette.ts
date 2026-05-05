export const NETWORK_IDENTITY_PALETTE: ReadonlyArray<readonly [string, string]> = [
  ["var(--color-accent-tint)", "var(--color-accent)"],
  ["var(--color-info-tint)", "var(--color-info)"],
  ["var(--color-success-tint)", "var(--color-success)"],
  ["var(--color-warning-tint)", "var(--color-warning)"],
  ["var(--color-danger-tint)", "var(--color-danger)"],
  ["var(--color-neutral-tint)", "var(--color-text-label)"],
  ["var(--color-surface-elevated)", "var(--color-text-primary)"],
  ["var(--color-surface-panel)", "var(--color-text-secondary)"],
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
