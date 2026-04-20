import type { HTMLAttributes } from "react";
import { Link, useMatchRoute } from "@tanstack/react-router";

import { cn } from "@agh/ui";

export interface AppHeaderProps extends HTMLAttributes<HTMLElement> {}

function AppHeader({ className, ...props }: AppHeaderProps) {
  const matchRoute = useMatchRoute();
  const dashboardActive = Boolean(matchRoute({ to: "/" }));

  return (
    <header
      data-testid="app-header"
      className={cn(
        "sticky top-0 z-40 flex h-14 shrink-0 items-center gap-5 border-b border-[color:var(--color-divider)] bg-[rgba(20,19,18,0.92)] px-5 backdrop-blur-xl",
        className
      )}
      {...props}
    >
      <Link
        to="/"
        data-testid="app-header-home"
        aria-label="Go to dashboard"
        className="flex items-center gap-2 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]"
      >
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
          ALPHA
        </span>
      </Link>
      <nav
        data-testid="app-header-nav"
        aria-label="Primary"
        className="flex flex-1 items-center gap-1 text-[13px] text-[color:var(--color-text-tertiary)]"
      >
        <Link
          to="/"
          data-testid="app-header-nav-dashboard"
          data-active={dashboardActive}
          className={cn(
            "inline-flex items-center rounded-md px-3 py-1.5 transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
            dashboardActive
              ? "bg-[color:var(--color-surface)] text-[color:var(--color-text-primary)]"
              : "text-[color:var(--color-text-tertiary)]"
          )}
        >
          Dashboard
        </Link>
      </nav>
    </header>
  );
}

export { AppHeader };
