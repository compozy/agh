"use client";

import type { LucideIcon } from "lucide-react";
import { ChevronLeft, MoreHorizontal } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import {
  isSameTopbarSlot,
  TopbarSlotContext,
  type TopbarSlotContextValue,
  type TopbarSlotValue,
  useTopbarSlot,
  useTopbarSlotContext,
  useTopbarSlotValue,
} from "./hooks/use-topbar-slot";

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

export interface TopbarSlotProviderProps {
  children: React.ReactNode;
}

function TopbarSlotProvider({ children }: TopbarSlotProviderProps) {
  const [slot, setSlotState] = React.useState<TopbarSlotValue | null>(null);
  const setSlot = React.useCallback((next: TopbarSlotValue | null) => {
    setSlotState(prev => (isSameTopbarSlot(prev, next) ? prev : next));
  }, []);
  const value = React.useMemo<TopbarSlotContextValue>(() => ({ slot, setSlot }), [slot, setSlot]);
  return <TopbarSlotContext.Provider value={value}>{children}</TopbarSlotContext.Provider>;
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
        "flex h-12 min-w-0 shrink-0 items-center gap-3 border-b border-line bg-canvas px-4",
        className
      )}
      {...props}
    >
      {back ? (
        <button
          aria-label={backLabel}
          className="inline-flex size-5 shrink-0 items-center justify-center rounded-sm text-muted transition-colors hover:bg-hover hover:text-fg focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-line-strong"
          data-slot="topbar-back"
          data-testid="topbar-back"
          onClick={back}
          type="button"
        >
          <ChevronLeft aria-hidden="true" className="size-3" />
        </button>
      ) : null}
      <div data-slot="topbar-title" className="flex min-w-0 items-center gap-2">
        {Icon ? (
          <span
            aria-hidden="true"
            data-slot="topbar-icon"
            className="inline-flex size-6 shrink-0 items-center justify-center rounded-sm bg-elevated text-accent"
          >
            <Icon className="size-3" />
          </span>
        ) : null}
        <h1
          ref={titleRef}
          tabIndex={-1}
          data-testid="topbar-title-text"
          className="truncate text-card-title font-medium tracking-tight text-fg-strong outline-none focus-visible:ring-1 focus-visible:ring-line-strong"
        >
          {renderedTitle}
        </h1>
        {hasCount ? (
          <span
            data-slot="topbar-count"
            data-testid="topbar-count"
            className="inline-flex h-count-chip min-w-count-chip items-center justify-center rounded-mono-badge bg-canvas-soft px-1.5 font-mono text-mono-id font-medium tabular-nums text-muted"
          >
            {resolvedCount}
          </span>
        ) : null}
        {slot?.meta ? (
          <div
            data-slot="topbar-meta"
            data-testid="topbar-meta"
            className="flex min-w-0 items-center gap-2 text-muted"
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
export type { TopbarSlotContextValue, TopbarSlotValue };
