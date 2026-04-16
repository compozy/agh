import type { ReactNode } from "react";
import { cn } from "@agh/ui/utils";

type Tone = "accent" | "neutral" | "success";

const TONE_CLASS: Record<Tone, string> = {
  accent: "bg-(--color-accent-tint) text-(--color-accent)",
  neutral: "bg-(--color-surface-elevated) text-(--color-text-tertiary)",
  success: "bg-(--color-success-tint) text-(--color-success)",
};

interface MonoBadgeProps {
  children: ReactNode;
  tone?: Tone;
  className?: string;
}

/** 11px uppercase mono chip — the ubiquitous eyebrow/label on the landing page. */
export function MonoBadge({ children, tone = "accent", className }: MonoBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-[6px] px-2 py-1 font-mono text-[10px] font-semibold uppercase tracking-[0.08em]",
        TONE_CLASS[tone],
        className
      )}
    >
      {children}
    </span>
  );
}
