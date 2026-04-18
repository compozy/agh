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

initialize({ onUnhandledRequest: "bypass" });

type StoryRenderer = () => ReactNode;

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

export function createStorybookRouter(Story: StoryRenderer = () => null) {
  const rootRoute = createRootRoute();
  const storyRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "/",
    component: Story,
  });

  return createRouter({
    routeTree: rootRoute.addChildren([storyRoute]),
    history: createMemoryHistory({
      initialEntries: ["/"],
    }),
  });
}

function StorybookQueryClientBoundary({ children }: { children: ReactNode }) {
  const [queryClient] = useState(createStorybookQueryClient);

  return createElement(QueryClientProvider, { client: queryClient }, children);
}

function StorybookRouterBoundary({ Story }: { Story: StoryRenderer }) {
  const [router] = useState(() => createStorybookRouter(Story));

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

export const routerDecorator = (Story: StoryRenderer) =>
  createElement(StorybookRouterBoundary, { Story });

export const storybookDecorators = [themeDecorator, queryClientDecorator, routerDecorator];
export const storybookLoaders = [mswLoader];

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
  },
};

export default preview;
