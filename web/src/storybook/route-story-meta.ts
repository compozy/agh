import type { Meta } from "@storybook/react-vite";

import { StorybookRouteCanvas } from "./route-story";

export {
  StorybookRestartBannerSetup,
  StorybookRouteCanvas,
  StorybookUserHomeDirSetup,
  StorybookWorkspaceSetup,
} from "./route-story";

export function createRouteStoryMeta(
  title: string,
  description: string
): Meta<typeof StorybookRouteCanvas> {
  return {
    title,
    component: StorybookRouteCanvas,
    parameters: {
      layout: "fullscreen",
      docs: {
        description: {
          component: description,
        },
      },
    },
  };
}

export function appRouteParameters(path: string) {
  return {
    layout: "fullscreen" as const,
    router: {
      kind: "app" as const,
      initialEntries: [path],
    },
  };
}
