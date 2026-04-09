import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

const statusDotVariants = cva("inline-flex size-2.5 rounded-full ring-2 ring-transparent", {
  variants: {
    tone: {
      neutral: "bg-[color:var(--color-text-tertiary)]",
      amber: "bg-[color:var(--color-warning)]",
      green: "bg-[color:var(--color-success)]",
      violet: "bg-[color:var(--color-info)]",
      danger: "bg-[color:var(--color-danger)]",
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
