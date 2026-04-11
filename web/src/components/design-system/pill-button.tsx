import { cva, type VariantProps } from "class-variance-authority";
import type { ButtonHTMLAttributes } from "react";

import { cn } from "@/lib/utils";

const pillButtonVariants = cva(
  "inline-flex items-center justify-center rounded-full border font-mono uppercase transition-colors duration-200",
  {
    variants: {
      active: {
        true: "border-[color:var(--color-accent)] bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)]",
        false:
          "border-[color:var(--color-divider)] bg-transparent text-[color:var(--color-text-secondary)] hover:border-[color:var(--color-text-label)] hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]",
      },
      size: {
        compact: "h-7 px-2.5 text-[0.64rem] tracking-[0.12em]",
        dense: "h-6 px-2 text-[0.6rem] tracking-[0.11em]",
      },
    },
    defaultVariants: {
      active: false,
      size: "compact",
    },
  }
);

function PillButton({
  active,
  className,
  size,
  type = "button",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & VariantProps<typeof pillButtonVariants>) {
  return (
    <button
      aria-pressed={active ?? undefined}
      className={cn(pillButtonVariants({ active, size }), className)}
      type={type}
      {...props}
    />
  );
}

export { PillButton, pillButtonVariants };
