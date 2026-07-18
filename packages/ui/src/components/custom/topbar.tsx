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

export interface TopbarProps extends React.ComponentProps<"header"> {
  /**
   * Leading zone content — the route breadcrumb built by the shell.
   * The topbar answers "where?": no route icon, H1, count, status, search,
   * or scope selectors live here (route chrome contract §04).
   */
  breadcrumb?: React.ReactNode;
}

function Topbar({ breadcrumb, className, ...props }: TopbarProps) {
  const slot = useTopbarSlotValue();

  return (
    <header
      data-slot="topbar"
      className={cn(
        "grid h-12 min-w-0 shrink-0 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-3 overflow-hidden border-b border-line bg-canvas px-4",
        className
      )}
      {...props}
    >
      <div data-slot="topbar-context" className="flex min-w-0 items-center overflow-hidden">
        {breadcrumb}
      </div>
      <div data-slot="topbar-route-nav" className="flex min-w-0 items-center justify-self-center">
        {slot?.routeNav}
      </div>
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
