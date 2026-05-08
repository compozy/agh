import { cn } from "@agh/ui";
import type { ComponentProps } from "react";

export type MonoEyebrowTone = "neutral" | "accent" | "success" | "danger" | "warning" | "info";

const toneClass: Record<MonoEyebrowTone, string> = {
  neutral: "text-(--color-text-label)",
  accent: "text-accent",
  success: "text-success",
  danger: "text-(--color-danger)",
  warning: "text-(--color-warning)",
  info: "text-(--color-info)",
};

export interface MonoEyebrowProps extends ComponentProps<"span"> {
  tone?: MonoEyebrowTone;
  tracking?: "default" | "wide";
}

export function MonoEyebrow({
  tone = "neutral",
  tracking = "default",
  className,
  ...props
}: MonoEyebrowProps) {
  return (
    <span
      {...props}
      className={cn(
        "font-mono text-eyebrow font-semibold uppercase",
        tracking === "wide" ? "tracking-badge" : "tracking-mono",
        toneClass[tone],
        className
      )}
    />
  );
}
