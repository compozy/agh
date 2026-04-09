import { Search } from "lucide-react";
import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

function Toolbar({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "ds-panel ds-panel-surface flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between",
        className
      )}
      {...props}
    />
  );
}

function ToolbarGroup({ className, ...props }: ComponentProps<"div">) {
  return (
    <div className={cn("flex flex-wrap items-center gap-2 md:gap-2.5", className)} {...props} />
  );
}

function ToolbarSearch({ className, ...props }: ComponentProps<"input">) {
  return (
    <label className={cn("ds-toolbar-field min-w-0 flex-1 sm:max-w-xs", className)}>
      <Search className="size-4 text-[color:var(--color-text-label)]" />
      <span className="sr-only">Search design system preview</span>
      <input
        className="min-w-0 flex-1 bg-transparent text-sm text-[color:var(--color-text-primary)] outline-none placeholder:text-[color:var(--color-text-tertiary)]"
        {...props}
      />
    </label>
  );
}

function ToolbarAction({ className, ...props }: ComponentProps<"button">) {
  return (
    <button
      className={cn(
        "inline-flex h-10 items-center justify-center rounded-full border border-[color:var(--color-accent)] bg-[color:var(--color-accent)] px-4 font-medium text-[color:var(--color-canvas)] transition-transform duration-200 hover:-translate-y-px focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
        className
      )}
      type="button"
      {...props}
    />
  );
}

export { Toolbar, ToolbarAction, ToolbarGroup, ToolbarSearch };
