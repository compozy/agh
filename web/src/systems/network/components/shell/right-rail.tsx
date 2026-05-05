import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type RightRailMode = "thread" | "members" | "work";

export interface RightRailProps {
  open: boolean;
  mode: RightRailMode;
  children?: ReactNode;
  className?: string;
}

export function RightRail({ open, mode, children, className }: RightRailProps) {
  if (!open) {
    return null;
  }

  return (
    <aside
      aria-label={
        mode === "thread"
          ? "Thread overlay"
          : mode === "members"
            ? "Channel members"
            : "Work inspector"
      }
      className={cn(
        "flex min-h-0 w-[420px] shrink-0 flex-col border-l border-[color:var(--color-divider)] bg-[color:var(--color-canvas-deep)]",
        className
      )}
      data-mode={mode}
      data-testid="network-right-rail"
    >
      {children}
    </aside>
  );
}
