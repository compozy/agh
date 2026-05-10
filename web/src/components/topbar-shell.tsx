import { useEffect, useRef } from "react";
import { useMatches, useRouter } from "@tanstack/react-router";

import {
  Topbar,
  TopbarSlotContext,
  TopbarSlotProvider,
  type TopbarRouteContext,
} from "@agh/ui";
import * as React from "react";

interface MaybeTopbarMatchContext {
  topbar?: TopbarRouteContext;
}

function pickDeepestTopbarContext(
  matches: ReadonlyArray<unknown> | undefined
): TopbarRouteContext | null {
  if (!matches) return null;
  for (let index = matches.length - 1; index >= 0; index -= 1) {
    const candidate = (matches[index] as { context?: MaybeTopbarMatchContext } | undefined)
      ?.context?.topbar;
    if (candidate && candidate.title) {
      return candidate;
    }
  }
  return null;
}

interface TopbarShellProps {
  children: React.ReactNode;
}

/**
 * Mounts the shell-level `<Topbar>` once for the entire `_app` outlet.
 *
 * Behavior:
 * - Reads `useRouterState({ select: matches })` to find the deepest match
 *   whose `context.topbar` declares a `title`.
 * - Hosts `<TopbarSlotProvider>` so any descendant route can call
 *   `useTopbarSlot` to push tabs/search/actions.
 * - Subscribes to `router.subscribe("onResolved")` to clear the slot on every
 *   navigation and to move focus to the topbar `h1` for the screen-reader and
 *   keyboard handoff after route resolution.
 */
export function TopbarShell({ children }: TopbarShellProps) {
  return (
    <TopbarSlotProvider>
      <TopbarShellInner>{children}</TopbarShellInner>
    </TopbarSlotProvider>
  );
}

function TopbarShellInner({ children }: TopbarShellProps) {
  const titleRef = useRef<HTMLHeadingElement | null>(null);
  const slotContext = React.use(TopbarSlotContext);
  const setSlot = slotContext?.setSlot;
  const router = useRouter();
  // useMatches exposes the active match chain ordered root → leaf so we can
  // scan from the deepest match upward for the topbar context augmentation.
  const matches = useMatches() as unknown as ReadonlyArray<unknown>;
  const route = pickDeepestTopbarContext(matches);

  useEffect(() => {
    const unsubscribe = router.subscribe("onResolved", () => {
      setSlot?.(null);
      const node = titleRef.current;
      if (node) {
        try {
          node.focus({ preventScroll: true });
        } catch {
          node.focus();
        }
      }
    });
    return unsubscribe;
  }, [router, setSlot]);

  return (
    <>
      <Topbar route={route} titleRef={titleRef} />
      {children}
    </>
  );
}
