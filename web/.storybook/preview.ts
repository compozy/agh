import type { Preview } from "@storybook/react-vite";
import { withThemeByClassName } from "@storybook/addon-themes";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from "@tanstack/react-router";
import { createElement, useState, type ReactNode } from "react";
import { initialize, mswLoader } from "msw-storybook-addon";

import "../src/styles.css";
import { handlers as agentHandlers } from "@/systems/agent/mocks";
import { handlers as daemonHandlers } from "@/systems/daemon/mocks";
import { handlers as knowledgeHandlers } from "@/systems/knowledge/mocks";
import { handlers as networkHandlers } from "@/systems/network/mocks";
import { handlers as sessionHandlers } from "@/systems/session/mocks";
import { handlers as skillHandlers } from "@/systems/skill/mocks";
import { handlers as workspaceHandlers } from "@/systems/workspace/mocks";
import { handlers as automationHandlers } from "@/systems/automation/mocks";
import { handlers as bridgeHandlers } from "@/systems/bridges/mocks";

initialize({ onUnhandledRequest: "bypass" });

type StoryRenderer = () => ReactNode;
type StorybookRouterOptions = {
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

export function createStorybookRouter(
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
  const automationRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "automation",
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

  return createRouter({
    routeTree: rootRoute.addChildren([
      storyRoute,
      sessionRoute,
      automationRoute,
      bridgesRoute,
      networkRoute,
      knowledgeRoute,
      skillsRoute,
    ]),
    history: createMemoryHistory({
      initialEntries: options?.initialEntries ?? ["/"],
    }),
  });
}

function StorybookQueryClientBoundary({ children }: { children: ReactNode }) {
  const [queryClient] = useState(createStorybookQueryClient);

  return createElement(QueryClientProvider, { client: queryClient }, children);
}

function StorybookRouterBoundary({
  Story,
  initialEntries,
}: {
  Story: StoryRenderer;
  initialEntries?: string[];
}) {
  const [router] = useState(() => createStorybookRouter(Story, { initialEntries }));

  return createElement(RouterProvider, { router });
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

export const routerDecorator = (
  Story: StoryRenderer,
  context?: { parameters?: { router?: StorybookRouterOptions } }
) =>
  createElement(StorybookRouterBoundary, {
    Story,
    initialEntries: context?.parameters?.router?.initialEntries,
  });

export const storybookDecorators = [themeDecorator, queryClientDecorator, routerDecorator];
export const storybookLoaders = [mswLoader];
export const storybookSystemHandlers = [
  ...agentHandlers,
  ...automationHandlers,
  ...bridgeHandlers,
  ...daemonHandlers,
  ...knowledgeHandlers,
  ...networkHandlers,
  ...sessionHandlers,
  ...skillHandlers,
  ...workspaceHandlers,
];

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
      handlers: storybookSystemHandlers,
    },
  },
};

export default preview;
