import type * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../lib/utils";

const alertVariants = cva(
  "group/alert relative grid w-full gap-0.5 rounded-lg border px-2.5 py-2 text-left text-[13px] has-data-[slot=alert-action]:relative has-data-[slot=alert-action]:pr-18 has-[>svg]:grid-cols-[auto_1fr] has-[>svg]:gap-x-2 *:[svg]:row-span-2 *:[svg]:translate-y-0.5 *:[svg]:text-current *:[svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-(--canvas-soft) border-(--line) text-(--fg)",
        neutral:
          "border-(--neutral)/20 bg-(--neutral-tint) text-(--fg) *:data-[slot=alert-description]:text-(--muted)",
        danger:
          "border-(--danger)/20 bg-(--danger-tint) text-(--danger) *:data-[slot=alert-description]:text-(--danger)/85",
        warning:
          "border-(--warning)/20 bg-(--warning-tint) text-(--warning) *:data-[slot=alert-description]:text-(--warning)/85",
        success:
          "border-(--success)/20 bg-(--success-tint) text-(--success) *:data-[slot=alert-description]:text-(--success)/85",
        info: "border-(--info)/20 bg-(--info-tint) text-(--info) *:data-[slot=alert-description]:text-(--info)/85",
        accent:
          "border-(--accent)/20 bg-(--accent-tint) text-(--accent) *:data-[slot=alert-description]:text-(--accent)/85",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

type AlertProps = React.ComponentProps<"div"> & VariantProps<typeof alertVariants>;

function Alert({ className, variant, role = "alert", ...props }: AlertProps) {
  return (
    <div
      data-slot="alert"
      data-variant={variant ?? "default"}
      role={role}
      className={cn(alertVariants({ variant }), className)}
      {...props}
    />
  );
}

function AlertTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-title"
      className={cn(
        "font-[510] tracking-[-0.005em] group-has-[>svg]/alert:col-start-2 [&_a]:underline [&_a]:underline-offset-3 [&_a]:hover:text-(--fg-strong)",
        className
      )}
      {...props}
    />
  );
}

function AlertDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-description"
      className={cn(
        "text-[13px] text-balance md:text-pretty [&_a]:underline [&_a]:underline-offset-3 [&_a]:hover:text-(--fg-strong) [&_p:not(:last-child)]:mb-4",
        className
      )}
      {...props}
    />
  );
}

function AlertAction({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div data-slot="alert-action" className={cn("absolute top-2 right-2", className)} {...props} />
  );
}

function AlertMeta({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-meta"
      className={cn(
        "mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 font-mono text-badge uppercase tracking-mono text-current/75 group-has-[>svg]/alert:col-start-2",
        className
      )}
      {...props}
    />
  );
}

function AlertActions({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="alert-actions"
      className={cn(
        "mt-2 flex flex-wrap items-center justify-end gap-2 group-has-[>svg]/alert:col-start-2",
        className
      )}
      {...props}
    />
  );
}

export { Alert, AlertTitle, AlertDescription, AlertAction, AlertMeta, AlertActions, alertVariants };
export type { AlertProps };
