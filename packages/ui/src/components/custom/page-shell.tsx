"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { PageHeader } from "./page-header";

interface PageShellProps extends Omit<React.ComponentProps<"div">, "title"> {
  slug?: string;
  title: React.ReactNode;
  eyebrow?: React.ReactNode;
  breadcrumb?: React.ReactNode;
  subtitle?: React.ReactNode;
  statusLine?: React.ReactNode;
  statusRow?: React.ReactNode;
  actions?: React.ReactNode;
  banner?: React.ReactNode;
  footer?: React.ReactNode;
  bodyClassName?: string;
}

function PageShell({
  slug,
  title,
  eyebrow = "Settings",
  breadcrumb,
  subtitle,
  statusLine,
  statusRow,
  actions,
  banner,
  footer,
  bodyClassName,
  className,
  children,
  ...props
}: PageShellProps) {
  const testId = slug ? `settings-page-${slug}` : undefined;
  const headerTestId = slug ? `settings-page-${slug}-header` : undefined;
  const bodyTestId = slug ? `settings-page-${slug}-body` : undefined;
  const footerTestId = slug ? `settings-page-${slug}-footer` : undefined;
  const bannerTestId = slug ? `settings-page-${slug}-banner-slot` : undefined;
  const eyebrowTestId = slug ? `settings-page-${slug}-eyebrow` : undefined;
  const statusTestId = slug ? `settings-page-${slug}-status` : undefined;
  const actionsTestId = slug ? `settings-page-${slug}-actions` : undefined;
  const headerBreadcrumb =
    breadcrumb ??
    (eyebrow ? (
      <span data-testid={eyebrowTestId} className="tracking-mono">
        {eyebrow} / {title}
      </span>
    ) : undefined);
  const headerStatus = statusRow ?? statusLine;

  return (
    <div
      className={cn("flex min-h-0 flex-1 flex-col overflow-hidden", className)}
      data-testid={testId}
      {...props}
    >
      <PageHeader
        title={title}
        breadcrumb={headerBreadcrumb}
        subtitle={subtitle}
        statusRow={headerStatus ? <div data-testid={statusTestId}>{headerStatus}</div> : undefined}
        meta={actions ? <div data-testid={actionsTestId}>{actions}</div> : undefined}
        className="px-4 py-5 sm:px-6 md:px-8 md:py-6 xl:px-10"
        data-testid={headerTestId}
      />

      {banner ? <div data-testid={bannerTestId}>{banner}</div> : null}

      <div
        className={cn(
          "flex min-h-0 flex-1 flex-col overflow-y-auto px-4 py-5 sm:px-6 md:px-8 md:py-6 xl:px-10",
          bodyClassName
        )}
        data-testid={bodyTestId}
      >
        <div className="flex min-h-full flex-col gap-6 pb-12 md:gap-8 md:pb-16">{children}</div>
      </div>

      {footer ? (
        <div className="border-t border-[color:var(--color-divider)]" data-testid={footerTestId}>
          {footer}
        </div>
      ) : null}
    </div>
  );
}

export { PageShell };
export type { PageShellProps };
