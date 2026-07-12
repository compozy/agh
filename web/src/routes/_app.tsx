import { createFileRoute } from "@tanstack/react-router";

import { AppRouteErrorBoundary, AppRouteNotFoundBoundary } from "./_app/-app-route-boundaries";
import { AppLayout } from "./_app/-app-shell";
import { preloadAppRoute } from "./_app/-app-preload";

export const Route = createFileRoute("/_app")({
  loader: ({ context }) => preloadAppRoute(context.queryClient),
  component: AppLayout,
  errorComponent: AppRouteErrorBoundary,
  notFoundComponent: AppRouteNotFoundBoundary,
});
