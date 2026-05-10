import * as React from "react";

import { cn } from "../../lib/utils";
import type { PillTone } from "./pill";

export type EyebrowWeight = "medium" | "semibold";
export type EyebrowCase = "sentence" | "upper";

export interface EyebrowProps extends Omit<React.ComponentProps<"span">, "case"> {
  tone?: PillTone;
  weight?: EyebrowWeight;
  case?: EyebrowCase;
}

const EYEBROW_TONE_CLASS: Record<PillTone, string> = {
  neutral: "text-(--muted)",
  accent: "text-(--accent)",
  success: "text-(--success)",
  warning: "text-(--warning)",
  danger: "text-(--danger)",
  info: "text-(--info)",
};

function Eyebrow({
  tone = "neutral",
  weight = "semibold",
  case: caseVariant = "sentence",
  className,
  ...props
}: EyebrowProps) {
  return (
    <span
      data-slot="eyebrow"
      data-tone={tone}
      data-weight={weight}
      data-case={caseVariant}
      className={cn(
        caseVariant === "upper"
          ? "font-mono text-[10.5px] uppercase tracking-[0.05em]"
          : "font-sans text-[12px] tracking-[-0.005em]",
        weight === "semibold" ? "font-semibold" : "font-medium",
        EYEBROW_TONE_CLASS[tone],
        className
      )}
      {...props}
    />
  );
}

export { Eyebrow };
