"use client";

import type { LucideIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

export interface TopbarRouteContext {
  title: string;
  icon?: LucideIcon;
  subtitle?: string;
  getCount?: () => number | string;
}

export interface TopbarSlotValue {
  /**
   * Optional override for the route context's static title. Lets routes that
   * resolve their title from loader data push it as a live React node.
   */
  title?: React.ReactNode;
  /** Optional override for the route context's count. */
  count?: React.ReactNode;
  tabs?: React.ReactNode;
  search?: React.ReactNode;
  actions?: React.ReactNode;
}

export interface TopbarSlotContextValue {
  slot: TopbarSlotValue | null;
  setSlot: (slot: TopbarSlotValue | null) => void;
}

const TopbarSlotContext = React.createContext<TopbarSlotContextValue | null>(null);

export interface TopbarSlotProviderProps {
  children: React.ReactNode;
}

function TopbarSlotProvider({ children }: TopbarSlotProviderProps) {
  const [slot, setSlot] = React.useState<TopbarSlotValue | null>(null);
  const value = React.useMemo<TopbarSlotContextValue>(() => ({ slot, setSlot }), [slot]);
  return <TopbarSlotContext.Provider value={value}>{children}</TopbarSlotContext.Provider>;
}

/**
 * Pushes a topbar slot for the lifetime of the calling component.
 *
 * The slot is re-pushed whenever the value changes so JSX nodes carrying live
 * data (counts, enabled state, mutations) stay current without manual setSlot
 * calls. Cleanup runs `setSlot(null)` on unmount so the slot does not leak
 * after the consumer disappears. Resilient to missing provider — when no
 * `<TopbarSlotProvider>` is in the tree (test fast paths, isolated stories)
 * the hook becomes a no-op instead of throwing.
 *
 * Note on nesting: when both a parent layout and a child route call this
 * hook, React commits child effects before parent effects. Routes that need
 * deepest-match-wins semantics should keep the parent slot stable (or skip
 * pushing one) and let only the deepest route call `useTopbarSlot`.
 */
function useTopbarSlot(slot: TopbarSlotValue | null): void {
  const ctx = React.use(TopbarSlotContext);
  const setSlot = ctx?.setSlot;
  React.useEffect(() => {
    if (!setSlot) {
      return;
    }
    setSlot(slot);
  }, [setSlot, slot]);
  React.useEffect(() => {
    if (!setSlot) {
      return;
    }
    return () => setSlot(null);
  }, [setSlot]);
}

function useTopbarSlotValue(): TopbarSlotValue | null {
  const ctx = React.use(TopbarSlotContext);
  return ctx?.slot ?? null;
}

function useTopbarSlotContext(): TopbarSlotContextValue | null {
  return React.use(TopbarSlotContext);
}

export interface TopbarProps extends Omit<React.ComponentProps<"header">, "title"> {
  route: TopbarRouteContext | null;
  /** Optional ref for the topbar title element so the shell can move focus on route resolve. */
  titleRef?: React.Ref<HTMLHeadingElement>;
}

function Topbar({ route, className, titleRef, ...props }: TopbarProps) {
  const slot = useTopbarSlotValue();
  const Icon = route?.icon;
  const routeCount = route?.getCount?.();
  const hasRouteCount = routeCount !== undefined && routeCount !== null && routeCount !== "";
  const slotCount = slot?.count;
  const hasSlotCount = slotCount !== undefined && slotCount !== null && slotCount !== "";
  const hasCount = hasSlotCount || hasRouteCount;
  const renderedTitle: React.ReactNode = slot?.title ?? route?.title ?? "Untitled";

  return (
    <header
      data-slot="topbar"
      className={cn(
        "flex h-12 min-w-0 shrink-0 items-center gap-3 border-b border-(--line) bg-(--canvas) px-4",
        className
      )}
      {...props}
    >
      <div data-slot="topbar-title" className="flex min-w-0 items-center gap-2">
        {Icon ? (
          <span
            aria-hidden="true"
            data-slot="topbar-icon"
            className="inline-flex size-6 shrink-0 items-center justify-center rounded-(--radius-sm) bg-(--elevated) text-(--accent)"
          >
            <Icon className="size-3.5" />
          </span>
        ) : null}
        <h1
          ref={titleRef}
          tabIndex={-1}
          data-testid="topbar-title-text"
          className="truncate text-[14px] font-medium tracking-[-0.014em] text-(--fg-strong) outline-none focus-visible:ring-1 focus-visible:ring-(--line-strong)"
        >
          {renderedTitle}
        </h1>
        {hasCount ? (
          <span
            data-slot="topbar-count"
            className="inline-flex h-[19px] min-w-[19px] items-center justify-center rounded-(--radius-mono-badge) bg-(--canvas-soft) px-1.5 font-mono text-[10.5px] font-medium tabular-nums text-(--muted)"
          >
            {hasSlotCount ? slotCount : routeCount}
          </span>
        ) : null}
      </div>
      {slot?.tabs ? (
        <div data-slot="topbar-tabs" className="flex min-w-0 items-center">
          {slot.tabs}
        </div>
      ) : null}
      <div data-slot="topbar-trailing" className="ml-auto flex shrink-0 items-center gap-2">
        {slot?.search ? <div data-slot="topbar-search">{slot.search}</div> : null}
        {slot?.actions ? <div data-slot="topbar-actions">{slot.actions}</div> : null}
      </div>
    </header>
  );
}

export {
  Topbar,
  TopbarSlotContext,
  TopbarSlotProvider,
  useTopbarSlot,
  useTopbarSlotContext,
  useTopbarSlotValue,
};
