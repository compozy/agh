import { useEffect } from "react";
import { Home } from "lucide-react";
import { Link, useRouter, useRouterState } from "@tanstack/react-router";

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
  Topbar,
  TopbarSlotProvider,
  useTopbarSlotValue,
} from "@agh/ui";
import * as React from "react";

import { useTopbarShellModel } from "@/hooks/routes/use-topbar-shell-model";
import type { TopbarCrumbContext } from "@/types/topbar";

interface TopbarShellProps {
  children: React.ReactNode;
}

/**
 * Mounts the shell-level `<Topbar>` once for the entire `_app` outlet.
 *
 * Behavior:
 * - Always leads the breadcrumb with a fixed Home icon that links to the
 *   dashboard (`/`); on `/` the icon is the current page (no duplicate label).
 * - Collects route-level `topbar.crumb` declarations into ancestry and a
 *   stable shell H1. Loader-derived names override that H1 via the slot.
 * - Hosts `<TopbarSlotProvider>` so any descendant route can push
 *   routeNav/actions/overflow into the topbar zones.
 * - Subscribes to `router.subscribe("onResolved")` to move focus to the
 *   always-mounted Topbar H1 after path navigation.
 */
export function TopbarShell({ children }: TopbarShellProps) {
  return (
    <TopbarSlotProvider>
      <TopbarShellInner>{children}</TopbarShellInner>
    </TopbarSlotProvider>
  );
}

function TopbarShellInner({ children }: TopbarShellProps) {
  const titleRef = React.useRef<HTMLHeadingElement | null>(null);
  const router = useRouter();
  const { crumbs, currentTitle } = useTopbarShellModel();
  const slot = useTopbarSlotValue();

  useEffect(() => {
    const unsubscribe = router.subscribe("onResolved", event => {
      if (!event.pathChanged) {
        return;
      }
      const title = titleRef.current;
      if (!title) {
        return;
      }
      try {
        title.focus({ preventScroll: true });
      } catch {
        title.focus();
      }
    });
    return unsubscribe;
  }, [router]);

  return (
    <>
      <Topbar
        breadcrumb={<ShellBreadcrumb crumbs={crumbs} />}
        title={slot?.crumb ?? currentTitle}
        titleRef={titleRef}
      />
      {children}
    </>
  );
}

interface ShellBreadcrumbProps {
  crumbs: ReadonlyArray<TopbarCrumbContext>;
}

function isDashboardCrumb(crumb: TopbarCrumbContext): boolean {
  return crumb.to === "/";
}

function ShellBreadcrumb({ crumbs }: ShellBreadcrumbProps) {
  const pathname = useRouterState({ select: state => state.location.pathname });
  const isHome = pathname === "/";
  const trail = crumbs.filter(crumb => !isDashboardCrumb(crumb));
  const ancestors = trail.slice(0, -1);

  return (
    <Breadcrumb aria-label="Breadcrumb" className="min-w-0 overflow-hidden">
      <BreadcrumbList className="flex-nowrap overflow-hidden whitespace-nowrap">
        <BreadcrumbItem>
          {isHome && trail.length === 0 ? (
            <BreadcrumbPage
              aria-label="Dashboard"
              className="inline-flex items-center px-1.5 py-0.5"
              data-testid="topbar-breadcrumb-home"
            >
              <Home aria-hidden className="size-3.5" />
            </BreadcrumbPage>
          ) : (
            <BreadcrumbLink
              aria-label="Dashboard"
              className="inline-flex items-center"
              data-testid="topbar-breadcrumb-home"
              render={<Link to="/" />}
            >
              <Home aria-hidden className="size-3.5" />
            </BreadcrumbLink>
          )}
        </BreadcrumbItem>
        {!isHome && trail.length > 0 ? <BreadcrumbSeparator /> : null}
        {ancestors.map(crumb => {
          return (
            <React.Fragment key={`${crumb.to ?? "current"}:${crumb.label}`}>
              <BreadcrumbItem>
                {crumb.to ? (
                  <BreadcrumbLink
                    render={<Link params={crumb.params} search={crumb.search} to={crumb.to} />}
                  >
                    {crumb.label}
                  </BreadcrumbLink>
                ) : (
                  <span className="px-1.5 py-0.5">{crumb.label}</span>
                )}
              </BreadcrumbItem>
              <BreadcrumbSeparator />
            </React.Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
