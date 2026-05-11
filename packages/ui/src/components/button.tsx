import { Button as ButtonPrimitive } from "@base-ui/react/button";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../lib/utils";

const buttonVariants = cva(
  "group/button inline-flex shrink-0 items-center justify-center rounded-md border border-transparent bg-clip-padding font-sans text-[12px] font-medium tracking-eyebrow whitespace-nowrap transition-all outline-none select-none focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] active:not-aria-[haspopup]:translate-y-px disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-(--danger) [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-(--accent) text-(--accent-ink) shadow-(--highlight) hover:bg-(--accent-hover) [a]:hover:bg-(--accent-hover)",
        primary:
          "bg-(--accent) text-(--accent-ink) shadow-(--highlight) hover:bg-(--accent-hover) [a]:hover:bg-(--accent-hover)",
        outline:
          "border-(--line) bg-transparent text-(--fg) hover:bg-(--hover) aria-expanded:bg-(--hover)",
        secondary:
          "bg-(--canvas-tint) border-(--line) text-(--fg) hover:bg-(--hover) aria-expanded:bg-(--hover)",
        ghost:
          "text-(--fg) hover:bg-(--hover) aria-expanded:bg-(--hover) aria-expanded:text-(--fg)",
        destructive: "bg-(--danger-tint) text-(--danger) hover:bg-(--danger-tint) hover:opacity-90",
        success: "bg-(--success-tint) text-(--success) hover:opacity-90",
        link: "text-(--accent) underline-offset-4 hover:underline",
        neutral: "bg-(--btn-default-fill) text-(--fg-strong) hover:bg-(--btn-default-hover)",
      },
      size: {
        default:
          "h-[26px] gap-1.5 px-2.5 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        xs: "h-[22px] gap-1 rounded-md px-2 text-[11px] has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-[22px] gap-1 rounded-md px-2.5 text-[11px] has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3.5",
        lg: "h-[30px] gap-1.5 px-3 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        cta: "h-9 gap-2 px-5 has-data-[icon=inline-end]:pr-3 has-data-[icon=inline-start]:pl-3",
        "cta-lg":
          "h-11 gap-2 rounded-md px-5 text-[13px] has-data-[icon=inline-end]:pr-3 has-data-[icon=inline-start]:pl-3",
        icon: "size-[26px]",
        "icon-xs": "size-[22px] rounded-md [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-[22px] rounded-md",
        "icon-lg": "size-[30px]",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
