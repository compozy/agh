import { createContext, useContext } from "react";

// Shared per-timeline row state fed through context so the row renderer stays
// referentially stable (no closure deps) and each memoized `TimelineRowContent`
// skips re-render when its structurally-shared row ref is unchanged. Expansion
// state + toggle callbacks propagate through the memo boundary instead of riding
// fresh render closures.
export interface TimelineRowSharedState {
  expandedTurns: ReadonlySet<string>;
  toggleWorkGroup: (groupId: string, button: HTMLElement | null) => void;
  toggleTurn: (turnId: string, button: HTMLElement | null) => void;
  toggleChangedFiles: (rowId: string, button: HTMLElement | null) => void;
}

export const TimelineRowContext = createContext<TimelineRowSharedState | null>(null);

export function useTimelineRowContext(): TimelineRowSharedState {
  const context = useContext(TimelineRowContext);
  if (!context) {
    throw new Error("Timeline row views must render within a TimelineRowContext provider");
  }
  return context;
}
