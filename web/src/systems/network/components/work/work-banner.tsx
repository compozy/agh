import { useEffect, useState } from "react";

import { Button } from "@agh/ui";

import { cn } from "@/lib/utils";

const FADE_OUT_MS = 200;
const COLLAPSE_MS = 200;
const TOTAL_HIDE_BUDGET_MS = FADE_OUT_MS + COLLAPSE_MS;

export interface WorkBannerProps {
  openCount: number;
  hasNeedsInput: boolean;
  onView?: () => void;
  className?: string;
}

type BannerPhase = "hidden" | "visible" | "fading";

export function WorkBanner({ openCount, hasNeedsInput, onView, className }: WorkBannerProps) {
  const [phase, setPhase] = useState<BannerPhase>(openCount > 0 ? "visible" : "hidden");

  useEffect(() => {
    if (openCount > 0) {
      setPhase("visible");
      return undefined;
    }

    setPhase(prev => (prev === "hidden" ? "hidden" : "fading"));
    const timer = setTimeout(() => {
      setPhase("hidden");
    }, TOTAL_HIDE_BUDGET_MS);
    return () => clearTimeout(timer);
  }, [openCount]);

  if (phase === "hidden") {
    return null;
  }

  const escalate = hasNeedsInput && openCount > 0;
  const message = buildMessage(openCount, hasNeedsInput);
  const fading = phase === "fading";

  return (
    <div
      aria-live="polite"
      className={cn(
        "flex h-9 items-center justify-between gap-3 overflow-hidden border-b border-[color:var(--color-divider)] px-5 transition-[opacity,max-height] duration-200 ease-out",
        escalate
          ? "bg-[color:var(--color-warning)] text-[color:var(--color-canvas)]"
          : "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
        fading ? "max-h-0 opacity-0" : "max-h-9 opacity-100",
        className
      )}
      data-escalate={escalate ? "true" : "false"}
      data-state={fading ? "fading" : "visible"}
      data-testid="network-work-banner"
      role="status"
    >
      <p className="truncate text-[13px] font-medium" data-testid="network-work-banner-message">
        {message}
      </p>
      {onView ? (
        <Button
          aria-label="View open work"
          className={cn(
            "h-7 px-2 text-[12px] font-medium",
            escalate
              ? "text-[color:var(--color-canvas)] hover:bg-[color:var(--color-canvas)]/10"
              : "text-[color:var(--color-warning)] hover:bg-[color:var(--color-warning-tint)]/40"
          )}
          data-testid="network-work-banner-view"
          onClick={onView}
          size="sm"
          type="button"
          variant="ghost"
        >
          view
        </Button>
      ) : null}
    </div>
  );
}

function buildMessage(openCount: number, hasNeedsInput: boolean): string {
  if (openCount === 0) {
    return "";
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
  // Escalation message — show the needs_input slice + remainder.
  return `1 needs input · ${openCount - 1} working`;
}
