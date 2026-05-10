"use client";

import * as React from "react";
import { PanelLeftIcon } from "lucide-react";

import { cn } from "../lib/utils";
import { useInitialState } from "./use-initial-state";

const SIDEBAR_RAIL_WIDTH = 56;
const SIDEBAR_PANEL_WIDTH_DEFAULT = 244;
const SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT = 768;

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
  const [uncontrolled, setUncontrolled] = useInitialState(defaultCollapsed);
  const [mobileOpen, setMobileOpen] = React.useState(false);
  const panelRef = React.useRef<HTMLDivElement | null>(null);
  const userCollapsed = isControlled ? Boolean(collapsedProp) : uncontrolled;
  const narrow = useNarrowViewport(collapseBreakpoint);
  const panelVisible = narrow ? mobileOpen : !userCollapsed;
  const effectivelyCollapsed = !panelVisible;

  const setCollapsed = React.useCallback(
    (next: boolean) => {
      if (!isControlled) setUncontrolled(next);
      onCollapse?.(next);
    },
    [isControlled, onCollapse]
  );

  const handleToggle = React.useCallback(() => {
    if (narrow) {
      setMobileOpen(current => !current);
      return;
    }

    setCollapsed(!userCollapsed);
  }, [narrow, setCollapsed, userCollapsed]);

  React.useEffect(() => {
    if (!narrow && mobileOpen) {
      setMobileOpen(false);
    }
  }, [mobileOpen, narrow]);

  React.useEffect(() => {
    if (!narrow || !mobileOpen || typeof window === "undefined") return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMobileOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [mobileOpen, narrow]);

  React.useEffect(() => {
    const panel = panelRef.current;
    if (!panel) return;

    panel.toggleAttribute("inert", effectivelyCollapsed);
    if ("inert" in panel) {
      panel.inert = effectivelyCollapsed;
    }

    return () => {
      panel.removeAttribute("inert");
      if ("inert" in panel) {
        panel.inert = false;
      }
    };
  }, [effectivelyCollapsed]);

  return (
    <aside
      data-slot="sidebar"
      data-state={effectivelyCollapsed ? "collapsed" : "expanded"}
      data-narrow={narrow ? "true" : "false"}
      className={cn(
        "relative flex h-full shrink-0 border-r border-(--line) bg-(--rail)",
        className
      )}
      {...props}
    >
      {narrow && panelVisible ? (
        <button
          type="button"
          aria-label="Close sidebar navigation"
          onClick={() => setMobileOpen(false)}
          className="absolute inset-y-0 z-40 bg-(--overlay-scrim)"
          style={{ left: SIDEBAR_RAIL_WIDTH, width: "100vw" }}
        />
      ) : null}
      <div
        data-slot="sidebar-rail"
        className="relative z-50 flex shrink-0 flex-col items-center gap-1.5 border-r border-(--line) bg-(--rail) py-3"
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
          aria-label={
            narrow
              ? panelVisible
                ? "Close sidebar navigation"
                : "Open sidebar navigation"
              : collapseLabel
          }
          aria-expanded={panelVisible}
          onClick={handleToggle}
          className="inline-flex size-7 items-center justify-center rounded-md text-(--muted) transition-colors hover:bg-(--hover) hover:text-(--fg) focus-visible:ring-2 focus-visible:ring-(--line-strong) focus-visible:outline-none"
        >
          <PanelLeftIcon aria-hidden="true" className="size-3.5" />
        </button>
      </div>
      <div
        ref={panelRef}
        data-slot="sidebar-panel"
        style={{
          width: panelVisible ? panelWidth : 0,
          ...(narrow ? { left: SIDEBAR_RAIL_WIDTH } : {}),
        }}
        className={cn(
          "flex min-h-0 flex-col overflow-hidden bg-(--sidebar)",
          panelVisible ? "visible pointer-events-auto" : "pointer-events-none invisible",
          narrow && "absolute inset-y-0 z-50 border-r border-(--line)"
        )}
        aria-hidden={effectivelyCollapsed}
      >
        <div className="flex h-full min-h-0 flex-col" style={{ width: panelWidth, flexShrink: 0 }}>
          {header ? (
            <div
              data-slot="sidebar-header"
              className="flex min-h-12 shrink-0 items-center gap-2 border-b border-(--line) px-3"
            >
              {header}
            </div>
          ) : null}
          <div data-slot="sidebar-nav" className="min-h-0 flex-1 overflow-y-auto">
            {nav}
          </div>
          {footer ? (
            <div
              data-slot="sidebar-footer"
              className="shrink-0 border-t border-(--line) px-3 py-2.5"
            >
              {footer}
            </div>
          ) : null}
        </div>
      </div>
    </aside>
  );
}

function SidebarSectionLabel({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      data-slot="sidebar-section-label"
      className={cn(
        "px-3 pt-3 pb-1.5 font-mono text-[10.5px] font-medium uppercase tracking-[0.05em] text-(--muted)",
        className
      )}
    />
  );
}

export { Sidebar, SidebarSectionLabel, SIDEBAR_RAIL_WIDTH, SIDEBAR_PANEL_WIDTH_DEFAULT };
