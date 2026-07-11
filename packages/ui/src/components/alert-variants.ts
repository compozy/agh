import { cva } from "class-variance-authority";

/**
 * Restrained operator alerts: neutral surface + hairline border; signal color
 * lives on the leading icon only. Titles stay fg-strong; descriptions stay muted.
 * Matches web/CLAUDE.md — signal palette is information, never solid semantic banners.
 */
const alertVariants = cva(
  [
    "group/alert relative grid w-full gap-x-2.5 gap-y-2 rounded-md border border-line bg-canvas-soft",
    "px-3.5 py-3 text-left text-small-body text-fg",
    "has-data-[slot=alert-action]:relative has-data-[slot=alert-action]:pr-18",
    "has-[>svg]:grid-cols-[auto_1fr]",
    "*:data-[slot=alert-title]:text-fg-strong",
    "*:data-[slot=alert-description]:text-muted",
    "*:[svg]:row-span-full *:[svg]:translate-y-0.5 *:[svg:not([class*='size-'])]:size-3.5",
  ].join(" "),
  {
    variants: {
      variant: {
        default: "[&>svg]:text-subtle",
        neutral: "[&>svg]:text-subtle",
        danger: "[&>svg]:text-danger",
        warning: "[&>svg]:text-warning",
        success: "[&>svg]:text-success",
        info: "[&>svg]:text-info",
        accent: "[&>svg]:text-accent",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export { alertVariants };
