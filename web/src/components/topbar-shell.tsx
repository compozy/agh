import { useEffect, useRef } from "react";
import { useRouter } from "@tanstack/react-router";

import { Topbar, TopbarSlotProvider } from "@agh/ui";
import * as React from "react";

import { useTopbarShellModel } from "@/hooks/routes/use-topbar-shell-model";

interface TopbarShellProps {
  children: React.ReactNode;
}

/**
 * Mounts the shell-level `<Topbar>` once for the entire `_app` outlet.
 *
 * Behavior:
 * - Resolves the deepest topbar route context plus the auto-resolved
 *   `useNavCounts()` value via `useTopbarShellModel`.
 * - Hosts `<TopbarSlotProvider>` so any descendant route can call
 *   `useTopbarSlot` to push tabs/search/actions.
 * - Subscribes to `router.subscribe("onResolved")` to move focus to the topbar
 *   `h1` after path navigation. Route unmount cleanup owns slot removal so a
 *   newly published destination slot cannot be cleared by a late event.
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
  const router = useRouter();
  const { route, navCount } = useTopbarShellModel();

  useEffect(() => {
    const unsubscribe = router.subscribe("onResolved", event => {
      if (!event.pathChanged) {
        return;
      }

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
  }, [router]);

  return (
    <>
      <Topbar navCount={navCount} route={route} titleRef={titleRef} />
      {children}
    </>
  );
}
