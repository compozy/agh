import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

const pillVariants = cva(
  "inline-flex items-center justify-center gap-2 rounded-full border font-mono uppercase transition-colors duration-200",
  {
    variants: {
      emphasis: {
        muted: "text-[color:var(--color-text-secondary)]",
        strong: "text-[color:var(--color-text-primary)]",
      },
      kind: {
        filter: "h-9 px-3 text-[0.64rem] tracking-[0.14em]",
        tag: "h-7 px-2.5 text-[0.625rem] tracking-[0.12em]",
        state: "h-6 px-2.5 text-[0.625rem] tracking-[0.12em]",
      },
      tone: {
        neutral: "bg-[color:var(--color-neutral-tint)] border-[color:var(--color-divider)]",
        amber: "bg-[color:var(--color-accent-tint)] border-[color:var(--color-accent)]",
        green: "bg-[color:var(--color-success-tint)] border-[color:var(--color-success)]",
        violet: "bg-[color:var(--color-info-tint)] border-[color:var(--color-info)]",
        danger: "bg-[color:var(--color-danger-tint)] border-[color:var(--color-danger)]",
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
