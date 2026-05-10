"use client";

import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../lib/utils";

function Tabs({ className, orientation = "horizontal", ...props }: TabsPrimitive.Root.Props) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      data-orientation={orientation}
      orientation={orientation}
      className={cn("group/tabs flex gap-2 data-horizontal:flex-col", className)}
      {...props}
    />
  );
}

const tabsListVariants = cva(
  "group/tabs-list inline-flex w-fit items-center justify-center rounded-md p-[3px] text-(--muted) group-data-horizontal/tabs:h-[26px] group-data-vertical/tabs:h-fit group-data-vertical/tabs:flex-col data-[variant=line]:rounded-none",
  {
    variants: {
      variant: {
        default: "bg-(--canvas-soft) border border-(--line)",
        line: "gap-1 bg-transparent border-b border-(--line)",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

function TabsList({
  className,
  variant = "default",
  ...props
}: TabsPrimitive.List.Props & VariantProps<typeof tabsListVariants>) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      data-variant={variant}
      className={cn(tabsListVariants({ variant }), className)}
      {...props}
    />
  );
}

export interface TabsTriggerProps extends TabsPrimitive.Tab.Props {
  count?: number;
  liveLabel?: React.ReactNode;
}

function TabsTrigger({ className, children, count, liveLabel, ...props }: TabsTriggerProps) {
  return (
    <TabsPrimitive.Tab
      data-slot="tabs-trigger"
      className={cn(
        "relative inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-sm border border-transparent px-2 py-0.5 text-[13px] font-[510] whitespace-nowrap text-(--muted) transition-all group-data-vertical/tabs:w-full group-data-vertical/tabs:justify-start hover:text-(--fg) focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] disabled:pointer-events-none disabled:opacity-50 has-data-[icon=inline-end]:pr-1 has-data-[icon=inline-start]:pl-1 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        "group-data-[variant=line]/tabs-list:bg-transparent group-data-[variant=line]/tabs-list:data-active:bg-transparent",
        "data-active:bg-(--elevated) data-active:text-(--fg-strong) data-active:shadow-[var(--highlight)]",
        "group-data-[variant=line]/tabs-list:data-active:bg-transparent group-data-[variant=line]/tabs-list:data-active:shadow-none",
        "after:absolute after:bg-(--accent) after:opacity-0 after:transition-opacity group-data-horizontal/tabs:after:inset-x-2 group-data-horizontal/tabs:after:bottom-[-2px] group-data-horizontal/tabs:after:h-[2px] group-data-vertical/tabs:after:inset-y-0 group-data-vertical/tabs:after:-right-1 group-data-vertical/tabs:after:w-[2px] group-data-[variant=line]/tabs-list:data-active:after:opacity-100",
        className
      )}
      {...props}
    >
      <span data-slot="tabs-trigger-label" className="inline-flex min-w-0 items-center">
        {children}
      </span>
      {typeof count === "number" ? (
        <span
          data-slot="tabs-trigger-count"
          className="inline-flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-(--canvas-tint) px-(--space-pill-group-badge-x) font-mono text-(--text-pill-group-badge) font-medium text-(--muted) group-data-[active=true]:bg-(--accent) group-data-[active=true]:text-(--accent-ink)"
        >
          {count}
        </span>
      ) : null}
      {liveLabel ? (
        <span
          aria-live="polite"
          data-slot="tabs-trigger-live"
          className="inline-flex h-4 items-center gap-1 rounded-sm bg-(--accent-tint) px-1.5 font-mono text-[9px] uppercase tracking-(--tracking-mono) text-(--accent)"
        >
          <span aria-hidden="true" className="size-1.5 rounded-full bg-(--accent)" />
          {liveLabel}
        </span>
      ) : null}
    </TabsPrimitive.Tab>
  );
}

function TabsContent({ className, ...props }: TabsPrimitive.Panel.Props) {
  return (
    <TabsPrimitive.Panel
      data-slot="tabs-content"
      className={cn("flex-1 text-[13px] outline-none", className)}
      {...props}
    />
  );
}

export { Tabs, TabsList, TabsTrigger, TabsContent, tabsListVariants };
