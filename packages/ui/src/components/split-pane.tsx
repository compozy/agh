"use client";

import * as React from "react";
import { AnimatePresence, motion, useReducedMotionConfig } from "motion/react";
import { ChevronLeftIcon } from "lucide-react";

import { cn } from "../lib/utils";

const SPLIT_LIST_WIDTH_DEFAULT = 340;
const SPLIT_NARROW_BREAKPOINT_DEFAULT = 768;
const SPLIT_DETAIL_DURATION = 0.15;

export interface SplitPaneProps extends Omit<React.ComponentProps<"div">, "onChange"> {
  list: React.ReactNode;
  detail?: React.ReactNode;
  listWidth?: number;
  detailEmpty?: React.ReactNode;
  onDetailClose?: () => void;
  narrowBreakpoint?: number;
  backLabel?: string;
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

function isDetailPresent(detail: React.ReactNode): boolean {
  return detail !== null && detail !== undefined && detail !== false;
}

function SplitPane({
  list,
  detail,
  listWidth = SPLIT_LIST_WIDTH_DEFAULT,
  detailEmpty,
  onDetailClose,
  narrowBreakpoint = SPLIT_NARROW_BREAKPOINT_DEFAULT,
  backLabel = "Back",
  className,
  ...props
}: SplitPaneProps) {
  const narrow = useNarrowViewport(narrowBreakpoint);
  const hasDetail = isDetailPresent(detail);
  const stackNarrowDetail = narrow && hasDetail && onDetailClose === undefined;
  const showList = stackNarrowDetail || !narrow || !hasDetail;
  const showDetail = stackNarrowDetail || !narrow || hasDetail;

  const reducedMotion = useReducedMotionConfig();
  const duration = reducedMotion ? 0 : SPLIT_DETAIL_DURATION;

  return (
    <div
      data-slot="split-pane"
      data-narrow={narrow ? "true" : "false"}
      className={cn("flex min-h-0 min-w-0 flex-1", stackNarrowDetail && "flex-col", className)}
      {...props}
    >
      {showList ? (
        <div
          data-slot="split-pane-list"
          className={cn(
            "flex min-h-0 shrink-0 flex-col bg-[color:var(--color-canvas)]",
            stackNarrowDetail ? "border-b border-border" : "border-r border-border"
          )}
          style={{ width: narrow ? "100%" : listWidth }}
        >
          {list}
        </div>
      ) : null}
      {showDetail ? (
        <div
          data-slot="split-pane-detail"
          className="flex min-h-0 min-w-0 flex-1 flex-col bg-[color:var(--color-canvas)]"
        >
          {narrow && hasDetail && !stackNarrowDetail ? (
            <div
              data-slot="split-pane-detail-bar"
              className="flex shrink-0 items-center gap-2 border-b border-border px-3 py-2"
            >
              <button
                type="button"
                data-slot="split-pane-back"
                onClick={onDetailClose}
                className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-[color:var(--color-hover)] hover:text-foreground focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:outline-none"
              >
                <ChevronLeftIcon aria-hidden="true" className="size-3.5" />
                <span>{backLabel}</span>
              </button>
            </div>
          ) : null}
          <AnimatePresence initial={false} mode="wait">
            <motion.div
              key={hasDetail ? "detail" : "empty"}
              data-slot={hasDetail ? "split-pane-detail-body" : "split-pane-detail-empty"}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration, ease: "easeOut" }}
              className="flex min-h-0 min-w-0 flex-1 flex-col"
            >
              {hasDetail ? detail : detailEmpty}
            </motion.div>
          </AnimatePresence>
        </div>
      ) : null}
    </div>
  );
}

export { SplitPane, SPLIT_LIST_WIDTH_DEFAULT };
