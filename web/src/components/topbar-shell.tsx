import { useEffect, useRef } from "react";
import { useRouter } from "@tanstack/react-router";

import { Topbar, TopbarSlotContext, TopbarSlotProvider } from "@agh/ui";
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
 * - Subscribes to `router.subscribe("onResolved")` to clear the slot on
 *   path-changing navigation and to move focus to the topbar `h1` for the
 *   screen-reader and keyboard handoff after route resolution.
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
  const { route, navCount } = useTopbarShellModel();

  useEffect(() => {
    const unsubscribe = router.subscribe("onResolved", event => {
      if (!event.pathChanged) {
        return;
      }

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
      <Topbar navCount={navCount} route={route} titleRef={titleRef} />
      {children}
    </>
  );
}
