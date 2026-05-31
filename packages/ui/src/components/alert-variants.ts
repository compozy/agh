import { cva } from "class-variance-authority";

const alertVariants = cva(
  "group/alert relative grid w-full gap-0.5 rounded-lg border px-2.5 py-2 text-left text-small-body has-data-[slot=alert-action]:relative has-data-[slot=alert-action]:pr-18 has-[>svg]:grid-cols-[auto_1fr] has-[>svg]:gap-x-2 *:[svg]:row-span-2 *:[svg]:translate-y-0.5 *:[svg]:text-current *:[svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-canvas-soft border-line text-fg",
        neutral:
          "border-neutral/20 bg-neutral-tint text-fg *:data-[slot=alert-description]:text-muted",
        danger:
          "border-danger/20 bg-danger-tint text-danger *:data-[slot=alert-description]:text-danger/85",
        warning:
          "border-warning/20 bg-warning-tint text-warning *:data-[slot=alert-description]:text-warning/85",
        success:
          "border-success/20 bg-success-tint text-success *:data-[slot=alert-description]:text-success/85",
        info: "border-info/20 bg-info-tint text-info *:data-[slot=alert-description]:text-info/85",
        accent:
          "border-accent/20 bg-accent-tint text-accent *:data-[slot=alert-description]:text-accent/85",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export { alertVariants };
