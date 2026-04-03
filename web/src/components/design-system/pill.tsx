import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

const pillVariants = cva(
  "inline-flex items-center justify-center gap-2 rounded-full border font-mono uppercase transition-colors duration-200",
  {
    variants: {
      emphasis: {
        muted: "text-[color:var(--ds-text-secondary)]",
        strong: "text-[color:var(--ds-text-primary)]",
      },
      kind: {
        filter: "h-9 px-3 text-[0.64rem] tracking-[0.14em]",
        tag: "h-7 px-2.5 text-[0.625rem] tracking-[0.12em]",
        state: "h-6 px-2.5 text-[0.625rem] tracking-[0.12em]",
      },
      tone: {
        neutral: "bg-[color:var(--ds-pill-neutral)] border-[color:var(--ds-pill-neutral-border)]",
        amber: "bg-[color:var(--ds-pill-amber)] border-[color:var(--ds-pill-amber-border)]",
        green: "bg-[color:var(--ds-pill-green)] border-[color:var(--ds-pill-green-border)]",
        violet: "bg-[color:var(--ds-pill-violet)] border-[color:var(--ds-pill-violet-border)]",
        danger: "bg-[color:var(--ds-pill-danger)] border-[color:var(--ds-pill-danger-border)]",
      },
    },
    defaultVariants: {
      emphasis: "muted",
      kind: "tag",
      tone: "neutral",
    },
  }
);

function Pill({
  className,
  emphasis,
  kind,
  tone,
  ...props
}: ComponentProps<"span"> & VariantProps<typeof pillVariants>) {
  return <span className={cn(pillVariants({ emphasis, kind, tone }), className)} {...props} />;
}

export { Pill, pillVariants };
