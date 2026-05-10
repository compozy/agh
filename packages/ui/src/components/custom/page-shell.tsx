"use client";

import * as React from "react";

import { cn } from "../../lib/utils";

type PageShellDensity = "comfortable" | "compact";

interface PageShellProps extends Omit<React.ComponentProps<"div">, "title"> {
  slug?: string;
  banner?: React.ReactNode;
  footer?: React.ReactNode;
  bodyClassName?: string;
  density?: PageShellDensity;
}

/**
 * PageShell hosts a route's body, banner, and sticky footer. After P4 the
 * shell-level `<Topbar>` owns the route title and chrome, so PageShell does
 * not render its own header. Routes push title/icon/count via TanStack Router
 * `beforeLoad` and dynamic tabs/search/actions via `useTopbarSlot`.
 */
function PageShell({
  slug,
  banner,
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
  const bodyPadding =
    density === "compact"
      ? "px-4 py-3 sm:px-5 md:px-6"
      : "px-4 py-5 sm:px-6 md:px-8 md:py-6 xl:px-10";
  const bodyGap =
    density === "compact" ? "gap-4 pb-8 md:gap-5 md:pb-10" : "gap-6 pb-12 md:gap-8 md:pb-16";

  return (
    <div
      data-density={density}
      className={cn("flex min-h-0 flex-1 flex-col overflow-hidden", className)}
      data-testid={testId}
      {...props}
    >
      {banner ? <div data-testid={bannerTestId}>{banner}</div> : null}

      <div
        className={cn("flex min-h-0 flex-1 flex-col overflow-y-auto", bodyPadding, bodyClassName)}
        data-testid={bodyTestId}
      >
        <div className={cn("flex min-h-full flex-col", bodyGap)}>{children}</div>
      </div>

      {footer ? (
        <div className="border-t border-(--line)" data-testid={footerTestId}>
          {footer}
        </div>
      ) : null}
    </div>
  );
}

export { PageShell };
export type { PageShellProps, PageShellDensity };
