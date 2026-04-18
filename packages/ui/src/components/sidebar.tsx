"use client";

import * as React from "react";
import { motion, useReducedMotionConfig } from "motion/react";
import { PanelLeftIcon } from "lucide-react";

import { cn } from "../lib/utils";

const SIDEBAR_RAIL_WIDTH = 44;
const SIDEBAR_PANEL_WIDTH_DEFAULT = 240;
const SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT = 768;
const SIDEBAR_MOTION_DURATION = 0.2;
const SIDEBAR_MOTION_EASE = [0.4, 0, 0.2, 1] as const;

export interface SidebarProps extends Omit<React.ComponentProps<"aside">, "onChange"> {
  rail?: React.ReactNode;
  header?: React.ReactNode;
  nav: React.ReactNode;
  footer?: React.ReactNode;
  collapsed?: boolean;
  defaultCollapsed?: boolean;
  onCollapse?: (next: boolean) => void;
  panelWidth?: number;
  collapseBreakpoint?: number;
  collapseLabel?: string;
}

function useNarrowViewport(breakpoint: number): boolean {
  const [narrow, setNarrow] = React.useState(false);
  React.useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return;
    const query = window.matchMedia(`(max-width: ${Math.max(0, breakpoint - 1)}px)`);
    const handler = (event: MediaQueryListEvent | MediaQueryList) => {
      setNarrow(event.matches);
    };
    handler(query);
    query.addEventListener("change", handler);
    return () => query.removeEventListener("change", handler);
  }, [breakpoint]);
  return narrow;
}

function Sidebar({
  rail,
  header,
  nav,
  footer,
  collapsed: collapsedProp,
  defaultCollapsed = false,
  onCollapse,
  panelWidth = SIDEBAR_PANEL_WIDTH_DEFAULT,
  collapseBreakpoint = SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT,
  collapseLabel = "Toggle sidebar",
  className,
  ...props
}: SidebarProps) {
  const isControlled = collapsedProp !== undefined;
  const [uncontrolled, setUncontrolled] = React.useState(defaultCollapsed);
  const userCollapsed = isControlled ? Boolean(collapsedProp) : uncontrolled;
  const narrow = useNarrowViewport(collapseBreakpoint);
  const effectivelyCollapsed = userCollapsed || narrow;

  const reducedMotion = useReducedMotionConfig();
  const duration = reducedMotion ? 0 : SIDEBAR_MOTION_DURATION;

  const setCollapsed = React.useCallback(
    (next: boolean) => {
      if (!isControlled) setUncontrolled(next);
      onCollapse?.(next);
    },
    [isControlled, onCollapse]
  );

  const handleToggle = React.useCallback(() => {
    setCollapsed(!userCollapsed);
  }, [setCollapsed, userCollapsed]);

  return (
    <aside
      data-slot="sidebar"
      data-state={effectivelyCollapsed ? "collapsed" : "expanded"}
      data-narrow={narrow ? "true" : "false"}
      className={cn(
        "flex h-full shrink-0 border-r border-border bg-[color:var(--color-canvas-deep)]",
        className
      )}
      {...props}
    >
      <div
        data-slot="sidebar-rail"
        className="flex shrink-0 flex-col items-center gap-1.5 border-r border-border py-3"
        style={{ width: SIDEBAR_RAIL_WIDTH }}
      >
        {rail ? (
          <div className="flex flex-1 flex-col items-center gap-1.5">{rail}</div>
        ) : (
          <div className="flex-1" />
        )}
        <button
          type="button"
          data-slot="sidebar-collapse-trigger"
          aria-label={collapseLabel}
          aria-expanded={!effectivelyCollapsed}
          onClick={handleToggle}
          className="inline-flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-[color:var(--color-hover)] hover:text-foreground focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:outline-none"
        >
          <PanelLeftIcon aria-hidden="true" className="size-3.5" />
        </button>
      </div>
      <motion.div
        data-slot="sidebar-panel"
        initial={false}
        animate={{ width: effectivelyCollapsed ? 0 : panelWidth }}
        transition={{ duration, ease: SIDEBAR_MOTION_EASE }}
        className="flex min-h-0 flex-col overflow-hidden bg-[color:var(--color-surface)]"
        aria-hidden={effectivelyCollapsed}
      >
        <div className="flex h-full min-h-0 flex-col" style={{ width: panelWidth, flexShrink: 0 }}>
          {header ? (
            <div
              data-slot="sidebar-header"
              className="flex min-h-11 shrink-0 items-center gap-2 border-b border-border px-3"
            >
              {header}
            </div>
          ) : null}
          <div data-slot="sidebar-nav" className="min-h-0 flex-1 overflow-y-auto">
            {nav}
          </div>
          {footer ? (
            <div data-slot="sidebar-footer" className="shrink-0 border-t border-border px-3 py-2.5">
              {footer}
            </div>
          ) : null}
        </div>
      </motion.div>
    </aside>
  );
}

export { Sidebar, SIDEBAR_RAIL_WIDTH, SIDEBAR_PANEL_WIDTH_DEFAULT };
