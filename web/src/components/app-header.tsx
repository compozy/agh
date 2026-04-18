import type { HTMLAttributes } from "react";

import { cn } from "@agh/ui";

export interface AppHeaderProps extends HTMLAttributes<HTMLElement> {}

function AppHeader({ className, ...props }: AppHeaderProps) {
  return (
    <header
      data-testid="app-header"
      className={cn(
        "sticky top-0 z-40 flex h-14 shrink-0 items-center gap-5 border-b border-[color:var(--color-divider)] bg-[rgba(20,19,18,0.92)] px-5 backdrop-blur-xl",
        className
      )}
      {...props}
    >
      <div className="flex items-center gap-2">
        <span
          data-testid="app-header-wordmark"
          className="font-wordmark text-[22px] leading-none tracking-[-0.02em] text-[color:var(--color-text-primary)]"
        >
          agh
        </span>
        <span
          data-testid="app-header-alpha-chip"
          className="inline-flex h-[18px] items-center rounded-[3px] border border-[color:var(--color-divider)] px-1.5 font-mono text-[10px] font-medium uppercase tracking-[0.14em] text-[color:var(--color-text-label)]"
        >
          Alpha
        </span>
      </div>
      <nav
        data-testid="app-header-nav"
        aria-label="Primary"
        className="flex flex-1 items-center gap-1 text-[13px] text-[color:var(--color-text-tertiary)]"
      />
    </header>
  );
}

export { AppHeader };
