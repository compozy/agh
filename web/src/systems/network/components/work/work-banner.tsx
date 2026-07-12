import { useEffect, useReducer } from "react";
import { toast } from "sonner";

import { Alert, AlertActions, AlertDescription, Button } from "@agh/ui";

import { cn } from "@/lib/utils";

const FADE_OUT_MS = 200;
const COLLAPSE_MS = 200;
const TOTAL_HIDE_BUDGET_MS = FADE_OUT_MS + COLLAPSE_MS;

/**
 * Hard-stop threshold Crossing this boundary upward fires a
 * single `<Sonner>` toast — the banner itself stays tint-only across every
 * severity (info/warning/danger). The debounce guarantee: one toast per
 * `needs_input > 3` transition, not one per message.
 */
export const WORK_BANNER_HARD_STOP_THRESHOLD = 3;

export type WorkBannerTone = "info" | "warning" | "danger";

export interface WorkBannerProps {
  openCount: number;
  hasNeedsInput: boolean;
  /**
   * Optional authoritative breakdown. Never pass counts derived from a bounded
   * timeline page: omitting these fields keeps the exact `openCount` summary
   * truthful when the backend does not expose per-state aggregates.
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

function resolveTone(hasNeedsInput: boolean, needsInputCount: number | undefined): WorkBannerTone {
  if ((needsInputCount ?? 0) > WORK_BANNER_HARD_STOP_THRESHOLD) {
    return "danger";
  }
  if (hasNeedsInput || (needsInputCount ?? 0) > 0) {
    return "warning";
  }
  return "info";
}

const TONE_BG: Record<WorkBannerTone, string> = {
  info: "bg-info-tint",
  warning: "bg-warning-tint",
  danger: "bg-danger-tint",
};

const TONE_VIEW_TEXT: Record<WorkBannerTone, string> = {
  info: "text-info hover:bg-info-tint/40",
  warning: "text-warning hover:bg-warning-tint/40",
  danger: "text-danger hover:bg-danger-tint/40",
};

const TONE_ALERT_VARIANT: Record<WorkBannerTone, "info" | "warning" | "danger"> = {
  info: "info",
  warning: "warning",
  danger: "danger",
};

function notifyHardStopOnMount(node: HTMLSpanElement | null) {
  if (!node || node.dataset.notified === "true") return;
  node.dataset.notified = "true";
  const count = Number(node.dataset.count);
  toast.warning("Multiple agents are blocked on input.", {
    id: "network-work-banner-hard-stop",
    description: `${Number.isFinite(count) ? count : 0} agents need input.`,
  });
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

  const hardStopNotifier =
    (needsInputCount ?? 0) > WORK_BANNER_HARD_STOP_THRESHOLD ? (
      <span
        ref={notifyHardStopOnMount}
        aria-hidden="true"
        data-count={needsInputCount}
        data-testid="network-work-banner-hard-stop-notifier"
        hidden
      />
    ) : null;

  if (phase === "hidden") {
    return hardStopNotifier;
  }

  const tone = resolveTone(hasNeedsInput, needsInputCount);
  const message = buildMessage(openCount, hasNeedsInput, needsInputCount, workingCount);
  const fading = phase === "fading";

  return (
    <>
      {hardStopNotifier}
      <Alert
        aria-live="polite"
        className={cn(
          "flex h-9 items-center justify-between gap-3 overflow-hidden rounded-none border-x-0 border-t-0 border-b border-line px-5 py-0 transition-all duration-slow ease-out",
          TONE_BG[tone],
          fading ? "max-h-0 opacity-0" : "max-h-9 opacity-100",
          className
        )}
        data-state={fading ? "fading" : "visible"}
        data-testid="network-work-banner"
        data-tone={tone}
        role="status"
        variant={TONE_ALERT_VARIANT[tone]}
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
              className={cn("h-7 px-2 text-xs font-medium", TONE_VIEW_TEXT[tone])}
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
    </>
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
