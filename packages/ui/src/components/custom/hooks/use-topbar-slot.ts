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

export interface TopbarSlotSetters {
  setSlot: (owner: object, slot: TopbarSlotValue | null) => void;
  clearSlot: (owner: object) => void;
}

export interface TopbarSlotContextValue extends TopbarSlotSetters {
  slot: TopbarSlotValue | null;
}

interface TopbarSlotStore extends TopbarSlotSetters {
  getSnapshot: () => TopbarSlotValue | null;
  subscribe: (listener: () => void) => () => void;
}

export const TopbarSlotSettersContext = React.createContext<TopbarSlotSetters | null>(null);

export const TopbarSlotContext = React.createContext<TopbarSlotStore | null>(null);

function slotKey(slot: TopbarSlotValue | null): string {
  if (slot === null) return "null";
  const seen = new WeakSet<object>();
  try {
    return JSON.stringify(slot, (key, value) => {
      if (typeof value === "function") return undefined;
      if (key === "ref" || key.startsWith("_")) return undefined;
      if (typeof value === "bigint") return String(value);
      if (typeof value === "object" && value !== null) {
        if (seen.has(value)) return undefined;
        seen.add(value);
      }
      return value;
    });
  } catch {
    return JSON.stringify({
      title: typeof slot.title === "string" ? slot.title : Boolean(slot.title),
      count: slot.count,
      tabs: Boolean(slot.tabs),
      search: Boolean(slot.search),
      actions: Boolean(slot.actions),
      back: Boolean(slot.back),
      backLabel: slot.backLabel,
      meta: Boolean(slot.meta),
      overflow: Boolean(slot.overflow),
    });
  }
}

function isSameTopbarBehavior(a: TopbarSlotValue, b: TopbarSlotValue): boolean {
  return (
    a.back === b.back &&
    a.title === b.title &&
    a.tabs === b.tabs &&
    a.search === b.search &&
    a.actions === b.actions &&
    a.meta === b.meta &&
    a.overflow === b.overflow
  );
}

export function isSameTopbarSlot(a: TopbarSlotValue | null, b: TopbarSlotValue | null): boolean {
  if (a === b) return true;
  if (a == null || b == null) return false;
  if (!isSameTopbarBehavior(a, b)) return false;
  return slotKey(a) === slotKey(b);
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
    setSlot: (owner, slot) => {
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
  const setSlot = setters?.setSlot;
  const clearSlot = setters?.clearSlot;
  const ownerRef = React.useRef<object>({});
  const slotRef = React.useRef(slot);
  slotRef.current = slot;
  const signature = slotKey(slot);
  React.useEffect(() => {
    if (!setSlot || !clearSlot) return;
    if (slotRef.current === null) {
      clearSlot(ownerRef.current);
      return;
    }
    setSlot(ownerRef.current, slotRef.current);
  }, [
    setSlot,
    clearSlot,
    signature,
    slot?.back,
    slot?.title,
    slot?.tabs,
    slot?.search,
    slot?.actions,
    slot?.meta,
    slot?.overflow,
  ]);
  React.useEffect(() => {
    if (!clearSlot) return;
    return () => clearSlot(ownerRef.current);
  }, [clearSlot]);
}

export function useTopbarSlotValue(): TopbarSlotValue | null {
  const store = React.use(TopbarSlotContext);
  return React.useSyncExternalStore(
    store?.subscribe ?? subscribeToNothing,
    store?.getSnapshot ?? getNullSnapshot,
    store?.getSnapshot ?? getNullSnapshot
  );
}

export function useTopbarSlotContext(): TopbarSlotContextValue | null {
  const store = React.use(TopbarSlotContext);
  const slot = useTopbarSlotValue();
  return store ? { slot, setSlot: store.setSlot, clearSlot: store.clearSlot } : null;
}
