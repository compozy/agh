import type { ReactNode } from "react";

import { Sidebar } from "@agh/ui";

import { cn } from "@/lib/utils";

interface StoryFrameProps {
  children: ReactNode;
  className?: string;
}

export function StorySurface({ children, className }: StoryFrameProps) {
  return (
    <div
      className={cn(
        "min-h-[640px] bg-background p-6 text-[color:var(--color-text-primary)]",
        className
      )}
    >
      {children}
    </div>
  );
}

export function CenteredSurface({ children, className }: StoryFrameProps) {
  return (
    <StorySurface className={cn("flex items-center justify-center", className)}>
      {children}
    </StorySurface>
  );
}

export function PanelSurface({ children, className }: StoryFrameProps) {
  return (
    <StorySurface
      className={cn(
        "flex min-h-[640px] overflow-hidden rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-0",
        className
      )}
    >
      {children}
    </StorySurface>
  );
}

export function SidebarSurface({ children, className }: StoryFrameProps) {
  return (
    <StorySurface className={cn("max-w-sm p-0", className)}>
      <div className="h-[420px]">
        <Sidebar nav={children} />
      </div>
    </StorySurface>
  );
}
