import { useEffect, useReducer } from "react";

import { Alert, AlertActions, AlertDescription, Button } from "@agh/ui";

import { cn } from "@/lib/utils";

const FADE_OUT_MS = 200;
const COLLAPSE_MS = 200;
const TOTAL_HIDE_BUDGET_MS = FADE_OUT_MS + COLLAPSE_MS;

export interface WorkBannerProps {
  openCount: number;
  hasNeedsInput: boolean;
  /**
   * Optional breakdown of open-work counts by lifecycle state. When provided,
   * the banner renders the explicit "needs input · working" segments instead of
   * the legacy summary message. `useOpenWork` derives this client-side from the
   * already-loaded message stream - see `_design.md` §5.8.2.
   */
  needsInputCount?: number;
  workingCount?: number;
  onView?: () => void;
  className?: string;
}

type BannerPhase = "hidden" | "visible" | "fading";
type BannerAction = { type: "show" } | { type: "fade" } | { type: "hide" };

function bannerPhaseReducer(phase: BannerPhase, action: BannerAction): BannerPhase {
  switch (action.type) {
    case "show":
      return "visible";
    case "fade":
      return phase === "hidden" ? "hidden" : "fading";
    case "hide":
      return "hidden";
  }
}

export function WorkBanner({
  openCount,
  hasNeedsInput,
  needsInputCount,
  workingCount,
  onView,
  className,
}: WorkBannerProps) {
  const [phase, dispatchPhase] = useReducer(
    bannerPhaseReducer,
    openCount > 0 ? "visible" : "hidden"
  );

  useEffect(() => {
    if (openCount > 0) {
      dispatchPhase({ type: "show" });
      return undefined;
    }

    dispatchPhase({ type: "fade" });
    const timer = setTimeout(() => {
      dispatchPhase({ type: "hide" });
    }, TOTAL_HIDE_BUDGET_MS);
    return () => clearTimeout(timer);
  }, [openCount]);

  if (phase === "hidden") {
    return null;
  }

  const escalate = hasNeedsInput && openCount > 0;
  const message = buildMessage(openCount, hasNeedsInput, needsInputCount, workingCount);
  const fading = phase === "fading";

  return (
    <Alert
      aria-live="polite"
      className={cn(
        "flex h-9 items-center justify-between gap-3 overflow-hidden rounded-none border-x-0 border-t-0 border-b border-(--color-divider) px-5 py-0 transition-[opacity,max-height] duration-200 ease-out",
        escalate
          ? "bg-(--color-warning) text-(--color-canvas) *:data-[slot=alert-description]:text-(--color-canvas)"
          : null,
        fading ? "max-h-0 opacity-0" : "max-h-9 opacity-100",
        className
      )}
      data-escalate={escalate ? "true" : "false"}
      data-state={fading ? "fading" : "visible"}
      data-testid="network-work-banner"
      role="status"
      variant="warning"
    >
      <AlertDescription
        className="truncate text-small-body font-medium"
        data-testid="network-work-banner-message"
      >
        {message}
      </AlertDescription>
      {onView ? (
        <AlertActions className="mt-0">
          <Button
            aria-label="View open work"
            className={cn(
              "h-7 px-2 text-xs font-medium",
              escalate
                ? "text-(--color-canvas) hover:bg-(--color-canvas)/10"
                : "text-(--color-warning) hover:bg-(--color-warning-tint)/40"
            )}
            data-testid="network-work-banner-view"
            onClick={onView}
            size="sm"
            type="button"
            variant="ghost"
          >
            view
          </Button>
        </AlertActions>
      ) : null}
    </Alert>
  );
}

function buildMessage(
  openCount: number,
  hasNeedsInput: boolean,
  needsInputCount: number | undefined,
  workingCount: number | undefined
): string {
  if (openCount === 0) {
    return "";
  }

  // Explicit breakdown wins when callers compute the per-state segmentation
  // from `useOpenWork` entries.
  if (needsInputCount != null || workingCount != null) {
    const segments: string[] = [];
    if ((needsInputCount ?? 0) > 0) {
      segments.push(`${needsInputCount} needs input`);
    }
    if ((workingCount ?? 0) > 0) {
      segments.push(`${workingCount} working`);
    }
    if (segments.length > 0) {
      return segments.join(" · ");
    }
  }

  if (!hasNeedsInput) {
    if (openCount === 1) {
      return "1 active work in flight";
    }
    return `${openCount} active work in flight`;
  }
  if (openCount === 1) {
    return "1 needs input";
  }
  // Escalation fallback when only `hasNeedsInput` flag is available.
  return `1 needs input · ${openCount - 1} working`;
}
