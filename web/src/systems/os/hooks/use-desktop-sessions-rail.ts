import { useEffect, useRef } from "react";
import { useShallow } from "zustand/shallow";

import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";

export function useDesktopSessionsRail() {
  const { coordinator, store } = useOsShell();
  const { open, presentation, collapsedAgentIds } = useDesktop(
    useShallow(state => ({
      open: state.railOpen,
      presentation: state.presentation,
      collapsedAgentIds: state.railCollapsedAgentIds,
    }))
  );
  const priorFocus = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (presentation !== "compact") return;
    if (open) {
      priorFocus.current =
        document.activeElement instanceof HTMLElement ? document.activeElement : null;
      return;
    }
    const target = priorFocus.current;
    priorFocus.current = null;
    target?.focus();
  }, [open, presentation]);

  return { coordinator, store, open, presentation, collapsedAgentIds };
}
