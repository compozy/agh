import type { ReactNode } from "react";

import { cn } from "@agh/ui";

interface SettingsPageShellProps {
  slug: string;
  title: string;
  eyebrow?: string;
  statusLine?: ReactNode;
  actions?: ReactNode;
  banner?: ReactNode;
  footer?: ReactNode;
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
  footer,
  children,
  className,
}: SettingsPageShellProps) {
  return (
    <div
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
      data-testid={`settings-page-${slug}`}
    >
      <header
        className="flex flex-col gap-4 border-b border-(--color-divider) px-4 py-5 sm:px-6 md:px-8 md:py-6 xl:px-10"
        data-testid={`settings-page-${slug}-header`}
      >
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <span
              className="font-mono text-eyebrow font-semibold uppercase tracking-mono text-(--color-text-label)"
              data-testid={`settings-page-${slug}-eyebrow`}
            >
              {eyebrow} / {title}
            </span>
            <h1 className="text-2xl font-semibold tracking-tight text-(--color-text-primary)">
              {title}
            </h1>
          </div>
          {actions ? (
            <div
              className="flex shrink-0 flex-wrap items-center gap-2"
              data-testid={`settings-page-${slug}-actions`}
            >
              {actions}
            </div>
          ) : null}
        </div>
        {statusLine ? (
          <div
            className="flex flex-wrap items-center gap-x-4 gap-y-2 text-small-body text-(--color-text-secondary)"
            data-testid={`settings-page-${slug}-status`}
          >
            {statusLine}
          </div>
        ) : null}
      </header>

      {banner ? <div data-testid={`settings-page-${slug}-banner-slot`}>{banner}</div> : null}

      <div
        className={cn(
          "flex min-h-0 flex-1 flex-col overflow-y-auto px-4 py-5 sm:px-6 md:px-8 md:py-6 xl:px-10",
          className
        )}
        data-testid={`settings-page-${slug}-body`}
      >
        <div className="flex min-h-full flex-col gap-6 pb-12 md:gap-8 md:pb-16">{children}</div>
      </div>

      {footer ? (
        <div
          className="border-t border-(--color-divider)"
          data-testid={`settings-page-${slug}-footer`}
        >
          {footer}
        </div>
      ) : null}
    </div>
  );
}

export { SettingsPageShell };
