import { cleanup, render, screen } from "@testing-library/react";
import { createElement } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";

const initialize = vi.fn();
const mswLoader = vi.fn(async () => ({}));

vi.mock("msw-storybook-addon", () => ({
  initialize,
  mswLoader,
}));

const webMain = (await import("../../.storybook/main")).default;
const webPreviewModule = await import("../../.storybook/preview");
const {
  createStorybookQueryClient,
  createStorybookRouter,
  queryClientDecorator,
  routerDecorator,
  storybookDecorators,
  storybookLoaders,
  storybookSystemHandlers,
  themeDecorator,
} = webPreviewModule;
const webPreview = webPreviewModule.default;

afterEach(() => {
  cleanup();
  document.documentElement.className = "";
});

function QueryClientProbe() {
  const queryClient = useQueryClient();
  const queryOptions = queryClient.getDefaultOptions().queries;

  return createElement(
    "output",
    { "data-testid": "query-client-defaults" },
    `${String(queryOptions?.retry)}|${String(queryOptions?.staleTime)}`
  );
}

describe("web Storybook config", () => {
  it("keeps the existing story glob and addons while serving static worker assets", () => {
    expect(webMain.stories).toEqual(["../src/**/*.stories.@(ts|tsx)"]);
    expect(webMain.addons).toEqual([
      "@storybook/addon-docs",
      "@storybook/addon-a11y",
      "@storybook/addon-themes",
    ]);
    expect(webMain.staticDirs).toEqual(["../public"]);
    expect(webMain.framework).toEqual({
      name: "@storybook/react-vite",
      options: {},
    });
  });

  it("registers MSW and preserves the theme decorator alongside one query and router decorator", () => {
    expect(initialize).toHaveBeenCalledWith({ onUnhandledRequest: "bypass" });
    expect(webPreview.loaders).toEqual(storybookLoaders);
    expect(storybookLoaders).toEqual([mswLoader]);
    expect(webPreview.decorators).toEqual(storybookDecorators);
    expect(webPreview.parameters?.msw?.handlers).toEqual(storybookSystemHandlers);
    expect(storybookSystemHandlers.length).toBeGreaterThan(0);
    expect(storybookDecorators.filter(decorator => decorator === themeDecorator)).toHaveLength(1);
    expect(
      storybookDecorators.filter(decorator => decorator === queryClientDecorator)
    ).toHaveLength(1);
    expect(storybookDecorators.filter(decorator => decorator === routerDecorator)).toHaveLength(1);
  });

  it("creates story-scoped query clients with retry disabled and infinite stale time", () => {
    const queryClient = createStorybookQueryClient();
    const queryOptions = queryClient.getDefaultOptions().queries;

    expect(queryOptions?.retry).toBe(false);
    expect(queryOptions?.staleTime).toBe(Number.POSITIVE_INFINITY);
  });

  it("wraps stories in a QueryClientProvider with the expected defaults", () => {
    render(queryClientDecorator(() => createElement(QueryClientProbe)));

    expect(screen.getByTestId("query-client-defaults")).toHaveTextContent("false|Infinity");
  });

  it("creates a memory router stub rooted at slash for story decorators", async () => {
    const router = createStorybookRouter();

    await router.load();

    expect(router.state.location.pathname).toBe("/");
    expect(storybookDecorators).toContain(routerDecorator);
  });

  it("includes placeholder routes for linked app surfaces used by stories", async () => {
    const router = createStorybookRouter();

    await router.navigate({ to: "/session/$id", params: { id: "sess-storybook" } });
    expect(router.state.location.pathname).toBe("/session/sess-storybook");

    await router.navigate({ to: "/automation" });
    expect(router.state.location.pathname).toBe("/automation");
  });

  it("renders stories through the router decorator stub", async () => {
    render(routerDecorator(() => createElement("div", { "data-testid": "router-story" }, "Story")));

    expect(await screen.findByTestId("router-story")).toHaveTextContent("Story");
  });
});
