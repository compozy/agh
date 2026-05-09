import * as React from "react";

import { cn } from "../../lib/utils";
import type { PillTone } from "./pill";

export type EyebrowWeight = "medium" | "semibold";

export interface EyebrowProps extends React.ComponentProps<"span"> {
  tone?: PillTone;
  weight?: EyebrowWeight;
}

const EYEBROW_TONE_CLASS: Record<PillTone, string> = {
  neutral: "text-(--color-text-tertiary)",
  accent: "text-(--color-accent)",
  success: "text-(--color-success)",
  warning: "text-(--color-warning)",
  danger: "text-(--color-danger)",
  info: "text-(--color-info)",
};

function Eyebrow({ tone = "neutral", weight = "semibold", className, ...props }: EyebrowProps) {
  return (
    <span
      data-slot="eyebrow"
      data-tone={tone}
      data-weight={weight}
      className={cn(
        "font-mono text-[11px] uppercase tracking-[0.06em]",
        weight === "semibold" ? "font-semibold" : "font-medium",
        EYEBROW_TONE_CLASS[tone],
        className
      )}
      {...props}
    />
  );
}

export { Eyebrow };
