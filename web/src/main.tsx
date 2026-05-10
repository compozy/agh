import { RouterProvider, createRouter } from "@tanstack/react-router";
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";

import { Toaster, TooltipProvider, UIProvider } from "@agh/ui";

import type { TopbarRouteContext } from "@/types/topbar";
import { routeTree } from "./routeTree.gen";

import * as TanStackQueryProvider from "./integrations/tanstack-query/root-provider";

import "./styles.css";

const TanStackQueryProviderContext = TanStackQueryProvider.getContext();
const router = createRouter({
  routeTree,
  context: {
    ...TanStackQueryProviderContext,
  },
  defaultPreload: "intent",
  defaultViewTransition: true,
  scrollRestoration: true,
  defaultStructuralSharing: true,
  defaultPreloadStaleTime: 0,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }

  interface RouteContext {
    /**
     * Static topbar metadata declared by every TanStack Router route's
     * `beforeLoad`. Read by the shell `<Topbar>` via `useRouterState` to
     * resolve the deepest match's title/icon/count.
     */
    topbar?: TopbarRouteContext;
  }
}

const rootElement = document.getElementById("app");
if (rootElement && !rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement);
  root.render(
    <StrictMode>
      <UIProvider reducedMotion="user">
        <TooltipProvider>
          <TanStackQueryProvider.Provider {...TanStackQueryProviderContext}>
            <RouterProvider router={router} />
          </TanStackQueryProvider.Provider>
          <Toaster />
        </TooltipProvider>
      </UIProvider>
    </StrictMode>
  );
}
