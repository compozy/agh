import type { ReactNode } from "react";

type RouteOptionKey = "component" | "errorComponent" | "notFoundComponent" | "beforeLoad";

type RouteLike = {
  component?: unknown;
  errorComponent?: unknown;
  notFoundComponent?: unknown;
  beforeLoad?: unknown;
  options?: {
    component?: unknown;
    errorComponent?: unknown;
    notFoundComponent?: unknown;
    beforeLoad?: unknown;
  };
};

function routeOption(route: RouteLike, key: RouteOptionKey): unknown {
  return route.options?.[key] ?? route[key];
}

function requireRouteFunction(route: RouteLike, key: RouteOptionKey, label: string): unknown {
  const value = routeOption(route, key);
  if (typeof value !== "function") throw new Error(`route ${label} is not registered`);
  return value;
}

export function routeComponent(route: RouteLike): () => ReactNode {
  return requireRouteFunction(route, "component", "component") as () => ReactNode;
}

export function routeErrorComponent<P>(route: RouteLike): (props: P) => ReactNode {
  return requireRouteFunction(route, "errorComponent", "error component") as (
    props: P
  ) => ReactNode;
}

export function routeNotFoundComponent<P>(route: RouteLike): (props: P) => ReactNode {
  return requireRouteFunction(route, "notFoundComponent", "not-found component") as (
    props: P
  ) => ReactNode;
}

export function routeBeforeLoad(route: RouteLike): () => unknown {
  return requireRouteFunction(route, "beforeLoad", "beforeLoad") as () => unknown;
}
