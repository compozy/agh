"use client";

import * as React from "react";

import { cn } from "../../lib/utils";
import { PageContent, type PageContentDensity } from "./page-content";

type PageShellDensity = Exclude<PageContentDensity, "listing">;

interface PageShellProps extends Omit<React.ComponentProps<"div">, "title"> {
  slug?: string;
  banner?: React.ReactNode;
  /** Route identity row (PageHead + status summary) rendered as the first body child. */
  head?: React.ReactNode;
  footer?: React.ReactNode;
  bodyClassName?: string;
  density?: PageShellDensity;
}

/**
 * PageShell hosts a route's body, banner, and sticky footer. The topbar owns
 * only location chrome (breadcrumb · route nav · actions); route identity
 * lives in the `head` slot (`PageHead`) inside content (route chrome §08).
 * Horizontal gutters match ListingPage via shared `PageContent`.
 */
function PageShell({
  slug,
  banner,
  head,
  footer,
  bodyClassName,
  className,
  children,
  density = "comfortable",
  ...props
}: PageShellProps) {
  const testId = slug ? `settings-page-${slug}` : undefined;
  const bodyTestId = slug ? `settings-page-${slug}-body` : undefined;
  const footerTestId = slug ? `settings-page-${slug}-footer` : undefined;
  const bannerTestId = slug ? `settings-page-${slug}-banner-slot` : undefined;
  const headTestId = slug ? `settings-page-${slug}-head-slot` : undefined;

  return (
    <div
      data-density={density}
      className={cn("flex min-h-0 flex-1 flex-col overflow-hidden", className)}
      data-testid={testId}
      {...props}
    >
      {banner ? <div data-testid={bannerTestId}>{banner}</div> : null}

      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto" data-testid={bodyTestId}>
        <PageContent density={density} className={bodyClassName}>
          {head ? <div data-testid={headTestId}>{head}</div> : null}
          {children}
        </PageContent>
      </div>

      {footer ? (
        <div className="border-t border-line" data-testid={footerTestId}>
          {footer}
        </div>
      ) : null}
    </div>
  );
}

export { PageShell };
export type { PageShellProps, PageShellDensity };
