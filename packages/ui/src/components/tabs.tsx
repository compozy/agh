"use client";

import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
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

function TabsList({ className, ...props }: TabsPrimitive.List.Props) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      className={cn(
        "group/tabs-list inline-flex w-fit items-center justify-center gap-0 rounded-none border-b border-line bg-transparent p-0 text-muted group-data-horizontal/tabs:h-[26px] group-data-vertical/tabs:h-fit group-data-vertical/tabs:flex-col",
        className
      )}
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
        "relative inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-sm border border-transparent bg-transparent px-2 py-0.5 text-[12.5px] font-medium tracking-[-0.006em] whitespace-nowrap text-muted transition-colors duration-base ease-out group-data-vertical/tabs:w-full group-data-vertical/tabs:justify-start hover:text-fg-strong focus-visible:outline-none focus-visible:shadow-[0_0_0_1px_var(--line-strong)] disabled:pointer-events-none disabled:opacity-50 has-data-[icon=inline-end]:pr-1 has-data-[icon=inline-start]:pl-1 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
        "data-active:bg-transparent data-active:font-[510] data-active:text-fg-strong data-active:shadow-none",
        "not-first:before:content-['·'] not-first:before:px-1.5 not-first:before:text-faint not-first:before:opacity-50 not-first:before:select-none",
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
          className="inline-flex items-center justify-center bg-transparent px-0 font-mono text-[10.5px] font-medium tracking-[0] tabular-nums text-faint group-data-[active=true]:text-muted"
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

export { Tabs, TabsContent, TabsList, TabsTrigger };
