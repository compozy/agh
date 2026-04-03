import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

const statusDotVariants = cva("inline-flex size-2.5 rounded-full ring-2 ring-transparent", {
  variants: {
    tone: {
      neutral: "bg-[color:var(--ds-text-muted)]",
      amber: "bg-[color:var(--ds-accent-amber)]",
      green: "bg-[color:var(--ds-accent-green)]",
      violet: "bg-[color:var(--ds-accent-violet)]",
      danger: "bg-[color:var(--ds-accent-danger)]",
    },
  },
  defaultVariants: {
    tone: "neutral",
  },
});

function StatusDot({
  className,
  tone,
  ...props
}: ComponentProps<"span"> & VariantProps<typeof statusDotVariants>) {
  return (
    <span aria-hidden="true" className={cn(statusDotVariants({ tone }), className)} {...props} />
  );
}

export { StatusDot };
