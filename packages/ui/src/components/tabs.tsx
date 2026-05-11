"use client";

import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

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
  "group/tabs-list inline-flex w-fit items-center justify-center rounded-none p-0 text-muted group-data-horizontal/tabs:h-[26px] group-data-vertical/tabs:h-fit group-data-vertical/tabs:flex-col",
  {
    variants: {
      variant: {
        line: "gap-1 bg-transparent border-b border-line",
        lane: "gap-0 bg-transparent border-b border-line",
      },
    },
    defaultVariants: {
      variant: "line",
    },
  }
);

export type TabsVariant = NonNullable<VariantProps<typeof tabsListVariants>["variant"]>;

function TabsList({
  className,
  variant = "line",
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
        "relative inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-sm border border-transparent px-2 py-0.5 text-[13px] font-medium whitespace-nowrap text-muted transition-colors duration-base ease-out group-data-vertical/tabs:w-full group-data-vertical/tabs:justify-start hover:text-fg focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] disabled:pointer-events-none disabled:opacity-50 has-data-[icon=inline-end]:pr-1 has-data-[icon=inline-start]:pl-1 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        "group-data-[variant=line]/tabs-list:bg-transparent group-data-[variant=line]/tabs-list:hover:text-fg",
        "group-data-[variant=line]/tabs-list:data-active:bg-transparent group-data-[variant=line]/tabs-list:data-active:text-fg-strong group-data-[variant=line]/tabs-list:data-active:shadow-none",
        "group-data-[variant=lane]/tabs-list:bg-transparent group-data-[variant=lane]/tabs-list:gap-1.5 group-data-[variant=lane]/tabs-list:tracking-[-0.006em] group-data-[variant=lane]/tabs-list:text-[12.5px] group-data-[variant=lane]/tabs-list:font-medium group-data-[variant=lane]/tabs-list:hover:text-fg-strong",
        "group-data-[variant=lane]/tabs-list:data-active:bg-transparent group-data-[variant=lane]/tabs-list:data-active:text-fg-strong group-data-[variant=lane]/tabs-list:data-active:font-[510] group-data-[variant=lane]/tabs-list:data-active:shadow-none",
        "group-data-[variant=lane]/tabs-list:not-first:before:content-['·'] group-data-[variant=lane]/tabs-list:not-first:before:text-faint group-data-[variant=lane]/tabs-list:not-first:before:opacity-50 group-data-[variant=lane]/tabs-list:not-first:before:px-1.5 group-data-[variant=lane]/tabs-list:not-first:before:select-none",
        "after:absolute after:bg-fg-strong after:opacity-0 after:transition-opacity group-data-horizontal/tabs:after:inset-x-2 group-data-horizontal/tabs:after:bottom-[-1.5px] group-data-horizontal/tabs:after:h-[1.5px] group-data-vertical/tabs:after:inset-y-0 group-data-vertical/tabs:after:-right-1 group-data-vertical/tabs:after:w-[1.5px] data-active:after:opacity-100",
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
          className={cn(
            "inline-flex items-center justify-center font-mono tabular-nums",
            "group-data-[variant=line]/tabs-list:h-3.5 group-data-[variant=line]/tabs-list:min-w-3.5 group-data-[variant=line]/tabs-list:rounded-full group-data-[variant=line]/tabs-list:bg-canvas-tint group-data-[variant=line]/tabs-list:px-(--space-pill-group-badge-x) group-data-[variant=line]/tabs-list:text-pill-group-badge group-data-[variant=line]/tabs-list:font-medium group-data-[variant=line]/tabs-list:text-muted",
            "group-data-[variant=line]/tabs-list:group-data-[active=true]:bg-btn-default-hover group-data-[variant=line]/tabs-list:group-data-[active=true]:text-fg",
            "group-data-[variant=lane]/tabs-list:bg-transparent group-data-[variant=lane]/tabs-list:px-0 group-data-[variant=lane]/tabs-list:text-[10.5px] group-data-[variant=lane]/tabs-list:font-medium group-data-[variant=lane]/tabs-list:tracking-[0] group-data-[variant=lane]/tabs-list:text-faint",
            "group-data-[variant=lane]/tabs-list:group-data-[active=true]:text-muted"
          )}
        >
          {count}
        </span>
      ) : null}
      {liveLabel ? (
        <span
          aria-live="polite"
          data-slot="tabs-trigger-live"
          className="eyebrow inline-flex h-4 items-center gap-1 rounded-sm bg-accent-tint px-1.5 text-accent"
        >
          <span aria-hidden="true" className="size-1.5 rounded-full bg-accent" />
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

export { Tabs, TabsContent, TabsList, tabsListVariants, TabsTrigger };
