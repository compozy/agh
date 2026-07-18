import * as React from "react";

export interface TopbarSlotValue {
  /**
   * Override for the leaf breadcrumb label. Lets routes that resolve their
   * identity from loader data (entity display names) push it as a live node.
   */
  crumb?: React.ReactNode;
  /**
   * Centered sister-route navigation. Real links with `aria-current="page"`
   * only — panel Tabs and mode PillGroups stay in body chrome.
   */
  routeNav?: React.ReactNode;
  /** Action buttons rendered in the trailing zone (sm, one accent CTA). */
  actions?: React.ReactNode;
  /** Overflow menu rendered last in the trailing zone. */
  overflow?: React.ReactNode;
}

interface TopbarSlotPublisher {
  publishSlot: (owner: object, slot: TopbarSlotValue | null) => void;
  clearSlot: (owner: object) => void;
}

interface TopbarSlotStore extends TopbarSlotPublisher {
  getSnapshot: () => TopbarSlotValue | null;
  subscribe: (listener: () => void) => () => void;
}

export const TopbarSlotSettersContext = React.createContext<TopbarSlotPublisher | null>(null);

export const TopbarSlotContext = React.createContext<TopbarSlotStore | null>(null);

/**
 * Reference equality per zone. ReactNode contents are compared by identity —
 * a publisher rendering fresh JSX republishes, which is required so replaced
 * action handlers reach the topbar.
 */
function isSameTopbarSlot(a: TopbarSlotValue | null, b: TopbarSlotValue | null): boolean {
  if (a === b) return true;
  if (a == null || b == null) return false;
  return (
    a.crumb === b.crumb &&
    a.routeNav === b.routeNav &&
    a.actions === b.actions &&
    a.overflow === b.overflow
  );
}

export function createTopbarSlotStore(): TopbarSlotStore {
  let active: { owner: object; slot: TopbarSlotValue | null } | null = null;
  const listeners = new Set<() => void>();
  const emit = () => {
    for (const listener of listeners) listener();
  };

  return {
    getSnapshot: () => active?.slot ?? null,
    subscribe: listener => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    publishSlot: (owner, slot) => {
      if (active?.owner === owner && isSameTopbarSlot(active.slot, slot)) return;
      active = { owner, slot };
      emit();
    },
    clearSlot: owner => {
      if (active?.owner !== owner) return;
      active = null;
      emit();
    },
  };
}

const subscribeToNothing = () => () => undefined;
const getNullSnapshot = () => null;

/**
 * Pushes a topbar slot for the lifetime of the calling component.
 */
export function useTopbarSlot(slot: TopbarSlotValue | null): void {
  const setters = React.use(TopbarSlotSettersContext);
  const [owner] = React.useState<object>(() => ({}));
  React.useLayoutEffect(() => {
    if (!setters) return;
    if (slot === null) {
      setters.clearSlot(owner);
      return;
    }
    setters.publishSlot(owner, slot);
  }, [owner, setters, slot]);
  React.useEffect(() => {
    if (!setters) return;
    return () => setters.clearSlot(owner);
  }, [owner, setters]);
}

export function useTopbarSlotValue(): TopbarSlotValue | null {
  const store = React.use(TopbarSlotContext);
  return React.useSyncExternalStore(
    store?.subscribe ?? subscribeToNothing,
    store?.getSnapshot ?? getNullSnapshot,
    store?.getSnapshot ?? getNullSnapshot
  );
}
