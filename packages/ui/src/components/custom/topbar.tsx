"use client";

import { ChevronLeft, MoreHorizontal } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

export interface TopbarRouteContext {
  title: string;
  icon?: LucideIcon;
  subtitle?: string;
  getCount?: () => number | string;
  /**
   * Opt-in identifier resolved by the shell via `useNavCounts`. When set and
   * the active slot omits `count`, the shell injects the resolved value into
   * `<Topbar navCount>`.
   */
  navCountKey?: string;
}

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

const TopbarSlotContext = React.createContext<TopbarSlotContextValue | null>(null);

export interface TopbarSlotProviderProps {
  children: React.ReactNode;
}

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

function isSameSlot(a: TopbarSlotValue | null, b: TopbarSlotValue | null): boolean {
  if (a === b) return true;
  return slotKey(a) === slotKey(b);
}

function TopbarSlotProvider({ children }: TopbarSlotProviderProps) {
  const [slot, setSlotState] = React.useState<TopbarSlotValue | null>(null);
  const setSlot = React.useCallback((next: TopbarSlotValue | null) => {
    setSlotState(prev => (isSameSlot(prev, next) ? prev : next));
  }, []);
  const value = React.useMemo<TopbarSlotContextValue>(() => ({ slot, setSlot }), [slot, setSlot]);
  return <TopbarSlotContext.Provider value={value}>{children}</TopbarSlotContext.Provider>;
}

/**
 * Pushes a topbar slot for the lifetime of the calling component.
 */
function useTopbarSlot(slot: TopbarSlotValue | null): void {
  const ctx = React.useContext(TopbarSlotContext);
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

function useTopbarSlotValue(): TopbarSlotValue | null {
  const ctx = React.useContext(TopbarSlotContext);
  return ctx?.slot ?? null;
}

function useTopbarSlotContext(): TopbarSlotContextValue | null {
  return React.useContext(TopbarSlotContext);
}

export interface TopbarProps extends Omit<React.ComponentProps<"header">, "title"> {
  route: TopbarRouteContext | null;
  /**
   * Count resolved by the shell from `useNavCounts()` when the route declares
   * a `navCountKey`. Falls back through `slot.count` -> `route.getCount()` ->
   * `navCount` / §8.
   */
  navCount?: number | string;
  /** Optional ref for the topbar title element so the shell can move focus on route resolve. */
  titleRef?: React.Ref<HTMLHeadingElement>;
}

function hasCountValue(value: number | string | undefined | null): value is number | string {
  if (value === undefined || value === null) return false;
  if (typeof value === "number") return true;
  return value !== "";
}

function Topbar({ route, navCount, className, titleRef, ...props }: TopbarProps) {
  const slot = useTopbarSlotValue();
  const Icon = route?.icon;
  const routeCount = route?.getCount?.();
  const slotCount = slot?.count;
  const resolvedCount: number | string | undefined = hasCountValue(slotCount)
    ? slotCount
    : hasCountValue(routeCount)
      ? routeCount
      : hasCountValue(navCount)
        ? navCount
        : undefined;
  const hasCount = hasCountValue(resolvedCount);
  const renderedTitle: React.ReactNode = slot?.title ?? route?.title ?? "Untitled";
  const back = slot?.back;
  const backLabel = slot?.backLabel ?? "Go back";

  return (
    <header
      data-slot="topbar"
      data-mode={back ? "detail" : "default"}
      className={cn(
        "flex h-12 min-w-0 shrink-0 items-center gap-3 border-b border-(--line) bg-(--canvas) px-4",
        className
      )}
      {...props}
    >
      {back ? (
        <button
          aria-label={backLabel}
          className="inline-flex size-5 shrink-0 items-center justify-center rounded-(--radius-sm) text-(--muted) transition-colors hover:bg-(--hover) hover:text-(--fg) focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-(--line-strong)"
          data-slot="topbar-back"
          data-testid="topbar-back"
          onClick={back}
          type="button"
        >
          <ChevronLeft aria-hidden="true" className="size-3.5" />
        </button>
      ) : null}
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
          className="truncate text-[14px] font-medium tracking-tight text-(--fg-strong) outline-none focus-visible:ring-1 focus-visible:ring-(--line-strong)"
        >
          {renderedTitle}
        </h1>
        {hasCount ? (
          <span
            data-slot="topbar-count"
            data-testid="topbar-count"
            className="inline-flex h-[19px] min-w-[19px] items-center justify-center rounded-(--radius-mono-badge) bg-(--canvas-soft) px-1.5 font-mono text-[10.5px] font-medium tabular-nums text-(--muted)"
          >
            {resolvedCount}
          </span>
        ) : null}
        {slot?.meta ? (
          <div
            data-slot="topbar-meta"
            data-testid="topbar-meta"
            className="flex min-w-0 items-center gap-2 text-(--muted)"
          >
            {slot.meta}
          </div>
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
        {slot?.overflow ? (
          <div
            data-slot="topbar-overflow"
            data-testid="topbar-overflow"
            className="inline-flex shrink-0 items-center"
          >
            {slot.overflow}
          </div>
        ) : null}
      </div>
    </header>
  );
}

const TopbarOverflowIcon = MoreHorizontal;

export {
  Topbar,
  TopbarOverflowIcon,
  TopbarSlotContext,
  TopbarSlotProvider,
  useTopbarSlot,
  useTopbarSlotContext,
  useTopbarSlotValue,
};
