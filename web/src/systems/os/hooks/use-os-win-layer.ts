import { useEffect, useRef, type RefObject } from "react";
import { useShallow } from "zustand/shallow";

import type { OsPresentation } from "../lib/os-types";
import { useDesktop } from "./use-desktop";
import { useOsShell } from "./use-os-shell";

export interface OsWinLayerModel {
  layerRef: RefObject<HTMLDivElement | null>;
  windowIds: string[];
  presentation: OsPresentation;
  anyVisible: boolean;
}

/**
 * Win-layer state + viewport re-clamping: shrinking the viewport pulls
 * windows back into reach; growing it never moves them (US-002.EC-1/EC-2).
 */
export function useOsWinLayer(): OsWinLayerModel {
  const { store } = useOsShell();
  const layerRef = useRef<HTMLDivElement | null>(null);
  const windowIds = useDesktop(useShallow(state => Object.keys(state.windows)));
  const presentation = useDesktop(state => state.presentation);
  const anyVisible = useDesktop(state => Object.values(state.windows).some(win => !win.minimized));

  useEffect(() => {
    const layer = layerRef.current;
    if (!layer) return;
    const observer = new ResizeObserver(entries => {
      const entry = entries[0];
      if (!entry) return;
      store.getState().clampToViewport({
        width: entry.contentRect.width,
        height: entry.contentRect.height,
      });
    });
    observer.observe(layer);
    return () => observer.disconnect();
  }, [store]);

  return { layerRef, windowIds, presentation, anyVisible };
}
