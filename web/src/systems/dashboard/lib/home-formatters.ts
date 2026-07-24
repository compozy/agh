/** Compact token counts for KPI and chart figures: 1.4M, 320K, 950. */
export function formatHomeTokens(total: number): string {
  if (total >= 1_000_000) {
    return `${(total / 1_000_000).toFixed(1)}M`;
  }
  if (total >= 1_000) {
    return `${Math.round(total / 1_000)}K`;
  }
  return String(total);
}
