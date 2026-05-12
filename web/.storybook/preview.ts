import type { Preview } from "@storybook/react-vite";
import { withThemeByClassName } from "@storybook/addon-themes";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TooltipProvider, UIProvider } from "@agh/ui";
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router";
import { Fragment, createElement, useState, type ReactNode } from "react";
import { initialize, mswLoader } from "msw-storybook-addon";

import "../src/styles.css";
import { routeTree } from "@/routeTree.gen";
import { storybookSystemHandlerGroups, storybookSystemHandlers } from "@/storybook/msw";
import { resetSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import { useActiveWorkspaceStore } from "@/systems/workspace/hooks/use-active-workspace-store";
import { useSessionStore } from "@/systems/session/hooks/use-session-store";
import { useSidebarStore } from "@/hooks/use-sidebar-store";

initialize({ onUnhandledRequest: "bypass" });

type StoryRenderer = () => ReactNode;
export type StorybookRouterMode = "app" | "stub";
type StorybookRouterOptions = {
  kind?: StorybookRouterMode;
  initialEntries?: string[];
};

export function createStorybookQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        staleTime: Number.POSITIVE_INFINITY,
      },
    },
  });
}

function createStubStorybookRouter(
  Story: StoryRenderer = () => null,
  options?: StorybookRouterOptions
) {
  const rootRoute = createRootRoute();
  const storyRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "/",
    component: Story,
  });
  const sessionRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "session/$id",
    component: Story,
  });
  const jobsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "jobs",
    component: Story,
  });
  const triggersRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "triggers",
    component: Story,
  });
  const bridgesRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "bridges",
    component: Story,
  });
  const networkRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "network",
    component: Story,
  });
  const knowledgeRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "knowledge",
    component: Story,
  });
  const skillsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "skills",
    component: Story,
  });
  const tasksRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "tasks",
    component: Story,
  });
  const taskNewRoute = createRoute({
    getParentRoute: () => tasksRoute,
    path: "new",
    component: Story,
  });
  const taskDetailRoute = createRoute({
    getParentRoute: () => tasksRoute,
    path: "$id",
    component: Story,
  });
  const taskEditRoute = createRoute({
    getParentRoute: () => taskDetailRoute,
    path: "edit",
    component: Story,
  });
  const taskRunDetailRoute = createRoute({
    getParentRoute: () => taskDetailRoute,
    path: "runs/$runId",
    component: Story,
  });

  return createRouter({
    routeTree: rootRoute.addChildren([
      storyRoute,
      sessionRoute,
      jobsRoute,
      triggersRoute,
      bridgesRoute,
      networkRoute,
      knowledgeRoute,
      skillsRoute,
      tasksRoute.addChildren([
        taskNewRoute,
        taskDetailRoute.addChildren([taskEditRoute, taskRunDetailRoute]),
      ]),
    ]),
    history: createMemoryHistory({
      initialEntries: options?.initialEntries ?? ["/"],
    }),
  });
}

function createAppStorybookRouter(queryClient: QueryClient, options?: StorybookRouterOptions) {
  return createRouter({
    routeTree,
    context: {
      queryClient,
    },
    history: createMemoryHistory({
      initialEntries: options?.initialEntries ?? ["/"],
    }),
    defaultPreload: "intent",
    scrollRestoration: true,
    defaultStructuralSharing: true,
    defaultPreloadStaleTime: 0,
  });
}

export function createStorybookRouter(
  Story: StoryRenderer = () => null,
  options?: StorybookRouterOptions,
  queryClient?: QueryClient
) {
  if (options?.kind === "app") {
    return createAppStorybookRouter(queryClient ?? createStorybookQueryClient(), options);
  }

  return createStubStorybookRouter(Story, options);
}

function resetStorybookAppState() {
  useSidebarStore.setState({ collapsed: false });
  useActiveWorkspaceStore.getState().clearSelectedWorkspaceId();
  useSessionStore.getState().clearAllDrafts();
  resetSettingsRestartStore();
}

function StorybookQueryClientBoundary({ children }: { children: ReactNode }) {
  const [queryClient] = useState(createStorybookQueryClient);

  return createElement(QueryClientProvider, { client: queryClient }, children);
}

function StorybookProvidersBoundary({
  Story,
  routerOptions,
}: {
  Story: StoryRenderer;
  routerOptions?: StorybookRouterOptions;
}) {
  const [queryClient] = useState(createStorybookQueryClient);
  const [router] = useState(() => {
    if (routerOptions?.kind === "app") {
      resetStorybookAppState();
    }
    return createStorybookRouter(Story, routerOptions, queryClient);
  });

  return createElement(
    QueryClientProvider,
    { client: queryClient },
    routerOptions?.kind === "app"
      ? createElement(
          Fragment,
          null,
          createElement(Story),
          createElement(RouterProvider, { router })
        )
      : createElement(RouterProvider, { router })
  );
}

export const themeDecorator = withThemeByClassName({
  themes: {
    light: "",
    dark: "dark",
  },
  defaultTheme: "dark",
});

export const queryClientDecorator = (Story: StoryRenderer) =>
  createElement(StorybookQueryClientBoundary, null, createElement(Story));

export const uiProviderDecorator = (Story: StoryRenderer) =>
  createElement(UIProvider, null, createElement(TooltipProvider, null, createElement(Story)));

export const routerDecorator = (
  Story: StoryRenderer,
  context?: { parameters?: { router?: StorybookRouterOptions } }
) =>
  createElement(StorybookProvidersBoundary, {
    Story,
    routerOptions: context?.parameters?.router,
  });

export const storybookDecorators = [themeDecorator, uiProviderDecorator, routerDecorator];
export const storybookLoaders = [mswLoader];
export { storybookSystemHandlerGroups, storybookSystemHandlers };

const preview: Preview = {
  decorators: storybookDecorators,
  loaders: storybookLoaders,
  parameters: {
    backgrounds: {
      disable: true,
    },
    controls: {
      expanded: true,
    },
    msw: {
      handlers: storybookSystemHandlerGroups,
    },
  },
};

export default preview;
