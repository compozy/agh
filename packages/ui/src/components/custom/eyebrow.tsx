import * as React from "react";

import { cn } from "../../lib/utils";

export type EyebrowWeight = "medium" | "semibold";
export type EyebrowCase = "sentence" | "upper";
export type EyebrowSize = "eyebrow" | "badge" | "micro";
export type EyebrowTone =
  | "neutral"
  | "muted"
  | "subtle"
  | "strong"
  | "accent"
  | "success"
  | "warning"
  | "danger"
  | "info";

export interface EyebrowProps extends Omit<React.ComponentProps<"span">, "case"> {
  tone?: EyebrowTone;
  weight?: EyebrowWeight;
  case?: EyebrowCase;
  size?: EyebrowSize;
}

const TONE_CLASS: Record<EyebrowTone, string> = {
  neutral: "text-(--muted)",
  muted: "text-(--muted)",
  subtle: "text-(--subtle)",
  strong: "text-(--fg-strong)",
  accent: "text-(--accent)",
  success: "text-(--success)",
  warning: "text-(--warning)",
  danger: "text-(--danger)",
  info: "text-(--info)",
};

const SIZE_UPPER_CLASS: Record<EyebrowSize, string> = {
  eyebrow: "text-eyebrow",
  badge: "text-badge",
  micro: "text-micro",
};

const SIZE_SENTENCE_CLASS: Record<EyebrowSize, string> = {
  eyebrow: "text-[12px]",
  badge: "text-[11px]",
  micro: "text-[10px]",
};

const WEIGHT_CLASS: Record<EyebrowWeight, string> = {
  medium: "font-medium",
  semibold: "font-semibold",
};

function Eyebrow({
  tone = "neutral",
  weight = "medium",
  case: caseVariant = "sentence",
  size = "eyebrow",
  className,
  ...props
}: EyebrowProps) {
  const isUpper = caseVariant === "upper";
  return (
    <span
      data-slot="eyebrow"
      data-tone={tone}
      data-weight={weight}
      data-case={caseVariant}
      data-size={size}
      className={cn(
        isUpper
          ? cn("font-mono uppercase tracking-mono", SIZE_UPPER_CLASS[size])
          : cn("font-sans tracking-[-0.005em]", SIZE_SENTENCE_CLASS[size]),
        WEIGHT_CLASS[weight],
        TONE_CLASS[tone],
        className
      )}
      {...props}
    />
  );
}

export { Eyebrow };
