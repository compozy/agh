import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type RightRailMode = "thread" | "inspector";

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
      aria-label={mode === "thread" ? "Thread overlay" : "Channel inspector"}
      className={cn(
        "flex min-h-0 w-[360px] shrink-0 flex-col border-l border-(--color-divider) bg-(--color-canvas-deep)",
        className
      )}
      data-mode={mode}
      data-testid="network-right-rail"
    >
      {children}
    </aside>
  );
}
