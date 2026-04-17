import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

interface SettingsPageShellProps {
  slug: string;
  title: string;
  eyebrow?: string;
  statusLine?: ReactNode;
  actions?: ReactNode;
  banner?: ReactNode;
  children: ReactNode;
  className?: string;
}

function SettingsPageShell({
  slug,
  title,
  eyebrow = "Settings",
  statusLine,
  actions,
  banner,
  children,
  className,
}: SettingsPageShellProps) {
  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid={`settings-page-${slug}`}
    >
      <header
        className="flex flex-col gap-3 border-b border-[color:var(--color-divider)] px-8 py-6"
        data-testid={`settings-page-${slug}-header`}
      >
        <div className="flex items-start justify-between gap-6">
          <div className="flex flex-col gap-1">
            <p className="font-mono text-[0.6rem] uppercase tracking-[0.22em] text-[color:var(--color-text-label)]">
              {eyebrow} / {title}
            </p>
            <h1 className="text-2xl font-semibold tracking-[-0.01em] text-[color:var(--color-text-primary)]">
              {title}
            </h1>
          </div>
          {actions ? (
            <div
              className="flex shrink-0 items-center gap-2"
              data-testid={`settings-page-${slug}-actions`}
            >
              {actions}
            </div>
          ) : null}
        </div>
        {statusLine ? (
          <div
            className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-[color:var(--color-text-secondary)]"
            data-testid={`settings-page-${slug}-status`}
          >
            {statusLine}
          </div>
        ) : null}
      </header>

      {banner ? <div data-testid={`settings-page-${slug}-banner-slot`}>{banner}</div> : null}

      <div
        className={cn("flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-8 py-6", className)}
        data-testid={`settings-page-${slug}-body`}
      >
        {children}
      </div>
    </div>
  );
}

export { SettingsPageShell };
