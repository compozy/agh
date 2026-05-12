"use client";

import { PanelLeftIcon } from "lucide-react";
import * as React from "react";

import { cn } from "../lib/utils";
import { useInitialState } from "./use-initial-state";

const SIDEBAR_RAIL_WIDTH = 56;
const SIDEBAR_PANEL_WIDTH_DEFAULT = 244;
const SIDEBAR_PANEL_WIDTH_MD = 220;
const SIDEBAR_PANEL_WIDTH_MD_BREAKPOINT = 1100;
const SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT = 880;

export type SidebarViewport = "default" | "md" | "drawer";

export interface SidebarProps extends Omit<React.ComponentProps<"aside">, "onChange"> {
  rail?: React.ReactNode;
  header?: React.ReactNode;
  nav: React.ReactNode;
  footer?: React.ReactNode;
  collapsed?: boolean;
  defaultCollapsed?: boolean;
  onCollapse?: (next: boolean) => void;
  /** Explicit panel width override. When omitted the panel resolves from the viewport ladder. */
  panelWidth?: number;
  /** Width below which the panel becomes the drawer (replaces the legacy 768 px breakpoint). */
  collapseBreakpoint?: number;
  /** Width below which the panel shrinks from 244 px (default) to 220 px (md). */
  mdBreakpoint?: number;
  collapseLabel?: string;
}

interface ViewportThresholds {
  drawer: number;
  md: number;
}

/**
 * Returns the active viewport tier — `"default"`
 * above 1100 px, `"md"` between the drawer threshold and 1100 px, `"drawer"`
 * at or below the drawer threshold. The previous boolean
 * `useNarrowViewport(breakpoint)` collapsed the ladder into one query and
 * lost the 220 px tier.
 */
export function useSidebarViewport(thresholds: ViewportThresholds): SidebarViewport {
  const { drawer, md } = thresholds;
  const [viewport, setViewport] = React.useState<SidebarViewport>("default");
  React.useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return;
    const drawerQuery = window.matchMedia(`(max-width: ${Math.max(0, drawer - 1)}px)`);
    const mdQuery = window.matchMedia(`(max-width: ${Math.max(0, md - 1)}px)`);
    function evaluate() {
      const nextViewport: SidebarViewport = drawerQuery.matches
        ? "drawer"
        : mdQuery.matches
          ? "md"
          : "default";
      setViewport(nextViewport);
    }
    evaluate();
    drawerQuery.addEventListener("change", evaluate);
    mdQuery.addEventListener("change", evaluate);
    return () => {
      drawerQuery.removeEventListener("change", evaluate);
      mdQuery.removeEventListener("change", evaluate);
    };
  }, [drawer, md]);
  return viewport;
}

function resolvePanelWidth(viewport: SidebarViewport, override?: number): number {
  if (override !== undefined) return override;
  if (viewport === "md") return SIDEBAR_PANEL_WIDTH_MD;
  return SIDEBAR_PANEL_WIDTH_DEFAULT;
}

function Sidebar({
  rail,
  header,
  nav,
  footer,
  collapsed: collapsedProp,
  defaultCollapsed = false,
  onCollapse,
  panelWidth: panelWidthOverride,
  collapseBreakpoint = SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT,
  mdBreakpoint = SIDEBAR_PANEL_WIDTH_MD_BREAKPOINT,
  collapseLabel = "Toggle sidebar",
  className,
  ...props
}: SidebarProps) {
  const isControlled = collapsedProp !== undefined;
  const [uncontrolled, setUncontrolled] = useInitialState(defaultCollapsed);
  const [mobileOpen, setMobileOpen] = React.useState(false);
  const panelRef = React.useRef<HTMLDivElement | null>(null);
  const userCollapsed = isControlled ? Boolean(collapsedProp) : uncontrolled;
  const viewport = useSidebarViewport({ drawer: collapseBreakpoint, md: mdBreakpoint });
  const isDrawer = viewport === "drawer";
  const panelVisible = isDrawer ? mobileOpen : !userCollapsed;
  const effectivelyCollapsed = !panelVisible;
  const resolvedPanelWidth = resolvePanelWidth(viewport, panelWidthOverride);

  const setCollapsed = React.useCallback(
    (next: boolean) => {
      if (!isControlled) setUncontrolled(next);
      onCollapse?.(next);
    },
    [isControlled, onCollapse]
  );

  const handleToggle = React.useCallback(() => {
    if (isDrawer) {
      setMobileOpen(current => !current);
      return;
    }
    setCollapsed(!userCollapsed);
  }, [isDrawer, setCollapsed, userCollapsed]);

  React.useEffect(() => {
    if (!isDrawer && mobileOpen) {
      setMobileOpen(false);
    }
  }, [isDrawer, mobileOpen]);

  React.useEffect(() => {
    if (!isDrawer || !mobileOpen || typeof window === "undefined") return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMobileOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isDrawer, mobileOpen]);

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
      data-narrow={isDrawer ? "true" : "false"}
      data-viewport={viewport}
      className={cn("relative flex h-full shrink-0 border-r border-line bg-rail", className)}
      {...props}
    >
      {isDrawer && panelVisible ? (
        <button
          type="button"
          aria-label="Close sidebar navigation"
          onClick={() => setMobileOpen(false)}
          className="absolute inset-y-0 z-40 bg-overlay-scrim"
          style={{ left: SIDEBAR_RAIL_WIDTH, width: "100vw" }}
        />
      ) : null}
      <div
        data-slot="sidebar-rail"
        className="relative z-50 flex shrink-0 flex-col items-center gap-1.5 border-r border-line bg-rail py-3"
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
            isDrawer
              ? panelVisible
                ? "Close sidebar navigation"
                : "Open sidebar navigation"
              : collapseLabel
          }
          aria-expanded={panelVisible}
          onClick={handleToggle}
          className="inline-flex size-7 items-center justify-center rounded-md text-muted transition-colors hover:bg-hover hover:text-fg focus-visible:ring-2 focus-visible:ring-line-strong focus-visible:outline-none"
        >
          <PanelLeftIcon aria-hidden="true" className="size-3" />
        </button>
      </div>
      <div
        ref={panelRef}
        data-slot="sidebar-panel"
        style={{
          width: panelVisible ? resolvedPanelWidth : 0,
          ...(isDrawer ? { left: SIDEBAR_RAIL_WIDTH } : {}),
        }}
        className={cn(
          "flex min-h-0 flex-col overflow-hidden bg-sidebar",
          panelVisible ? "visible pointer-events-auto" : "pointer-events-none invisible",
          isDrawer && "absolute inset-y-0 z-50 border-r border-line"
        )}
        aria-hidden={effectivelyCollapsed}
      >
        <div
          className="flex h-full min-h-0 flex-col"
          style={{ width: resolvedPanelWidth, flexShrink: 0 }}
        >
          {header ? (
            <div
              data-slot="sidebar-header"
              className="flex min-h-12 shrink-0 items-center gap-2 border-b border-line px-2"
            >
              {header}
            </div>
          ) : null}
          <div data-slot="sidebar-nav" className="min-h-0 flex-1 overflow-y-auto">
            {nav}
          </div>
          {footer ? (
            <div data-slot="sidebar-footer" className="shrink-0 border-t border-line px-2 py-2.5">
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
      className={cn("eyebrow flex items-center px-2 pt-3 pb-1.5 text-muted", className)}
    />
  );
}

export {
  Sidebar,
  SIDEBAR_COLLAPSE_BREAKPOINT_DEFAULT,
  SIDEBAR_PANEL_WIDTH_DEFAULT,
  SIDEBAR_PANEL_WIDTH_MD,
  SIDEBAR_PANEL_WIDTH_MD_BREAKPOINT,
  SIDEBAR_RAIL_WIDTH,
  SidebarSectionLabel,
};
