import { useMatches } from "@tanstack/react-router";

import type { TopbarCrumbContext, TopbarRouteContext } from "@/types/topbar";

interface MaybeTopbarMatchContext {
  topbar?: TopbarRouteContext;
}

/**
 * Collects the breadcrumb trail from the active match chain (root → leaf).
 * Matches without their own `topbar` context inherit the parent's merged
 * object, so consecutive duplicates are folded by reference identity.
 *
 * Route `beforeLoad` must return a stable module-level `{ topbar }` object
 * when the crumb does not depend on params — a fresh object per invocation
 * makes every nested match contribute a duplicate crumb.
 */
export function collectCrumbs(matches: ReadonlyArray<unknown>): ReadonlyArray<TopbarCrumbContext> {
  const crumbs: TopbarCrumbContext[] = [];
  let previous: TopbarRouteContext | undefined;
  for (const match of matches) {
    const context = (match as { context?: MaybeTopbarMatchContext } | undefined)?.context?.topbar;
    if (!context || context === previous) continue;
    if (context.parentCrumb) crumbs.push(context.parentCrumb);
    crumbs.push(context.crumb);
    previous = context;
  }
  return crumbs;
}

export interface TopbarShellViewModel {
  crumbs: ReadonlyArray<TopbarCrumbContext>;
}

/**
 * Resolves the breadcrumb trail for `<TopbarShellInner>` from every route
 * level's `topbar.crumb` declaration.
 */
export function useTopbarShellModel(): TopbarShellViewModel {
  // useMatches exposes the active match chain ordered root → leaf so the trail
  // reads parent › child › page without extra sorting.
  const matches = useMatches() as unknown as ReadonlyArray<unknown>;
  return { crumbs: collectCrumbs(matches) };
}
