import { cn } from "@agh/ui";
import type { ComponentProps } from "react";

export type MonoBadgeTone = "neutral" | "accent" | "success" | "danger" | "warning" | "info";

const toneClass: Record<MonoBadgeTone, string> = {
  neutral: "border-(--color-divider) text-(--color-text-label)",
  accent: "border-transparent bg-(--color-accent-tint) text-(--color-accent)",
  success: "border-transparent bg-(--color-success-tint) text-(--color-success)",
  danger: "border-transparent bg-(--color-danger-tint) text-(--color-danger)",
  warning: "border-transparent bg-(--color-warning-tint) text-(--color-warning)",
  info: "border-transparent bg-(--color-info-tint) text-(--color-info)",
};

export interface MonoBadgeProps extends ComponentProps<"span"> {
  tone?: MonoBadgeTone;
}

export function MonoBadge({ tone = "neutral", className, ...props }: MonoBadgeProps) {
  return (
    <span
      {...props}
      className={cn(
        "inline-flex items-center rounded-md border px-1.5 py-0.5 font-mono text-[11px] font-medium tracking-[0.06em]",
        toneClass[tone],
        className
      )}
    />
  );
}
