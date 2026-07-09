import type { ListingViewMode } from "@agh/ui";

/**
 * Shared URL search parsers for listing routes. Extracted so /loops, /skills,
 * and /bridges resolve `q`/`view` identically — the per-page drift the listing
 * redesign fights (LISTING-REDESIGN-BRIEF §5: one URL contract across pages).
 */

/** Trim a raw search-param value; blank/whitespace/non-string → undefined. */
export function normalizeListingSearchValue(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

/** Parse the `view` search param; anything other than the two modes → undefined. */
export function parseListingView(value: unknown): ListingViewMode | undefined {
  return value === "rows" || value === "cards" ? value : undefined;
}
