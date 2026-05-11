import { Toggle as TogglePrimitive } from "@base-ui/react/toggle";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../lib/utils";

const toggleVariants = cva(
  "group/toggle inline-flex items-center justify-center gap-1 rounded-md text-[12px] font-medium tracking-[-0.005em] whitespace-nowrap transition-all outline-none hover:bg-(--hover) hover:text-(--fg) focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-(--danger) aria-pressed:bg-(--elevated) aria-pressed:text-(--fg-strong) aria-pressed:shadow-[var(--highlight)] data-[state=on]:bg-(--elevated) data-[state=on]:text-(--fg-strong) data-[state=on]:shadow-[var(--highlight)] [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-transparent text-(--muted)",
        outline: "border border-(--line) bg-transparent text-(--fg) hover:bg-(--hover)",
      },
      size: {
        default:
          "h-9 min-w-9 px-2.5 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
        sm: "h-7 min-w-7 px-2.5 text-[11px] has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3.5",
        lg: "h-11 min-w-11 px-3 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

function Toggle({
  className,
  variant = "default",
  size = "default",
  ...props
}: TogglePrimitive.Props & VariantProps<typeof toggleVariants>) {
  return (
    <TogglePrimitive
      data-slot="toggle"
      className={cn(toggleVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Toggle, toggleVariants };
