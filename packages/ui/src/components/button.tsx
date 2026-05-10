import { Button as ButtonPrimitive } from "@base-ui/react/button";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../lib/utils";

const buttonVariants = cva(
  "group/button inline-flex shrink-0 items-center justify-center rounded-md border border-transparent bg-clip-padding font-sans text-[12px] font-medium tracking-[-0.005em] whitespace-nowrap transition-all outline-none select-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 active:not-aria-[haspopup]:translate-y-px disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground shadow-[var(--highlight)] [a]:hover:bg-primary/80",
        outline:
          "border-(--line) bg-transparent text-(--fg) hover:bg-(--hover) aria-expanded:bg-(--hover)",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-(--hover) aria-expanded:bg-(--hover) aria-expanded:text-secondary-foreground",
        ghost:
          "text-(--fg) hover:bg-(--hover) aria-expanded:bg-(--hover) aria-expanded:text-(--fg)",
        destructive:
          "bg-(--danger-tint) text-(--danger) hover:bg-(--danger-tint) hover:opacity-90 focus-visible:border-(--danger)/40 focus-visible:ring-(--danger)/20",
        success:
          "bg-(--success-tint) text-(--success) hover:opacity-90 focus-visible:border-(--success)/40 focus-visible:ring-(--success)/20",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default:
          "h-[26px] gap-1.5 px-2.5 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        xs: "h-[22px] gap-1 rounded-sm px-2 text-[11px] has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: "h-[22px] gap-1 rounded-sm px-2.5 text-[11px] has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3.5",
        lg: "h-[30px] gap-1.5 px-3 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        icon: "size-[26px]",
        "icon-xs": "size-[22px] rounded-sm [&_svg:not([class*='size-'])]:size-3",
        "icon-sm": "size-[22px] rounded-sm",
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
