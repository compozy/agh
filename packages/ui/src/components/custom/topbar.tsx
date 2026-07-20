"use client";

import { MoreHorizontal } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import {
  createTopbarSlotStore,
  TopbarSlotContext,
  TopbarSlotSettersContext,
  type TopbarSlotValue,
  useTopbarSlot,
  useTopbarSlotValue,
} from "./hooks/use-topbar-slot";

export interface TopbarSlotProviderProps {
  children: React.ReactNode;
}

function TopbarSlotProvider({ children }: TopbarSlotProviderProps) {
  const [store] = React.useState(createTopbarSlotStore);
  return (
    <TopbarSlotSettersContext.Provider value={store}>
      <TopbarSlotContext.Provider value={store}>{children}</TopbarSlotContext.Provider>
    </TopbarSlotSettersContext.Provider>
  );
}

export interface TopbarProps extends Omit<React.ComponentProps<"header">, "title"> {
  /**
   * Optional leading zone content anchored at the start edge (e.g. OS window
   * controls). When present the no-routeNav grid becomes `1fr auto 1fr` so the
   * context zone centers; routeNav + leading use a four-column grid. Omit to
   * preserve the default DOM and classes exactly.
   */
  leading?: React.ReactNode;
  /**
   * Leading route ancestry built by the shell. The current route title is
   * rendered separately as this Topbar's single H1.
   */
  breadcrumb?: React.ReactNode;
  /** Current route identity rendered as the shell-level H1. */
  title: React.ReactNode;
  /** Ref used by the shell to transfer focus after path navigation. */
  titleRef?: React.Ref<HTMLHeadingElement>;
}

function Topbar({ leading, breadcrumb, title, titleRef, className, ...props }: TopbarProps) {
  const slot = useTopbarSlotValue();
  const hasRouteNav = Boolean(slot?.routeNav);
  const hasLeading = leading != null;

  return (
    <header
      data-slot="topbar"
      className={cn(
        "grid h-12 min-w-0 shrink-0 items-center gap-3 overflow-hidden border-b border-line bg-canvas px-4",
        hasRouteNav
          ? hasLeading
            ? "grid-cols-[auto_minmax(0,1fr)_minmax(0,auto)_auto]"
            : "grid-cols-[minmax(0,1fr)_minmax(0,2fr)_auto] lg:grid-cols-[minmax(0,1fr)_minmax(0,auto)_minmax(0,1fr)]"
          : hasLeading
            ? "grid-cols-[1fr_auto_1fr]"
            : "grid-cols-[minmax(0,1fr)_auto]",
        className
      )}
      {...props}
    >
      {hasLeading ? (
        <div data-slot="topbar-leading" className="flex min-w-0 items-center justify-self-start">
          {leading}
        </div>
      ) : null}
      <div
        data-slot="topbar-context"
        className={cn(
          "flex min-w-0 items-center gap-2 overflow-hidden",
          hasLeading && !hasRouteNav && "justify-self-center"
        )}
      >
        {breadcrumb ? (
          <div data-slot="topbar-breadcrumb" className="hidden min-w-0 sm:block">
            {breadcrumb}
          </div>
        ) : null}
        <h1
          ref={titleRef}
          tabIndex={-1}
          data-slot="topbar-title"
          data-testid="topbar-title-text"
          className="min-w-0 truncate text-card-title font-medium tracking-tight text-fg-strong outline-none focus-visible:shadow-focus-ring"
        >
          {title}
        </h1>
      </div>
      {hasRouteNav ? (
        <div
          data-slot="topbar-route-nav"
          className="no-scrollbar flex min-w-0 max-w-full items-center justify-self-stretch overflow-x-auto overscroll-x-contain lg:justify-self-center"
        >
          {slot?.routeNav}
        </div>
      ) : null}
      <div
        data-slot="topbar-trailing"
        className="flex min-w-0 items-center justify-end gap-2 justify-self-end"
      >
        {slot?.actions ? (
          // The zone owns the row layout so block-level action composites
          // (e.g. a selector div + button cluster) never stack into a second line.
          <div data-slot="topbar-actions" className="flex min-w-0 items-center gap-2">
            {slot.actions}
          </div>
        ) : null}
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

export { Topbar, TopbarOverflowIcon, TopbarSlotProvider, useTopbarSlot, useTopbarSlotValue };
export type { TopbarSlotValue };
