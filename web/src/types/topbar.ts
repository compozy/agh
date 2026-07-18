import type { LinkProps } from "@tanstack/react-router";

/**
 * One breadcrumb level for the topbar context zone. Every TanStack Router
 * route's `beforeLoad` declares its own crumb; the shell collects the active
 * match chain into the trail (route chrome contract §05).
 */
export interface TopbarCrumbContext {
  /** Crumb label for this route level (static or params-derived). */
  label: string;
  /**
   * Router path for non-leaf levels. Leaf crumbs render as the current page
   * (`aria-current="page"`), so they omit it.
   */
  to?: LinkProps["to"];
  /** Route params for parameterized non-leaf levels (e.g. `/loops/$name`). */
  params?: LinkProps["params"];
  /** Search params for non-leaf levels that scope by query (e.g. marketplace kind). */
  search?: LinkProps["search"];
}

/**
 * Topbar metadata declared by every route's `beforeLoad`. The topbar answers
 * "where?" only — identity (icon well, H1, count, meta) lives in the content
 * `PageHead`, and body chrome owns search/filters/scopes.
 */
export interface TopbarRouteContext {
  crumb: TopbarCrumbContext;
  /**
   * Extra trail level for path segments without their own route file
   * (e.g. the marketplace kind between Marketplace and an entry).
   */
  parentCrumb?: TopbarCrumbContext;
}
