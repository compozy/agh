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
 * - Collects every route level's `topbar.crumb` declaration into the trailing
 *   trail; the deepest crumb renders as the current page and can be overridden
 *   live via `useTopbarSlot({ crumb })` for loader-derived names.
 * - Hosts `<TopbarSlotProvider>` so any descendant route can push
 *   routeNav/actions/overflow into the topbar zones.
 * - Subscribes to `router.subscribe("onResolved")` to move focus to the
 *   content `PageHead` H1 after path navigation (route chrome §08).
 */
export function TopbarShell({ children }: TopbarShellProps) {
  return (
    <TopbarSlotProvider>
      <TopbarShellInner>{children}</TopbarShellInner>
    </TopbarSlotProvider>
  );
}

const FOCUS_RETRY_FRAMES = 5;

function focusPageTitle(attempt = 0) {
  const node = document.querySelector<HTMLElement>("#app-content [data-slot='page-head-title']");
  if (node) {
    try {
      node.focus({ preventScroll: true });
    } catch {
      node.focus();
    }
    return;
  }
  // The destination outlet may not be committed yet when onResolved fires;
  // retry across a few frames instead of observing the whole subtree.
  if (attempt < FOCUS_RETRY_FRAMES) {
    requestAnimationFrame(() => focusPageTitle(attempt + 1));
  }
}

function TopbarShellInner({ children }: TopbarShellProps) {
  const router = useRouter();
  const { crumbs } = useTopbarShellModel();
  const slot = useTopbarSlotValue();

  useEffect(() => {
    const unsubscribe = router.subscribe("onResolved", event => {
      if (!event.pathChanged) {
        return;
      }
      focusPageTitle();
    });
    return unsubscribe;
  }, [router]);

  return (
    <>
      <Topbar breadcrumb={<ShellBreadcrumb crumbs={crumbs} leafOverride={slot?.crumb} />} />
      {children}
    </>
  );
}

interface ShellBreadcrumbProps {
  crumbs: ReadonlyArray<TopbarCrumbContext>;
  leafOverride?: React.ReactNode;
}

function isDashboardCrumb(crumb: TopbarCrumbContext): boolean {
  return crumb.to === "/";
}

function ShellBreadcrumb({ crumbs, leafOverride }: ShellBreadcrumbProps) {
  const pathname = useRouterState({ select: state => state.location.pathname });
  const isHome = pathname === "/";
  const trail = crumbs.filter(crumb => !isDashboardCrumb(crumb));
  const leafIndex = trail.length - 1;

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
        {trail.map((crumb, index) => {
          const isLeaf = index === leafIndex;
          return (
            <React.Fragment key={`${crumb.to ?? "current"}:${crumb.label}`}>
              <BreadcrumbSeparator />
              <BreadcrumbItem className={isLeaf ? "min-w-0" : undefined}>
                {isLeaf ? (
                  <BreadcrumbPage className="min-w-0 truncate" data-testid="topbar-breadcrumb-page">
                    {leafOverride ?? crumb.label}
                  </BreadcrumbPage>
                ) : crumb.to ? (
                  <BreadcrumbLink
                    render={<Link params={crumb.params} search={crumb.search} to={crumb.to} />}
                  >
                    {crumb.label}
                  </BreadcrumbLink>
                ) : (
                  <span className="px-1.5 py-0.5">{crumb.label}</span>
                )}
              </BreadcrumbItem>
            </React.Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
}
