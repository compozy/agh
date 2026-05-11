import { useMatches } from "@tanstack/react-router";

import type { TopbarRouteContext } from "@agh/ui";

import { type NavCountKey, useNavCounts } from "@/systems/runtime";

interface MaybeTopbarMatchContext {
  topbar?: TopbarRouteContext;
}

function pickDeepestTopbarContext(
  matches: ReadonlyArray<unknown> | undefined
): TopbarRouteContext | null {
  if (!matches) return null;
  for (let index = matches.length - 1; index >= 0; index -= 1) {
    const candidate = (matches[index] as { context?: MaybeTopbarMatchContext } | undefined)?.context
      ?.topbar;
    if (candidate && candidate.title) {
      return candidate;
    }
  }
  return null;
}

function resolveNavCount(
  result: ReturnType<typeof useNavCounts>,
  key: string | undefined
): number | string | undefined {
  if (!key) return undefined;
  const entry = result.counts[key as NavCountKey];
  if (!entry) return undefined;
  if (entry.stale && entry.count === 0) return undefined;
  return entry.count;
}

export interface TopbarShellViewModel {
  route: TopbarRouteContext | null;
  navCount: number | string | undefined;
}

/**
 * Resolves the deepest topbar route context and threads the nav count from
 * `useNavCounts()` so `<TopbarShellInner>` stays within the project's
 * max-component-complexity ceiling (5 hook calls).
 */
export function useTopbarShellModel(): TopbarShellViewModel {
  // useMatches exposes the active match chain ordered root → leaf so we can
  // scan from the deepest match upward for the topbar context augmentation.
  const matches = useMatches() as unknown as ReadonlyArray<unknown>;
  const route = pickDeepestTopbarContext(matches);
  const navCounts = useNavCounts();
  const navCount = resolveNavCount(navCounts, route?.navCountKey);
  return { route, navCount };
}
