import * as React from "react";

export interface TopbarSlotValue {
  /**
   * Optional override for the route context's static title. Lets routes that
   * resolve their title from loader data push it as a live React node.
   */
  title?: React.ReactNode;
  /**
   * Numeric / textual count rendered as the topbar chip. Narrowed from
   * `ReactNode` — the chip is data, not a render slot. Auto-resolves from
   * `useNavCounts()` when omitted and the route declares a `navCountKey`.
   */
  count?: number | string;
  /** Lane / mode tabs rendered between title and trailing slots. */
  tabs?: React.ReactNode;
  /** Search affordance rendered in the trailing slot. */
  search?: React.ReactNode;
  /** Action buttons rendered in the trailing slot. */
  actions?: React.ReactNode;
  /**
   * Detail-mode back affordance. When present, renders a leading 20x20 ghost
   * chevron button.
   */
  back?: () => void;
  /** Optional aria-label override for the back button (default "Go back"). */
  backLabel?: string;
  /** Detail-mode meta chips rendered after the title and count. */
  meta?: React.ReactNode;
  /** Detail-mode overflow menu rendered at the trailing edge. */
  overflow?: React.ReactNode;
}

export interface TopbarSlotContextValue {
  slot: TopbarSlotValue | null;
  setSlot: (slot: TopbarSlotValue | null) => void;
}

export const TopbarSlotContext = React.createContext<TopbarSlotContextValue | null>(null);

function slotKey(slot: TopbarSlotValue | null): string {
  if (slot === null) return "null";
  try {
    return JSON.stringify(slot, (key, value) => {
      if (typeof value === "function") return undefined;
      if (key === "ref" || key === "_owner" || key === "_store") return undefined;
      return value;
    });
  } catch {
    return String(Math.random());
  }
}

export function isSameTopbarSlot(a: TopbarSlotValue | null, b: TopbarSlotValue | null): boolean {
  if (a === b) return true;
  return slotKey(a) === slotKey(b);
}

/**
 * Pushes a topbar slot for the lifetime of the calling component.
 */
export function useTopbarSlot(slot: TopbarSlotValue | null): void {
  const ctx = React.use(TopbarSlotContext);
  const setSlot = ctx?.setSlot;
  React.useEffect(() => {
    if (!setSlot) return;
    setSlot(slot);
  }, [setSlot, slot]);
  React.useEffect(() => {
    if (!setSlot) return;
    return () => setSlot(null);
  }, [setSlot]);
}

export function useTopbarSlotValue(): TopbarSlotValue | null {
  const ctx = React.use(TopbarSlotContext);
  return ctx?.slot ?? null;
}

export function useTopbarSlotContext(): TopbarSlotContextValue | null {
  return React.use(TopbarSlotContext);
}
