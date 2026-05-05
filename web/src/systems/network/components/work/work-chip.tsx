import { cn } from "@/lib/utils";

import {
  formatNetworkWorkStateLabel,
  isNetworkWorkState,
  shouldRenderNetworkWorkChip,
  type NetworkWorkState,
} from "../../lib/network-formatters";
import { formatElapsedSeconds, useElapsedSeconds } from "../../lib/use-elapsed";

export interface WorkChipProps {
  /** Network work state — silent for `submitted` / `completed` per `_design.md` §6.6. */
  state: string | null | undefined;
  /** When set + state is `working`, the chip ticks with elapsed seconds. */
  startedAt?: string | null;
  className?: string;
  onClick?: () => void;
  /** Optional `aria-label` override (e.g. "open work · working · 12s"). */
  ariaLabel?: string;
}

const STATE_BG: Record<NetworkWorkState, string | null> = {
  submitted: null,
  working: "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)]",
  needs_input:
    "bg-[color:var(--color-warning-tint)] text-[color:var(--color-warning)] motion-safe:animate-pulse",
  completed: null,
  failed: "bg-[color:var(--color-danger-tint)] text-[color:var(--color-danger)]",
  canceled: "text-[color:var(--color-text-tertiary)]",
};

export function WorkChip({ state, startedAt, className, onClick, ariaLabel }: WorkChipProps) {
  const elapsed = useElapsedSeconds(startedAt, {
    enabled: state === "working",
  });

  if (!shouldRenderNetworkWorkChip(state)) {
    return null;
  }
  if (!isNetworkWorkState(state)) {
    return null;
  }

  const label = formatNetworkWorkStateLabel(state);
  const elapsedText = state === "working" && startedAt ? formatElapsedSeconds(elapsed) : "";
  const text = elapsedText ? `${label} · ${elapsedText}` : label;
  const bg = STATE_BG[state];

  if (onClick) {
    return (
      <button
        aria-label={ariaLabel ?? text}
        className={cn(
          "inline-flex items-center gap-1 rounded-[4px] px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-[0.06em]",
          bg,
          className
        )}
        data-testid="network-work-chip"
        data-state={state}
        onClick={onClick}
        type="button"
      >
        {text}
      </button>
    );
  }

  return (
    <span
      aria-label={ariaLabel ?? text}
      className={cn(
        "inline-flex items-center gap-1 rounded-[4px] px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-[0.06em]",
        bg,
        className
      )}
      data-testid="network-work-chip"
      data-state={state}
    >
      {text}
    </span>
  );
}
