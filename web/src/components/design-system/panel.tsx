import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

const panelVariants = cva("ds-panel relative flex flex-col gap-5", {
  variants: {
    tone: {
      default: "ds-panel-surface",
      elevated: "ds-panel-elevated",
      accented: "ds-panel-accented",
    },
  },
  defaultVariants: {
    tone: "default",
  },
});

function Panel({
  className,
  tone,
  ...props
}: ComponentProps<"section"> & VariantProps<typeof panelVariants>) {
  return <section className={cn(panelVariants({ tone }), className)} {...props} />;
}

function PanelHeader({ className, ...props }: ComponentProps<"header">) {
  return <header className={cn("flex flex-col gap-3", className)} {...props} />;
}

function PanelTitle({ className, ...props }: ComponentProps<"h2">) {
  return (
    <h2
      className={cn(
        "text-balance text-lg font-medium text-[color:var(--color-text-primary)]",
        className
      )}
      {...props}
    />
  );
}

function PanelDescription({ className, ...props }: ComponentProps<"p">) {
  return (
    <p
      className={cn(
        "max-w-2xl text-sm leading-6 text-[color:var(--color-text-secondary)]",
        className
      )}
      {...props}
    />
  );
}

function PanelBody({ className, ...props }: ComponentProps<"div">) {
  return <div className={cn("flex flex-col gap-4", className)} {...props} />;
}

function PanelFooter({ className, ...props }: ComponentProps<"footer">) {
  return (
    <footer
      className={cn(
        "mt-auto flex flex-wrap items-center justify-between gap-3 border-t border-[color:var(--color-divider)] pt-4",
        className
      )}
      {...props}
    />
  );
}

export { Panel, PanelBody, PanelDescription, PanelFooter, PanelHeader, PanelTitle };
