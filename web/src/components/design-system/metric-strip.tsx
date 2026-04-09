import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

import { StatusDot } from "./status-dot";

const metricStripVariants = cva(
  "rounded-[0.75rem] border bg-[color:var(--color-surface-elevated)] p-4 text-[color:var(--color-text-primary)] sm:p-5",
  {
    variants: {
      tone: {
        neutral: "border-[color:var(--color-divider)]",
        amber: "border-[color:var(--color-accent)]",
        green: "border-[color:var(--color-success)]",
        violet: "border-[color:var(--color-info)]",
      },
    },
    defaultVariants: {
      tone: "neutral",
    },
  }
);

interface MetricStripProps extends ComponentProps<"div">, VariantProps<typeof metricStripVariants> {
  detail?: string;
  label: string;
  value: string;
}

function MetricStrip({ className, detail, label, tone, value, ...props }: MetricStripProps) {
  const dotTone = tone === "neutral" ? "amber" : tone;

  return (
    <div className={cn(metricStripVariants({ tone }), className)} {...props}>
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <StatusDot tone={dotTone} />
            <p className="font-mono text-[0.64rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
              {label}
            </p>
          </div>
          <p className="font-sans text-4xl leading-none font-semibold tracking-[-0.04em]">
            {value}
          </p>
        </div>
        {detail ? (
          <p className="max-w-[11rem] text-right text-sm leading-6 text-[color:var(--color-text-secondary)]">
            {detail}
          </p>
        ) : null}
      </div>
    </div>
  );
}

export { MetricStrip };
